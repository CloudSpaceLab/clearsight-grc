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
