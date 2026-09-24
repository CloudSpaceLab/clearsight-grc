//go:build load

package ropa_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

// TestRegisterListAtTargetVolume exercises the bounded in-memory register read
// at the very-large-bank target population. The activities intentionally have
// no child collections: the list path's own bounded cost is under test, and
// retaining 100,000 fully hydrated child collections would measure a different
// memory profile rather than the register read being budgeted here.
func TestRegisterListAtTargetVolume(t *testing.T) {
	if testing.Short() {
		t.Skip("load test")
	}

	const (
		population    = 100_000
		pageSize      = 50
		listBudget    = 750 * time.Millisecond
		summaryBudget = 500 * time.Millisecond
	)

	ctx := context.Background()
	repository := ropa.NewMemoryRepository()
	summaries := ropa.NewMemorySummaryRepository()
	service := ropa.NewService(repository, summaries)
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	scope := ropa.ActivityScope{TenantID: "tenant-load", LegalEntityID: "entity-load"}

	seedStart := time.Now()
	for index := 0; index < population; index++ {
		reviewDate := now.AddDate(0, 0, index%365)
		_, err := service.CreateActivity(ctx, ropa.CreateActivityInput{
			TenantID:              scope.TenantID,
			LegalEntityID:         scope.LegalEntityID,
			Code:                  fmt.Sprintf("PA-LOAD-%06d", index),
			Name:                  fmt.Sprintf("Large-bank processing activity %06d", index),
			Controller:            "Clear Bank Nigeria",
			LawfulBasis:           "Contract",
			DataSubjectCategories: "Customers",
			OwnerPrincipalID:      "owner-load",
			NextReviewDate:        &reviewDate,
			ActorID:               "load-seed",
		})
		if err != nil {
			t.Fatalf("seed activity %d: %v", index, err)
		}
	}
	seedElapsed := time.Since(seedStart)
	t.Logf("seeded %d activities without children in %s", population, seedElapsed)

	firstStart := time.Now()
	first, err := service.ListActivities(ctx, scope, ropa.ListActivitiesFilter{Limit: pageSize})
	firstElapsed := time.Since(firstStart)
	if err != nil {
		t.Fatalf("first register page: %v", err)
	}
	if len(first.Rows) != pageSize || !first.HasMore || first.NextCursor == "" {
		t.Fatalf("first page = rows %d, has_more %t, cursor %q; want a full first page and cursor", len(first.Rows), first.HasMore, first.NextCursor)
	}
	if firstElapsed > listBudget {
		t.Fatalf("first register page took %s at %d activities, budget is %s", firstElapsed, population, listBudget)
	}
	t.Logf("first page (%d rows) at %d activities: %s (budget %s)", pageSize, population, firstElapsed, listBudget)

	secondStart := time.Now()
	second, err := service.ListActivities(ctx, scope, ropa.ListActivitiesFilter{
		Limit:  pageSize,
		Cursor: first.NextCursor,
	})
	secondElapsed := time.Since(secondStart)
	if err != nil {
		t.Fatalf("cursor register page: %v", err)
	}
	if len(second.Rows) != pageSize || !second.HasMore || second.NextCursor == "" || second.NextCursor == first.NextCursor {
		t.Fatalf("second page = rows %d, has_more %t, cursor %q; want a full second page with an advancing cursor", len(second.Rows), second.HasMore, second.NextCursor)
	}
	if secondElapsed > listBudget {
		t.Fatalf("cursor register page took %s at %d activities, budget is %s", secondElapsed, population, listBudget)
	}
	firstIDs := make(map[string]struct{}, len(first.Rows))
	for _, activity := range first.Rows {
		if activity.TenantID != scope.TenantID || activity.LegalEntityID != scope.LegalEntityID {
			t.Fatalf("first page escaped the requested scope: %#v", activity)
		}
		firstIDs[activity.ID] = struct{}{}
	}
	for _, activity := range second.Rows {
		if activity.TenantID != scope.TenantID || activity.LegalEntityID != scope.LegalEntityID {
			t.Fatalf("second page escaped the requested scope: %#v", activity)
		}
		if _, duplicate := firstIDs[activity.ID]; duplicate {
			t.Fatalf("cursor page repeated first-page activity %s", activity.ID)
		}
	}
	t.Logf("second page (%d rows) via cursor at %d activities: %s (budget %s)", pageSize, population, secondElapsed, listBudget)

	maintainer := ropa.NewSummaryMaintainer(repository, summaries, service)
	// A larger bounded batch reduces traversal overhead while staying within
	// the service's explicit projection batch bound.
	maintainer.BatchSize = 1000
	projectionStart := time.Now()
	if err := maintainer.Maintain(ctx, scope); err != nil {
		t.Fatalf("maintain dashboard projection: %v", err)
	}
	projectionElapsed := time.Since(projectionStart)
	summary, err := service.RegisterSummary(ctx, scope.TenantID, scope.LegalEntityID)
	if err != nil {
		t.Fatalf("read dashboard projection: %v", err)
	}
	if summary.Counts.Total != population || summary.Counts.New != population || summary.Counts.Open != 0 || summary.Counts.Closed != 0 {
		t.Fatalf("projection counts = %#v, want total/new=%d/%d and no open/closed rows", summary.Counts, population, population)
	}
	if summary.Counts.Total != summary.Counts.New+summary.Counts.Open+summary.Counts.Closed {
		t.Fatalf("projection lifecycle counts do not reconcile: %#v", summary.Counts)
	}
	if summary.Coverage.Population != population {
		t.Fatalf("projection population = %d, want %d", summary.Coverage.Population, population)
	}
	if summary.Coverage.Excluded == nil || summary.Coverage.Unknown == nil || *summary.Coverage.Excluded != 0 || *summary.Coverage.Unknown != 0 {
		t.Fatalf("projection coverage = %#v, want explicit zero excluded/unknown", summary.Coverage)
	}
	t.Logf("dashboard projection maintain at %d activities: %s; summary counts reconciled: %#v", population, projectionElapsed, summary.Counts)

	summaryReadStart := time.Now()
	if _, err := service.RegisterSummary(ctx, scope.TenantID, scope.LegalEntityID); err != nil {
		t.Fatalf("read dashboard projection for timing: %v", err)
	}
	summaryReadElapsed := time.Since(summaryReadStart)
	if summaryReadElapsed > summaryBudget {
		t.Fatalf("dashboard projection read took %s, budget is %s", summaryReadElapsed, summaryBudget)
	}
	t.Logf("dashboard projection read at %d activities: %s (budget %s)", population, summaryReadElapsed, summaryBudget)
}
