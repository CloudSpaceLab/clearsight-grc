//go:build postgres && postgresintegration

package main

import (
	"context"
	"testing"
)

func TestSourceEmployeesRepeatSafeAndLimitedPerformerProvisioning(t *testing.T) {
	pool, _, seed := sampleTestSetup(t)
	ctx := context.Background()
	var bindings, scim int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM position_role_bindings),(SELECT count(*) FROM scim_users)`).Scan(&bindings, &scim); err != nil {
		t.Fatal(err)
	}
	first, err := seedSourceEmployees(ctx, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	if first.Count != 17 || first.Created != 17 || len(first.Employees) != 17 {
		t.Fatalf("first receipt: %+v", first)
	}
	second, err := seedSourceEmployees(ctx, pool, seed)
	if err != nil {
		t.Fatal(err)
	}
	if second.Count != 17 || second.Created != 0 {
		t.Fatalf("repeat receipt: %+v", second)
	}
	var afterBindings, afterSCIM, personCount, positionCount int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM position_role_bindings),(SELECT count(*) FROM scim_users),(SELECT count(*) FROM principals WHERE external_ref LIKE 'fidelity-source-samples-v1:person:%'),(SELECT count(*) FROM org_positions WHERE code LIKE 'FIDELITY-SAMPLE-%')`).Scan(&afterBindings, &afterSCIM, &personCount, &positionCount); err != nil {
		t.Fatal(err)
	}
	if bindings+17 != afterBindings || scim != afterSCIM || personCount != 17 || positionCount != 17 {
		t.Fatalf("unexpected directory/access counts: %d %d %d %d", afterBindings, afterSCIM, personCount, positionCount)
	}
	if _, err = pool.Exec(ctx, `UPDATE principals SET display_name='User-edited name' WHERE id=$1::uuid`, first.Employees[16].PrincipalID); err != nil {
		t.Fatal(err)
	}
	if _, err = seedSourceEmployees(ctx, pool, seed); err == nil {
		t.Fatal("expected conflict for changed user record")
	}
	var name string
	if err = pool.QueryRow(ctx, `SELECT display_name FROM principals WHERE id=$1::uuid`, first.Employees[16].PrincipalID).Scan(&name); err != nil || name != "User-edited name" {
		t.Fatalf("user change overwritten: %q %v", name, err)
	}
}

func TestSourceEmployeesRejectWrongScopeAndRollbackConflict(t *testing.T) {
	pool, _, seed := sampleTestSetup(t)
	ctx := context.Background()
	wrong := seed
	wrong.LegalEntityID = "00000000-0000-4000-8000-000000000999"
	if _, err := seedSourceEmployees(ctx, pool, wrong); err == nil {
		t.Fatal("accepted wrong entity")
	}
	if _, err := pool.Exec(ctx, `UPDATE tenants SET slug='different-demo' WHERE id=$1::uuid`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := seedSourceEmployees(ctx, pool, seed); err == nil {
		t.Fatal("accepted wrong tenant slug")
	}
	if _, err := pool.Exec(ctx, `UPDATE tenants SET slug='clearsight-demo' WHERE id=$1::uuid`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	people, err := sourceEmployees()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name,external_ref) VALUES($1::uuid,$2::uuid,'PERSON','Existing user',$3)`, people[16].PrincipalID, seed.TenantID, people[16].ExternalRef); err != nil {
		t.Fatal(err)
	}
	if _, err = seedSourceEmployees(ctx, pool, seed); err == nil {
		t.Fatal("accepted conflicting identity")
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM principals WHERE external_ref LIKE 'fidelity-source-samples-v1:person:%'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("partial inserts committed: %d %v", count, err)
	}
}
