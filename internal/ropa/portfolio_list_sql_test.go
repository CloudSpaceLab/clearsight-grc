package ropa_test

import (
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

const registerStatusRankSQL = "CASE status WHEN 'NEW' THEN 1 WHEN 'OPEN' THEN 2 WHEN 'CLOSED' THEN 3 END"
const registerReviewDateSQL = "COALESCE(next_review_date, '0001-01-01'::date)"

func TestListSQLIsBoundedScopedAndCurrentRowOnly(t *testing.T) {
	query := ropa.ListActivitiesSQL()
	upper := strings.ToUpper(query)

	for _, fragment := range []string{
		"WITH PAGE AS MATERIALIZED",
		"LIMIT $12",
		"TENANT_ID = $1::UUID",
		"LEGAL_ENTITY_ID = $2::UUID",
		"END_DATE IS NULL",
	} {
		if !strings.Contains(upper, fragment) {
			t.Errorf("register list SQL must contain %q", fragment)
		}
	}
	if strings.Contains(upper, "OFFSET") {
		t.Error("register list SQL must use keyset pagination without OFFSET")
	}
	if strings.Contains(query, "ropa_events") {
		t.Error("ordinary register list must not read the event ledger")
	}
}

func TestListSQLUsesTheMigrationKeysetTuple(t *testing.T) {
	query := ropa.ListActivitiesSQL()
	orderBy := "ORDER BY " + registerStatusRankSQL + ",\n         " + registerReviewDateSQL + ",\n         id"
	keyset := "AND (NOT $8 OR (" + registerStatusRankSQL + ",\n         " + registerReviewDateSQL + ",\n         id) >\n         ($9::integer, $10::date, $11::uuid))"

	if !strings.Contains(query, orderBy) {
		t.Errorf("register list ORDER BY must match ropa_register_keyset_idx; missing %q", orderBy)
	}
	if !strings.Contains(query, keyset) {
		t.Errorf("register list cursor predicate must use the indexed tuple; missing %q", keyset)
	}
}

func TestSummarySQLGroupsTheScopedPopulationInPostgreSQL(t *testing.T) {
	query := ropa.RegisterSummarySQL()
	upper := strings.ToUpper(query)

	for _, fragment := range []string{
		"COUNT(",
		"GROUP BY",
		"'TOTAL'",
		"'NEW'",
		"'OPEN'",
		"'CLOSED'",
		"'REVIEW_OVERDUE'",
		"'MISSING_LAWFUL_BASIS'",
		"'MISSING_OWNER'",
		"'NO_DATA_SUBJECTS'",
		"'RETIRED'",
		"ON CONFLICT (TENANT_ID, LEGAL_ENTITY_ID)",
		"EXCLUDED.GENERATED_AT >= ROPA_REGISTER_SUMMARY.GENERATED_AT",
	} {
		if !strings.Contains(upper, fragment) {
			t.Errorf("register summary SQL must contain %q", fragment)
		}
	}
	if strings.Contains(query, "jsonb_array_elements") {
		t.Error("register summary SQL must aggregate rows in PostgreSQL, not expand per-row application data")
	}
}
