package documentimport

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
)

var (
	obligationWhitespace = regexp.MustCompile(`\s+`)
	obligationToken      = regexp.MustCompile(`[a-z0-9]+`)
	citationPattern      = regexp.MustCompile(`\b(?:section|article|regulation|paragraph|rule|part)\s+[a-z0-9][a-z0-9().-]*`)
	boundedTimePattern   = regexp.MustCompile(`\bwithin\s+\d+\s+(?:(?:business|calendar)\s+)?(?:hours?|days?|weeks?|months?|years?)\b`)
	deadlinePattern      = regexp.MustCompile(`\b(?:annually|quarterly|monthly|weekly|daily|no later than\s+[^,.;]{1,48}|before\s+\d{1,2}(?:st|nd|rd|th)?\s+[a-z]+|deadline\s+for\s+[^,.;]{1,64})\b`)
	leadingArticle       = regexp.MustCompile(`^(?:a|an|the)\s+`)
	modalityPatterns     = []struct {
		value string
		rx    *regexp.Regexp
	}{
		{value: "MUST_NOT", rx: regexp.MustCompile(`\b(?:must\s+not|shall\s+not)\b`)},
		{value: "MUST", rx: regexp.MustCompile(`\b(?:must|shall|is\s+required\s+to|are\s+required\s+to|is\s+to|are\s+to)\b`)},
		{value: "SHOULD", rx: regexp.MustCompile(`\bshould\b`)},
		{value: "EXPECTED", rx: regexp.MustCompile(`\b(?:is|are)\s+expected\s+to\b`)},
		{value: "MAY", rx: regexp.MustCompile(`\bmay\b`)},
	}
	obligationStopWords = map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {}, "for": {},
		"from": {}, "in": {}, "is": {}, "it": {}, "must": {}, "not": {}, "of": {}, "on": {}, "or": {},
		"shall": {}, "should": {}, "the": {}, "their": {}, "to": {}, "under": {}, "was": {}, "were": {},
		"which": {}, "with": {}, "within": {},
	}
)

func ParseObligation(statement, kind string) Obligation {
	normalized := normalizeObligationText(statement)
	value := Obligation{
		Eligible:    kind == "REQUIREMENT_CANDIDATE" || kind == "DEADLINE_CANDIDATE" || kind == "CONTROL_EXPECTATION",
		Citations:   normalizedMatches(citationPattern, normalized),
		Dates:       append(normalizedMatches(boundedTimePattern, normalized), normalizedMatches(deadlinePattern, normalized)...),
		Topics:      obligationTopics(normalized),
		Uncertainty: []string{},
	}
	value.Dates = uniqueSorted(value.Dates)

	modalStart, modalEnd := -1, -1
	for _, candidate := range modalityPatterns {
		location := candidate.rx.FindStringIndex(normalized)
		if location == nil {
			continue
		}
		value.Modality = candidate.value
		modalStart, modalEnd = location[0], location[1]
		break
	}
	if modalStart >= 0 {
		actor := strings.TrimSpace(strings.Trim(normalized[:modalStart], " ,:;.-"))
		value.Actor = leadingArticle.ReplaceAllString(actor, "")
		remainder := strings.TrimSpace(strings.Trim(normalized[modalEnd:], " ,:;.-"))
		parts := strings.Fields(remainder)
		if len(parts) > 0 {
			value.Action = strings.Trim(parts[0], " ,:;.-")
			value.Object = strings.TrimSpace(strings.Trim(strings.TrimPrefix(remainder, parts[0]), " ,:;.-"))
		}
	}
	if value.Eligible && value.Modality == "" {
		value.Uncertainty = append(value.Uncertainty, "MODALITY_NOT_EXPLICIT")
	}
	if value.Eligible && value.Actor == "" {
		value.Uncertainty = append(value.Uncertainty, "ACTOR_NOT_EXPLICIT")
	}
	if value.Eligible && value.Action == "" {
		value.Uncertainty = append(value.Uncertainty, "ACTION_NOT_EXPLICIT")
	}
	sort.Strings(value.Uncertainty)
	canonical := strings.Join([]string{
		value.Modality, value.Actor, value.Action, value.Object,
		strings.Join(value.Citations, "|"), strings.Join(value.Dates, "|"), strings.Join(value.Topics, "|"),
	}, "\x00")
	digest := sha256.Sum256([]byte(canonical))
	value.Fingerprint = hex.EncodeToString(digest[:])
	return value
}

func normalizeObligationText(value string) string {
	return strings.TrimSpace(obligationWhitespace.ReplaceAllString(strings.ToLower(value), " "))
}

func normalizedMatches(pattern *regexp.Regexp, value string) []string {
	matches := pattern.FindAllString(value, -1)
	for index := range matches {
		matches[index] = strings.Trim(matches[index], " ,:;.-")
	}
	return uniqueSorted(matches)
}

func obligationTopics(value string) []string {
	tokens := obligationToken.FindAllString(value, -1)
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if len(token) < 3 {
			continue
		}
		if _, ignored := obligationStopWords[token]; ignored {
			continue
		}
		result = append(result, token)
	}
	return uniqueSorted(result)
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
