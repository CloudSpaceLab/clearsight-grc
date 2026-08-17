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
	checkpointOutboxID     = "7daaaaaa-aaaa-7aaa-8aaa-aaaaaaaaaaaa"
)

func TestCheckpointReplayUsesExistingRuntimeLeaseRetryAndInbox(t *testing.T) {
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

	checkpointRepository := sourceaccess.NewPostgresCheckpointRepository(pool)
	runtimeRepository := runtime.NewPostgresRepository(pool)
	service := sourceaccess.NewCheckpointService(checkpointRepository, runtimeRepository)
	checkpointAt := time.Date(2026, 8, 15, 6, 0, 0, 0, time.UTC)
	checkpoint, err := service.Ensure(ctx, checkpointTenantID, checkpointSourceID, checkpointBindingID, 1, checkpointAt)
	if err != nil {
		t.Fatal(err)
	}
	if checkpoint.Position.Kind != "" || checkpoint.Generation != 0 {
		t.Fatalf("unexpected starting checkpoint: %#v", checkpoint)
	}

	// Use an isolated early runtime clock so only this fixture is due even when
	// other integration packages have retained newer outbox history.
	runtimeAt := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO outbox_events(
			id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at
		) VALUES (
			$1::uuid,$2::uuid,'SOURCE_BINDING',$3::uuid,'SourceBindingPollRequested',
			jsonb_build_object('binding_id',$3::text,'binding_version',1),$4,$4,$4
		)`, checkpointOutboxID, checkpointTenantID, checkpointBindingID, runtimeAt); err != nil {
		t.Fatal(err)
	}

	first, err := runtimeRepository.ClaimOutbox(ctx, "source-worker", runtimeAt, time.Minute, 1)
	if err != nil || len(first) != 1 || first[0].ID != checkpointOutboxID || first[0].Attempts != 1 {
		t.Fatalf("first runtime claim=%#v err=%v", first, err)
	}
	terminal, err := runtimeRepository.MarkFailed(ctx, first[0], 3, "SOURCE_UNAVAILABLE", runtimeAt.Add(10*time.Second), runtimeAt.Add(2*time.Minute))
	if err != nil || terminal {
		t.Fatalf("first runtime failure terminal=%v err=%v", terminal, err)
	}
	if claims, err := runtimeRepository.ClaimOutbox(ctx, "source-worker", runtimeAt.Add(time.Minute), time.Minute, 1); err != nil || len(claims) != 0 {
		t.Fatalf("runtime backoff was ignored: claims=%#v err=%v", claims, err)
	}

	second, err := runtimeRepository.ClaimOutbox(ctx, "source-worker", runtimeAt.Add(3*time.Minute), time.Minute, 1)
	if err != nil || len(second) != 1 || second[0].ID != checkpointOutboxID || second[0].Attempts != 2 {
		t.Fatalf("second runtime claim=%#v err=%v", second, err)
	}
	position := sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointCursor, Value: "cursor-100"}
	eventID, err := sourceaccess.CheckpointInboxEventID(checkpoint, "source-pull", position)
	if err != nil {
		t.Fatal(err)
	}
	inserted, err := runtimeRepository.RecordInbox(ctx, checkpointTenantID, "source-pull", eventID, runtimeAt.Add(3*time.Minute+10*time.Second))
	if err != nil || !inserted {
		t.Fatalf("durable processing receipt inserted=%v err=%v", inserted, err)
	}

	// Crash after durable processing but before outbox completion/checkpoint advance.
	// Runtime—not the checkpoint table—owns lease expiry and retry.
	third, err := runtimeRepository.ClaimOutbox(ctx, "source-worker", runtimeAt.Add(5*time.Minute), time.Minute, 1)
	if err != nil || len(third) != 1 || third[0].ID != checkpointOutboxID || third[0].Attempts != 3 {
		t.Fatalf("runtime did not replay expired source work: claims=%#v err=%v", third, err)
	}
	if second[0].LeaseUntil == nil || third[0].LeaseUntil == nil || second[0].LeaseUntil.Equal(*third[0].LeaseUntil) {
		t.Fatalf("runtime lease generation did not change: second=%#v third=%#v", second[0], third[0])
	}
	if err := runtimeRepository.MarkPublished(ctx, second[0], runtimeAt.Add(5*time.Minute+5*time.Second)); err == nil {
		t.Fatal("stale same-worker runtime claim published the newer lease")
	}

	inserted, err = runtimeRepository.RecordInbox(ctx, checkpointTenantID, "source-pull", eventID, runtimeAt.Add(5*time.Minute+5*time.Second))
	if err != nil || inserted {
		t.Fatalf("runtime inbox did not suppress duplicate domain processing: inserted=%v err=%v", inserted, err)
	}
	advanced, err := service.AdvanceAfterInbox(ctx, checkpoint, "source-pull", position, checkpointAt.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if advanced.Position != position || advanced.Generation != 1 {
		t.Fatalf("checkpoint did not advance after durable replay proof: %#v", advanced)
	}
	if err := runtimeRepository.MarkPublished(ctx, third[0], runtimeAt.Add(5*time.Minute+10*time.Second)); err != nil {
		t.Fatalf("current runtime lease could not publish: %v", err)
	}
	if _, err := checkpointRepository.AdvanceBindingCheckpoint(ctx, checkpoint, sourceaccess.CheckpointPosition{Kind: sourceaccess.CheckpointCursor, Value: "cursor-101"}, checkpointAt.Add(20*time.Second)); !errors.Is(err, sourceaccess.ErrCheckpointConflict) {
		t.Fatalf("stale checkpoint generation remained writable: %v", err)
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
		`DELETE FROM outbox_events WHERE tenant_id=$1::uuid`,
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
