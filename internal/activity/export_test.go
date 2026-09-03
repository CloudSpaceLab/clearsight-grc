package activity

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestExportServiceCreatesCompleteTenantScopedCSVAndAuditedDownload(t *testing.T) {
	now := time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC)
	activityService := NewService(NewMemoryRepository(
		Event{TenantID: "bank-a", ID: "visible", OccurredAt: now.Add(-time.Minute), EventType: "THIRD_PARTY_ASSESSMENT_COMPLETED", ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: "assessment-1", ActorKind: "EXTERNAL_PARTY", ActorID: "vendor-user", ActorDisplayName: "Acme Payments", LegalEntityID: "entity-a"},
		Event{TenantID: "bank-b", ID: "hidden", OccurredAt: now.Add(-2 * time.Minute), EventType: "THIRD_PARTY_ASSESSMENT_COMPLETED", ObjectType: "THIRD_PARTY_ASSESSMENT", ObjectID: "assessment-2", ActorKind: "EXTERNAL_PARTY", ActorID: "vendor-user", ActorDisplayName: "Other Vendor", LegalEntityID: "entity-a"},
	))
	receipts := NewMemoryExportRepository()
	receipts.now = func() time.Time { return now }
	objects := evidence.NewMemoryObjectStore()
	service := NewExportService(activityService, receipts, objects)
	service.now = func() time.Time { return now }

	receipt, err := service.Create(context.Background(), "bank-a", "entity-a", "admin-1", ExportFormatCSV, Query{
		Category: CategoryVendor, ActorQuery: "Acme", LegalEntityID: "entity-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != ExportStatusReady || receipt.RowCount != 1 || receipt.DataSHA256 == "" || receipt.ManifestSHA256 == "" {
		t.Fatalf("unexpected export receipt: %#v", receipt)
	}
	if receipt.Filter.Category != CategoryVendor || receipt.Filter.ActorQuery != "Acme" || receipt.Filter.To == nil || !receipt.Filter.To.Equal(now) {
		t.Fatalf("export did not retain the exact bounded filter: %#v", receipt.Filter)
	}

	opened, reader, err := service.Open(context.Background(), "bank-a", receipt.ID, "auditor-2")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if opened.ID != receipt.ID || !strings.Contains(text, "event_id,occurred_at,category") || !strings.Contains(text, "visible") || strings.Contains(text, "hidden") {
		t.Fatalf("unexpected CSV export: %s", text)
	}
}

func TestExportServiceCreatesNDJSONWithoutHTMLRewriting(t *testing.T) {
	now := time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC)
	activityService := NewService(NewMemoryRepository(
		Event{TenantID: "bank-a", ID: "event-1", OccurredAt: now.Add(-time.Minute), EventType: "MATTER_CREATED", ObjectType: "MATTER", ObjectID: "matter-1", ActorDisplayName: "Risk & Control"},
	))
	receipts := NewMemoryExportRepository()
	objects := evidence.NewMemoryObjectStore()
	service := NewExportService(activityService, receipts, objects)
	service.now = func() time.Time { return now }

	receipt, err := service.Create(context.Background(), "bank-a", "", "admin-1", ExportFormatNDJSON, Query{})
	if err != nil {
		t.Fatal(err)
	}
	_, reader, err := service.Open(context.Background(), "bank-a", receipt.ID, "admin-1")
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"actor_display_name":"Risk & Control"`) || strings.Contains(string(data), `\u0026`) {
		t.Fatalf("unexpected NDJSON export: %s", data)
	}
}

func TestExportServiceRefusesSilentTruncation(t *testing.T) {
	now := time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC)
	events := make([]Event, 0, maxExportRows+1)
	for index := 0; index <= maxExportRows; index++ {
		events = append(events, Event{
			TenantID: "bank-a", ID: fmt.Sprintf("event-%05d", index), OccurredAt: now.Add(-time.Duration(index) * time.Second),
			EventType: "MATTER_UPDATED", ObjectType: "MATTER", ObjectID: fmt.Sprintf("matter-%05d", index),
		})
	}
	receipts := NewMemoryExportRepository()
	receipts.now = func() time.Time { return now }
	service := NewExportService(NewService(NewMemoryRepository(events...)), receipts, evidence.NewMemoryObjectStore())
	service.now = func() time.Time { return now }

	if _, err := service.Create(context.Background(), "bank-a", "", "admin-1", ExportFormatCSV, Query{}); err != ErrExportTooLarge {
		t.Fatalf("expected ErrExportTooLarge, got %v", err)
	}
	if len(receipts.receipts) != 1 {
		t.Fatalf("expected one retained failure receipt, got %d", len(receipts.receipts))
	}
	for _, receipt := range receipts.receipts {
		if receipt.Status != ExportStatusFailed || receipt.FailureCode != "EXPORT_TOO_LARGE" {
			t.Fatalf("oversized export was not recorded as an explicit failure: %#v", receipt)
		}
	}
}
