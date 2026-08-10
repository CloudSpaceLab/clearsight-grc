//go:build postgres && postgresintegration

package assurance

import (
	"context"
	"os"
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
