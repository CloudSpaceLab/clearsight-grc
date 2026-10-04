package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
)

type matrixReaderStub struct {
	matrix        metricview.Matrix
	err           error
	tenantID      string
	legalEntityID string
	kind          metricview.MatrixKind
}

func (s *matrixReaderStub) RiskAppetiteMatrix(_ context.Context, tenantID, legalEntityID string, _ time.Time) (metricview.Matrix, error) {
	s.tenantID, s.legalEntityID, s.kind = tenantID, legalEntityID, metricview.MatrixRiskAppetite
	return s.matrix, s.err
}

func (s *matrixReaderStub) AssuranceCoverageMatrix(_ context.Context, tenantID, legalEntityID string, _ time.Time) (metricview.Matrix, error) {
	s.tenantID, s.legalEntityID, s.kind = tenantID, legalEntityID, metricview.MatrixAssuranceCoverage
	return s.matrix, s.err
}

func TestMetricMatrixBindsActorLegalEntity(t *testing.T) {
	reader := &matrixReaderStub{matrix: metricview.Matrix{Kind: metricview.MatrixRiskAppetite, ScopeID: "bank-ng", ScopeKind: "LEGAL_ENTITY"}}
	api := &API{deps: Dependencies{MetricMatrices: reader}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/matrices/risk-appetite", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.riskAppetiteMatrix(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.tenantID != "bank" || reader.legalEntityID != "bank-ng" || reader.kind != metricview.MatrixRiskAppetite {
		t.Fatalf("bound reader=%#v", reader)
	}
}

func TestMetricMatrixRejectsOrganizationScope(t *testing.T) {
	reader := &matrixReaderStub{}
	api := &API{deps: Dependencies{MetricMatrices: reader}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/matrices/assurance?organization_scope_id=scope-risk", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.assuranceCoverageMatrix(response, request)

	if response.Code != http.StatusBadRequest || reader.kind != "" {
		t.Fatalf("status=%d reader=%#v body=%s", response.Code, reader, response.Body.String())
	}
}
