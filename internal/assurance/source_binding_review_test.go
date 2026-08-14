package assurance

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestLegacyPopulationEvaluationPreservesInspectedSchemaFingerprint(t *testing.T) {
	connection := sourceaccess.NewPostgresConnection("core-read", "core-source", "connection-v1", "secret-ref")
	session := &reusableSourceSession{
		connection: connection,
		fields: []sourceaccess.NativeField{
			{Name: "account_id", NativeType: "uuid", Nullable: false},
			{Name: "status", NativeType: "text", Nullable: false},
			{Name: "owner", NativeType: "text", Nullable: true},
		},
	}
	executor, err := NewPostgresSourceExecutorWithSession(connection.SourceID, session)
	if err != nil {
		t.Fatal(err)
	}
	population := PopulationDefinition{
		ID:         "accounts",
		Query:      "SELECT account_id,status,owner FROM accounts",
		SubjectKey: "account_id",
	}
	inspected, err := executor.InspectSchema(context.Background(), population)
	if err != nil {
		t.Fatal(err)
	}
	condition, err := CompileCondition(inspected.Schema, Condition{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")}, ConditionLimits{})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := executor.Evaluate(context.Background(), population, condition)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.SchemaFingerprint != inspected.SchemaFingerprint {
		t.Fatalf("legacy evaluation schema fingerprint=%q want inspected=%q", receipt.SchemaFingerprint, inspected.SchemaFingerprint)
	}
}

func TestSharedExecutorRejectsInvalidOrMismatchedPostgresSessions(t *testing.T) {
	base := sourceaccess.NewPostgresConnection("core-read", "core-source", "connection-v1", "secret-ref")
	tests := []struct {
		name       string
		connection sourceaccess.Connection
		sourceID   string
	}{
		{name: "invalid connection", connection: func() sourceaccess.Connection {
			value := base
			value.Version = " "
			return value
		}(), sourceID: base.SourceID},
		{name: "adapter version", connection: func() sourceaccess.Connection {
			value := base
			value.AdapterVersion = "postgres-v0"
			return value
		}(), sourceID: base.SourceID},
		{name: "adapter kind", connection: func() sourceaccess.Connection {
			value := base
			value.AdapterKind = sourceaccess.AdapterKind("REST")
			return value
		}(), sourceID: base.SourceID},
		{name: "source", connection: func() sourceaccess.Connection {
			value := base
			value.SourceID = "other-source"
			return value
		}(), sourceID: base.SourceID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := &reusableSourceSession{connection: test.connection}
			if _, err := NewPostgresSourceExecutorWithSession(test.sourceID, session); !errors.Is(err, ErrPopulationInvalid) {
				t.Fatalf("mismatched session should fail with ErrPopulationInvalid, got %v", err)
			}
		})
	}
}

func TestSelectedLogicalSchemaRejectsDuplicateSourceFields(t *testing.T) {
	_, err := logicalSchemaForFields([]sourceaccess.NativeField{
		{Name: "account_id", NativeType: "uuid"},
		{Name: "status", NativeType: "text"},
		{Name: "status", NativeType: "numeric"},
	}, []string{"account_id", "status"})
	if !errors.Is(err, ErrPopulationInvalid) {
		t.Fatalf("duplicate source field should fail closed, got %v", err)
	}
}

func TestUnknownSourceSessionErrorsAreSanitized(t *testing.T) {
	marker := "postgres://reader:SUPER_SECRET@source.internal/risk"
	mapped := mapSourceAccessError(errors.New(marker))
	if !errors.Is(mapped, ErrSourceExecution) {
		t.Fatalf("unknown source error should map to ErrSourceExecution, got %v", mapped)
	}
	if strings.Contains(mapped.Error(), marker) {
		t.Fatalf("unknown source error leaked implementation detail: %v", mapped)
	}
}
