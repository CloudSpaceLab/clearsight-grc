package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formpolicy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type httpPolicyFormReader struct {
	form evidence.DistributionFormRevision
}

func (reader httpPolicyFormReader) GetDistributionFormRevision(context.Context, string, string, string, int64) (evidence.DistributionFormRevision, error) {
	return reader.form, nil
}

type emptyPolicyResponses struct{}

func (emptyPolicyResponses) ListCompletedResponses(context.Context, evidence.CompletedResponseQuery) (evidence.CompletedResponsePage, error) {
	return evidence.CompletedResponsePage{}, nil
}

type unavailableFormPolicyAuthority struct{}

func (unavailableFormPolicyAuthority) Resolve(context.Context, authority.ResolveInput) (authority.Resolution, error) {
	return authority.Resolution{}, errors.New("authority storage unavailable")
}
func (unavailableFormPolicyAuthority) Simulate(context.Context, authority.ResolveInput) (authority.Simulation, error) {
	return authority.Simulation{}, errors.New("authority storage unavailable")
}
func (unavailableFormPolicyAuthority) Integrity(context.Context, string) ([]authority.IntegrityFinding, error) {
	return nil, errors.New("authority storage unavailable")
}
func (unavailableFormPolicyAuthority) Policies(context.Context, string) ([]authority.PolicySummary, error) {
	return nil, errors.New("authority storage unavailable")
}

func TestFormPolicyRoutesUseVerifiedGovernanceClasses(t *testing.T) {
	routes := (&API{}).formPolicyRoutes()
	want := map[string]routeClass{
		"GET /api/v1/forms/response-policies":                routeAuthenticatedRead,
		"POST /api/v1/forms/response-policies":               routeMaterialCommand,
		"GET /api/v1/forms/response-policies/{id}":           routeAuthenticatedRead,
		"POST /api/v1/forms/response-policies/{id}/simulate": routeMaterialCommand,
		"POST /api/v1/forms/response-policies/{id}/submit":   routeMaterialCommand,
		"POST /api/v1/forms/response-policies/{id}/approve":  routeMaterialCommand,
		"POST /api/v1/forms/response-policies/{id}/activate": routeMaterialCommand,
		"POST /api/v1/forms/response-policies/{id}/suspend":  routeMaterialCommand,
		"POST /api/v1/forms/response-policies/{id}/rollback": routeMaterialCommand,
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		class, exists := want[key]
		if !exists {
			continue
		}
		if route.Class != class {
			t.Fatalf("%s class = %s, want %s", key, route.Class, class)
		}
		if route.Permission != identity.PermissionConfigRead && route.Permission != identity.PermissionConfigWrite {
			t.Fatalf("%s lacks a configuration permission: %#v", key, route)
		}
		if class == routeMaterialCommand {
			if route.Command == nil || route.Command.Policy.ObjectType != "FORM_RESPONSE_POLICY" || route.Command.Policy.ActorField != noActorField || !route.Command.Policy.BindLegalEntity {
				t.Fatalf("%s is not bound to verified policy authority: %#v", key, route)
			}
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing form policy routes: %#v", want)
	}
}

func TestCreateFormPolicyIgnoresClientIdentityFields(t *testing.T) {
	handler := formPolicyHTTPHandler(t)
	input := validHTTPPolicyInput()
	payload, err := json.Marshal(struct {
		formpolicy.CreateInput
		TenantID      string `json:"tenant_id"`
		LegalEntityID string `json:"legal_entity_id"`
		MakerID       string `json:"maker_id"`
		CheckerID     string `json:"checker_id"`
		ActorID       string `json:"actor_id"`
	}{input, "foreign-bank", "foreign-entity", "forged-maker", "forged-checker", "forged-actor"})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/forms/response-policies", bytes.NewReader(payload)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	var policy formpolicy.Policy
	if err := json.NewDecoder(response.Body).Decode(&policy); err != nil {
		t.Fatal(err)
	}
	if policy.TenantID != "bank-a" || policy.LegalEntityID != "entity-a" || policy.MakerID != "maker-a" || policy.CheckerID != "" {
		t.Fatalf("policy trusted client identity fields: %#v", policy)
	}
}

func TestFormPolicyMakerCheckerFailureIsConflict(t *testing.T) {
	handler := formPolicyHTTPHandler(t)
	created := createHTTPPolicy(t, handler)
	simulated := policyAction(t, handler, created.ID, "simulate", "maker-a", map[string]any{"expected_version": created.RecordVersion})
	if simulated.Code != http.StatusOK {
		t.Fatalf("simulate: %d %s", simulated.Code, simulated.Body.String())
	}
	var receipt formpolicy.SimulationReceipt
	if err := json.NewDecoder(simulated.Body).Decode(&receipt); err != nil {
		t.Fatal(err)
	}
	submitted := policyAction(t, handler, created.ID, "submit", "maker-a", map[string]any{"expected_version": created.RecordVersion, "simulation_id": receipt.ID, "actor_id": "forged"})
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit: %d %s", submitted.Code, submitted.Body.String())
	}
	var pending formpolicy.Policy
	if err := json.NewDecoder(submitted.Body).Decode(&pending); err != nil {
		t.Fatal(err)
	}
	approved := policyAction(t, handler, created.ID, "approve", "maker-a", map[string]any{"expected_version": pending.RecordVersion, "simulation_id": receipt.ID, "checker_id": "someone-else"})
	if approved.Code != http.StatusConflict {
		t.Fatalf("maker approval = %d %s", approved.Code, approved.Body.String())
	}
	approved = policyAction(t, handler, created.ID, "approve", "checker-a", map[string]any{"expected_version": pending.RecordVersion, "simulation_id": receipt.ID, "checker_id": "forged-checker", "actor_id": "forged-actor"})
	if approved.Code != http.StatusOK {
		t.Fatalf("checker approval = %d %s", approved.Code, approved.Body.String())
	}
	var value formpolicy.Policy
	if err := json.NewDecoder(approved.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value.CheckerID != "checker-a" {
		t.Fatalf("approval trusted client checker: %#v", value)
	}
}

func TestFormPolicyCommandFailsClosedAfterAuthorityRevocation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(commandAuthorityStub{principal: "replacement-owner"}, commandauth.ModeEnforce, logger)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a", "GRC_ADMIN"), CommandGuard: guard,
		FormPolicies: newHTTPFormPolicyService(),
	})
	payload, _ := json.Marshal(validHTTPPolicyInput())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/forms/response-policies", bytes.NewReader(payload)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("revoked authority command = %d %s", response.Code, response.Body.String())
	}
	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/v1/forms/response-policies", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"items":[]`) {
		t.Fatalf("revoked command mutated policy state: %d %s", list.Code, list.Body.String())
	}
}

