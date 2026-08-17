package sourceaccess

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestBoundedPagePayloadQueryDoesNotBudgetTheLookaheadRecord(t *testing.T) {
	query := boundedPagePayloadQuery(`SELECT id,payload FROM source`, "id", 25, 3)
	for _, expected := range []string{
		`CASE WHEN row_number <= 25 THEN octet_length(payload) ELSE 0 END`,
		`row_number <= 25 AND cumulative_bytes <= $3`,
		`row_number > 25) AS lookahead`,
		`THEN to_jsonb(sort_key)::text ELSE '' END AS lookahead_key`,
	} {
		if !strings.Contains(query, expected) {
			t.Fatalf("bounded page query missing %q: %s", expected, query)
		}
	}
}

func TestPostgresScalarArgumentsUseTheNativeKeyType(t *testing.T) {
	tests := []struct {
		name       string
		value      Scalar
		nativeType string
		check      func(any) bool
	}{
		{name: "bigint", value: Scalar{Kind: ScalarNumber, Text: "9223372036854775807"}, nativeType: "int8", check: func(value any) bool {
			_, ok := value.(int64)
			return ok
		}},
		{name: "numeric", value: Scalar{Kind: ScalarNumber, Text: "9007199254740994.125"}, nativeType: "numeric", check: func(value any) bool {
			_, ok := value.(pgtype.Numeric)
			return ok
		}},
		{name: "date", value: Scalar{Kind: ScalarTime, Text: "2026-08-14"}, nativeType: "date", check: func(value any) bool {
			_, ok := value.(pgtype.Date)
			return ok
		}},
		{name: "timestamp", value: Scalar{Kind: ScalarTime, Text: "2026-08-14T09:50:58.123456Z"}, nativeType: "timestamp", check: func(value any) bool {
			_, ok := value.(pgtype.Timestamp)
			return ok
		}},
		{name: "timestamptz", value: Scalar{Kind: ScalarTime, Text: "2026-08-14T09:50:58.123456Z"}, nativeType: "timestamptz", check: func(value any) bool {
			_, ok := value.(pgtype.Timestamptz)
			return ok
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			argument, err := postgresScalarArgument(test.value, test.nativeType)
			if err != nil {
				t.Fatal(err)
			}
			if !test.check(argument) {
				t.Fatalf("argument type=%T does not match native type %q", argument, test.nativeType)
			}
		})
	}
}

func TestPostgresNumericArgumentPreservesScientificNotationExactly(t *testing.T) {
	argument, err := postgresScalarArgument(Scalar{Kind: ScalarNumber, Text: "-1.5e+12"}, "numeric")
	if err != nil {
		t.Fatal(err)
	}
	numeric := argument.(pgtype.Numeric)
	if numeric.Int == nil || numeric.Int.String() != "-15" || numeric.Exp != 11 || !numeric.Valid {
		t.Fatalf("scientific numeric lost precision: %#v", numeric)
	}
}

func TestPostgresScalarArgumentsRejectNativeTypeMismatches(t *testing.T) {
	for _, test := range []struct {
		value      Scalar
		nativeType string
	}{
		{value: Scalar{Kind: ScalarString, Text: "1"}, nativeType: "bigint"},
		{value: Scalar{Kind: ScalarNumber, Text: "1.5"}, nativeType: "bigint"},
		{value: Scalar{Kind: ScalarTime, Text: "2026-08-14T09:00:00Z"}, nativeType: "date"},
	} {
		if _, err := postgresScalarArgument(test.value, test.nativeType); !errors.Is(err, ErrDefinitionInvalid) {
			t.Fatalf("value %#v for %q should fail, got %v", test.value, test.nativeType, err)
		}
	}
}

func TestSelectedNativeFieldsDoNotExposeUnselectedMetadata(t *testing.T) {
	selected, err := selectedNativeFields([]NativeField{
		{Name: "id", NativeType: "uuid"},
		{Name: "status", NativeType: "text"},
		{Name: "internal_note", NativeType: "text"},
	}, []string{"id", "status"})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 || selected[0].Name != "id" || selected[1].Name != "status" {
		t.Fatalf("unexpected selected schema: %#v", selected)
	}
}
