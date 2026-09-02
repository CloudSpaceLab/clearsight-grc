package thirdparty

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type ActivationPolicyStatus string

const (
	ActivationPolicyDraft           ActivationPolicyStatus = "DRAFT"
	ActivationPolicyPendingApproval ActivationPolicyStatus = "PENDING_APPROVAL"
	ActivationPolicyActive          ActivationPolicyStatus = "ACTIVE"
	ActivationPolicyRetired         ActivationPolicyStatus = "RETIRED"

	ActivationPolicyProposeCommand  = "thirdparty.activation_policy.propose"
	ActivationPolicySimulateCommand = "thirdparty.activation_policy.simulate"
	ActivationPolicySubmitCommand   = "thirdparty.activation_policy.submit"
	ActivationPolicyApproveCommand  = "thirdparty.activation_policy.approve"
	ActivationPolicyRollbackCommand = "thirdparty.activation_policy.rollback"
	RelationshipActivateCommand     = "thirdparty.relationship.activate"
)

var (
	ErrActivationPolicyUnavailable  = errors.New("third-party activation policy is unavailable")
	ErrActivationMakerChecker       = errors.New("activation policy requires an independent checker")
	ErrActivationSimulationRequired = errors.New("a current complete activation policy simulation is required")
	ErrActivationIneligible         = errors.New("vendor relationship is not eligible for activation")
)

type ActivationPolicy struct {
	ID                              string                 `json:"id"`
	TenantID                        string                 `json:"tenant_id"`
	LegalEntityID                   string                 `json:"legal_entity_id"`
	PolicyNumber                    int                    `json:"policy_number"`
	AllowedConclusions              []AssessmentConclusion `json:"allowed_conclusions"`
	MaximumAssessmentAgeDays        int                    `json:"maximum_assessment_age_days"`
	RequiredDecisionTypes           []string               `json:"required_decision_types"`
	AddressVerificationRequired     bool                   `json:"address_verification_required"`
	BlockingMatterTypes             []string               `json:"blocking_matter_types"`
	ConditionalConclusionNeedsTerms bool                   `json:"conditional_conclusion_needs_terms"`
	EffectiveFrom                   time.Time              `json:"effective_from"`
	EffectiveUntil                  *time.Time             `json:"effective_until,omitempty"`
	RollbackOfPolicyID              string                 `json:"rollback_of_policy_id,omitempty"`
	Status                          ActivationPolicyStatus `json:"status"`
	ProposedBy                      string                 `json:"proposed_by"`
	ApprovedBy                      string                 `json:"approved_by,omitempty"`
	ProposalRationale               string                 `json:"proposal_rationale"`
	ApprovalRationale               string                 `json:"approval_rationale,omitempty"`
	CreatedAt                       time.Time              `json:"created_at"`
	UpdatedAt                       time.Time              `json:"updated_at"`
	Version                         int64                  `json:"version"`
}

type ProposeActivationPolicyInput struct {
	LegalEntityID                   string                 `json:"legal_entity_id"`
	AllowedConclusions              []AssessmentConclusion `json:"allowed_conclusions"`
	MaximumAssessmentAgeDays        int                    `json:"maximum_assessment_age_days"`
	RequiredDecisionTypes           []string               `json:"required_decision_types,omitempty"`
	AddressVerificationRequired     bool                   `json:"address_verification_required"`
	BlockingMatterTypes             []string               `json:"blocking_matter_types,omitempty"`
	ConditionalConclusionNeedsTerms bool                   `json:"conditional_conclusion_needs_terms"`
	EffectiveFrom                   time.Time              `json:"effective_from"`
	Rationale                       string                 `json:"rationale"`
}

type RollbackActivationPolicyInput struct {
	EffectiveFrom time.Time `json:"effective_from"`
	Rationale     string    `json:"rationale"`
}

