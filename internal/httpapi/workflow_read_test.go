package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func TestWorkflowTaskReadRejectsCrossPrincipalAndUnsupportedWorkButKeepsExactAssignments(t *testing.T) {
	allowed := json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["actor-1"]}`)
	blocked := json.RawMessage(`{"access":"RESTRICTED","allowed_principal_ids":["actor-2"]}`)
	now := time.Date(2026, 8, 7, 20, 0, 0, 0, time.UTC)
	service := workflow.NewService(workflow.NewMemoryRepository([]workflow.Task{
		{
			ID: "visible", TenantID: "bank", PrincipalID: "actor-1", WorkflowKind: workflow.MatterActionWorkflowKind,
			MatterID: "matter-visible", MatterScope: allowed, Status: workflow.StatusReady, Title: "Visible action", UpdatedAt: now,
		},
		{
			ID: "blocked", TenantID: "bank", PrincipalID: "actor-1", WorkflowKind: workflow.MatterActionWorkflowKind,
			MatterID: "matter-blocked", MatterScope: blocked, Status: workflow.StatusReady, Title: "Protected action", UpdatedAt: now.Add(time.Hour),
		},
		{
			ID: "legacy", TenantID: "bank", PrincipalID: "actor-1", WorkflowKind: "REVIEW",
			MatterID: "matter-visible", MatterScope: allowed, Status: workflow.StatusReady, Title: "Legacy task", UpdatedAt: now.Add(2 * time.Hour),
		},
		{
			ID: "other", TenantID: "bank", PrincipalID: "actor-2", WorkflowKind: workflow.MatterActionWorkflowKind,
			MatterID: "matter-other", MatterScope: blocked, Status: workflow.StatusReady, Title: "Other actor work", UpdatedAt: now.Add(3 * time.Hour),
		},
	}))
	api := &API{deps: Dependencies{Workflow: service}}
	actor := identity.Actor{TenantID: "bank", PrincipalID: "actor-1", LegalEntityID: "*"}

	cross := httptest.NewRequest(http.MethodGet, "/api/v1/workflow/tasks?tenant_id=bank&principal_id=actor-2", nil)
	cross = cross.WithContext(identity.WithActor(cross.Context(), actor))
	crossRecorder := httptest.NewRecorder()
	api.listWorkflowTasks(crossRecorder, cross)
	if crossRecorder.Code != http.StatusForbidden {
		t.Fatalf("expected cross-principal read to be forbidden, got %d: %s", crossRecorder.Code, crossRecorder.Body.String())
	}

	// Both Matter rows are exact current assignments, including the restricted
	// Matter. The unsupported row is newer and must be filtered before LIMIT.
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workflow/tasks?tenant_id=bank&limit=1", nil)
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	recorder := httptest.NewRecorder()
	api.listWorkflowTasks(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected actor-scoped task read, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Items []workflow.Task `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "blocked" {
		t.Fatalf("expected the newest exact supported assignment, got %#v", body.Items)
	}
}