func TestFormPolicyCommandReturnsServiceUnavailableWhenAuthorityCannotBeChecked(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(unavailableFormPolicyAuthority{}, commandauth.ModeEnforce, logger)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a", "GRC_ADMIN"), CommandGuard: guard,
		FormPolicies: newHTTPFormPolicyService(),
	})
	payload, _ := json.Marshal(validHTTPPolicyInput())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/forms/response-policies", bytes.NewReader(payload)))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("authority failure = %d %s", response.Code, response.Body.String())
	}
}

func TestMissingFormPolicyCommandReturnsNotFoundBeforeAuthority(t *testing.T) {
	handler := formPolicyHTTPHandler(t)
	response := policyAction(t, handler, "missing", "simulate", "maker-a", map[string]any{"expected_version": 1})
	if response.Code != http.StatusNotFound {
		t.Fatalf("missing policy = %d %s", response.Code, response.Body.String())
	}
}

func formPolicyHTTPHandler(t *testing.T) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guard, err := commandauth.New(nil, commandauth.ModeOff, logger)
	if err != nil {
		t.Fatal(err)
	}
	return New(Dependencies{
		Logger: logger, Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker-a", "entity-a", "GRC_ADMIN"), CommandGuard: guard,
		FormPolicies: newHTTPFormPolicyService(),
	})
}

func newHTTPFormPolicyService() *formpolicy.Service {
	forms := httpPolicyFormReader{form: evidence.DistributionFormRevision{
		ID: "form-a", TenantID: "bank-a", LegalEntityID: "entity-a", Version: 3, Active: true,
		ScoringMode: formcontract.ScoringRisk, ScoreProfile: &formcontract.ScoreProfile{Version: "risk-v2", Mode: formcontract.ScoringRisk},
	}}
	return formpolicy.NewService(formpolicy.NewMemoryRepository(), forms, emptyPolicyResponses{})
}

func validHTTPPolicyInput() formpolicy.CreateInput {
	return formpolicy.CreateInput{
		Code: "poor-vendor-score", Name: "Poor vendor response", Purpose: "Create an issue when a current vendor response needs review.",
		AutomationPolicyID: "automation-a", AutomationPolicyVersion: 1,
		Eligibility: formpolicy.Eligibility{FormTemplateID: "form-a", FormTemplateVersion: 3, SubjectTypes: []string{"VENDOR"}, CurrentOnly: true, MinimumCoverage: 1, Bands: []formcontract.ConcernBand{formcontract.ConcernHigh, formcontract.ConcernCritical}},
		Action:      formpolicy.MatterAction{Type: "VENDOR_DEFICIENCY", Priority: 4, TitleTemplate: "Review {{form_title}} for {{subject_id}}", SummaryTemplate: "The response score is {{score}} with {{concern}} concern.", RequestedHandling: "Review the response and confirm the corrective action."},
		BlastRadius: formpolicy.BlastRadius{PerRun: 10, PerDay: 50},
		Outcome:     formpolicy.OutcomeContract{ExpectedOutcome: "The vendor response concern is resolved and verified.", CheckAfterMinutes: 1440, FailureResponse: "REOPEN"},
		Rollout:     formpolicy.RolloutShadow,
	}
}

func createHTTPPolicy(t *testing.T, handler http.Handler) formpolicy.Policy {
	t.Helper()
	payload, _ := json.Marshal(validHTTPPolicyInput())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/forms/response-policies", bytes.NewReader(payload)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", response.Code, response.Body.String())
	}
	var value formpolicy.Policy
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func policyAction(t *testing.T, handler http.Handler, id, action, actor string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	payload, _ := json.Marshal(body)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/forms/response-policies/"+id+"/"+action, bytes.NewReader(payload))
	request.Header.Set("X-ClearSight-Demo-Principal", actor)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
