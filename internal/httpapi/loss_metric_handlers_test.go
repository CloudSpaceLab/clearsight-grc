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

type lossPeriodReaderStub struct {
	bundle metricview.LossPeriodBundle
	err    error
	got    struct {
		tenantID, legalEntityID, organizationScopeID string
		organizationScopeIDs                         []string
		start, end, generatedAt                     time.Time
	}
}

func (s *lossPeriodReaderStub) CurrentLossPeriod(
	_ context.Context,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	organizationScopeIDs []string,
	start time.Time,
	end time.Time,
	generatedAt time.Time,
) (metricview.LossPeriodBundle, error) {
	s.got.tenantID = tenantID
	s.got.legalEntityID = legalEntityID
	s.got.organizationScopeID = organizationScopeID
	s.got.organizationScopeIDs = append([]string(nil), organizationScopeIDs...)
	s.got.start = start
	s.got.end = end
	s.got.generatedAt = generatedAt
	return s.bundle, s.err
}

func TestLossPeriodMetricsBindsAuthorizedScopeAndPeriod(t *testing.T) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := today.Add(-30 * 24 * time.Hour)
	reader := &lossPeriodReaderStub{bundle: metricview.LossPeriodBundle{
		GeneratedAt:           now,
		PeriodStart:           start,
		PeriodEnd:             now,
		ScopeID:               "scope-ops",
		ScopeKind:             "ORGANIZATION_SCOPE",
		SourceID:              "8f780000-0000-4000-8000-000000000001",
		SourceRevision:        metricview.LossPeriodSourceRevision,
		DefinitionRevision:    metricview.LossPeriodDefinitionRevision,
		EventCount:            3,
		ContributingLossCount: 4,
		Currencies: []metricview.LossCurrencyFlow{{
			Currency: "NGN",
			Gross:    metricview.MoneyValue{MinorUnits: "1500", Currency: "NGN"},
			Recovery: metricview.MoneyValue{MinorUnits: "500", Currency: "NGN"},
			Reversal: metricview.MoneyValue{MinorUnits: "50", Currency: "NGN"},
			Net:      metricview.MoneyValue{MinorUnits: "1050", Currency: "NGN"},
		}},
	}}
	resolver := &exactOrganizationScopeResolverStub{selection: runtimecontext.OrganizationScopeSelection{
		Node: runtimecontext.ScopeNode{ID: "scope-ops", Filterable: true},
		IDs:  []string{"scope-ops", "scope-branch"},
	}}
	api := &API{deps: Dependencies{LossPeriodMetrics: reader, RuntimeContext: resolver}}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/losses/period?organization_scope_id=scope-ops&start_date="+start.Format("2006-01-02")+"&end_date="+today.Format("2006-01-02"),
		nil,
	)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: now.Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.lossPeriodMetrics(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.got.tenantID != "bank" || reader.got.legalEntityID != "bank-ng" ||
		reader.got.organizationScopeID != "scope-ops" || len(reader.got.organizationScopeIDs) != 2 ||
		!reader.got.start.Equal(start) {
		t.Fatalf("bound request=%#v", reader.got)
	}
	var bundle metricview.LossPeriodBundle
	if err := json.Unmarshal(response.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.EventCount != 3 || bundle.ScopeID != "scope-ops" || len(bundle.Currencies) != 1 {
		t.Fatalf("bundle=%#v", bundle)
	}
}

func TestLossMetricPeriodRejectsFutureAndOversizedPeriods(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	for _, rawURL := range []string{
		"/api/v1/metrics/losses/period?start_date=2025-10-06",
		"/api/v1/metrics/losses/period?start_date=2026-10-01&end_date=2026-10-08",
	} {
		request := httptest.NewRequest(http.MethodGet, rawURL, nil)
		response := httptest.NewRecorder()
		if _, _, ok := lossMetricPeriod(response, request, now); ok {
			t.Fatalf("url %q unexpectedly accepted", rawURL)
		}
		if response.Code != http.StatusBadRequest {
			t.Fatalf("url=%s status=%d body=%s", rawURL, response.Code, response.Body.String())
		}
	}
}
