package thirdparty

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const (
	AssessmentStartCommand          = "thirdparty.assessment.start"
	AssessmentRequestIssuedCommand  = "thirdparty.assessment.request_issued"
	AssessmentReviewCommand         = "thirdparty.assessment.review"
	AssessmentDocumentReviewCommand = "thirdparty.assessment.document.review"
	AssessmentCompleteCommand       = "thirdparty.assessment.complete"
	AssessmentCancelCommand         = "thirdparty.assessment.cancel"
	assessmentObjectType            = "THIRD_PARTY_ASSESSMENT"
	assessmentRelationshipType      = "THIRD_PARTY_RELATIONSHIP"
	assessmentMaxIdentifierLength   = 128
	assessmentMaxRationaleLength    = 4000
	assessmentMaxUncertaintyLength  = 2000
	assessmentMaxCancelReasonLength = 2000
)

type AssessmentCommandGuard interface {
	Authorize(context.Context, commandauth.Request) (commandauth.Decision, error)
}

type AssessmentCancellationRevoker interface {
	RevokeRequestCapabilities(context.Context, string, string) error
}

type AssessmentService struct {
	repo                AssessmentRepository
	guard               AssessmentCommandGuard
	readiness           AssessmentCompletionReadiness
	cancellationRevoker AssessmentCancellationRevoker
	now                 func() time.Time
	newID               func() (string, error)
}

func (s *AssessmentService) ConfigureCancellationRevoker(revoker AssessmentCancellationRevoker) {
	if s != nil {
		s.cancellationRevoker = revoker
	}
}

func NewAssessmentService(repo AssessmentRepository, guard AssessmentCommandGuard) *AssessmentService {
	return &AssessmentService{repo: repo, guard: guard, now: time.Now, newID: id.NewUUIDv7}
}

func (s *AssessmentService) ConfigureCompletionReadiness(readiness AssessmentCompletionReadiness) {
	if s != nil {
		s.readiness = readiness
	}
}

