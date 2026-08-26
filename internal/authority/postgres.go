//go:build postgres

package authority

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresService struct{ pool *pgxpool.Pool }

func NewPostgresService(pool *pgxpool.Pool) Service { return &postgresService{pool: pool} }

type routeGroup struct {
	RuleID        string
	PolicyVersion string
	Strategy      string
	Priority      int
	Specificity   int
	Candidates    []Principal
}

type effectiveCandidate struct {
	Principal Principal
	OriginID  string
}

func (s *postgresService) Policies(ctx context.Context, tenantID string) ([]PolicySummary, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT rp.id::text,rp.code,rp.name,rp.status,rpv.version,
		       COALESCE(rpv.effective_from,'epoch'::timestamptz)
		FROM routing_policies rp
		JOIN routing_policy_versions rpv ON rpv.policy_id=rp.id AND rpv.version=rp.current_version
		WHERE rp.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		ORDER BY rp.code`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list routing policies: %w", err)
	}
	defer rows.Close()
	result := []PolicySummary{}
	for rows.Next() {
		var value PolicySummary
		var effective time.Time
		if err := rows.Scan(&value.ID, &value.Code, &value.Name, &value.Status, &value.Version, &effective); err != nil {
			return nil, err
		}
		if !effective.Equal(time.Unix(0, 0).UTC()) {
			value.EffectiveFrom = &effective
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
	if err := validateInput(input); err != nil {
		return Simulation{}, err
	}
	if input.At.IsZero() {
		input.At = time.Now().UTC()
	}
	groups, err := s.resolveRouteGroups(ctx, input)
	if err != nil {
		return Simulation{}, err
	}
	if len(groups) == 0 {
		return Simulation{Candidates: []Candidate{}}, nil
	}

	top := groups[0]
	for _, candidate := range groups[1:] {
		if candidate.Priority != top.Priority || candidate.Specificity != top.Specificity {
			break
		}
		if !samePrincipalSlice(top.Candidates, candidate.Candidates) {
			return Simulation{}, fmt.Errorf("%w: rules %s and %s have the same effective rank", ErrAmbiguousRoute, top.RuleID, candidate.RuleID)
		}
	}

	effective, err := s.expandDelegations(ctx, input, top.Candidates)
	if err != nil {
		return Simulation{}, err
	}
	effective, err = s.applyGrantBoundary(ctx, input, effective)
	if err != nil {
		return Simulation{}, err
	}
	effective, err = s.applySegregationBoundary(ctx, input, effective)
	if err != nil {
		return Simulation{}, err
	}
	principals := uniquePrincipals(effective)
	if len(principals) == 0 {
		return Simulation{Candidates: []Candidate{}, PolicyVersion: top.PolicyVersion}, nil
	}

	strategy := top.Strategy
	if len(principals) > 1 {
		strategy = "CANDIDATE_SET"
	}
	selected := Resolution{
		Principal:           principals[0],
		CandidatePrincipals: principals,
		EffectiveOrigins:    uniqueEffectiveOrigins(effective),
		Strategy:            strategy,
		RuleID:              top.RuleID,
		PolicyVersion:       top.PolicyVersion,
		Explanation:         resolutionExplanation(principals[0], principals, strategy, input),
	}
	candidates := make([]Candidate, 0, len(principals))
	for _, principal := range principals {
		candidates = append(candidates, Candidate{Principal: principal, RuleID: top.RuleID, Priority: top.Priority, Eligible: true, Reason: "eligible after current delegation, grant and segregation checks"})
	}
	return Simulation{Selected: &selected, Candidates: candidates, PolicyVersion: top.PolicyVersion}, nil
}

func (s *postgresService) resolveRouteGroups(ctx context.Context, input ResolveInput) ([]routeGroup, error) {
	rows, err := s.pool.Query(ctx, `
		WITH route_defs AS (
			SELECT
				ear.source_rule_id AS rule_id,
				ear.policy_version,
				ear.legal_entity_ref,
				ear.object_type,
				ear.object_id,
				ear.responsibility,
				ear.decision_type,
				ear.min_materiality,
				ear.priority,
				ear.selector_kind,
				ear.selector_ref,
				ear.resolution_strategy,
				ear.valid_from,
				ear.valid_until,
				(CASE WHEN ear.legal_entity_ref <> '*' THEN 8 ELSE 0 END +
				 CASE WHEN ear.object_type <> '*' THEN 4 ELSE 0 END +
				 CASE WHEN ear.object_id <> '*' THEN 2 ELSE 0 END +
				 CASE WHEN ear.decision_type <> '' THEN 1 ELSE 0 END) AS specificity
			FROM effective_authority_routes ear
			WHERE ear.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND (ear.legal_entity_ref='*' OR ear.legal_entity_ref=$2 OR ear.legal_entity_ref IN (
				SELECT le.id::text FROM legal_entities le
				WHERE le.tenant_id=ear.tenant_id AND (le.id::text=$2 OR le.code=$2)
			  ))
			  AND (ear.object_type='*' OR upper(ear.object_type)=upper($3))
			  AND (ear.object_id='*' OR ear.object_id=$4)
			  AND ear.responsibility=$5
			  AND (ear.decision_type='' OR upper(ear.decision_type)=upper($6))
			  AND ear.min_materiality <= $7
			  AND ear.valid_from <= $8
			  AND (ear.valid_until IS NULL OR $8 < ear.valid_until)
			UNION ALL
			SELECT
				'assignment:' || ra.id::text,
				ra.policy_version,
				COALESCE(le.code,'*'),
				ra.object_type,
				COALESCE(ra.object_id::text,'*'),
				ra.responsibility,
				COALESCE(ra.decision_type,''),
				0,
				ra.priority,
				CASE WHEN ra.principal_id IS NOT NULL THEN 'PRINCIPAL_ID' WHEN ra.position_id IS NOT NULL THEN 'POSITION_ID' ELSE 'ROLE_ID' END,
				COALESCE(ra.principal_id::text,ra.position_id::text,ra.role_template_id::text),
				upper(COALESCE(NULLIF(ra.resolution_strategy,''),'DIRECT')),
				ra.valid_from,
				ra.valid_until,
				(CASE WHEN ra.legal_entity_id IS NOT NULL THEN 8 ELSE 0 END + 4 +
				 CASE WHEN ra.object_id IS NOT NULL THEN 2 ELSE 0 END +
				 CASE WHEN COALESCE(ra.decision_type,'') <> '' THEN 1 ELSE 0 END)
			FROM responsibility_assignments ra
			LEFT JOIN legal_entities le ON le.id=ra.legal_entity_id
			WHERE ra.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND (ra.legal_entity_id IS NULL OR ra.legal_entity_id IN (
				SELECT le2.id FROM legal_entities le2
				WHERE le2.tenant_id=ra.tenant_id AND (le2.id::text=$2 OR le2.code=$2)
			  ))
			  AND upper(ra.object_type)=upper($3)
			  AND (ra.object_id IS NULL OR ra.object_id::text=$4)
			  AND ra.responsibility=$5
			  AND (COALESCE(ra.decision_type,'')='' OR upper(ra.decision_type)=upper($6))
			  AND ra.valid_from <= $8
			  AND (ra.valid_until IS NULL OR $8 < ra.valid_until)
		), resolved AS (
			SELECT rd.*, p.id, p.display_name, p.kind, p.role
			FROM route_defs rd
			JOIN LATERAL (
				SELECT p.id::text AS id,p.display_name,p.kind,COALESCE(rt.name,'') AS role
				FROM principals p
				LEFT JOIN role_templates rt ON false
				WHERE rd.selector_kind IN ('PRINCIPAL','TEAM','QUEUE','COMMITTEE')
				  AND p.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
				  AND (p.id::text=rd.selector_ref OR p.external_ref=rd.selector_ref)
				  AND p.status='ACTIVE' AND p.valid_from <= $8 AND (p.valid_until IS NULL OR $8 < p.valid_until)
				  AND (rd.selector_kind='PRINCIPAL' OR p.kind=rd.selector_kind)
				UNION ALL
				SELECT p.id::text,p.display_name,p.kind,''
				FROM principals p
				WHERE rd.selector_kind='PRINCIPAL_ID' AND p.id::text=rd.selector_ref
				  AND p.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
				  AND p.status='ACTIVE' AND p.valid_from <= $8 AND (p.valid_until IS NULL OR $8 < p.valid_until)
				UNION ALL
				SELECT p.id::text,p.display_name,p.kind,COALESCE(op.title,'')
				FROM org_positions op
				JOIN principals p ON p.id=op.occupant_principal_id
				WHERE rd.selector_kind IN ('POSITION','POSITION_ID')
				  AND op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
				  AND ((rd.selector_kind='POSITION' AND (op.code=rd.selector_ref OR op.id::text=rd.selector_ref)) OR (rd.selector_kind='POSITION_ID' AND op.id::text=rd.selector_ref))
				  AND (op.legal_entity_id IS NULL OR op.legal_entity_id IN (
					SELECT le.id FROM legal_entities le WHERE le.tenant_id=op.tenant_id AND (le.id::text=$2 OR le.code=$2)
				  ))
				  AND op.valid_from <= $8 AND (op.valid_until IS NULL OR $8 < op.valid_until)
				  AND p.status='ACTIVE' AND p.valid_from <= $8 AND (p.valid_until IS NULL OR $8 < p.valid_until)
				UNION ALL
				SELECT p.id::text,p.display_name,p.kind,rt.name
				FROM role_templates rt
				JOIN position_role_bindings prb ON prb.role_template_id=rt.id
				JOIN org_positions op ON op.id=prb.position_id
				JOIN principals p ON p.id=op.occupant_principal_id
				WHERE rd.selector_kind IN ('ROLE','ROLE_ID')
				  AND rt.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
				  AND ((rd.selector_kind='ROLE' AND (rt.code=rd.selector_ref OR rt.id::text=rd.selector_ref)) OR (rd.selector_kind='ROLE_ID' AND rt.id::text=rd.selector_ref))
				  AND (op.legal_entity_id IS NULL OR op.legal_entity_id IN (
					SELECT le.id FROM legal_entities le WHERE le.tenant_id=op.tenant_id AND (le.id::text=$2 OR le.code=$2)
				  ))
				  AND rt.valid_from <= $8 AND (rt.valid_until IS NULL OR $8 < rt.valid_until)
				  AND prb.valid_from <= $8 AND (prb.valid_until IS NULL OR $8 < prb.valid_until)
				  AND op.valid_from <= $8 AND (op.valid_until IS NULL OR $8 < op.valid_until)
				  AND p.status='ACTIVE' AND p.valid_from <= $8 AND (p.valid_until IS NULL OR $8 < p.valid_until)
			) p ON true
		)
		SELECT rule_id,policy_version,resolution_strategy,priority,specificity,id,display_name,kind,role
		FROM resolved
		ORDER BY priority DESC,specificity DESC,rule_id,id`, input.TenantID, input.LegalEntityID, input.ObjectType, input.ObjectID, string(input.Responsibility), input.DecisionType, input.Materiality, input.At)
	if err != nil {
		return nil, fmt.Errorf("resolve effective authority routes: %w", err)
	}
	defer rows.Close()

	groupsByKey := map[string]*routeGroup{}
	order := []string{}
	for rows.Next() {
		var ruleID, policyVersion, strategy, id, displayName, kind, role string
		var priority, specificity int
		if err := rows.Scan(&ruleID, &policyVersion, &strategy, &priority, &specificity, &id, &displayName, &kind, &role); err != nil {
			return nil, err
		}
		key := ruleID + "\x00" + policyVersion
		group := groupsByKey[key]
		if group == nil {
			group = &routeGroup{RuleID: ruleID, PolicyVersion: policyVersion, Strategy: strategy, Priority: priority, Specificity: specificity}
			groupsByKey[key] = group
			order = append(order, key)
		}
		group.Candidates = append(group.Candidates, Principal{ID: id, DisplayName: displayName, Kind: kind, Role: role})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]routeGroup, 0, len(order))
	for _, key := range order {
		group := groupsByKey[key]
		group.Candidates = uniquePrincipalList(group.Candidates)
		result = append(result, *group)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Priority != result[j].Priority {
			return result[i].Priority > result[j].Priority
		}
		if result[i].Specificity != result[j].Specificity {
			return result[i].Specificity > result[j].Specificity
		}
		return result[i].RuleID < result[j].RuleID
	})
	return result, nil
}

func (s *postgresService) expandDelegations(ctx context.Context, input ResolveInput, seeds []Principal) ([]effectiveCandidate, error) {
	seedIDs := make([]string, 0, len(seeds))
	seedByID := make(map[string]Principal, len(seeds))
	for _, principal := range seeds {
		if principal.ID == "" {
			continue
		}
		seedIDs = append(seedIDs, principal.ID)
		seedByID[principal.ID] = principal
	}
	encoded, err := json.Marshal(seedIDs)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		WITH RECURSIVE seed(origin_id,principal_id,path,depth) AS (
			SELECT value::uuid,value::uuid,ARRAY[value::uuid],0
			FROM jsonb_array_elements_text($3::jsonb)
		), chain(origin_id,principal_id,path,depth) AS (
			SELECT origin_id,principal_id,path,depth FROM seed
			UNION ALL
			SELECT c.origin_id,d.to_principal_id,c.path || d.to_principal_id,c.depth+1
			FROM chain c
			JOIN delegations d ON d.from_principal_id=c.principal_id
			WHERE d.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND d.responsibility=$4
			  AND d.status='ACTIVE'
			  AND d.starts_at <= $2 AND $2 < d.ends_at
			  AND c.depth < 8
			  AND NOT d.to_principal_id=ANY(c.path)
			  AND (NOT (d.scope ? 'legal_entity_id') OR d.scope->>'legal_entity_id' IN ('*',$5))
			  AND (NOT (d.scope ? 'object_type') OR upper(d.scope->>'object_type') IN ('*',upper($6)))
			  AND (NOT (d.scope ? 'object_id') OR d.scope->>'object_id' IN ('*',$7))
			  AND (NOT (d.scope ? 'decision_type') OR upper(d.scope->>'decision_type') IN ('*',upper($8)))
		)
		SELECT DISTINCT c.origin_id::text,p.id::text,p.display_name,p.kind,''
		FROM chain c
		JOIN principals p ON p.id=c.principal_id
		WHERE p.status='ACTIVE' AND p.valid_from <= $2 AND (p.valid_until IS NULL OR $2 < p.valid_until)
		ORDER BY c.origin_id::text,p.id::text`, input.TenantID, input.At, string(encoded), string(input.Responsibility), input.LegalEntityID, input.ObjectType, input.ObjectID, input.DecisionType)
	if err != nil {
		return nil, fmt.Errorf("expand active delegations: %w", err)
	}
	defer rows.Close()
	result := []effectiveCandidate{}
	for rows.Next() {
		var originID string
		var principal Principal
		if err := rows.Scan(&originID, &principal.ID, &principal.DisplayName, &principal.Kind, &principal.Role); err != nil {
			return nil, err
		}
		if original, ok := seedByID[principal.ID]; ok && original.Role != "" {
			principal.Role = original.Role
		}
		result = append(result, effectiveCandidate{Principal: principal, OriginID: originID})
	}
	return result, rows.Err()
}

