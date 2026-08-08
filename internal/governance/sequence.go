package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoLifecycleSequence        = errors.New("no governed lifecycle sequence")
	ErrAmbiguousLifecycleSequence = errors.New("ambiguous governed lifecycle sequence")
)

type LifecycleSequenceInput struct {
	TenantID         string
	LegalEntityID    string
	MatterID         string
	MatterType       string
	CommandName      string
	LifecycleType    string
	LifecycleSubtype string
	LifecycleState   string
	Materiality      int
	At               time.Time
}

type LifecycleSequenceResolution struct {
	Responsibility string
	RuleID         string
	PolicyVersion  string
}

type LifecycleSequenceResolver interface {
	ResolveLifecycleSequence(context.Context, LifecycleSequenceInput) (LifecycleSequenceResolution, error)
}

type lifecycleSequenceRule struct {
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
}

type lifecycleSequenceCandidate struct {
	Responsibility string
	RuleID         string
	PolicyVersion  string
	Priority       int
	Specificity    int
}

// ResolveLifecycleSequence reads the already maker-checker-governed routing
// policies. Only rules that explicitly declare lifecycle_type and
// lifecycle_state participate in sequence selection; existing routing rules
// remain authority-only. Sequence selection chooses the next responsibility,
// never a substantive lifecycle outcome.
func (s *Service) ResolveLifecycleSequence(ctx context.Context, input LifecycleSequenceInput) (LifecycleSequenceResolution, error) {
	if s == nil || s.repo == nil {
		return LifecycleSequenceResolution{}, fmt.Errorf("governance service is unavailable")
	}
	if err := validateLifecycleSequenceInput(input); err != nil {
		return LifecycleSequenceResolution{}, err
	}
	policies, err := s.repo.ListPolicies(ctx, input.TenantID)
	if err != nil {
		return LifecycleSequenceResolution{}, err
	}
	return resolveLifecycleSequence(policies, input)
}

func resolveLifecycleSequence(policies []RoutingPolicy, input LifecycleSequenceInput) (LifecycleSequenceResolution, error) {
	at := input.At.UTC()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	candidates := make([]lifecycleSequenceCandidate, 0, 4)
	for _, policy := range policies {
		if policy.Status != PolicyActive || !policyEffectiveAt(policy, at) {
			continue
		}
		var definition struct {
			Rules []lifecycleSequenceRule `json:"rules"`
		}
		if err := json.Unmarshal(policy.Definition, &definition); err != nil {
			return LifecycleSequenceResolution{}, fmt.Errorf("decode routing policy %s: %w", policy.Code, err)
		}
		for _, rule := range definition.Rules {
			candidate, ok, err := matchLifecycleSequenceRule(policy, rule, input)
			if err != nil {
				return LifecycleSequenceResolution{}, err
			}
			if ok {
				candidates = append(candidates, candidate)
			}
		}
	}
	if len(candidates) == 0 {
		return LifecycleSequenceResolution{}, ErrNoLifecycleSequence
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		if candidates[i].Specificity != candidates[j].Specificity {
			return candidates[i].Specificity > candidates[j].Specificity
		}
		return candidates[i].RuleID < candidates[j].RuleID
	})
	top := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Priority != top.Priority || candidate.Specificity != top.Specificity {
			break
		}
		if candidate.Responsibility != top.Responsibility {
			return LifecycleSequenceResolution{}, fmt.Errorf("%w: %s and %s select different next responsibilities", ErrAmbiguousLifecycleSequence, top.RuleID, candidate.RuleID)
		}
	}
	return LifecycleSequenceResolution{Responsibility: top.Responsibility, RuleID: top.RuleID, PolicyVersion: top.PolicyVersion}, nil
}

