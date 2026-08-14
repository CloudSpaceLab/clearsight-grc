package assurance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type reusableSourceSession struct {
	connection sourceaccess.Connection
	fields     []sourceaccess.NativeField
	closed     bool
}

func (s *reusableSourceSession) Connection() sourceaccess.Connection { return s.connection }
func (s *reusableSourceSession) Capabilities() sourceaccess.CapabilitySet {
	return sourceaccess.NewCapabilitySet(sourceaccess.CapabilityInspect, sourceaccess.CapabilityLookup, sourceaccess.CapabilityAggregate)
}
func (s *reusableSourceSession) Close() error { s.closed = true; return nil }

func (s *reusableSourceSession) Inspect(_ context.Context, view sourceaccess.View) (sourceaccess.SchemaResult, error) {
	fingerprint, err := sourceaccess.ViewFingerprint(view)
	if err != nil {
		return sourceaccess.SchemaResult{}, err
	}
	return sourceaccess.SchemaResult{
		Fields: append([]sourceaccess.NativeField(nil), s.fields...),
		Receipt: sourceaccess.OperationReceipt{
			SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version,
			AdapterKind: sourceaccess.AdapterPostgres, AdapterVersion: sourceaccess.PostgresAdapterVersion,
			ViewID: view.ID, ViewVersion: view.Version, DefinitionFingerprint: fingerprint,
			Operation: sourceaccess.OperationInspect, ObservedAt: time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC),
			Count: int64(len(s.fields)), Completeness: sourceaccess.CompletenessComplete,
		},
	}, nil
}

func (s *reusableSourceSession) Lookup(_ context.Context, view sourceaccess.View, binding sourceaccess.Binding, request sourceaccess.LookupRequest) (sourceaccess.LookupResult, error) {
	fingerprint, err := sourceaccess.BindingFingerprint(view, binding)
	if err != nil {
		return sourceaccess.LookupResult{}, err
	}
	records := make([]sourceaccess.Record, 0, len(request.Values))
	for _, value := range request.Values {
		records = append(records, sourceaccess.Record{
			"account_id": value,
			"status":     sourceaccess.StringValue("ACTIVE"),
			"patch_age":  {Kind: sourceaccess.ScalarNumber, Text: "45"},
		})
	}
	return sourceaccess.LookupResult{
		Records: records,
		Receipt: sourceaccess.OperationReceipt{
			SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version,
			AdapterKind: sourceaccess.AdapterPostgres, AdapterVersion: sourceaccess.PostgresAdapterVersion,
			ViewID: view.ID, ViewVersion: view.Version, BindingID: binding.ID, BindingVersion: binding.Version,
			DefinitionFingerprint: fingerprint, Operation: sourceaccess.OperationLookup,
			ObservedAt: time.Date(2026, 8, 14, 9, 1, 0, 0, time.UTC), Count: int64(len(records)), Completeness: sourceaccess.CompletenessComplete,
		},
	}, nil
}

func (s *reusableSourceSession) EvaluatePredicate(_ context.Context, view sourceaccess.View, binding sourceaccess.Binding, _ sourceaccess.PostgresPredicate, guard sourceaccess.SchemaGuard) (sourceaccess.AggregateResult, error) {
	if guard != nil {
		if err := guard(append([]sourceaccess.NativeField(nil), s.fields...)); err != nil {
			return sourceaccess.AggregateResult{}, err
		}
	}
	fingerprint, err := sourceaccess.BindingFingerprint(view, binding)
	if err != nil {
		return sourceaccess.AggregateResult{}, err
	}
	return sourceaccess.AggregateResult{
		Fields: append([]sourceaccess.NativeField(nil), s.fields...), TotalCount: 4, MatchCount: 1, UnknownCount: 2, ClearCount: 1,
		Receipt: sourceaccess.OperationReceipt{
			SourceID: s.connection.SourceID, ConnectionID: s.connection.ID, ConnectionVersion: s.connection.Version,
			AdapterKind: sourceaccess.AdapterPostgres, AdapterVersion: sourceaccess.PostgresAdapterVersion,
			ViewID: view.ID, ViewVersion: view.Version, BindingID: binding.ID, BindingVersion: binding.Version,
			DefinitionFingerprint: fingerprint, Operation: sourceaccess.OperationAggregate,
			ObservedAt: time.Date(2026, 8, 14, 9, 2, 0, 0, time.UTC), Count: 4, Completeness: sourceaccess.CompletenessComplete,
		},
	}, nil
}

