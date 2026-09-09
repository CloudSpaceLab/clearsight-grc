package evidence

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestVendorSubmittedFormAttentionKeepsGapsSeparateFromCompletion(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	req := Request{ID: "request", SubjectID: "vendor", Status: RequestSubmitted, Deadline: now.Add(-time.Hour), ScoringMode: formcontract.ScoringRisk, Sections: []formcontract.Section{{ID: "checks", Title: "Evidence checks"}}, Fields: []Field{
		{ID: "assurance", SectionID: "checks", Label: "Current security assurance", Type: "yes_no", Required: true, Scoring: &formcontract.Scoring{ID: "assurance", Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}},
		{ID: "certificate", SectionID: "checks", Label: "ISO 27001 certificate", Type: "vendor_document", Required: true},
		{ID: "missing", SectionID: "checks", Label: "Security testing report", Type: "file", Required: true},
		{ID: "neutral", SectionID: "checks", Label: "Uses subcontractors", Type: "yes_no"},
	}}
	answers := formcontract.TextAnswers(map[string]string{"assurance": "No", "neutral": "No"})
	answers["certificate"] = formcontract.AnswerValue{Document: &formcontract.DocumentAnswer{ArtifactID: "certificate", ExpiresOn: "2026-09-08"}}
	row := vendorFormRow(req, answers, true, &ResponseRevision{ID: "response", Current: true}, now)
	data, _ := json.Marshal(row)
	var result map[string]any
	_ = json.Unmarshal(data, &result)
	items, ok := result["attention_items"].([]any)
	if !ok || len(items) != 3 {
		t.Fatalf("expected missing evidence, expired certificate and configured gap: %s", data)
	}
	if row.ResponseState != "SUBMITTED" || vendorFormOutstanding(row) {
		t.Fatalf("admitted gaps changed submission: %+v", row)
	}
	if result["outdated"] != true {
		t.Fatalf("expired response not outdated: %s", data)
	}
}

func TestVendorFormFreshnessNeverUsesRequestDeadlineOrUnknownExpiry(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	req := Request{ID: "request", Status: RequestSubmitted, Deadline: now.Add(-24 * time.Hour), Sections: []formcontract.Section{{ID: "docs", Title: "Documents"}}, Fields: []Field{{ID: "certificate", SectionID: "docs", Label: "Certificate", Type: "vendor_document", Required: true}}}
	for _, tc := range []struct {
		expiry string
		want   any
	}{{"2026-09-09", false}, {"2026-09-10", false}, {"", nil}, {"not-a-date", nil}, {"2026-09-08", true}} {
		answers := map[string]formcontract.AnswerValue{"certificate": {Document: &formcontract.DocumentAnswer{ArtifactID: "artifact", ExpiresOn: tc.expiry}}}
		row := vendorFormRow(req, answers, true, &ResponseRevision{ID: "response", Current: true}, now)
		data, _ := json.Marshal(row)
		var got map[string]any
		_ = json.Unmarshal(data, &got)
		if got["outdated"] != tc.want {
			t.Fatalf("expiry %q: got %v want %v: %s", tc.expiry, got["outdated"], tc.want, data)
		}
	}
}

