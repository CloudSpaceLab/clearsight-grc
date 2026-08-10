package assurance

import (
	"regexp"
	"sort"
	"strings"
)

var safeCategoricalPattern = regexp.MustCompile(`(?i)(status|state|severity|tier|type|category|class|region|risk[_ -]?(level|rating))`)

var lexicalHintRules = []struct {
	pattern    *regexp.Regexp
	kind       HintKind
	detail     string
	confidence float64
	types      map[LogicalType]bool
}{
	{regexp.MustCompile(`(?i)(expir|valid[_ -]?until|due[_ -]?(at|date)?|review[_ -]?date)`), HintTimeBound, "expiry_or_due", 0.96, typeSet(TypeTime)},
	{regexp.MustCompile(`(?i)(last[_ -]?(login|seen|activity)|inactive[_ -]?since)`), HintActivity, "last_activity", 0.95, typeSet(TypeTime)},
	{regexp.MustCompile(`(?i)(^|[_ -])(status|state)([_ -]|$)`), HintStatus, "state", 0.92, typeSet(TypeString, TypeBool)},
	{regexp.MustCompile(`(?i)(severity|risk[_ -]?(level|rating|score)|criticality)`), HintStatus, "risk_state", 0.88, typeSet(TypeString, TypeNumber)},
	{regexp.MustCompile(`(?i)(owner|assignee|responsib)`), HintOwner, "ownership", 0.93, typeSet(TypeString)},
	{regexp.MustCompile(`(?i)(^|[_ -])(rto|rpo|mao)([_ -]|$)`), HintResilienceTarget, "resilience", 0.98, typeSet(TypeNumber, TypeString, TypeTime)},
	{regexp.MustCompile(`(?i)(kyc|bvn|nin|aml|identity[_ -]?review)`), HintIdentityCompliance, "identity_compliance", 0.95, typeSet(TypeString, TypeBool, TypeTime)},
	{regexp.MustCompile(`(?i)(patch|cve|cvss|edr|firmware|antimalware|vulnerab)`), HintSecurityPosture, "security_posture", 0.94, typeSet(TypeString, TypeNumber, TypeBool, TypeTime)},
	{regexp.MustCompile(`(?i)(backup|restore|recovery[_ -]?test)`), HintSecurityPosture, "recovery_posture", 0.90, typeSet(TypeString, TypeBool, TypeTime, TypeNumber)},
	{regexp.MustCompile(`(?i)(approved|approval|verified|compliant|attested)`), HintApprovalState, "approval_or_assurance", 0.88, typeSet(TypeString, TypeBool, TypeTime)},
	{regexp.MustCompile(`(?i)(password|secret|token|bvn|nin|ssn|account[_ -]?(number|no)|email|phone)`), HintSensitive, "sensitive", 0.99, nil},
	{regexp.MustCompile(`(?i)(^id$|[_ -]id$|code$|reference$|[_ -](number|no)$)`), HintIdentifier, "identifier", 0.84, typeSet(TypeString, TypeNumber)},
}

func inferHints(field Field, profile FieldProfile) []FieldHint {
	values := make([]FieldHint, 0, 4)
	for _, rule := range lexicalHintRules {
		if !rule.pattern.MatchString(field.Name) || (rule.types != nil && !rule.types[field.Type]) {
			continue
		}
		values = append(values, FieldHint{Kind: rule.kind, Detail: rule.detail, Source: HintLexical, Confidence: rule.confidence})
	}
	if (field.Type == TypeString || field.Type == TypeBool) && profile.RowsObserved > 0 && !profile.DistinctCapped && profile.DistinctObserved > 0 && profile.DistinctObserved <= 20 {
		values = append(values, FieldHint{Kind: HintCategorical, Detail: "low_cardinality", Source: HintStatistical, Confidence: 0.76})
	}
	if field.Type != TypeUnknown {
		values = append(values, FieldHint{Kind: HintLogicalType, Detail: string(field.Type), Source: HintStructural, Confidence: 1})
	}
	return dedupeHints(values)
}

func safeCategoricalField(field Field) bool {
	name := strings.ToLower(field.Name)
	if lexicalMatches(HintSensitive, name) {
		return false
	}
	return safeCategoricalPattern.MatchString(name)
}

func lexicalMatches(kind HintKind, value string) bool {
	for _, rule := range lexicalHintRules {
		if rule.kind == kind && rule.pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func typeSet(values ...LogicalType) map[LogicalType]bool {
	result := make(map[LogicalType]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func dedupeHints(values []FieldHint) []FieldHint {
	seen := make(map[string]struct{}, len(values))
	result := make([]FieldHint, 0, len(values))
	for _, value := range values {
		key := string(value.Kind) + "\x1f" + value.Detail + "\x1f" + string(value.Source)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Confidence == result[j].Confidence {
			if result[i].Kind == result[j].Kind {
				return result[i].Detail < result[j].Detail
			}
			return result[i].Kind < result[j].Kind
		}
		return result[i].Confidence > result[j].Confidence
	})
	return result
}
