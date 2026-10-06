package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

type indicatorPortfolioReaderStub struct {
	filter metricview.IndicatorPortfolioFilter
	page   metricview.IndicatorPortfolioPage
	err    error
}

func (s *indicatorPortfolioReaderStub) LatestDomainMetrics(context.Context, string, string) (metricview.DomainBundle, error) {
	return metricview.DomainBundle{}, metricview.ErrDomainMetricsNotFound
}

func (s *indicatorPortfolioReaderStub) ListIndicators(_ context.Context, tenantID, legalEntityID string, filter metricview.IndicatorPortfolioFilter) (metricview.IndicatorPortfolioPage, error) {
	if tenantID != "bank-a" || legalEntityID != "entity-a" {
		return metricview.IndicatorPortfolioPage{}, metricview.ErrIndicatorPortfolioInvalid
	}
	s.filter = filter
	return s.page, s.err
}

func TestIndicatorPortfolioBindsVerifiedScopeAndFilters(t *testing.T) {
	reader := &indicatorPortfolioReaderStub{page: metricview.IndicatorPortfolioPage{
		GeneratedAt: time.Date(2026, 10, 6, 7, 0, 0, 0, time.UTC),
		Items: []metricview.IndicatorPortfolioItem{{
			Kind: risk.IndicatorKRI, CheckID: "check-1", CheckVersion: 3, CheckName: "Mobile success rate",
			State: metricview.IndicatorPortfolioBreach,
		}},
	}}
	api := &API{deps: Dependencies{DomainMetrics: reader}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/indicators?kind=kri&state=breach&search=mobile&limit=25", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "viewer-a",
	}))
	response := httptest.NewRecorder()

	api.indicatorPortfolio(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("indicator portfolio returned %d: %s", response.Code, response.Body.String())
	}
	if reader.filter.Kind != risk.IndicatorKRI || reader.filter.State != metricview.IndicatorPortfolioBreach || reader.filter.Search != "mobile" || reader.filter.Limit != 25 {
		t.Fatalf("filter = %#v", reader.filter)
	}
	if !strings.Contains(response.Body.String(), "Mobile success rate") {
		t.Fatalf("response = %s", response.Body.String())
	}
}

func TestIndicatorPortfolioRejectsInvalidFilter(t *testing.T) {
	reader := &indicatorPortfolioReaderStub{}
	api := &API{deps: Dependencies{DomainMetrics: reader}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/indicators?kind=kpi", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "viewer-a",
	}))
	response := httptest.NewRecorder()

	api.indicatorPortfolio(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid filter returned %d: %s", response.Code, response.Body.String())
	}
}
