package documentcoverage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

var normalizedTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

var matchStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {}, "for": {}, "from": {},
	"in": {}, "is": {}, "it": {}, "must": {}, "of": {}, "on": {}, "or": {}, "shall": {}, "should": {},
	"the": {}, "their": {}, "to": {}, "under": {}, "was": {}, "were": {}, "which": {}, "with": {},
}

func MatchCandidate(candidate Candidate, programs []ProgramSnapshot) []Match {
	matches := make([]Match, 0, 5)
	for _, program := range programs {
		if hardScopeConflict(candidate, program) {
			continue
		}
		for _, requirement := range program.Requirements {
			if requirement.Status != continuity.RequirementApproved || requirement.Applicability == continuity.ApplicabilityNotApplicable {
				continue
			}
			components := scoreComponents(candidate, program, requirement)
			score := 0.0
			for _, component := range components {
				score += component.Score * component.Weight
			}
			if score < PossibleMatchThreshold {
				continue
			}
			match := Match{
				ProgramID: program.ProgramID, ProgramCode: program.Code, ProgramName: program.Name, ProgramVersion: program.Version,
				RequirementID: requirement.ID, RequirementCode: requirement.Code, RequirementTitle: requirement.Title, RequirementVersion: requirement.Version,
				Score: score, Band: matchBand(score), Components: components,
				Rationale: matchRationale(components), Conflicts: []string{}, Coverage: requirement.Coverage,
			}
			match.ID = stableMatchID(candidate.ID, program.ProgramID, requirement.ID)
			matches = append(matches, match)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].ProgramCode != matches[j].ProgramCode {
			return matches[i].ProgramCode < matches[j].ProgramCode
		}
		if matches[i].RequirementCode != matches[j].RequirementCode {
			return matches[i].RequirementCode < matches[j].RequirementCode
		}
		return matches[i].RequirementID < matches[j].RequirementID
	})
	if len(matches) > 5 {
		matches = matches[:5]
	}
	return matches
}

func hardScopeConflict(candidate Candidate, program ProgramSnapshot) bool {
	if candidate.TenantID != "" && program.TenantID != "" && candidate.TenantID != program.TenantID {
		return true
	}
	if candidate.LegalEntityID != "" && program.LegalEntityID != "" && candidate.LegalEntityID != program.LegalEntityID {
		return true
	}
	if program.Status == continuity.ProgramRetired {
		return true
	}
	if candidate.Jurisdiction != "" && program.Jurisdiction != "" && normalizeJurisdiction(candidate.Jurisdiction) != normalizeJurisdiction(program.Jurisdiction) {
		return true
	}
	if candidate.Regulator != "" && program.Regulator != "" && normalizeText(candidate.Regulator) != normalizeText(program.Regulator) {
		return true
	}
	if candidate.ProgramType != "" && program.Type != "" && normalizeText(candidate.ProgramType) != normalizeText(program.Type) {
		return true
	}
	return false
}

func scoreComponents(candidate Candidate, program ProgramSnapshot, requirement RequirementTarget) []ScoreComponent {
	citation := similarity(candidate.Citations, requirement.Citations)
	if len(candidate.Citations) == 0 && len(requirement.Citations) == 0 {
		citation = codeCitationSimilarity(candidate, requirement)
	}
	candidateTerms := append(append(append([]string{}, candidate.Topics...), candidate.Action), candidate.Object)
	targetTerms := append(append(append([]string{}, requirement.Topics...), requirement.Action), requirement.Object)
	content := similarity(tokenize(strings.Join(candidateTerms, " ")), tokenize(strings.Join(targetTerms, " ")))
	scopeParts := []float64{}
	scopeParts = append(scopeParts, compatibleField(candidate.Actor, requirement.Actor))
	scopeParts = append(scopeParts, compatibleField(candidate.Jurisdiction, program.Jurisdiction))
	scopeParts = append(scopeParts, compatibleField(candidate.ProgramType, program.Type))
	scope := averageKnown(scopeParts)
	cadenceParts := []float64{similarity(candidate.Dates, requirement.Dates), compatibleField(candidate.Modality, requirement.Modality)}
	cadence := averageKnown(cadenceParts)
	return []ScoreComponent{
		{Name: "Citation and instrument", Weight: .35, Score: citation, Reason: scoreReason(citation, "citation or instrument agreement")},
		{Name: "Action, object and topics", Weight: .30, Score: content, Reason: scoreReason(content, "action and subject agreement")},
		{Name: "Scope", Weight: .20, Score: scope, Reason: scoreReason(scope, "actor, jurisdiction and Program scope")},
		{Name: "Cadence and modality", Weight: .15, Score: cadence, Reason: scoreReason(cadence, "timing and obligation modality")},
	}
}

func matchBand(score float64) MatchBand {
	if score >= StrongMatchThreshold {
		return MatchStrong
	}
	if score >= PossibleMatchThreshold {
		return MatchPossible
	}
	return MatchWeak
}

func normalizeText(value string) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func normalizeJurisdiction(value string) string {
	normalized := normalizeText(value)
	switch normalized {
	case "ng", "nga", "nigeria", "federal republic of nigeria":
		return "nigeria"
	case "us", "usa", "united states", "united states of america":
		return "united states"
	case "uk", "gbr", "united kingdom", "great britain":
		return "united kingdom"
	default:
		return normalized
	}
}

func tokenize(value string) []string {
	values := normalizedTokenPattern.FindAllString(normalizeText(value), -1)
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if len(value) < 3 {
			continue
		}
		if _, stop := matchStopWords[value]; stop {
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

func similarity(left, right []string) float64 {
	leftTokens := tokenize(strings.Join(left, " "))
	rightTokens := tokenize(strings.Join(right, " "))
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return 0
	}
	rightSet := make(map[string]struct{}, len(rightTokens))
	for _, value := range rightTokens {
		rightSet[value] = struct{}{}
	}
	intersection := 0
	for _, value := range leftTokens {
		if _, ok := rightSet[value]; ok {
			intersection++
		}
	}
	return float64(2*intersection) / float64(len(leftTokens)+len(rightTokens))
}

func compatibleField(left, right string) float64 {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return -1
	}
	if normalizeText(left) == normalizeText(right) || normalizeJurisdiction(left) == normalizeJurisdiction(right) {
		return 1
	}
	return similarity(tokenize(left), tokenize(right))
}

func averageKnown(values []float64) float64 {
	total, count := 0.0, 0
	for _, value := range values {
		if value < 0 {
			continue
		}
		total += value
		count++
	}
	if count == 0 {
		return 0.5
	}
	return total / float64(count)
}

func codeCitationSimilarity(candidate Candidate, requirement RequirementTarget) float64 {
	terms := tokenize(candidate.Statement)
	code := tokenize(requirement.Code + " " + requirement.SourceAnchor)
	return similarity(terms, code)
}

func scoreReason(score float64, label string) string {
	switch {
	case score >= .85:
		return "Strong " + label
	case score >= .55:
		return "Partial " + label
	default:
		return "Limited " + label
	}
}

func matchRationale(components []ScoreComponent) string {
	best := components[0]
	for _, component := range components[1:] {
		if component.Score*component.Weight > best.Score*best.Weight {
			best = component
		}
	}
	return fmt.Sprintf("Best signal: %s (%d%%).", best.Name, int(best.Score*100+.5))
}

func stableMatchID(candidateID, programID, requirementID string) string {
	digest := sha256.Sum256([]byte(candidateID + "\x00" + programID + "\x00" + requirementID))
	return hex.EncodeToString(digest[:16])
}
