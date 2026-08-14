//go:build postgres && postgresintegration

package sourceaccess

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
	sourceAccessFixture = "sourceaccess_postgres_fixture"
	sourceAccessRole    = "sourceaccess_postgres_reader"
	sourceAccessPass    = "sourceaccess-test-password"
)

func TestPostgresSessionIsolationReusableOperationsAndReceipts(t *testing.T) {
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
	_, _ = setup.Exec(ctx, `DROP TABLE IF EXISTS `+sourceAccessFixture)
	_, _ = setup.Exec(ctx, `DROP OWNED BY `+sourceAccessRole)
	_, _ = setup.Exec(ctx, `DROP ROLE IF EXISTS `+sourceAccessRole)
	t.Cleanup(func() {
		_, _ = setup.Exec(context.Background(), `DROP TABLE IF EXISTS `+sourceAccessFixture)
		_, _ = setup.Exec(context.Background(), `DROP OWNED BY `+sourceAccessRole)
		_, _ = setup.Exec(context.Background(), `DROP ROLE IF EXISTS `+sourceAccessRole)
	})
	if _, err := setup.Exec(ctx, `CREATE TABLE `+sourceAccessFixture+` (
		id uuid PRIMARY KEY,
		status text NOT NULL,
		patch_age_days numeric,
		owner_id text,
		active boolean NOT NULL,
		updated_at timestamp NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `INSERT INTO `+sourceAccessFixture+`(id,status,patch_age_days,owner_id,active,updated_at) VALUES
		('81111111-1111-7111-8111-111111111111','ACTIVE',45,'owner-1',true,'2026-08-14 09:50:58.123456'),
		('82222222-2222-7222-8222-222222222222','ACTIVE',NULL,'owner-2',true,'2026-08-14 09:51:58'),
		('83333333-3333-7333-8333-333333333333','DORMANT',45,'owner-3',false,'2026-08-14 09:52:58'),
		('84444444-4444-7444-8444-444444444444','ACTIVE',9007199254740994.125,'owner-4',true,'2026-08-14 09:53:58')`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `CREATE ROLE `+sourceAccessRole+` LOGIN PASSWORD '`+sourceAccessPass+`' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `GRANT USAGE ON SCHEMA public TO `+sourceAccessRole); err != nil {
		t.Fatal(err)
	}
	// DELETE is deliberate: the adapter's read-only transaction remains the
	// final write guard even when an approved source role is over-privileged.
	if _, err := setup.Exec(ctx, `GRANT SELECT,DELETE ON `+sourceAccessFixture+` TO `+sourceAccessRole); err != nil {
		t.Fatal(err)
	}

	adapter := NewPostgresAdapter(PostgresOptions{
		MaxConns:         1,
		StatementTimeout: 150 * time.Millisecond,
		LockTimeout:      100 * time.Millisecond,
		PingTimeout:      time.Second,
	})
	connection := NewPostgresConnection("fixture-read", "fixture-source", "connection-v1", "secret-ref")
	if unsafe, err := adapter.Open(ctx, connection, testSecretResolver{value: databaseURL}); !errors.Is(err, ErrPrivileges) {
		if unsafe != nil {
			_ = unsafe.Close()
		}
		t.Fatalf("superuser source credential must be rejected, got %v", err)
	}

	readerURL := postgresRoleURL(t, databaseURL, sourceAccessRole, sourceAccessPass)
	opened, err := adapter.Open(ctx, connection, testSecretResolver{value: readerURL})
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	session, ok := opened.(*PostgresSession)
	if !ok {
		t.Fatalf("opened session type=%T want *PostgresSession", opened)
	}
	if session.pool.Config().MaxConns != 1 {
		t.Fatalf("source pool max conns=%d want=1", session.pool.Config().MaxConns)
	}
	if !session.Capabilities().Has(CapabilityInspect) || !session.Capabilities().Has(CapabilityPage) || !session.Capabilities().Has(CapabilityLookup) || !session.Capabilities().Has(CapabilityAggregate) || session.Capabilities().Has(CapabilityChanges) {
		t.Fatalf("unexpected PostgreSQL capabilities: %#v", session.Capabilities())
	}

	var readOnly, timezone, statementTimeout, applicationName string
	if err := session.pool.QueryRow(ctx, `SHOW default_transaction_read_only`).Scan(&readOnly); err != nil {
		t.Fatal(err)
	}
	if err := session.pool.QueryRow(ctx, `SHOW TimeZone`).Scan(&timezone); err != nil {
		t.Fatal(err)
	}
	if err := session.pool.QueryRow(ctx, `SHOW statement_timeout`).Scan(&statementTimeout); err != nil {
		t.Fatal(err)
	}
	if err := session.pool.QueryRow(ctx, `SHOW application_name`).Scan(&applicationName); err != nil {
		t.Fatal(err)
	}
	if readOnly != "on" || timezone != "UTC" || statementTimeout == "0" || applicationName != "clearsight-sourceaccess" {
		t.Fatalf("unexpected source session guardrails: read_only=%q timezone=%q statement_timeout=%q application_name=%q", readOnly, timezone, statementTimeout, applicationName)
	}

	tx, err := session.beginReadOnly(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM `+sourceAccessFixture); err == nil {
		_ = tx.Rollback(ctx)
		t.Fatal("write-capable source credential mutated data through read-only source session")
	}
	_ = tx.Rollback(ctx)
	var remaining int
	if err := setup.QueryRow(ctx, `SELECT count(*) FROM `+sourceAccessFixture).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 4 {
		t.Fatalf("read-only session changed source population: remaining=%d", remaining)
	}

	view, err := NewPostgresView(
		"accounts",
		connection.ID,
		"view-v1",
		`SELECT id,status,patch_age_days,owner_id,active,updated_at FROM `+sourceAccessFixture,
		"id",
	)
	if err != nil {
		t.Fatal(err)
	}
	binding := Binding{
		ID:             "account-governance",
		ViewID:         view.ID,
		Version:        "binding-v1",
		Purpose:        "assurance-form-and-gateway-lookup",
		Operations:     []Operation{OperationAggregate, OperationLookup, OperationPage},
		SelectedFields: []string{"id", "status", "patch_age_days", "owner_id", "active", "updated_at"},
		KeyFields:      []string{"id"},
		Limits: ResourceLimits{
			PageRows:      2,
			ResponseBytes: 64 << 10,
			LookupValues:  4,
			Timeout:       150 * time.Millisecond,
		},
	}
	if err := binding.Validate(view); err != nil {
		t.Fatal(err)
	}

	inspected, err := session.Inspect(ctx, view)
	if err != nil {
		t.Fatal(err)
	}
	wantNativeTypes := map[string]string{
		"id": "uuid", "status": "text", "patch_age_days": "numeric", "owner_id": "text", "active": "bool", "updated_at": "timestamp",
	}
	for name, want := range wantNativeTypes {
		field := findNativeField(t, inspected.Fields, name)
		if field.NativeType != want {
			t.Fatalf("field %s native type=%q want=%q", name, field.NativeType, want)
		}
	}
	assertSourceAccessReceipt(t, inspected.Receipt, connection, view, Binding{}, OperationInspect, CompletenessComplete)

	first, err := session.ReadPage(ctx, view, binding, PageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Records) != 2 || first.NextCursor == nil || first.NextCursor.Text != "82222222-2222-7222-8222-222222222222" || first.Receipt.Completeness != CompletenessPartial {
		t.Fatalf("unexpected first page: %#v", first)
	}
	assertSourceAccessReceipt(t, first.Receipt, connection, view, binding, OperationPage, CompletenessPartial)
	second, err := session.ReadPage(ctx, view, binding, PageRequest{After: first.NextCursor})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Records) != 2 || second.NextCursor != nil || second.Receipt.Completeness != CompletenessComplete {
		t.Fatalf("unexpected second page: %#v", second)
	}
	assertSourceAccessReceipt(t, second.Receipt, connection, view, binding, OperationPage, CompletenessComplete)
	if got := second.Records[1]["patch_age_days"]; got.Kind != ScalarNumber || got.Text != "9007199254740994.125" {
		t.Fatalf("exact numeric source value was not preserved: %#v", got)
	}
	if got := first.Records[0]["updated_at"]; got.Kind != ScalarTime || got.Text != "2026-08-14T09:50:58.123456Z" {
		t.Fatalf("timestamp was not canonicalized: %#v", got)
	}

	lookup, err := session.Lookup(ctx, view, binding, LookupRequest{Values: []Scalar{
		StringValue("81111111-1111-7111-8111-111111111111"),
		StringValue("84444444-4444-7444-8444-444444444444"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Records) != 2 || lookup.Records[0]["owner_id"].Text != "owner-1" || lookup.Records[1]["owner_id"].Text != "owner-4" {
		t.Fatalf("unexpected lookup result: %#v", lookup)
	}
	assertSourceAccessReceipt(t, lookup.Receipt, connection, view, binding, OperationLookup, CompletenessComplete)

	// Bank-native field names remain usable. The adapter quotes configured
	// identifiers instead of forcing a universal rename or schema mapping.
	nativeView, err := NewPostgresView("native-account-shape", connection.ID, "view-v1", `SELECT id AS "Account ID",status AS "Account.Status" FROM `+sourceAccessFixture, "Account ID")
	if err != nil {
		t.Fatal(err)
	}
	nativeBinding := Binding{
		ID: "native-account-lookup", ViewID: nativeView.ID, Version: "binding-v1", Purpose: "native-schema-proof",
		Operations: []Operation{OperationLookup}, SelectedFields: []string{"Account ID", "Account.Status"}, KeyFields: []string{"Account ID"}, Limits: DefaultResourceLimits(),
	}
	nativeLookup, err := session.Lookup(ctx, nativeView, nativeBinding, LookupRequest{Values: []Scalar{StringValue("81111111-1111-7111-8111-111111111111")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeLookup.Records) != 1 || nativeLookup.Records[0]["Account.Status"].Text != "ACTIVE" {
		t.Fatalf("native source shape was not preserved: %#v", nativeLookup)
	}

	aggregate, err := session.EvaluatePredicate(ctx, view, binding, PostgresPredicate{
		MatchSQL: `status = $1`, UnknownSQL: `FALSE`, Args: []any{"ACTIVE"},
	}, func(fields []NativeField) error {
		if findNativeField(t, fields, "status").NativeType != "text" {
			return errors.New("status schema changed")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if aggregate.TotalCount != 4 || aggregate.MatchCount != 3 || aggregate.UnknownCount != 0 || aggregate.ClearCount != 1 {
		t.Fatalf("unexpected aggregate: %#v", aggregate)
	}
	assertSourceAccessReceipt(t, aggregate.Receipt, connection, view, binding, OperationAggregate, CompletenessComplete)

	if _, err := session.Lookup(ctx, view, binding, LookupRequest{Values: []Scalar{StringValue("81111111-1111-7111-8111-111111111111"), StringValue("81111111-1111-7111-8111-111111111111")}}); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("duplicate lookup values should fail, got %v", err)
	}
	if _, err := session.Lookup(ctx, view, binding, LookupRequest{Values: []Scalar{{Kind: ScalarNumber, Text: "1"}}}); !errors.Is(err, ErrDefinitionInvalid) {
		t.Fatalf("wrong lookup scalar kind should fail, got %v", err)
	}
	tiny := binding
	tiny.Limits.ResponseBytes = 8
	if _, err := session.Lookup(ctx, view, tiny, LookupRequest{Values: []Scalar{StringValue("81111111-1111-7111-8111-111111111111")}}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("response byte limit should fail, got %v", err)
	}

	duplicateKeyView, err := NewPostgresView("duplicate-accounts", connection.ID, "view-v1", `SELECT id,status FROM `+sourceAccessFixture+` UNION ALL SELECT id,status FROM `+sourceAccessFixture, "id")
	if err != nil {
		t.Fatal(err)
	}
	duplicateKeyBinding := binding
	duplicateKeyBinding.ID = "duplicate-account-lookup"
	duplicateKeyBinding.ViewID = duplicateKeyView.ID
	duplicateKeyBinding.SelectedFields = []string{"id", "status"}
	duplicateKeyBinding.Operations = []Operation{OperationLookup}
	duplicateKeyBinding.Limits.LookupValues = 1
	if _, err := session.Lookup(ctx, duplicateKeyView, duplicateKeyBinding, LookupRequest{Values: []Scalar{StringValue("81111111-1111-7111-8111-111111111111")}}); !errors.Is(err, ErrExecution) {
		t.Fatalf("source violating stable-key uniqueness should fail, got %v", err)
	}

	slowView, err := NewPostgresView("slow-accounts", connection.ID, "view-v1", `SELECT id,status FROM `+sourceAccessFixture+` WHERE pg_sleep(1) IS NULL /* QUERY_SECRET_MARKER */`, "id")
	if err != nil {
		t.Fatal(err)
	}
	slowBinding := binding
	slowBinding.ID = "slow-account-aggregate"
	slowBinding.ViewID = slowView.ID
	slowBinding.SelectedFields = []string{"id", "status"}
	slowBinding.Operations = []Operation{OperationAggregate}
	started := time.Now()
	_, err = session.EvaluatePredicate(ctx, slowView, slowBinding, PostgresPredicate{MatchSQL: "TRUE", UnknownSQL: "FALSE"}, nil)
	if !errors.Is(err, ErrExecution) {
		t.Fatalf("statement timeout should fail source execution, got %v", err)
	}
	if strings.Contains(err.Error(), "QUERY_SECRET_MARKER") || strings.Contains(err.Error(), sourceAccessFixture) {
		t.Fatalf("source execution error leaked query text: %v", err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatalf("source timeout exceeded bounded execution window: %s", time.Since(started))
	}
}

func assertSourceAccessReceipt(t *testing.T, receipt OperationReceipt, connection Connection, view View, binding Binding, operation Operation, completeness Completeness) {
	t.Helper()
	if receipt.SourceID != connection.SourceID || receipt.ConnectionID != connection.ID || receipt.ConnectionVersion != connection.Version || receipt.AdapterKind != AdapterPostgres || receipt.AdapterVersion != PostgresAdapterVersion || receipt.ViewID != view.ID || receipt.ViewVersion != view.Version || receipt.Operation != operation || receipt.Completeness != completeness || receipt.ObservedAt.IsZero() || receipt.DefinitionFingerprint == "" || receipt.SchemaFingerprint == "" {
		t.Fatalf("incomplete or incorrect source receipt: %#v", receipt)
	}
	if binding.ID == "" {
		if receipt.BindingID != "" || receipt.BindingVersion != "" {
			t.Fatalf("inspection receipt unexpectedly contains a binding: %#v", receipt)
		}
		return
	}
	if receipt.BindingID != binding.ID || receipt.BindingVersion != binding.Version {
		t.Fatalf("receipt binding=%s/%s want=%s/%s", receipt.BindingID, receipt.BindingVersion, binding.ID, binding.Version)
	}
}

func findNativeField(t *testing.T, fields []NativeField, name string) NativeField {
	t.Helper()
	for _, field := range fields {
		if field.Name == name {
			return field
		}
	}
	t.Fatalf("field %q not found in %#v", name, fields)
	return NativeField{}
}

func postgresRoleURL(t *testing.T, raw, username, password string) string {
	t.Helper()
	parsed, err := neturl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = neturl.UserPassword(username, password)
	return parsed.String()
}
