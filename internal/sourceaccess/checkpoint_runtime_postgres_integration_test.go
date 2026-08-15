//go:build postgres && postgresintegration

package sourceaccess_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	checkpointTenantID     = "7d111111-1111-7111-8111-111111111111"
	checkpointActorID      = "7d222222-2222-7222-8222-222222222222"
	checkpointSourceID     = "7d333333-3333-7333-8333-333333333333"
	checkpointConnectionID = "7d444444-4444-7444-8444-444444444444"
	checkpointViewID       = "7d555555-5555-7555-8555-555555555555"
	checkpointBindingID    = "7d666666-6666-7666-8666-666666666666"
)

func TestCheckpointReplayUsesExistingRuntimeInboxReceipt(t *testing.T) {
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
	cleanupCheckpointFixture(ctx, pool)
	t.Cleanup(func() { cleanupCheckpointFixture(context.Background(), pool) })
	seedCheckpointFixture(t, ctx, pool)

	repository := sourceaccess.NewPostgresCheckpointRepository(pool)
	runtimeRepository := runtime.NewPostgresRepository(pool)
	service := sourceaccess.NewCheckpointService(repository, runtimeRepository)
	now := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	checkpoint, err := service.Ensure(ctx, checkpointTenantID, checkpointSourceID, checkpointBindingID, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Position.Kind != "" {
		t.Fatalf("unexpected starting checkpoint: %#v", checkpoint)
	}
	first, err := service.Claim(ctx, "worker-a", now, time.Minute, 10)
	if err != nil || len(first) != 1 {
		t.Fatalf("first claim=%#v err=%v", first, err)
	}

	inserted, err := runtimeRepository.RecordInbox(ctx, checkpointTenantID, "source-pull", "event-100", now.Add(10*time.Second))
	if err != nil || !inserted {
		t.Fatalf("durable inbox receipt inserted=%v err=%v", inserted, err)
	}
	// Simulate a crash after durable processing but before checkpoint advancement.
	// The same worker identity is deliberately reused to prove the lease generation,
	// not just the worker name, fences the abandoned execution.
	replayed, err := service.Claim(ctx, "worker-a", now.Add(2*time.Minute), time.Minute, 10)
	if err != nil || len(replayed) != 1 {
		t.Fatalf("expired checkpoint was not replayed: claims=%#v err=%v", replayed, err)
	}
	if replayed[0].Position != first[0].Position || replayed[0].Attempts != 2 {
		t.Fatalf("replay skipped source position: first=%#v replay=%#v", first[0], replayed[0])
	}
	if first[0].LeaseUntil == nil || replayed[0].LeaseUntil == nil || first[0].LeaseUntil.Equal(*replayed[0].LeaseUntil) {
		t.Fatalf("checkpoint lease generation did not change: first=%#v replay=%#v", first[0], replayed[0])
	}
	staleAt := now.Add(2*time.Minute + 5*time.Second)
	if _, err := repository.AdvanceBindingCheckpoint(ctx, first[0], sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointCursor, Value: "stale-cursor"}, staleAt, now.Add(3*time.Minute)); !errors.Is(err, sourceaccess.ErrCheckpointClaimLost) {
		t.Fatalf("stale same-worker claim advanced the newer lease: %v", err)
	}
	inserted, err = runtimeRepository.RecordInbox(ctx, checkpointTenantID, "source-pull", "event-100", now.Add(2*time.Minute))
	if err != nil || inserted {
		t.Fatalf("duplicate domain processing was not suppressed: inserted=%v err=%v", inserted, err)
	}

	position := sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointCursor, Value: "cursor-100"}
	advanced, err := service.AdvanceAfterInbox(ctx, replayed[0], "source-pull", "event-100", position, now.Add(2*time.Minute+10*time.Second), now.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if advanced.Position != position || advanced.Attempts != 0 {
		t.Fatalf("checkpoint did not advance after durable proof: %#v", advanced)
	}
}

func seedCheckpointFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	now := time.Date(2026, 8, 15, 5, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,'source-checkpoint-test','Source checkpoint test')`, checkpointTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,external_ref,display_name) VALUES($1::uuid,$2::uuid,'PERSON','source-checkpoint-actor','Source checkpoint actor')`, checkpointActorID, checkpointTenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version)
		VALUES($1::uuid,$2::uuid,'CHECKPOINT-SOURCE','Checkpoint source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,15,'UNKNOWN','ACTIVE',1)`, checkpointSourceID, checkpointTenantID, checkpointActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO source_connections(
			revision_id,connection_id,tenant_id,source_id,code,name,adapter_kind,adapter_version,secret_ref,definition,
			declared_capabilities,verified_capabilities,owner_principal_id,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			'7d777777-7777-7777-8777-777777777777'::uuid,$1::uuid,$2::uuid,$3::uuid,'CHECKPOINT_DB','Checkpoint DB','POSTGRES',$4,
			'vault://checkpoint/reader','{}'::jsonb,'["PAGE"]'::jsonb,'["PAGE"]'::jsonb,$5::uuid,'ACTIVE',true,$6,1,$5::uuid,$6,$6
		)`, checkpointConnectionID, checkpointTenantID, checkpointSourceID, sourceaccess.PostgresAdapterVersion, checkpointActorID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO source_views(
			revision_id,view_id,tenant_id,source_id,connection_id,connection_version,code,name,definition,output_kind,
			stable_keys,native_schema,schema_fingerprint,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			'7d888888-8888-7888-8888-888888888888'::uuid,$1::uuid,$2::uuid,$3::uuid,$4::uuid,1,'ACCOUNTS','Accounts',
			'{"query":"SELECT account_id FROM accounts"}'::jsonb,'RECORDS','["account_id"]'::jsonb,
			'[{"name":"account_id","native_type":"uuid","nullable":false}]'::jsonb,$5,'ACTIVE',true,$6,1,$7::uuid,$6,$6
		)`, checkpointViewID, checkpointTenantID, checkpointSourceID, checkpointConnectionID, strings.Repeat("a", 64), now, checkpointActorID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO source_bindings(
			revision_id,binding_id,tenant_id,source_id,view_id,view_version,code,name,purpose,operations,selected_fields,key_fields,
			page_rows,response_bytes,lookup_values,timeout_ms,mapping,parameter_schema,output_schema,required_freshness_minutes,completeness,
			sensitivity_handling,status,is_current,effective_from,version,created_by,created_at,updated_at
		) VALUES (
			'7d999999-9999-7999-8999-999999999999'::uuid,$1::uuid,$2::uuid,$3::uuid,$4::uuid,1,'ACCOUNT_PAGE','Account page','assurance',
			'["PAGE"]'::jsonb,'["account_id"]'::jsonb,'["account_id"]'::jsonb,25,65536,10,2000,
			'{}'::jsonb,'{}'::jsonb,'{}'::jsonb,15,'REQUIRE_COMPLETE','{}'::jsonb,'ACTIVE',true,$5,1,$6::uuid,$5,$5
		)`, checkpointBindingID, checkpointTenantID, checkpointSourceID, checkpointViewID, now, checkpointActorID); err != nil {
		t.Fatal(err)
	}
}

func cleanupCheckpointFixture(ctx context.Context, pool *pgxpool.Pool) {
	for _, statement := range []string{
		`DELETE FROM inbox_receipts WHERE tenant_id=$1::uuid`,
		`DELETE FROM source_binding_checkpoints WHERE tenant_id=$1::uuid`,
		`DELETE FROM source_observations WHERE tenant_id=$1::uuid`,
		`DELETE FROM source_bindings WHERE tenant_id=$1::uuid`,
		`DELETE FROM source_views WHERE tenant_id=$1::uuid`,
		`DELETE FROM source_connections WHERE tenant_id=$1::uuid`,
		`DELETE FROM evidence_sources WHERE tenant_id=$1::uuid`,
		`DELETE FROM principals WHERE tenant_id=$1::uuid`,
		`DELETE FROM tenants WHERE id=$1::uuid`,
	} {
		_, _ = pool.Exec(ctx, statement, checkpointTenantID)
	}
}
