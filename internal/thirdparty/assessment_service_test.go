package thirdparty

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type assessmentGuardStub struct {
	requests      []commandauth.Request
	decisionActor identity.Actor
	overrideActor bool
	err           error
}

func (g *assessmentGuardStub) Authorize(ctx context.Context, request commandauth.Request) (commandauth.Decision, error) {
	g.requests = append(g.requests, request)
	if g.err != nil {
		return commandauth.Decision{}, g.err
	}
	actor := g.decisionActor
	if !g.overrideActor {
		actor, _ = identity.FromContext(ctx)
	}
	return commandauth.Decision{Allowed: true, Enforced: true, Actor: actor}, nil
}

func TestStartAssessmentBindsScopeAndStarterToVerifiedIdentity(t *testing.T) {
	service, _, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	caller := Actor{TenantID: "forged-bank", LegalEntityID: "forged-entity", PrincipalID: "forged-owner"}

	got, err := service.StartAssessment(assessmentContext(), caller, relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if err != nil {
		t.Fatal(err)
	}
	if got.TenantID != "bank" || got.LegalEntityID != "entity" || got.StartedByPrincipalID != "verified-owner" {
		t.Fatalf("assessment trusted caller identity: %#v", got)
	}
}

func TestAssessmentRejectsGuardIdentityThatDiffersFromVerifiedContext(t *testing.T) {
	guard := newAssessmentGuard()
	guard.decisionActor.PrincipalID = "other-principal"
	guard.overrideActor = true
	service, _, relationship := newAssessmentServiceFixture(t, guard)

	_, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if !errors.Is(err, ErrAssessmentIdentityMismatch) {
		t.Fatalf("expected guard identity mismatch, got %v", err)
	}
}

func TestAssessmentCommandsFailClosedWithoutVerifiedIdentityOrAuthority(t *testing.T) {
	service, _, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	_, err := service.StartAssessment(context.Background(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if !errors.Is(err, identity.ErrMissingIdentity) {
		t.Fatalf("expected verified identity error, got %v", err)
	}

	service, _, relationship = newAssessmentServiceFixture(t, nil)
	_, err = service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if !errors.Is(err, ErrAssessmentAuthorityUnavailable) {
		t.Fatalf("expected fail-closed authority error, got %v", err)
	}
}

func TestStartAssessmentReusesCurrentEpisodeAfterRelationshipAdvances(t *testing.T) {
	service, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	input := validStartAssessmentInput(relationship.Relationship.Version)
	first, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	updatedInput := UpdateRelationshipInput{ExpectedVersion: relationship.Relationship.Version, ServiceName: relationship.Relationship.ServiceName + " updated", Criticality: relationship.Relationship.Criticality, PrivacyRole: relationship.Relationship.PrivacyRole}
	if _, err := NewService(repo).UpdateRelationship(assessmentContext(), assessmentActor(), relationship.Relationship.ID, updatedInput); err != nil {
		t.Fatal(err)
	}
	input.FormTemplateID = "different-form"
	input.FormTemplateVersion = 99
	second, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.Version != first.Version || second.FormTemplateID != "form-1" || second.FormTemplateVersion != 3 {
		t.Fatalf("stable replay did not return immutable episode: first=%#v second=%#v", first, second)
	}
}

func TestStartAssessmentRequiresCurrentScopedRelationshipAndVersionForNewEpisode(t *testing.T) {
	service, _, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	otherContext := assessmentContextFor("bank", "other-entity", "verified-owner")
	_, err := service.StartAssessment(otherContext, assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected scoped not found, got %v", err)
	}
	_, err = service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version+1))
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected relationship version conflict, got %v", err)
	}
}

