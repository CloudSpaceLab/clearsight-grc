//go:build postgres

package access

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/jackc/pgx/v5"
)

const organizationPositionSimulationLimit = 100

type organizationPositionRouteInput struct {
	ObjectType     string
	ObjectID       string
	Responsibility string
	DecisionType   string
	Materiality    int
}

func (a *PostgresAdministrator) SimulateOrganizationPosition(ctx context.Context, input SimulateOrganizationPositionInput) (OrganizationPositionRouteSimulation, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.TrimSpace(input.LegalEntityID)
	input.RevisionID = strings.TrimSpace(input.RevisionID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.RevisionID == "" || input.ActorID == "" {
		return OrganizationPositionRouteSimulation{}, ErrAdminInvalid
	}
	tenantID, entityID, err := a.scopeIDs(ctx, input.TenantID, input.LegalEntityID)
	if err != nil {
		return OrganizationPositionRouteSimulation{}, err
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return OrganizationPositionRouteSimulation{}, err
	}
	defer tx.Rollback(ctx)
	if err := ensureAdminActor(ctx, tx, input.TenantID, input.ActorID); err != nil {
		return OrganizationPositionRouteSimulation{}, err
	}
	revision, err := organizationPositionRevision(ctx, tx, tenantID, entityID, input.RevisionID, true)
	if err != nil {
		return OrganizationPositionRouteSimulation{}, err
	}
	if revision.Status != "PENDING" && revision.Status != "SCHEDULED" {
		return OrganizationPositionRouteSimulation{}, ErrAdminConflict
	}
	effectiveAt := time.Now().UTC()
	if revision.EffectiveFrom != nil && revision.EffectiveFrom.After(effectiveAt) {
		effectiveAt = revision.EffectiveFrom.UTC()
	}
	scenarios, truncated, err := organizationPositionRouteInputs(ctx, tx, tenantID, entityID, revision.PositionID, revision.Proposed.Code, effectiveAt)
	if err != nil {
		return OrganizationPositionRouteSimulation{}, err
	}
	service := authority.NewEffectivePostgresService(a.pool)
	authorityCtx := authority.WithPostgresTransaction(ctx, tx)
	current := make([]OrganizationPositionRouteSnapshot, len(scenarios))
	for index, scenario := range scenarios {
		current[index], err = simulateOrganizationPositionRoute(authorityCtx, service, tenantID, entityID, scenario, effectiveAt)
		if err != nil {
			return OrganizationPositionRouteSimulation{}, err
		}
	}
	if err := applyOrganizationPositionRevision(ctx, tx, tenantID, entityID, revision); err != nil {
		return OrganizationPositionRouteSimulation{}, err
	}
	result := OrganizationPositionRouteSimulation{
		RevisionID: revision.ID, PositionID: revision.PositionID, SourcePositionVersion: revision.BaseVersion,
		EffectiveAt: effectiveAt, Checked: len(scenarios), Truncated: truncated,
		Scenarios: make([]OrganizationPositionRouteScenario, 0, len(scenarios)),
	}
	for index, scenario := range scenarios {
		proposed, err := simulateOrganizationPositionRoute(authorityCtx, service, tenantID, entityID, scenario, effectiveAt)
		if err != nil {
			return OrganizationPositionRouteSimulation{}, err
		}
		result.Scenarios = append(result.Scenarios, OrganizationPositionRouteScenario{
			ObjectType: scenario.ObjectType, ObjectID: scenario.ObjectID,
			Responsibility: scenario.Responsibility, DecisionType: scenario.DecisionType, Materiality: scenario.Materiality,
			Current: current[index], Proposed: proposed, Changed: !sameOrganizationPositionRouteSnapshot(current[index], proposed),
		})
	}
	return result, nil
}

type organizationPositionRouteQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func organizationPositionRouteInputs(ctx context.Context, q organizationPositionRouteQuerier, tenantID, entityID, positionID, positionCode string, at time.Time) ([]organizationPositionRouteInput, bool, error) {
	rows, err := q.Query(ctx, `
		WITH current_entity AS (
			SELECT id,code
			FROM legal_entities
			WHERE tenant_id=$1::uuid AND id=$2::uuid
			  AND valid_from<=$5 AND (valid_until IS NULL OR $5<valid_until)
		), role_refs AS (
			SELECT role.id AS role_id,role.id::text AS role_id_text,role.code AS role_code
			FROM position_role_bindings binding
			JOIN role_templates role ON role.tenant_id=binding.tenant_id AND role.id=binding.role_template_id
			WHERE binding.tenant_id=$1::uuid AND binding.position_id=$3::uuid
			  AND binding.valid_from<=$5 AND (binding.valid_until IS NULL OR $5<binding.valid_until)
			  AND role.valid_from<=$5 AND (role.valid_until IS NULL OR $5<role.valid_until)
			  AND (
			    NOT (binding.scope ? 'legal_entity_id')
			    OR binding.scope->>'legal_entity_id' IN ('*',$2)
			  )
		), scenarios AS (
			SELECT route.object_type,route.object_id,route.responsibility,route.decision_type,route.min_materiality AS materiality
			FROM effective_authority_routes route
			CROSS JOIN current_entity entity
			WHERE route.tenant_id=$1::uuid
			  AND route.valid_from<=$5 AND (route.valid_until IS NULL OR $5<route.valid_until)
			  AND route.legal_entity_ref IN ('*',$2,entity.code)
			  AND (
			    (
			      route.selector_kind IN ('POSITION','POSITION_ID')
			      AND route.selector_ref IN ($3,$4)
			    )
			    OR (
			      route.selector_kind IN ('ROLE','ROLE_ID')
			      AND EXISTS (
			        SELECT 1 FROM role_refs role
			        WHERE route.selector_ref IN (role.role_code,role.role_id_text)
			      )
			    )
			  )
			UNION
			SELECT assignment.object_type,COALESCE(assignment.object_id::text,'*'),assignment.responsibility,
			       COALESCE(assignment.decision_type,''),0
			FROM responsibility_assignments assignment
			WHERE assignment.tenant_id=$1::uuid
			  AND (assignment.legal_entity_id IS NULL OR assignment.legal_entity_id=$2::uuid)
			  AND assignment.valid_from<=$5 AND (assignment.valid_until IS NULL OR $5<assignment.valid_until)
			  AND (
			    assignment.position_id=$3::uuid
			    OR assignment.role_template_id IN (SELECT role_id FROM role_refs)
			  )
		)
		SELECT object_type,object_id,responsibility,decision_type,materiality
		FROM scenarios
		ORDER BY object_type,object_id,responsibility,decision_type,materiality
		LIMIT $6`,
		tenantID, entityID, positionID, positionCode, at, organizationPositionSimulationLimit+1)
	if err != nil {
		return nil, false, fmt.Errorf("list organization position route scenarios: %w", err)
	}
	defer rows.Close()
	values := make([]organizationPositionRouteInput, 0, organizationPositionSimulationLimit+1)
	for rows.Next() {
		var value organizationPositionRouteInput
		if err := rows.Scan(&value.ObjectType, &value.ObjectID, &value.Responsibility, &value.DecisionType, &value.Materiality); err != nil {
			return nil, false, err
		}
		value.ObjectType = strings.TrimSpace(value.ObjectType)
		value.ObjectID = strings.TrimSpace(value.ObjectID)
		value.Responsibility = strings.TrimSpace(value.Responsibility)
		value.DecisionType = strings.TrimSpace(value.DecisionType)
		if value.ObjectType == "" || value.ObjectID == "" || value.Responsibility == "" {
			return nil, false, ErrAdminConflict
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	truncated := len(values) > organizationPositionSimulationLimit
	if truncated {
		values = values[:organizationPositionSimulationLimit]
	}
	return values, truncated, nil
}

func simulateOrganizationPositionRoute(ctx context.Context, service authority.Service, tenantID, entityID string, scenario organizationPositionRouteInput, at time.Time) (OrganizationPositionRouteSnapshot, error) {
	simulation, err := service.Simulate(ctx, authority.ResolveInput{
		TenantID: tenantID, LegalEntityID: entityID,
		ObjectType: scenario.ObjectType, ObjectID: scenario.ObjectID,
		Responsibility: authority.Responsibility(scenario.Responsibility),
		DecisionType: scenario.DecisionType, Materiality: scenario.Materiality, At: at,
	})
	if err != nil {
		if errors.Is(err, authority.ErrAmbiguousRoute) {
			return OrganizationPositionRouteSnapshot{Status: "AMBIGUOUS_ROUTE", CandidateIDs: []string{}}, nil
		}
		if errors.Is(err, authority.ErrNoRoute) {
			return OrganizationPositionRouteSnapshot{Status: "NO_ROUTE", CandidateIDs: []string{}}, nil
		}
		return OrganizationPositionRouteSnapshot{}, fmt.Errorf("simulate organization position route: %w", err)
	}
	candidates := make([]string, 0, len(simulation.Candidates))
	seen := make(map[string]struct{}, len(simulation.Candidates))
	for _, candidate := range simulation.Candidates {
		id := strings.TrimSpace(candidate.Principal.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		candidates = append(candidates, id)
	}
	sort.Strings(candidates)
	status := "RESOLVED"
	if len(candidates) == 0 || simulation.Selected == nil {
		status = "NO_ROUTE"
	}
	return OrganizationPositionRouteSnapshot{
		Status: status, CandidateIDs: candidates, PolicyVersion: simulation.PolicyVersion,
	}, nil
}

func sameOrganizationPositionRouteSnapshot(a, b OrganizationPositionRouteSnapshot) bool {
	if a.Status != b.Status || a.PolicyVersion != b.PolicyVersion || len(a.CandidateIDs) != len(b.CandidateIDs) {
		return false
	}
	for index := range a.CandidateIDs {
		if a.CandidateIDs[index] != b.CandidateIDs[index] {
			return false
		}
	}
	return true
}