func TestVendorAttentionUsesConfiguredConcernThresholdAndKeepsAutomaticCriticalFinding(t *testing.T) {
	now := time.Now().UTC()
	req := Request{Status: RequestSubmitted, ScoringMode: formcontract.ScoringRisk, Sections: []formcontract.Section{{ID: "checks", Title: "Checks"}}, Fields: []Field{{ID: "current", SectionID: "checks", Label: "Certificate current", Type: "yes_no", Required: true}}, ScoreProfile: &formcontract.ScoreProfile{Version: "configured-v1", Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor, Bands: []formcontract.ScoreBandRange{{Band: formcontract.ConcernLow, From: 0, Through: 49}, {Band: formcontract.ConcernModerate, From: 50, Through: 69}, {Band: formcontract.ConcernHigh, From: 70, Through: 89}, {Band: formcontract.ConcernCritical, From: 90, Through: 100}}, Contributions: []formcontract.ScoreContribution{{ID: "current", Label: "Certificate condition", Weight: 1, Predicate: formcontract.Predicate{FieldID: "current", Operator: formcontract.PredicateEquals, Values: []string{"Yes"}}, MatchPoints: 20, NonMatchPoints: 70, Missing: formcontract.MissingIndeterminate}}, Rules: []formcontract.ScoreRule{{ID: "critical", Label: "Required certification absent", Predicate: formcontract.Predicate{Operator: formcontract.PredicateOr, Children: []formcontract.Predicate{{FieldID: "current", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, {FieldID: "current", Operator: formcontract.PredicateEquals, Values: []string{"Yes"}}}}, Effect: formcontract.RuleEffect{Kind: formcontract.EffectDisqualify}}}}}
	row := VendorFormRow{ResponseState: "SUBMITTED", Score: &ResponseScoreResult{State: ResponseScoreFinal, Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor, ContributionResults: []formcontract.ContributionResult{{ID: "current", Points: 20}}, RuleResults: []formcontract.AdvancedRuleResult{{ID: "critical", Matched: true, Effect: formcontract.EffectDisqualify}}}, AssessedScore: &ResponseScoreResult{State: ResponseScoreFinal, Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor, ContributionResults: []formcontract.ContributionResult{{ID: "current", Points: 0}}}}
	if _, err := workspaceScoringContract(req); err != nil {
		t.Fatal(err)
	}
	populateVendorFormAttention(req, formcontract.TextAnswers(map[string]string{"current": "Yes"}), true, &row, nil, now)
	if len(row.AttentionItems) != 1 || row.AttentionItems[0].RuleID != "critical" || row.AttentionItems[0].Source != "RESPONSE" || row.AttentionItems[0].FieldID != "" {
		t.Fatalf("configured low contribution was treated as failure or critical finding lost: %+v", row.AttentionItems)
	}
}

func TestVendorAttentionRuleEffectsUseAdverseFloorAndRawContribution(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name   string
		effect formcontract.RuleEffectKind
		value  int
		want   bool
	}{
		{"compliance floor", formcontract.EffectFloor, 100, true},
		{"low floor", formcontract.EffectFloor, 10, false},
		{"favourable cap", formcontract.EffectCap, 0, false},
		{"cap alone", formcontract.EffectCap, 70, false},
		{"failed raw contribution", formcontract.EffectContribution, 0, true},
		{"passing raw contribution", formcontract.EffectContribution, 100, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := Request{Status: RequestSubmitted, ScoringMode: formcontract.ScoringCompliance, Sections: []formcontract.Section{{ID: "checks", Title: "Checks"}}, Fields: []Field{{ID: "current", SectionID: "checks", Label: "Certificate current", Type: "yes_no"}}, ScoreProfile: &formcontract.ScoreProfile{Version: "custom-v1", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, Bands: formcontract.DefaultConcernBands(), Contributions: []formcontract.ScoreContribution{{ID: "base", Label: "Base check", Weight: 1, Predicate: formcontract.Predicate{FieldID: "current", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, MatchPoints: 100, NonMatchPoints: 100, Missing: formcontract.MissingIndeterminate}}, Rules: []formcontract.ScoreRule{{ID: "configured", Label: "Required certificate", Predicate: formcontract.Predicate{FieldID: "current", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, Effect: formcontract.RuleEffect{Kind: tc.effect, Value: tc.value}}}}}
			if tc.effect == formcontract.EffectContribution {
				req.ScoreProfile.Rules[0].Effect.Weight = 1
			}
			revision, err := buildResponseRevision(req, AccessAssurance(""), nil, formcontract.TextAnswers(map[string]string{"current": "No"}))
			if err != nil {
				t.Fatal(err)
			}
			row := vendorFormRow(req, formcontract.TextAnswers(map[string]string{"current": "No"}), true, &revision, now)
			if (len(row.AttentionItems) > 0) != tc.want {
				t.Fatalf("attention %+v score %+v want gap %v", row.AttentionItems, row.Score, tc.want)
			}
		})
	}
}