func (s *postgresService) applyGrantBoundary(ctx context.Context, input ResolveInput, candidates []effectiveCandidate) ([]effectiveCandidate, error) {
	var hasGrants bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM authority_grants ag
			WHERE ag.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND (ag.legal_entity_id IS NULL OR ag.legal_entity_id IN (
				SELECT le.id FROM legal_entities le WHERE le.tenant_id=ag.tenant_id AND (le.id::text=$2 OR le.code=$2)
			  ))
			  AND (ag.decision_type='*' OR upper(ag.decision_type)=upper($3))
			  AND ag.valid_from <= $4 AND (ag.valid_until IS NULL OR $4 < ag.valid_until)
		)`, input.TenantID, input.LegalEntityID, input.DecisionType, input.At).Scan(&hasGrants); err != nil {
		return nil, fmt.Errorf("check authority grants: %w", err)
	}
	if !hasGrants {
		return candidates, nil
	}

	ids := make([]string, 0, len(candidates)*2)
	for _, candidate := range candidates {
		ids = append(ids, candidate.Principal.ID, candidate.OriginID)
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		WITH requested(id) AS (
			SELECT DISTINCT value::uuid FROM jsonb_array_elements_text($5::jsonb)
		), grants AS (
			SELECT ag.* FROM authority_grants ag
			WHERE ag.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND (ag.legal_entity_id IS NULL OR ag.legal_entity_id IN (
				SELECT le.id FROM legal_entities le WHERE le.tenant_id=ag.tenant_id AND (le.id::text=$2 OR le.code=$2)
			  ))
			  AND (ag.decision_type='*' OR upper(ag.decision_type)=upper($3))
			  AND ag.valid_from <= $4 AND (ag.valid_until IS NULL OR $4 < ag.valid_until)
			  AND COALESCE(NULLIF(ag.limits->>'min_materiality','')::integer,0) <= $6
			  AND COALESCE(NULLIF(ag.limits->>'max_materiality','')::integer,5) >= $6
		), granted AS (
			SELECT g.principal_id AS id FROM grants g WHERE g.principal_id IS NOT NULL
			UNION
			SELECT op.occupant_principal_id FROM grants g JOIN org_positions op ON op.id=g.position_id
			WHERE g.position_id IS NOT NULL AND op.valid_from <= $4 AND (op.valid_until IS NULL OR $4 < op.valid_until)
			UNION
			SELECT op.occupant_principal_id
			FROM grants g
			JOIN position_role_bindings prb ON prb.role_template_id=g.role_template_id
			JOIN org_positions op ON op.id=prb.position_id
			WHERE g.role_template_id IS NOT NULL
			  AND prb.valid_from <= $4 AND (prb.valid_until IS NULL OR $4 < prb.valid_until)
			  AND op.valid_from <= $4 AND (op.valid_until IS NULL OR $4 < op.valid_until)
		)
		SELECT DISTINCT r.id::text FROM requested r JOIN granted g ON g.id=r.id`, input.TenantID, input.LegalEntityID, input.DecisionType, input.At, string(encoded), input.Materiality)
	if err != nil {
		return nil, fmt.Errorf("resolve authority grants: %w", err)
	}
	defer rows.Close()
	allowed := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		allowed[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	filtered := candidates[:0]
	for _, candidate := range candidates {
		_, direct := allowed[candidate.Principal.ID]
		_, origin := allowed[candidate.OriginID]
		if direct || origin {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

func (s *postgresService) applySegregationBoundary(ctx context.Context, input ResolveInput, candidates []effectiveCandidate) ([]effectiveCandidate, error) {
	ids := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.Principal.ID)
	}
	encoded, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		WITH requested(id) AS (
			SELECT DISTINCT value::uuid FROM jsonb_array_elements_text($4::jsonb)
		)
		SELECT DISTINCT r.id::text
		FROM requested r
		JOIN org_positions op ON op.occupant_principal_id=r.id
		JOIN position_role_bindings prb ON prb.position_id=op.id
		JOIN role_templates rt ON rt.id=prb.role_template_id
		JOIN segregation_rules sr ON sr.tenant_id=op.tenant_id AND sr.prohibited_role_code=rt.code
		WHERE op.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND sr.responsibility=$2 AND sr.status='ACTIVE'
		  AND sr.valid_from <= $3 AND (sr.valid_until IS NULL OR $3 < sr.valid_until)
		  AND op.valid_from <= $3 AND (op.valid_until IS NULL OR $3 < op.valid_until)
		  AND prb.valid_from <= $3 AND (prb.valid_until IS NULL OR $3 < prb.valid_until)
		  AND rt.valid_from <= $3 AND (rt.valid_until IS NULL OR $3 < rt.valid_until)`, input.TenantID, string(input.Responsibility), input.At, string(encoded))
	if err != nil {
		return nil, fmt.Errorf("apply segregation rules: %w", err)
	}
	defer rows.Close()
	blocked := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		blocked[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	filtered := candidates[:0]
	for _, candidate := range candidates {
		if _, denied := blocked[candidate.Principal.ID]; !denied {
			filtered = append(filtered, candidate)
		}
	}
	return filtered, nil
}

func (s *postgresService) Integrity(ctx context.Context, tenantID string) ([]IntegrityFinding, error) {
	if strings.TrimSpace(tenantID) == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}
	findings := []IntegrityFinding{}
	var authorizers int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM effective_authority_routes WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND responsibility='AUTHORIZER' AND valid_from<=clock_timestamp() AND (valid_until IS NULL OR clock_timestamp()<valid_until)`, tenantID).Scan(&authorizers); err != nil {
		return nil, err
	}
	if authorizers == 0 {
		findings = append(findings, IntegrityFinding{Type: "MISSING_AUTHORIZER", Severity: "CRITICAL", Summary: "No active authorizer route exists for this tenant.", RequiredAction: "Create and approve at least one scoped authorizer route."})
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.source_rule_id,b.source_rule_id
		FROM effective_authority_routes a
		JOIN effective_authority_routes b ON b.tenant_id=a.tenant_id AND b.id>a.id
		 AND b.legal_entity_ref=a.legal_entity_ref AND b.object_type=a.object_type AND b.object_id=a.object_id
		 AND b.responsibility=a.responsibility AND b.decision_type=a.decision_type
		 AND b.min_materiality=a.min_materiality AND b.priority=a.priority
		WHERE a.tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
		  AND (a.selector_kind<>b.selector_kind OR a.selector_ref<>b.selector_ref)`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var left, right string
		if err := rows.Scan(&left, &right); err != nil {
			return nil, err
		}
		findings = append(findings, IntegrityFinding{Type: "AMBIGUOUS_ROUTE", Severity: "HIGH", Summary: "Two active rules have the same scope and priority but different selectors.", RequiredAction: "Change priority or narrow one rule before activation.", RuleIDs: []string{left, right}})
	}
	return findings, rows.Err()
}

func samePrincipalSlice(left, right []Principal) bool {
	left = uniquePrincipalList(left)
	right = uniquePrincipalList(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ID != right[index].ID {
			return false
		}
	}
	return true
}

func uniquePrincipalList(values []Principal) []Principal {
	seen := make(map[string]Principal, len(values))
	for _, value := range values {
		if value.ID != "" {
			seen[value.ID] = value
		}
	}
	result := make([]Principal, 0, len(seen))
	for _, value := range seen {
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func uniquePrincipals(values []effectiveCandidate) []Principal {
	principals := make([]Principal, 0, len(values))
	for _, value := range values {
		principals = append(principals, value.Principal)
	}
	return uniquePrincipalList(principals)
}

func uniqueEffectiveOrigins(values []effectiveCandidate) []EffectiveOrigin {
	seen := make(map[string]EffectiveOrigin, len(values))
	for _, value := range values {
		if value.Principal.ID == "" || value.OriginID == "" {
			continue
		}
		key := value.Principal.ID + "\x00" + value.OriginID
		seen[key] = EffectiveOrigin{PrincipalID: value.Principal.ID, OriginPrincipalID: value.OriginID}
	}
	result := make([]EffectiveOrigin, 0, len(seen))
	for _, value := range seen {
		result = append(result, value)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].PrincipalID == result[j].PrincipalID {
			return result[i].OriginPrincipalID < result[j].OriginPrincipalID
		}
		return result[i].PrincipalID < result[j].PrincipalID
	})
	return result
}
