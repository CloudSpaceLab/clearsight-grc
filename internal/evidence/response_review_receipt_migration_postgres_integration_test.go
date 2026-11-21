//go:build postgres && postgresintegration

package evidence

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestResponseReviewReceiptMigrationPreservesHistoryAndBlocksRollback(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "review_receipt_" + strings.ReplaceAll(value, "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err = pool.Exec(ctx, `CREATE TABLE capture_response_reviews(id uuid PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000153_response_review_authority_receipt.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000153_response_review_authority_receipt.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO capture_response_reviews(id) VALUES(gen_random_uuid())`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("legacy row prevents safe rollback: %v", err)
	}
	if _, err = pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO capture_response_reviews(id,authority_source_type,authority_source_id,authority_policy_version) VALUES(gen_random_uuid(),'POLICY',gen_random_uuid(),'v1')`); err == nil {
		t.Fatal("partial receipt accepted")
	}
	if _, err = pool.Exec(ctx, `INSERT INTO capture_response_reviews(id,authority_source_type,authority_source_id,authority_rule_id,authority_policy_version) VALUES(gen_random_uuid(),'ASSIGNMENT',gen_random_uuid(),'rule','v1')`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(down)); err == nil {
		t.Fatal("rollback discarded a recorded receipt")
	}
}

func TestResponseReviewRequestKeyMigrationPreservesAndProtectsReceipts(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	value, err := id.NewUUIDv7()
	if err != nil {
		t.Fatal(err)
	}
	schema := "review_keys_" + strings.ReplaceAll(value, "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`)
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err = pool.Exec(ctx, `CREATE TABLE capture_response_reviews(id uuid PRIMARY KEY,tenant_id uuid NOT NULL,legal_entity_id uuid NOT NULL,reviewer_id uuid NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000154_response_review_request_key.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000154_response_review_request_key.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO capture_response_reviews VALUES(gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),gen_random_uuid(),NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("historical null prevented rollback: %v", err)
	}
	if _, err = pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO capture_response_reviews VALUES(gen_random_uuid(),'00000000-0000-4000-8000-000000000001','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000004')`); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO capture_response_reviews SELECT gen_random_uuid(),tenant_id,legal_entity_id,reviewer_id,request_key FROM capture_response_reviews WHERE request_key IS NOT NULL`); err == nil {
		t.Fatal("duplicate scoped key accepted")
	}
	if _, err = pool.Exec(ctx, string(down)); err == nil {
		t.Fatal("rollback discarded a request key")
	}
}
