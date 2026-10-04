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
)

type metricMembershipReaderStub struct {
	page metricview.MemberPage
	err  error
	got  struct {
		tenantID, legalEntityID, sourceID, metricID, revision, cursor string
		limit                                                         int
	}
}

func (s *metricMembershipReaderStub) ListSnapshotMembers(
	_ context.Context,
	tenantID string,
	legalEntityID string,
	sourceID string,
	metricID string,
	revision string,
	cursor string,
	limit int,
) (metricview.MemberPage, error) {
	s.got.tenantID = tenantID
	s.got.legalEntityID = legalEntityID
	s.got.sourceID = sourceID
	s.got.metricID = metricID
	s.got.revision = revision
	s.got.cursor = cursor
	s.got.limit = limit
	return s.page, s.err
}

func TestHomeMetricMembersBindVerifiedLegalEntityAndExactSource(t *testing.T) {
	reader := &metricMembershipReaderStub{page: metricview.MemberPage{
		SourceID:           "8f600000-0000-4000-8000-000000000001",
		MetricID:           "routing_gaps",
		DefinitionRevision: "home-oversight-v3",
		Count:              2,
		Items: []metricview.Member{
			{MemberID: "8f600000-0000-4000-8000-000000000010", TargetType: "MATTER", TargetID: "8f600000-0000-4000-8000-000000000020", TargetTitle: "Assign issue", State: "READY"},
			{MemberID: "8f600000-0000-4000-8000-000000000011", TargetType: "PROGRAM", TargetID: "8f600000-0000-4000-8000-000000000021", TargetTitle: "Assign Program review", State: "BLOCKED"},
		},
	}}
	api := &API{deps: Dependencies{MetricMembership: reader}}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/home/routing_gaps/members?source_id=8f600000-0000-4000-8000-000000000001&definition_revision=home-oversight-v3&cursor=8f600000-0000-4000-8000-000000000009&limit=25",
		nil,
	)
	request.SetPathValue("metric_id", "routing_gaps")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.homeMetricMembers(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.got.tenantID != "bank" || reader.got.legalEntityID != "bank-ng" ||
		reader.got.metricID != "routing_gaps" || reader.got.revision != "home-oversight-v3" ||
		reader.got.limit != 25 {
		t.Fatalf("bound request=%#v", reader.got)
	}
	var page metricview.MemberPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Count != 2 || len(page.Items) != 2 || page.Items[1].TargetType != "PROGRAM" {
		t.Fatalf("page=%#v", page)
	}
}

func TestHomeMetricMembersMapMissingSnapshotToNotFound(t *testing.T) {
	reader := &metricMembershipReaderStub{err: metricview.ErrMetricMembershipNotFound}
	api := &API{deps: Dependencies{MetricMembership: reader}}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/home/critical_high_open/members?source_id=8f600000-0000-4000-8000-000000000001&definition_revision=home-oversight-v3",
		nil,
	)
	request.SetPathValue("metric_id", "critical_high_open")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.homeMetricMembers(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestHomeMetricMembersRejectInvalidPageSizeBeforeRepositoryRead(t *testing.T) {
	reader := &metricMembershipReaderStub{}
	api := &API{deps: Dependencies{MetricMembership: reader}}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/home/critical_high_open/members?source_id=8f600000-0000-4000-8000-000000000001&definition_revision=home-oversight-v3&limit=101",
		nil,
	)
	request.SetPathValue("metric_id", "critical_high_open")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.homeMetricMembers(response, request)

	if response.Code != http.StatusBadRequest || reader.got.metricID != "" {
		t.Fatalf("status=%d repository=%#v body=%s", response.Code, reader.got, response.Body.String())
	}
}
