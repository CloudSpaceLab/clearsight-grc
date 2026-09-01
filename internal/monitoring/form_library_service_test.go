package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type formAuthorityStub struct {
	principal string
	err       error
}

func (s formAuthorityStub) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	if s.err != nil {
		return authority.Resolution{}, s.err
	}
	return authority.Resolution{Principal: authority.Principal{ID: s.principal}, RuleID: "form-rule", PolicyVersion: "v1"}, nil
}

func (formAuthorityStub) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, nil
}
func (formAuthorityStub) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, nil
}
func (formAuthorityStub) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, nil
}

func formActorContext(tenantID, legalEntityID, principalID string) context.Context {
	now := time.Now().UTC()
	return identity.WithActor(context.Background(), identity.Actor{
		TenantID: tenantID, LegalEntityID: legalEntityID, PrincipalID: principalID,
		Kind: "PERSON", AuthenticationMethod: "TEST", AssuranceLevel: "HIGH",
		SessionID: "session-" + principalID, IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	})
}

func libraryService(t *testing.T, repo Repository, principalID string) *Service {
	t.Helper()
	guard, err := commandauth.New(formAuthorityStub{principal: principalID}, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	service.ConfigureCommandGuard(guard)
	return service
}

func validLibraryFormInput() CreateFormInput {
	return CreateFormInput{
		LegalEntityID: "tampered-entity", Code: "VENDOR", Name: "Vendor review", Purpose: "Collect current vendor evidence.",
		ApprovedUses: []string{"VENDOR_DUE_DILIGENCE"}, Tags: []string{"third-party"},
		Presentation: formcontract.Presentation{DefaultMode: formcontract.PresentationAutomatic},
		Sections:     []formcontract.Section{{ID: "identity", Title: "Identity"}},
		Fields:       []TemplateField{{ID: "name", SectionID: "identity", Label: "Registered name", Type: formcontract.TypeShortText, Required: true}},
	}
}

func partialComplianceLibraryFormInput() CreateFormInput {
	input := validLibraryFormInput()
	input.ScoringMode = formcontract.ScoringCompliance
	input.Sections[0].Weight = 80
	input.Fields[0].Type = formcontract.TypeYesNo
	input.Fields[0].Scoring = &formcontract.Scoring{
		Weight:       80,
		AnswerScores: map[string]int{"Yes": 100, "No": 0},
	}
	return input
}

func TestCreateLibraryFormUsesVerifiedIdentity(t *testing.T) {
	repo := NewMemoryRepository()
	service := libraryService(t, repo, "maker-a")
	service.newID = func() (string, error) { return "form-a", nil }

	created, err := service.CreateLibraryForm(formActorContext("bank-a", "entity-a", "maker-a"), validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}
	if created.TenantID != "bank-a" || created.LegalEntityID != "entity-a" || created.CreatedBy != "maker-a" || created.OwnerPrincipalID != "maker-a" || created.ProgramID != "" {
		t.Fatalf("unverified scope used: %#v", created)
	}
}

func TestCreateLibraryFormPreservesAdvancedScoreProfile(t *testing.T) {
	repo := NewMemoryRepository()
	service := libraryService(t, repo, "maker-a")
	service.newID = func() (string, error) { return "form-a", nil }
	input := validLibraryFormInput()
	input.ScoringMode = formcontract.ScoringRisk
	input.ScoreProfile = &formcontract.ScoreProfile{
		Version: "risk-v2", Mode: formcontract.ScoringRisk,
		Bands: formcontract.DefaultConcernBands(),
		Contributions: []formcontract.ScoreContribution{{
			ID: "name-score", Weight: 100,
			Predicate:   formcontract.Predicate{FieldID: "name", Operator: formcontract.PredicateAnswered},
			MatchPoints: 0, NonMatchPoints: 100, Missing: formcontract.MissingZero,
		}},
	}

	created, err := service.CreateLibraryForm(formActorContext("bank-a", "entity-a", "maker-a"), input)
	if err != nil {
		t.Fatal(err)
	}
	input.ScoreProfile.Version = "mutated-later"
	stored, err := service.GetLibraryForm(formActorContext("bank-a", "entity-a", "maker-a"), created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ScoreProfile == nil || stored.ScoreProfile.Version != "risk-v2" || stored.ScoreProfile.Direction != formcontract.DirectionHighIsPoor {
		t.Fatalf("stored profile = %#v", stored.ScoreProfile)
	}
}

func TestCreateLibraryFormFailsClosedWithoutCurrentAuthority(t *testing.T) {
	guard, err := commandauth.New(formAuthorityStub{err: authority.ErrNoRoute}, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(NewMemoryRepository(), nil)
	service.ConfigureCommandGuard(guard)
	_, err = service.CreateLibraryForm(formActorContext("bank-a", "entity-a", "maker-a"), validLibraryFormInput())
	if !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("authority error = %v", err)
	}
}

func TestCreateFormRevisionPinsExpectedVersionAndScope(t *testing.T) {
	repo := NewMemoryRepository()
	service := libraryService(t, repo, "maker-a")
	service.newID = func() (string, error) { return "form-a", nil }
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	created, err := service.CreateLibraryForm(ctx, validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}
	input := validLibraryFormInput()
	input.Name = "Updated vendor review"
	revised, err := service.CreateFormRevision(ctx, created.ID, CreateFormRevisionInput{ExpectedVersion: 1, Form: input})
	if err != nil {
		t.Fatal(err)
	}
	if revised.ID != created.ID || revised.Version != 2 || revised.Status != LifecycleDraft || revised.IsCurrent || revised.TenantID != "bank-a" || revised.LegalEntityID != "entity-a" || revised.CreatedBy != "maker-a" {
		t.Fatalf("revision = %#v", revised)
	}
	if _, err := service.CreateFormRevision(ctx, created.ID, CreateFormRevisionInput{ExpectedVersion: 1, Form: input}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale revision error = %v, want conflict", err)
	}
}

func TestLibraryFormAllowsPartialComplianceDraftButBlocksApproval(t *testing.T) {
	repo := NewMemoryRepository()
	service := libraryService(t, repo, "maker-a")
	service.newID = func() (string, error) { return "form-a", nil }
	ctx := formActorContext("bank-a", "entity-a", "maker-a")

	draft, err := service.CreateLibraryForm(ctx, partialComplianceLibraryFormInput())
	if err != nil {
		t.Fatalf("partial compliance draft was rejected: %v", err)
	}
	if draft.Status != LifecycleDraft || draft.Version != 1 {
		t.Fatalf("draft = %#v", draft)
	}
	if _, err := service.TransitionLibraryForm(ctx, draft.ID, TransitionInput{ExpectedVersion: draft.Version, To: LifecyclePendingApproval}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("incomplete compliance draft approval error = %v, want invalid", err)
	}
	stored, err := service.GetLibraryForm(ctx, draft.ID, draft.Version)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != LifecycleDraft || stored.Version != draft.Version {
		t.Fatalf("rejected approval mutated draft: %#v", stored)
	}

	complete := partialComplianceLibraryFormInput()
	complete.Sections[0].Weight = 100
	complete.Fields[0].Scoring.Weight = 100
	revised, err := service.CreateFormRevision(ctx, draft.ID, CreateFormRevisionInput{ExpectedVersion: draft.Version, Form: complete})
	if err != nil {
		t.Fatalf("complete compliance revision failed: %v", err)
	}
	pending, err := service.TransitionLibraryForm(ctx, revised.ID, TransitionInput{ExpectedVersion: revised.Version, To: LifecyclePendingApproval})
	if err != nil {
		t.Fatalf("complete compliance revision could not enter approval: %v", err)
	}
	if pending.Status != LifecyclePendingApproval {
		t.Fatalf("pending = %#v", pending)
	}
}

func TestLibraryReadsAndSavedViewsUseVerifiedScope(t *testing.T) {
	repo := NewMemoryRepository()
	service := libraryService(t, repo, "maker-a")
	service.newID = func() (string, error) { return "form-a", nil }
	ctx := formActorContext("bank-a", "entity-a", "maker-a")
	if _, err := service.CreateLibraryForm(ctx, validLibraryFormInput()); err != nil {
		t.Fatal(err)
	}
	page, err := service.ListFormLibrary(ctx, FormLibraryFilter{TenantID: "other", LegalEntityID: "other", Limit: 25})
	if err != nil || len(page.Items) != 1 || page.Items[0].Template.LegalEntityID != "entity-a" {
		t.Fatalf("page = %#v, err = %v", page, err)
	}
	service.newID = func() (string, error) { return "view-a", nil }
	view, err := service.SaveFormView(ctx, SavedFormView{TenantID: "other", LegalEntityID: "other", PrincipalID: "other", Name: "Vendor forms", Filter: FormLibraryFilter{Search: "vendor", Limit: 25}})
	if err != nil {
		t.Fatal(err)
	}
	if view.PrincipalID != "maker-a" || view.TenantID != "bank-a" || view.LegalEntityID != "entity-a" {
		t.Fatalf("saved view scope = %#v", view)
	}
}

func TestLibraryFormTransitionEnforcesMakerCheckerAndKeepsExactHistoryReadable(t *testing.T) {
	repo := NewMemoryRepository()
	service := libraryService(t, repo, "maker-a")
	service.newID = func() (string, error) { return "form-a", nil }
	makerContext := formActorContext("bank-a", "entity-a", "maker-a")
	draft, err := service.CreateLibraryForm(makerContext, validLibraryFormInput())
	if err != nil {
		t.Fatal(err)
	}
	pending, err := service.TransitionLibraryForm(makerContext, draft.ID, TransitionInput{ExpectedVersion: draft.Version, To: LifecyclePendingApproval})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionLibraryForm(makerContext, draft.ID, TransitionInput{ExpectedVersion: pending.Version, To: LifecycleActive}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("maker activation error = %v", err)
	}

	reviewerGuard, err := commandauth.New(formAuthorityStub{principal: "reviewer-a"}, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureCommandGuard(reviewerGuard)
	reviewerContext := formActorContext("bank-a", "entity-a", "reviewer-a")
	active, err := service.TransitionLibraryForm(reviewerContext, draft.ID, TransitionInput{ExpectedVersion: pending.Version, To: LifecycleActive})
	if err != nil {
		t.Fatal(err)
	}

	ownerGuard, err := commandauth.New(formAuthorityStub{principal: "owner-a"}, commandauth.ModeEnforce, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureCommandGuard(ownerGuard)
	ownerContext := formActorContext("bank-a", "entity-a", "owner-a")
	retired, err := service.TransitionLibraryForm(ownerContext, draft.ID, TransitionInput{ExpectedVersion: active.Version, To: LifecycleRetired})
	if err != nil || retired.Status != LifecycleRetired || retired.IsCurrent {
		t.Fatalf("retired = %#v, err = %v", retired, err)
	}
	if _, err := service.GetLibraryForm(ownerContext, draft.ID, active.Version); err != nil {
		t.Fatalf("exact referenced revision was no longer readable: %v", err)
	}
	reusable, err := service.ListReusableForms(ownerContext, Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner-a"}, 25)
	if err != nil || len(reusable) != 0 {
		t.Fatalf("retired form remained available for new use: %#v, err = %v", reusable, err)
	}
}
