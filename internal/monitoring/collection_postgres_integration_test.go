//go:build postgres && postgresintegration

package monitoring

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCollectionPolicyCycleScopeAndLease(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	const (
		tenantID       = "636f6c6c-6563-7469-8f6e-72656e657761"
		programID      = "636f6c6c-6563-7469-8f6e-72656e657762"
		principal      = "636f6c6c-6563-7469-8f6e-72656e657763"
		formID         = "636f6c6c-6563-7469-8f6e-72656e657764"
		checkID        = "636f6c6c-6563-7469-8f6e-72656e657765"
		cycleID        = "636f6c6c-6563-7469-8f6e-72656e657766"
		legacyID       = "636f6c6c-6563-7469-8f6e-72656e657767"
		requestID      = "636f6c6c-6563-7469-8f6e-72656e657768"
		originID       = "636f6c6c-6563-7469-8f6e-72656e657769"
		otherProgramID = "636f6c6c-6563-7469-8f6e-72656e65776a"
		otherCheckID   = "636f6c6c-6563-7469-8f6e-72656e65776b"
		otherCycleID   = "636f6c6c-6563-7469-8f6e-72656e65776c"
	)
	_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id=$1::uuid`, tenantID)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM tenants WHERE id=$1::uuid`, tenantID) })
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'collection-renewal-test','Collection Renewal Test');
		INSERT INTO principals(id,tenant_id,kind,display_name,status) VALUES($2::uuid,$1::uuid,'PERSON','Collection Owner','ACTIVE');
		INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,scope,effective_from,version)
			VALUES($3::uuid,$1::uuid,'COLLECTION','Collection renewal','COMPLIANCE','ACTIVE','Compliance','{}'::jsonb,$4,1);
		INSERT INTO monitoring_form_templates(id,tenant_id,code,name,purpose,fields,status,is_current,effective_from,version,created_by,approved_by,created_at,updated_at)
			VALUES($5::uuid,$1::uuid,'VENDOR','Vendor review','Collect a current response.','[{"id":"answer","label":"Answer","type":"text","required":true}]'::jsonb,'ACTIVE',true,$4,1,$2::uuid,$2::uuid,$4,$4)`,
		tenantID, principal, programID, now.Add(-time.Hour), formID); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresRepository(pool)
	check := MonitoringCheck{
		ID: checkID, TenantID: tenantID, ProgramID: programID, Code: "VENDOR-CHECK", Name: "Vendor review", Claim: "The response remains current.",
		InputKind: InputForm, FormTemplateID: formID, FormTemplateVersion: 1, CollectionPolicy: &CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 3},
		Thresholds: DefaultThresholds(), FreshnessMinutes: 10080, MinimumCoverage: 1, FailureAction: FailureReview,
		Lifecycle: Lifecycle{Status: LifecycleActive, IsCurrent: true, EffectiveFrom: timePtr(now.Add(-time.Hour)), Version: 1, CreatedBy: principal, ApprovedBy: principal, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)},
	}
	storedCheck, err := repo.CreateCheckRevision(ctx, check)
	if err != nil || storedCheck.CollectionPolicy == nil || *storedCheck.CollectionPolicy != *check.CollectionPolicy {
		t.Fatalf("stored check = %#v, err = %v", storedCheck, err)
	}
	legacy := check
	legacy.ID = legacyID
	legacy.Code = "LEGACY-CHECK"
	legacy.CollectionPolicy = nil
	legacy.Lifecycle = Lifecycle{Status: LifecycleDraft, Version: 1, CreatedBy: principal, CreatedAt: now, UpdatedAt: now}
	storedLegacy, err := repo.CreateCheckRevision(ctx, legacy)
	if err != nil || storedLegacy.CollectionPolicy != nil {
		t.Fatalf("legacy check = %#v, err = %v", storedLegacy, err)
	}

	requestInsert := `INSERT INTO capture_requests(
		id,tenant_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,
		known_facts,fields,status,created_by,version,created_at,updated_at,origin_type,origin_id,origin_sequence)
		VALUES($1::uuid,$2::uuid,'PROGRAM',$3::text,'Vendor review','Collect a current response.','You own this response.','INTERNAL','INTERNAL',5,$4,
		'{}'::jsonb,'[{"id":"answer","label":"Answer","type":"text","required":true}]'::jsonb,'READY',$5::uuid,1,$4,$4,'MONITORING_COLLECTION',$6::uuid,1)`
	if _, err := pool.Exec(ctx, requestInsert, requestID, tenantID, programID, now, principal, originID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, requestInsert, cycleID, tenantID, programID, now, principal, originID); err == nil {
		t.Fatal("duplicate request origin was accepted")
	}
	cycle := collectionCycleFixture(cycleID, tenantID, programID, checkID, 1, now.Add(-time.Minute))
	cycle.Recipient.PrincipalID = principal
	if _, err := repo.UpsertCollectionCycle(ctx, cycle); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CollectionCycle(ctx, "wrong-tenant", cycleID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant read error = %v", err)
	}
	first, err := repo.ClaimDueCollectionCycles(ctx, "worker-a", now, time.Minute, 1)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim = %#v, err = %v", first, err)
	}
	second, err := repo.ClaimDueCollectionCycles(ctx, "worker-b", now.Add(2*time.Minute), time.Minute, 1)
	if err != nil || len(second) != 1 || second[0].LeaseToken == first[0].LeaseToken {
		t.Fatalf("second claim = %#v, err = %v", second, err)
	}
	if _, err := repo.CompleteCollectionAction(ctx, first[0], CollectionActionCompletion{State: CycleComplete, At: now.Add(2*time.Minute + time.Second)}); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale lease completion error = %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO programs(id,tenant_id,code,name,program_type,status,owning_function,scope,effective_from,version)
		VALUES($1::uuid,$2::uuid,'OTHER-COLLECTION','Other collection','COMPLIANCE','ACTIVE','Compliance','{}'::jsonb,$3,1)`, otherProgramID, tenantID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	otherCheck := check
	otherCheck.ID = otherCheckID
	otherCheck.ProgramID = otherProgramID
	otherCheck.Code = "OTHER-CHECK"
	if _, err := repo.CreateCheckRevision(ctx, otherCheck); err != nil {
		t.Fatal(err)
	}
	otherCycle := collectionCycleFixture(otherCycleID, tenantID, otherProgramID, otherCheckID, 1, now.Add(-time.Hour))
	otherCycle.Recipient.PrincipalID = principal
	if _, err := repo.UpsertCollectionCycle(ctx, otherCycle); err != nil {
		t.Fatal(err)
	}
	summaries, err := repo.ListCollectionSummaries(ctx, tenantID, programID, 1)
	if err != nil || len(summaries) != 1 || summaries[0].ProgramID != programID || summaries[0].MonitoringCheckID != checkID {
		t.Fatalf("scoped summaries = %#v, err = %v", summaries, err)
	}
}