type ActivationSimulation struct {
	ID                   string         `json:"id"`
	PolicyID             string         `json:"policy_id"`
	PolicyVersion        int64          `json:"policy_version"`
	CandidateCount       int            `json:"candidate_count"`
	EligibleCount        int            `json:"eligible_count"`
	MissingGateCounts    map[string]int `json:"missing_gate_counts"`
	EvaluatedAt          time.Time      `json:"evaluated_at"`
	PopulationIsComplete bool           `json:"population_is_complete"`
	EvaluatedBy          string         `json:"evaluated_by"`
	ExpiresAt            time.Time      `json:"expires_at"`
}

type ActivationFacts struct {
	AssessmentID               string
	AssessmentVersion          int64
	AssessmentStatus           AssessmentStatus
	AssessmentConclusion       AssessmentConclusion
	AssessmentCompletedAt      time.Time
	ConditionsRecorded         bool
	SatisfiedDecisionTypes     []string
	DecisionIDs                []string
	DecisionDependencies       []ActivationDecisionDependency
	DecisionAuthoritiesCurrent bool
	AddressMatterID            string
	AddressMatterClosed        bool
	VerificationResultID       string
	VerificationPassed         bool
	HasBlockingMatter          bool
	HasUnresolvedContradiction bool
}

type ActivationDecisionDependency struct {
	ID                   string
	MatterID             string
	DecisionType         string
	AuthorityPrincipalID string
}

type ActivationGate struct {
	Code        string `json:"code"`
	Satisfied   bool   `json:"satisfied"`
	Explanation string `json:"explanation"`
}

type ActivateRelationshipInput struct {
	ExpectedVersion     int64     `json:"expected_version"`
	IntendedEffectiveAt time.Time `json:"intended_effective_at"`
	Rationale           string    `json:"rationale"`
}

type ActivationReceipt struct {
	ID                   string    `json:"id"`
	TenantID             string    `json:"tenant_id"`
	LegalEntityID        string    `json:"legal_entity_id"`
	RelationshipID       string    `json:"relationship_id"`
	RelationshipVersion  int64     `json:"relationship_version"`
	PolicyID             string    `json:"policy_id"`
	PolicyVersion        int64     `json:"policy_version"`
	AssessmentID         string    `json:"assessment_id"`
	AssessmentVersion    int64     `json:"assessment_version"`
	DecisionIDs          []string  `json:"decision_ids,omitempty"`
	AddressMatterID      string    `json:"address_matter_id,omitempty"`
	VerificationResultID string    `json:"verification_result_id,omitempty"`
	ActivatedBy          string    `json:"activated_by"`
	ActivatedAt          time.Time `json:"activated_at"`
	Rationale            string    `json:"rationale"`
}

type ActivationResult struct {
	Eligible     bool              `json:"eligible"`
	Policy       ActivationPolicy  `json:"policy"`
	Gates        []ActivationGate  `json:"gates"`
	Relationship Relationship      `json:"relationship"`
	Receipt      ActivationReceipt `json:"receipt,omitempty"`
}

type ActivationCommit struct {
	Scope
	RelationshipID  string
	ExpectedVersion int64
	Policy          ActivationPolicy
	Facts           ActivationFacts
	ActorID         string
	Rationale       string
	EffectiveAt     time.Time
	ReceiptID       string
}

type ActivationRepository interface {
	ProposeActivationPolicy(context.Context, ActivationPolicy) (ActivationPolicy, error)
	TransitionActivationPolicy(context.Context, Scope, string, int64, ActivationPolicyStatus, string, string, time.Time) (ActivationPolicy, error)
	StoreActivationSimulation(context.Context, Scope, ActivationSimulation) (ActivationSimulation, error)
	GetActivationSimulation(context.Context, Scope, string) (ActivationSimulation, error)
	GetActivationPolicy(context.Context, Scope, string) (ActivationPolicy, error)
	CurrentActivationPolicy(context.Context, Scope, time.Time) (ActivationPolicy, error)
	ListActivationCandidates(context.Context, Scope, int) ([]Relationship, error)
	ReadActivationFacts(context.Context, Scope, string, ActivationPolicy) (Relationship, ActivationFacts, error)
	CommitRelationshipActivation(context.Context, ActivationCommit) (Relationship, ActivationReceipt, error)
}

