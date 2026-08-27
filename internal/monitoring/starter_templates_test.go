package monitoring

import (
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestStarterCatalogProvidesReviewableVendorDueDiligenceDraft(t *testing.T) {
	starters, err := StarterTemplates()
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
		Sections: starter.Template.Sections, Fields: starter.Template.Fields,
	})
	if err != nil || contract.ScoringMode != formcontract.ScoringCompliance {
		t.Fatalf("starter contract invalid: %#v, %v", contract, err)
	}
}

func TestStarterCatalogRejectsUnknownCode(t *testing.T) {
	if _, err := StarterTemplateByCode("UNKNOWN"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown starter error = %v", err)
	}
}
