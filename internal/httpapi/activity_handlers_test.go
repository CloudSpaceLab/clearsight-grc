package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/activity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestSystemActivityUsesVerifiedTenantAndPlatformPermission(t *testing.T) {
	now := time.Date(2026, 9, 3, 10, 30, 0, 0, time.UTC)
	service := activity.NewService(activity.NewMemoryRepository(
		activity.Event{TenantID: "bank-demo", ID: "visible", OccurredAt: now, EventType: "MATTER_STATE_CHANGED", ObjectType: "MATTER", ObjectID: "matter-1"},
		activity.Event{TenantID: "other-bank", ID: "hidden", OccurredAt: now.Add(-time.Minute), EventType: "MATTER_CREATED", ObjectType: "MATTER", ObjectID: "matter-2"},
	))

	withoutPermission := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "user", "bank-ng"),
		Activity: service,
	})
	response := httptest.NewRecorder()
	withoutPermission.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/system-activity", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("activity without permission returned %d: %s", response.Code, response.Body.String())
	}

	admin := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "admin", "bank-ng", "SYSTEM_ADMIN"),
		Activity: service,
	})
	response = httptest.NewRecorder()
	admin.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/system-activity?tenant_id=other-bank&limit=20", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("activity returned %d: %s", response.Code, response.Body.String())
	}
	var page activity.Page
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "visible" {
		t.Fatalf("verified tenant was not authoritative: %#v", page.Items)
	}
}

func TestSystemActivityRejectsInvalidDateRange(t *testing.T) {
	service := activity.NewService(activity.NewMemoryRepository())
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "admin", "bank-ng", "SYSTEM_ADMIN"),
		Activity: service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/system-activity?from=2026-09-04&to=2026-09-03", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid range returned %d: %s", response.Code, response.Body.String())
	}
}
