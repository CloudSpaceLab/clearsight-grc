package thirdparty

import (
	"errors"
	"testing"
	"time"
)

func TestProgramVendorExpectationLinksRequireScopedActiveMembership(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	created := now.Add(-time.Hour)
	ended := now.Add(-time.Minute)
	expected := []ProgramVendorAssessmentExpectation{{
		Key: "vendor-a:security", RelationshipID: "vendor-a",
		FormTemplateID: "security", FormTemplateVersion: 1,
		ReviewKind: AssessmentReviewPeriodic, Weight: 1,
	}}
	link := RelationshipLink{
		TenantID: "bank", LegalEntityID: "entity", RelationshipID: "vendor-a",
		TargetType: LinkTargetProgram, TargetID: "program", State: RelationshipLinkActive,
		CreatedAt: created,
	}
	verify := func(links []RelationshipLink, complete bool, at time.Time) error {
		return ValidateProgramVendorExpectationLinks("bank", "entity", "program", expected, links, complete, at)
	}
	if err := verify([]RelationshipLink{link}, true, now); err != nil {
		t.Fatal(err)
	}
	if err := verify([]RelationshipLink{link}, false, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("partial link page authorized vendor score: %v", err)
	}
	if err := verify(nil, true, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unlinked vendor authorized score: %v", err)
	}
	link.TargetID = "another-program"
	if err := verify([]RelationshipLink{link}, true, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another Program's vendor authorized score: %v", err)
	}
	link.TargetID = "program"
	link.LegalEntityID = "other-entity"
	if err := verify([]RelationshipLink{link}, true, now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-entity link authorized score: %v", err)
	}
	link.LegalEntityID = "entity"
	link.State, link.EndedAt = RelationshipLinkEnded, &ended
	if err := verify([]RelationshipLink{link}, true, ended.Add(-time.Second)); err != nil {
		t.Fatalf("historically valid link was rejected: %v", err)
	}
	if err := verify([]RelationshipLink{link}, true, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ended relationship still affected today's Program score: %v", err)
	}
	if err := verify([]RelationshipLink{link}, true, created.Add(-time.Second)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("future-created link entered past Program score: %v", err)
	}
}
