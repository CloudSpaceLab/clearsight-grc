//go:build postgres && postgresintegration

package assurance

import (
	"context"
	"errors"
	neturl "net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sourceExecutorFixture = "assurance_source_executor_fixture"
	sourceExecutorRole    = "assurance_source_executor_reader"
	sourceExecutorPass    = "assurance-source-test-password"
)

func TestPostgresSourceExecutorIsolationAndEvaluation(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	setup, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer setup.Close()
	_, _ = setup.Exec(ctx, `DROP TABLE IF EXISTS `+sourceExecutorFixture)
	_, _ = setup.Exec(ctx, `DROP OWNED BY `+sourceExecutorRole)
	_, _ = setup.Exec(ctx, `DROP ROLE IF EXISTS `+sourceExecutorRole)
	t.Cleanup(func() {
		_, _ = setup.Exec(context.Background(), `DROP TABLE IF EXISTS `+sourceExecutorFixture)
		_, _ = setup.Exec(context.Background(), `DROP OWNED BY `+sourceExecutorRole)
		_, _ = setup.Exec(context.Background(), `DROP ROLE IF EXISTS `+sourceExecutorRole)
	})
	if _, err := setup.Exec(ctx, `CREATE TABLE `+sourceExecutorFixture+` (
		id uuid PRIMARY KEY,
		status text NOT NULL,
		patch_age_days numeric,
		owner_id text
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `INSERT INTO `+sourceExecutorFixture+`(id,status,patch_age_days,owner_id) VALUES
		('81111111-1111-7111-8111-111111111111','ACTIVE',45,'owner-1'),
		('82222222-2222-7222-8222-222222222222','ACTIVE',NULL,'owner-2'),
		('83333333-3333-7333-8333-333333333333','DORMANT',45,'owner-3'),
		('84444444-4444-7444-8444-444444444444','ACTIVE',9007199254740994,'owner-4')`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `CREATE ROLE `+sourceExecutorRole+` LOGIN PASSWORD '`+sourceExecutorPass+`' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `GRANT USAGE ON SCHEMA public TO `+sourceExecutorRole); err != nil {
		t.Fatal(err)
	}
	// DELETE is intentional: the test proves the executor's READ ONLY
	// transaction blocks a mutation that the database role itself could perform.
	if _, err := setup.Exec(ctx, `GRANT SELECT,DELETE ON `+sourceExecutorFixture+` TO `+sourceExecutorRole); err != nil {
		t.Fatal(err)
	}

	if unsafe, err := OpenPostgresSourceExecutor(ctx, "source-fixture", "superuser-ref", testSecretResolver{value: databaseURL}, PostgresSourceOptions{}); !errors.Is(err, ErrSourcePrivileges) {
		if unsafe != nil {
			unsafe.Close()
		}
		t.Fatalf("superuser source credential must be rejected, got %v", err)
	}

	readerURL := postgresTestRoleURL(t, databaseURL, sourceExecutorRole, sourceExecutorPass)
	executor, err := OpenPostgresSourceExecutor(ctx, "source-fixture", "secret-ref", testSecretResolver{value: readerURL}, PostgresSourceOptions{
		MaxConns:         1,
		StatementTimeout: 150 * time.Millisecond,
		LockTimeout:      100 * time.Millisecond,
		PingTimeout:      time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer executor.Close()
	if executor.pool.Config().MaxConns != 1 {
		t.Fatalf("source pool max conns=%d want=1", executor.pool.Config().MaxConns)
	}

	var readOnly, timezone, statementTimeout string
	if err := executor.pool.QueryRow(ctx, `SHOW default_transaction_read_only`).Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if err := executor.pool.QueryRow(ctx, `SHOW TimeZone`).Scan(&timezone); err != nil {
		t.Fatal(err)
	}
	if err := executor.pool.QueryRow(ctx, `SHOW statement_timeout`).Scan(&statementTimeout); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" || timezone != "UTC" || statementTimeout == "0" {
		t.Fatalf("unexpected source session guardrails: read_only=%q timezone=%q statement_timeout=%q", readOnly, timezone, statementTimeout)
	}

	tx, err := executor.beginReadOnly(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM `+sourceExecutorFixture); err == nil {
		_ = tx.Rollback(ctx)
		t.Fatal("write-capable source credential mutated data through read-only executor transaction")
	}
	_ = tx.Rollback(ctx)
	var remaining int
	if err := setup.QueryRow(ctx, `SELECT count(*) FROM `+sourceExecutorFixture).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 4 {
		t.Fatalf("read-only executor changed source population: remaining=%d", remaining)
	}

	population := PopulationDefinition{
		ID:         "accounts",
		Query:      `SELECT id,status,patch_age_days,owner_id FROM ` + sourceExecutorFixture,
		SubjectKey: "id",
	}
	inspected, err := executor.InspectSchema(ctx, population)
	if err != nil {
		t.Fatal(err)
	}
	wantTypes := map[string]LogicalType{"id": TypeString, "status": TypeString, "patch_age_days": TypeNumber, "owner_id": TypeString}
	for name, want := range wantTypes {
		field, exists := inspected.Schema.Field(name)
		if !exists || field.Type != want {
			t.Fatalf("field %s=%#v exists=%v want type=%s", name, field, exists, want)
		}
	}
	if inspected.PopulationFingerprint == "" || inspected.SchemaFingerprint == "" {
		t.Fatalf("missing inspection fingerprints: %#v", inspected)
	}

	condition, err := CompileCondition(inspected.Schema, Condition{Op: OpAnd, Children: []Condition{
		{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
		{Op: OpGT, Field: "patch_age_days", Value: NumberLiteral(30)},
	}}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := executor.Evaluate(ctx, population, condition)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Complete || receipt.TotalCount != 4 || receipt.MatchCount != 1 || receipt.UnknownCount != 2 || receipt.ClearCount != 1 {
		t.Fatalf("unexpected evaluation receipt: %#v", receipt)
	}
	if receipt.PopulationFingerprint != inspected.PopulationFingerprint || receipt.SchemaFingerprint != inspected.SchemaFingerprint {
		t.Fatalf("evaluation did not preserve inspected identities: inspected=%#v receipt=%#v", inspected, receipt)
	}

	changedPopulation := population
	changedPopulation.Query = `SELECT id,status,patch_age_days::text AS patch_age_days,owner_id FROM ` + sourceExecutorFixture
	if _, err := executor.Evaluate(ctx, changedPopulation, condition); !errors.Is(err, ErrSourceSchemaChanged) {
		t.Fatalf("schema change must fail closed before evaluation, got %v", err)
	}

	slowPopulation := population
	slowPopulation.Query = `SELECT id,status,patch_age_days,owner_id FROM ` + sourceExecutorFixture + ` WHERE pg_sleep(1) IS NULL /* QUERY_SECRET_MARKER */`
	started := time.Now()
	_, err = executor.Evaluate(ctx, slowPopulation, condition)
	if !errors.Is(err, ErrSourceExecution) {
		t.Fatalf("statement timeout should fail source execution, got %v", err)
	}
	if strings.Contains(err.Error(), "QUERY_SECRET_MARKER") || strings.Contains(err.Error(), sourceExecutorFixture) {
		t.Fatalf("source execution error leaked query text: %v", err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatalf("source timeout exceeded bounded execution window: %s", time.Since(started))
	}
}

func postgresTestRoleURL(t *testing.T, raw, username, password string) string {
	t.Helper()
	parsed, err := neturl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = neturl.UserPassword(username, password)
	return parsed.String()
}
