//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

func TestPostgresAssignedDistributionReadFiltersBeforePagination(t *testing.T) {
	pool, ctx := distributionTestPool(t)
	tenant, entity, owner, recipient, other, form, program := mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t), mustResponseWorkspaceID(t)
	now := time.Now().UTC()
	setupDistributionFixture(t, ctx, pool, tenant, entity, owner, recipient, form, now)
	t.Cleanup(func() { cleanupDistributionTenant(context.Background(), pool, tenant) })
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,status,valid_from) VALUES($1::uuid,$2::uuid,'PERSON','Other','ACTIVE',$3)`, other, tenant, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,legal_entity_id,code,name,program_type,status,owning_function,jurisdiction,scope,effective_from) VALUES($1::uuid,$2::uuid,$3::uuid,'ASSIGNED','Assigned test','COMPLIANCE','ACTIVE','Risk','NG','{"access":"INTERNAL"}'::jsonb,$4)`, program, tenant, entity, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM programs WHERE id=$1::uuid`, program) })
	store := NewPostgresDistributionStore(NewPostgresRepository(pool), postgresTestRecipientProtector{})
	store.now = func() time.Time { return now }
	makeDist := func(principal string) DistributionBundle {
		v, e := store.CreateDistribution(ctx, CreateDistributionInput{TenantID: tenant, LegalEntityID: entity, FormTemplateID: form, FormTemplateVersion: 1, SubjectType: "VENDOR", SubjectID: program, Title: "Review", Purpose: "Check controls", AccessPolicy: AccessDirectEmailOTP, EstimatedMinutes: 8, Deadline: now.Add(72 * time.Hour), RouteExpiresAt: now.Add(48 * time.Hour), CreatedBy: owner, Recipients: []DistributionRecipientInput{{Role: RecipientTo, Type: RecipientInternalPrincipal, PrincipalID: principal}}})
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	visible, hidden := makeDist(recipient), makeDist(other)
	if _, err := pool.Exec(ctx, `UPDATE capture_form_distributions SET subject_type='PROGRAM',status='OPEN',updated_at=CASE WHEN id=$2::uuid THEN $4::timestamptz ELSE $5::timestamptz END WHERE tenant_id=$1::uuid AND id IN ($2::uuid,$3::uuid); UPDATE capture_requests SET subject_type='PROGRAM' WHERE tenant_id=$1::uuid AND distribution_id IN ($2::uuid,$3::uuid)`, pgx.QueryExecModeSimpleProtocol, tenant, hidden.Distribution.ID, visible.Distribution.ID, now.Add(2*time.Second), now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	actor := identity.Actor{TenantID: tenant, LegalEntityID: entity, PrincipalID: recipient, Kind: "PERSON", ExpiresAt: now.Add(time.Hour)}
	actorCtx := identity.WithActor(ctx, actor)
	rows, err := store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if err != nil || len(rows) != 1 || rows[0].ID != visible.Distribution.ID {
		t.Fatalf("scoped first page: %+v %v", rows, err)
	}
	collision := mustResponseWorkspaceID(t)
	if _, e := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Ambiguous scope')`, collision, tenant); e != nil {
		t.Fatal(e)
	}
	ambiguous, e := store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if e != nil || len(ambiguous) != 0 {
		t.Fatalf("ambiguous tenant: %+v %v", ambiguous, e)
	}
	if _, e = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, collision); e != nil {
		t.Fatal(e)
	}
	ownerActor := actor
	ownerActor.PrincipalID = owner
	ownerRows, e := store.listAssignedDistributions(identity.WithActor(ctx, ownerActor), DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if e != nil || len(ownerRows) != 0 {
		t.Fatalf("creator without assignment: %+v %v", ownerRows, e)
	}
	otherActor := actor
	otherActor.PrincipalID = other
	otherRows, e := store.listAssignedDistributions(identity.WithActor(ctx, otherActor), DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if e != nil || len(otherRows) != 1 || otherRows[0].ID != hidden.Distribution.ID {
		t.Fatalf("other recipient: %+v %v", otherRows, e)
	}
	if _, e = store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: mustResponseWorkspaceID(t), LegalEntityID: entity, Limit: 1}); e == nil {
		t.Fatal("mismatched tenant accepted")
	}
	if _, e = pool.Exec(ctx, `UPDATE principals SET status='INACTIVE' WHERE id=$1::uuid`, recipient); e != nil {
		t.Fatal(e)
	}
	inactive, e := store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if e != nil || len(inactive) != 0 {
		t.Fatalf("inactive recipient: %+v %v", inactive, e)
	}
	if _, e = pool.Exec(ctx, `UPDATE principals SET status='ACTIVE' WHERE id=$1::uuid`, recipient); e != nil {
		t.Fatal(e)
	}
	matter := mustResponseWorkspaceID(t)
	if _, e = pool.Exec(ctx, `INSERT INTO matters(id,tenant_id,legal_entity_id,reference,matter_type,priority,title,summary,scope) VALUES($1::uuid,$2::uuid,$3::uuid,'ASSIGNED','CONTROL_GAP',3,'Assigned matter','Review','{"access":"INTERNAL"}'::jsonb)`, matter, tenant, entity); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM matters WHERE id=$1::uuid`, matter) })
	if _, e = pool.Exec(ctx, `UPDATE capture_form_distributions SET subject_type='MATTER',subject_id=$2::uuid WHERE id=$1::uuid; UPDATE capture_requests SET subject_type='MATTER',subject_id=$2 WHERE distribution_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, visible.Distribution.ID, matter); e != nil {
		t.Fatal(e)
	}
	matterRows, e := store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if e != nil || len(matterRows) != 1 || matterRows[0].ID != visible.Distribution.ID {
		t.Fatalf("visible Matter: %+v %v", matterRows, e)
	}
	if _, e = pool.Exec(ctx, `UPDATE matters SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, matter); e != nil {
		t.Fatal(e)
	}
	matterRows, e = store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if e != nil || len(matterRows) != 0 {
		t.Fatalf("restricted Matter: %+v %v", matterRows, e)
	}
	if _, e = pool.Exec(ctx, `UPDATE capture_form_distributions SET subject_type='PROGRAM',subject_id=$2::uuid WHERE id=$1::uuid; UPDATE capture_requests SET subject_type='PROGRAM',subject_id=$2 WHERE distribution_id=$1::uuid`, pgx.QueryExecModeSimpleProtocol, visible.Distribution.ID, program); e != nil {
		t.Fatal(e)
	}
	for _, scenario := range []struct{ name, mutation, restore string }{
		{"draft", `UPDATE capture_form_distributions SET status='DRAFT' WHERE id=$1::uuid`, `UPDATE capture_form_distributions SET status='OPEN' WHERE id=$1::uuid`},
		{"request subject mismatch", `UPDATE capture_requests SET subject_id='other' WHERE distribution_id=$1::uuid`, `UPDATE capture_requests SET subject_id=$2 WHERE distribution_id=$1::uuid`},
		{"third-party origin", `UPDATE capture_requests SET origin_type='THIRD_PARTY_WORK',origin_id='protected',origin_version=1 WHERE distribution_id=$1::uuid`, `UPDATE capture_requests SET origin_type=NULL,origin_id=NULL,origin_version=NULL WHERE distribution_id=$1::uuid`},
		{"unknown origin", `UPDATE capture_requests SET origin_type='UNKNOWN_WORK',origin_id='protected',origin_version=1 WHERE distribution_id=$1::uuid`, `UPDATE capture_requests SET origin_type=NULL,origin_id=NULL,origin_version=NULL WHERE distribution_id=$1::uuid`},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			if _, e := pool.Exec(ctx, scenario.mutation, visible.Distribution.ID); e != nil {
				t.Fatal(e)
			}
			got, e := store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
			if e != nil || len(got) != 0 {
				t.Fatalf("%s: %+v %v", scenario.name, got, e)
			}
			if scenario.name == "request subject mismatch" {
				_, e = pool.Exec(ctx, scenario.restore, visible.Distribution.ID, program)
			} else {
				_, e = pool.Exec(ctx, scenario.restore, visible.Distribution.ID)
			}
			if e != nil {
				t.Fatal(e)
			}
		})
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_distribution_recipients SET state='REVOKED' WHERE distribution_id=$1::uuid`, visible.Distribution.ID); err != nil {
		t.Fatal(err)
	}
	rows, err = store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if err != nil || len(rows) != 0 {
		t.Fatalf("revoked assignment: %+v %v", rows, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE capture_distribution_recipients SET state='PENDING' WHERE distribution_id=$1::uuid`, visible.Distribution.ID); err != nil {
		t.Fatal(err)
	}
	expired := actor
	expired.ExpiresAt = now.Add(-time.Second)
	if _, err = store.listAssignedDistributions(identity.WithActor(ctx, expired), DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1}); err == nil {
		t.Fatal("expired actor was accepted")
	}
	if _, err = pool.Exec(ctx, `UPDATE legal_entities SET valid_until=clock_timestamp()-interval '1 second' WHERE id=$1::uuid`, entity); err != nil {
		t.Fatal(err)
	}
	rows, err = store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if err != nil || len(rows) != 0 {
		t.Fatalf("expired entity: %+v %v", rows, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE legal_entities SET valid_until=NULL WHERE id=$1::uuid`, entity); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `UPDATE programs SET scope='{"access":"RESTRICTED","allowed_principal_ids":[]}'::jsonb WHERE id=$1::uuid`, program); err != nil {
		t.Fatal(err)
	}
	rows, err = store.listAssignedDistributions(actorCtx, DistributionListQuery{TenantID: tenant, LegalEntityID: entity, Limit: 1})
	if err != nil || len(rows) != 0 {
		t.Fatalf("revoked subject scope: %+v %v", rows, err)
	}
}
