package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/activity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

func TestAuditExportRequiresSeparatePermissionAndStreamsExactPopulation(t *testing.T) {
	now := time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC)
	activityService := activity.NewService(activity.NewMemoryRepository(
		activity.Event{TenantID: "bank-demo", ID: "visible", OccurredAt: now.Add(-time.Minute), EventType: "MATTER_CREATED", ObjectType: "MATTER", ObjectID: "matter-1", ActorKind: "PERSON", ActorID: "user-1", ActorDisplayName: "Ada Bello"},
		activity.Event{TenantID: "other-bank", ID: "hidden", OccurredAt: now.Add(-2 * time.Minute), EventType: "MATTER_CREATED", ObjectType: "MATTER", ObjectID: "matter-2"},
	))
	exports := activity.NewExportService(activityService, activity.NewMemoryExportRepository(), evidence.NewMemoryObjectStore())

	readerOnly := New(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:       identity.NewDevelopmentAuthenticator("bank-demo", "reader", "bank-ng", "CISO"),
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Activity:       activityService,
		AuditExports:   exports,
	})
	denied := httptest.NewRecorder()
	readerOnly.ServeHTTP(denied, httptest.NewRequest(http.MethodPost, "/api/v1/audit-exports", strings.NewReader(`{"format":"CSV"}`)))
	if denied.Code != http.StatusForbidden {
		t.Fatalf("audit export without AUDIT_EXPORT returned %d: %s", denied.Code, denied.Body.String())
	}

	admin := New(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:       identity.NewDevelopmentAuthenticator("bank-demo", "admin", "bank-ng", "GRC_ADMIN"),
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Activity:       activityService,
		AuditExports:   exports,
	})
	created := httptest.NewRecorder()
	admin.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/audit-exports", strings.NewReader(`{"format":"CSV","event_type":"MATTER_CREATED"}`)))
	if created.Code != http.StatusCreated {
		t.Fatalf("audit export returned %d: %s", created.Code, created.Body.String())
	}
	var receipt activity.ExportReceipt
	if err := json.Unmarshal(created.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != activity.ExportStatusReady || receipt.RowCount != 1 || receipt.RequestedBy != "admin" {
		t.Fatalf("unexpected audit export receipt: %#v", receipt)
	}

	downloaded := httptest.NewRecorder()
	admin.ServeHTTP(downloaded, httptest.NewRequest(http.MethodGet, "/api/v1/audit-exports/"+receipt.ID+"/download", nil))
	if downloaded.Code != http.StatusOK {
		t.Fatalf("audit export download returned %d: %s", downloaded.Code, downloaded.Body.String())
	}
	if downloaded.Header().Get("Content-Type") != "text/csv; charset=utf-8" || downloaded.Header().Get("Cache-Control") != "private, no-store" || downloaded.Header().Get("ETag") == "" {
		t.Fatalf("audit export download headers are incomplete: %#v", downloaded.Header())
	}
	if !strings.Contains(downloaded.Body.String(), "visible") || strings.Contains(downloaded.Body.String(), "hidden") {
		t.Fatalf("download leaked or omitted tenant activity: %s", downloaded.Body.String())
	}
}

func TestAuditExportRoutesUseBulkExportPermission(t *testing.T) {
	expected := map[string]string{
		http.MethodPost + " /api/v1/audit-exports":               identity.PermissionAuditExport,
		http.MethodGet + " /api/v1/audit-exports/{id}":          identity.PermissionAuditExport,
		http.MethodGet + " /api/v1/audit-exports/{id}/download": identity.PermissionAuditExport,
	}
	seen := map[string]bool{}
	for _, route := range (&API{}).productionRoutes() {
		key := route.Method + " " + route.Path
		permission, ok := expected[key]
		if !ok {
			continue
		}
		seen[key] = true
		if route.Permission != permission {
			t.Fatalf("%s permission = %q, want %q", key, route.Permission, permission)
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("missing governed audit export routes: %#v", seen)
	}
}
