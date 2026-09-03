package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type recordingRequestCreator struct {
	input evidence.CreateRequestInput
}

type recordingEvidenceReader struct {
	request    evidence.Request
	submission evidence.Submission
}

type recordingSourceReader struct {
	binding      sourceaccess.BindingRevision
	bindingErr   error
	previewCalls int
}

func (r *recordingSourceReader) Binding(_ context.Context, _, _ string, _ int64) (sourceaccess.BindingRevision, error) {
	return r.binding, r.bindingErr
}

func (r *recordingSourceReader) PreviewBinding(_ context.Context, _, _ string, _ int64, _ sourceaccess.PageRequest) (sourceaccess.RecordPage, error) {
	r.previewCalls++
	return sourceaccess.RecordPage{}, nil
}

type recordingSourceScopeValidator struct {
	err           error
	calls         int
	tenant        string
	legalEntityID string
	sourceIDs     []string
}

func (v *recordingSourceScopeValidator) ValidateActiveSourcesForEntity(_ context.Context, tenant, legalEntityID string, sourceIDs []string) error {
	v.calls++
	v.tenant = tenant
	v.legalEntityID = legalEntityID
	v.sourceIDs = append([]string(nil), sourceIDs...)
	return v.err
}

func activeSourceBinding(now time.Time) sourceaccess.BindingRevision {
	effectiveFrom := now.Add(-time.Hour)
	return sourceaccess.BindingRevision{
		BindingID: "binding-1", TenantID: "bank-a", SourceID: "source-1", Operations: []sourceaccess.Operation{sourceaccess.OperationPage},
		RevisionLifecycle: sourceaccess.RevisionLifecycle{Status: sourceaccess.RevisionActive, IsCurrent: true, EffectiveFrom: &effectiveFrom, Version: 3},
	}
}

func sourceCheckInput() CreateCheckInput {
	return CreateCheckInput{
		ProgramID: "program-1", Code: "SOURCE", Name: "Source health", Claim: "The source remains healthy.", InputKind: InputSource,
		BindingID: "binding-1", BindingVersion: 3, SourceRules: []SourceRule{{ID: "healthy", Field: "healthy", Operator: OperatorEquals, Expected: "true", RiskPoints: 100}},
		FreshnessMinutes: 60, MinimumCoverage: 1,
	}
}

func (r recordingEvidenceReader) GetRequest(_ context.Context, tenant, id string) (evidence.Request, error) {
	if r.request.TenantID != tenant || r.request.ID != id {
		return evidence.Request{}, evidence.ErrNotFound
	}
	return r.request, nil
}

func (r recordingEvidenceReader) GetSubmission(_ context.Context, tenant, id string) (evidence.Submission, error) {
	if r.submission.TenantID != tenant || r.submission.ID != id {
		return evidence.Submission{}, evidence.ErrNotFound
	}
	return r.submission, nil
}

func (r *recordingRequestCreator) CreateRequest(_ context.Context, input evidence.CreateRequestInput) (evidence.Request, error) {
	r.input = input
	return evidence.Request{
		ID: "request-1", TenantID: input.TenantID, SubjectType: input.SubjectType, SubjectID: input.SubjectID,
		Title: input.Title, Purpose: input.Purpose, WhyYou: input.WhyYou, Sensitivity: input.Sensitivity,
		AudienceType: input.AudienceType, EstimatedMinutes: input.EstimatedMinutes, Deadline: input.Deadline,
		Presentation: input.Presentation, ScoringMode: input.ScoringMode, ScoreProfile: input.ScoreProfile, Sections: input.Sections, Fields: input.Fields, FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion,
		CollectionPeriodStart: input.CollectionPeriodStart, CollectionPeriodEnd: input.CollectionPeriodEnd, Version: 1,
	}, nil
}

