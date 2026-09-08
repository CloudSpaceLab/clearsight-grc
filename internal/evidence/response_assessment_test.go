package evidence

import (
	"context"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"testing"
	"time"
)

func assessmentFixture(t *testing.T) (*DistributionService, *MemoryDistributionStore, context.Context) {
	t.Helper()
	f := Field{ID: "report", Label: "Test report", Type: "file", Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentManual, Required: true, Weight: 100, ReviewerRole: "RISK", Rubric: []formcontract.AssessmentOutcome{{ID: "poor", Label: "Incomplete", Points: 80}, {ID: "good", Label: "Sufficient", Points: 10}}}}
	repo := NewMemoryRepositoryWithRecipientCandidates(nil, []Request{{ID: "request", TenantID: "tenant", LegalEntityID: "entity", SubjectType: "PROGRAM", SubjectID: "program", FormTemplateID: "form", FormTemplateVersion: 2, ScoringMode: formcontract.ScoringRisk, Fields: []Field{f}}}, []RecipientCandidate{{PrincipalID: "reviewer", TenantID: "tenant", Kind: "PERSON", Active: true, LegalEntityIDs: []string{"entity"}, ReadableSubjects: map[string]bool{"PROGRAM:program": true}}})
	repo.submissions["submission"] = Submission{ID: "submission", TenantID: "tenant", RequestID: "request", Answers: map[string]formcontract.AnswerValue{"report": {ArtifactIDs: []string{"artifact"}}}}
	store := NewMemoryDistributionStore(repo, nil, nil)
	store.distributions["distribution"] = FormDistribution{ID: "distribution", TenantID: "tenant", LegalEntityID: "entity", FormTemplateID: "form", FormTemplateVersion: 2, SubjectType: "PROGRAM", SubjectID: "program"}
	store.responseRevisions["distribution"] = []ResponseRevision{{ID: "response", TenantID: "tenant", LegalEntityID: "entity", DistributionID: "distribution", SubmissionID: "submission", Revision: 1, Current: true, State: ResponseRevisionFinal}}
	ctx := identity.WithActor(context.Background(), identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", ExpiresAt: time.Now().Add(time.Hour)})
	service := NewDistributionService(store).WithAssessmentAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary, formcontract.Field) (string, error) {
		return "route-v3", nil
	})
	return service, store, ctx
}
func TestResponseAssessmentImmutableConflictAndAuthority(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	v, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response")
	if err != nil || v.State != "AWAITING_REVIEW" || v.RequiredCount != 1 {
		t.Fatalf("pending=%+v err=%v", v, err)
	}
	input := RecordResponseAssessmentInput{ExpectedVersion: 0, Decisions: []FieldAssessmentInput{{FieldID: "report", OutcomeID: "poor", Rationale: "Report omits the tested systems."}}}
	got, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 1 || got.State != "ASSESSED" || got.AssessedScore == nil || *got.AssessedScore.RawScore != 80 {
		t.Fatalf("result=%+v", got)
	}
	if _, err = s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatalf("replay accepted: %v", err)
	}
	input.ExpectedVersion = 1
	input.Decisions[0].OutcomeID = "good"
	got2, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Fields[0].Decision.SupersedesID != got.Fields[0].Decision.ID {
		t.Fatal("correction lost lineage")
	}
	if len(store.events) != 2 || len(store.outbox) != 2 {
		t.Fatal("event/outbox not atomic")
	}
	if store.repo.submissions["submission"].Answers["report"].ArtifactIDs[0] != "artifact" {
		t.Fatal("vendor answer mutated")
	}
	s.WithAssessmentAuthorizer(nil)
	input.ExpectedVersion = 2
	if _, err = s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("missing authority accepted %v", err)
	}
}

func TestResponseAssessmentRejectsScopeSupersessionAndInvalidDecision(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	input := RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "report", OutcomeID: "poor", Rationale: "Missing scope."}}}
	if _, err := s.RecordResponseAssessment(context.Background(), "tenant", "entity", "response", input); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatal(err)
	}
	if _, err := s.RecordResponseAssessment(ctx, "tenant", "other", "response", input); !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatal(err)
	}
	if _, err := s.GetResponseAssessment(ctx, "tenant", "entity", "stranger", "response"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("restricted read allowed %v", err)
	}
	input.Decisions[0].OutcomeID = "unapproved"
	if _, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input); !errors.Is(err, ErrAssessmentInvalid) {
		t.Fatal(err)
	}
	input.Decisions[0].OutcomeID = "poor"
	store.responseRevisions["distribution"][0].Current = false
	if _, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input); !errors.Is(err, ErrAssessmentConflict) {
		t.Fatal(err)
	}
	if len(store.events) != 0 || len(store.assessmentDecisions) != 0 {
		t.Fatal("failed command changed material state")
	}
}
func TestAssessmentSubmissionMetadata(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	_ = s
	_ = ctx
	r := store.repo.requests["request"]
	revision, err := buildResponseRevision(r, AssuranceLinkPossession, nil, store.repo.submissions["submission"].Answers)
	if err != nil {
		t.Fatal(err)
	}
	if revision.Score.AssessmentRequiredCount != 1 || revision.Score.AssessmentReviewCount != 1 {
		t.Fatal("immutable submission lost review population")
	}
}

func TestResponseAssessmentRejectsRespondentReview(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	sub := store.repo.submissions["submission"]
	sub.SubmittedBy = "reviewer"
	store.repo.submissions["submission"] = sub
	_, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "report", OutcomeID: "poor", Rationale: "Reason."}}})
	if !errors.Is(err, ErrAssessmentForbidden) {
		t.Fatalf("respondent self-review accepted %v", err)
	}
}

