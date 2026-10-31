//go:build postgres && postgresintegration

package workflow

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// An empty source is sufficient to exercise both fully authorized SQL
// selections, parameter bindings, repeatable-read aggregation and shape.
func TestActorWorkDueCountsPostgresQueriesUseAuthorizedSelections(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	service := NewService(NewPostgresRepository(pool))
	value, err := service.ReadActorWorkDueCounts(ctx,
		"missing-work-count-tenant-20261026",
		"missing-entity",
		"00000000-0000-4000-8000-000000000123",
	)
	if err != nil {
		t.Fatalf("authorized workflow/proposal COUNT SQL failed: %v", err)
	}
	if value.Total != 0 || value.Overdue != 0 || value.DueSoon != 0 || value.Undated != 0 {
		t.Fatalf("nonexistent source must be empty: %#v", value)
	}
	if value.CountBasis != "SOURCE_SNAPSHOT" || value.Coverage != "WORKFLOW_ASSIGNMENTS_ONLY" {
		t.Fatalf("count receipt lost source qualification: %#v", value)
	}
}
