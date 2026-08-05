package evidence

import (
	"context"
	"testing"
	"time"
)

func TestLatestRequestForSubjectPrefersActionableWork(t *testing.T) {
	now := time.Date(2026, 8, 5, 20, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository(nil, []Request{
		{ID: "submitted", TenantID: "bank-a", SubjectType: "MATTER", SubjectID: "matter-1", Status: RequestSubmitted, Deadline: now.Add(-time.Hour), UpdatedAt: now},
		{ID: "ready-later", TenantID: "bank-a", SubjectType: "MATTER", SubjectID: "matter-1", Status: RequestReady, Deadline: now.Add(48 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "ready-urgent", TenantID: "bank-a", SubjectType: "MATTER", SubjectID: "matter-1", Status: RequestInProgress, Deadline: now.Add(24 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
	})
	service := NewService(repo, NewMemoryObjectStore())
	request, err := service.LatestRequestForSubject(context.Background(), "bank-a", "MATTER", "matter-1")
	if err != nil {
		t.Fatal(err)
	}
	if request.ID != "ready-urgent" {
		t.Fatalf("expected earliest actionable request, got %s", request.ID)
	}
}
