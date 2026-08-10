package assurance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

const DefaultHintPackVersion = "bank-core-v1"

const (
	hardMaxProfileFields          = 512
	hardMaxProfileRows            = 4096
	hardMaxProfileCells           = 64 << 10
	hardMaxProfileDistinct        = 4096
	hardMaxProfileTopValues       = 32
	hardMaxProfileValueBytes      = 512
	hardMaxProfileCellBytes       = 64 << 10
	maxPublishedCategoricalValues = 20
)

type ProfileLimits struct {
	MaxFields     int
	MaxRows       int
	MaxDistinct   int
	MaxTopValues  int
	MaxValueBytes int
	MaxCellBytes  int
}

func DefaultProfileLimits() ProfileLimits {
	return ProfileLimits{MaxFields: 128, MaxRows: 512, MaxDistinct: 512, MaxTopValues: 8, MaxValueBytes: 128, MaxCellBytes: 4096}
}

type ValueCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type HintSource string

const (
	HintLexical     HintSource = "LEXICAL"
	HintStructural  HintSource = "STRUCTURAL"
	HintStatistical HintSource = "STATISTICAL"
)

type HintKind string

const (
	HintTimeBound          HintKind = "TIME_BOUND"
	HintActivity           HintKind = "ACTIVITY"
	HintStatus             HintKind = "STATUS"
	HintOwner              HintKind = "OWNER"
	HintResilienceTarget   HintKind = "RESILIENCE_TARGET"
	HintIdentityCompliance HintKind = "IDENTITY_COMPLIANCE"
	HintSecurityPosture    HintKind = "SECURITY_POSTURE"
	HintApprovalState      HintKind = "APPROVAL_STATE"
	HintIdentifier         HintKind = "IDENTIFIER"
	HintSensitive          HintKind = "SENSITIVE"
	HintCategorical        HintKind = "CATEGORICAL"
	HintLogicalType        HintKind = "LOGICAL_TYPE"
)

type FieldHint struct {
	Kind       HintKind   `json:"kind"`
	Detail     string     `json:"detail,omitempty"`
	Source     HintSource `json:"source"`
	Confidence float64    `json:"confidence"`
}

type FieldProfile struct {
	Name             string       `json:"name"`
	Type             LogicalType  `json:"type"`
	RowsObserved     int          `json:"rows_observed"`
	NullCount        int          `json:"null_count"`
	BlankCount       int          `json:"blank_count"`
	InvalidCount     int          `json:"invalid_count"`
	DistinctObserved int          `json:"distinct_observed"`
	DistinctCapped   bool         `json:"distinct_capped"`
	TopValues        []ValueCount `json:"top_values,omitempty"`
	MinNumber        *float64     `json:"min_number,omitempty"`
	MaxNumber        *float64     `json:"max_number,omitempty"`
	MinTime          *time.Time   `json:"min_time,omitempty"`
	MaxTime          *time.Time   `json:"max_time,omitempty"`
	Hints            []FieldHint  `json:"hints,omitempty"`
}

type PopulationProfile struct {
	SchemaFingerprint string         `json:"schema_fingerprint"`
	RowsObserved      int            `json:"rows_observed"`
	RowsOmitted       int            `json:"rows_omitted"`
	HintPackVersion   string         `json:"hint_pack_version"`
	Fields            []FieldProfile `json:"fields"`
}

func ProfileRows(schema Schema, rows []map[string]any, limits ProfileLimits) (PopulationProfile, error) {
	if err := schema.Validate(); err != nil {
		return PopulationProfile{}, err
	}
	limits = normalizedProfileLimits(limits)
	if len(schema.Fields) > limits.MaxFields {
		return PopulationProfile{}, fmt.Errorf("schema has %d fields; maximum is %d", len(schema.Fields), limits.MaxFields)
	}
	fingerprint, err := schema.Fingerprint()
	if err != nil {
		return PopulationProfile{}, err
	}

	observed := boundedProfileRows(len(rows), len(schema.Fields), limits.MaxRows)
	profile := PopulationProfile{
		SchemaFingerprint: fingerprint,
		RowsObserved:      observed,
		RowsOmitted:       len(rows) - observed,
		HintPackVersion:   DefaultHintPackVersion,
		Fields:            make([]FieldProfile, len(schema.Fields)),
	}
	states := make([]profileState, len(schema.Fields))
	for index, field := range schema.Fields {
		profile.Fields[index] = FieldProfile{Name: field.Name, Type: field.Type, RowsObserved: observed}
		states[index] = profileState{
			distinct:    make(map[string]struct{}),
			valueCounts: make(map[string]int),
			captureTop:  safeCategoricalField(field),
		}
	}

	for rowIndex := 0; rowIndex < observed; rowIndex++ {
		row := rows[rowIndex]
		for fieldIndex, field := range schema.Fields {
			value, exists := row[field.Name]
			if !exists || value == nil {
				profile.Fields[fieldIndex].NullCount++
				continue
			}
			if text, ok := value.(string); ok && len(text) > limits.MaxCellBytes {
				profile.Fields[fieldIndex].InvalidCount++
				continue
			}
			typed, ok := coerceValue(field.Type, value)
			if !ok {
				profile.Fields[fieldIndex].InvalidCount++
				continue
			}
			if typed.kind == TypeString && strings.TrimSpace(typed.stringValue) == "" {
				profile.Fields[fieldIndex].BlankCount++
			}
			states[fieldIndex].observeDistinct(typed, limits)
			states[fieldIndex].observeRange(&profile.Fields[fieldIndex], typed)
		}
	}

	for index, field := range schema.Fields {
		state := &states[index]
		profile.Fields[index].DistinctObserved = len(state.distinct)
		profile.Fields[index].DistinctCapped = state.distinctCapped
		if !state.distinctCapped && len(state.distinct) <= maxPublishedCategoricalValues {
			profile.Fields[index].TopValues = state.topValues(limits.MaxTopValues)
		}
		profile.Fields[index].Hints = inferHints(field, profile.Fields[index])
	}
	return profile, nil
}

