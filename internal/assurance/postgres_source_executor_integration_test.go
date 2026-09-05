//go:build postgres && postgresintegration

package assurance

import (
	"context"
	"errors"
	neturl "net/url"
	"os"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	assuranceSourceFixture = "assurance_source_binding_fixture"
	assuranceSourceRole    = "assurance_source_binding_reader"
	assuranceSourcePass    = "assurance-source-binding-password"
)

func TestAssuranceConsumesReusableSourceBindingWithLegacyParity(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	setup, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(setup.Close)
	_, _ = setup.Exec(ctx, `DROP TABLE IF EXISTS `+assuranceSourceFixture)
	_, _ = setup.Exec(ctx, `DROP OWNED BY `+assuranceSourceRole)
	_, _ = setup.Exec(ctx, `DROP ROLE IF EXISTS `+assuranceSourceRole)
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if _, err := setup.Exec(cleanupCtx, `DROP TABLE IF EXISTS `+assuranceSourceFixture); err != nil {
			t.Errorf("drop assurance source fixture: %v", err)
		}
		if _, err := setup.Exec(cleanupCtx, `DROP OWNED BY `+assuranceSourceRole); err != nil {
			t.Errorf("drop assurance source role ownership: %v", err)
		}
		if _, err := setup.Exec(cleanupCtx, `DROP ROLE IF EXISTS `+assuranceSourceRole); err != nil {
			t.Errorf("drop assurance source role: %v", err)
		}
	})
	if _, err := setup.Exec(ctx, `CREATE TABLE `+assuranceSourceFixture+` (
		id uuid PRIMARY KEY,
		status text NOT NULL,
		patch_age_days numeric,
		owner_id text
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `INSERT INTO `+assuranceSourceFixture+`(id,status,patch_age_days,owner_id) VALUES
		('81111111-1111-7111-8111-111111111111','ACTIVE',45,'owner-1'),
		('82222222-2222-7222-8222-222222222222','ACTIVE',NULL,'owner-2'),
		('83333333-3333-7333-8333-333333333333','DORMANT',45,'owner-3'),
		('84444444-4444-7444-8444-444444444444','ACTIVE',9007199254740994,'owner-4')`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `CREATE ROLE `+assuranceSourceRole+` LOGIN PASSWORD '`+assuranceSourcePass+`' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `GRANT USAGE ON SCHEMA public TO `+assuranceSourceRole); err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Exec(ctx, `GRANT SELECT ON `+assuranceSourceFixture+` TO `+assuranceSourceRole); err != nil {
		t.Fatal(err)
	}

	connection := sourceaccess.NewPostgresConnection("fixture-read", "source-fixture", "connection-v1", "secret-ref")
	readerURL := assurancePostgresRoleURL(t, databaseURL, assuranceSourceRole, assuranceSourcePass)
	session, err := sourceaccess.NewPostgresAdapter(sourceaccess.PostgresOptions{MaxConns: 1}).Open(ctx, connection, testSecretResolver{value: readerURL})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	executor, err := NewPostgresSourceExecutorWithSession("source-fixture", session)
	if err != nil {
		t.Fatal(err)
	}

	query := `SELECT id,status,patch_age_days,owner_id FROM ` + assuranceSourceFixture
	view, err := sourceaccess.NewPostgresView("accounts", connection.ID, "view-v1", query, "id")
	if err != nil {
		t.Fatal(err)
	}
	binding := sourceaccess.Binding{
		ID:             "account-governance",
		ViewID:         view.ID,
		Version:        "binding-v1",
		Purpose:        "assurance-and-form-validation",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationAggregate, sourceaccess.OperationLookup},
		SelectedFields: []string{"id", "status", "patch_age_days", "owner_id"},
		KeyFields:      []string{"id"},
		Limits:         sourceaccess.DefaultResourceLimits(),
	}

	inspected, err := executor.InspectBinding(ctx, view, binding)
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
	if inspected.PopulationID != binding.ID || inspected.PopulationFingerprint == "" || inspected.SchemaFingerprint == "" {
		t.Fatalf("binding inspection lost reusable identity: %#v", inspected)
	}

	condition, err := CompileCondition(inspected.Schema, Condition{Op: OpAnd, Children: []Condition{
		{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
		{Op: OpGT, Field: "patch_age_days", Value: NumberLiteral(30)},
	}}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	bindingReceipt, err := executor.EvaluateBinding(ctx, view, binding, condition)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceReceiptCounts(t, bindingReceipt)
	if bindingReceipt.PopulationID != binding.ID || bindingReceipt.PopulationFingerprint != inspected.PopulationFingerprint || bindingReceipt.SchemaFingerprint != inspected.SchemaFingerprint {
		t.Fatalf("binding evaluation did not preserve inspected identities: inspected=%#v receipt=%#v", inspected, bindingReceipt)
	}

	lookupReader, ok := session.(sourceaccess.LookupReader)
	if !ok {
		t.Fatalf("shared session type=%T lacks lookup capability", session)
	}
	lookup, err := lookupReader.Lookup(ctx, view, binding, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{
		sourceaccess.StringValue("81111111-1111-7111-8111-111111111111"),
		sourceaccess.StringValue("84444444-4444-7444-8444-444444444444"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Records) != 2 || lookup.Receipt.BindingID != binding.ID || lookup.Receipt.DefinitionFingerprint != bindingReceipt.PopulationFingerprint {
		t.Fatalf("same binding did not drive the non-assurance lookup: %#v", lookup)
	}

	// The compatibility PopulationDefinition path still produces the same
	// assurance result while compiling its query into a transient source View.
	population := PopulationDefinition{ID: "accounts", Query: query, SubjectKey: "id"}
	legacyInspected, err := executor.InspectSchema(ctx, population)
	if err != nil {
		t.Fatal(err)
	}
	legacyCondition, err := CompileCondition(legacyInspected.Schema, Condition{Op: OpAnd, Children: []Condition{
		{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
		{Op: OpGT, Field: "patch_age_days", Value: NumberLiteral(30)},
	}}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	legacyReceipt, err := executor.Evaluate(ctx, population, legacyCondition)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceReceiptCounts(t, legacyReceipt)
	if legacyReceipt.PopulationFingerprint != legacyInspected.PopulationFingerprint || legacyReceipt.SchemaFingerprint != legacyInspected.SchemaFingerprint {
		t.Fatalf("legacy compatibility identities changed: inspected=%#v receipt=%#v", legacyInspected, legacyReceipt)
	}

	unrelatedView, err := sourceaccess.NewPostgresView("accounts", connection.ID, "view-v2", `SELECT id,status,patch_age_days,length(owner_id)::numeric AS owner_id FROM `+assuranceSourceFixture, "id")
	if err != nil {
		t.Fatal(err)
	}
	unrelatedReceipt, err := executor.EvaluateBinding(ctx, unrelatedView, binding, condition)
	if err != nil {
		t.Fatalf("unrelated projected-column type change should remain evaluable: %v", err)
	}
	assertSourceReceiptCounts(t, unrelatedReceipt)
	if unrelatedReceipt.SchemaFingerprint == inspected.SchemaFingerprint {
		t.Fatal("complete schema fingerprint did not record unrelated projected-column change")
	}

	dependencyView, err := sourceaccess.NewPostgresView("accounts", connection.ID, "view-v3", `SELECT id,status,patch_age_days::text AS patch_age_days,owner_id FROM `+assuranceSourceFixture, "id")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executor.EvaluateBinding(ctx, dependencyView, binding, condition); !errors.Is(err, ErrSourceSchemaChanged) {
		t.Fatalf("condition dependency schema change must fail closed before evaluation, got %v", err)
	}

	subjectView, err := sourceaccess.NewPostgresView("accounts", connection.ID, "view-v4", `SELECT length(id::text)::numeric AS id,status,patch_age_days,owner_id FROM `+assuranceSourceFixture, "id")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := executor.EvaluateBinding(ctx, subjectView, binding, condition); !errors.Is(err, ErrSourceSchemaChanged) {
		t.Fatalf("subject-key schema change must fail closed before evaluation, got %v", err)
	}

	missingConditionField := binding
	missingConditionField.SelectedFields = []string{"id", "status", "owner_id"}
	if _, err := executor.EvaluateBinding(ctx, view, missingConditionField, condition); !errors.Is(err, ErrPopulationInvalid) {
		t.Fatalf("condition may not bypass binding field selection, got %v", err)
	}

	// The assurance wrapper did not open this shared session and therefore must
	// not close it. Another consumer can continue using it after Close.
	executor.Close()
	if _, err := lookupReader.Lookup(ctx, view, binding, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{sourceaccess.StringValue("81111111-1111-7111-8111-111111111111")}}); err != nil {
		t.Fatalf("assurance closed a shared source session: %v", err)
	}
}

func assertSourceReceiptCounts(t *testing.T, receipt EvaluationReceipt) {
	t.Helper()
	if !receipt.Complete || receipt.TotalCount != 4 || receipt.MatchCount != 1 || receipt.UnknownCount != 2 || receipt.ClearCount != 1 {
		t.Fatalf("unexpected evaluation receipt: %#v", receipt)
	}
}

func assurancePostgresRoleURL(t *testing.T, raw, username, password string) string {
	t.Helper()
	parsed, err := neturl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	parsed.User = neturl.UserPassword(username, password)
	return parsed.String()
}