func (s *AssessmentService) StartAssessment(ctx context.Context, _ Actor, relationshipID string, input StartAssessmentInput) (Assessment, error) {
	relationshipID = strings.TrimSpace(relationshipID)
	if !validAssessmentIdentifier(relationshipID) {
		return Assessment{}, ErrInvalid
	}
	actor, err := s.authorize(ctx, relationshipID, assessmentRelationshipType, AssessmentStartCommand, authority.ResponsibilityOwner)
	if err != nil {
		return Assessment{}, err
	}
	scope := scopeFrom(actor)
	kind, sourceTrigger, restartAssessmentID, err := normalizeAssessmentEpisode(input.ReviewKind, input.SourceTrigger, input.RestartAssessmentID)
	if err != nil {
		return Assessment{}, err
	}
	if restartAssessmentID != "" {
		prior, lookupErr := s.repo.GetAssessment(ctx, scope, restartAssessmentID)
		if lookupErr != nil {
			return Assessment{}, lookupErr
		}
		if prior.RelationshipID != relationshipID || prior.ReviewKind != AssessmentReviewOnboarding || prior.Status != AssessmentCancelled {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		sourceTrigger = "RESTART:" + prior.ID
		if !validAssessmentIdentifier(sourceTrigger) {
			return Assessment{}, ErrInvalid
		}
	}
	stableKey := assessmentEpisodeKey(scope, relationshipID, kind, sourceTrigger)
	current, err := s.repo.GetCurrentAssessment(ctx, scope, relationshipID, kind)
	if err == nil && current.StableEpisodeKey == stableKey {
		return current, nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Assessment{}, err
	}
	input.FormTemplateID = strings.TrimSpace(input.FormTemplateID)
	now := s.now().UTC()
	if input.RelationshipVersion < 1 || !validAssessmentIdentifier(input.FormTemplateID) || input.FormTemplateVersion < 1 || input.ReviewDueAt.IsZero() || !input.ReviewDueAt.After(now) {
		return Assessment{}, ErrInvalid
	}
	relationship, err := s.repo.GetRelationship(ctx, scope, relationshipID)
	if err != nil {
		return Assessment{}, err
	}
	if relationship.Relationship.Version != input.RelationshipVersion {
		return Assessment{}, ErrVersionConflict
	}
	for _, reviewKind := range []AssessmentReviewKind{AssessmentReviewOnboarding, AssessmentReviewPeriodic, AssessmentReviewTriggered} {
		candidate, lookupErr := s.repo.GetCurrentAssessment(ctx, scope, relationshipID, reviewKind)
		if lookupErr == nil && assessmentEpisodeActive(candidate.Status) {
			return Assessment{}, ErrVersionConflict
		}
		if lookupErr != nil && !errors.Is(lookupErr, ErrNotFound) {
			return Assessment{}, lookupErr
		}
	}
	if !assessmentKindAllowedForRelationship(kind, relationship.Relationship.Status) {
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	assessmentID, err := s.newID()
	if err != nil {
		return Assessment{}, err
	}
	assessment := Assessment{
		ID: assessmentID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		RelationshipID: relationshipID, ReviewKind: kind, SourceTrigger: sourceTrigger,
		StableEpisodeKey: stableKey,
		Status:           AssessmentSetupPending, FormTemplateID: input.FormTemplateID, FormTemplateVersion: input.FormTemplateVersion,
		ReviewDueAt: input.ReviewDueAt.UTC(), StartedByPrincipalID: actor.PrincipalID, StartedAt: now,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	return s.repo.CreateAssessment(ctx, CreateAssessmentRecord{Scope: scope, RelationshipID: relationshipID, RelationshipVersion: input.RelationshipVersion, Assessment: assessment})
}

func (s *AssessmentService) GetAssessment(ctx context.Context, actor Actor, assessmentID string) (Assessment, error) {
	if !validActor(actor) || !validAssessmentIdentifier(assessmentID) {
		return Assessment{}, ErrInvalid
	}
	return s.repo.GetAssessment(ctx, scopeFrom(actor), strings.TrimSpace(assessmentID))
}

func (s *AssessmentService) GetCurrentAssessment(ctx context.Context, actor Actor, relationshipID string) (Assessment, error) {
	if !validActor(actor) || !validAssessmentIdentifier(relationshipID) {
		return Assessment{}, ErrInvalid
	}
	var current Assessment
	found := false
	for _, kind := range []AssessmentReviewKind{AssessmentReviewOnboarding, AssessmentReviewPeriodic, AssessmentReviewTriggered} {
		candidate, err := s.repo.GetCurrentAssessment(ctx, scopeFrom(actor), strings.TrimSpace(relationshipID), kind)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return Assessment{}, err
		}
		if !found || (assessmentEpisodeActive(candidate.Status) && !assessmentEpisodeActive(current.Status)) ||
			(assessmentEpisodeActive(candidate.Status) == assessmentEpisodeActive(current.Status) && (candidate.UpdatedAt.After(current.UpdatedAt) || (candidate.UpdatedAt.Equal(current.UpdatedAt) && candidate.ID > current.ID))) {
			current, found = candidate, true
		}
	}
	if !found {
		return Assessment{}, ErrNotFound
	}
	return current, nil
}

func (s *AssessmentService) RecordAssessmentSetupCompleted(ctx context.Context, input AssessmentSetupCompletedInput) (Assessment, error) {
	if !validAssessmentScope(input.Scope) || input.ExpectedVersion < 1 || !validAssessmentIdentifiers(input.AssessmentID, input.CausationID, input.SetupJobID, input.ReviewMatterID) {
		return Assessment{}, ErrInvalid
	}
	return s.repo.ApplyAssessmentReaction(ctx, AssessmentReactionRecord{
		Scope: normalizeAssessmentScope(input.Scope), AssessmentID: strings.TrimSpace(input.AssessmentID), ExpectedVersion: input.ExpectedVersion,
		Kind: AssessmentReactionSetupCompleted, CausationID: strings.TrimSpace(input.CausationID), JobID: strings.TrimSpace(input.SetupJobID),
		MatterID: strings.TrimSpace(input.ReviewMatterID), At: s.now().UTC(),
	})
}

func (s *AssessmentService) RecordAssessmentSubmitted(ctx context.Context, input AssessmentSubmittedInput) (Assessment, error) {
	if !validAssessmentScope(input.Scope) || input.ExpectedVersion < 1 || !validAssessmentIdentifiers(input.AssessmentID, input.CausationID, input.EventID, input.RequestID, input.SubmissionID) {
		return Assessment{}, ErrInvalid
	}
	return s.repo.ApplyAssessmentReaction(ctx, AssessmentReactionRecord{
		Scope: normalizeAssessmentScope(input.Scope), AssessmentID: strings.TrimSpace(input.AssessmentID), ExpectedVersion: input.ExpectedVersion,
		Kind: AssessmentReactionSubmitted, CausationID: strings.TrimSpace(input.CausationID), EventID: strings.TrimSpace(input.EventID),
		RequestID: strings.TrimSpace(input.RequestID), SubmissionID: strings.TrimSpace(input.SubmissionID), At: s.now().UTC(),
	})
}

func (s *AssessmentService) RecordRequestIssued(ctx context.Context, _ Actor, assessmentID string, input RecordRequestIssuedInput) (RecordRequestIssuedOutcome, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	input.RequestID, input.InvitationID = strings.TrimSpace(input.RequestID), strings.TrimSpace(input.InvitationID)
	input.OriginType, input.OriginID = strings.TrimSpace(input.OriginType), strings.TrimSpace(input.OriginID)
	if !validAssessmentIdentifiers(assessmentID, input.RequestID, input.InvitationID, input.OriginID) || input.ExpectedVersion < 1 || !validAssessmentRequestPurpose(input.Purpose) || input.OriginType != AssessmentRequestOrigin || input.OriginID != assessmentID || input.OriginSequence < 1 {
		return RecordRequestIssuedOutcome{}, ErrInvalid
	}
	responsibility := authority.ResponsibilityOwner
	if input.Purpose == AssessmentRequestClarification {
		responsibility = authority.ResponsibilityReviewer
	}
	actor, err := s.authorize(ctx, assessmentID, assessmentObjectType, AssessmentRequestIssuedCommand, responsibility)
	if err != nil {
		return RecordRequestIssuedOutcome{}, err
	}
	link, assessment, err := s.repo.RecordRequestIssued(ctx, RecordRequestIssuedRecord{
		Scope: scopeFrom(actor), AssessmentID: assessmentID, ExpectedVersion: input.ExpectedVersion, ActorPrincipalID: actor.PrincipalID,
		RequestID: input.RequestID, Purpose: input.Purpose, OriginType: input.OriginType, OriginID: input.OriginID,
		OriginSequence: input.OriginSequence, InvitationID: input.InvitationID, IssuedAt: s.now().UTC(),
	})
	if err != nil {
		return RecordRequestIssuedOutcome{}, err
	}
	return RecordRequestIssuedOutcome{Assessment: assessment, Link: link}, nil
}

func (s *AssessmentService) StartAssessmentReview(ctx context.Context, _ Actor, assessmentID string, expectedVersion int64) (Assessment, error) {
	return s.transition(ctx, assessmentID, expectedVersion, []AssessmentStatus{AssessmentSubmitted}, AssessmentUnderReview, AssessmentReviewCommand, authority.ResponsibilityReviewer, nil)
}

func (s *AssessmentService) CompleteAssessment(ctx context.Context, _ Actor, assessmentID string, input CompleteAssessmentInput) (Assessment, error) {
	input.Rationale = strings.TrimSpace(input.Rationale)
	input.Uncertainty = strings.TrimSpace(input.Uncertainty)
	now := s.now().UTC()
	if !validAssessmentConclusion(input.Conclusion) || input.Rationale == "" || len(input.Rationale) > assessmentMaxRationaleLength || len(input.Uncertainty) > assessmentMaxUncertaintyLength {
		return Assessment{}, ErrInvalid
	}
	if input.NextReviewRecommendedAt != nil {
		value := input.NextReviewRecommendedAt.UTC()
		if !value.After(now) {
			return Assessment{}, ErrInvalid
		}
		input.NextReviewRecommendedAt = &value
	}
	assessmentID = strings.TrimSpace(assessmentID)
	if !validAssessmentIdentifier(assessmentID) || input.ExpectedVersion < 1 {
		return Assessment{}, ErrInvalid
	}
	actor, err := s.authorize(ctx, assessmentID, assessmentObjectType, AssessmentCompleteCommand, authority.ResponsibilityReviewer)
	if err != nil {
		return Assessment{}, err
	}
	current, err := s.repo.GetAssessment(ctx, scopeFrom(actor), assessmentID)
	if err != nil {
		return Assessment{}, err
	}
	if current.Version != input.ExpectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentUnderReview {
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	if s.readiness == nil {
		return Assessment{}, ErrAssessmentReadinessUnavailable
	}
	commandActor := Actor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID}
	if err := s.readiness.CheckAssessmentCompletion(ctx, commandActor, assessmentID); err != nil {
		return Assessment{}, err
	}
	record := AssessmentTransitionRecord{
		Scope: scopeFrom(actor), ID: assessmentID, ExpectedVersion: input.ExpectedVersion, From: []AssessmentStatus{AssessmentUnderReview}, To: AssessmentCompleted,
		At: now, ActorPrincipalID: actor.PrincipalID, Conclusion: input.Conclusion, ConclusionUncertainty: input.Uncertainty,
		ConclusionRationale: input.Rationale, NextReviewRecommendedAt: input.NextReviewRecommendedAt,
	}
	return s.repo.TransitionAssessment(ctx, record)
}

func (s *AssessmentService) CancelAssessment(ctx context.Context, _ Actor, assessmentID string, input CancelAssessmentInput) (Assessment, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Reason == "" || len(input.Reason) > assessmentMaxCancelReasonLength {
		return Assessment{}, ErrInvalid
	}
	cancelled, err := s.transition(ctx, assessmentID, input.ExpectedVersion, []AssessmentStatus{AssessmentSetupPending, AssessmentReadyToSend, AssessmentCollecting, AssessmentSubmitted, AssessmentUnderReview}, AssessmentCancelled, AssessmentCancelCommand, authority.ResponsibilityOwner, func(record *AssessmentTransitionRecord) {
		record.CancellationReason = input.Reason
	})
	if err != nil {
		return Assessment{}, err
	}
	// The cancellation and its outbox event are already committed. Immediate
	// revocation narrows the exposure window; a worker retries from the event.
	if s.cancellationRevoker != nil && cancelled.CurrentRequestID != "" {
		_ = s.cancellationRevoker.RevokeRequestCapabilities(ctx, cancelled.TenantID, cancelled.CurrentRequestID)
	}
	return cancelled, nil
}

func (s *AssessmentService) transition(ctx context.Context, assessmentID string, expectedVersion int64, from []AssessmentStatus, to AssessmentStatus, command string, responsibility authority.Responsibility, amend func(*AssessmentTransitionRecord)) (Assessment, error) {
	return s.transitionAt(ctx, assessmentID, expectedVersion, from, to, command, responsibility, s.now().UTC(), amend)
}

func (s *AssessmentService) transitionAt(ctx context.Context, assessmentID string, expectedVersion int64, from []AssessmentStatus, to AssessmentStatus, command string, responsibility authority.Responsibility, at time.Time, amend func(*AssessmentTransitionRecord)) (Assessment, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if !validAssessmentIdentifier(assessmentID) || expectedVersion < 1 {
		return Assessment{}, ErrInvalid
	}
	actor, err := s.authorize(ctx, assessmentID, assessmentObjectType, command, responsibility)
	if err != nil {
		return Assessment{}, err
	}
	current, err := s.repo.GetAssessment(ctx, scopeFrom(actor), assessmentID)
	if err != nil {
		return Assessment{}, err
	}
	if current.Version != expectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	if !containsAssessmentStatus(from, current.Status) {
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	record := AssessmentTransitionRecord{Scope: scopeFrom(actor), ID: assessmentID, ExpectedVersion: expectedVersion, From: from, To: to, At: at.UTC(), ActorPrincipalID: actor.PrincipalID}
	if amend != nil {
		amend(&record)
	}
	return s.repo.TransitionAssessment(ctx, record)
}

func (s *AssessmentService) authorize(ctx context.Context, objectID, objectType, command string, responsibility authority.Responsibility) (Actor, error) {
	contextActor, err := identity.Require(ctx)
	if err != nil {
		return Actor{}, err
	}
	if err := contextActor.Valid(s.now().UTC()); err != nil || contextActor.LegalEntityID == "*" {
		if err != nil {
			return Actor{}, err
		}
		return Actor{}, identity.ErrInvalidIdentity
	}
	if s.guard == nil {
		return Actor{}, errors.Join(ErrAssessmentAuthorityUnavailable, commandauth.ErrGuardUnavailable)
	}
	decision, err := s.guard.Authorize(ctx, commandauth.Request{
		TenantID: contextActor.TenantID, LegalEntityID: contextActor.LegalEntityID, ObjectType: objectType,
		ObjectID: objectID, Responsibility: responsibility, DecisionType: command, Materiality: 3,
	})
	if err != nil {
		return Actor{}, err
	}
	if !decision.Allowed {
		return Actor{}, commandauth.ErrNotAuthorized
	}
	if err := decision.Actor.Valid(s.now().UTC()); err != nil || !sameAssessmentIdentity(contextActor, decision.Actor) {
		return Actor{}, ErrAssessmentIdentityMismatch
	}
	return Actor{TenantID: decision.Actor.TenantID, LegalEntityID: decision.Actor.LegalEntityID, PrincipalID: decision.Actor.PrincipalID}, nil
}

func assessmentEpisodeKey(scope Scope, relationshipID string, kind AssessmentReviewKind, sourceTrigger ...string) string {
	trigger := "INITIAL"
	if len(sourceTrigger) > 0 && strings.TrimSpace(sourceTrigger[0]) != "" {
		trigger = strings.TrimSpace(sourceTrigger[0])
	}
	value := strings.Join([]string{scope.TenantID, scope.LegalEntityID, relationshipID, string(kind), trigger}, "\x00")
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func normalizeAssessmentEpisode(kind AssessmentReviewKind, sourceTrigger, restartAssessmentID string) (AssessmentReviewKind, string, string, error) {
	if kind == "" {
		kind = AssessmentReviewOnboarding
	}
	sourceTrigger = strings.TrimSpace(sourceTrigger)
	restartAssessmentID = strings.TrimSpace(restartAssessmentID)
	switch kind {
	case AssessmentReviewOnboarding:
		if restartAssessmentID != "" {
			if sourceTrigger != "" || !validAssessmentIdentifier(restartAssessmentID) {
				return "", "", "", ErrInvalid
			}
			return kind, "", restartAssessmentID, nil
		}
		if sourceTrigger == "" {
			sourceTrigger = "INITIAL"
		}
		if sourceTrigger != "INITIAL" {
			return "", "", "", ErrInvalid
		}
	case AssessmentReviewPeriodic, AssessmentReviewTriggered:
		if restartAssessmentID != "" || !validAssessmentIdentifier(sourceTrigger) || sourceTrigger == "INITIAL" {
			return "", "", "", ErrInvalid
		}
	default:
		return "", "", "", ErrInvalid
	}
	return kind, sourceTrigger, "", nil
}

func assessmentEpisodeActive(status AssessmentStatus) bool {
	return status != AssessmentCompleted && status != AssessmentCancelled
}

func assessmentKindAllowedForRelationship(kind AssessmentReviewKind, status RelationshipStatus) bool {
	if kind == AssessmentReviewOnboarding {
		return status == RelationshipProposed || status == RelationshipUnderReview
	}
	return status == RelationshipActive || status == RelationshipRestricted || status == RelationshipSuspended
}

func validAssessmentConclusion(value AssessmentConclusion) bool {
	return value == AssessmentSatisfactory || value == AssessmentSatisfactoryWithConditions || value == AssessmentUnsatisfactory || value == AssessmentInconclusive
}

func validAssessmentRequestPurpose(value AssessmentRequestPurpose) bool {
	return value == AssessmentRequestInitial || value == AssessmentRequestClarification
}

func containsAssessmentStatus(values []AssessmentStatus, target AssessmentStatus) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validAssessmentScope(scope Scope) bool {
	return validAssessmentIdentifiers(scope.TenantID, scope.LegalEntityID) && strings.TrimSpace(scope.LegalEntityID) != "*"
}

func normalizeAssessmentScope(scope Scope) Scope {
	return Scope{TenantID: strings.TrimSpace(scope.TenantID), LegalEntityID: strings.TrimSpace(scope.LegalEntityID)}
}

func validAssessmentIdentifiers(values ...string) bool {
	for _, value := range values {
		if !validAssessmentIdentifier(value) {
			return false
		}
	}
	return true
}

func validAssessmentIdentifier(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > assessmentMaxIdentifierLength {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func sameAssessmentIdentity(left, right identity.Actor) bool {
	return left.TenantID == right.TenantID && left.LegalEntityID == right.LegalEntityID && left.PrincipalID == right.PrincipalID
}
