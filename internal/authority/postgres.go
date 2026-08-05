//go:build postgres

package authority

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresService struct{ pool *pgxpool.Pool }

func NewPostgresService(pool *pgxpool.Pool) Service { return &postgresService{pool: pool} }

type persistedPolicy struct {
	ID            string
	Code          string
	Name          string
	Status        string
	Version       int
	EffectiveFrom time.Time
	Definition    json.RawMessage
}
type policyDefinition struct {
	Rules []policyRule `json:"rules"`
}
type policyRule struct {
	ID             string         `json:"id"`
	LegalEntityID  string         `json:"legal_entity_id"`
	ObjectType     string         `json:"object_type"`
	ObjectID       string         `json:"object_id"`
	Responsibility Responsibility `json:"responsibility"`
	DecisionType   string         `json:"decision_type,omitempty"`
	MinMateriality int            `json:"min_materiality"`
	Priority       int            `json:"priority"`
	Selector       selector       `json:"selector"`
}
type selector struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
	Role string `json:"role,omitempty"`
}

func (s *postgresService) Policies(ctx context.Context, tenantID string) ([]PolicySummary, error) {
	rows, err := s.pool.Query(ctx, `SELECT rp.id::text,rp.code,rp.name,rp.status,rpv.version,rpv.effective_from FROM routing_policies rp JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id AND rpv.version=rp.current_version WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) ORDER BY rp.code`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list routing policies: %w", err)
	}
	defer rows.Close()
	result := []PolicySummary{}
	for rows.Next() {
		var value PolicySummary
		if err := rows.Scan(&value.ID, &value.Code, &value.Name, &value.Status, &value.Version, &value.EffectiveFrom); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func (s *postgresService) Resolve(ctx context.Context, input ResolveInput) (Resolution, error) {
	simulation, err := s.Simulate(ctx, input)
	if err != nil {
		return Resolution{}, err
	}
	if simulation.Selected == nil {
		return Resolution{}, ErrNoRoute
	}
	return *simulation.Selected, nil
}
func (s *postgresService) Simulate(ctx context.Context, input ResolveInput) (Simulation, error) {
	version, rules, err := s.loadRules(ctx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return Simulation{}, err
	}
	static := NewResolver(version, rules)
	return static.Simulate(ctx, input)
}
func (s *postgresService) Integrity(ctx context.Context, tenantID string) ([]IntegrityFinding, error) {
	_, rules, err := s.loadRules(ctx, tenantID, "*")
	if err != nil {
		return nil, err
	}
	findings := integrityFindings(rules, tenantID)
	for _, rule := range rules {
		if rule.Principal.ID == "" {
			findings = append(findings, IntegrityFinding{Type: "UNRESOLVED_SELECTOR", Severity: "CRITICAL", Summary: "A routing selector does not resolve to an active principal.", RequiredAction: "Assign an occupant, bind the role, or replace the selector.", RuleIDs: []string{rule.ID}})
		}
	}
	return findings, nil
}
func (s *postgresService) loadRules(ctx context.Context, tenantID, legalEntityID string) (string, []Rule, error) {
	rows, err := s.pool.Query(ctx, `SELECT rp.id::text,rp.code,rp.name,rp.status,rpv.version,rpv.effective_from,rpv.definition FROM routing_policies rp JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id AND rpv.version=rp.current_version WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rp.status='ACTIVE' AND (rpv.effective_from IS NULL OR rpv.effective_from<=clock_timestamp()) AND (rpv.effective_until IS NULL OR clock_timestamp()<rpv.effective_until) ORDER BY rp.code`, tenantID)
	if err != nil {
		return "", nil, fmt.Errorf("load routing policies: %w", err)
	}
	defer rows.Close()
	version := ""
	rules := []Rule{}
	for rows.Next() {
		var policy persistedPolicy
		if err := rows.Scan(&policy.ID, &policy.Code, &policy.Name, &policy.Status, &policy.Version, &policy.EffectiveFrom, &policy.Definition); err != nil {
			return "", nil, err
		}
		var definition policyDefinition
		if err := json.Unmarshal(policy.Definition, &definition); err != nil {
			return "", nil, fmt.Errorf("decode routing policy %s: %w", policy.Code, err)
		}
		if version != "" {
			version += ","
		}
		version += fmt.Sprintf("%s:v%d", policy.Code, policy.Version)
		for _, stored := range definition.Rules {
			principal, err := s.resolveSelector(ctx, tenantID, legalEntityID, stored.Selector)
			if err != nil {
				return "", nil, err
			}
			rules = append(rules, Rule{ID: stored.ID, TenantID: tenantID, LegalEntityID: stored.LegalEntityID, ObjectType: stored.ObjectType, ObjectID: stored.ObjectID, Responsibility: stored.Responsibility, DecisionType: stored.DecisionType, MinMateriality: stored.MinMateriality, Principal: principal, Priority: stored.Priority})
		}
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].Priority > rules[j].Priority })
	return version, rules, nil
}
func (s *postgresService) resolveSelector(ctx context.Context, tenantID, legalEntityID string, value selector) (Principal, error) {
	var query string
	args := []any{tenantID, value.Ref}
	switch value.Kind {
	case "POSITION":
		query = `SELECT p.id::text,p.display_name,p.kind,COALESCE(op.title,'') FROM org_positions op JOIN principals p ON p.id=op.occupant_principal_id WHERE op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND op.code=$2 AND op.valid_until IS NULL AND p.valid_until IS NULL LIMIT 1`
	case "ROLE":
		query = `SELECT p.id::text,p.display_name,p.kind,rt.name FROM role_templates rt JOIN position_role_bindings prb ON prb.role_template_id=rt.id AND prb.valid_until IS NULL JOIN org_positions op ON op.id=prb.position_id AND op.valid_until IS NULL JOIN principals p ON p.id=op.occupant_principal_id AND p.valid_until IS NULL WHERE rt.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND rt.code=$2 ORDER BY prb.priority DESC,op.code LIMIT 1`
	default:
		query = `SELECT id::text,display_name,kind,COALESCE($2,'') FROM principals WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND (id::text=$2 OR external_ref=$2) AND valid_until IS NULL LIMIT 1`
	}
	var principal Principal
	if err := s.pool.QueryRow(ctx, query, args...).Scan(&principal.ID, &principal.DisplayName, &principal.Kind, &principal.Role); err != nil {
		return Principal{}, fmt.Errorf("resolve %s selector %s: %w", value.Kind, value.Ref, err)
	}
	return principal, nil
}
