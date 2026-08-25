package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func TestProgramSummaryEndpointIsBoundedAndCursorBased(t *testing.T) {
	handler := continuityTestHandler()
	for index, code := range []string{"ALPHA", "BETA"} {
		body := []byte(`{"tenant_id":"bank","code":"` + code + `","name":"Program ` + code + `","type":"ASSURANCE","owning_function":"Control Assurance","scope":{},"effective_from":"2026-08-05T10:00:00Z"}`)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(body)))
		if response.Code != http.StatusCreated {
			t.Fatalf("create program %d: %d %s", index, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/program-summaries?tenant_id=bank&limit=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", response.Code, response.Body.String())
	}
	var page continuity.ProgramSummaryPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("expected bounded page and cursor: %#v", page)
	}
	if page.Items[0].Program.Name == "" || page.Items[0].StateLabel == "" {
		t.Fatalf("summary lacks operating labels: %#v", page.Items[0])
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/api/v1/program-summaries?tenant_id=bank&cursor=invalid", nil))
	if invalid.Code != http.StatusBadRequest || !bytes.Contains(invalid.Body.Bytes(), []byte("page cursor")) {
		t.Fatalf("expected plain invalid cursor response, got %d %s", invalid.Code, invalid.Body.String())
	}
}

func TestMatterSummaryEndpointUsesOperationalLabels(t *testing.T) {
	handler := continuityTestHandler()
	body := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Confirm access review owners","summary":"Four accounts need a current owner.","scope":{},"known_facts":{"accounts":4},"missing_facts":[],"contradictions":[]}`)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(body)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create matter: %d %s", created.Code, created.Body.String())
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/matter-summaries?tenant_id=bank&status=OPEN", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", response.Code, response.Body.String())
	}
	var page continuity.MatterSummaryPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].TypeLabel != "Control gap" || page.Items[0].NextAction != "Start initial review" {
		t.Fatalf("unexpected matter summary: %#v", page.Items)
	}
}

func TestMatterSummaryEndpointAcceptsExactProgramFilter(t *testing.T) {
	handler := continuityTestHandler()
	programBody := []byte(`{"tenant_id":"bank","code":"FILTER","name":"Filtered Program","type":"ASSURANCE","owning_function":"Control Assurance","scope":{},"effective_from":"2026-08-05T10:00:00Z"}`)
	programResponse := httptest.NewRecorder()
	handler.ServeHTTP(programResponse, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(programBody)))
	var program continuity.ProgramAggregate
	if err := json.NewDecoder(programResponse.Body).Decode(&program); err != nil {
		t.Fatal(err)
	}
	linkedBody := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Linked issue","summary":"This issue belongs to the Program.","scope":{},"known_facts":{},"missing_facts":[],"contradictions":[],"program_id":"` + program.Program.ID + `"}`)
	linkedResponse := httptest.NewRecorder()
	handler.ServeHTTP(linkedResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(linkedBody)))
	unlinkedBody := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Other issue","summary":"This issue is unrelated.","scope":{},"known_facts":{},"missing_facts":[],"contradictions":[]}`)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(unlinkedBody)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/matter-summaries?tenant_id=bank&status=OPEN&program_id="+program.Program.ID, nil))
	var page continuity.MatterSummaryPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Matter.Title != "Linked issue" {
		t.Fatalf("unexpected filtered matters: %#v", page.Items)
	}
}