func boundedProfileRows(rowCount, fieldCount, configuredMaxRows int) int {
	observed := rowCount
	if observed > configuredMaxRows {
		observed = configuredMaxRows
	}
	if fieldCount <= 0 {
		return observed
	}
	maxRowsByCells := hardMaxProfileCells / fieldCount
	if maxRowsByCells < 1 {
		maxRowsByCells = 1
	}
	if observed > maxRowsByCells {
		observed = maxRowsByCells
	}
	return observed
}

type profileState struct {
	distinct       map[string]struct{}
	distinctCapped bool
	valueCounts    map[string]int
	captureTop     bool
}

func (s *profileState) observeDistinct(value typedValue, limits ProfileLimits) {
	key, display := profileValueKey(value, limits.MaxValueBytes)
	if _, exists := s.distinct[key]; !exists {
		if len(s.distinct) >= limits.MaxDistinct {
			s.distinctCapped = true
			s.captureTop = false
			s.valueCounts = nil
		} else {
			s.distinct[key] = struct{}{}
			if len(s.distinct) > maxPublishedCategoricalValues {
				s.captureTop = false
				s.valueCounts = nil
			}
		}
	}
	if s.captureTop && display != "" {
		s.valueCounts[display]++
	}
}

func (s *profileState) observeRange(profile *FieldProfile, value typedValue) {
	switch value.kind {
	case TypeNumber:
		if profile.MinNumber == nil || value.numberValue < *profile.MinNumber {
			copy := value.numberValue
			profile.MinNumber = &copy
		}
		if profile.MaxNumber == nil || value.numberValue > *profile.MaxNumber {
			copy := value.numberValue
			profile.MaxNumber = &copy
		}
	case TypeTime:
		if profile.MinTime == nil || value.timeValue.Before(*profile.MinTime) {
			copy := value.timeValue
			profile.MinTime = &copy
		}
		if profile.MaxTime == nil || value.timeValue.After(*profile.MaxTime) {
			copy := value.timeValue
			profile.MaxTime = &copy
		}
	}
}

func (s *profileState) topValues(limit int) []ValueCount {
	if !s.captureTop || limit <= 0 || len(s.valueCounts) == 0 {
		return nil
	}
	values := make([]ValueCount, 0, len(s.valueCounts))
	for value, count := range s.valueCounts {
		values = append(values, ValueCount{Value: value, Count: count})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Count == values[j].Count {
			return values[i].Value < values[j].Value
		}
		return values[i].Count > values[j].Count
	})
	if len(values) > limit {
		values = values[:limit]
	}
	return values
}

func normalizedProfileLimits(value ProfileLimits) ProfileLimits {
	defaults := DefaultProfileLimits()
	if value.MaxFields <= 0 {
		value.MaxFields = defaults.MaxFields
	}
	if value.MaxRows <= 0 {
		value.MaxRows = defaults.MaxRows
	}
	if value.MaxDistinct <= 0 {
		value.MaxDistinct = defaults.MaxDistinct
	}
	if value.MaxTopValues <= 0 {
		value.MaxTopValues = defaults.MaxTopValues
	}
	if value.MaxValueBytes <= 0 {
		value.MaxValueBytes = defaults.MaxValueBytes
	}
	if value.MaxCellBytes <= 0 {
		value.MaxCellBytes = defaults.MaxCellBytes
	}
	value.MaxFields = minPositive(value.MaxFields, hardMaxProfileFields)
	value.MaxRows = minPositive(value.MaxRows, hardMaxProfileRows)
	value.MaxDistinct = minPositive(value.MaxDistinct, hardMaxProfileDistinct)
	value.MaxTopValues = minPositive(value.MaxTopValues, hardMaxProfileTopValues)
	value.MaxValueBytes = minPositive(value.MaxValueBytes, hardMaxProfileValueBytes)
	value.MaxCellBytes = minPositive(value.MaxCellBytes, hardMaxProfileCellBytes)
	return value
}

func minPositive(value, maximum int) int {
	if value > maximum {
		return maximum
	}
	return value
}

func profileValueKey(value typedValue, maxDisplayBytes int) (string, string) {
	var canonical string
	switch value.kind {
	case TypeString:
		canonical = value.stringValue
	case TypeNumber:
		if value.numberValue == 0 {
			canonical = "0"
		} else {
			canonical = fmt.Sprintf("%.17g", value.numberValue)
		}
	case TypeBool:
		if value.boolValue {
			canonical = "true"
		} else {
			canonical = "false"
		}
	case TypeTime:
		canonical = value.timeValue.UTC().Format(time.RFC3339Nano)
	default:
		canonical = "unknown"
	}
	digest := sha256.Sum256([]byte(string(value.kind) + "\x1f" + canonical))
	key := hex.EncodeToString(digest[:])
	if len(canonical) > maxDisplayBytes {
		return key, ""
	}
	return key, canonical
}
