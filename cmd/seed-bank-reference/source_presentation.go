//go:build postgres

package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

var sourcePlaceholderTitle = regexp.MustCompile(`(?i)^(?:.*[[:space:]])?(?:row|record|line)[[:space:]#:-]*[0-9]+//go:build postgres

package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

)
var sourceTitleRowPrefix = regexp.MustCompile(`(?i)^(?:.*[[:space:]])?(?:row|record|line)[[:space:]#:-]*[0-9]+[[:space:]]*[:–—-][[:space:]]*(.+)//go:build postgres

package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

)

// Header matching is semantic only. Preserve the original label, cell and
// value in immutable source_fields; never normalize or mutate those facts.
func sourceHeaderKey(value string) string {
	value = strings.Map(func(r rune) rune {
		switch r {
		case '\u00a0', '\u2007', '\u202f', '\ufeff', '\r', '\n', '\t':
			return ' '
		default:
			return r
		}
	}, value)
	normalized := strings.ToLower(strings.Join(strings.Fields(value), " "))
	normalized = strings.NewReplacer(" / ", "/", "/ ", "/", " /", "/", " : ", ":", " & ", "&").Replace(normalized)
	return normalized
}

func sourceLookupField(record sourceRecord, label string) (string, bool) {
	key := sourceHeaderKey(label)
	if key == "" {
		return "", false
	}
	var found bool
	var value string
	for _, field := range record.Fields {
		if sourceHeaderKey(field.Label) != key {
			continue
		}
		candidate := strings.TrimSpace(field.Value)
		if !found {
			found, value = true, candidate
		} else if candidate != value {
			// Never arbitrarily select one of two conflicting source columns.
			return "", false
		}
	}
	return value, found
}

func sourceRecordDisplayTitle(group sourceRecordGroup, record sourceRecord) string {
	existing := strings.TrimSpace(record.Title)
	if parts := sourceTitleRowPrefix.FindStringSubmatch(existing); len(parts) == 2 && sourceDescriptiveValue(parts[1]) {
		return sourceShort(strings.TrimSpace(parts[1]), 200)
	}
	if existing != "" && !sourcePlaceholderTitle.MatchString(existing) {
		return sourceShort(existing, 200)
	}
	for _, label := range []string{
		"Branch", "RISK DESCRIPTION", "Risk Event Description", "RISK METRICS",
		"Requirement / Checklist Item", "TRAN_PARTICULAR", "Process Name",
		"Business Process", "Activity", "SERVICE", "Asset Name", "FINDINGS",
		"Control Area", "Risk Driver Descriptions Level 1",
	} {
		if value, ok := sourceLookupField(record, label); ok && sourceDescriptiveValue(value) {
			return sourceShort(value, 200)
		}
	}
	for _, field := range record.Fields {
		if key := sourceHeaderKey(field.Label); key != "" && !sourceIdentityOnlyField(key) && sourceDescriptiveValue(field.Value) {
			return sourceShort(strings.TrimSpace(field.Value), 200)
		}
	}
	if title := strings.TrimSpace(group.Title); title != "" && !sourcePlaceholderTitle.MatchString(title) {
		return sourceShort(title, 200)
	}
	return "Source record"
}

func sourceIdentityOnlyField(key string) bool {
	switch key {
	case "id", "risk id", "code", "month", "month number", "year", "quarter",
		"qtr number", "date", "period", "status", "amount", "risk level",
		"currency of loss", "source row", "source cell", "source range":
		return true
	default:
		return false
	}
}

func sourceDescriptiveValue(value string) bool {
	value = strings.TrimSpace(value)
	if len([]rune(value)) < 3 || sourcePlaceholderTitle.MatchString(value) {
		return false
	}
	for _, character := range value {
		if unicode.IsLetter(character) {
			return true
		}
	}
	return false
}

// Only presentation v2 creates this revised, structured answer. Legacy source
// distributions keep their original immutable response/template contracts.
func sourceRecordTextV2(record sourceRecord) string {
	lines := make([]string, 0, len(record.Fields))
	seen := make(map[string]int, len(record.Fields))
	for _, field := range record.Fields {
		key := sourceHeaderKey(field.Label)
		seen[key]++
	}
	for _, field := range record.Fields {
		label := strings.TrimSpace(field.Label)
		if label == "" {
			label = "Source value"
		}
		if seen[sourceHeaderKey(field.Label)] > 1 || sourceHeaderKey(field.Label) == "" {
			if cell := strings.TrimSpace(field.SourceCell); cell != "" {
				label += " (" + cell + ")"
			}
		}
		value := strings.TrimSpace(field.Value)
		if value == "" {
			value = "Not recorded in source"
		} else {
			value = strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n")
			value = strings.ReplaceAll(value, "\n", "\n  ")
		}
		lines = append(lines, fmt.Sprintf("%s: %s", label, value))
	}
	return strings.Join(lines, "\n")
}