func matchLifecycleSequenceRule(policy RoutingPolicy, rule lifecycleSequenceRule, input LifecycleSequenceInput) (lifecycleSequenceCandidate, bool, error) {
	lifecycleType := strings.ToUpper(strings.TrimSpace(rule.LifecycleType))
	lifecycleState := strings.ToUpper(strings.TrimSpace(rule.LifecycleState))
	if lifecycleType == "" && lifecycleState == "" {
		return lifecycleSequenceCandidate{}, false, nil
	}
	if lifecycleType == "" || lifecycleState == "" {
		return lifecycleSequenceCandidate{}, false, fmt.Errorf("routing policy %s rule %s must declare both lifecycle_type and lifecycle_state", policy.Code, rule.ID)
	}
	if lifecycleType != "DECISION" && lifecycleType != "RESPONSE" {
		return lifecycleSequenceCandidate{}, false, fmt.Errorf("routing policy %s rule %s has unsupported lifecycle_type %s", policy.Code, rule.ID, lifecycleType)
	}
	if lifecycleType != strings.ToUpper(input.LifecycleType) || lifecycleState != strings.ToUpper(input.LifecycleState) {
		return lifecycleSequenceCandidate{}, false, nil
	}
	responsibility := strings.ToUpper(strings.TrimSpace(rule.Responsibility))
	if responsibility == "" {
		return lifecycleSequenceCandidate{}, false, fmt.Errorf("routing policy %s rule %s has no lifecycle responsibility", policy.Code, rule.ID)
	}
	if rule.MinMateriality > input.Materiality {
		return lifecycleSequenceCandidate{}, false, nil
	}
	if !wildcardMatch(rule.LegalEntityID, input.LegalEntityID) || !wildcardMatch(rule.ObjectType, "MATTER") || !wildcardMatch(rule.ObjectID, input.MatterID) {
		return lifecycleSequenceCandidate{}, false, nil
	}
	if strings.TrimSpace(rule.DecisionType) != "" && !strings.EqualFold(strings.TrimSpace(rule.DecisionType), input.CommandName) {
		return lifecycleSequenceCandidate{}, false, nil
	}
	if strings.TrimSpace(rule.LifecycleSubtype) != "" && strings.TrimSpace(rule.LifecycleSubtype) != "*" && !strings.EqualFold(strings.TrimSpace(rule.LifecycleSubtype), input.LifecycleSubtype) {
		return lifecycleSequenceCandidate{}, false, nil
	}
	specificity := 0
	if !isWildcard(rule.LegalEntityID) {
		specificity += 8
	}
	if !isWildcard(rule.ObjectType) {
		specificity += 4
	}
	if !isWildcard(rule.ObjectID) {
		specificity += 2
	}
	if strings.TrimSpace(rule.DecisionType) != "" {
		specificity++
	}
	if strings.TrimSpace(rule.LifecycleSubtype) != "" && strings.TrimSpace(rule.LifecycleSubtype) != "*" {
		specificity++
	}
	return lifecycleSequenceCandidate{
		Responsibility: responsibility,
		RuleID:         policy.Code + "/" + strings.TrimSpace(rule.ID),
		PolicyVersion:  policy.Code + ":v" + strconv.Itoa(policy.CurrentVersion),
		Priority:       rule.Priority,
		Specificity:    specificity,
	}, true, nil
}

func validateLifecycleSequenceInput(input LifecycleSequenceInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.MatterID) == "" || strings.TrimSpace(input.LifecycleType) == "" || strings.TrimSpace(input.LifecycleState) == "" {
		return fmt.Errorf("tenant, matter, lifecycle type and lifecycle state are required")
	}
	if input.Materiality < 0 || input.Materiality > 5 {
		return fmt.Errorf("materiality must be between 0 and 5")
	}
	return nil
}

func policyEffectiveAt(policy RoutingPolicy, at time.Time) bool {
	if policy.EffectiveFrom != nil && at.Before(policy.EffectiveFrom.UTC()) {
		return false
	}
	return policy.EffectiveUntil == nil || at.Before(policy.EffectiveUntil.UTC())
}

func wildcardMatch(rule, value string) bool {
	rule = strings.TrimSpace(rule)
	return rule == "" || rule == "*" || strings.EqualFold(rule, strings.TrimSpace(value))
}

func isWildcard(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == "*"
}

var _ LifecycleSequenceResolver = (*Service)(nil)
