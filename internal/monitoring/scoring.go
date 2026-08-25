package monitoring

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func EvaluateForm(fields []FormField, answers map[string]formcontract.AnswerValue, thresholds Thresholds) (Evaluation, error) {
	if err := validateThresholds(thresholds); err != nil {
		return Evaluation{}, err
	}
	scored, err := formcontract.ScoreAnswers(fields, answers)
	if err != nil {
		return Evaluation{}, err
	}
	result := Evaluation{Score: scored.Score, Band: RiskNotAssessed, Coverage: scored.Coverage, RuleResults: make([]RuleResult, 0, len(scored.RuleResults))}
	for _, scoredRule := range scored.RuleResults {
		rule := RuleResult{FieldID: scoredRule.FieldID, Outcome: RuleOutcome(scoredRule.Outcome), Points: scoredRule.Points, Critical: scoredRule.Critical, Reason: "Answer evaluated against the active form rule."}
		if scoredRule.Outcome == formcontract.ScoreIndeterminate {
			rule.Reason = "Required answer is missing."
		}
		result.RuleResults = append(result.RuleResults, rule)
		if rule.Critical {
			result.CriticalFailures = append(result.CriticalFailures, rule)
		}
	}
	if result.Score != nil {
		result.Band = bandFor(*result.Score, thresholds)
	}
	if len(result.CriticalFailures) > 0 {
		result.Band = RiskCritical
	}
	return result, nil
}

func EvaluateSource(rules []SourceRule, resolution evidence.SourceResolution, thresholds Thresholds, now time.Time) (Evaluation, error) {
	if err := validateThresholds(thresholds); err != nil {
		return Evaluation{}, err
	}
	result := Evaluation{Band: RiskNotAssessed}
	if resolution.State != evidence.SourceResolutionCurrent || resolution.Receipt == nil || resolution.Receipt.Completeness != sourceaccess.CompletenessComplete || len(resolution.Records) != 1 {
		return result, nil
	}
	if len(rules) == 0 {
		return Evaluation{}, fmt.Errorf("at least one source rule is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var totalPoints, evaluated int
	result.RuleResults = make([]RuleResult, 0, len(rules))
	for _, rule := range rules {
		if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.Field) == "" || rule.RiskPoints < 0 || rule.RiskPoints > 100 {
			return Evaluation{}, fmt.Errorf("source rule id, field and risk points from 0 to 100 are required")
		}
		scalar, ok := resolution.Records[0][rule.Field]
		if !ok || scalar.Kind == sourceaccess.ScalarNull {
			result.RuleResults = append(result.RuleResults, RuleResult{RuleID: rule.ID, FieldID: rule.Field, Outcome: RuleIndeterminate, Critical: rule.Critical, Reason: "Required source field is missing."})
			continue
		}
		passed, err := compareScalar(scalar, rule, now)
		if err != nil {
			return Evaluation{}, err
		}
		evaluated++
		ruleResult := RuleResult{RuleID: rule.ID, FieldID: rule.Field, Outcome: RulePassed, Critical: rule.Critical, Reason: "Source value met the active rule."}
		if !passed {
			ruleResult.Outcome = RuleFailed
			ruleResult.Points = rule.RiskPoints
			ruleResult.Reason = "Source value did not meet the active rule."
			totalPoints += rule.RiskPoints
			if rule.Critical {
				result.CriticalFailures = append(result.CriticalFailures, ruleResult)
			}
		}
		result.RuleResults = append(result.RuleResults, ruleResult)
	}
	result.Coverage = float64(evaluated) / float64(len(rules))
	if result.Coverage < 1 {
		return result, nil
	}
	score := roundScore(float64(totalPoints) / float64(len(rules)))
	result.Score = &score
	result.Band = bandFor(score, thresholds)
	if len(result.CriticalFailures) > 0 {
		result.Band = RiskCritical
	}
	return result, nil
}

func compareScalar(value sourceaccess.Scalar, rule SourceRule, now time.Time) (bool, error) {
	actual := strings.TrimSpace(value.Text)
	expected := strings.TrimSpace(rule.Expected)
	switch rule.Operator {
	case OperatorPresent:
		return actual != "", nil
	case OperatorEquals:
		return normalizedComparable(actual, value.Kind) == normalizedComparable(expected, value.Kind), nil
	case OperatorNotEquals:
		return normalizedComparable(actual, value.Kind) != normalizedComparable(expected, value.Kind), nil
	case OperatorMaxAgeMinutes:
		maximum, err := strconv.Atoi(expected)
		if err != nil || maximum < 0 {
			return false, fmt.Errorf("rule %s maximum age must be a non-negative integer", rule.ID)
		}
		observed, err := time.Parse(time.RFC3339, actual)
		if err != nil {
			return false, fmt.Errorf("rule %s source time is invalid", rule.ID)
		}
		return !observed.After(now) && now.Sub(observed) <= time.Duration(maximum)*time.Minute, nil
	case OperatorGreaterThan, OperatorGreaterOrEqual, OperatorLessThan, OperatorLessOrEqual:
		left, leftErr := strconv.ParseFloat(actual, 64)
		right, rightErr := strconv.ParseFloat(expected, 64)
		if leftErr != nil || rightErr != nil {
			return false, fmt.Errorf("rule %s requires numeric values", rule.ID)
		}
		switch rule.Operator {
		case OperatorGreaterThan:
			return left > right, nil
		case OperatorGreaterOrEqual:
			return left >= right, nil
		case OperatorLessThan:
			return left < right, nil
		default:
			return left <= right, nil
		}
	default:
		return false, fmt.Errorf("rule %s operator is unsupported", rule.ID)
	}
}

func validateThresholds(value Thresholds) error {
	if value.ModerateFrom < 1 || value.ModerateFrom > 100 || value.HighFrom <= value.ModerateFrom || value.HighFrom > 100 || value.CriticalFrom <= value.HighFrom || value.CriticalFrom > 100 {
		return fmt.Errorf("risk thresholds must increase within 1 to 100")
	}
	return nil
}

func bandFor(score float64, thresholds Thresholds) RiskBand {
	switch {
	case score >= float64(thresholds.CriticalFrom):
		return RiskCritical
	case score >= float64(thresholds.HighFrom):
		return RiskHigh
	case score >= float64(thresholds.ModerateFrom):
		return RiskModerate
	default:
		return RiskLow
	}
}

func normalizedComparable(value string, kind sourceaccess.ScalarKind) string {
	if kind == sourceaccess.ScalarBool {
		return strings.ToLower(value)
	}
	return value
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func roundScore(value float64) float64 {
	return math.Round(value*100) / 100
}
