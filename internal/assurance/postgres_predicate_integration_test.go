//go:build postgres && postgresintegration

package assurance

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresPredicateMatchesPureTriStateEvaluator(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `SET TIME ZONE 'UTC'`); err != nil {
		t.Fatal(err)
	}

	t.Run("nested boolean semantics", func(t *testing.T) {
		schema := Schema{Fields: []Field{
			{Name: "status", Type: TypeString},
			{Name: "score", Type: TypeNumber, Nullable: true},
			{Name: "owner_id", Type: TypeString, Nullable: true},
		}}
		condition := Condition{Op: OpOr, Children: []Condition{
			{Op: OpAnd, Children: []Condition{
				{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")},
				{Op: OpGTE, Field: "score", Value: NumberLiteral(80)},
			}},
			{Op: OpNot, Children: []Condition{{Op: OpMissing, Field: "owner_id"}}},
		}}
		compiled, err := CompileCondition(schema, condition, ConditionLimits{})
		if err != nil {
			t.Fatal(err)
		}

		cases := []postgresParityCase{
			{name: "match-first-branch", sourceSQL: `SELECT 'ACTIVE'::text AS status, 90::double precision AS score, NULL::text AS owner_id`, row: map[string]any{"status": "ACTIVE", "score": 90, "owner_id": nil}},
			{name: "unknown-without-match", sourceSQL: `SELECT 'ACTIVE'::text AS status, NULL::double precision AS score, NULL::text AS owner_id`, row: map[string]any{"status": "ACTIVE", "score": nil, "owner_id": nil}},
			{name: "match-second-branch", sourceSQL: `SELECT 'DORMANT'::text AS status, 90::double precision AS score, 'owner-1'::text AS owner_id`, row: map[string]any{"status": "DORMANT", "score": 90, "owner_id": "owner-1"}},
			{name: "clear", sourceSQL: `SELECT 'DORMANT'::text AS status, 90::double precision AS score, NULL::text AS owner_id`, row: map[string]any{"status": "DORMANT", "score": 90, "owner_id": nil}},
		}
		assertPostgresParity(t, ctx, pool, compiled, cases)
	})

	t.Run("field comparison semantics", func(t *testing.T) {
		schema := Schema{Fields: []Field{
			{Name: "target_rto_minutes", Type: TypeNumber},
			{Name: "actual_rto_minutes", Type: TypeNumber, Nullable: true},
		}}
		compiled, err := CompileCondition(schema, Condition{Op: OpGT, Field: "actual_rto_minutes", OtherField: "target_rto_minutes"}, ConditionLimits{})
		if err != nil {
			t.Fatal(err)
		}
		cases := []postgresParityCase{
			{name: "match", sourceSQL: `SELECT 30::double precision AS target_rto_minutes, 47::double precision AS actual_rto_minutes`, row: map[string]any{"target_rto_minutes": 30, "actual_rto_minutes": 47}},
			{name: "unknown", sourceSQL: `SELECT 30::double precision AS target_rto_minutes, NULL::double precision AS actual_rto_minutes`, row: map[string]any{"target_rto_minutes": 30, "actual_rto_minutes": nil}},
			{name: "clear", sourceSQL: `SELECT 30::double precision AS target_rto_minutes, 20::double precision AS actual_rto_minutes`, row: map[string]any{"target_rto_minutes": 30, "actual_rto_minutes": 20}},
		}
		assertPostgresParity(t, ctx, pool, compiled, cases)
	})

	t.Run("bounded source domain", func(t *testing.T) {
		t.Run("oversized string is unknown", func(t *testing.T) {
			schema := Schema{Fields: []Field{{Name: "status", Type: TypeString}}}
			compiled, err := CompileCondition(schema, Condition{Op: OpEQ, Field: "status", Value: StringLiteral("ACTIVE")}, ConditionLimits{})
			if err != nil {
				t.Fatal(err)
			}
			value := strings.Repeat("A", hardMaxEvaluatedStringBytes+1)
			assertPostgresParity(t, ctx, pool, compiled, []postgresParityCase{{
				name:      "oversized",
				sourceSQL: `SELECT repeat('A', 65537)::text AS status`,
				row:       map[string]any{"status": value},
			}})
		})

		t.Run("out-of-domain number is unknown", func(t *testing.T) {
			schema := Schema{Fields: []Field{{Name: "amount", Type: TypeNumber}}}
			compiled, err := CompileCondition(schema, Condition{Op: OpGT, Field: "amount", Value: NumberLiteral(1)}, ConditionLimits{})
			if err != nil {
				t.Fatal(err)
			}
			assertPostgresParity(t, ctx, pool, compiled, []postgresParityCase{
				{name: "integer", sourceSQL: `SELECT 9007199254740993::numeric AS amount`, row: map[string]any{"amount": int64(maxExactFloatInteger + 1)}},
				{name: "float", sourceSQL: `SELECT 9007199254740994::double precision AS amount`, row: map[string]any{"amount": float64(maxExactFloatInteger) + 2}},
				{name: "nan", sourceSQL: `SELECT 'NaN'::double precision AS amount`, row: map[string]any{"amount": math.NaN()}},
			})
		})
	})

	t.Run("logical normalization parity", func(t *testing.T) {
		t.Run("uuid string", func(t *testing.T) {
			const id = "83333333-3333-7333-8333-333333333331"
			schema := Schema{Fields: []Field{{Name: "subject_id", Type: TypeString}}}
			compiled, err := CompileCondition(schema, Condition{Op: OpEQ, Field: "subject_id", Value: StringLiteral(id)}, ConditionLimits{})
			if err != nil {
				t.Fatal(err)
			}
			assertPostgresParity(t, ctx, pool, compiled, []postgresParityCase{{
				name:      "uuid",
				sourceSQL: `SELECT '83333333-3333-7333-8333-333333333331'::uuid AS subject_id`,
				row:       map[string]any{"subject_id": id},
			}})
		})

		t.Run("numeric field comparison uses float semantics", func(t *testing.T) {
			schema := Schema{Fields: []Field{{Name: "left_value", Type: TypeNumber}, {Name: "right_value", Type: TypeNumber}}}
			compiled, err := CompileCondition(schema, Condition{Op: OpEQ, Field: "left_value", OtherField: "right_value"}, ConditionLimits{})
			if err != nil {
				t.Fatal(err)
			}
			assertPostgresParity(t, ctx, pool, compiled, []postgresParityCase{{
				name:      "rounded-to-same-float",
				sourceSQL: `SELECT 0.1::numeric AS left_value, 0.10000000000000001::numeric AS right_value`,
				row:       map[string]any{"left_value": float64(0.1), "right_value": float64(0.1)},
			}})
		})
	})
}

type postgresParityCase struct {
	name      string
	sourceSQL string
	row       map[string]any
}

func assertPostgresParity(t *testing.T, ctx context.Context, pool *pgxpool.Pool, compiled *CompiledCondition, cases []postgresParityCase) {
	t.Helper()
	predicate, err := compiled.PostgresPredicate()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			query := `SELECT ` + predicate.MatchSQL + ` AS matched, ` + predicate.UnknownSQL + ` AS unknown FROM (` + test.sourceSQL + `) AS population`
			var matched, unknown bool
			if err := pool.QueryRow(ctx, query, predicate.Args...).Scan(&matched, &unknown); err != nil {
				t.Fatalf("execute compiled predicate: %v\nSQL: %s", err, query)
			}
			if matched && unknown {
				t.Fatalf("compiled predicate returned MATCH and UNKNOWN simultaneously")
			}
			result := ResultClear
			if unknown {
				result = ResultUnknown
			} else if matched {
				result = ResultMatch
			}
			if want := compiled.Evaluate(test.row); result != want {
				t.Fatalf("PostgreSQL result=%s pure result=%s; match=%t unknown=%t", result, want, matched, unknown)
			}
		})
	}
}
