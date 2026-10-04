//go:build postgres && postgresintegration

package access

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationPositionRouteSimulationUsesProposedStateWithoutMutation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID   = "8a888888-8888-7888-8888-888888888801"
		entityID   = "8a888888-8888-7888-8888-888888888802"
		makerID    = "8a888888-8888-7888-8888-888888888803"
		occupantA  = "8a888888-8888-7888-8888-888888888804"
		occupantB  = "8a888888-8888-7888-8888-888888888805"
		positionID = "8a888888-8888-7888-8888-888888888806"
		policyID   = "8a888888-8888-7888-8888-888888888807"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'position-simulation-test','Position Simulation Test')`, tenantID)
	mustAdminExec(t, ctx, pool, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`,
		entityID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$4::uuid,'PERSON','Maker','ACTIVE',$5),
			($2::uuid,$4::uuid,'PERSON','Alice','ACTIVE',$5),
			($3::uuid,$4::uuid,'PERSON','Bob','ACTIVE',$5)`,
		makerID, occupantA, occupantB, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO org_positions(
			id,tenant_id,legal_entity_id,code,title,occupant_principal_id,department_path,valid_from,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'RISK_MANAGER','Risk Manager',$4::uuid,ARRAY[]::text[],$5,1)`,
		positionID, tenantID, entityID, occupantA, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO routing_policies(id,tenant_id,legal_entity_id,code,name,status,current_version)
		VALUES($1::uuid,$2::uuid,$3::uuid,'POSITION-SIM','Position simulation','ACTIVE',1)`,
		policyID, tenantID, entityID)
	mustAdminExec(t, ctx, pool, `
		INSERT INTO effective_authority_routes(
			tenant_id,source_policy_id,source_rule_id,policy_version,legal_entity_ref,
			object_type,object_id,responsibility,decision_type,min_materiality,priority,
			selector_kind,selector_ref,resolution_strategy,valid_from
		) VALUES(
			$1::uuid,$2::uuid,'risk-manager-owner','POSITION-SIM:v1',$3,
			'MATTER','*','ACCOUNTABLE_OWNER','matter.assign',0,100,
			'POSITION','RISK_MANAGER','DIRECT',$4
		)`,
		tenantID, policyID, entityID, now.Add(-time.Hour))

	admin := NewPostgresAdministrator(pool)
	revision, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID,
		Operation: OrganizationPositionUpdate, Title: "Risk Manager",
		OccupantPrincipalID: occupantB, ExpectedVersion: 1, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}

	simulation, err := admin.SimulateOrganizationPosition(ctx, SimulateOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: revision.ID, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if simulation.RevisionID != revision.ID || simulation.PositionID != positionID || simulation.SourcePositionVersion != 1 {
		t.Fatalf("simulation provenance=%#v", simulation)
	}
	if simulation.Checked != 1 || simulation.Truncated || len(simulation.Scenarios) != 1 {
		t.Fatalf("simulation population checked=%d truncated=%v scenarios=%#v", simulation.Checked, simulation.Truncated, simulation.Scenarios)
	}
	scenario := simulation.Scenarios[0]
	if !scenario.Changed || scenario.Current.Status != "RESOLVED" || scenario.Proposed.Status != "RESOLVED" {
		t.Fatalf("route status=%#v", scenario)
	}
	if len(scenario.Current.CandidateIDs) != 1 || scenario.Current.CandidateIDs[0] != occupantA {
		t.Fatalf("current route candidates=%#v", scenario.Current.CandidateIDs)
	}
	if len(scenario.Proposed.CandidateIDs) != 1 || scenario.Proposed.CandidateIDs[0] != occupantB {
		t.Fatalf("proposed route candidates=%#v", scenario.Proposed.CandidateIDs)
	}

	var currentOccupant string
	if err := pool.QueryRow(ctx, `
		SELECT occupant_principal_id::text
		FROM org_positions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
		tenantID, entityID, positionID).Scan(&currentOccupant); err != nil {
		t.Fatal(err)
	}
	if currentOccupant != occupantA {
		t.Fatalf("simulation mutated live position occupant=%s", currentOccupant)
	}
}
