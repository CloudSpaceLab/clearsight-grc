package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type recordingRequestCreator struct {
	input evidence.CreateRequestInput
}

type recordingEvidenceReader struct {
	request    evidence.Request
	submission evidence.Submission
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
		Fields: input.Fields, FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion,
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
	maker := Actor{TenantID: "bank-a", PrincipalID: "maker"}

	form, err := service.CreateForm(context.Background(), maker, CreateFormInput{
		Code: "PASSWORD-RESET", Name: "Password reset review", Purpose: "Collect the password reset control review.",
		Fields: []TemplateField{{ID: "identity", Label: "Identity verification completed", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}, CriticalAnswers: []string{"No"}}}},
	})
	if err != nil || form.Status != LifecycleDraft || form.Version != 1 {
		t.Fatalf("created form = %#v, err = %v", form, err)
	}
	pending, err := service.TransitionForm(context.Background(), maker, TransitionInput{ID: form.ID, ExpectedVersion: form.Version, To: LifecyclePendingApproval})
	if err != nil || pending.Status != LifecyclePendingApproval || pending.SubmittedBy != maker.PrincipalID {
		t.Fatalf("pending form = %#v, err = %v", pending, err)
	}
	if _, err := service.TransitionForm(context.Background(), maker, TransitionInput{ID: form.ID, ExpectedVersion: pending.Version, To: LifecycleActive}); !errors.Is(err, ErrMakerChecker) {
		t.Fatalf("maker approval error = %v, want maker-checker error", err)
	}

	active, err := service.TransitionForm(context.Background(), Actor{TenantID: "bank-a", PrincipalID: "reviewer"}, TransitionInput{ID: form.ID, ExpectedVersion: pending.Version, To: LifecycleActive})
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
		ID: "form-1", TenantID: "bank-a", Code: "PASSWORD-RESET", Name: "Password reset review", Purpose: "Collect the password reset control review.",
		Fields:    []TemplateField{{ID: "identity", Label: "Identity verification completed", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{ID: "identity", Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}}},
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

	request, err := service.StartCollection(context.Background(), Actor{TenantID: "bank-a", PrincipalID: "owner"}, StartCollectionInput{
		FormTemplateID: "form-1", FormTemplateVersion: 3, ProgramID: "program-1", RespondentPrincipalID: "respondent",
		ReviewerPrincipalID: "reviewer", PeriodStart: periodStart, PeriodEnd: periodEnd, Deadline: now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("start collection: %v", err)
	}
	if request.FormTemplateID != "form-1" || request.FormTemplateVersion != 3 || len(request.Fields) != 1 || requests.input.Recipient.PrincipalID != "respondent" || requests.input.SubjectType != "PROGRAM" {
		t.Fatalf("request = %#v, input = %#v", request, requests.input)
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
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", Code: "FORM", Name: "Form", Purpose: "Purpose", Fields: []TemplateField{{ID: "answer", Label: "Answer", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{ID: "answer", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 1}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}

	_, err := service.StartCollection(context.Background(), Actor{TenantID: "bank-a", PrincipalID: "owner"}, StartCollectionInput{FormTemplateID: form.ID, FormTemplateVersion: form.Version, ProgramID: "program-1", RespondentPrincipalID: "respondent", ReviewerPrincipalID: "reviewer", PeriodStart: now.Add(-time.Hour), PeriodEnd: now, Deadline: now.Add(time.Hour)})
	if !errors.Is(err, ErrInactive) {
		t.Fatalf("collection without active check error = %v, want inactive", err)
	}
}

func TestServiceRejectsCollectionFromDraftForm(t *testing.T) {
	repo := NewMemoryRepository()
	service := NewService(repo, &recordingRequestCreator{})
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", Code: "FORM", Name: "Form", Purpose: "Purpose", Fields: []TemplateField{{ID: "answer", Label: "Answer", Type: "text"}}, Lifecycle: Lifecycle{Status: LifecycleDraft, Version: 1}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	_, err := service.StartCollection(context.Background(), Actor{TenantID: "bank-a", PrincipalID: "owner"}, StartCollectionInput{FormTemplateID: "form-1", FormTemplateVersion: 1, ProgramID: "program-1", RespondentPrincipalID: "respondent", ReviewerPrincipalID: "reviewer", PeriodStart: time.Now().Add(-time.Hour), PeriodEnd: time.Now(), Deadline: time.Now().Add(time.Hour)})
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
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", Code: "FORM", Name: "Form", Purpose: "Purpose", Fields: []TemplateField{{ID: "answer", Label: "Answer", Type: "text"}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	actor := Actor{TenantID: "bank-a", PrincipalID: "owner"}
	paused, err := service.TransitionForm(context.Background(), actor, TransitionInput{ID: form.ID, ExpectedVersion: 3, To: LifecyclePaused})
	if err != nil || paused.Status != LifecyclePaused || !paused.IsCurrent || paused.Version != 4 {
		t.Fatalf("paused = %#v, err = %v", paused, err)
	}
	now = now.Add(time.Minute)
	resumed, err := service.TransitionForm(context.Background(), actor, TransitionInput{ID: form.ID, ExpectedVersion: 4, To: LifecycleActive})
	if err != nil || resumed.Status != LifecycleActive || !resumed.IsCurrent || resumed.Version != 5 {
		t.Fatalf("resumed = %#v, err = %v", resumed, err)
	}
	now = now.Add(time.Minute)
	retired, err := service.TransitionForm(context.Background(), actor, TransitionInput{ID: form.ID, ExpectedVersion: 5, To: LifecycleRetired})
	if err != nil || retired.Status != LifecycleRetired || retired.IsCurrent || retired.EffectiveUntil == nil || retired.Version != 6 {
		t.Fatalf("retired = %#v, err = %v", retired, err)
	}
	versions, err := repo.ListFormRevisions(context.Background(), "bank-a", 20)
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
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", Code: "FORM", Name: "Review", Purpose: "Review the control.", Fields: []TemplateField{{ID: "secure", Label: "Secure", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{ID: "secure", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}}}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3}}
	if _, err := repo.CreateFormRevision(context.Background(), form); err != nil {
		t.Fatal(err)
	}
	maker := Actor{TenantID: "bank-a", PrincipalID: "maker"}
	check, err := service.CreateCheck(context.Background(), maker, CreateCheckInput{
		ProgramID: "program-1", Code: "RESET", Name: "Password reset safeguards", Claim: "Password reset safeguards are operating.",
		InputKind: InputForm, FormTemplateID: form.ID, FormTemplateVersion: form.Version, Thresholds: DefaultThresholds(), FreshnessMinutes: 10080, MinimumCoverage: 1, FailureAction: FailureReview,
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
	active, err := service.TransitionCheck(context.Background(), Actor{TenantID: "bank-a", PrincipalID: "reviewer"}, TransitionInput{ID: check.ID, ExpectedVersion: pending.Version, To: LifecycleActive})
	if err != nil || active.Status != LifecycleActive || !active.IsCurrent || active.ApprovedBy != "reviewer" {
		t.Fatalf("active = %#v, err = %v", active, err)
	}
}

func TestServiceRejectsInvalidFormScoring(t *testing.T) {
	service := NewService(NewMemoryRepository(), &recordingRequestCreator{})
	service.newID = func() (string, error) { return "form-1", nil }
	_, err := service.CreateForm(context.Background(), Actor{TenantID: "bank-a", PrincipalID: "maker"}, CreateFormInput{
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
	form := FormTemplate{ID: "form-1", TenantID: "bank-a", Code: "RESET", Name: "Password reset review", Purpose: "Review safeguards.", Fields: []TemplateField{{ID: "secure", Label: "Secure", Type: "single_select", Required: true, Options: []string{"Yes", "No"}, Scoring: &FormField{ID: "secure", Required: true, Weight: 1, AnswerScores: map[string]int{"Yes": 0, "No": 100}, CriticalAnswers: []string{"No"}}}}, Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: &activeAt, Version: 3}}
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
		request:    evidence.Request{ID: "request-1", TenantID: "bank-a", SubjectType: "PROGRAM", SubjectID: "program-1", FormTemplateID: form.ID, FormTemplateVersion: form.Version},
		submission: evidence.Submission{ID: "submission-1", TenantID: "bank-a", RequestID: "request-1", Channel: "INTERNAL", Answers: map[string]string{"secure": "No"}, SubmittedBy: "operator", SubmittedAt: now},
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
