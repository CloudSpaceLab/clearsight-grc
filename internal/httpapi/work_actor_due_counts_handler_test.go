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

func TestActorWorkCountsEndpointReturnsVerifiedSourcePopulation(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	late := base.Add(-time.Hour)
	soon := base.Add(24 * time.Hour)
	far := base.Add(14 * 24 * time.Hour)
	makeTask := func(id string, due *time.Time) workflow.Task {
		return workflow.Task{
			ID: id, TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor",
			WorkflowKind: workflow.MatterActionWorkflowKind, MatterID: id,
			MatterScope: json.RawMessage(`{"access":"INTERNAL"}`),
			Title: "Review issue", Status: workflow.StatusReady,
			DueAt: due, UpdatedAt: base,
		}
	}
	items := []workflow.Task{
		makeTask("late", &late), makeTask("soon", &soon),
		makeTask("later", &far), makeTask("undated", nil),
	}
	other := makeTask("other", &late)
	other.LegalEntityID = "another"
	items = append(items, other)
	service := workflow.NewService(workflow.NewMemoryRepository(items))
	api := &API{deps: Dependencies{Workflow: service}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor"}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/work/assignment-counts?tenant_id=bank", nil)
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response := httptest.NewRecorder()
	api.actorWorkDueCounts(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("counts HTTP %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Total, Overdue, DueSoon, Undated int64
		Coverage, CountBasis             string
		AsOf                             time.Time
		Scope                            struct {
			TenantID      string `json:"tenant_id"`
			LegalEntityID string `json:"legal_entity_id"`
			PrincipalID   string `json:"principal_id"`
		} `json:"scope"`
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for key := range map[string]bool{"approvals": true, "evidence_needed": true, "completed": true} {
		if _, exists := raw[key]; exists {
			t.Fatalf("unsupported counts cannot be represented as zeros: %s", key)
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	// Explicit decode prevents default Go field-name matching hiding wire errors.
	for name, want := range map[string]int64{"total": 4, "overdue": 1, "due_soon": 1, "undated": 1} {
		var got int64
		if err := json.Unmarshal(raw[name], &got); err != nil || got != want {
			t.Fatalf("count %s: got %d want %d, error %v", name, got, want, err)
		}
	}
	if body.Scope.TenantID != "bank" || body.Scope.LegalEntityID != "entity" ||
		body.Scope.PrincipalID != "actor" || body.AsOf.IsZero() {
		t.Fatalf("verified actor scope/as-of missing: %#v", body)
	}
	if string(raw["coverage"]) != `"WORKFLOW_ASSIGNMENTS_ONLY"` ||
		string(raw["count_basis"]) != `"SOURCE_SNAPSHOT"` {
		t.Fatalf("counts cannot claim enterprise completeness: %s", response.Body.String())
	}
}

func TestActorWorkCountsEndpointRejectsUnsupportedAndCrossScopeQueries(t *testing.T) {
	api := &API{deps: Dependencies{Workflow: workflow.NewService(workflow.NewMemoryRepository(nil))}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "actor"}
	cases := []struct {
		path string
		status int
	}{
		{"/api/v1/work/assignment-counts?tenant_id=other", http.StatusForbidden},
		{"/api/v1/work/assignment-counts?principal_id=other", http.StatusForbidden},
		{"/api/v1/work/assignment-counts?legal_entity_id=other", http.StatusForbidden},
		{"/api/v1/work/assignment-counts?organization_scope_id=branch", http.StatusBadRequest},
		{"/api/v1/work/assignment-counts?status=COMPLETED", http.StatusBadRequest},
		{"/api/v1/work/assignment-counts?tenant_id=bank&tenant_id=bank", http.StatusBadRequest},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodGet, tc.path, nil)
		request = request.WithContext(identity.WithActor(request.Context(), actor))
		response := httptest.NewRecorder()
		api.actorWorkDueCounts(response, request)
		if response.Code != tc.status {
			t.Errorf("%s: got %d want %d", tc.path, response.Code, tc.status)
		}
	}
	response := httptest.NewRecorder()
	api.actorWorkDueCounts(response, httptest.NewRequest(http.MethodGet, "/api/v1/work/assignment-counts", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing actor must be unauthorized, got %d", response.Code)
	}
}