type ActivationService struct {
	repo  ActivationRepository
	guard AssessmentCommandGuard
	now   func() time.Time
	newID func() (string, error)

	lastAuthorizationResponsibility authority.Responsibility
}

func NewActivationService(repo ActivationRepository, guard AssessmentCommandGuard) *ActivationService {
	return &ActivationService{repo: repo, guard: guard, now: time.Now, newID: id.NewUUIDv7}
}

func (s *ActivationService) ProposePolicy(ctx context.Context, input ProposeActivationPolicyInput) (ActivationPolicy, error) {
	actor, err := s.authorize(ctx, strings.TrimSpace(input.LegalEntityID), "THIRD_PARTY_ACTIVATION_POLICY", strings.TrimSpace(input.LegalEntityID), ActivationPolicyProposeCommand, authority.ResponsibilityOwner)
	if err != nil {
		return ActivationPolicy{}, err
	}
	input.LegalEntityID = actor.LegalEntityID
	if err := validateActivationPolicyInput(input); err != nil {
		return ActivationPolicy{}, err
	}
	policyID, err := s.newID()
	if err != nil {
		return ActivationPolicy{}, err
	}
	now := s.currentTime()
	policy := ActivationPolicy{
		ID: policyID, TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID,
		AllowedConclusions: normalizeConclusions(input.AllowedConclusions), MaximumAssessmentAgeDays: input.MaximumAssessmentAgeDays,
		RequiredDecisionTypes: normalizeCodes(input.RequiredDecisionTypes), AddressVerificationRequired: input.AddressVerificationRequired,
		BlockingMatterTypes: normalizeCodes(input.BlockingMatterTypes), ConditionalConclusionNeedsTerms: input.ConditionalConclusionNeedsTerms,
		EffectiveFrom: input.EffectiveFrom.UTC(), Status: ActivationPolicyDraft, ProposedBy: actor.PrincipalID,
		ProposalRationale: strings.TrimSpace(input.Rationale), CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	return s.repo.ProposeActivationPolicy(ctx, policy)
}

func (s *ActivationService) SubmitPolicy(ctx context.Context, policyID string, expectedVersion int64, rationale string) (ActivationPolicy, error) {
	actor, scope, policy, err := s.authorizePolicy(ctx, policyID, ActivationPolicySubmitCommand, authority.ResponsibilityOwner)
	if err != nil {
		return ActivationPolicy{}, err
	}
	if policy.Status != ActivationPolicyDraft || expectedVersion != policy.Version || strings.TrimSpace(rationale) == "" {
		return ActivationPolicy{}, ErrVersionConflict
	}
	return s.repo.TransitionActivationPolicy(ctx, scope, policy.ID, expectedVersion, ActivationPolicyPendingApproval, actor.PrincipalID, strings.TrimSpace(rationale), s.currentTime())
}

// PrepareRollback restores an exact approved configuration as a new draft.
// It cannot affect the current policy until the normal simulation,
// maker-checker approval and effective-dating gates complete.
func (s *ActivationService) PrepareRollback(ctx context.Context, sourcePolicyID string, input RollbackActivationPolicyInput) (ActivationPolicy, error) {
	actor, scope, source, err := s.authorizePolicy(ctx, sourcePolicyID, ActivationPolicyRollbackCommand, authority.ResponsibilityOwner)
	if err != nil {
		return ActivationPolicy{}, err
	}
	rationale := strings.TrimSpace(input.Rationale)
	if source.ApprovedBy == "" || input.EffectiveFrom.IsZero() || input.EffectiveFrom.Before(s.currentTime()) || len(rationale) < 20 {
		return ActivationPolicy{}, ErrInvalid
	}
	policyID, err := s.newID()
	if err != nil {
		return ActivationPolicy{}, err
	}
	now := s.currentTime()
	rollback := cloneActivationPolicy(source)
	rollback.ID, rollback.PolicyNumber, rollback.Status = policyID, 0, ActivationPolicyDraft
	rollback.EffectiveFrom, rollback.EffectiveUntil = input.EffectiveFrom.UTC(), nil
	rollback.RollbackOfPolicyID, rollback.ProposedBy, rollback.ApprovedBy = source.ID, actor.PrincipalID, ""
	rollback.ProposalRationale, rollback.ApprovalRationale = rationale, ""
	rollback.CreatedAt, rollback.UpdatedAt, rollback.Version = now, now, 1
	rollback.TenantID, rollback.LegalEntityID = scope.TenantID, scope.LegalEntityID
	return s.repo.ProposeActivationPolicy(ctx, rollback)
}

func (s *ActivationService) ApprovePolicy(ctx context.Context, policyID string, expectedVersion int64, simulationID, rationale string) (ActivationPolicy, error) {
	actor, scope, policy, err := s.authorizePolicy(ctx, policyID, ActivationPolicyApproveCommand, authority.ResponsibilityAuthorizer)
	if err != nil {
		return ActivationPolicy{}, err
	}
	if policy.Status != ActivationPolicyPendingApproval || expectedVersion != policy.Version || strings.TrimSpace(rationale) == "" {
		return ActivationPolicy{}, ErrVersionConflict
	}
	if policy.ProposedBy == actor.PrincipalID {
		return ActivationPolicy{}, ErrActivationMakerChecker
	}
	simulation, err := s.repo.GetActivationSimulation(ctx, scope, strings.TrimSpace(simulationID))
	if err != nil || simulation.PolicyID != policy.ID || simulation.PolicyVersion != policy.Version || !simulation.PopulationIsComplete || !simulation.ExpiresAt.After(s.currentTime()) {
		return ActivationPolicy{}, ErrActivationSimulationRequired
	}
	return s.repo.TransitionActivationPolicy(ctx, scope, policy.ID, expectedVersion, ActivationPolicyActive, actor.PrincipalID, strings.TrimSpace(rationale), s.currentTime())
}

func (s *ActivationService) CurrentPolicy(ctx context.Context, legalEntityID string, at time.Time) (ActivationPolicy, error) {
	actor, ok := identity.FromContext(ctx)
	if !ok || actor.TenantID == "" || actor.PrincipalID == "" || (actor.LegalEntityID != "*" && actor.LegalEntityID != strings.TrimSpace(legalEntityID)) {
		return ActivationPolicy{}, commandauth.ErrIdentityRequired
	}
	if at.IsZero() {
		at = s.currentTime()
	}
	return s.repo.CurrentActivationPolicy(ctx, Scope{TenantID: actor.TenantID, LegalEntityID: strings.TrimSpace(legalEntityID)}, at.UTC())
}

func (s *ActivationService) SimulatePolicy(ctx context.Context, policyID string) (ActivationSimulation, error) {
	actor, scope, policy, err := s.authorizePolicy(ctx, policyID, ActivationPolicySimulateCommand, authority.ResponsibilityOwner)
	if err != nil {
		return ActivationSimulation{}, err
	}
	candidates, err := s.repo.ListActivationCandidates(ctx, scope, 500)
	if err != nil {
		return ActivationSimulation{}, err
	}
	simulationID, err := s.newID()
	if err != nil {
		return ActivationSimulation{}, err
	}
	evaluatedAt := s.currentTime()
	result := ActivationSimulation{ID: simulationID, PolicyID: policy.ID, PolicyVersion: policy.Version, CandidateCount: len(candidates), MissingGateCounts: map[string]int{}, EvaluatedAt: evaluatedAt, PopulationIsComplete: len(candidates) < 500, EvaluatedBy: actor.PrincipalID, ExpiresAt: evaluatedAt.Add(24 * time.Hour)}
	for _, candidate := range candidates {
		_, facts, readErr := s.repo.ReadActivationFacts(ctx, scope, candidate.ID, policy)
		if readErr != nil {
			return ActivationSimulation{}, readErr
		}
		facts.DecisionAuthoritiesCurrent = s.decisionAuthoritiesCurrent(ctx, scope, facts)
		gates := activationGates(candidate, facts, policy, policy.EffectiveFrom)
		eligible := true
		for _, gate := range gates {
			if !gate.Satisfied {
				eligible = false
				result.MissingGateCounts[gate.Code]++
			}
		}
		if eligible {
			result.EligibleCount++
		}
	}
	return s.repo.StoreActivationSimulation(ctx, scope, result)
}

func (s *ActivationService) ActivateRelationship(ctx context.Context, relationshipID string, input ActivateRelationshipInput) (ActivationResult, error) {
	actor, err := s.authorize(ctx, "", "VENDOR_RELATIONSHIP", strings.TrimSpace(relationshipID), RelationshipActivateCommand, authority.ResponsibilityAuthorizer)
	if err != nil {
		return ActivationResult{}, err
	}
	if input.ExpectedVersion < 1 || strings.TrimSpace(input.Rationale) == "" || len(strings.TrimSpace(input.Rationale)) > 2000 {
		return ActivationResult{}, ErrInvalid
	}
	at := input.IntendedEffectiveAt.UTC()
	if at.IsZero() {
		at = s.currentTime()
	}
	scope := Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}
	policy, err := s.repo.CurrentActivationPolicy(ctx, scope, at)
	if err != nil {
		return ActivationResult{}, err
	}
	relationship, facts, err := s.repo.ReadActivationFacts(ctx, scope, strings.TrimSpace(relationshipID), policy)
	if err != nil {
		return ActivationResult{}, err
	}
	if relationship.Version != input.ExpectedVersion {
		return ActivationResult{}, ErrVersionConflict
	}
	facts.DecisionAuthoritiesCurrent = s.decisionAuthoritiesCurrent(ctx, scope, facts)
	gates := activationGates(relationship, facts, policy, at)
	result := ActivationResult{Policy: policy, Gates: gates, Relationship: relationship, Eligible: gatesSatisfied(gates)}
	if !result.Eligible {
		return result, ErrActivationIneligible
	}
	receiptID, err := s.newID()
	if err != nil {
		return ActivationResult{}, err
	}
	activated, receipt, err := s.repo.CommitRelationshipActivation(ctx, ActivationCommit{
		Scope: scope, RelationshipID: relationship.ID, ExpectedVersion: relationship.Version, Policy: policy, Facts: facts,
		ActorID: actor.PrincipalID, Rationale: strings.TrimSpace(input.Rationale), EffectiveAt: at, ReceiptID: receiptID,
	})
	if err != nil {
		return ActivationResult{}, err
	}
	result.Relationship, result.Receipt = activated, receipt
	return result, nil
}

