package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestMonitoringRoutesAreRegisteredOnce(t *testing.T) {
	routes := (&API{}).routes()
	want := map[string]bool{
		"GET /api/v1/form-templates":                            false,
		"POST /api/v1/form-templates":                           false,
		"POST /api/v1/form-templates/{id}/transition":           false,
		"POST /api/v1/form-templates/{id}/collections":          false,
		"GET /api/v1/programs/{id}/monitoring-checks":           false,
		"POST /api/v1/programs/{id}/monitoring-checks":          false,
		"POST /api/v1/monitoring-checks/{id}/transition":        false,
		"POST /api/v1/monitoring-checks/{id}/collection-policy": false,
		"POST /api/v1/monitoring-checks/{id}/evaluate-source":   false,
		"GET /api/v1/monitoring-checks/{id}/results":            false,
		"GET /api/v1/programs/{id}/collection-summaries":        false,
	}
	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, expected := want[key]; expected {
			if want[key] {
				t.Fatalf("duplicate route %s", key)
			}
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Errorf("missing route %s", route)
		}
	}
}

func TestUpdateCollectionPolicyUsesVerifiedIdentityAndDefaults(t *testing.T) {
	repo := monitoring.NewMemoryRepository()
	createdAt := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)
	check, err := repo.CreateCheckRevision(t.Context(), monitoring.MonitoringCheck{
		ID: "check-form", TenantID: "bank-a", ProgramID: "program-a", Code: "VENDOR_CERT", Name: "Vendor certifications", Claim: "Current certifications are supplied.",
		InputKind: monitoring.InputForm, FormTemplateID: "form-a", FormTemplateVersion: 1,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1, CreatedBy: "approver", CreatedAt: createdAt, UpdatedAt: createdAt},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", "policy-maker", "entity-a"),
		Monitoring: monitoring.NewService(repo, nil),
	})
	body := []byte(`{"tenant_id":"bank-b","actor_id":"forged","expected_version":1,"collection_policy":{"validity_months":12}}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/monitoring-checks/"+check.ID+"/collection-policy", bytes.NewReader(body)))
	if response.Code != http.StatusOK {
		t.Fatalf("update policy returned %d: %s", response.Code, response.Body.String())
	}
	var revised monitoring.MonitoringCheck
	if err := json.Unmarshal(response.Body.Bytes(), &revised); err != nil {
		t.Fatal(err)
	}
	if revised.TenantID != "bank-a" || revised.CreatedBy != "policy-maker" || revised.Version != 2 || revised.Status != monitoring.LifecycleDraft {
		t.Fatalf("revised check = %#v", revised)
	}
	if revised.CollectionPolicy == nil || revised.CollectionPolicy.RenewalWindowDays != 30 || revised.CollectionPolicy.ReminderCount != 3 {
		t.Fatalf("defaulted policy = %#v", revised.CollectionPolicy)
	}

	stale := httptest.NewRecorder()
	handler.ServeHTTP(stale, httptest.NewRequest(http.MethodPost, "/api/v1/monitoring-checks/"+check.ID+"/collection-policy", bytes.NewReader(body)))
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale update returned %d: %s", stale.Code, stale.Body.String())
	}
}

func TestUpdateCollectionPolicyRejectsSourceAndCrossTenantChecks(t *testing.T) {
	repo := monitoring.NewMemoryRepository()
	now := time.Now().UTC()
	_, err := repo.CreateCheckRevision(t.Context(), monitoring.MonitoringCheck{
		ID: "source-check", TenantID: "bank-a", ProgramID: "program-a", Code: "SOURCE", Name: "Source check", Claim: "Source remains current.", InputKind: monitoring.InputSource,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1, CreatedBy: "maker", CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateCheckRevision(t.Context(), monitoring.MonitoringCheck{
		ID: "foreign-check", TenantID: "bank-b", ProgramID: "program-b", Code: "FOREIGN", Name: "Foreign form", Claim: "Foreign claim.", InputKind: monitoring.InputForm,
		Lifecycle: monitoring.Lifecycle{Status: monitoring.LifecycleActive, IsCurrent: true, Version: 1, CreatedBy: "maker", CreatedAt: now, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", "maker", "entity-a"), Monitoring: monitoring.NewService(repo, nil)})
	body := []byte(`{"expected_version":1,"collection_policy":{"validity_months":12}}`)

	source := httptest.NewRecorder()
	handler.ServeHTTP(source, httptest.NewRequest(http.MethodPost, "/api/v1/monitoring-checks/source-check/collection-policy", bytes.NewReader(body)))
	if source.Code != http.StatusUnprocessableEntity {
		t.Fatalf("source policy returned %d: %s", source.Code, source.Body.String())
	}
	foreign := httptest.NewRecorder()
	handler.ServeHTTP(foreign, httptest.NewRequest(http.MethodPost, "/api/v1/monitoring-checks/foreign-check/collection-policy", bytes.NewReader(body)))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant policy returned %d: %s", foreign.Code, foreign.Body.String())
	}
}

func TestCollectionSummaryReturnsBoundedSafeFreshnessState(t *testing.T) {
	monitoringRepo := monitoring.NewMemoryRepository()
	programs := continuity.NewService(continuity.NewMemoryRepository())
	program, err := programs.CreateProgram(t.Context(), continuity.CreateProgramInput{
		TenantID: "bank-a", Code: "VENDOR", Name: "Vendor assurance", Type: "THIRD_PARTY", OwningFunction: "Procurement", Scope: json.RawMessage(`{}`), EffectiveFrom: time.Now().UTC(), ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	submittedAt := now.AddDate(0, -11, 0)
	cycle, err := monitoringRepo.UpsertCollectionCycle(t.Context(), monitoring.CollectionCycle{
		ID: "cycle-a", TenantID: "bank-a", ProgramID: program.Program.ID, MonitoringCheckID: "check-a", MonitoringCheckVersion: 4, Sequence: 2,
		Policy: monitoring.CollectionPolicy{ValidityMonths: 12, RenewalWindowDays: 30, ReminderCount: 5}, CurrentRequestID: "request-a", LatestSubmissionID: "submission-a", LatestSubmittedAt: &submittedAt,
		LatestRespondentLabel: "Vendor assurance owner", ExpiresAt: now.AddDate(0, 1, 0), RenewalOpensAt: now.AddDate(0, 1, -30), NextActionAt: ptrTime(now.Add(24 * time.Hour)),
		Recipient: monitoring.RecipientRoute{Type: monitoring.RouteExternalContact, ContactRef: "opaque-contact", SafeHint: "v***@supplier.example"}, DeliveryState: monitoring.DeliveryDelivered,
		State: monitoring.CycleAwaitingResponse, RemindersSent: 2, CreatedAt: now.Add(-time.Hour), UpdatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank-a", "reader", "entity-a"),
		Monitoring: monitoring.NewService(monitoringRepo, nil), Continuity: programs,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/programs/"+program.Program.ID+"/collection-summaries?limit=500", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("summary returned %d: %s", response.Code, response.Body.String())
	}
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("summary items = %#v", payload.Items)
	}
	item := payload.Items[0]
	if item["monitoring_check_id"] != cycle.MonitoringCheckID || item["respondent_label"] != "Vendor assurance owner" || item["recipient_hint"] != "v***@supplier.example" || item["currency_state"] != "AWAITING_RESPONSE" {
		t.Fatalf("summary item = %#v", item)
	}
	if item["projection_source_version"] != float64(4) || item["reminders_sent"] != float64(2) || item["reminder_count"] != float64(5) || item["active_request_deadline"] == nil {
		t.Fatalf("summary progress = %#v", item)
	}
	if _, leaked := item["recipient_contact_ref"]; leaked {
		t.Fatalf("summary leaked opaque contact: %#v", item)
	}

	foreignProgram, err := programs.CreateProgram(t.Context(), continuity.CreateProgramInput{
		TenantID: "bank-b", Code: "FOREIGN", Name: "Foreign vendor assurance", Type: "THIRD_PARTY", OwningFunction: "Procurement", Scope: json.RawMessage(`{}`), EffectiveFrom: now, ActorID: "owner",
	})
	if err != nil {
		t.Fatal(err)
	}
	foreign := httptest.NewRecorder()
	handler.ServeHTTP(foreign, httptest.NewRequest(http.MethodGet, "/api/v1/programs/"+foreignProgram.Program.ID+"/collection-summaries", nil))
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant summary returned %d: %s", foreign.Code, foreign.Body.String())
	}
}

func ptrTime(value time.Time) *time.Time { return &value }

func TestCreateFormTemplateUsesVerifiedIdentity(t *testing.T) {
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	handler := New(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:   identity.NewDevelopmentAuthenticator("bank-a", "maker", "entity-a"),
		Monitoring: service,
	})
	body := []byte(`{"tenant_id":"bank-b","created_by":"other","code":"RESET","name":"Password reset review","purpose":"Review password reset safeguards.","fields":[{"id":"secure","label":"Identity checks completed","type":"single_select","required":true,"options":["Yes","No"],"scoring":{"weight":1,"answer_scores":{"Yes":0,"No":100},"critical_answers":["No"]}}]}`)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/form-templates", bytes.NewReader(body)))
	if response.Code != http.StatusCreated {
		t.Fatalf("create form returned %d: %s", response.Code, response.Body.String())
	}
	var form monitoring.FormTemplate
	if err := json.Unmarshal(response.Body.Bytes(), &form); err != nil {
		t.Fatal(err)
	}
	if form.TenantID != "bank-a" || form.CreatedBy != "maker" || form.Status != monitoring.LifecycleDraft || form.Fields[0].Scoring.ID != "secure" {
		t.Fatalf("created form = %#v", form)
	}
}

func TestMonitoringListCannotCrossTenant(t *testing.T) {
	service := monitoring.NewService(monitoring.NewMemoryRepository(), nil)
	handler := New(Dependencies{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:   identity.NewDevelopmentAuthenticator("bank-a", "maker", "entity-a"),
		Monitoring: service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/form-templates?tenant_id=bank-b", nil))
	if response.Code == http.StatusOK {
		t.Fatalf("cross-tenant list returned %d: %s", response.Code, response.Body.String())
	}
}
