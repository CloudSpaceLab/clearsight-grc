package authority

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrNoRoute      = errors.New("no eligible authority route")
	ErrInvalidInput = errors.New("invalid authority resolution input")
)

type Resolver struct {
	version string
	rules   []Rule
	policy  PolicySummary
}

func NewResolver(version string, rules []Rule) *Resolver {
	copied := append([]Rule(nil), rules...)
	sort.SliceStable(copied, func(i, j int) bool { return copied[i].Priority > copied[j].Priority })
	now := time.Now().UTC()
	return &Resolver{
		version: version,
		rules:   copied,
		policy: PolicySummary{
			ID:            "policy-demo",
			Code:          "bank-default",
			Name:          "Bank default routing",
			Status:        "ACTIVE",
			Version:       1,
			EffectiveFrom: &now,
		},
	}
}

func (r *Resolver) Resolve(ctx context.Context, input ResolveInput) (Resolution, error) {
	simulation, err := r.Simulate(ctx, input)
	if err != nil {
		return Resolution{}, err
	}
	if simulation.Selected == nil {
		return Resolution{}, ErrNoRoute
	}
	return *simulation.Selected, nil
}

func (r *Resolver) Simulate(_ context.Context, input ResolveInput) (Simulation, error) {
	if err := validateInput(input); err != nil {
		return Simulation{}, err
	}
	at := input.At
	if at.IsZero() {
		at = time.Now().UTC()
	}
	candidates := make([]Candidate, 0, len(r.rules))
	var selected *Resolution
	for _, rule := range r.rules {
		eligible, reason := matchReason(rule, input, at)
		candidates = append(candidates, Candidate{Principal: rule.Principal, RuleID: rule.ID, Priority: rule.Priority, Eligible: eligible, Reason: reason})
		if eligible && selected == nil {
			value := Resolution{
				Principal:     rule.Principal,
				RuleID:        rule.ID,
				PolicyVersion: r.version,
				Explanation:   fmt.Sprintf("%s selected for %s on %s:%s within legal entity %s", rule.Principal.DisplayName, input.Responsibility, input.ObjectType, input.ObjectID, input.LegalEntityID),
			}
			selected = &value
		}
	}
	return Simulation{Selected: selected, Candidates: candidates, PolicyVersion: r.version}, nil
}

func (r *Resolver) Policies(_ context.Context, tenantID string) ([]PolicySummary, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return []PolicySummary{r.policy}, nil
}

func (r *Resolver) Integrity(_ context.Context, tenantID string) ([]IntegrityFinding, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	return integrityFindings(r.rules, tenantID), nil
}

func validateInput(input ResolveInput) error {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.LegalEntityID) == "" || strings.TrimSpace(input.ObjectType) == "" || strings.TrimSpace(input.ObjectID) == "" || strings.TrimSpace(string(input.Responsibility)) == "" {
		return fmt.Errorf("%w: tenant, legal entity, object and responsibility are required", ErrInvalidInput)
	}
	if input.Materiality < 0 || input.Materiality > 5 {
		return fmt.Errorf("%w: materiality must be between 0 and 5", ErrInvalidInput)
	}
	return nil
}

func matchReason(rule Rule, input ResolveInput, at time.Time) (bool, string) {
	if rule.TenantID != input.TenantID {
		return false, "tenant does not match"
	}
	if rule.Responsibility != input.Responsibility {
		return false, "responsibility does not match"
	}
	if rule.LegalEntityID != "*" && rule.LegalEntityID != input.LegalEntityID {
		return false, "legal entity does not match"
	}
	if rule.ObjectType != "*" && !strings.EqualFold(rule.ObjectType, input.ObjectType) {
		return false, "object type does not match"
	}
	if rule.ObjectID != "*" && rule.ObjectID != input.ObjectID {
		return false, "object does not match"
	}
	if rule.DecisionType != "" && !strings.EqualFold(rule.DecisionType, input.DecisionType) {
		return false, "decision type does not match"
	}
	if input.Materiality < rule.MinMateriality {
		return false, "materiality is below the rule threshold"
	}
	if !rule.ValidFrom.IsZero() && at.Before(rule.ValidFrom) {
		return false, "rule is not effective yet"
	}
	if !rule.ValidUntil.IsZero() && !at.Before(rule.ValidUntil) {
		return false, "rule has expired"
	}
	if strings.TrimSpace(rule.Principal.ID) == "" {
		return false, "selector does not resolve to an active principal"
	}
	return true, "eligible"
}

func integrityFindings(rules []Rule, tenantID string) []IntegrityFinding {
	findings := []IntegrityFinding{}
	authorizers := 0
	seen := map[string]Rule{}
	for _, rule := range rules {
		if rule.TenantID != tenantID {
			continue
		}
		if rule.Principal.ID == "" {
			findings = append(findings, IntegrityFinding{
				Type:           "UNRESOLVED_SELECTOR",
				Severity:       "CRITICAL",
				Summary:        "A routing selector does not resolve to an active principal.",
				RequiredAction: "Assign an occupant, bind the role, or replace the selector.",
				RuleIDs:        []string{rule.ID},
			})
			continue
		}
		if rule.Responsibility == ResponsibilityAuthorizer {
			authorizers++
		}
		key := strings.Join([]string{rule.LegalEntityID, rule.ObjectType, rule.ObjectID, string(rule.Responsibility), rule.DecisionType, fmt.Sprint(rule.MinMateriality), fmt.Sprint(rule.Priority)}, "|")
		if prior, ok := seen[key]; ok && prior.Principal.ID != rule.Principal.ID {
			findings = append(findings, IntegrityFinding{Type: "AMBIGUOUS_ROUTE", Severity: "HIGH", Summary: "Two active rules have the same scope and priority but resolve to different principals.", RequiredAction: "Change priority or narrow one rule before activation.", RuleIDs: []string{prior.ID, rule.ID}})
		} else {
			seen[key] = rule
		}
	}
	if authorizers == 0 {
		findings = append(findings, IntegrityFinding{Type: "MISSING_AUTHORIZER", Severity: "CRITICAL", Summary: "No active authorizer route exists for this tenant.", RequiredAction: "Create and approve at least one scoped authorizer route."})
	}
	return findings
}

func DemoPolicySet() (string, []Rule) {
	return "demo-2026-08-05", []Rule{
		{ID: "route-ndpa-dpo", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "PROGRAM", ObjectID: "ndpa", Responsibility: ResponsibilityAuthorizer, Principal: Principal{ID: "role-dpo", DisplayName: "Data Protection Officer", Kind: "ROLE", Role: "DPO"}, Priority: 100},
		{ID: "route-control-assurance", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityReviewer, Principal: Principal{ID: "team-control-assurance", DisplayName: "Control Assurance", Kind: "TEAM", Role: "Second-line reviewer"}, Priority: 80},
		{ID: "route-cro-material", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityAuthorizer, MinMateriality: 4, Principal: Principal{ID: "role-cro", DisplayName: "Chief Risk Officer", Kind: "ROLE", Role: "CRO"}, Priority: 70},
		{ID: "route-risk-owner", TenantID: "bank-demo", LegalEntityID: "bank-ng", ObjectType: "MATTER", ObjectID: "*", Responsibility: ResponsibilityOwner, Principal: Principal{ID: "queue-risk-owners", DisplayName: "Scoped Risk Owners", Kind: "QUEUE", Role: "Accountable owner queue"}, Priority: 50},
	}
}
