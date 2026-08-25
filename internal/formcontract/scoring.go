package formcontract

import (
	"math"
	"strings"
)

type ScoreOutcome string

const (
	ScorePassed        ScoreOutcome = "PASS"
	ScoreFailed        ScoreOutcome = "FAIL"
	ScoreIndeterminate ScoreOutcome = "INDETERMINATE"
)

type ScoreRuleResult struct {
	FieldID  string       `json:"field_id"`
	Outcome  ScoreOutcome `json:"outcome"`
	Points   int          `json:"points"`
	Critical bool         `json:"critical,omitempty"`
}

type ScoreResult struct {
	Score            *float64          `json:"score,omitempty"`
	Coverage         float64           `json:"coverage"`
	CriticalFailures []ScoreRuleResult `json:"critical_failures,omitempty"`
	RuleResults      []ScoreRuleResult `json:"rule_results"`
}

func ScoreAnswers(fields []Scoring, answers map[string]AnswerValue) (ScoreResult, error) {
	var weightedPoints, totalWeight, requiredScored, answeredRequired int
	result := ScoreResult{RuleResults: make([]ScoreRuleResult, 0, len(fields))}
	for _, field := range fields {
		if len(field.AnswerScores) == 0 {
			continue
		}
		if strings.TrimSpace(field.ID) == "" || field.Weight < 1 || field.Weight > 100 {
			return ScoreResult{}, invalid("scored field id and weight from 1 to 100 are required")
		}
		if field.Required {
			requiredScored++
		}
		answer, answered := answers[field.ID]
		text, scalar := answer.ScalarText()
		if !answered || !scalar || text == "" {
			result.RuleResults = append(result.RuleResults, ScoreRuleResult{FieldID: field.ID, Outcome: ScoreIndeterminate})
			continue
		}
		points, ok := field.AnswerScores[text]
		if !ok || points < 0 || points > 100 {
			return ScoreResult{}, invalid("field %s answer is not an allowed scored choice", field.ID)
		}
		if field.Required {
			answeredRequired++
		}
		critical := slicesContains(field.CriticalAnswers, text)
		rule := ScoreRuleResult{FieldID: field.ID, Outcome: ScorePassed, Points: points, Critical: critical}
		if points > 0 || critical {
			rule.Outcome = ScoreFailed
		}
		result.RuleResults = append(result.RuleResults, rule)
		if critical {
			result.CriticalFailures = append(result.CriticalFailures, rule)
		}
		weightedPoints += points * field.Weight
		totalWeight += field.Weight
	}
	if totalWeight == 0 {
		return ScoreResult{}, invalid("at least one scored form field is required")
	}
	result.Coverage = 1
	if requiredScored > 0 {
		result.Coverage = float64(answeredRequired) / float64(requiredScored)
	}
	if result.Coverage < 1 {
		return result, nil
	}
	score := math.Round((float64(weightedPoints)/float64(totalWeight))*100) / 100
	result.Score = &score
	return result, nil
}

func slicesContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
