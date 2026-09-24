//go:build postgres

package reporting

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestExceptionPredicateSQLAgreesWithGo(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not configured; SQL/register parity was not run")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	connection, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Release()

	schema := fmt.Sprintf("reporting_exception_parity_%d", time.Now().UnixNano())
	if _, err := connection.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = connection.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
	}()
	if _, err := connection.Exec(ctx, "SET search_path TO "+schema+", public"); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = connection.Exec(context.Background(), "RESET search_path") }()

	if _, err := connection.Exec(ctx, `
		CREATE TABLE ropa_processing_activities(
			id uuid PRIMARY KEY,
			tenant_id uuid NOT NULL,
			legal_entity_id uuid NOT NULL,
			lawful_basis text NOT NULL,
			owner_principal_id uuid,
			data_subject_categories text NOT NULL
		);
		CREATE TABLE ropa_processing_activity_reviews(
			id uuid PRIMARY KEY,
			tenant_id uuid NOT NULL,
			legal_entity_id uuid NOT NULL,
			activity_id uuid NOT NULL,
			completed_at timestamptz,
			outcome text,
			FOREIGN KEY (activity_id) REFERENCES ropa_processing_activities(id)
		);
	`); err != nil {
		t.Fatal(err)
	}

	const (
		tenantID      = "00000000-0000-7000-8000-000000000101"
		entityID      = "00000000-0000-7000-8000-000000000102"
		ownerID       = "00000000-0000-7000-8000-000000000103"
		completedTime = "2026-09-24T10:00:00Z"
	)
	type postgresCase struct {
		id       string
		activity ropa.ProcessingActivity
		review   *struct {
			completedAt *time.Time
			outcome     string
		}
	}
	completed := func(value string) *time.Time {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return &parsed
	}
	review := func(value *time.Time, outcome string) *struct {
		completedAt *time.Time
		outcome     string
	} {
		return &struct {
			completedAt *time.Time
			outcome     string
		}{completedAt: value, outcome: outcome}
	}
	cases := []postgresCase{
		{id: "00000000-0000-7000-8000-000000000111", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "CONFIRMED"}}}, review: review(completed(completedTime), "CONFIRMED")},
		{id: "00000000-0000-7000-8000-000000000112", activity: ropa.ProcessingActivity{OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "CONFIRMED"}}}, review: review(completed(completedTime), "CONFIRMED")},
		{id: "00000000-0000-7000-8000-000000000113", activity: ropa.ProcessingActivity{LawfulBasis: "   ", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "CONFIRMED"}}}, review: review(completed(completedTime), "CONFIRMED")},
		{id: "00000000-0000-7000-8000-000000000114", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "CONFIRMED"}}}, review: review(completed(completedTime), "CONFIRMED")},
		{id: "00000000-0000-7000-8000-000000000115", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "CONFIRMED"}}}, review: review(completed(completedTime), "CONFIRMED")},
		{id: "00000000-0000-7000-8000-000000000116", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS"}},
		{id: "00000000-0000-7000-8000-000000000117", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{}}}, review: review(nil, "")},
		{id: "00000000-0000-7000-8000-000000000118", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "WITHDRAWN"}}}, review: review(completed(completedTime), "WITHDRAWN")},
		{id: "00000000-0000-7000-8000-000000000119", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "  "}}}, review: review(completed(completedTime), "  ")},
		{id: "00000000-0000-7000-8000-000000000120", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: "REVISED"}}}, review: review(completed(completedTime), "REVISED")},
		{id: "00000000-0000-7000-8000-000000000121", activity: ropa.ProcessingActivity{LawfulBasis: "CONSENT", OwnerPrincipalID: ownerID, DataSubjectCategories: "CUSTOMERS", Reviews: []ropa.Review{{CompletedAt: completed(completedTime), Outcome: " CONFIRMED "}}}, review: review(completed(completedTime), " CONFIRMED ")},
	}

	expected := make([]string, 0)
	for index, testCase := range cases {
		var owner any
		if testCase.activity.OwnerPrincipalID != "" {
			owner = testCase.activity.OwnerPrincipalID
		}
		if _, err := connection.Exec(ctx, `
			INSERT INTO ropa_processing_activities
				(id, tenant_id, legal_entity_id, lawful_basis, owner_principal_id, data_subject_categories)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5::uuid,$6)
		`, testCase.id, tenantID, entityID, testCase.activity.LawfulBasis, owner, testCase.activity.DataSubjectCategories); err != nil {
			t.Fatalf("insert activity %d: %v", index, err)
		}
		if testCase.review != nil {
			if _, err := connection.Exec(ctx, `
				INSERT INTO ropa_processing_activity_reviews
					(id, tenant_id, legal_entity_id, activity_id, completed_at, outcome)
				VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6)
			`, fmt.Sprintf("00000000-0000-7000-8000-%012d", 200+index), tenantID, entityID, testCase.id, testCase.review.completedAt, testCase.review.outcome); err != nil {
				t.Fatalf("insert review %d: %v", index, err)
			}
		}
		if ExceptionPredicate(testCase.activity) {
			expected = append(expected, testCase.id)
		}
	}

	rows, err := connection.Query(ctx, `SELECT a.id::text FROM ropa_processing_activities a WHERE `+ExceptionPredicateSQL+` ORDER BY a.id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	actual := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		actual = append(actual, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(expected)
	sort.Strings(actual)
	if len(actual) != len(expected) {
		t.Fatalf("SQL selected %d activities, Go selected %d: SQL=%v Go=%v", len(actual), len(expected), actual, expected)
	}
	for index := range actual {
		if actual[index] != expected[index] {
			t.Fatalf("SQL/Go exception parity differs at %d: SQL=%v Go=%v", index, actual, expected)
		}
	}
}