func (s *ActivationService) RelationshipEligibility(ctx context.Context, relationshipID string, at time.Time) (ActivationResult, error) {
	actor, ok := identity.FromContext(ctx)
	if !ok || actor.TenantID == "" || actor.PrincipalID == "" || actor.LegalEntityID == "*" {
		return ActivationResult{}, commandauth.ErrIdentityRequired
	}
	if at.IsZero() {
		at = s.currentTime()
	}
	scope := Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}
	policy, err := s.repo.CurrentActivationPolicy(ctx, scope, at.UTC())
	if err != nil {
		return ActivationResult{}, err
	}
	relationship, facts, err := s.repo.ReadActivationFacts(ctx, scope, strings.TrimSpace(relationshipID), policy)
	if err != nil {
		return ActivationResult{}, err
	}
	facts.DecisionAuthoritiesCurrent = s.decisionAuthoritiesCurrent(ctx, scope, facts)
	gates := activationGates(relationship, facts, policy, at.UTC())
	return ActivationResult{Eligible: gatesSatisfied(gates), Policy: policy, Gates: gates, Relationship: relationship}, nil
}

func activationGates(relationship Relationship, facts ActivationFacts, policy ActivationPolicy, at time.Time) []ActivationGate {
	allowedStatus := relationship.Status == RelationshipProposed || relationship.Status == RelationshipUnderReview
	assessmentCurrent := facts.AssessmentStatus == AssessmentCompleted && !facts.AssessmentCompletedAt.IsZero() && !facts.AssessmentCompletedAt.After(at) && facts.AssessmentCompletedAt.AddDate(0, 0, policy.MaximumAssessmentAgeDays).After(at)
	allowedConclusion := containsConclusion(policy.AllowedConclusions, facts.AssessmentConclusion)
	decisions := allCodesSatisfied(policy.RequiredDecisionTypes, facts.SatisfiedDecisionTypes)
	decisionAuthority := len(policy.RequiredDecisionTypes) == 0 || facts.DecisionAuthoritiesCurrent
	address := !policy.AddressVerificationRequired || (facts.AddressMatterID != "" && facts.AddressMatterClosed && facts.VerificationResultID != "" && facts.VerificationPassed)
	conditions := !policy.ConditionalConclusionNeedsTerms || facts.AssessmentConclusion != AssessmentSatisfactoryWithConditions || facts.ConditionsRecorded
	return []ActivationGate{
		{Code: "RELATIONSHIP_STATE", Satisfied: allowedStatus, Explanation: gateExplanation(allowedStatus, "The relationship is ready for an activation decision.", "The relationship is not in a state that can be activated.")},
		{Code: "CURRENT_ASSESSMENT", Satisfied: assessmentCurrent, Explanation: gateExplanation(assessmentCurrent, "The completed onboarding assessment is current.", "A current completed onboarding assessment is required.")},
		{Code: "ASSESSMENT_CONCLUSION", Satisfied: allowedConclusion, Explanation: gateExplanation(allowedConclusion, "The assessment conclusion is permitted by the current policy.", "The assessment conclusion is not permitted by the current policy.")},
		{Code: "REQUIRED_DECISIONS", Satisfied: decisions, Explanation: gateExplanation(decisions, "Every required decision is current and approved.", "One or more required approval decisions are missing or no longer current.")},
		{Code: "DECISION_AUTHORITY", Satisfied: decisionAuthority, Explanation: gateExplanation(decisionAuthority, "The recorded decision makers remain in the current authority route.", "A required decision was made under authority that is no longer current and must be reviewed again.")},
		{Code: "ADDRESS_OUTCOME", Satisfied: address, Explanation: gateExplanation(address, "The address issue is closed with a passing independent outcome check.", "Address verification must be independently confirmed and closed.")},
		{Code: "CONDITIONS", Satisfied: conditions, Explanation: gateExplanation(conditions, "Any conditional conclusion has recorded terms.", "The assessment conditions must be recorded before activation.")},
		{Code: "BLOCKING_ISSUES", Satisfied: !facts.HasBlockingMatter, Explanation: gateExplanation(!facts.HasBlockingMatter, "No policy-defined blocking issue remains open.", "A policy-defined blocking issue remains open.")},
		{Code: "CONTRADICTIONS", Satisfied: !facts.HasUnresolvedContradiction, Explanation: gateExplanation(!facts.HasUnresolvedContradiction, "No unresolved evidence contradiction blocks activation.", "An unresolved evidence contradiction blocks activation.")},
	}
}

