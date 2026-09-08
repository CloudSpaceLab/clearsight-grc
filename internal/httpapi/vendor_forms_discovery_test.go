package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVendorFormReadDoesNotRequireVendorProfileAccess(t *testing.T) {
	service := evidence.NewDistributionService(evidence.NewMemoryDistributionStore(evidence.NewMemoryRepository(nil, nil), nil, nil))
	api := &API{deps: Dependencies{FormDistributions: service}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendors/relationships/service/forms", nil)
	req.SetPathValue("id", "service")
	req = req.WithContext(identity.WithActor(req.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "reviewer", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}))
	result := httptest.NewRecorder()
	api.listVendorForms(result, req)
	if result.Code != http.StatusOK {
		t.Fatalf("form-scoped read depends on profile authority: status=%d body=%s", result.Code, result.Body.String())
	}
}
