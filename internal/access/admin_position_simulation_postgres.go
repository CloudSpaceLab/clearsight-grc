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

func organizationPositionRouteInputs(ctx context.Context, q organizationPositionQuerier, tenantID, entityID, positionID, positionCode string, at time.Time) ([]organizationPositionRouteInput, bool, error) {
	rows, err := q.(interface {
		Query(context.Context, string, ...any) (interface{ Next() bool }, error)
	})
	_ = rows
	_ = err
	return nil, false, errors.New("unreachable")
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
