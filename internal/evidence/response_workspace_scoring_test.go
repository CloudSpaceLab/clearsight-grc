package evidence

import (
	"encoding/json"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestBuildResponseRevisionUsesArrayForEmptyCriticalResults(t *testing.T) {
	request := Request{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		Sections:     []formcontract.Section{{ID: "general", Title: "General"}},
		Fields: []Field{{
			ID: "answer", SectionID: "general", Label: "Answer", Type: string(formcontract.TypeShortText),
		}},
	}

	revision, err := buildResponseRevision(request, AssuranceEmailVerified, nil, map[string]formcontract.AnswerValue{
		"answer": formcontract.TextAnswer("confirmed"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision.CriticalFieldResults == nil || len(revision.CriticalFieldResults) != 0 {
		t.Fatalf("empty critical results must be a non-nil empty slice: %#v", revision.CriticalFieldResults)
	}

	encoded, err := json.Marshal(revision.CriticalFieldResults)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != "[]" {
		t.Fatalf("empty critical results must persist as a JSON array, got %s", encoded)
	}
}

func TestBuildResponseRevisionStoresGeneralizedRiskScoreFromPinnedProfile(t *testing.T) {
	request := Request{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		ScoringMode:  formcontract.ScoringRisk,
		ScoreProfile: &formcontract.ScoreProfile{
			Version: "risk-v2", Mode: formcontract.ScoringRisk,
			Bands: formcontract.DefaultConcernBands(),
			Contributions: []formcontract.ScoreContribution{{
				ID: "control-score", Weight: 100,
				Predicate:   formcontract.Predicate{FieldID: "control", Operator: formcontract.PredicateEquals, Values: []string{"No"}},
				MatchPoints: 100, Missing: formcontract.MissingIndeterminate,
			}},
		},
		Sections: []formcontract.Section{{ID: "general", Title: "General"}},
		Fields:   []Field{{ID: "control", SectionID: "general", Label: "Control operating", Type: string(formcontract.TypeYesNo), Required: true, Options: []string{"Yes", "No"}}},
	}
	revision, err := buildResponseRevision(request, AssuranceEmailVerified, nil, formcontract.TextAnswers(map[string]string{"control": "No"}))
	if err != nil {
		t.Fatal(err)
	}
	if revision.Score == nil || revision.Score.Mode != formcontract.ScoringRisk || revision.Score.RawScore == nil || *revision.Score.RawScore != 100 || revision.Score.AdverseScore == nil || *revision.Score.AdverseScore != 100 || revision.Score.Band != formcontract.ConcernCritical {
		t.Fatalf("revision score = %#v", revision.Score)
	}
}

func TestBuildResponseRevisionKeepsValidResponseWhenPinnedScoringFails(t *testing.T) {
	request := Request{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		ScoringMode:  formcontract.ScoringRisk,
		ScoreProfile: &formcontract.ScoreProfile{
			Version: "broken-risk-v2", Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionLowIsPoor,
			// A risk profile cannot reverse the governed adverse-score direction.
			Contributions: []formcontract.ScoreContribution{{
				ID: "control-score", Weight: 100,
				Predicate:   formcontract.Predicate{FieldID: "control", Operator: formcontract.PredicateEquals, Values: []string{"No"}},
				MatchPoints: 100, Missing: formcontract.MissingIndeterminate,
			}},
		},
		Sections: []formcontract.Section{{ID: "general", Title: "General"}},
		Fields:   []Field{{ID: "control", SectionID: "general", Label: "Control operating", Type: string(formcontract.TypeYesNo), Required: true, Options: []string{"Yes", "No"}}},
	}

	revision, err := buildResponseRevision(request, AssuranceEmailVerified, nil, formcontract.TextAnswers(map[string]string{"control": "No"}))
	if err != nil {
		t.Fatalf("a scoring configuration failure must not discard a valid response: %v", err)
	}
	if revision.Score == nil || revision.Score.State != ResponseScoreFailed || revision.Score.FailureCode != "SCORE_CONFIGURATION_INVALID" {
		t.Fatalf("revision score = %#v", revision.Score)
	}
}

func TestCloneResponseRevisionDeepCopiesGeneralizedScore(t *testing.T) {
	raw := 82.0
	original := ResponseRevision{Score: &ResponseScoreResult{
		RawScore:            &raw,
		ContributionResults: []formcontract.ContributionResult{{ID: "control", Points: 82}},
		RuleResults:         []formcontract.AdvancedRuleResult{{ID: "critical-control", Matched: true}},
	}}
	cloned := cloneResponseRevision(original)
	*cloned.Score.RawScore = 12
	cloned.Score.ContributionResults[0].Points = 12
	cloned.Score.RuleResults[0].Matched = false
	if *original.Score.RawScore != 82 || original.Score.ContributionResults[0].Points != 82 || !original.Score.RuleResults[0].Matched {
		t.Fatalf("clone mutated original score: %#v", original.Score)
	}
}

func TestBuildResponseRevisionGeneralizesLegacyRiskComplianceAndUnscoredModes(t *testing.T) {
	base := Request{
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		Sections:     []formcontract.Section{{ID: "general", Title: "General"}},
		Fields: []Field{{
			ID: "control", SectionID: "general", Label: "Control operating", Type: string(formcontract.TypeYesNo), Required: true, Options: []string{"Yes", "No"},
			Scoring: &formcontract.Scoring{Weight: 100, AnswerScores: map[string]int{"Yes": 0, "No": 100}},
		}},
	}

	risk := base
	risk.ScoringMode = formcontract.ScoringRisk
	riskRevision, err := buildResponseRevision(risk, AssuranceEmailVerified, nil, formcontract.TextAnswers(map[string]string{"control": "No"}))
	if err != nil {
		t.Fatal(err)
	}
	if riskRevision.Score == nil || riskRevision.Score.State != ResponseScoreFinal || riskRevision.Score.Band != formcontract.ConcernCritical || riskRevision.Score.AdverseScore == nil || *riskRevision.Score.AdverseScore != 100 {
		t.Fatalf("legacy risk score = %#v", riskRevision.Score)
	}

	compliance := base
	compliance.ScoringMode = formcontract.ScoringCompliance
	compliance.Sections = append([]formcontract.Section(nil), base.Sections...)
	compliance.Sections[0].Weight = 100
	compliance.Fields = append([]Field(nil), base.Fields...)
	compliance.Fields[0].Scoring = &formcontract.Scoring{Weight: 100, AnswerScores: map[string]int{"Yes": 100, "No": 0}}
	complianceRevision, err := buildResponseRevision(compliance, AssuranceEmailVerified, nil, formcontract.TextAnswers(map[string]string{"control": "No"}))
	if err != nil {
		t.Fatal(err)
	}
	if complianceRevision.Score == nil || complianceRevision.Score.State != ResponseScoreFinal || complianceRevision.Score.Band != formcontract.ConcernCritical || complianceRevision.Score.RawScore == nil || *complianceRevision.Score.RawScore != 0 || complianceRevision.Score.AdverseScore == nil || *complianceRevision.Score.AdverseScore != 100 {
		t.Fatalf("legacy compliance score = %#v", complianceRevision.Score)
	}

	unscored := base
	unscored.ScoringMode = formcontract.ScoringNone
	unscored.Fields = append([]Field(nil), base.Fields...)
	unscored.Fields[0].Scoring = nil
	unscoredRevision, err := buildResponseRevision(unscored, AssuranceEmailVerified, nil, formcontract.TextAnswers(map[string]string{"control": "No"}))
	if err != nil {
		t.Fatal(err)
	}
	if unscoredRevision.Score == nil || unscoredRevision.Score.State != ResponseScoreNotConfigured || unscoredRevision.Score.RawScore != nil {
		t.Fatalf("unscored result = %#v", unscoredRevision.Score)
	}
}