func TestServiceEnforcesMakerCheckerFormActivation(t *testing.T) {
	repo := NewMemoryRepository()
	requests := &recordingRequestCreator{}
	service := NewService(repo, requests)
	now := time.Date(2026, 8, 17, 11, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "form-1", nil }
	maker := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker"}

	form, err := service.CreateForm(context.Background(), maker, CreateFormInput{
		ProgramID: "program-1", LegalEntityID: "entity-a",
		Code: "PASSWORD-RESET", Name: "Password reset review", Purpose: "Collect the password reset control review.",
		Fields: []TemplateField{{ID: "identity", Label: "Identity verification completed", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}, CriticalAnswers: []string{"No"}}}},
	})
	if err != nil || form.Status != LifecycleDraft || form.Version != 1 {
		t.Fatalf("created form = %#v, err = %v", form, err)
	}
	pending, err := service.TransitionForm(context.Background(), maker, TransitionInput{ID: form.ID, ProgramID: "program-1", LegalEntityID: "entity-a", ExpectedVersion: form.Version, To: LifecyclePendingApproval})
	if err != nil || pending.Status != LifecyclePendingApproval || pending.SubmittedBy != maker.PrincipalID {
		t.Fatalf("pending form = %#v, err = %v", pending, err)
	}
	if _, err := service.TransitionForm(context.Background(), maker, TransitionInput{ID: form.ID, ProgramID: "program-1", LegalEntityID: "entity-a", ExpectedVersion: pending.Version, To: LifecycleActive}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("maker approval error = %v, want maker-checker error", err)
	}

	active, err := service.TransitionForm(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer"}, TransitionInput{ID: form.ID, ProgramID: "program-1", LegalEntityID: "entity-a", ExpectedVersion: pending.Version, To: LifecycleActive})
	if err != nil || active.Status != LifecycleActive || !active.IsCurrent || active.ApprovedBy != "reviewer" || active.EffectiveFrom == nil {
		t.Fatalf("active form = %#v, err = %v", active, err)
	}
}

func TestServiceStartsCollectionFromExactActiveForm(t *testing.T) {
	repo := NewMemoryRepository()
	requests := &recordingRequestCreator{}
	service := NewService(repo, requests)
	now := time.Date(2026, 8, 17, 11, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	activeAt := now.Add(-time.Hour)
	form := FormTemplate{
		ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "PASSWORD-RESET", Name: "Password reset review", Purpose: "Collect the password reset control review.",
		ScoringMode: formcontract.ScoringRisk,
		ScoreProfile: &formcontract.ScoreProfile{Version: "risk-v2", Mode: formcontract.ScoringRisk, Direction: formcontract.DirectionHighIsPoor, Bands: formcontract.DefaultConcernBands(), Contributions: []formcontract.ScoreContribution{{
			ID: "identity-score", Weight: 100, Predicate: formcontract.Predicate{FieldID: "identity", Operator: formcontract.PredicateEquals, Values: []string{"No"}}, MatchPoints: 100, Missing: formcontract.MissingIndeterminate,
		}}},
		Fields:    []TemplateField{{ID: "identity", Label: "Identity verification completed", Type: "single_select", Required: true, Options: []string{"Yes", "No"}}},
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3},
	}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	check := MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", Code: "PASSWORD-RESET-CHECK", Name: "Password reset review", Claim: "Password reset safeguards operated.",
		InputKind: InputForm, FormTemplateID: form.ID, FormTemplateVersion: form.Version, Thresholds: DefaultThresholds(), FreshnessMinutes: 10080, MinimumCoverage: 1, FailureAction: FailureReview,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 1},
	}
	if _, err := repo.CreateCheckRevision(context.Background(), check); err != nil {
		t.Fatal(err)
	}
	periodStart := now.AddDate(0, 0, -7)
	periodEnd := now

	request, err := service.StartCollection(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner"}, StartCollectionInput{
		FormTemplateID: "form-1", FormTemplateVersion: 3, ProgramID: "program-1", RespondentPrincipalID: "respondent",
		LegalEntityID: "entity-a", ReviewerPrincipalID: "reviewer", PeriodStart: periodStart, PeriodEnd: periodEnd, Deadline: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("start collection: %v", err)
	}
	if request.FormTemplateID != "form-1" || request.FormTemplateVersion != 3 || len(request.Fields) != 1 || requests.input.Recipient.PrincipalID != "respondent" || requests.input.SubjectType != "PROGRAM" {
		t.Fatalf("request = %#v, input = %#v", request, requests.input)
	}
	if requests.input.ScoringMode != formcontract.ScoringRisk || requests.input.ScoreProfile == nil || requests.input.ScoreProfile.Version != "risk-v2" {
		t.Fatalf("collection did not pin the exact score profile: %#v", requests.input.ScoreProfile)
	}
	if requests.input.KnownFacts["reviewer"] != "reviewer" {
		t.Fatalf("reviewer fact missing: %#v", requests.input.KnownFacts)
	}
}