func TestSetupCompletionReactionIsIdempotentAndCarriesExactReferences(t *testing.T) {
	service, _, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustStartAssessment(t, service, relationship)
	input := AssessmentSetupCompletedInput{
		Scope: Scope{TenantID: "bank", LegalEntityID: "entity"}, AssessmentID: assessment.ID,
		ExpectedVersion: assessment.Version, CausationID: "event-setup-1", SetupJobID: "job-1", ReviewMatterID: "matter-1",
	}
	first, err := service.RecordAssessmentSetupCompleted(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.RecordAssessmentSetupCompleted(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.Status != AssessmentReadyToSend || first.ReviewMatterID != "matter-1" {
		t.Fatalf("setup reaction was not idempotent: first=%#v second=%#v", first, second)
	}
}

func TestMemoryAssessmentRepositoryRejectsSyntheticInternalTransition(t *testing.T) {
	service, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustStartAssessment(t, service, relationship)
	_, err := repo.TransitionAssessment(context.Background(), AssessmentTransitionRecord{
		Scope: scopeFromVerified(), ID: assessment.ID, ExpectedVersion: assessment.Version,
		From: []AssessmentStatus{AssessmentSetupPending}, To: AssessmentReadyToSend,
		At: time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC), ReviewMatterID: "matter-bypass",
	})
	if !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("synthetic internal transition was accepted: %v", err)
	}
}

func TestRecordRequestIssuedAtomicallyLinksOriginAndStartsCollection(t *testing.T) {
	service, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, service, mustStartAssessment(t, service, relationship))
	input := issuedRequestInput(assessment, "request-1", "invitation-1", AssessmentRequestInitial, 1)

	issued, err := service.RecordRequestIssued(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Assessment.Status != AssessmentCollecting || issued.Assessment.CurrentRequestID != "request-1" {
		t.Fatalf("request issue did not start collection: %#v", issued)
	}
	if issued.Link.Sequence != 1 || issued.Link.OriginType != AssessmentRequestOrigin || issued.Link.OriginID != assessment.ID || issued.Link.OriginSequence != 1 || issued.Link.InvitationID != "invitation-1" {
		t.Fatalf("request origin was not preserved: %#v", issued.Link)
	}
	links, err := repo.ListAssessmentRequestLinks(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil || len(links) != 1 || links[0] != issued.Link {
		t.Fatalf("unexpected request history %#v err=%v", links, err)
	}
}

func TestSubmissionReactionIsIdempotentAndRequiresCurrentRequest(t *testing.T) {
	service, _, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustCollectingAssessment(t, service, relationship)
	input := AssessmentSubmittedInput{
		Scope: Scope{TenantID: "bank", LegalEntityID: "entity"}, AssessmentID: assessment.ID,
		ExpectedVersion: assessment.Version, CausationID: "consumer-1", EventID: "event-submitted-1", RequestID: "request-1", SubmissionID: "submission-1",
	}
	first, err := service.RecordAssessmentSubmitted(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.RecordAssessmentSubmitted(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first != second || first.Status != AssessmentSubmitted || first.SubmissionID != "submission-1" {
		t.Fatalf("submission reaction was not idempotent: first=%#v second=%#v", first, second)
	}
}

func TestClarificationReturnsAssessmentToCollectionAndBlocksCompletion(t *testing.T) {
	service, _, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := advanceAssessmentToReview(t, service, relationship)
	issued, err := service.RecordRequestIssued(assessmentContext(), assessmentActor(), assessment.ID, issuedRequestInput(assessment, "request-2", "invitation-2", AssessmentRequestClarification, 2))
	if err != nil {
		t.Fatal(err)
	}
	if issued.Assessment.Status != AssessmentCollecting || issued.Link.Sequence != 2 {
		t.Fatalf("clarification did not resume collection: %#v", issued)
	}
	_, err = service.CompleteAssessment(assessmentContext(), assessmentActor(), assessment.ID, CompleteAssessmentInput{
		ExpectedVersion: issued.Assessment.Version, Conclusion: AssessmentSatisfactory, Rationale: "Current evidence supports the scoped conclusion.",
	})
	if !errors.Is(err, ErrInvalidAssessmentTransition) {
		t.Fatalf("clarification did not block completion, got %v", err)
	}
}

func TestCompleteAssessmentRequiresVerifiedReviewerAndValidBoundedConclusion(t *testing.T) {
	guard := newAssessmentGuard()
	service, _, relationship := newAssessmentServiceFixture(t, guard)
	assessment := advanceAssessmentToReview(t, service, relationship)

	_, err := service.CompleteAssessment(assessmentContext(), assessmentActor(), assessment.ID, CompleteAssessmentInput{ExpectedVersion: assessment.Version, Conclusion: AssessmentSatisfactory, Rationale: strings.Repeat("r", 4001)})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected rationale bound, got %v", err)
	}
	_, err = service.CompleteAssessment(assessmentContext(), assessmentActor(), assessment.ID, CompleteAssessmentInput{ExpectedVersion: assessment.Version, Conclusion: AssessmentSatisfactory, Rationale: "Current evidence supports the scoped conclusion.", Uncertainty: strings.Repeat("u", 2001)})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected uncertainty bound, got %v", err)
	}
	past := time.Date(2026, 8, 26, 9, 59, 0, 0, time.UTC)
	_, err = service.CompleteAssessment(assessmentContext(), assessmentActor(), assessment.ID, CompleteAssessmentInput{ExpectedVersion: assessment.Version, Conclusion: AssessmentSatisfactory, Rationale: "Current evidence supports the scoped conclusion.", NextReviewRecommendedAt: &past})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected future next review, got %v", err)
	}

	guard.decisionActor.PrincipalID = "other-reviewer"
	guard.overrideActor = true
	_, err = service.CompleteAssessment(assessmentContext(), assessmentActor(), assessment.ID, CompleteAssessmentInput{ExpectedVersion: assessment.Version, Conclusion: AssessmentSatisfactory, Rationale: "Current evidence supports the scoped conclusion."})
	if !errors.Is(err, ErrAssessmentIdentityMismatch) {
		t.Fatalf("expected reviewer identity mismatch, got %v", err)
	}
	guard.decisionActor = verifiedIdentity()
	guard.overrideActor = false
	future := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	completed, err := service.CompleteAssessment(assessmentContext(), Actor{PrincipalID: "forged-reviewer"}, assessment.ID, CompleteAssessmentInput{
		ExpectedVersion: assessment.Version, Conclusion: AssessmentSatisfactoryWithConditions,
		Rationale: "Current evidence supports onboarding subject to the recorded remediation conditions.", NextReviewRecommendedAt: &future,
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != AssessmentCompleted || completed.ReviewerPrincipalID != "verified-owner" || completed.Conclusion != AssessmentSatisfactoryWithConditions {
		t.Fatalf("unexpected conclusion %#v", completed)
	}
	last := guard.requests[len(guard.requests)-1]
	if last.DecisionType != AssessmentCompleteCommand || last.ObjectID != assessment.ID {
		t.Fatalf("completion did not re-evaluate current route: %#v", last)
	}
}

func TestAssessmentBoundsIdentifiersAndCancellationReason(t *testing.T) {
	service, _, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	input := validStartAssessmentInput(relationship.Relationship.Version)
	input.FormTemplateID = strings.Repeat("x", 129)
	if _, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, input); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected template identifier bound, got %v", err)
	}
	assessment := mustStartAssessment(t, service, relationship)
	if _, err := service.CancelAssessment(assessmentContext(), assessmentActor(), assessment.ID, CancelAssessmentInput{ExpectedVersion: assessment.Version, Reason: strings.Repeat("x", 2001)}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected cancellation reason bound, got %v", err)
	}
}

