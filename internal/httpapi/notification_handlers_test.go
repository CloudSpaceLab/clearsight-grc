package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func TestNotificationRoutesBindVerifiedActorAndReadState(t *testing.T) {
	at := time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)
	currentID := "10000000-0000-4000-8000-000000000001"
	otherActorID := "10000000-0000-4000-8000-000000000002"
	repo := workflow.NewMemoryRepositoryWithNotifications(nil, []workflow.InAppNotification{
		{ID: currentID, Kind: "MATTER_OWNER_ASSIGNED", Title: "Issue assigned to you", Summary: "Access review", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000001", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000001", OccurredAt: at, TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "person-a", OutboxEventID: currentID},
		{ID: otherActorID, Kind: "MATTER_OWNER_ASSIGNED", Title: "Other actor", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000002", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000002", OccurredAt: at.Add(time.Minute), TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "person-b", OutboxEventID: otherActorID},
		{ID: "10000000-0000-4000-8000-000000000003", Kind: "MATTER_OWNER_ASSIGNED", Title: "Sibling entity", SubjectType: "MATTER", SubjectID: "20000000-0000-4000-8000-000000000003", ActionPath: "#work/matters/20000000-0000-4000-8000-000000000003", OccurredAt: at.Add(2 * time.Minute), TenantID: "bank", LegalEntityID: "entity-b", PrincipalID: "person-a", OutboxEventID: "10000000-0000-4000-8000-000000000003"},
	})
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank", "person-a", "entity-a"),
		Workflow: workflow.NewService(repo),
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/notifications?limit=10", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", response.Code, response.Body.String())
	}
	var page workflow.NotificationPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != currentID || page.UnreadCount != 1 {
		t.Fatalf("actor-scoped page = %#v", page)
	}

	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+currentID+"/read", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", read.Code, read.Body.String())
	}

	unread := httptest.NewRecorder()
	handler.ServeHTTP(unread, httptest.NewRequest(http.MethodGet, "/api/v1/notifications?unread_only=true", nil))
	if unread.Code != http.StatusOK {
		t.Fatalf("unread status=%d body=%s", unread.Code, unread.Body.String())
	}
	if err := json.Unmarshal(unread.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.UnreadCount != 0 || len(page.Items) != 0 {
		t.Fatalf("unread page = %#v", page)
	}

	crossActor := httptest.NewRecorder()
	handler.ServeHTTP(crossActor, httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+otherActorID+"/read", nil))
	if crossActor.Code != http.StatusNotFound {
		t.Fatalf("cross-actor read status=%d body=%s", crossActor.Code, crossActor.Body.String())
	}
}

func TestNotificationRoutesRejectInvalidFilters(t *testing.T) {
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank", "person-a", "entity-a"),
		Workflow: workflow.NewService(workflow.NewMemoryRepository(nil)),
	})
	for _, target := range []string{
		"/api/v1/notifications?limit=101",
		"/api/v1/notifications?unread_only=yes",
		"/api/v1/notifications?cursor=not-a-cursor",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", target, response.Code, response.Body.String())
		}
	}
}