func TestResponseAssessmentRetiredDistributionIsHistorical(t *testing.T) {
	for _, status := range []DistributionStatus{DistributionRevoked, DistributionSuperseded} {
		t.Run(string(status), func(t *testing.T) {
			s, store, ctx := assessmentFixture(t)
			d := store.distributions["distribution"]
			d.Status = status
			store.distributions[d.ID] = d
			v, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response")
			if err != nil || v.Current || v.MayReview {
				t.Fatalf("retired response editable: %+v %v", v, err)
			}
			_, err = s.RecordResponseAssessment(ctx, "tenant", "entity", "response", RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "report", OutcomeID: "poor", Rationale: "Reason."}}})
			if !errors.Is(err, ErrAssessmentConflict) {
				t.Fatalf("retired response write: %v", err)
			}
		})
	}
}

func TestResponseAssessmentRoutedReadDoesNotGrantFieldWrite(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	d := store.distributions["distribution"]
	d.SubjectType = "VENDOR_RELATIONSHIP"
	store.distributions[d.ID] = d
	r := store.repo.requests["request"]
	r.SubjectType = d.SubjectType
	store.repo.requests[r.ID] = r
	s.WithAssessmentAuthorizer(nil).WithResponseDiscoveryAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary) error { return nil })
	v, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response")
	if err != nil || v.MayReview || v.Fields[0].MayReview {
		t.Fatalf("routed read wrongly needs/grants field write: %+v %v", v, err)
	}
}
func TestResponseAssessmentPolicyGetterAndLegacy(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	auto := &ResponseScoreResult{State: ResponseScoreNotConfigured}
	store.responseRevisions["distribution"][0].Score = auto
	_, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "report", OutcomeID: "poor", Rationale: "Missing scope."}}})
	if err != nil {
		t.Fatal(err)
	}
	summary, err := s.GetCompletedResponseForExecution(ctx, "tenant", "response")
	if err != nil || summary.BankAssessment == nil || summary.BankAssessment.Score == nil || !summary.BankAssessment.Score.Final {
		t.Fatalf("policy summary %+v %v", summary, err)
	}
	request := store.repo.requests["request"]
	request.Fields[0].Assessment = nil
	store.repo.requests["request"] = request
	store.assessments = nil
	got, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response")
	if err != nil || got.State != "NOT_REQUIRED" || got.AssessedScore.State != auto.State {
		t.Fatalf("legacy changed %+v %v", got, err)
	}
}

func TestResponseAssessmentExecutionGuardRejectsSupersededVersion(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	input := RecordResponseAssessmentInput{Decisions: []FieldAssessmentInput{{FieldID: "report", OutcomeID: "poor", Rationale: "Missing scope."}}}
	if _, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input); err != nil {
		t.Fatal(err)
	}
	called := false
	apply := func() error { called = true; return nil }
	if err := s.WithAssessedResponseForExecution(ctx, "tenant", "response", 1, apply); err != nil || !called {
		t.Fatalf("current guard: %v", err)
	}
	input.ExpectedVersion = 1
	input.Decisions[0].OutcomeID = "good"
	if _, err := s.RecordResponseAssessment(ctx, "tenant", "entity", "response", input); err != nil {
		t.Fatal(err)
	}
	called = false
	if err := s.WithAssessedResponseForExecution(ctx, "tenant", "response", 1, apply); !errors.Is(err, ErrAssessmentConflict) || called {
		t.Fatalf("stale guard: %v called=%v", err, called)
	}
	store.responseRevisions["distribution"][0].Current = false
	if err := s.WithAssessedResponseForExecution(ctx, "tenant", "response", 2, apply); !errors.Is(err, ErrAssessmentConflict) || called {
		t.Fatalf("superseded response: %v called=%v", err, called)
	}
}

func TestResponseAssessmentVendorReviewerReadPreservesRestrictedWork(t *testing.T) {
	s, store, ctx := assessmentFixture(t)
	d := store.distributions["distribution"]
	d.SubjectType = "VENDOR_RELATIONSHIP"
	store.distributions[d.ID] = d
	r := store.repo.requests["request"]
	r.SubjectType = d.SubjectType
	store.repo.requests[r.ID] = r
	v, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response")
	if err != nil || !v.MayReview || !v.Fields[0].MayReview {
		t.Fatalf("routed reviewer cannot read: %+v %v", v, err)
	}
	store.repo.artifacts["artifact"] = Artifact{ID: "artifact", TenantID: "tenant", RequestID: "request", FileName: "test-report.pdf", MediaType: "application/pdf", Status: ArtifactAvailable}
	query := DocumentQuery{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "reviewer", ResponseRevisionID: "response", Limit: 10}
	page, err := s.ListDocuments(ctx, query)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("reviewer document inventory: %+v %v", page, err)
	}
	query.SubmissionID = "submission"
	query.FieldID = "report"
	query.ArtifactID = "artifact"
	page, err = s.ListDocuments(ctx, query)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("reviewer content occurrence: %+v %v", page, err)
	}
	query.ResponseRevisionID = ""
	page, err = s.ListDocuments(ctx, query)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("unscoped reviewer content: %+v %v", page, err)
	}
	s.WithAssessmentAuthorizer(nil)
	if _, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked reviewer can read: %v", err)
	}
	s.WithAssessmentAuthorizer(func(context.Context, identity.Actor, CompletedResponseSummary, formcontract.Field) (string, error) {
		return "route", nil
	})
	r.Origin.Type = "THIRD_PARTY_WORK"
	store.repo.requests[r.ID] = r
	if _, err := s.GetResponseAssessment(ctx, "tenant", "entity", "reviewer", "response"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("review route bypassed restricted work: %v", err)
	}
}
