package formcontract

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	MaxSections      = 20
	MaxFields        = 200
	MaxChoices       = 50
	DefaultSectionID = "general"
	maxFieldIDLength = 80
	maxFieldLabel    = 200
	maxFieldHelp     = 1000
	maxSectionTitle  = 200
	maxSectionHelp   = 1000
	maxAttestation   = 1000
)

var ErrInvalid = errors.New("invalid form contract")

type PresentationMode string

const (
	PresentationClassic   PresentationMode = "CLASSIC"
	PresentationWizard    PresentationMode = "WIZARD"
	PresentationAutomatic PresentationMode = "AUTOMATIC"
)

type Presentation struct {
	DefaultMode     PresentationMode `json:"default_mode"`
	AllowModeSwitch bool             `json:"allow_mode_switch"`
}

type ScoringMode string

const (
	ScoringNone       ScoringMode = "NONE"
	ScoringRisk       ScoringMode = "RISK"
	ScoringCompliance ScoringMode = "COMPLIANCE"
)

type ScoreDirection string

const (
	DirectionHighIsPoor ScoreDirection = "HIGH_IS_POOR"
	DirectionLowIsPoor  ScoreDirection = "LOW_IS_POOR"
)

type MissingScoreBehaviour string

const (
	MissingIndeterminate MissingScoreBehaviour = "INDETERMINATE"
	MissingExclude       MissingScoreBehaviour = "EXCLUDE"
	MissingZero          MissingScoreBehaviour = "ZERO"
)

type ConcernBand string

const (
	ConcernLow      ConcernBand = "LOW"
	ConcernModerate ConcernBand = "MODERATE"
	ConcernHigh     ConcernBand = "HIGH"
	ConcernCritical ConcernBand = "CRITICAL"
)

type PredicateOperator string

const (
	PredicateEquals        PredicateOperator = "EQUALS"
	PredicateNotEquals     PredicateOperator = "NOT_EQUALS"
	PredicateIn            PredicateOperator = "IN"
	PredicateNotIn         PredicateOperator = "NOT_IN"
	PredicateContains      PredicateOperator = "CONTAINS"
	PredicateContainsAny   PredicateOperator = "CONTAINS_ANY"
	PredicateContainsAll   PredicateOperator = "CONTAINS_ALL"
	PredicateGreaterThan   PredicateOperator = "GREATER_THAN"
	PredicateGreaterEqual  PredicateOperator = "GREATER_OR_EQUAL"
	PredicateLessThan      PredicateOperator = "LESS_THAN"
	PredicateLessEqual     PredicateOperator = "LESS_OR_EQUAL"
	PredicateNumberBetween PredicateOperator = "NUMBER_BETWEEN"
	PredicateDateBefore    PredicateOperator = "DATE_BEFORE"
	PredicateDateOnOrAfter PredicateOperator = "DATE_ON_OR_AFTER"
	PredicateDateBetween   PredicateOperator = "DATE_BETWEEN"
	PredicateAnswered      PredicateOperator = "ANSWERED"
	PredicateUnanswered    PredicateOperator = "UNANSWERED"
	PredicateAnd           PredicateOperator = "AND"
	PredicateOr            PredicateOperator = "OR"
	PredicateNot           PredicateOperator = "NOT"
)

type Predicate struct {
	FieldID  string            `json:"field_id,omitempty"`
	Operator PredicateOperator `json:"operator"`
	Values   []string          `json:"values,omitempty"`
	Children []Predicate       `json:"children,omitempty"`
}

type ScoreContribution struct {
	ID             string                `json:"id"`
	Label          string                `json:"label"`
	Weight         int                   `json:"weight"`
	Predicate      Predicate             `json:"predicate"`
	MatchPoints    int                   `json:"match_points"`
	NonMatchPoints int                   `json:"non_match_points"`
	Missing        MissingScoreBehaviour `json:"missing"`
	Required       bool                  `json:"required,omitempty"`
}

type RuleEffectKind string

const (
	EffectContribution RuleEffectKind = "CONTRIBUTION"
	EffectFloor        RuleEffectKind = "FLOOR"
	EffectCap          RuleEffectKind = "CAP"
	EffectDisqualify   RuleEffectKind = "DISQUALIFY"
)

type RuleEffect struct {
	Kind   RuleEffectKind `json:"kind"`
	Value  int            `json:"value,omitempty"`
	Weight int            `json:"weight,omitempty"`
}

type ScoreRule struct {
	ID        string     `json:"id"`
	Label     string     `json:"label"`
	Predicate Predicate  `json:"predicate"`
	Effect    RuleEffect `json:"effect"`
}

type ScoreBandRange struct {
	Band    ConcernBand `json:"band"`
	From    int         `json:"from"`
	Through int         `json:"through"`
}

type ScoreProfile struct {
	Version       string              `json:"version"`
	Mode          ScoringMode         `json:"mode"`
	Direction     ScoreDirection      `json:"direction"`
	Contributions []ScoreContribution `json:"contributions"`
	Rules         []ScoreRule         `json:"rules,omitempty"`
	Bands         []ScoreBandRange    `json:"bands"`
}

type CollectionIntent string

const (
	IntentCapture             CollectionIntent = "CAPTURE"
	IntentConfirmOrCorrect    CollectionIntent = "CONFIRM_OR_CORRECT"
	IntentReplaceHeldDocument CollectionIntent = "REPLACE_HELD_DOCUMENT"
)

type BrowserCachePolicy string

const (
	BrowserCacheAllowed BrowserCachePolicy = "ALLOWED"
	BrowserCacheDenied  BrowserCachePolicy = "NO_BROWSER_CACHE"
)

type RecordTarget struct {
	Key                 string `json:"key"`
	RequiredSubjectType string `json:"required_subject_type"`
}

