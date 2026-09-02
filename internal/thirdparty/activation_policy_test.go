package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type activationGuard struct {
	actor identity.Actor
}

type revokedDecisionAuthorityGuard struct{}

func (revokedDecisionAuthorityGuard) Authorize(ctx context.Context, request commandauth.Request) (commandauth.Decision, error) {
	actor, _ := identity.FromContext(ctx)
	if request.DecisionType == "matter.decision.record" && actor.PrincipalID == "former-authorizer" {
		return commandauth.Decision{}, commandauth.ErrNotAuthorized
	}
	return commandauth.Decision{Allowed: true, Enforced: true, Actor: actor}, nil
}

func (g activationGuard) Authorize(_ context.Context, request commandauth.Request) (commandauth.Decision, error) {
	if request.TenantID != g.actor.TenantID || request.LegalEntityID != g.actor.LegalEntityID {
		return commandauth.Decision{}, commandauth.ErrTenantMismatch
	}
	return commandauth.Decision{Allowed: true, Enforced: true, Actor: g.actor}, nil
}

func activationContext(actor identity.Actor) context.Context {
	return identity.WithActor(context.Background(), actor)
}

func TestActivationPolicyRequiresIndependentApprovalAndEffectiveDating(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryActivationRepository()
	maker := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker", Kind: "HUMAN"}
	service := NewActivationService(repo, activationGuard{actor: maker})
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "policy-1", nil }

	policy, err := service.ProposePolicy(activationContext(maker), ProposeActivationPolicyInput{
		LegalEntityID: "entity", AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory},
		MaximumAssessmentAgeDays: 90, AddressVerificationRequired: true,
		BlockingMatterTypes: []string{"VENDOR_DEFICIENCY"}, EffectiveFrom: now.Add(time.Hour),
		Rationale: "Require current vendor review and independently verified address evidence.",
	})
	if err != nil || policy.Status != ActivationPolicyDraft || policy.Version != 1 {
		t.Fatalf("propose policy: %+v %v", policy, err)
	}
	policy, err = service.SubmitPolicy(activationContext(maker), policy.ID, policy.Version, "Submit the reviewed activation gates for independent approval.")
	if err != nil || policy.Status != ActivationPolicyPendingApproval || policy.Version != 2 {
		t.Fatalf("submit policy: %+v %v", policy, err)
	}
	if _, err := service.ApprovePolicy(activationContext(maker), policy.ID, policy.Version, "", "Approve the reviewed policy for the intended effective time."); !errors.Is(err, ErrActivationMakerChecker) {
		t.Fatalf("maker approved own policy: %v", err)
	}
	simulation, err := service.SimulatePolicy(activationContext(maker), policy.ID)
	if err != nil || simulation.PolicyVersion != policy.Version || simulation.ID == "" {
		t.Fatalf("simulate policy: %+v %v", simulation, err)
	}

	checker := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker", Kind: "HUMAN"}
	service.guard = activationGuard{actor: checker}
	policy, err = service.ApprovePolicy(activationContext(checker), policy.ID, policy.Version, simulation.ID, "The current gates and effective date are approved.")
	if err != nil || policy.Status != ActivationPolicyActive || policy.ApprovedBy != checker.PrincipalID || policy.Version != 3 {
		t.Fatalf("approve policy: %+v %v", policy, err)
	}
	if _, err := service.CurrentPolicy(activationContext(checker), "entity", now); !errors.Is(err, ErrActivationPolicyUnavailable) {
		t.Fatalf("future policy was selected early: %v", err)
	}
	current, err := service.CurrentPolicy(activationContext(checker), "entity", now.Add(2*time.Hour))
	if err != nil || current.ID != policy.ID {
		t.Fatalf("effective policy unavailable: %+v %v", current, err)
	}
}