func TestOneTransientBindingDrivesAssuranceAndLookupConsumers(t *testing.T) {
	connection := sourceaccess.NewPostgresConnection("core-read", "core-source", "connection-v1", "secret-ref")
	session := &reusableSourceSession{
		connection: connection,
		fields: []sourceaccess.NativeField{
			{Name: "account_id", NativeType: "uuid", Nullable: false},
			{Name: "status", NativeType: "text", Nullable: false},
			{Name: "patch_age", NativeType: "numeric", Nullable: true},
		},
	}
	executor, err := NewPostgresSourceExecutorWithSession(connection.SourceID, session)
	if err != nil {
		t.Fatal(err)
	}
	view, err := sourceaccess.NewPostgresView("active-accounts", connection.ID, "view-v1", "SELECT account_id,status,patch_age FROM accounts", "account_id")
	if err != nil {
		t.Fatal(err)
	}
	binding := sourceaccess.Binding{
		ID: "account-governance", ViewID: view.ID, Version: "binding-v1", Purpose: "assurance-and-form-validation",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationAggregate, sourceaccess.OperationLookup},
		SelectedFields: []string{"account_id", "status", "patch_age"}, KeyFields: []string{"account_id"},
		Limits: sourceaccess.DefaultResourceLimits(),
	}

	inspected, err := executor.InspectBinding(context.Background(), view, binding)
	if err != nil {
		t.Fatal(err)
	}
	condition, err := CompileCondition(inspected.Schema, Condition{Op: OpAnd, Children: []Condition{
		{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
		{Op: OpGT, Field: "patch_age", Value: NumberLiteral(30)},
	}}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	evaluated, err := executor.EvaluateBinding(context.Background(), view, binding, condition)
	if err != nil {
		t.Fatal(err)
	}
	if evaluated.PopulationID != binding.ID || evaluated.PopulationFingerprint != inspected.PopulationFingerprint || evaluated.MatchCount != 1 || evaluated.UnknownCount != 2 || evaluated.ClearCount != 1 {
		t.Fatalf("unexpected assurance result: %#v", evaluated)
	}

	lookup, err := session.Lookup(context.Background(), view, binding, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{sourceaccess.StringValue("A-100")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Records) != 1 || lookup.Receipt.BindingID != binding.ID || lookup.Receipt.DefinitionFingerprint != evaluated.PopulationFingerprint {
		t.Fatalf("lookup did not reuse the assurance binding: %#v", lookup)
	}

	executor.Close()
	if session.closed {
		t.Fatal("assurance closed a caller-owned shared source session")
	}
}

func TestReusableBindingSchemaGuardFailsOnlyForCriticalChanges(t *testing.T) {
	connection := sourceaccess.NewPostgresConnection("core-read", "core-source", "connection-v1", "secret-ref")
	session := &reusableSourceSession{
		connection: connection,
		fields: []sourceaccess.NativeField{
			{Name: "account_id", NativeType: "uuid", Nullable: false},
			{Name: "status", NativeType: "text", Nullable: false},
			{Name: "patch_age", NativeType: "numeric", Nullable: true},
			{Name: "owner", NativeType: "text", Nullable: true},
		},
	}
	executor, err := NewPostgresSourceExecutorWithSession(connection.SourceID, session)
	if err != nil {
		t.Fatal(err)
	}
	view, err := sourceaccess.NewPostgresView("active-accounts", connection.ID, "view-v1", "SELECT account_id,status,patch_age,owner FROM accounts", "account_id")
	if err != nil {
		t.Fatal(err)
	}
	binding := sourceaccess.Binding{
		ID: "account-governance", ViewID: view.ID, Version: "binding-v1", Purpose: "continuous-assurance",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationAggregate},
		SelectedFields: []string{"account_id", "status", "patch_age", "owner"}, KeyFields: []string{"account_id"},
		Limits: sourceaccess.DefaultResourceLimits(),
	}
	inspected, err := executor.InspectBinding(context.Background(), view, binding)
	if err != nil {
		t.Fatal(err)
	}
	condition, err := CompileCondition(inspected.Schema, Condition{Op: OpGT, Field: "patch_age", Value: NumberLiteral(30)}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}

	session.fields[3].NativeType = "numeric"
	if _, err := executor.EvaluateBinding(context.Background(), view, binding, condition); err != nil {
		t.Fatalf("unrelated source field change should remain evaluable: %v", err)
	}
	session.fields[2].NativeType = "text"
	if _, err := executor.EvaluateBinding(context.Background(), view, binding, condition); !errors.Is(err, ErrSourceSchemaChanged) {
		t.Fatalf("critical source field change must fail closed, got %v", err)
	}
}

func TestAssuranceBindingRequiresASelectedStableSubjectKey(t *testing.T) {
	connection := sourceaccess.NewPostgresConnection("core-read", "core-source", "connection-v1", "secret-ref")
	session := &reusableSourceSession{
		connection: connection,
		fields: []sourceaccess.NativeField{
			{Name: "account_id", NativeType: "uuid", Nullable: false},
			{Name: "status", NativeType: "text", Nullable: false},
		},
	}
	executor, err := NewPostgresSourceExecutorWithSession(connection.SourceID, session)
	if err != nil {
		t.Fatal(err)
	}
	view, err := sourceaccess.NewPostgresView("active-accounts", connection.ID, "view-v1", "SELECT account_id,status FROM accounts")
	if err != nil {
		t.Fatal(err)
	}
	binding := sourceaccess.Binding{
		ID: "account-governance", ViewID: view.ID, Version: "binding-v1", Purpose: "continuous-assurance",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationAggregate},
		SelectedFields: []string{"account_id", "status"}, KeyFields: []string{"account_id"},
		Limits: sourceaccess.DefaultResourceLimits(),
	}
	if _, err := executor.InspectBinding(context.Background(), view, binding); !errors.Is(err, ErrPopulationInvalid) {
		t.Fatalf("assurance accepted an undeclared stable subject key: %v", err)
	}
}

