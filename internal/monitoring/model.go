package monitoring

type RiskBand string

const (
	RiskLow         RiskBand = "LOW"
	RiskModerate    RiskBand = "MODERATE"
	RiskHigh        RiskBand = "HIGH"
	RiskCritical    RiskBand = "CRITICAL"
	RiskNotAssessed RiskBand = "NOT_ASSESSED"
)

type Thresholds struct {
	ModerateFrom int `json:"moderate_from"`
	HighFrom     int `json:"high_from"`
	CriticalFrom int `json:"critical_from"`
}

func DefaultThresholds() Thresholds {
	return Thresholds{ModerateFrom: 25, HighFrom: 50, CriticalFrom: 75}
}

type FormField struct {
	ID              string         `json:"id"`
	Required        bool           `json:"required"`
	Weight          int            `json:"weight,omitempty"`
	AnswerScores    map[string]int `json:"answer_scores,omitempty"`
	CriticalAnswers []string       `json:"critical_answers,omitempty"`
}

type SourceOperator string

const (
	OperatorEquals         SourceOperator = "EQUALS"
	OperatorNotEquals      SourceOperator = "NOT_EQUALS"
	OperatorGreaterThan    SourceOperator = "GREATER_THAN"
	OperatorGreaterOrEqual SourceOperator = "GREATER_OR_EQUAL"
	OperatorLessThan       SourceOperator = "LESS_THAN"
	OperatorLessOrEqual    SourceOperator = "LESS_OR_EQUAL"
	OperatorPresent        SourceOperator = "PRESENT"
	OperatorMaxAgeMinutes  SourceOperator = "MAX_AGE_MINUTES"
)

type SourceRule struct {
	ID         string         `json:"id"`
	Field      string         `json:"field"`
	Operator   SourceOperator `json:"operator"`
	Expected   string         `json:"expected,omitempty"`
	RiskPoints int            `json:"risk_points"`
	Critical   bool           `json:"critical,omitempty"`
}

type RuleOutcome string

const (
	RulePassed        RuleOutcome = "PASS"
	RuleFailed        RuleOutcome = "FAIL"
	RuleIndeterminate RuleOutcome = "INDETERMINATE"
)

type RuleResult struct {
	RuleID   string      `json:"rule_id,omitempty"`
	FieldID  string      `json:"field_id"`
	Outcome  RuleOutcome `json:"outcome"`
	Points   int         `json:"points"`
	Critical bool        `json:"critical,omitempty"`
	Reason   string      `json:"reason"`
}

type Evaluation struct {
	Score            *float64     `json:"score,omitempty"`
	Band             RiskBand     `json:"band"`
	Coverage         float64      `json:"coverage"`
	CriticalFailures []RuleResult `json:"critical_failures,omitempty"`
	RuleResults      []RuleResult `json:"rule_results"`
}
