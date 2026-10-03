package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type groupOversightAccessStub struct {
	page access.OversightScopePage
	err  error
}

func (s groupOversightAccessStub) ResolveOIDC(context.Context, string, string, string, string) (access.Resolution, error) {
	return access.Resolution{}, access.ErrIdentityNotProvisioned
}

func (s groupOversightAccessStub) ResolvePrincipal(context.Context, string, string, string) (access.Resolution, error) {
	return access.Resolution{}, s.err
}

func (s groupOversightAccessStub) ResolveOversightLegalEntities(_ context.Context, tenantID, _ string, _ int) (access.OversightScopePage, error) {
	page := s.page
	if page.TenantID == "" {
		page.TenantID = tenantID
	}
	if page.TenantName == "" {
		page.TenantName = "Clear Bank"
	}
	return page, s.err
}

func TestGroupOversightRouteUsesChildSetAuthorization(t *testing.T) {
	for _, route := range (&API{}).routes() {
		if route.Method != http.MethodGet || route.Path != "/api/v1/oversight/group" {
			continue
		}
		if route.Class != routeAuthenticatedRead || route.Permission != "" || route.Command != nil {
			t.Fatalf("Group posture route must be an authenticated read with independent child authorization: %#v", route)
		}
		return
	}
	t.Fatal("Group posture route was not registered")
}

func TestGroupOversightReturnsAggregateWithoutCrossEntityRecordDetails(t *testing.T) {
	now := time.Date(2026, 10, 3, 20, 0, 0, 0, time.UTC)
	zero := 0
	repo := oversight.NewMemoryRepository([]oversight.Snapshot{
		{
			SnapshotID: "snap-a", TenantID: "bank", LegalEntityID: "entity-a",
			GeneratedAt: now, PeriodStart: now.Add(-90 * 24 * time.Hour), PeriodEnd: now,
			PostureAsOf: now, ProjectionVersion: oversight.ProjectionVersion,
			Coverage: oversight.Coverage{Population: 4, Excluded: &zero, Unknown: &zero},
			Counts: oversight.Counts{CriticalHigh: 2, Overdue: 1},
			Interventions: []oversight.Intervention{{TargetType: "MATTER", TargetID: "secret-matter", Title: "Restricted sibling detail"}},
		},
		{
			SnapshotID: "snap-b", TenantID: "bank", LegalEntityID: "entity-b",
			GeneratedAt: now, PeriodStart: now.Add(-90 * 24 * time.Hour), PeriodEnd: now,
			PostureAsOf: now, ProjectionVersion: oversight.ProjectionVersion,
			Coverage: oversight.Coverage{Population: 6, Excluded: &zero, Unknown: &zero},
			Counts: oversight.Counts{CriticalHigh: 3, RoutingFailures: 2},
		},
	})
	service := oversight.NewService(repo)
	service.Now = func() time.Time { return now }
	handler := New(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:       identity.NewDevelopmentAuthenticator("bank", "group-reader", "entity-a"),
		Access: groupOversightAccessStub{page: access.OversightScopePage{Items: []access.OversightLegalEntity{
			{ID: "entity-a", Code: "A", Name: "Alpha"},
			{ID: "entity-b", Code: "B", Name: "Beta"},
		}}},
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Oversight:      service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/oversight/group", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "Restricted sibling detail") || strings.Contains(response.Body.String(), "secret-matter") {
		t.Fatalf("group response leaked record details: %s", response.Body.String())
	}
	var payload struct {
		Snapshot oversight.GroupSnapshot `json:"snapshot"`
		Metrics  metricview.GroupBundle  `json:"metrics"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Snapshot.Counts.CriticalHigh != 5 || payload.Snapshot.Coverage.ContributingChildren != 2 ||
		len(payload.Snapshot.Contributors) != 2 || payload.Metrics.ScopeKind != "ORGANIZATION" ||
		payload.Metrics.Items[0].Value != 5 {
		t.Fatalf("payload=%#v", payload)
	}
}

func TestActorContextAdvertisesGroupOnlyWhenMultipleOversightEntitiesAreAuthorized(t *testing.T) {
	handler := New(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:       identity.NewDevelopmentAuthenticator("bank", "group-reader", "entity-a"),
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Access: groupOversightAccessStub{page: access.OversightScopePage{Items: []access.OversightLegalEntity{
			{ID: "entity-a", Name: "Alpha"}, {ID: "entity-b", Name: "Beta"},
		}}},
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/context", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Capabilities map[string]bool `json:"capabilities"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Capabilities["group_oversight"] || payload.Capabilities["scope_switch"] {
		t.Fatalf("capabilities=%#v", payload.Capabilities)
	}

	handler = New(Dependencies{
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:       identity.NewDevelopmentAuthenticator("bank", "group-reader", "entity-a"),
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Access: groupOversightAccessStub{page: access.OversightScopePage{Items: []access.OversightLegalEntity{
			{ID: "entity-a", Name: "Alpha"},
		}}},
	})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/context", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("single entity status=%d body=%s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Capabilities["group_oversight"] {
		t.Fatalf("single entity sign-in advertised Group posture: %#v", payload.Capabilities)
	}
}

func TestGroupOversightKeepsMissingChildIncomplete(t *testing.T) {
	now := time.Date(2026, 10, 3, 20, 0, 0, 0, time.UTC)
	zero := 0
	service := oversight.NewService(oversight.NewMemoryRepository([]oversight.Snapshot{{
		SnapshotID: "snap-a", TenantID: "bank", LegalEntityID: "entity-a",
		GeneratedAt: now, PeriodStart: now.Add(-90 * 24 * time.Hour), PeriodEnd: now,
		PostureAsOf: now, ProjectionVersion: oversight.ProjectionVersion,
		Coverage: oversight.Coverage{Population: 4, Excluded: &zero, Unknown: &zero},
	}}))
	service.Now = func() time.Time { return now }
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank", "group-reader", "entity-a"),
		Access: groupOversightAccessStub{page: access.OversightScopePage{Items: []access.OversightLegalEntity{
			{ID: "entity-a", Name: "Alpha"}, {ID: "entity-b", Name: "Beta"},
		}}},
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Oversight:      service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/oversight/group", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Snapshot oversight.GroupSnapshot `json:"snapshot"`
		Metrics  metricview.GroupBundle  `json:"metrics"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Snapshot.Coverage.MissingChildren != 1 || payload.Snapshot.Freshness != oversight.FreshnessStale ||
		payload.Metrics.Completeness != metricview.CompletenessUnknown {
		t.Fatalf("missing child quality=%#v metrics=%#v", payload.Snapshot, payload.Metrics)
	}
}

func TestGroupOversightRequiresAtLeastTwoAuthorizedLegalEntities(t *testing.T) {
	handler := New(Dependencies{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity: identity.NewDevelopmentAuthenticator("bank", "single-reader", "entity-a"),
		Access: groupOversightAccessStub{page: access.OversightScopePage{Items: []access.OversightLegalEntity{
			{ID: "entity-a", Name: "Alpha"},
		}}},
		RuntimeContext: runtimecontext.IdentifierResolver{},
		Oversight:      oversight.NewService(oversight.NewMemoryRepository(nil)),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/oversight/group", nil))
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