func TestFutureActivationPolicyKeepsIncumbentEffectiveUntilReplacementStarts(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryActivationRepository()
	repo.PutPolicyForTest(ActivationPolicy{
		ID: "incumbent", TenantID: "bank", LegalEntityID: "entity", PolicyNumber: 1,
		AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory}, MaximumAssessmentAgeDays: 90,
		Status: ActivationPolicyActive, EffectiveFrom: now.Add(-24 * time.Hour), ProposedBy: "prior-maker", ApprovedBy: "prior-checker",
		ProposalRationale: "Keep the current independently approved activation gates in force.", ApprovalRationale: "Approved for current vendor activation decisions.",
		CreatedAt: now.Add(-48 * time.Hour), UpdatedAt: now.Add(-24 * time.Hour), Version: 3,
	})
	maker := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker", Kind: "HUMAN"}
	service := NewActivationService(repo, activationGuard{actor: maker})
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "replacement", nil }
	replacement, err := service.ProposePolicy(activationContext(maker), ProposeActivationPolicyInput{
		LegalEntityID: "entity", AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory}, MaximumAssessmentAgeDays: 30,
		EffectiveFrom: now.Add(24 * time.Hour), Rationale: "Tighten the evidence age after an independently reviewed future boundary.",
	})
	if err != nil {
		t.Fatal(err)
	}
	replacement, err = service.SubmitPolicy(activationContext(maker), replacement.ID, replacement.Version, "Submit the future replacement for independent approval.")
	if err != nil {
		t.Fatal(err)
	}
	simulation, err := service.SimulatePolicy(activationContext(maker), replacement.ID)
	if err != nil {
		t.Fatal(err)
	}
	checker := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "checker", Kind: "HUMAN"}
	service.guard = activationGuard{actor: checker}
	if _, err = service.ApprovePolicy(activationContext(checker), replacement.ID, replacement.Version, simulation.ID, "Approve the future replacement after reviewing its gates and boundary."); err != nil {
		t.Fatal(err)
	}
	current, err := service.CurrentPolicy(activationContext(checker), "entity", now)
	if err != nil || current.ID != "incumbent" {
		t.Fatalf("incumbent unavailable before replacement: %+v %v", current, err)
	}
	current, err = service.CurrentPolicy(activationContext(checker), "entity", now.Add(25*time.Hour))
	if err != nil || current.ID != "replacement" {
		t.Fatalf("replacement unavailable after boundary: %+v %v", current, err)
	}
}

func TestActivationPolicyRollbackCreatesGovernedDraftWithoutChangingCurrentPolicy(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryActivationRepository()
	source := ActivationPolicy{
		ID: "prior-policy", TenantID: "bank", LegalEntityID: "entity", PolicyNumber: 1,
		AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory}, MaximumAssessmentAgeDays: 60,
		RequiredDecisionTypes: []string{"VENDOR_APPROVAL"}, AddressVerificationRequired: true,
		Status: ActivationPolicyRetired, EffectiveFrom: now.Add(-30 * 24 * time.Hour), ProposedBy: "prior-maker", ApprovedBy: "prior-checker",
		ProposalRationale: "Use the reviewed vendor activation gates.", ApprovalRationale: "The gates were independently approved.",
		CreatedAt: now.Add(-40 * 24 * time.Hour), UpdatedAt: now.Add(-10 * 24 * time.Hour), Version: 4,
	}
	current := source
	current.ID, current.PolicyNumber, current.Status, current.EffectiveFrom, current.EffectiveUntil, current.Version = "current-policy", 2, ActivationPolicyActive, now.Add(-10*24*time.Hour), nil, 3
	repo.PutPolicyForTest(source)
	repo.PutPolicyForTest(current)
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker", Kind: "HUMAN"}
	service := NewActivationService(repo, activationGuard{actor: actor})
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "rollback-draft", nil }
	draft, err := service.PrepareRollback(activationContext(actor), source.ID, RollbackActivationPolicyInput{
		EffectiveFrom: now.Add(24 * time.Hour), Rationale: "Restore the prior approved gates through the normal simulation and approval route.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if draft.Status != ActivationPolicyDraft || draft.RollbackOfPolicyID != source.ID || draft.ID != "rollback-draft" || draft.ApprovedBy != "" || draft.PolicyNumber != 3 {
		t.Fatalf("rollback draft = %+v", draft)
	}
	stillCurrent, err := service.CurrentPolicy(activationContext(actor), "entity", now)
	if err != nil || stillCurrent.ID != current.ID {
		t.Fatalf("rollback draft changed current policy: %+v %v", stillCurrent, err)
	}
}

