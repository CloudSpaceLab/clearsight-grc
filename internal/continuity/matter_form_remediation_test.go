//go:build !postgres

package continuity

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestMatterFormApplicationIsAtomicExactAndIdempotent(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 9, 2, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	program, err := service.CreateProgram(ctx, CreateProgramInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "VENDOR", Name: "Vendor assurance", Type: "COMPLIANCE",
		OwningFunction: "Risk", OwnerPrincipalID: "owner-1", AuthorityPrincipalID: "reviewer-1", Scope: json.RawMessage(`{}`), EffectiveFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, CreateMatterInput{
		TenantID: "bank", LegalEntityID: "entity-a", Type: MatterVendorDeficiency, Priority: 3,
		Title: "Obtain current vendor evidence", Summary: "The vendor evidence required for review is missing.", Scope: json.RawMessage(`{}`),
		KnownFacts: json.RawMessage(`{"vendor":"Acme"}`), MissingFacts: json.RawMessage(`["ISO 27001 certificate"]`),
		ProgramID: program.Program.ID, OwnerPrincipalID: "owner-1", RequiredAuthority: "CONTROL_ASSURANCE",
	})
	if err != nil {
		t.Fatal(err)
	}
	matter, err = service.AddVerificationContract(ctx, AddVerificationContractInput{
		TenantID: "bank", MatterID: matter.Matter.ID, ExpectedVersion: matter.Matter.Version,
		ExpectedOutcome: "A reviewer confirms that the submitted ISO 27001 certificate is current.",
		Baseline:        json.RawMessage(`{"certificate":"missing"}`), Scope: json.RawMessage(`{"vendor":"Acme"}`),
		Threshold: json.RawMessage(`{"certificate":"current"}`), FailureResponse: "BLOCK_CLOSE", AuthorityPrincipalID: "reviewer-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	binding := MatterFormRemediationBinding{
		ID: "binding-1", TenantID: "bank", LegalEntityID: "entity-a", ProgramID: program.Program.ID, MatterID: matter.Matter.ID,
		MatterVersionAtBinding: matter.Matter.Version, FormTemplateID: "form-1", FormTemplateVersion: 3,
		Mappings:               []MatterFormFieldMapping{{FieldID: "iso-certificate", MissingItem: "ISO 27001 certificate", FactKey: "iso_27001_certificate"}},
		VerificationContractID: matter.VerificationContracts[0].ID, CreatedBy: "owner-1", CreatedAt: now, Version: 1,
	}
	if _, err := repo.CreateMatterFormBinding(ctx, binding); err != nil {
		t.Fatal(err)
	}
	command := MatterFormApplicationCommand{
		Binding: binding, Aggregate: matter, ExpectedMatterVersion: matter.Matter.Version,
		DistributionID: "distribution-1", ResponseRevisionID: "revision-1", ResponseRevision: 1, SubmissionID: "submission-1",
		Answers: map[string]formcontract.AnswerValue{"iso-certificate": formcontract.TextAnswer("Certificate ISO-2026")},
		ActorID: "reviewer-1", Rationale: "The submitted response supplies the exact certificate requested for this issue.", AppliedAt: now,
		ApplicationID: "application-1", EventID: "event-1", TimerID: "timer-1",
	}
	updated, application, err := repo.ApplyMatterFormApplication(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if application.MatterVersion != matter.Matter.Version+1 || updated.Matter.Version != matter.Matter.Version+1 {
		t.Fatalf("application versions = %#v matter=%d", application, updated.Matter.Version)
	}
	if string(updated.Matter.MissingFacts) != "[]" || string(updated.Matter.KnownFacts) != `{"iso_27001_certificate":"Certificate ISO-2026","vendor":"Acme"}` {
		t.Fatalf("matter facts were not updated atomically: known=%s missing=%s", updated.Matter.KnownFacts, updated.Matter.MissingFacts)
	}

	replayed, duplicate, err := repo.ApplyMatterFormApplication(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID != application.ID || replayed.Matter.Version != updated.Matter.Version {
		t.Fatalf("idempotent replay changed state: application=%#v matter=%d", duplicate, replayed.Matter.Version)
	}
	history, err := repo.MatterEvents(ctx, "bank", matter.Matter.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	if history[len(history)-1].Type != EventMatterFormApplied {
		t.Fatalf("last event = %q", history[len(history)-1].Type)
	}
}

func TestMatterFormBindingRejectsOverlappingMissingInformation(t *testing.T) {
	ctx := WithTrustedSystemScope(t.Context())
	repo := NewMemoryRepository()
	service := NewService(repo)
	now := time.Date(2026, 9, 2, 13, 0, 0, 0, time.UTC)
	program, err := service.CreateProgram(ctx, CreateProgramInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "P", Name: "Program", Type: "COMPLIANCE", OwningFunction: "Risk", Scope: json.RawMessage(`{}`), EffectiveFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	matter, err := service.CreateMatter(ctx, CreateMatterInput{TenantID: "bank", LegalEntityID: "entity-a", Type: MatterControlGap, Priority: 3, Title: "Issue", Summary: "Information is missing.", Scope: json.RawMessage(`{}`), MissingFacts: json.RawMessage(`["Owner confirmation"]`), ProgramID: program.Program.ID})
	if err != nil {
		t.Fatal(err)
	}
	first := MatterFormRemediationBinding{ID: "binding-a", TenantID: "bank", LegalEntityID: "entity-a", ProgramID: program.Program.ID, MatterID: matter.Matter.ID, MatterVersionAtBinding: matter.Matter.Version, FormTemplateID: "form-a", FormTemplateVersion: 1, Mappings: []MatterFormFieldMapping{{FieldID: "confirm", MissingItem: "Owner confirmation", FactKey: "owner_confirmation"}}, VerificationContractID: "contract-a", CreatedAt: now, Version: 1}
	if _, err := repo.CreateMatterFormBinding(ctx, first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.ID, second.FormTemplateID = "binding-b", "form-b"
	if _, err := repo.CreateMatterFormBinding(ctx, second); err != ErrMatterFormBindingInvalid {
		t.Fatalf("overlapping binding error = %v", err)
	}
}
