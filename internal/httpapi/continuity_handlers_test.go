package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func continuityTestHandler() http.Handler {
	return New(Dependencies{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AllowedOrigin: "http://localhost:5173",
		Mode:          "test-memory",
		Identity:      identity.NewDevelopmentAuthenticator("bank", "role-cro", "bank-ng"),
		Continuity:    continuity.NewService(continuity.NewMemoryRepository()),
	})
}

func TestProgramAPIUsesHumanStatusLabels(t *testing.T) {
	handler := continuityTestHandler()
	payload := []byte(`{"tenant_id":"bank","code":"NDPA","name":"Data protection","type":"PRIVACY","owning_function":"Privacy Office","owner_principal_id":"owner","authority_principal_id":"approver","scope":{"entity":"Bank NG"},"effective_from":"2026-08-05T10:00:00Z","effective_until":"2027-08-05T10:00:00Z","actor_id":"owner"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var program continuity.ProgramAggregate
	if err := json.NewDecoder(response.Body).Decode(&program); err != nil {
		t.Fatal(err)
	}
	if program.StateLabel != "Setup in progress" || program.Program.Status != continuity.ProgramDraft || program.Program.EffectiveUntil == nil {
		t.Fatalf("unexpected program label/status/period %#v", program)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/programs?tenant_id=bank", nil))
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"state_label":"Setup in progress"`)) {
		t.Fatalf("expected human status in list, got %d: %s", list.Code, list.Body.String())
	}
}

func TestMatterAPIReportsVersionConflictInPlainLanguage(t *testing.T) {
	handler := continuityTestHandler()
	payload := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":3,"title":"Complete missing owner approvals","summary":"Four privileged accounts need current owner approval.","scope":{},"known_facts":{"accounts":4},"missing_facts":[],"contradictions":[]}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(payload)))
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var matter continuity.MatterAggregate
	if err := json.NewDecoder(response.Body).Decode(&matter); err != nil {
		t.Fatal(err)
	}
	if matter.TypeLabel != "Control gap" || matter.StatusLabel != "Draft" || matter.NextAction != "Start initial review" {
		t.Fatalf("unexpected labels %#v", matter)
	}

	transition := []byte(`{"tenant_id":"bank","expected_version":999,"to":"TRIAGE","rationale":"Start review."}`)
	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/transition", bytes.NewReader(transition)))
	if conflict.Code != http.StatusConflict || !bytes.Contains(conflict.Body.Bytes(), []byte("This record changed")) {
		t.Fatalf("expected plain version conflict, got %d: %s", conflict.Code, conflict.Body.String())
	}
}

func TestMatterLinkAPIAddsASecondProgramAndIsIdempotent(t *testing.T) {
	handler := continuityTestHandler()
	createProgram := func(code, name string) continuity.ProgramAggregate {
		body := []byte(`{"tenant_id":"bank","code":"` + code + `","name":"` + name + `","type":"ASSURANCE","owning_function":"Control Assurance","scope":{},"effective_from":"2026-08-05T10:00:00Z"}`)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(body)))
		if response.Code != http.StatusCreated {
			t.Fatalf("create program: expected 201, got %d: %s", response.Code, response.Body.String())
		}
		var value continuity.ProgramAggregate
		if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
			t.Fatal(err)
		}
		return value
	}

	firstProgram := createProgram("ACCESS", "Access governance")
	secondProgram := createProgram("CYBER", "Cybersecurity oversight")
	matterPayload := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Privileged account approvals are incomplete","summary":"Four accounts do not have current owner approval.","scope":{},"known_facts":{"affected_accounts":4},"missing_facts":[],"contradictions":[],"program_id":"` + firstProgram.Program.ID + `"}`)
	matterResponse := httptest.NewRecorder()
	handler.ServeHTTP(matterResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(matterPayload)))
	if matterResponse.Code != http.StatusCreated {
		t.Fatalf("create matter: expected 201, got %d: %s", matterResponse.Code, matterResponse.Body.String())
	}
	var matter continuity.MatterAggregate
	if err := json.NewDecoder(matterResponse.Body).Decode(&matter); err != nil {
		t.Fatal(err)
	}

	linkPayload := func(version int64) []byte {
		return []byte(fmt.Sprintf(`{"tenant_id":"bank","expected_version":%d,"program_id":"%s","relationship":"AFFECTS"}`, version, secondProgram.Program.ID))
	}
	linkResponse := httptest.NewRecorder()
	handler.ServeHTTP(linkResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/links", bytes.NewReader(linkPayload(matter.Matter.Version))))
	if linkResponse.Code != http.StatusCreated {
		t.Fatalf("link matter: expected 201, got %d: %s", linkResponse.Code, linkResponse.Body.String())
	}
	var linked continuity.MatterAggregate
	if err := json.NewDecoder(linkResponse.Body).Decode(&linked); err != nil {
		t.Fatal(err)
	}
	if len(linked.Links) != 2 {
		t.Fatalf("expected two program links, got %#v", linked.Links)
	}

	duplicateResponse := httptest.NewRecorder()
	handler.ServeHTTP(duplicateResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters/"+matter.Matter.ID+"/links", bytes.NewReader(linkPayload(linked.Matter.Version))))
	if duplicateResponse.Code != http.StatusCreated {
		t.Fatalf("duplicate link should return the existing aggregate, got %d: %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
	var duplicate continuity.MatterAggregate
	if err := json.NewDecoder(duplicateResponse.Body).Decode(&duplicate); err != nil {
		t.Fatal(err)
	}
	if len(duplicate.Links) != 2 {
		t.Fatalf("duplicate link created another relationship: %#v", duplicate.Links)
	}
}

func TestProgramHistorySupportsPointInTime(t *testing.T) {
	handler := continuityTestHandler()
	payload := []byte(`{"tenant_id":"bank","code":"TIME","name":"Time travel","type":"ASSURANCE","owning_function":"Control Assurance","scope":{},"effective_from":"2026-08-05T10:00:00Z"}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(payload)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	var program continuity.ProgramAggregate
	if err := json.NewDecoder(response.Body).Decode(&program); err != nil {
		t.Fatal(err)
	}

	history := httptest.NewRecorder()
	handler.ServeHTTP(history, httptest.NewRequest(http.MethodGet, "/api/v1/programs/"+program.Program.ID+"/history?tenant_id=bank&at=2026-08-05T09:59:00Z", nil))
	if history.Code != http.StatusNotFound {
		t.Fatalf("expected no aggregate before creation, got %d: %s", history.Code, history.Body.String())
	}
}