func TestServiceRejectsCollectionWithoutActiveProgramCheck(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, &recordingRequestCreator{})
	now := time.Date(2026, 8, 17, 11, 0, 0, 0, time.UTC)
	activeAt := now.Add(-time.Hour)
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "FORM", Name: "Form", Purpose: "Purpose", Fields: []TemplateField{{ID: "answer", Label: "Answer", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{ID: "answer", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 1}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}

	_, err := service.StartCollection(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner"}, StartCollectionInput{FormTemplateID: form.ID, FormTemplateVersion: form.Version, ProgramID: "program-1", LegalEntityID: "entity-a", RespondentPrincipalID: "respondent", ReviewerPrincipalID: "reviewer", PeriodStart: now.Add(-time.Hour), PeriodEnd: now, Deadline: now.Add(time.Hour)})
	if !errors.Is(err, ErrInactive) {
		t.Fatalf("collection without active check error = %v, want inactive", err)
	}
}

func TestServiceRejectsCollectionFromDraftForm(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, &recordingRequestCreator{})
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "FORM", Name: "Form", Purpose: "Purpose", Fields: []TemplateField{{ID: "answer", Label: "Answer", Type: "text"}}, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	_, err := service.StartCollection(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner"}, StartCollectionInput{FormTemplateID: "form-1", FormTemplateVersion: 1, ProgramID: "program-1", LegalEntityID: "entity-a", RespondentPrincipalID: "respondent", ReviewerPrincipalID: "reviewer", PeriodStart: time.Now().Add(-time.Hour), PeriodEnd: time.Now(), Deadline: time.Now().Add(time.Hour)})
	if !errors.Is(err, ErrInactive) {
		t.Fatalf("draft collection error = %v, want inactive", err)
	}
}

func TestServiceFormPauseResumeAndRetire(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, &recordingRequestCreator{})
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	activeAt := now.Add(-time.Hour)
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "FORM", Name: "Form", Purpose: "Purpose", Fields: []TemplateField{{ID: "answer", Label: "Answer", Type: "text"}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	actor := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner"}
	paused, err := service.TransitionForm(context.Background(), actor, TransitionInput{ID: form.ID, ProgramID: "program-1", LegalEntityID: "entity-a", ExpectedVersion: 3, To: LifecyclePaused})
	if err != nil || paused.Status != LifecyclePaused || !paused.IsCurrent || paused.Version != 4 {
		t.Fatalf("paused = %#v, err = %v", paused, err)
	}
	now = now.Add(time.Minute)
	resumed, err := service.TransitionForm(context.Background(), actor, TransitionInput{ID: form.ID, ProgramID: "program-1", LegalEntityID: "entity-a", ExpectedVersion: 4, To: LifecycleActive})
	if err != nil || resumed.Status != LifecycleActive || !resumed.IsCurrent || resumed.Version != 5 {
		t.Fatalf("resumed = %#v, err = %v", resumed, err)
	}
	now = now.Add(time.Minute)
	retired, err := service.TransitionForm(context.Background(), actor, TransitionInput{ID: form.ID, ProgramID: "program-1", LegalEntityID: "entity-a", ExpectedVersion: 5, To: LifecycleRetired})
	if err != nil || retired.Status != LifecycleRetired || retired.IsCurrent || retired.EffectiveUntil == nil || retired.Version != 6 {
		t.Fatalf("retired = %#v, err = %v", retired, err)
	}
	versions, err := repo.ListFormRevisions(context.Background(), "bank-a", "entity-a", "program-1", 20)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range versions {
		if version.ID == form.ID && version.IsCurrent {
			t.Fatalf("retired form still has a current revision: %#v", version)
		}
	}
}

func TestServiceGovernsFormMonitoringCheck(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, &recordingRequestCreator{})
	now := time.Date(2026, 8, 17, 13, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "check-1", nil }
	activeAt := now.Add(-time.Hour)
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "FORM", Name: "Review", Purpose: "Review the control.", Fields: []TemplateField{{ID: "secure", Label: "Secure", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{ID: "secure", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	maker := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker"}
	check, err := service.CreateCheck(context.Background(), maker, CreateCheckInput{
		ProgramID: "program-1", Code: "RESET", Name: "Password reset safeguards", Claim: "Password reset safeguards are operating.",
		InputKind: InputForm, FormTemplateID: form.ID, FormTemplateVersion: form.Version, CollectionPolicy: &CollectionPolicy{ValidityMonths: 12}, Thresholds: DefaultThresholds(), FreshnessMinutes: 10080, MinimumCoverage: 1, FailureAction: FailureReview,
	})
	if err != nil || check.Status != LifecycleDraft || check.Version != 1 {
		t.Fatalf("check = %#v, err = %v", check, err)
	}
	pending, err := service.TransitionCheck(context.Background(), maker, TransitionInput{ID: check.ID, ExpectedVersion: 1, To: LifecyclePendingApproval})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransitionCheck(context.Background(), maker, TransitionInput{ID: check.ID, ExpectedVersion: pending.Version, To: LifecycleActive}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("same-maker approval error = %v", err)
	}
	active, err := service.TransitionCheck(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer"}, TransitionInput{ID: check.ID, ExpectedVersion: pending.Version, To: LifecycleActive})
	if err != nil || active.Status != LifecycleActive || !active.IsCurrent || active.ApprovedBy != "reviewer" {
		t.Fatalf("active = %#v, err = %v", active, err)
	}
}

func TestServiceRejectsFormMonitoringCheckOutsideExactProgram(t *testing.T) {
	repo := NewMemoryRepository()
	activeAt := time.Now().UTC().Add(-time.Hour)
	form := FormTemplate{ID: "form-other", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-other", Code: "FORM", Name: "Other Program review", Purpose: "Review another Program.", Fields: []TemplateField{{ID: "answer", Label: "Answer", Type: "text"}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 2}}
	if _, err := repo.CreateFormRevision(t.Context(), form); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	_, err := service.CreateCheck(t.Context(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner"}, CreateCheckInput{
		ProgramID: "program-1", Code: "FORM-CHECK", Name: "Form check", Claim: "The exact Program form is used.", InputKind: InputForm,
		FormTemplateID: form.ID, FormTemplateVersion: form.Version, FreshnessMinutes: 60, MinimumCoverage: 1,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-Program form check error = %v, want not found", err)
	}
}

func TestServiceValidatesExactConnectedSourceScopeWhenCreatingCheck(t *testing.T) {
	now := time.Date(2026, 8, 17, 13, 30, 0, 0, time.UTC)
	actor := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner"}

	t.Run("accepts an active current effective page binding for an active source in the exact entity", func(t *testing.T) {
		repo := NewMemoryRepository()
		reader := &recordingSourceReader{binding: activeSourceBinding(now)}
		validator := &recordingSourceScopeValidator{}
		service := NewService(repo, nil)
		service.now = func() time.Time { return now }
		service.newID = func() (string, error) { return "check-1", nil }
		service.ConfigureSourceReader(reader)
		service.ConfigureSourceValidator(validator)

		check, err := service.CreateCheck(t.Context(), actor, sourceCheckInput())
		if err != nil {
			t.Fatalf("create source check: %v", err)
		}
		if check.BindingID != "binding-1" || check.BindingVersion != 3 {
			t.Fatalf("check binding = %s:%d", check.BindingID, check.BindingVersion)
		}
		if validator.calls != 1 || validator.tenant != "bank-a" || validator.legalEntityID != "entity-a" || len(validator.sourceIDs) != 1 || validator.sourceIDs[0] != "source-1" {
			t.Fatalf("source validation = calls %d, tenant %q, entity %q, sources %#v", validator.calls, validator.tenant, validator.legalEntityID, validator.sourceIDs)
		}
	})

	tests := []struct {
		name         string
		actor        Actor
		configure    func(*Service, *recordingSourceReader, *recordingSourceScopeValidator)
		mutate       func(*sourceaccess.BindingRevision)
		validatorErr error
		wantInactive bool
	}{
		{name: "verified legal entity missing", actor: Actor{TenantID: "bank-a", PrincipalID: "owner"}},
		{name: "source reader unavailable", actor: actor, configure: func(service *Service, _ *recordingSourceReader, validator *recordingSourceScopeValidator) {
			service.ConfigureSourceValidator(validator)
		}},
		{name: "source validator unavailable", actor: actor, configure: func(service *Service, reader *recordingSourceReader, _ *recordingSourceScopeValidator) {
			service.ConfigureSourceReader(reader)
		}},
		{name: "binding id differs", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { binding.BindingID = "binding-other" }},
		{name: "binding version differs", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { binding.Version = 4 }},
		{name: "binding tenant differs", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { binding.TenantID = "bank-b" }},
		{name: "binding source missing", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { binding.SourceID = "" }},
		{name: "binding is draft", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { binding.Status = sourceaccess.RevisionDraft }, wantInactive: true},
		{name: "binding is not current", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { binding.IsCurrent = false }, wantInactive: true},
		{name: "binding effective start missing", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { binding.EffectiveFrom = nil }, wantInactive: true},
		{name: "binding not yet effective", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) {
			value := now.Add(time.Minute)
			binding.EffectiveFrom = &value
		}, wantInactive: true},
		{name: "binding effectiveness ended", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) { value := now; binding.EffectiveUntil = &value }, wantInactive: true},
		{name: "binding does not allow page", actor: actor, mutate: func(binding *sourceaccess.BindingRevision) {
			binding.Operations = []sourceaccess.Operation{sourceaccess.OperationLookup}
		}},
		{name: "source is not active in entity", actor: actor, validatorErr: evidence.ErrSourceScopeMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMemoryRepository()
			binding := activeSourceBinding(now)
			if tt.mutate != nil {
				tt.mutate(&binding)
			}
			reader := &recordingSourceReader{binding: binding}
			validator := &recordingSourceScopeValidator{err: tt.validatorErr}
			service := NewService(repo, nil)
			service.now = func() time.Time { return now }
			service.newID = func() (string, error) { return "check-1", nil }
			if tt.configure != nil {
				tt.configure(service, reader, validator)
			} else {
				service.ConfigureSourceReader(reader)
				service.ConfigureSourceValidator(validator)
			}

			_, err := service.CreateCheck(t.Context(), tt.actor, sourceCheckInput())
			if err == nil {
				t.Fatal("create source check succeeded, want fail-closed error")
			}
			if tt.wantInactive && !errors.Is(err, ErrInactive) {
				t.Fatalf("create source check error = %v, want inactive", err)
			}
			checks, listErr := repo.ListCheckRevisions(t.Context(), "bank-a", "program-1", 10)
			if listErr != nil {
				t.Fatal(listErr)
			}
			if len(checks) != 0 {
				t.Fatalf("invalid source check was persisted: %#v", checks)
			}
		})
	}
}

func TestServiceDistinguishesInvalidSourceScopeFromUnavailableValidation(t *testing.T) {
	now := time.Date(2026, 8, 17, 13, 30, 0, 0, time.UTC)
	actor := Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "owner"}

	for _, scopeErr := range []error{evidence.ErrSourceScopeMismatch, evidence.ErrSourceScopeRequired} {
		service := NewService(NewMemoryRepository(), nil)
		service.now = func() time.Time { return now }
		service.ConfigureSourceReader(&recordingSourceReader{binding: activeSourceBinding(now)})
		service.ConfigureSourceValidator(&recordingSourceScopeValidator{err: scopeErr})
		_, err := service.CreateCheck(t.Context(), actor, sourceCheckInput())
		if !errors.Is(err, ErrInvalid) || errors.Is(err, ErrSourceValidationUnavailable) {
			t.Fatalf("exact source-scope error %v classified as %v", scopeErr, err)
		}
	}

	for _, catalogErr := range []error{sourceaccess.ErrCatalogNotFound, sourceaccess.ErrCatalogInvalid} {
		service := NewService(NewMemoryRepository(), nil)
		service.now = func() time.Time { return now }
		service.ConfigureSourceReader(&recordingSourceReader{bindingErr: catalogErr})
		service.ConfigureSourceValidator(&recordingSourceScopeValidator{})
		_, err := service.CreateCheck(t.Context(), actor, sourceCheckInput())
		if !errors.Is(err, ErrInvalid) || errors.Is(err, ErrSourceValidationUnavailable) {
			t.Fatalf("correctable source-catalog error %v classified as %v", catalogErr, err)
		}
	}

	for _, dependencyErr := range []error{sourceaccess.ErrCatalogStorage, errors.New("validator storage unavailable")} {
		service := NewService(NewMemoryRepository(), nil)
		service.now = func() time.Time { return now }
		reader := &recordingSourceReader{binding: activeSourceBinding(now)}
		validator := &recordingSourceScopeValidator{}
		if errors.Is(dependencyErr, sourceaccess.ErrCatalogStorage) {
			reader.bindingErr = dependencyErr
		} else {
			validator.err = dependencyErr
		}
		service.ConfigureSourceReader(reader)
		service.ConfigureSourceValidator(validator)
		_, err := service.CreateCheck(t.Context(), actor, sourceCheckInput())
		if !errors.Is(err, ErrSourceValidationUnavailable) || errors.Is(err, ErrInvalid) {
			t.Fatalf("dependency error %v classified as %v", dependencyErr, err)
		}
	}
}

func TestServiceRevalidatesConnectedSourceScopeAtActivation(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 17, 14, 0, 0, 0, time.UTC)
	pending := MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", Code: "SOURCE", Name: "Source health", Claim: "The source remains healthy.",
		InputKind: InputSource, BindingID: "binding-1", BindingVersion: 3, SourceRules: sourceCheckInput().SourceRules, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: FailureReview,
		Lifecycle: Lifecycle{Status: LifecyclePendingApproval, Version: 2, SubmittedBy: "maker"},
	}
	if _, err := repo.CreateCheckRevision(t.Context(), pending); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	service.ConfigureSourceReader(&recordingSourceReader{binding: activeSourceBinding(now)})
	service.ConfigureSourceValidator(&recordingSourceScopeValidator{err: evidence.ErrSourceScopeMismatch})

	_, err := service.TransitionCheck(t.Context(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "reviewer"}, TransitionInput{ID: pending.ID, ExpectedVersion: pending.Version, To: LifecycleActive})
	if err == nil {
		t.Fatal("activation succeeded after source left the legal entity")
	}
	if _, loadErr := repo.CheckRevision(t.Context(), "bank-a", pending.ID, 3); !errors.Is(loadErr, ErrNotFound) {
		t.Fatalf("activation revision error = %v, want not found", loadErr)
	}
}

