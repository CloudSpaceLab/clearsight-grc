//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	sourceAccessCorrectnessFixture      = "sourceaccess_correctness_fixture"
	sourceAccessCorrectnessOwnerFixture = "sourceaccess_correctness_owner_fixture"
	sourceAccessCorrectnessRole         = "sourceaccess_correctness_reader"
	sourceAccessCorrectnessPass         = "sourceaccess-correctness-password"
	sourceAccessCorrectnessOwnerRole    = "sourceaccess_correctness_owner"
	sourceAccessCorrectnessOwnerPass    = "sourceaccess-owner-password"
)

func TestPostgresSourceAccessCorrectnessBoundaries(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	setup, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	cleanupPostgresCorrectnessFixtures(ctx, setup)
	t.Cleanup(func() {
		cleanupPostgresCorrectnessFixtures(context.Background(), setup)
		setup.Close()
	})

	if _, err := setup.Exec(ctx, `CREATE TABLE `+sourceAccessCorrectnessFixture+` (
		id bigint PRIMARY KEY,
		effective_date date UNIQUE NOT NULL,
		observed_at timestamp UNIQUE NOT NULL,
		observed_at_tz timestamptz UNIQUE NOT NULL,
		payload text NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `INSERT INTO `+sourceAccessCorrectnessFixture+`(id,effective_date,observed_at,observed_at_tz,payload) VALUES
		(1,'2026-08-14','2026-08-14 09:50:58.123456','2026-08-14 09:50:58.123456+00','small-1'),
		(2,'2026-08-15','2026-08-15 09:50:58.123456','2026-08-15 09:50:58.123456+00','small-2'),
		(3,'2026-08-16','2026-08-16 09:50:58.123456','2026-08-16 09:50:58.123456+00',repeat('x',4096)),
		(4,'2026-08-17','2026-08-17 09:50:58.123456','2026-08-17 09:50:58.123456+00','small-4')`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `CREATE ROLE `+sourceAccessCorrectnessRole+` LOGIN PASSWORD '`+sourceAccessCorrectnessPass+`' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `GRANT USAGE ON SCHEMA public TO `+sourceAccessCorrectnessRole); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `GRANT SELECT ON `+sourceAccessCorrectnessFixture+` TO `+sourceAccessCorrectnessRole); err != nil {
		t.Fatal(err)
	}

	if _, err := setup.Exec(ctx, `CREATE ROLE `+sourceAccessCorrectnessOwnerRole+` LOGIN PASSWORD '`+sourceAccessCorrectnessOwnerPass+`' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `CREATE TABLE `+sourceAccessCorrectnessOwnerFixture+` (id bigint PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `ALTER TABLE `+sourceAccessCorrectnessOwnerFixture+` OWNER TO `+sourceAccessCorrectnessOwnerRole); err != nil {
		t.Fatal(err)
	}

	adapter := NewPostgresAdapter(PostgresOptions{MaxConns: 1, StatementTimeout: 2 * time.Second})
	connection := NewPostgresConnection("correctness-read", "correctness-source", "connection-v1", "secret-ref")
	ownerURL := postgresRoleURL(t, databaseURL, sourceAccessCorrectnessOwnerRole, sourceAccessCorrectnessOwnerPass)
	if ownerSession, err := adapter.Open(ctx, connection, testSecretResolver{value: ownerURL}); !errors.Is(err, ErrPrivileges) {
		if ownerSession != nil {
			_ = ownerSession.Close()
		}
		t.Fatalf("relation-owning source principal should fail, got %v", err)
	}

	readerURL := postgresRoleURL(t, databaseURL, sourceAccessCorrectnessRole, sourceAccessCorrectnessPass)
	opened, err := adapter.Open(ctx, connection, testSecretResolver{value: readerURL})
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	session := opened.(*PostgresSession)

	pageView, err := NewPostgresView("page-budget", connection.ID, "view-v1", `SELECT id,payload FROM `+sourceAccessCorrectnessFixture, "id")
	if err != nil {
		t.Fatal(err)
	}
	pageBinding := Binding{
		ID:             "page-budget",
		ViewID:         pageView.ID,
		Version:        "binding-v1",
		Purpose:        "bounded-page",
		Operations:     []Operation{OperationPage},
		SelectedFields: []string{"id", "payload"},
		KeyFields:      []string{"id"},
		Limits: ResourceLimits{
			PageRows:      2,
			ResponseBytes: 256,
			LookupValues:  4,
			Timeout:       time.Second,
		},
	}
	firstPage, err := session.ReadPage(ctx, pageView, pageBinding, PageRequest{})
	if err != nil {
		t.Fatalf("look-ahead payload must not consume the returned-page byte budget: %v", err)
	}
	if len(firstPage.Records) != 2 || firstPage.NextCursor == nil || firstPage.NextCursor.Kind != ScalarNumber || firstPage.NextCursor.Text != "2" || firstPage.Receipt.Completeness != CompletenessPartial {
		t.Fatalf("unexpected bounded first page: %#v", firstPage)
	}
	if firstPage.Receipt.Bytes > pageBinding.Limits.ResponseBytes {
		t.Fatalf("page receipt bytes=%d exceed binding limit=%d", firstPage.Receipt.Bytes, pageBinding.Limits.ResponseBytes)
	}
	if _, err := session.ReadPage(ctx, pageView, pageBinding, PageRequest{After: firstPage.NextCursor}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("a returned oversized record must fail the page, got %v", err)
	}

	duplicatePageView, err := NewPostgresView("duplicate-page-key", connection.ID, "view-v1", `SELECT id FROM `+sourceAccessCorrectnessFixture+` WHERE id=1 UNION ALL SELECT id FROM `+sourceAccessCorrectnessFixture+` WHERE id=1`, "id")
	if err != nil {
		t.Fatal(err)
	}
	duplicatePageBinding := Binding{
		ID: "duplicate-page-key", ViewID: duplicatePageView.ID, Version: "binding-v1", Purpose: "duplicate-page-key",
		Operations: []Operation{OperationPage}, SelectedFields: []string{"id"}, KeyFields: []string{"id"},
		Limits: ResourceLimits{PageRows: 1, ResponseBytes: 256, LookupValues: 1, Timeout: time.Second},
	}
	if _, err := session.ReadPage(ctx, duplicatePageView, duplicatePageBinding, PageRequest{}); !errors.Is(err, ErrExecution) {
		t.Fatalf("duplicate page key across the look-ahead boundary should fail, got %v", err)
	}

	lookupBinding := pageBinding
	lookupBinding.ID = "numeric-lookup"
	lookupBinding.Operations = []Operation{OperationLookup}
	lookupBinding.Limits.ResponseBytes = 1024
	lookup, err := session.Lookup(ctx, pageView, lookupBinding, LookupRequest{Values: []Scalar{
		{Kind: ScalarNumber, Text: "1"},
		{Kind: ScalarNumber, Text: "4"},
	}})
	if err != nil {
		t.Fatalf("bigint lookup failed: %v", err)
	}
	if len(lookup.Records) != 2 || lookup.Records[0]["id"].Text != "1" || lookup.Records[1]["id"].Text != "4" {
		t.Fatalf("unexpected bigint lookup: %#v", lookup)
	}

	dateView, err := NewPostgresView("date-lookup", connection.ID, "view-v1", `SELECT effective_date,id FROM `+sourceAccessCorrectnessFixture, "effective_date")
	if err != nil {
		t.Fatal(err)
	}
	dateBinding := Binding{
		ID: "date-lookup", ViewID: dateView.ID, Version: "binding-v1", Purpose: "date-lookup",
		Operations: []Operation{OperationLookup}, SelectedFields: []string{"effective_date", "id"}, KeyFields: []string{"effective_date"}, Limits: DefaultResourceLimits(),
	}
	dateLookup, err := session.Lookup(ctx, dateView, dateBinding, LookupRequest{Values: []Scalar{{Kind: ScalarTime, Text: "2026-08-14"}}})
	if err != nil {
		t.Fatalf("date lookup failed: %v", err)
	}
	if len(dateLookup.Records) != 1 || dateLookup.Records[0]["id"].Text != "1" {
		t.Fatalf("unexpected date lookup: %#v", dateLookup)
	}

	timestampView, err := NewPostgresView("timestamp-lookup", connection.ID, "view-v1", `SELECT observed_at,id FROM `+sourceAccessCorrectnessFixture, "observed_at")
	if err != nil {
		t.Fatal(err)
	}
	timestampBinding := Binding{
		ID: "timestamp-lookup", ViewID: timestampView.ID, Version: "binding-v1", Purpose: "timestamp-lookup",
		Operations: []Operation{OperationLookup}, SelectedFields: []string{"observed_at", "id"}, KeyFields: []string{"observed_at"}, Limits: DefaultResourceLimits(),
	}
	timestampLookup, err := session.Lookup(ctx, timestampView, timestampBinding, LookupRequest{Values: []Scalar{{Kind: ScalarTime, Text: "2026-08-14T09:50:58.123456Z"}}})
	if err != nil {
		t.Fatalf("timestamp lookup failed: %v", err)
	}
	if len(timestampLookup.Records) != 1 || timestampLookup.Records[0]["id"].Text != "1" {
		t.Fatalf("unexpected timestamp lookup: %#v", timestampLookup)
	}

	timestamptzView, err := NewPostgresView("timestamptz-lookup", connection.ID, "view-v1", `SELECT observed_at_tz,id FROM `+sourceAccessCorrectnessFixture, "observed_at_tz")
	if err != nil {
		t.Fatal(err)
	}
	timestamptzBinding := Binding{
		ID: "timestamptz-lookup", ViewID: timestamptzView.ID, Version: "binding-v1", Purpose: "timestamptz-lookup",
		Operations: []Operation{OperationLookup}, SelectedFields: []string{"observed_at_tz", "id"}, KeyFields: []string{"observed_at_tz"}, Limits: DefaultResourceLimits(),
	}
	timestamptzLookup, err := session.Lookup(ctx, timestamptzView, timestamptzBinding, LookupRequest{Values: []Scalar{{Kind: ScalarTime, Text: "2026-08-14T09:50:58.123456Z"}}})
	if err != nil {
		t.Fatalf("timestamptz lookup failed: %v", err)
	}
	if len(timestamptzLookup.Records) != 1 || timestamptzLookup.Records[0]["id"].Text != "1" {
		t.Fatalf("unexpected timestamptz lookup: %#v", timestamptzLookup)
	}

	aggregateView, err := NewPostgresView("aggregate-projection", connection.ID, "view-v1", `SELECT id,effective_date,payload FROM `+sourceAccessCorrectnessFixture)
	if err != nil {
		t.Fatal(err)
	}
	aggregateBinding := Binding{
		ID: "aggregate-projection", ViewID: aggregateView.ID, Version: "binding-v1", Purpose: "aggregate-projection",
		Operations: []Operation{OperationAggregate}, SelectedFields: []string{"id"}, Limits: DefaultResourceLimits(),
	}
	aggregate, err := session.EvaluatePredicate(ctx, aggregateView, aggregateBinding, PostgresPredicate{MatchSQL: "TRUE", UnknownSQL: "FALSE"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(aggregate.Fields) != 1 || aggregate.Fields[0].Name != "id" {
		t.Fatalf("aggregate exposed fields outside the binding: %#v", aggregate.Fields)
	}
}

func cleanupPostgresCorrectnessFixtures(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS `+sourceAccessCorrectnessFixture)
	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS `+sourceAccessCorrectnessOwnerFixture)
	_, _ = pool.Exec(ctx, `DROP OWNED BY `+sourceAccessCorrectnessRole)
	_, _ = pool.Exec(ctx, `DROP OWNED BY `+sourceAccessCorrectnessOwnerRole)
	_, _ = pool.Exec(ctx, `DROP ROLE IF EXISTS `+sourceAccessCorrectnessRole)
	_, _ = pool.Exec(ctx, `DROP ROLE IF EXISTS `+sourceAccessCorrectnessOwnerRole)
}