func (s *ActivationService) decisionAuthoritiesCurrent(ctx context.Context, scope Scope, facts ActivationFacts) bool {
	for _, dependency := range facts.DecisionDependencies {
		if strings.TrimSpace(dependency.AuthorityPrincipalID) == "" || strings.TrimSpace(dependency.MatterID) == "" {
			return false
		}
		base, ok := identity.FromContext(ctx)
		if !ok {
			return false
		}
		base.PrincipalID = dependency.AuthorityPrincipalID
		if _, err := s.guard.Authorize(identity.WithActor(ctx, base), commandauth.Request{
			TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, ObjectType: "MATTER", ObjectID: dependency.MatterID,
			Responsibility: authority.ResponsibilityAuthorizer, DecisionType: "matter.decision.record", Materiality: 4,
		}); err != nil {
			return false
		}
	}
	return true
}

func (s *ActivationService) authorizePolicy(ctx context.Context, policyID, decisionType string, responsibility authority.Responsibility) (identity.Actor, Scope, ActivationPolicy, error) {
	actor, ok := identity.FromContext(ctx)
	if !ok {
		return identity.Actor{}, Scope{}, ActivationPolicy{}, commandauth.ErrIdentityRequired
	}
	scope := Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}
	policy, err := s.repo.GetActivationPolicy(ctx, scope, strings.TrimSpace(policyID))
	if err != nil {
		return identity.Actor{}, Scope{}, ActivationPolicy{}, err
	}
	verified, err := s.authorize(ctx, policy.LegalEntityID, "THIRD_PARTY_ACTIVATION_POLICY", policy.ID, decisionType, responsibility)
	return verified, scope, policy, err
}