func TestServiceRevalidatesConnectedSourceScopeBeforeEvaluation(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 17, 14, 0, 0, 0, time.UTC)
	effectiveFrom := now.Add(-time.Hour)
	active := MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", Code: "SOURCE", Name: "Source health", Claim: "The source remains healthy.",
		InputKind: InputSource, BindingID: "binding-1", BindingVersion: 3, SourceRules: sourceCheckInput().SourceRules, Thresholds: DefaultThresholds(), FreshnessMinutes: 60, MinimumCoverage: 1, FailureAction: FailureReview,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &effectiveFrom, Version: 3},
	}
	if _, err := repo.CreateCheckRevision(t.Context(), active); err != nil {
		t.Fatal(err)
	}
	reader := &recordingSourceReader{binding: activeSourceBinding(now)}
	service := NewService(repo, nil)
	service.now = func() time.Time { return now }
	service.ConfigureSourceReader(reader)
	service.ConfigureSourceValidator(&recordingSourceScopeValidator{err: evidence.ErrSourceScopeMismatch})

	_, err := service.EvaluateSource(t.Context(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "operator"}, EvaluateSourceInput{CheckID: active.ID, CheckVersion: active.Version})
	if err == nil {
		t.Fatal("evaluation succeeded after source left the legal entity")
	}
	if reader.previewCalls != 0 {
		t.Fatalf("connected source was read %d times after scope validation failed", reader.previewCalls)
	}
	results, listErr := repo.ListResults(t.Context(), "bank-a", active.ID, 10)
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(results) != 0 {
		t.Fatalf("invalid source evaluation was persisted: %#v", results)
	}
}