type ConditionOperator string

const (
	ConditionEquals    ConditionOperator = "EQUALS"
	ConditionNotEquals ConditionOperator = "NOT_EQUALS"
	ConditionIn        ConditionOperator = "IN"
	ConditionNotIn     ConditionOperator = "NOT_IN"
	ConditionAnswered  ConditionOperator = "ANSWERED"
)

type VisibilityCondition struct {
	FieldID  string            `json:"field_id"`
	Operator ConditionOperator `json:"operator"`
	Values   []string          `json:"values,omitempty"`
}

type Section struct {
	ID        string               `json:"id"`
	Title     string               `json:"title"`
	Help      string               `json:"help,omitempty"`
	Weight    int                  `json:"weight,omitempty"`
	Condition *VisibilityCondition `json:"condition,omitempty"`
}

type Constraints struct {
	MinLength         *int     `json:"min_length,omitempty"`
	MaxLength         *int     `json:"max_length,omitempty"`
	Minimum           *float64 `json:"minimum,omitempty"`
	Maximum           *float64 `json:"maximum,omitempty"`
	Step              *float64 `json:"step,omitempty"`
	DecimalPrecision  *int     `json:"decimal_precision,omitempty"`
	Currency          string   `json:"currency,omitempty"`
	MinDate           string   `json:"min_date,omitempty"`
	MaxDate           string   `json:"max_date,omitempty"`
	MinSelections     *int     `json:"min_selections,omitempty"`
	MaxSelections     *int     `json:"max_selections,omitempty"`
	MinFiles          *int     `json:"min_files,omitempty"`
	MaxFiles          *int     `json:"max_files,omitempty"`
	MaxFileBytes      *int64   `json:"max_file_bytes,omitempty"`
	MaxTotalFileBytes *int64   `json:"max_total_file_bytes,omitempty"`
}

type Type string

const (
	TypeShortText      Type = "short_text"
	TypeLongText       Type = "long_text"
	TypeEmail          Type = "email"
	TypeTelephone      Type = "telephone"
	TypeURL            Type = "url"
	TypeInteger        Type = "integer"
	TypeDecimal        Type = "decimal"
	TypePercentage     Type = "percentage"
	TypeCurrency       Type = "currency"
	TypeDate           Type = "date"
	TypeYesNo          Type = "yes_no"
	TypeSingleSelect   Type = "single_select"
	TypeMultiSelect    Type = "multi_select"
	TypeCheckbox       Type = "checkbox"
	TypeAttestation    Type = "attestation"
	TypeFile           Type = "file"
	TypePhoto          Type = "photo"
	TypeSignature      Type = "signature"
	TypeVendorDocument Type = "vendor_document"
)

type Scoring struct {
	ID              string         `json:"id,omitempty"`
	Required        bool           `json:"required,omitempty"`
	Weight          int            `json:"weight"`
	AnswerScores    map[string]int `json:"answer_scores"`
	CriticalAnswers []string       `json:"critical_answers,omitempty"`
}

type Field struct {
	ID                 string               `json:"id"`
	SectionID          string               `json:"section_id"`
	Label              string               `json:"label"`
	Type               Type                 `json:"type"`
	Required           bool                 `json:"required"`
	Description        string               `json:"description,omitempty"`
	Options            []string             `json:"options,omitempty"`
	AcceptedFormats    []string             `json:"accepted_formats,omitempty"`
	Attestation        string               `json:"attestation,omitempty"`
	Constraints        Constraints          `json:"constraints,omitempty"`
	Condition          *VisibilityCondition `json:"condition,omitempty"`
	Scoring            *Scoring             `json:"scoring,omitempty"`
	CollectionIntent   CollectionIntent     `json:"collection_intent,omitempty"`
	RecordTarget       *RecordTarget        `json:"record_target,omitempty"`
	BrowserCachePolicy BrowserCachePolicy   `json:"browser_cache_policy,omitempty"`
}

type Contract struct {
	Presentation Presentation  `json:"presentation"`
	ScoringMode  ScoringMode   `json:"scoring_mode,omitempty"`
	ScoreProfile *ScoreProfile `json:"score_profile,omitempty"`
	Sections     []Section     `json:"sections"`
	Fields       []Field       `json:"fields"`
}

type DocumentAnswer struct {
	ArtifactID   string `json:"artifact_id"`
	DocumentType string `json:"document_type"`
	Reference    string `json:"reference,omitempty"`
	IssuedBy     string `json:"issued_by,omitempty"`
	IssuedOn     string `json:"issued_on,omitempty"`
	ExpiresOn    string `json:"expires_on,omitempty"`
}

type AnswerValue struct {
	Text        *string         `json:"text,omitempty"`
	Values      []string        `json:"values,omitempty"`
	ArtifactIDs []string        `json:"artifact_ids,omitempty"`
	Document    *DocumentAnswer `json:"document,omitempty"`
}

func TextAnswer(value string) AnswerValue {
	return AnswerValue{Text: &value}
}

func TextAnswers(values map[string]string) map[string]AnswerValue {
	answers := make(map[string]AnswerValue, len(values))
	for key, value := range values {
		answers[key] = TextAnswer(value)
	}
	return answers
}

func (value AnswerValue) ScalarText() (string, bool) {
	if value.Text == nil {
		return "", false
	}
	return strings.TrimSpace(*value.Text), true
}

func (value AnswerValue) Answered() bool {
	if text, present := value.ScalarText(); present && text != "" {
		return true
	}
	return len(value.Values) > 0 || len(value.ArtifactIDs) > 0 || value.Document != nil
}

func (value *AnswerValue) UnmarshalJSON(data []byte) error {
	type answerValue AnswerValue
	var decoded answerValue
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*value = AnswerValue(decoded)
	return nil
}
