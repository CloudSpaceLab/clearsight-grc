package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func TestActorWorkEndpointPagesOverFiftyWithoutChangingToday(t *testing.T) {
	base := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	due := base.Add(96 * time.Hour)
	tasks := make([]workflow.Task, 0, 74)
	for i := 0; i < 72; i++ {
		tasks = append(tasks, workflow.Task{
			ID:       fmt.Sprintf("00000000-0000-4000-8000-%012x", i+1),
			TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor",
			WorkflowKind: workflow.MatterActionWorkflowKind,
			MatterID:     fmt.Sprintf("00000000-0000-4000-8000-%012x", i+200),
			MatterScope:  json.RawMessage(`{"access":"RESTRICTED"}`),
			Title:        fmt.Sprintf("Review evidence %03d", i), Status: workflow.StatusReady,
			DueAt: &due, UpdatedAt: base.Add(time.Duration(i) * time.Minute),
		})
	}
	other := tasks[0]
	other.ID = "00000000-0000-4000-8000-000000009999"
	other.LegalEntityID = "other"
	tasks = append(tasks, other)
	api := &API{deps: Dependencies{Workflow: workflow.NewService(workflow.NewMemoryRepository(tasks))}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor"}
	seen := map[string]bool{}
	cursor := ""
	for {
		path := "/api/v1/work/assignments?limit=25"
		if cursor != "" {
			path += "&cursor=" + url.QueryEscape(cursor)
		}
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request = request.WithContext(identity.WithActor(request.Context(), actor))
		response := httptest.NewRecorder()
		api.actorWorkAssignments(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("page HTTP %d: %s", response.Code, response.Body.String())
		}
		var page struct {
			Items      []today.AttentionItem `json:"items"`
			NextCursor string                `json:"next_cursor"`
			HasMore    bool                  `json:"has_more"`
			Coverage   string                `json:"coverage"`
			CountBasis string                `json:"count_basis"`
			Scope      struct {
				TenantID      string `json:"tenant_id"`
				LegalEntityID string `json:"legal_entity_id"`
				PrincipalID   string `json:"principal_id"`
			} `json:"scope"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
			t.Fatal(err)
		}
		if page.CountBasis != "NOT_AVAILABLE" || page.Coverage != "WORKFLOW_ASSIGNMENTS_ONLY" {
			t.Fatalf("endpoint must not claim exact totals: %#v", page)
		}
		if page.Scope.TenantID != actor.TenantID ||
			page.Scope.LegalEntityID != actor.LegalEntityID ||
			page.Scope.PrincipalID != actor.PrincipalID {
			t.Fatalf("page scope must match the verified actor: %#v", page.Scope)
		}
		for _, item := range page.Items {
			if seen[item.ID] {
				t.Fatalf("duplicated assignment across pages: %s", item.ID)
			}
			seen[item.ID] = true
		}
		if !page.HasMore {
			if len(seen) != 72 || page.NextCursor != "" {
				t.Fatalf("missing/extra actor assignments: count=%d next=%q", len(seen), page.NextCursor)
			}
			return
		}
		if page.NextCursor == "" {
			t.Fatal("continuation cursor missing")
		}
		cursor = page.NextCursor
	}
}

func TestActorWorkEndpointRejectsWideningAndUnsupportedFilters(t *testing.T) {
	api := &API{deps: Dependencies{Workflow: workflow.NewService(workflow.NewMemoryRepository(nil))}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor"}
	cases := []struct {
		path   string
		status int
	}{
		{"/api/v1/work/assignments?tenant_id=another", http.StatusForbidden},
		{"/api/v1/work/assignments?principal_id=another", http.StatusForbidden},
		{"/api/v1/work/assignments?legal_entity_id=other", http.StatusForbidden},
		{"/api/v1/work/assignments?organization_scope_id=branch", http.StatusBadRequest},
		{"/api/v1/work/assignments?group_id=group", http.StatusBadRequest},
		{"/api/v1/work/assignments?branch_id=branch", http.StatusBadRequest},
		{"/api/v1/work/assignments?team_queue=true", http.StatusBadRequest},
		{"/api/v1/work/assignments?q=one&q=two", http.StatusBadRequest},
		{"/api/v1/work/assignments?status=READY&status=BLOCKED", http.StatusBadRequest},
		{"/api/v1/work/assignments?limit=25&limit=50", http.StatusBadRequest},
		{"/api/v1/work/assignments?q=bad%00title", http.StatusBadRequest},
		{"/api/v1/work/assignments?status=COMPLETED", http.StatusBadRequest},
		{"/api/v1/work/assignments?limit=201", http.StatusBadRequest},
		{"/api/v1/work/assignments?cursor=invalid", http.StatusBadRequest},
		{"/api/v1/work/assignments?due_from=yesterday", http.StatusBadRequest},
		{"/api/v1/work/assignments?due_from=2026-10-23T00:00:00Z&due_before=2026-10-21T00:00:00Z", http.StatusBadRequest},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		request = request.WithContext(identity.WithActor(request.Context(), actor))
		response := httptest.NewRecorder()
		api.actorWorkAssignments(response, request)
		if response.Code != tc.status {
			t.Errorf("%s returned %d, want %d: %s", tc.path, response.Code, tc.status, response.Body.String())
		}
	}
	actor.LegalEntityID = ""
	request := httptest.NewRequest(http.MethodGet, "/api/v1/work/assignments", nil)
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response := httptest.NewRecorder()
	api.actorWorkAssignments(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing verified entity returned %d: %s", response.Code, response.Body.String())
	}
}