func TestServiceLoadsExactAndLatestCheckForVerifiedTenant(t *testing.T) {
	repo := NewMemoryRepository()
	created, err := repo.CreateCheckRevision(t.Context(), MonitoringCheck{
		ID: "check-1", TenantID: "bank-a", ProgramID: "program-a", Code: "SOURCE", Name: "Source health", Claim: "The source remains healthy.", InputKind: InputSource,
		BindingID: "binding-1", BindingVersion: 1, SourceRules: []SourceRule{{ID: "healthy", Field: "healthy", Operator: OperatorEquals, Expected: "true", RiskPoints: 100}},
		FreshnessMinutes: 60, MinimumCoverage: 1, Thresholds: DefaultThresholds(), FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, nil)
	latest, err := service.LatestCheck(t.Context(), Actor{TenantID: "bank-a", PrincipalID: "viewer"}, created.ID)
	if err != nil || latest.ProgramID != "program-a" || latest.Version != 1 {
		t.Fatalf("latest check = %#v, err = %v", latest, err)
	}
	exact, err := service.Check(t.Context(), Actor{TenantID: "bank-a", PrincipalID: "viewer"}, created.ID, 1)
	if err != nil || exact.ID != created.ID {
		t.Fatalf("exact check = %#v, err = %v", exact, err)
	}
	if _, err := service.LatestCheck(t.Context(), Actor{TenantID: "bank-b", PrincipalID: "viewer"}, created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant latest check error = %v, want not found", err)
	}
}

func TestServiceRejectsInvalidFormScoring(t *testing.T) {
	service := NewService(NewMemoryRepository(), &recordingRequestCreator{})
	service.newID = func() (string, error) { return "form-1", nil }
	_, err := service.CreateForm(context.Background(), Actor{TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker"}, CreateFormInput{
		ProgramID: "program-1", LegalEntityID: "entity-a",
		Code: "FORM", Name: "Review", Purpose: "Review the control.",
		Fields: []TemplateField{{ID: "secure", Label: "Secure", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{Weight: 1, AnswerScores: map[string]int{"Maybe": 100}}}},
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid score error = %v", err)
	}
}

func TestServiceEvaluatesSubmissionAgainstExactActiveRevisions(t *testing.T) {
	repo := NewMemoryRepository()
	now := time.Date(2026, 8, 17, 14, 0, 0, 0, time.UTC)
	activeAt := now.Add(-time.Hour)
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", LegalEntityID: "entity-a", ProgramID: "program-1", Code: "RESET", Name: "Password reset review", Purpose: "Review safeguards.", Fields: []TemplateField{{ID: "secure", Label: "Secure", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{ID: "secure", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}, CriticalAnswers: []string{"No"}}}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3}}
	check := MonitoringCheck{ID: "check-1", TenantID: "bank-a", ProgramID: "program-1", Code: "RESET-CHECK", Name: "Password reset safeguards", Claim: "Safeguards operated.", InputKind: InputForm, FormTemplateID: form.ID, FormTemplateVersion: form.Version, Thresholds: DefaultThresholds(), FreshnessMinutes: 10080, MinimumCoverage: 1, FailureAction: FailureReview, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 2}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateCheckRevision(context.Background(), check); err != nil {
		t.Fatal(err)
	}
	service := NewService(repo, &recordingRequestCreator{})
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "result-1", nil }
	service.ConfigureEvidenceReader(recordingEvidenceReader{
		request:    evidence.Request{ID: "request-1", TenantID: "bank-a", SubjectType: "PROGRAM", SubjectID: "program-1", FormTemplateID: form.ID, FormTemplateVersion: form.Version, KnownFacts: map[string]string{"legal_entity_id": "entity-a"}},
		submission: evidence.Submission{ID: "submission-1", TenantID: "bank-a", RequestID: "request-1", Channel: "INTERNAL", Answers: map[string]formcontract.AnswerValue{"secure": formcontract.TextAnswer("No")}, SubmittedBy: "operator", SubmittedAt: now},
	})

	results, err := service.EvaluateSubmission(context.Background(), "bank-a", "submission-1")
	if err != nil || len(results) != 1 || results[0].Evaluation.Score == nil || *results[0].Evaluation.Score != 100 || results[0].Evaluation.Band != RiskCritical {
		t.Fatalf("results = %#v, err = %v", results, err)
	}
	stored, err := repo.ListResults(context.Background(), "bank-a", check.ID, 10)
	if err != nil || len(stored) != 1 || stored[0].InputReferenceID != "submission-1" || len(stored[0].SubmissionProvenance) == 0 {
		t.Fatalf("stored = %#v, err = %v", stored, err)
	}
}
