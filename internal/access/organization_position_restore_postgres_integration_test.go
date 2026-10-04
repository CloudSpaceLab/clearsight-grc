//go:build postgres && postgresintegration

package access

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationPositionRestorePreservesAssignmentsAndRevalidatesCurrentState(t *testing.T) {
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
		tenantID  = "8a666666-6666-7666-8666-666666666661"
		entityID  = "8a666666-6666-7666-8666-666666666662"
		makerID   = "8a666666-6666-7666-8666-666666666663"
		checkerID = "8a666666-6666-7666-8666-666666666664"
		occupantA = "8a666666-6666-7666-8666-666666666665"
		occupantB = "8a666666-6666-7666-8666-666666666666"
		scopeID   = "8a666666-6666-7666-8666-666666666667"
		programID = "8a666666-6666-7666-8666-666666666668"
		matterID  = "8a666666-6666-7666-8666-666666666669"
		actionID  = "8a666666-6666-7666-8666-666666666670"
	)
	cleanup := func(cleanCtx context.Context) {
		_, _ = pool.Exec(cleanCtx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	}
	cleanup(ctx)
	t.Cleanup(func() { cleanup(context.Background()) })

	now := time.Now().UTC().Truncate(time.Second)
	mustAdminExec(t, ctx, pool, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'position-restore-test','Position Restore Test')`, tenantID)
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

	admin := NewPostgresAdministrator(pool)
	created, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, Operation: OrganizationPositionCreate,
		Code: "RISK_LEAD", Title: "Risk Lead", FunctionName: "Risk", OrganizationScopeID: scopeID,
		OccupantPrincipalID: occupantA, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: created.ID,
		ActorID: checkerID, Rationale: "Initial position reviewed",
	}); err != nil {
		t.Fatal(err)
	}

	updated, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: created.PositionID,
		Operation: OrganizationPositionUpdate, Title: "Senior Risk Lead", FunctionName: "Risk",
		OrganizationScopeID: scopeID, OccupantPrincipalID: occupantB, ExpectedVersion: 1, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: updated.ID,
		ActorID: checkerID, Rationale: "Occupant change reviewed",
	}); err != nil {
		t.Fatal(err)
	}

	mustAdminExec(t, ctx, pool, `
		INSERT INTO programs(
			id,tenant_id,legal_entity_id,organization_scope_id,code,name,program_type,status,owning_function,
			owner_principal_id,jurisdiction,scope,effective_from,created_at,updated_at,version
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,'RISK-PROGRAM','Risk Program','COMPLIANCE','ACTIVE','Risk',
			$5::uuid,'NG','{}'::jsonb,$6,$6,$6,1
		)`, programID, tenantID, entityID, scopeID, occupantB, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO matters(
			id,tenant_id,legal_entity_id,organization_scope_id,reference,matter_type,status,priority,title,summary,
			scope,owner_principal_id,created_at,updated_at,version
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,'RESTORE-1','CONTROL_GAP','ACTION_IN_PROGRESS',3,'Restore impact','Restore impact',
			'{}'::jsonb,$5::uuid,$6,$6,1
		)`, matterID, tenantID, entityID, scopeID, occupantB, now.Add(-time.Hour))
	mustAdminExec(t, ctx, pool, `
		INSERT INTO matter_actions(
			id,tenant_id,matter_id,title,description,owner_principal_id,status,created_at,updated_at,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,'Restore action','Current assigned work',$4::uuid,'IN_PROGRESS',$5,$5,1)`,
		actionID, tenantID, matterID, occupantB, now.Add(-time.Hour))

	restore, err := admin.RestoreOrganizationPosition(ctx, RestoreOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: created.ID, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if restore.Status != "PENDING" || restore.Operation != OrganizationPositionUpdate ||
		restore.RestoredFromRevisionID != created.ID || restore.Proposed != created.Proposed {
		t.Fatalf("restore proposal = %#v", restore)
	}
	if restore.Impact.ActiveProgramsOwned != 1 || restore.Impact.OpenMattersOwned != 1 || restore.Impact.OpenActionsOwned != 1 {
		t.Fatalf("restore impact = %#v", restore.Impact)
	}

	var title, occupant string
	var version int64
	if err := pool.QueryRow(ctx, `
		SELECT title,occupant_principal_id::text,version
		FROM org_positions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
		tenantID, entityID, created.PositionID).Scan(&title, &occupant, &version); err != nil {
		t.Fatal(err)
	}
	if title != "Senior Risk Lead" || occupant != occupantB || version != 2 {
		t.Fatalf("position changed before approval title=%q occupant=%q version=%d", title, occupant, version)
	}
	assertRestoreWorkOwners(t, ctx, pool, tenantID, programID, matterID, actionID, occupantB)

	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: restore.ID,
		ActorID: makerID, Rationale: "self approval",
	}); !errors.Is(err, ErrAdminMakerChecker) {
		t.Fatalf("maker approved restore: %v", err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: restore.ID,
		ActorID: checkerID, Rationale: "Historical state reviewed against current hierarchy",
	}); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT title,occupant_principal_id::text,version
		FROM org_positions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
		tenantID, entityID, created.PositionID).Scan(&title, &occupant, &version); err != nil {
		t.Fatal(err)
	}
	if title != "Risk Lead" || occupant != occupantA || version != 3 {
		t.Fatalf("restored position title=%q occupant=%q version=%d", title, occupant, version)
	}
	assertRestoreWorkOwners(t, ctx, pool, tenantID, programID, matterID, actionID, occupantB)

	var restoredFrom string
	if err := pool.QueryRow(ctx, `
		SELECT restored_from_revision_id::text
		FROM organization_position_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid`,
		tenantID, entityID, restore.ID).Scan(&restoredFrom); err != nil {
		t.Fatal(err)
	}
	if restoredFrom != created.ID {
		t.Fatalf("restore source=%q want=%q", restoredFrom, created.ID)
	}
	if _, err := admin.RestoreOrganizationPosition(ctx, RestoreOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: created.ID, ActorID: makerID,
	}); !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("no-op restore error=%v", err)
	}

	backToBob, err := admin.ProposeOrganizationPosition(ctx, ProposeOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, PositionID: created.PositionID,
		Operation: OrganizationPositionUpdate, Title: "Senior Risk Lead", FunctionName: "Risk",
		OrganizationScopeID: scopeID, OccupantPrincipalID: occupantB, ExpectedVersion: 3, ActorID: makerID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := admin.ApproveOrganizationPosition(ctx, DecideOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: backToBob.ID,
		ActorID: checkerID, Rationale: "Current occupant restored",
	}); err != nil {
		t.Fatal(err)
	}
	mustAdminExec(t, ctx, pool, `
		UPDATE principals SET valid_until=$3
		WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, occupantA, now.Add(-time.Minute))
	if _, err := admin.RestoreOrganizationPosition(ctx, RestoreOrganizationPositionInput{
		TenantID: tenantID, LegalEntityID: entityID, RevisionID: created.ID, ActorID: makerID,
	}); !errors.Is(err, ErrAdminInvalid) {
		t.Fatalf("inactive historical occupant restore error=%v", err)
	}

	overview, err := admin.Overview(ctx, tenantID, entityID, 50)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, item := range overview.OrganizationPositionHistory {
		if item.ID == restore.ID && item.RestoredFromRevisionID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("restore history missing: %#v", overview.OrganizationPositionHistory)
	}
}

func assertRestoreWorkOwners(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tenantID, programID, matterID, actionID, want string) {
	t.Helper()
	var programOwner, matterOwner, actionOwner string
	if err := pool.QueryRow(ctx, `
		SELECT owner_principal_id::text FROM programs
		WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, programID).Scan(&programOwner); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT owner_principal_id::text FROM matters
		WHERE tenant_id=$1::uuid AND id=$2::uuid`, tenantID, matterID).Scan(&matterOwner); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT owner_principal_id::text FROM matter_actions
		WHERE tenant_id=$1::uuid AND matter_id=$2::uuid AND id=$3::uuid`, tenantID, matterID, actionID).Scan(&actionOwner); err != nil {
		t.Fatal(err)
	}
	if programOwner != want || matterOwner != want || actionOwner != want {
		t.Fatalf("work owners program=%q matter=%q action=%q want=%q", programOwner, matterOwner, actionOwner, want)
	}
}
