package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestMonitoringRoutesAreRegisteredOnce(t *testing.T) {
	routes := (&API{}).routes()
	want := map[string]bool{
		"GET /api/v1/form-templates":                          false,
		"POST /api/v1/form-templates":                         false,
		"POST /api/v1/form-templates/{id}/transition":         false,
		"POST /api/v1/form-templates/{id}/collections":        false,
		"GET /api/v1/programs/{id}/monitoring-checks":         false,
		"POST /api/v1/programs/{id}/monitoring-checks":        false,
		"POST /api/v1/monitoring-checks/{id}/transition":      false,
		"POST /api/v1/monitoring-checks/{id}/evaluate-source": false,
		"GET /api/v1/monitoring-checks/{id}/results":          false,
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