func (s *ActivationService) authorize(ctx context.Context, legalEntityID, objectType, objectID, decisionType string, responsibility authority.Responsibility) (identity.Actor, error) {
	actor, ok := identity.FromContext(ctx)
	if !ok || s.guard == nil {
		return identity.Actor{}, commandauth.ErrIdentityRequired
	}
	if strings.TrimSpace(legalEntityID) == "" {
		legalEntityID = actor.LegalEntityID
	}
	s.lastAuthorizationResponsibility = responsibility
	decision, err := s.guard.Authorize(ctx, commandauth.Request{TenantID: actor.TenantID, LegalEntityID: legalEntityID, ObjectType: objectType, ObjectID: objectID, Responsibility: responsibility, DecisionType: decisionType, Materiality: 4})
	if err != nil || !decision.Allowed {
		if err == nil {
			err = commandauth.ErrNotAuthorized
		}
		return identity.Actor{}, err
	}
	if decision.Actor.TenantID == "" || decision.Actor.PrincipalID == "" {
		return identity.Actor{}, commandauth.ErrIdentityRequired
	}
	return decision.Actor, nil
}

func validateActivationPolicyInput(input ProposeActivationPolicyInput) error {
	if strings.TrimSpace(input.LegalEntityID) == "" || len(input.AllowedConclusions) == 0 || input.MaximumAssessmentAgeDays < 1 || input.MaximumAssessmentAgeDays > 730 || input.EffectiveFrom.IsZero() || len(strings.TrimSpace(input.Rationale)) < 20 || len(strings.TrimSpace(input.Rationale)) > 2000 {
		return ErrInvalid
	}
	for _, conclusion := range input.AllowedConclusions {
		if conclusion != AssessmentSatisfactory && conclusion != AssessmentSatisfactoryWithConditions {
			return ErrInvalid
		}
	}
	return nil
}

func normalizeConclusions(values []AssessmentConclusion) []AssessmentConclusion {
	seen := map[AssessmentConclusion]bool{}
	result := make([]AssessmentConclusion, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func normalizeCodes(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func containsConclusion(values []AssessmentConclusion, target AssessmentConclusion) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func allCodesSatisfied(required, satisfied []string) bool {
	set := map[string]bool{}
	for _, value := range satisfied {
		set[strings.ToUpper(strings.TrimSpace(value))] = true
	}
	for _, value := range required {
		if !set[strings.ToUpper(strings.TrimSpace(value))] {
			return false
		}
	}
	return true
}

func gateExplanation(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}
func gatesSatisfied(values []ActivationGate) bool {
	for _, value := range values {
		if !value.Satisfied {
			return false
		}
	}
	return true
}
func (s *ActivationService) currentTime() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func validatePolicyTransition(from ActivationPolicyStatus, to ActivationPolicyStatus) error {
	if (from == ActivationPolicyDraft && to == ActivationPolicyPendingApproval) || (from == ActivationPolicyPendingApproval && to == ActivationPolicyActive) {
		return nil
	}
	return fmt.Errorf("invalid activation policy transition %s to %s", from, to)
}
