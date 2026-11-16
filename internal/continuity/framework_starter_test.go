package continuity

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"testing"
	"time"
)

func starterFixture() []StarterRequirement {
	return []StarterRequirement{{Code: "ACCESS", Title: "Review privileged digital banking access", Statement: "Review administrator access and retain sign-off.", Version: 1, FormID: "form-1", FormVersion: 1, FormApproved: true, GroupID: "access", Fields: []formcontract.Field{{ID: "date", Type: formcontract.TypeDate}, {ID: "record", Type: formcontract.TypeFile}}}}
}
func TestStarterConfigurationRejectsForgedLinksAndPreservesSource(t *testing.T) {
	source := starterFixture()
	input := StarterConfiguration{OwnerID: "owner", RespondentScopeIDs: []string{"department"}, Requirements: starterFixture()}
	input.Requirements[0].Decision = "TAILOR"
	input.Requirements[0].Statement = "Review mobile and internet banking administrators."
	input.Requirements[0].Reason = "Digital banking scope"
	saved, err := prepareStarterConfiguration(source, input, "owner", false)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Requirements[0].OriginalStatement != source[0].Statement || source[0].Statement == saved.Requirements[0].Statement {
		t.Fatal("Program edit changed source or lost original")
	}
	for _, mutate := range []func(*StarterConfiguration){func(c *StarterConfiguration) { c.Requirements[0].GroupID = "other" }, func(c *StarterConfiguration) { c.Requirements[0].FormVersion = 2 }, func(c *StarterConfiguration) { c.Requirements[0].Version = 2 }, func(c *StarterConfiguration) { c.OwnerID = "attacker" }, func(c *StarterConfiguration) { c.Requirements[0].Reason = "" }, func(c *StarterConfiguration) { c.Requirements[0].Decision = "KEEP" }} {
		v := input
		v.Requirements = append([]StarterRequirement{}, input.Requirements...)
		mutate(&v)
		if _, err := prepareStarterConfiguration(source, v, "owner", false); err == nil {
			t.Fatal("forged configuration accepted")
		}
	}
	second := StarterConfiguration{OwnerID: "owner", RespondentScopeIDs: []string{"department"}, Requirements: starterFixture()}
	second.Requirements[0].Decision = "KEEP"
	next, err := prepareStarterConfiguration(source, second, "owner", false)
	if err != nil || next.Requirements[0].Statement != source[0].Statement {
		t.Fatal("second Program inherited first edit", err)
	}
	source[0].FormApproved = false
	if _, err := prepareStarterConfiguration(source, input, "owner", false); err == nil {
		t.Fatal("unapproved native form accepted")
	}
}

type starterLookupFixture struct{ fixtureEditionLookup }

func (starterLookupFixture) StarterRequirements(context.Context, string, string, string) ([]StarterRequirement, error) {
	return starterFixture(), nil
}
func (starterLookupFixture) ValidateStarterRespondents(context.Context, string, string, []string) error {
	return nil
}
func (starterLookupFixture) ValidateStarterSnapshot(context.Context, string, string, StarterConfiguration) error {
	return nil
}
func TestStarterDraftSubmitApproveAndReconstruct(t *testing.T) {
	ctx := WithTrustedSystemScope(context.Background())
	s := NewService(NewMemoryRepository())
	now := time.Date(2026, 11, 14, 10, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	p := createProgramForCalculationPolicy(t, s, ctx, now)
	s.ConfigureFrameworkEditionLookup(starterLookupFixture{})
	config := StarterConfiguration{OwnerID: p.Program.OwnerPrincipalID, RespondentScopeIDs: []string{"department"}, Requirements: starterFixture()}
	config.Requirements[0].Decision = "KEEP"
	input := ProposeFrameworkAdoptionInput{TenantID: "bank", ProgramID: p.Program.ID, ExpectedVersion: p.Program.Version, ActorID: p.Program.OwnerPrincipalID, Rationale: "Review access within the service boundary", TemplateCode: "ISO_27001_2022", Configuration: &config}
	draft, err := s.SaveFrameworkAdoptionDraft(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if draft.FrameworkAdoptions[0].Configuration.Requirements[0].FormVersion != 1 {
		t.Fatal("version pin missing")
	}
	input.ExpectedVersion = draft.Program.Version
	proposed, err := s.ProposeFrameworkAdoption(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ApproveFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{TenantID: "bank", ProgramID: p.Program.ID, ExpectedVersion: proposed.Program.Version, ActorID: p.Program.OwnerPrincipalID, AdoptionID: proposed.FrameworkAdoptions[0].ID, Rationale: "Self approve"})
	if err == nil {
		t.Fatal("self approval accepted")
	}
	approved, err := s.ApproveFrameworkAdoption(ctx, DecideFrameworkAdoptionInput{TenantID: "bank", ProgramID: p.Program.ID, ExpectedVersion: proposed.Program.Version, ActorID: p.Program.AuthorityPrincipalID, AdoptionID: proposed.FrameworkAdoptions[0].ID, Rationale: "Independent review of scope and questions"})
	if err != nil {
		t.Fatal(err)
	}
	if len(approved.Requirements) != len(p.Requirements)+1 || approved.FrameworkAdoptions[0].Status != FrameworkAdoptionActive {
		t.Fatal("approved starter requirements not materialized")
	}
	got, err := s.GetProgram(ctx, "bank", p.Program.ID)
	if err != nil || got.FrameworkAdoptions[0].Configuration.Requirements[0].FormVersion != 1 {
		t.Fatal("readback lost version pin", err)
	}
}
