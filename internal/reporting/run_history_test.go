package reporting

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryRunHistoryUsesStableKeysetPagination(t *testing.T) {
	repository := NewMemoryRepository()
	scope := ReportScope{TenantID: DemoTenant, LegalEntityID: DemoLegalEntity}
	newest := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	older := newest.Add(-time.Hour)

	runs := []ReportRun{
		{ID: "00000000-0000-7000-8000-000000000603", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, CreatedAt: newest},
		{ID: "00000000-0000-7000-8000-000000000602", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, CreatedAt: newest},
		{ID: "00000000-0000-7000-8000-000000000601", TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID, CreatedAt: older},
	}
	for _, run := range runs {
		repository.runs[run.ID] = run
	}

	first, err := repository.ListRunHistory(context.Background(), scope, "", "", 2)
	if err != nil {
		t.Fatalf("list first report history page: %v", err)
	}
	if len(first.Items) != 2 || first.Items[0].ID != runs[0].ID || first.Items[1].ID != runs[1].ID {
		t.Fatalf("first report history page = %#v", first.Items)
	}
	if first.NextCursor == "" {
		t.Fatal("first report history page did not expose a continuation cursor")
	}

	second, err := repository.ListRunHistory(context.Background(), scope, "", first.NextCursor, 2)
	if err != nil {
		t.Fatalf("list second report history page: %v", err)
	}
	if len(second.Items) != 1 || second.Items[0].ID != runs[2].ID {
		t.Fatalf("second report history page = %#v", second.Items)
	}
	if second.NextCursor != "" {
		t.Fatalf("terminal report history page retained cursor %q", second.NextCursor)
	}
}

func TestRunHistoryCursorRejectsMalformedInput(t *testing.T) {
	if _, err := decodeRunHistoryCursor("not-a-valid-cursor"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("malformed history cursor error = %v, want ErrInvalid", err)
	}
}

func TestRunHistoryCursorRoundTripsExactBoundary(t *testing.T) {
	run := ReportRun{
		ID:        "00000000-0000-7000-8000-000000000611",
		CreatedAt: time.Date(2026, 9, 25, 12, 34, 56, 789, time.UTC),
	}
	encoded, err := encodeRunHistoryCursor(run)
	if err != nil {
		t.Fatalf("encode report history cursor: %v", err)
	}
	decoded, err := decodeRunHistoryCursor(encoded)
	if err != nil {
		t.Fatalf("decode report history cursor: %v", err)
	}
	if decoded.ID != run.ID || !decoded.CreatedAt.Equal(run.CreatedAt) {
		t.Fatalf("history cursor = %#v, want id %q at %s", decoded, run.ID, run.CreatedAt)
	}
}