func TestActivationFailsClosedUntilEveryStoredGatePasses(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryActivationRepository()
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "authorizer", Kind: "HUMAN"}
	service := NewActivationService(repo, activationGuard{actor: actor})
	service.now = func() time.Time { return now }
	repo.PutPolicyForTest(ActivationPolicy{
		ID: "policy", TenantID: "bank", LegalEntityID: "entity", PolicyNumber: 1,
		AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory}, MaximumAssessmentAgeDays: 90,
		AddressVerificationRequired: true, Status: ActivationPolicyActive, EffectiveFrom: now.Add(-time.Hour),
		ProposedBy: "maker", ApprovedBy: "checker", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour), Version: 3,
	})
	repo.PutRelationshipForTest(Relationship{ID: "relationship", TenantID: "bank", LegalEntityID: "entity", Status: RelationshipProposed, Version: 1})
	repo.PutActivationFactsForTest("relationship", ActivationFacts{})

	result, err := service.ActivateRelationship(activationContext(actor), "relationship", ActivateRelationshipInput{
		ExpectedVersion: 1, IntendedEffectiveAt: now, Rationale: "Activate only after the complete current evidence and outcome checks pass.",
	})
	if !errors.Is(err, ErrActivationIneligible) || result.Eligible || len(result.Gates) == 0 {
		t.Fatalf("missing gates did not fail closed: %+v %v", result, err)
	}
	if got := repo.RelationshipForTest("relationship"); got.Status != RelationshipProposed || got.Version != 1 {
		t.Fatalf("ineligible activation mutated relationship: %+v", got)
	}

	repo.PutActivationFactsForTest("relationship", ActivationFacts{
		AssessmentID: "assessment", AssessmentVersion: 4, AssessmentStatus: AssessmentCompleted,
		AssessmentConclusion: AssessmentSatisfactory, AssessmentCompletedAt: now.Add(-24 * time.Hour),
		AddressMatterID: "matter", AddressMatterClosed: true, VerificationResultID: "result", VerificationPassed: true,
	})
	result, err = service.ActivateRelationship(activationContext(actor), "relationship", ActivateRelationshipInput{
		ExpectedVersion: 1, IntendedEffectiveAt: now, Rationale: "Activate after the current assessment and independent address outcome have passed.",
	})
	if err != nil || !result.Eligible || result.Relationship.Status != RelationshipActive || result.Relationship.Version != 2 {
		t.Fatalf("eligible activation failed: %+v %v", result, err)
	}
	if result.Receipt.PolicyID != "policy" || result.Receipt.AssessmentID != "assessment" || result.Receipt.VerificationResultID != "result" {
		t.Fatalf("activation receipt lacks exact dependencies: %+v", result.Receipt)
	}
}

func TestActivationRejectsApprovedDecisionWhoseAuthorityWasRevoked(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryActivationRepository()
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "current-authorizer", Kind: "HUMAN"}
	service := NewActivationService(repo, revokedDecisionAuthorityGuard{})
	service.now = func() time.Time { return now }
	repo.PutPolicyForTest(ActivationPolicy{
		ID: "policy", TenantID: "bank", LegalEntityID: "entity", PolicyNumber: 1,
		AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory}, MaximumAssessmentAgeDays: 90,
		RequiredDecisionTypes: []string{"VENDOR_APPROVAL"}, Status: ActivationPolicyActive, EffectiveFrom: now.Add(-time.Hour),
		ProposedBy: "maker", ApprovedBy: "checker", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour), Version: 3,
	})
	repo.PutRelationshipForTest(Relationship{ID: "relationship", TenantID: "bank", LegalEntityID: "entity", Status: RelationshipUnderReview, Version: 1})
	repo.PutActivationFactsForTest("relationship", ActivationFacts{
		AssessmentID: "assessment", AssessmentVersion: 3, AssessmentStatus: AssessmentCompleted, AssessmentConclusion: AssessmentSatisfactory, AssessmentCompletedAt: now.Add(-time.Hour),
		SatisfiedDecisionTypes: []string{"VENDOR_APPROVAL"}, DecisionIDs: []string{"decision"},
		DecisionDependencies: []ActivationDecisionDependency{{ID: "decision", MatterID: "review-matter", DecisionType: "VENDOR_APPROVAL", AuthorityPrincipalID: "former-authorizer"}},
	})
	result, err := service.ActivateRelationship(activationContext(actor), "relationship", ActivateRelationshipInput{ExpectedVersion: 1, IntendedEffectiveAt: now, Rationale: "Activate only after every current authority and evidence gate passes."})
	if !errors.Is(err, ErrActivationIneligible) || result.Eligible {
		t.Fatalf("revoked decision authority permitted activation: %+v %v", result, err)
	}
	found := false
	for _, gate := range result.Gates {
		if gate.Code == "DECISION_AUTHORITY" && !gate.Satisfied {
			found = true
		}
	}
	if !found {
		t.Fatalf("decision authority gate missing: %+v", result.Gates)
	}
}

