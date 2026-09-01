package monitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestStarterCatalogProvidesReviewableVendorDueDiligenceDraft(t *testing.T) {
	repo := NewMemoryRepository()
	repo.SeedStarterTemplates(starterFixture())
	starters, err := repo.ListStarterTemplates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(starters) != 1 {
		t.Fatalf("starter count = %d", len(starters))
	}
	starter := starters[0]
	if starter.Code != "VENDOR_DUE_DILIGENCE" || starter.CatalogVersion != 1 || starter.PublishedOn != "2026-08-27" || starter.ReferenceLabel == "" {
		t.Fatalf("unexpected starter metadata: %#v", starter)
	}
	if starter.Template.Status != LifecycleDraft || starter.Template.IsCurrent || starter.Template.StarterCatalogCode != starter.Code || starter.Template.StarterCatalogVersion != starter.CatalogVersion {
		t.Fatalf("starter is not an ordinary governed draft: %#v", starter.Template)
	}
	contract, err := formcontract.Normalize(formcontract.Contract{
		Presentation: starter.Template.Presentation, ScoringMode: starter.Template.ScoringMode,
		ScoreProfile: starter.Template.ScoreProfile, Sections: starter.Template.Sections, Fields: starter.Template.Fields,
	})
	if err != nil || contract.ScoringMode != formcontract.ScoringCompliance || contract.ScoreProfile == nil || contract.ScoreProfile.Version != "vendor-due-diligence-v1" {
		t.Fatalf("starter contract invalid: %#v, %v", contract, err)
	}
}

func TestStarterCatalogRejectsUnknownCode(t *testing.T) {
	repo := NewMemoryRepository()
	if _, err := repo.StarterTemplateByCode(context.Background(), "UNKNOWN"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown starter error = %v", err)
	}
}

func starterFixture() StarterTemplate {
	return StarterTemplate{Code: "VENDOR_DUE_DILIGENCE", CatalogVersion: 1, PublishedOn: "2026-08-27", ReferenceLabel: "Reference data for review.", Template: FormTemplate{
		Code: "VENDOR_DUE_DILIGENCE", Name: "Vendor due diligence review", Purpose: "Collect current vendor evidence.", Sensitivity: "CONFIDENTIAL", ScoringMode: formcontract.ScoringCompliance,
		ScoreProfile: &formcontract.ScoreProfile{Version: "vendor-due-diligence-v1", Mode: formcontract.ScoringCompliance, Direction: formcontract.DirectionLowIsPoor, Contributions: []formcontract.ScoreContribution{{ID: "security-policy", Weight: 100, Predicate: formcontract.Predicate{FieldID: "security_policy", Operator: formcontract.PredicateEquals, Values: []string{"Yes"}}, MatchPoints: 100, Missing: formcontract.MissingIndeterminate}}, Bands: formcontract.DefaultConcernBands()},
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationWizard}, Sections: []formcontract.Section{{ID: "operations", Title: "Operating safeguards", Weight: 100}},
		Fields:             []TemplateField{{ID: "security_policy", SectionID: "operations", Label: "Security policy is current", Type: formcontract.TypeYesNo, Required: true, Scoring: &formcontract.Scoring{Weight: 100, AnswerScores: map[string]int{"Yes": 100, "No": 0}}}},
		StarterCatalogCode: "VENDOR_DUE_DILIGENCE", StarterCatalogVersion: 1, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1},
	}}
}
