package governance

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) ListPolicies(ctx context.Context, tenantID string) ([]RoutingPolicy, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return s.repo.ListPolicies(ctx, tenantID)
}

func (s *Service) CreatePolicy(ctx context.Context, input CreatePolicyInput) (RoutingPolicy, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.Code) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.MakerID) == "" {
		return RoutingPolicy{}, fmt.Errorf("tenant_id, code, name and maker_id are required")
	}
	definition, checksum, err := normalizeDefinition(input.Definition)
	if err != nil {
		return RoutingPolicy{}, err
	}
	policyID, err := id.NewUUIDv7()
	if err != nil {
		return RoutingPolicy{}, err
	}
	now := s.now().UTC()
	return s.repo.CreatePolicy(ctx, RoutingPolicy{ID: policyID, TenantID: input.TenantID, Code: strings.ToUpper(strings.TrimSpace(input.Code)), Name: strings.TrimSpace(input.Name), Status: PolicyDraft, CurrentVersion: 1, Definition: definition, Checksum: checksum, MakerID: input.MakerID, EffectiveFrom: input.EffectiveFrom, CreatedAt: now, UpdatedAt: now, Version: 1})
}

func (s *Service) SubmitPolicy(ctx context.Context, input TransitionInput) (RoutingPolicy, error) {
	if err := validateTransition(input); err != nil {
		return RoutingPolicy{}, err
	}
	current, err := s.repo.GetPolicy(ctx, input.TenantID, input.ID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if current.Status != PolicyDraft {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if input.ActorID != current.MakerID {
		return RoutingPolicy{}, ErrMakerChecker
	}
	if err := validatePolicyDefinition(current.Definition); err != nil {
		return RoutingPolicy{}, err
	}
	return s.repo.TransitionPolicy(ctx, input.TenantID, input.ID, input.ExpectedVersion, PolicyDraft, PolicyPendingApproval, input.ActorID, input.Rationale, s.now().UTC())
}

func (s *Service) ApprovePolicy(ctx context.Context, input TransitionInput) (RoutingPolicy, error) {
	if err := validateTransition(input); err != nil {
		return RoutingPolicy{}, err
	}
	current, err := s.repo.GetPolicy(ctx, input.TenantID, input.ID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if current.Status != PolicyPendingApproval {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if input.ActorID == "" || input.ActorID == current.MakerID {
		return RoutingPolicy{}, ErrMakerChecker
	}
	findings, err := s.repo.PolicyConflicts(ctx, current)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if len(findings) > 0 {
		return RoutingPolicy{}, fmt.Errorf("%w: %s", ErrConflict, findings[0].Summary)
	}
	return s.repo.TransitionPolicy(ctx, input.TenantID, input.ID, input.ExpectedVersion, PolicyPendingApproval, PolicyActive, input.ActorID, input.Rationale, s.now().UTC())
}

func (s *Service) RejectPolicy(ctx context.Context, input TransitionInput) (RoutingPolicy, error) {
	if err := validateTransition(input); err != nil {
		return RoutingPolicy{}, err
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return RoutingPolicy{}, fmt.Errorf("rationale is required to reject a policy")
	}
	current, err := s.repo.GetPolicy(ctx, input.TenantID, input.ID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if current.Status != PolicyPendingApproval {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if input.ActorID == "" || input.ActorID == current.MakerID {
		return RoutingPolicy{}, ErrMakerChecker
	}
	return s.repo.TransitionPolicy(ctx, input.TenantID, input.ID, input.ExpectedVersion, PolicyPendingApproval, PolicyDraft, input.ActorID, input.Rationale, s.now().UTC())
}

func (s *Service) RetirePolicy(ctx context.Context, input TransitionInput) (RoutingPolicy, error) {
	if err := validateTransition(input); err != nil {
		return RoutingPolicy{}, err
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return RoutingPolicy{}, fmt.Errorf("rationale is required to retire a policy")
	}
	current, err := s.repo.GetPolicy(ctx, input.TenantID, input.ID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if current.Status != PolicyActive || strings.TrimSpace(input.ActorID) == "" {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	return s.repo.TransitionPolicy(ctx, input.TenantID, input.ID, input.ExpectedVersion, PolicyActive, PolicyRetired, input.ActorID, input.Rationale, s.now().UTC())
}

func (s *Service) ListDelegations(ctx context.Context, tenantID string) ([]Delegation, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return s.repo.ListDelegations(ctx, tenantID)
}

func (s *Service) CreateDelegation(ctx context.Context, input CreateDelegationInput) (Delegation, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.FromPrincipalID) == "" || strings.TrimSpace(input.ToPrincipalID) == "" || strings.TrimSpace(input.Responsibility) == "" || strings.TrimSpace(input.MakerID) == "" {
		return Delegation{}, fmt.Errorf("tenant, principals, responsibility and maker are required")
	}
	if input.FromPrincipalID == input.ToPrincipalID {
		return Delegation{}, ErrConflict
	}
	if !input.StartsAt.Before(input.EndsAt) {
		return Delegation{}, fmt.Errorf("starts_at must be before ends_at")
	}
	scope := input.Scope
	if len(scope) == 0 {
		scope = json.RawMessage(`{}`)
	}
	if !json.Valid(scope) {
		return Delegation{}, fmt.Errorf("scope must be valid JSON")
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return Delegation{}, err
	}
	now := s.now().UTC()
	return s.repo.CreateDelegation(ctx, Delegation{ID: valueID, TenantID: input.TenantID, FromPrincipalID: input.FromPrincipalID, ToPrincipalID: input.ToPrincipalID, Responsibility: input.Responsibility, Scope: scope, StartsAt: input.StartsAt.UTC(), EndsAt: input.EndsAt.UTC(), Status: DelegationDraft, Reason: strings.TrimSpace(input.Reason), MakerID: input.MakerID, CreatedAt: now, UpdatedAt: now, Version: 1})
}

func (s *Service) SubmitDelegation(ctx context.Context, input TransitionInput) (Delegation, error) {
	if err := validateTransition(input); err != nil {
		return Delegation{}, err
	}
	current, err := s.repo.GetDelegation(ctx, input.TenantID, input.ID)
	if err != nil {
		return Delegation{}, err
	}
	if current.Status != DelegationDraft || input.ActorID != current.MakerID {
		return Delegation{}, ErrInvalidTransition
	}
	return s.repo.TransitionDelegation(ctx, input.TenantID, input.ID, input.ExpectedVersion, DelegationDraft, DelegationPendingApproval, input.ActorID, input.Rationale, s.now().UTC())
}

func (s *Service) ApproveDelegation(ctx context.Context, input TransitionInput) (Delegation, error) {
	if err := validateTransition(input); err != nil {
		return Delegation{}, err
	}
	current, err := s.repo.GetDelegation(ctx, input.TenantID, input.ID)
	if err != nil {
		return Delegation{}, err
	}
	if current.Status != DelegationPendingApproval {
		return Delegation{}, ErrInvalidTransition
	}
	if input.ActorID == "" || input.ActorID == current.MakerID || input.ActorID == current.FromPrincipalID || input.ActorID == current.ToPrincipalID {
		return Delegation{}, ErrMakerChecker
	}
	cycle, err := s.repo.HasDelegationCycle(ctx, current)
	if err != nil {
		return Delegation{}, err
	}
	if cycle {
		return Delegation{}, ErrConflict
	}
	findings, err := s.repo.DelegationConflicts(ctx, current.TenantID, current.ToPrincipalID, current.Responsibility)
	if err != nil {
		return Delegation{}, err
	}
	if len(findings) > 0 {
		return Delegation{}, fmt.Errorf("%w: %s", ErrConflict, findings[0].Summary)
	}
	now := s.now().UTC()
	target := DelegationApproved
	if !now.Before(current.StartsAt) && now.Before(current.EndsAt) {
		target = DelegationActive
	}
	return s.repo.TransitionDelegation(ctx, input.TenantID, input.ID, input.ExpectedVersion, DelegationPendingApproval, target, input.ActorID, input.Rationale, now)
}

func (s *Service) RevokeDelegation(ctx context.Context, input TransitionInput) (Delegation, error) {
	if err := validateTransition(input); err != nil {
		return Delegation{}, err
	}
	if strings.TrimSpace(input.Rationale) == "" {
		return Delegation{}, fmt.Errorf("rationale is required to revoke a delegation")
	}
	current, err := s.repo.GetDelegation(ctx, input.TenantID, input.ID)
	if err != nil {
		return Delegation{}, err
	}
	if current.Status != DelegationApproved && current.Status != DelegationActive {
		return Delegation{}, ErrInvalidTransition
	}
	if input.ActorID != current.FromPrincipalID && input.ActorID != current.ApproverID {
		return Delegation{}, ErrMakerChecker
	}
	return s.repo.TransitionDelegation(ctx, input.TenantID, input.ID, input.ExpectedVersion, current.Status, DelegationRevoked, input.ActorID, input.Rationale, s.now().UTC())
}

func validateTransition(input TransitionInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.ActorID) == "" {
		return fmt.Errorf("tenant_id, id and actor_id are required")
	}
	if input.ExpectedVersion < 1 {
		return fmt.Errorf("expected_version must be positive")
	}
	return nil
}

func normalizeDefinition(value json.RawMessage) (json.RawMessage, string, error) {
	if len(value) == 0 || !json.Valid(value) {
		return nil, "", fmt.Errorf("definition must be valid JSON")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, value); err != nil {
		return nil, "", err
	}
	if err := validatePolicyDefinition(compact.Bytes()); err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(compact.Bytes())
	return json.RawMessage(compact.Bytes()), hex.EncodeToString(sum[:]), nil
}

func validatePolicyDefinition(value json.RawMessage) error {
	var definition struct {
		Rules []struct {
			ID               string `json:"id"`
			LegalEntityID    string `json:"legal_entity_id"`
			ObjectType       string `json:"object_type"`
			ObjectID         string `json:"object_id"`
			Responsibility   string `json:"responsibility"`
			DecisionType     string `json:"decision_type"`
			MinMateriality   int    `json:"min_materiality"`
			Priority         int    `json:"priority"`
			LifecycleType    string `json:"lifecycle_type"`
			LifecycleSubtype string `json:"lifecycle_subtype"`
			LifecycleState   string `json:"lifecycle_state"`
			Selector         struct {
				Kind string `json:"kind"`
				Ref  string `json:"ref"`
			} `json:"selector"`
		} `json:"rules"`
	}
	if err := json.Unmarshal(value, &definition); err != nil {
		return fmt.Errorf("decode policy definition: %w", err)
	}
	if len(definition.Rules) == 0 {
		return fmt.Errorf("policy must contain at least one rule")
	}
	allowedResponsibilities := map[string]bool{
		"PERFORMER":                true,
		"ACCOUNTABLE_OWNER":        true,
		"PROPOSER":                 true,
		"REVIEWER":                 true,
		"INDEPENDENT_CHALLENGER":   true,
		"AUTHORIZER":               true,
		"SIGNATORY":                true,
		"TRANSMITTER":              true,
		"ACKNOWLEDGEMENT_RECORDER": true,
		"ESCALATION_OWNER":         true,
	}
	allowedSelectors := map[string]bool{"PRINCIPAL": true, "POSITION": true, "ROLE": true}
	seenIDs := map[string]struct{}{}
	seenRoutes := map[string]string{}
	for _, rule := range definition.Rules {
		kind := strings.ToUpper(strings.TrimSpace(rule.Selector.Kind))
		selectorRef := strings.TrimSpace(rule.Selector.Ref)
		responsibility := strings.ToUpper(strings.TrimSpace(rule.Responsibility))
		if strings.TrimSpace(rule.ID) == "" || !allowedResponsibilities[responsibility] {
			return fmt.Errorf("each policy rule requires a unique id and supported responsibility")
		}
		if rule.MinMateriality < 0 || rule.MinMateriality > 5 {
			return fmt.Errorf("rule %s materiality must be between 0 and 5", rule.ID)
		}
		if err := validateLifecycleRuleDeclaration(rule.ID, rule.LifecycleType, rule.LifecycleState); err != nil {
			return err
		}
		sequenceRule := strings.TrimSpace(rule.LifecycleType) != "" || strings.TrimSpace(rule.LifecycleState) != ""
		if sequenceRule {
			if kind != "" || selectorRef != "" {
				return fmt.Errorf("lifecycle sequence rule %s must not define an actor selector", rule.ID)
			}
		} else if !allowedSelectors[kind] || selectorRef == "" {
			return fmt.Errorf("authority routing rule %s requires a supported selector", rule.ID)
		}
		if _, ok := seenIDs[rule.ID]; ok {
			return fmt.Errorf("duplicate policy rule id %s", rule.ID)
		}
		seenIDs[rule.ID] = struct{}{}
		if sequenceRule {
			continue
		}
		routeKey := strings.Join([]string{rule.LegalEntityID, rule.ObjectType, rule.ObjectID, responsibility, rule.DecisionType, fmt.Sprint(rule.MinMateriality), fmt.Sprint(rule.Priority)}, "|")
		selectorKey := kind + ":" + selectorRef
		if prior, ok := seenRoutes[routeKey]; ok && prior != selectorKey {
			return fmt.Errorf("ambiguous policy rules share route priority with different selectors")
		}
		seenRoutes[routeKey] = selectorKey
	}
	return nil
}