func TestCompleteAssessmentDoesNotActivateRelationship(t *testing.T) {
	service, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := advanceAssessmentToReview(t, service, relationship)
	_, err := service.CompleteAssessment(assessmentContext(), assessmentActor(), assessment.ID, CompleteAssessmentInput{
		ExpectedVersion: assessment.Version, Conclusion: AssessmentSatisfactory,
		Rationale: "The submitted evidence is current and supports the scoped assessment conclusion.",
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := repo.GetRelationship(context.Background(), scopeFromVerified(), relationship.Relationship.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Relationship.Status != RelationshipProposed || current.Relationship.Version != relationship.Relationship.Version {
		t.Fatalf("assessment changed relationship lifecycle: %#v", current.Relationship)
	}
}

func newAssessmentServiceFixture(t *testing.T, guard AssessmentCommandGuard) (*AssessmentService, *MemoryAssessmentRepository, Aggregate) {
	t.Helper()
	repo := NewMemoryAssessmentRepository()
	relationshipService := NewService(repo)
	relationshipService.now = func() time.Time { return time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC) }
	relationship, err := relationshipService.CreateRelationship(context.Background(), Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "verified-owner"}, validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	service := NewAssessmentService(repo, guard)
	service.now = func() time.Time { return time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC) }
	return service, repo, relationship
}

func newAssessmentGuard() *assessmentGuardStub {
	return &assessmentGuardStub{decisionActor: verifiedIdentity()}
}

func verifiedIdentity() identity.Actor {
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	return identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "verified-owner", Kind: "PERSON", IssuedAt: now, ExpiresAt: now.Add(24 * time.Hour)}
}

