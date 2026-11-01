package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

func TestAutomationPolicyReadReportsVerifiedTenantAndLatestBoundedPopulation(t *testing.T) {
	policies := []autonomy.AutomationPolicy{{ID: "private", TenantID: "other-bank", Code: "000", Version: 99}}
	for i := 0; i < 201; i++ {
		code := fmt.Sprintf("POLICY-%03d", i)
		policies = append(policies,
			autonomy.AutomationPolicy{ID: code + "-active", TenantID: "bank", Code: code, Status: autonomy.AutomationPolicyActive, Version: 1},
			autonomy.AutomationPolicy{ID: code + "-draft", TenantID: "bank", Code: code, Status: autonomy.AutomationPolicyDraft, Version: 2},
		)
	}
	handler := New(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:       identity.NewDevelopmentAuthenticator("bank", "admin", "entity-a", "GRC_ADMIN"),
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Autonomy:       autonomy.NewService(autonomy.NewMemoryRepository(policies...)),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/compliance/automation-policies", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var page autonomy.AutomationPolicyPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.TenantID != "bank" || page.Scope != "TENANT" || page.RevisionBasis != "LATEST_PER_CODE" || page.Limit != 200 || page.ReturnedCount != 200 || len(page.Items) != 200 || !page.Truncated {
		t.Fatalf("incorrect population metadata: %#v", page)
	}
	contextResponse := httptest.NewRecorder()
	handler.ServeHTTP(contextResponse, httptest.NewRequest(http.MethodGet, "/api/v1/context", nil))
	var runtime struct {
		Tenant struct {
			ID string `json:"id"`
		} `json:"tenant"`
	}
	if err := json.Unmarshal(contextResponse.Body.Bytes(), &runtime); err != nil || contextResponse.Code != http.StatusOK || runtime.Tenant.ID != page.TenantID {
		t.Fatalf("runtime/page scope mismatch: %d %s page tenant=%s err=%v", contextResponse.Code, contextResponse.Body.String(), page.TenantID, err)
	}
	for i, policy := range page.Items {
		if policy.TenantID != "bank" || policy.Code != fmt.Sprintf("POLICY-%03d", i) || policy.Version != 2 || policy.Status != autonomy.AutomationPolicyDraft {
			t.Fatalf("policy read exposed an older revision or another tenant: %#v", policy)
		}
	}
	for _, query := range []string{"tenant_id=other-bank", "principal_id=other-reader", "legal_entity_id=entity-b"} {
		t.Run(query, func(t *testing.T) {
			denied := httptest.NewRecorder()
			handler.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/api/v1/compliance/automation-policies?"+query, nil))
			if denied.Code != http.StatusNotFound || strings.Contains(denied.Body.String(), "POLICY-") || strings.Contains(denied.Body.String(), "private") {
				t.Fatalf("scope conflict exposed policies: %d %s", denied.Code, denied.Body.String())
			}
		})
	}
}

type unavailableAutomationPolicyRepository struct{ autonomy.Repository }

func (unavailableAutomationPolicyRepository) ListAutomationPolicies(context.Context, string) ([]autonomy.AutomationPolicy, error) {
	return nil, errors.New("database unavailable")
}

func TestAutomationPolicyReadRequiresPermissionAndDistinguishesEmptyFromFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		auth    identity.Authenticator
		service *autonomy.Service
		status  int
	}{
		{"anonymous", nil, autonomy.NewService(autonomy.NewMemoryRepository()), http.StatusUnauthorized},
		{"no permission", identity.NewDevelopmentAuthenticator("bank", "reader", "entity-a"), autonomy.NewService(autonomy.NewMemoryRepository()), http.StatusForbidden},
		{"missing service", identity.NewDevelopmentAuthenticator("bank", "admin", "entity-a", "GRC_ADMIN"), nil, http.StatusServiceUnavailable},
		{"failed read", identity.NewDevelopmentAuthenticator("bank", "admin", "entity-a", "GRC_ADMIN"), autonomy.NewService(unavailableAutomationPolicyRepository{}), http.StatusInternalServerError},
		{"empty", identity.NewDevelopmentAuthenticator("bank", "admin", "entity-a", "GRC_ADMIN"), autonomy.NewService(autonomy.NewMemoryRepository()), http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: test.auth, Autonomy: test.service})
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/compliance/automation-policies", nil))
			if response.Code != test.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.status, response.Body.String())
			}
			if test.status == http.StatusOK {
				var page autonomy.AutomationPolicyPage
				if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
					t.Fatal(err)
				}
				if page.Items == nil || len(page.Items) != 0 || page.ReturnedCount != 0 || page.Truncated || page.TenantID != "bank" {
					t.Fatalf("incorrect empty population: %#v", page)
				}
			} else if strings.Contains(response.Body.String(), `"items"`) {
				t.Fatalf("failed read claimed an empty result: %s", response.Body.String())
			}
		})
	}
}
