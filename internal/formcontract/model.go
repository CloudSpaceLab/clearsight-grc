package formcontract

import (
	"bytes"
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
	Condition *VisibilityCondition `json:"condition,omitempty"`
}

type Constraints struct {
	MinLength        *int     `json:"min_length,omitempty"`
	MaxLength        *int     `json:"max_length,omitempty"`
	Minimum          *float64 `json:"minimum,omitempty"`
	Maximum          *float64 `json:"maximum,omitempty"`
	Step             *float64 `json:"step,omitempty"`
	DecimalPrecision *int     `json:"decimal_precision,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	MinDate          string   `json:"min_date,omitempty"`
	MaxDate          string   `json:"max_date,omitempty"`
	MinSelections    *int     `json:"min_selections,omitempty"`
	MaxSelections    *int     `json:"max_selections,omitempty"`
	MaxFiles         *int     `json:"max_files,omitempty"`
	MaxFileBytes     *int64   `json:"max_file_bytes,omitempty"`
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
	ID              string               `json:"id"`
	SectionID       string               `json:"section_id"`
	Label           string               `json:"label"`
	Type            Type                 `json:"type"`
	Required        bool                 `json:"required"`
	Description     string               `json:"description,omitempty"`
	Options         []string             `json:"options,omitempty"`
	AcceptedFormats []string             `json:"accepted_formats,omitempty"`
	Attestation     string               `json:"attestation,omitempty"`
	Constraints     Constraints          `json:"constraints,omitempty"`
	Condition       *VisibilityCondition `json:"condition,omitempty"`
	Scoring         *Scoring             `json:"scoring,omitempty"`
}

type Contract struct {
	Presentation Presentation `json:"presentation"`
	Sections     []Section    `json:"sections"`
	Fields       []Field      `json:"fields"`
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

// UnmarshalJSON accepts legacy scalar answers while all newly written answers use
// the structured representation. This keeps exact historical submissions readable.
func (value *AnswerValue) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		*value = AnswerValue{}
		return nil
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		*value = TextAnswer(text)
		return nil
	}
	type answerValue AnswerValue
	var decoded answerValue
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*value = AnswerValue(decoded)
	return nil
}