func assessmentContext() context.Context {
	return identity.WithActor(context.Background(), verifiedIdentity())
}

func assessmentContextFor(tenantID, legalEntityID, principalID string) context.Context {
	actor := verifiedIdentity()
	actor.TenantID, actor.LegalEntityID, actor.PrincipalID = tenantID, legalEntityID, principalID
	return identity.WithActor(context.Background(), actor)
}

func assessmentActor() Actor {
	return Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "verified-owner"}
}

func scopeFromVerified() Scope {
	return Scope{TenantID: "bank", LegalEntityID: "entity"}
}

func validStartAssessmentInput(version int64) StartAssessmentInput {
	return StartAssessmentInput{RelationshipVersion: version, FormTemplateID: "form-1", FormTemplateVersion: 3, ReviewDueAt: time.Date(2026, 9, 9, 10, 0, 0, 0, time.UTC)}
}

func mustStartAssessment(t *testing.T, service *AssessmentService, relationship Aggregate) Assessment {
	t.Helper()
	assessment, err := service.StartAssessment(assessmentContext(), assessmentActor(), relationship.Relationship.ID, validStartAssessmentInput(relationship.Relationship.Version))
	if err != nil {
		t.Fatal(err)
	}
	return assessment
}

func mustReadyAssessment(t *testing.T, service *AssessmentService, assessment Assessment) Assessment {
	t.Helper()
	ready, err := service.RecordAssessmentSetupCompleted(context.Background(), AssessmentSetupCompletedInput{
		Scope: scopeFromVerified(), AssessmentID: assessment.ID, ExpectedVersion: assessment.Version,
		CausationID: "event-setup-1", SetupJobID: "job-1", ReviewMatterID: "matter-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	return ready
}

func issuedRequestInput(assessment Assessment, requestID, invitationID string, purpose AssessmentRequestPurpose, sequence int) RecordRequestIssuedInput {
	return RecordRequestIssuedInput{ExpectedVersion: assessment.Version, RequestID: requestID, Purpose: purpose, InvitationID: invitationID, OriginType: AssessmentRequestOrigin, OriginID: assessment.ID, OriginSequence: sequence}
}

func mustCollectingAssessment(t *testing.T, service *AssessmentService, relationship Aggregate) Assessment {
	t.Helper()
	assessment := mustReadyAssessment(t, service, mustStartAssessment(t, service, relationship))
	issued, err := service.RecordRequestIssued(assessmentContext(), assessmentActor(), assessment.ID, issuedRequestInput(assessment, "request-1", "invitation-1", AssessmentRequestInitial, 1))
	if err != nil {
		t.Fatal(err)
	}
	return issued.Assessment
}

func advanceAssessmentToReview(t *testing.T, service *AssessmentService, relationship Aggregate) Assessment {
	t.Helper()
	assessment := mustCollectingAssessment(t, service, relationship)
	var err error
	assessment, err = service.RecordAssessmentSubmitted(context.Background(), AssessmentSubmittedInput{
		Scope: scopeFromVerified(), AssessmentID: assessment.ID, ExpectedVersion: assessment.Version,
		CausationID: "consumer-1", EventID: "event-submitted-1", RequestID: assessment.CurrentRequestID, SubmissionID: "submission-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	assessment, err = service.StartAssessmentReview(assessmentContext(), assessmentActor(), assessment.ID, assessment.Version)
	if err != nil {
		t.Fatal(err)
	}
	return assessment
}