func TestInspectBindingReturnsOnlyPurposeBoundSelectedFields(t *testing.T) {
	connection := sourceaccess.NewPostgresConnection("core-read", "core-source", "connection-v1", "secret-ref")
	session := &reusableSourceSession{
		connection: connection,
		fields: []sourceaccess.NativeField{
			{Name: "account_id", NativeType: "uuid", Nullable: false},
			{Name: "status", NativeType: "text", Nullable: false},
			{Name: "internal_note", NativeType: "text", Nullable: true},
		},
	}
	executor, err := NewPostgresSourceExecutorWithSession(connection.SourceID, session)
	if err != nil {
		t.Fatal(err)
	}
	view, err := sourceaccess.NewPostgresView("active-accounts", connection.ID, "view-v1", "SELECT account_id,status,internal_note FROM accounts", "account_id")
	if err != nil {
		t.Fatal(err)
	}
	binding := sourceaccess.Binding{
		ID: "account-governance", ViewID: view.ID, Version: "binding-v1", Purpose: "continuous-assurance",
		Operations:     []sourceaccess.Operation{sourceaccess.OperationAggregate},
		SelectedFields: []string{"account_id", "status"}, KeyFields: []string{"account_id"},
		Limits: sourceaccess.DefaultResourceLimits(),
	}
	inspected, err := executor.InspectBinding(context.Background(), view, binding)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := inspected.Schema.Field("internal_note"); exists {
		t.Fatal("binding inspection exposed an unselected source field")
	}
	if _, exists := inspected.Schema.Field("status"); !exists {
		t.Fatal("binding inspection omitted a selected source field")
	}
}
