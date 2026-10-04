//go:build postgres && postgresintegration

package access

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationPositionEffectiveActivationIsDueTimeBoundAndFailsClosed(t *testing.T) {
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
		tenantID   = "8a777777-7777-7777-8777-777777777701"
		entityID   = "8a777777-7777-7777-8777-777777777702"
		makerID    = "8a777777-7777-7777-8777-777777777703"
		checkerID  = "8a777777-7777-7777-8777-777777777704"
		occupantA  = "8a777777-7777-7777-8777-777777777705"
		occupantB  = "8a777777-7777-7777-8777-777777777706"
		positionID = "8a777777-7777-7777-8777-777777777707"
		scopeID    = "8a777777-7777-7777-8777-777777777708"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'position-activation-test','Position Activation Test')`, tenantID)
	mustAdminExec(t, ctx, pool, `
		INSERT INTO legal_entities(id,tenant_id,code,name,jurisdiction,valid_from)
		VALUES($1::uuid,$2::uuid,'BANK-NG','Bank NG','NG',$3)`,
		entityID, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES
			($1::uuid,$5::uuid,'PERSON','Maker','ACTIVE',$6),
			($2::uuid,$5::uuid,'PERSON','Checker','ACTIVE',$6),
			($3::uuid,$5::uuid,'PERSON','Alice','ACTIVE',$6),
			($4::uuid,$5::uuid,'PERSON','Bob','ACTIVE',$6)`,
		makerID, checkerID, occupantA, occupantB, tenantID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO organization_scopes(
			id,tenant_id,legal_entity_id,code,name,kind,department_path,origin,status,valid_from
		) VALUES($1::uuid,$2::uuid,$3::uuid,'RISK','Risk','DEPARTMENT',ARRAY['BANK','RISK'],'MANAGED','ACTIVE',$4)`,
		scopeID, tenantID, entityID, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO org_positions(
			id,tenant_id,legal_entity_id,code,title,organization_scope_id,occupant_principal_id,department_path,valid_from,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'RISK_MANAGER','Risk Manager',$4::uuid,$5::uuid,ARRAY['BANK','RISK'],$6,1)`,
		positionID, tenantID, entityID, scopeID, occupantA, now.Add(-time.Hour))

	admin := NewPostgresAdministrator(pool)
	effectiveFrom := now.Add(time.Hour)
	revision, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID,
		Operation: OrganizationPositionUpdate, Title: "Risk Manager", OrganizationScopeID: scopeID,
		OccupantPrincipalID: occupantB, ExpectedVersion: 1, EffectiveFrom: &effectiveFrom, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: revision.ID,
		ActorID: checkerID, Rationale: "activate after planned handover",
	}); err != nil {
		t.Fatal(err)
	}

	var status, currentOccupant string
	var attempts int
	if err := pool.QueryRow(ctx, `
		SELECT status,activation_attempts FROM organization_position_revisions WHERE id=$1::uuid`,
		revision.ID).Scan(&status, &attempts); err != nil {
		t.Fatal(err)
	}
	if status != "SCHEDULED" || attempts != 0 {
		t.Fatalf("scheduled revision status=%s attempts=%d", status, attempts)
	}
	if err := pool.QueryRow(ctx, `SELECT occupant_principal_id::text FROM org_positions WHERE id=$1::uuid`, positionID).Scan(&currentOccupant); err != nil {
		t.Fatal(err)
	}
	if currentOccupant != occupantA {
		t.Fatalf("future change activated early: occupant=%s", currentOccupant)
	}

	maintainer := NewOrganizationPositionActivationMaintainer(pool)
	if count, err := maintainer.Maintain(ctx, effectiveFrom.Add(-time.Second), 10); err != nil || count != 0 {
		t.Fatalf("early activation count=%d err=%v", count, err)
	}
	if count, err := maintainer.Maintain(ctx, effectiveFrom, 10); err != nil || count != 1 {
		t.Fatalf("due activation count=%d err=%v", count, err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT r.status,r.activation_attempts,p.occupant_principal_id::text
		FROM organization_position_revisions r
		JOIN org_positions p ON p.id=r.position_id
		WHERE r.id=$1::uuid`, revision.ID).Scan(&status, &attempts, &currentOccupant); err != nil {
		t.Fatal(err)
	}
	if status != "APPLIED" || attempts != 1 || currentOccupant != occupantB {
		t.Fatalf("activated status=%s attempts=%d occupant=%s", status, attempts, currentOccupant)
	}
	var actorType, fromState, toState string
	var actorIsNull bool
	if err := pool.QueryRow(ctx, `
		SELECT actor_type,actor_id IS NULL,from_state,to_state
		FROM governance_decisions
		WHERE tenant_id=$1::uuid AND object_type='ORGANIZATION_POSITION' AND object_id=$2::uuid
		ORDER BY decided_at DESC,id DESC LIMIT 1`,
		tenantID, positionID).Scan(&actorType, &actorIsNull, &fromState, &toState); err != nil {
		t.Fatal(err)
	}
	if actorType != "SYSTEM" || !actorIsNull || fromState != "SCHEDULED" || toState != "ACTIVE" {
		t.Fatalf("activation provenance actor=%s null=%v transition=%s->%s", actorType, actorIsNull, fromState, toState)
	}

	secondEffective := effectiveFrom.Add(time.Hour)
	stale, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: positionID,
		Operation: OrganizationPositionUpdate, Title: "Senior Risk Manager", OrganizationScopeID: scopeID,
		OccupantPrincipalID: occupantB, ExpectedVersion: 2, EffectiveFrom: &secondEffective, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: stale.ID,
		ActorID: checkerID, Rationale: "future title change",
	}); err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `
		UPDATE org_positions SET title='Risk Manager interim',version=version+1
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
		tenantID, entityID, positionID)

	if count, err := maintainer.Maintain(ctx, secondEffective, 10); err != nil || count != 1 {
		t.Fatalf("stale activation count=%d err=%v", count, err)
	}
	var errorCode, title string
	if err := pool.QueryRow(ctx, `
		SELECT r.status,r.activation_attempts,r.activation_error_code,p.title,p.occupant_principal_id::text
		FROM organization_position_revisions r
		JOIN org_positions p ON p.id=r.position_id
		WHERE r.id=$1::uuid`, stale.ID).Scan(&status, &attempts, &errorCode, &title, &currentOccupant); err != nil {
		t.Fatal(err)
	}
	if status != "FAILED" || attempts != 1 || errorCode != "STALE_OR_CONFLICTING_STATE" {
		t.Fatalf("stale activation status=%s attempts=%d code=%s", status, attempts, errorCode)
	}
	if err := pool.QueryRow(ctx, `
		SELECT actor_type,actor_id IS NULL,from_state,to_state
		FROM governance_decisions
		WHERE tenant_id=$1::uuid AND object_type='ORGANIZATION_POSITION_REVISION' AND object_id=$2::uuid
		  AND to_state='FAILED'
		ORDER BY decided_at DESC,id DESC LIMIT 1`,
		tenantID, stale.ID).Scan(&actorType, &actorIsNull, &fromState, &toState); err != nil {
		t.Fatal(err)
	}
	if actorType != "SYSTEM" || !actorIsNull || fromState != "SCHEDULED" || toState != "FAILED" {
		t.Fatalf("failure provenance actor=%s null=%v transition=%s->%s", actorType, actorIsNull, fromState, toState)
	}
	if title != "Risk Manager interim" || currentOccupant != occupantB {
		t.Fatalf("failed activation mutated current position: title=%q occupant=%s", title, currentOccupant)
	}
}
