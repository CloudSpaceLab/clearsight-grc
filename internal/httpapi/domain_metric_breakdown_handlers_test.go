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

type organizationMembershipReaderStub struct {
	counts metricview.OrganizationMemberCounts
	err    error
	called bool
}

func (s *organizationMembershipReaderStub) ListSnapshotMembers(
	context.Context, string, string, string, string, string, string, string, string, int,
) (metricview.MemberPage, error) {
	return metricview.MemberPage{}, nil
}

func (s *organizationMembershipReaderStub) CountSnapshotMembersByOrganization(
	_ context.Context,
	_, _, _, _, _, _ string,
) (metricview.OrganizationMemberCounts, error) {
	s.called = true
	return s.counts, s.err
}

func TestOrganizeMetricMemberCountsUsesImmediateChildrenAndExplicitResidualBuckets(t *testing.T) {
	nodes := []runtimecontext.ScopeNode{
		{ID: "technology", Name: "Technology", Kind: runtimecontext.ScopeKindBusinessUnit, ParentID: "entity", Filterable: true},
		{ID: "infrastructure", Name: "Infrastructure", Kind: runtimecontext.ScopeKindDepartment, ParentID: "technology", Filterable: true},
		{ID: "platform", Name: "Platform", Kind: runtimecontext.ScopeKindDepartment, ParentID: "technology", Filterable: true},
		{ID: "operations", Name: "Operations", Kind: runtimecontext.ScopeKindBusinessUnit, ParentID: "entity", Filterable: true},
	}
	counts := []metricview.OrganizationMemberCount{
		{OrganizationScopeID: "infrastructure", Count: 3},
		{OrganizationScopeID: "platform", Count: 2},
		{OrganizationScopeID: "technology", Count: 1},
		{OrganizationScopeID: "", Count: 2},
		{OrganizationScopeID: "retired-scope", Count: 1},
		{OrganizationScopeID: "operations", Count: 4},
	}

	root := organizeMetricMemberCounts(counts, nodes, "")
	if len(root) != 4 {
		t.Fatalf("root buckets=%#v", root)
	}
	assertOrganizationBucket(t, root, "scope:technology", "technology", "Technology", "ORGANIZATION_SCOPE", 6)
	assertOrganizationBucket(t, root, "scope:operations", "operations", "Operations", "ORGANIZATION_SCOPE", 4)
	assertOrganizationBucket(t, root, "unattributed", "", "Unattributed", "UNATTRIBUTED", 2)
	assertOrganizationBucket(t, root, "unavailable", "", "Organization unavailable", "UNAVAILABLE", 1)

	selected := organizeMetricMemberCounts(counts[:3], nodes, "technology")
	if len(selected) != 3 {
		t.Fatalf("selected buckets=%#v", selected)
	}
	assertOrganizationBucket(t, selected, "scope:infrastructure", "infrastructure", "Infrastructure", "ORGANIZATION_SCOPE", 3)
	assertOrganizationBucket(t, selected, "scope:platform", "platform", "Platform", "ORGANIZATION_SCOPE", 2)
	assertOrganizationBucket(t, selected, "direct", "", "Direct", "DIRECT", 1)
}

func TestDomainMetricOrganizationBreakdownBindsExactSourceAndRejectsTruncatedHierarchy(t *testing.T) {
	entity := runtimecontext.ScopeNode{ID: "entity-a", Name: "Entity A", Kind: runtimecontext.ScopeKindLegalEntity}
	technology := runtimecontext.ScopeNode{
		ID: "technology", Name: "Technology", Kind: runtimecontext.ScopeKindBusinessUnit,
		ParentID: entity.ID, DepartmentPath: []string{"TECH"}, Filterable: true,
	}
	infrastructure := runtimecontext.ScopeNode{
		ID: "infrastructure", Name: "Infrastructure", Kind: runtimecontext.ScopeKindDepartment,
		ParentID: technology.ID, DepartmentPath: []string{"TECH", "INFRA"}, Filterable: true,
	}
	reader := &organizationMembershipReaderStub{counts: metricview.OrganizationMemberCounts{
		SourceID: "8f730000-0000-4000-8000-000000000001",
		MetricID: "risks_outside_appetite",
		DefinitionRevision: metricview.DomainDefinitionRevision,
		Count: 3,
		Items: []metricview.OrganizationMemberCount{
			{OrganizationScopeID: technology.ID, Count: 1},
			{OrganizationScopeID: infrastructure.ID, Count: 2},
		},
	}}
	api := &API{deps: Dependencies{
		MetricMembership: reader,
		RuntimeContext: scopeContextResolverStub{hierarchy: runtimecontext.ScopeHierarchy{
			State: runtimecontext.HierarchyComplete, Current: entity, LegalEntities: []runtimecontext.ScopeNode{entity},
			OrganizationScopes: []runtimecontext.ScopeNode{technology, infrastructure},
		}},
	}}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/metrics/domain/risks_outside_appetite/organization-breakdown?source_id=8f730000-0000-4000-8000-000000000001&definition_revision=enterprise-domain-v1&organization_scope_id=technology",
		nil,
	)
	request.SetPathValue("metric_id", "risks_outside_appetite")
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: entity.ID, PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.domainMetricOrganizationBreakdown(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var breakdown organizationMetricBreakdown
	if err := json.Unmarshal(response.Body.Bytes(), &breakdown); err != nil {
		t.Fatal(err)
	}
	if breakdown.Count != 3 || breakdown.ScopeID != technology.ID || len(breakdown.Items) != 2 {
		t.Fatalf("breakdown=%#v", breakdown)
	}
	assertOrganizationBucket(t, breakdown.Items, "scope:infrastructure", "infrastructure", "Infrastructure", "ORGANIZATION_SCOPE", 2)
	assertOrganizationBucket(t, breakdown.Items, "direct", "", "Direct", "DIRECT", 1)

	reader.called = false
	api.deps.RuntimeContext = scopeContextResolverStub{hierarchy: runtimecontext.ScopeHierarchy{
		State: runtimecontext.HierarchyComplete, Current: entity, LegalEntities: []runtimecontext.ScopeNode{entity},
		OrganizationScopes: []runtimecontext.ScopeNode{technology}, OrganizationScopesTruncated: true,
	}}
	response = httptest.NewRecorder()
	api.domainMetricOrganizationBreakdown(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("truncated status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.called {
		t.Fatal("truncated hierarchy reached organization membership reader")
	}
}

func assertOrganizationBucket(
	t *testing.T,
	items []organizationMetricBucket,
	key string,
	scopeID string,
	label string,
	kind string,
	value int,
) {
	t.Helper()
	for _, item := range items {
		if item.Key != key {
			continue
		}
		if item.ScopeID != scopeID || item.Label != label || item.Kind != kind || item.Value != value {
			t.Fatalf("bucket %q=%#v", key, item)
		}
		return
	}
	t.Fatalf("bucket %q not found in %#v", key, items)
}
