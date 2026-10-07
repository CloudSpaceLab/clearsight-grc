package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type metricTrendReaderStub struct {
	series              metricview.TrendSeries
	organizationSeries  metricview.OrganizationTrendSeries
	err                 error
	tenantID            string
	legalEntityID       string
	organizationScopeID string
	metricID            string
	start               time.Time
	end                 time.Time
}

func (s *metricTrendReaderStub) Trend(_ context.Context, tenantID, legalEntityID, metricID string, start, end time.Time) (metricview.TrendSeries, error) {
	s.tenantID, s.legalEntityID, s.metricID, s.start, s.end = tenantID, legalEntityID, metricID, start, end
	return s.series, s.err
}

func (s *metricTrendReaderStub) OrganizationTrend(
	_ context.Context,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	metricID string,
	start time.Time,
	end time.Time,
) (metricview.OrganizationTrendSeries, error) {
	s.tenantID = tenantID
	s.legalEntityID = legalEntityID
	s.organizationScopeID = organizationScopeID
	s.metricID = metricID
	s.start = start
	s.end = end
	return s.organizationSeries, s.err
}

func TestHomeMetricTrendBindsVerifiedLegalEntityAndPeriod(t *testing.T) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := today.Add(-30 * 24 * time.Hour)
	reader := &metricTrendReaderStub{series: metricview.TrendSeries{
		MetricID: "overdue_open", DefinitionRevision: metricview.HomeDefinitionRevision, Start: start, End: now,
		Resolution: metricview.TrendResolutionDay, Points: []metricview.TrendPoint{},
		Direction: metricview.TrendImproved, ComparisonQuality: metricview.ComparisonComplete,
	}}
	api := &API{deps: Dependencies{MetricTrends: reader}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/home/overdue_open/trend?start_date="+start.Format("2006-01-02")+"&end_date="+today.Format("2006-01-02"), nil)
	request.SetPathValue("metric_id", "overdue_open")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: now.Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.homeMetricTrend(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.tenantID != "bank" || reader.legalEntityID != "bank-ng" || reader.metricID != "overdue_open" || !reader.start.Equal(start) {
		t.Fatalf("bound trend tenant=%q entity=%q metric=%q start=%s end=%s", reader.tenantID, reader.legalEntityID, reader.metricID, reader.start, reader.end)
	}
	var series metricview.TrendSeries
	if err := json.Unmarshal(response.Body.Bytes(), &series); err != nil {
		t.Fatal(err)
	}
	if series.MetricID != "overdue_open" || series.Direction != metricview.TrendImproved {
		t.Fatalf("series=%#v", series)
	}
}

func TestHomeMetricTrendRejectsOrganizationScopeAndOversizedWindow(t *testing.T) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	reader := &metricTrendReaderStub{}
	api := &API{deps: Dependencies{MetricTrends: reader}}
	for _, rawURL := range []string{
		"/api/v1/metrics/home/overdue_open/trend?start_date=" + today.Add(-30*24*time.Hour).Format("2006-01-02") + "&organization_scope_id=scope-risk",
		"/api/v1/metrics/home/overdue_open/trend?start_date=" + today.Add(-366*24*time.Hour).Format("2006-01-02"),
	} {
		request := httptest.NewRequest(http.MethodGet, rawURL, nil)
		request.SetPathValue("metric_id", "overdue_open")
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: now.Add(time.Hour),
		}))
		response := httptest.NewRecorder()
		api.homeMetricTrend(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("url=%s status=%d body=%s", rawURL, response.Code, response.Body.String())
		}
	}
}

func TestDomainMetricOrganizationTrendBindsAuthorizedScopeAndPeriod(t *testing.T) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := today.Add(-30 * 24 * time.Hour)
	reader := &metricTrendReaderStub{organizationSeries: metricview.OrganizationTrendSeries{
		MetricID:            "risks_outside_appetite",
		DefinitionRevision:  metricview.DomainDefinitionRevision,
		OrganizationScopeID: "scope-risk",
		Start:               start,
		End:                 now,
		Resolution:          metricview.TrendResolutionDay,
		Points:              []metricview.OrganizationTrendPoint{{
			Date:           start.Format("2006-01-02"),
			At:             start.Add(23 * time.Hour),
			Value:          7,
			SourceRevision: metricview.DomainSourceRevision,
			SourceComplete: true,
		}},
	}}
	resolver := &exactOrganizationScopeResolverStub{selection: runtimecontext.OrganizationScopeSelection{
		Node: runtimecontext.ScopeNode{ID: "scope-risk", Filterable: true},
		IDs:  []string{"scope-risk"},
	}}
	api := &API{deps: Dependencies{MetricTrends: reader, RuntimeContext: resolver}}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/domain/risks_outside_appetite/organization-trend?organization_scope_id=scope-risk&start_date="+start.Format("2006-01-02")+"&end_date="+today.Format("2006-01-02"),
		nil,
	)
	request.SetPathValue("metric_id", "risks_outside_appetite")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: now.Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.domainMetricOrganizationTrend(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.tenantID != "bank" || reader.legalEntityID != "bank-ng" ||
		reader.organizationScopeID != "scope-risk" || reader.metricID != "risks_outside_appetite" ||
		!reader.start.Equal(start) {
		t.Fatalf(
			"bound organization trend tenant=%q entity=%q scope=%q metric=%q start=%s end=%s",
			reader.tenantID, reader.legalEntityID, reader.organizationScopeID, reader.metricID, reader.start, reader.end,
		)
	}
	if resolver.calls != 1 {
		t.Fatalf("scope resolver calls=%d", resolver.calls)
	}
	var series metricview.OrganizationTrendSeries
	if err := json.Unmarshal(response.Body.Bytes(), &series); err != nil {
		t.Fatal(err)
	}
	if series.OrganizationScopeID != "scope-risk" || len(series.Points) != 1 || series.Points[0].Value != 7 {
		t.Fatalf("series=%#v", series)
	}
}