func TestActivationUsesVerifiedIdentityNotActorPayload(t *testing.T) {
	now := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	repo := NewMemoryActivationRepository()
	verified := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "verified", Kind: "HUMAN"}
	service := NewActivationService(repo, activationGuard{actor: verified})
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "policy", nil }
	policy, err := service.ProposePolicy(activationContext(verified), ProposeActivationPolicyInput{
		LegalEntityID: "entity", AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory}, MaximumAssessmentAgeDays: 30,
		EffectiveFrom: now, Rationale: "Use the verified maker identity for this policy proposal.",
	})
	if err != nil || policy.ProposedBy != verified.PrincipalID {
		t.Fatalf("policy did not use verified identity: %+v %v", policy, err)
	}
	if service.lastAuthorizationResponsibility != authority.ResponsibilityOwner {
		t.Fatalf("proposal used wrong current authority: %s", service.lastAuthorizationResponsibility)
	}
}

func TestMemoryActivationRepositoryUsesCanonicalRelationshipAndAssessmentState(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	source := NewMemoryAssessmentRepository()
	scope := Scope{TenantID: "tenant-1", LegalEntityID: "entity-1"}
	_, err := source.CreateRelationship(context.Background(), CreateRecord{
		Vendor:       Vendor{ID: "vendor-1", TenantID: scope.TenantID, LegalName: "Example Vendor", CreatedAt: now, UpdatedAt: now, Version: 1},
		Relationship: Relationship{ID: "relationship-1", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, VendorID: "vendor-1", Status: RelationshipUnderReview, CreatedAt: now, UpdatedAt: now, Version: 1},
	})
	if err != nil {
		t.Fatalf("create relationship: %v", err)
	}
	completedAt := now.Add(-24 * time.Hour)
	source.assessments["assessment-1"] = Assessment{ID: "assessment-1", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, RelationshipID: "relationship-1", ReviewKind: AssessmentReviewOnboarding, Status: AssessmentCompleted, Conclusion: AssessmentSatisfactory, CompletedAt: &completedAt, UpdatedAt: completedAt, Version: 3}

	repo := NewMemoryActivationRepository(source)
	policy := ActivationPolicy{ID: "policy-1", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, PolicyNumber: 1, AllowedConclusions: []AssessmentConclusion{AssessmentSatisfactory}, MaximumAssessmentAgeDays: 30, AddressVerificationRequired: true, Status: ActivationPolicyActive, EffectiveFrom: now.Add(-time.Hour), ProposedBy: "maker", ApprovedBy: "checker", CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-time.Hour), Version: 3}
	repo.PutPolicyForTest(policy)
	repo.PutActivationFactsForTest("relationship-1", ActivationFacts{AddressMatterClosed: true, AddressMatterID: "matter-1", VerificationResultID: "verification-1", VerificationPassed: true})

	relationship, facts, err := repo.ReadActivationFacts(context.Background(), scope, "relationship-1", policy)
	if err != nil {
		t.Fatalf("read activation facts: %v", err)
	}
	if relationship.Version != 1 || facts.AssessmentID != "assessment-1" || facts.AssessmentVersion != 3 {
		t.Fatalf("expected canonical relationship and assessment, got relationship=%+v facts=%+v", relationship, facts)
	}
	activated, _, err := repo.CommitRelationshipActivation(context.Background(), ActivationCommit{
		ReceiptID: "receipt-1", Scope: scope, RelationshipID: relationship.ID, ExpectedVersion: relationship.Version,
		Policy: policy, Facts: facts, ActorID: "actor-1", EffectiveAt: now, Rationale: "All policy gates have been independently reviewed.",
	})
	if err != nil {
		t.Fatalf("commit activation: %v", err)
	}
	stored, err := source.GetRelationship(context.Background(), scope, relationship.ID)
	if err != nil {
		t.Fatalf("get canonical relationship: %v", err)
	}
	if activated.Status != RelationshipActive || stored.Relationship.Status != RelationshipActive || stored.Relationship.Version != 2 {
		t.Fatalf("expected canonical relationship activation, got returned=%+v stored=%+v", activated, stored.Relationship)
	}
}
