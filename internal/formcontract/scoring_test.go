package formcontract

import (
	"errors"
	"testing"
)

func TestScoreAnswersUsesWeightsCoverageAndCriticalAnswers(t *testing.T) {
	result, err := ScoreAnswers([]Scoring{
		{ID: "access", Required: true, Weight: 9, AnswerScores: map[string]int{"Yes": 0, "No": 20}, CriticalAnswers: []string{"No"}},
		{ID: "logging", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
	}, map[string]AnswerValue{"access": TextAnswer("No"), "logging": TextAnswer("Yes")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Score == nil || *result.Score != 18 || result.Coverage != 1 || len(result.CriticalFailures) != 1 {
		t.Fatalf("unexpected result %#v", result)
	}
}

func TestScoreAnswersLeavesIncompleteRequiredResponsesUnscored(t *testing.T) {
	result, err := ScoreAnswers([]Scoring{
		{ID: "access", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
		{ID: "logging", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
	}, map[string]AnswerValue{"access": TextAnswer("Yes")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Score != nil || result.Coverage != .5 {
		t.Fatalf("unexpected incomplete result %#v", result)
	}
}

func TestScoreAnswersRejectsUnconfiguredAnswer(t *testing.T) {
	_, err := ScoreAnswers([]Scoring{{ID: "access", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}}, map[string]AnswerValue{"access": TextAnswer("Probably")})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected invalid answer, got %v", err)
	}
}
