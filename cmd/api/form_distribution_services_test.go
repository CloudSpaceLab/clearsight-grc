package main

import (
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestFormDistributionReaderPreservesExactScoreProfile(t *testing.T) {
	repo := monitoring.NewMemoryRepository()
	profile := &formcontract.ScoreProfile{
		Version: "risk-v2", Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor,
		Bands: formcontract.DefaultConcernBands(), Contributions: []formcontract.ScoreContribution{{
			ID: "missing-cert", Label: "Certification", Predicate: formcontract.Predicate{FieldID: "certified", Operator: formcontract.PredicateEquals, Values: []string{"No"}},
			Weight: 1, MatchPoints: 100, Missing: formcontract.MissingZero,
		}},
	}
	_, err := repo.CreateFormRevision(t.Context(), monitoring.FormTemplate{
		ID: "form-a", TenantID: "bank-a", LegalEntityID: "entity-a", Code: "VENDOR", Name: "Vendor review", Purpose: "Review current evidence.", Sensitivity: "INTERNAL",
		ScoringMode: formcontract.ScoringRisk, ScoreProfile: profile, Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		Sections: []formcontract.Section{{ID: "risk", Title: "Risk"}}, Fields: []formcontract.Field{{ID: "certified", SectionID: "risk", Label: "Certified", Type: formcontract.TypeYesNo, Required: true}},
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 2, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()},
	})
	if err != nil {
		t.Fatal(err)
	}
	value, err := (formDistributionReader{repo: repo}).GetDistributionFormRevision(t.Context(), "bank-a", "entity-a", "form-a", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !value.Active || value.ScoringMode != formcontract.ScoringRisk || value.ScoreProfile == nil || value.ScoreProfile.Version != "risk-v2" {
		t.Fatalf("distribution form lost scoring contract: %#v", value)
	}
}
