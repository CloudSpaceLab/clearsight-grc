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

type scopedDomainReaderStub struct {
	latest metricview.DomainBundle
	scoped metricview.DomainBundle
	err    error
	got    struct {
		tenantID, legalEntityID, organizationScopeID string
		organizationScopeIDs                         []string
	}
}

func (s *scopedDomainReaderStub) LatestDomainMetrics(_ context.Context, tenantID, legalEntityID string) (metricview.DomainBundle, error) {
	s.got.tenantID, s.got.legalEntityID = tenantID, legalEntityID
	return s.latest, s.err
}

func (s *scopedDomainReaderStub) CurrentDomainMetrics(
	_ context.Context,
	tenantID string,
	legalEntityID string,
	organizationScopeID string,
	organizationScopeIDs []string,
	_ time.Time,
) (metricview.DomainBundle, error) {
	s.got.tenantID = tenantID
	s.got.legalEntityID = legalEntityID
	s.got.organizationScopeID = organizationScopeID
	s.got.organizationScopeIDs = append([]string(nil), organizationScopeIDs...)
	return s.scoped, s.err
}

func TestDomainMetricsBindAuthorizedOrganizationScopeAndDescendants(t *testing.T) {
	entity := runtimecontext.ScopeNode{ID: "entity-a", Name: "Entity A", Kind: runtimecontext.ScopeKindLegalEntity}
	parent := runtimecontext.ScopeNode{
		ID: "scope-risk", Name: "Risk", Kind: runtimecontext.ScopeKindDepartment,
		ParentID: entity.ID, DepartmentPath: []string{"BANK", "RISK"}, Filterable: true,
	}
	child := runtimecontext.ScopeNode{
		ID: "scope-risk-ops", Name: "Risk Operations", Kind: runtimecontext.ScopeKindDepartment,
		ParentID: parent.ID, DepartmentPath: []string{"BANK", "RISK", "OPERATIONS"}, Filterable: true,
	}
	sibling := runtimecontext.ScopeNode{
		ID: "scope-finance", Name: "Finance", Kind: runtimecontext.ScopeKindDepartment,
		ParentID: entity.ID, DepartmentPath: []string{"BANK", "FINANCE"}, Filterable: true,
	}
	reader := &scopedDomainReaderStub{scoped: metricview.DomainBundle{
		ScopeID: parent.ID, ScopeKind: "ORGANIZATION_SCOPE", SourceID: "source-1",
		DefinitionRevision: metricview.DomainDefinitionRevision,
	}}
	api := &API{deps: Dependencies{
		DomainMetrics: reader,
		RuntimeContext: scopeContextResolverStub{hierarchy: runtimecontext.ScopeHierarchy{
			State: runtimecontext.HierarchyComplete, Current: entity, LegalEntities: []runtimecontext.ScopeNode{entity},
			OrganizationScopes: []runtimecontext.ScopeNode{parent, child, sibling},
		}},
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/domain?organization_scope_id="+parent.ID, nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: entity.ID, PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.domainMetrics(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.got.organizationScopeID != parent.ID || len(reader.got.organizationScopeIDs) != 2 {
		t.Fatalf("scoped request=%#v", reader.got)
	}
	foundParent, foundChild := false, false
	for _, id := range reader.got.organizationScopeIDs {
		foundParent = foundParent || id == parent.ID
		foundChild = foundChild || id == child.ID
		if id == sibling.ID {
			t.Fatalf("sibling scope leaked into descendant selection: %#v", reader.got.organizationScopeIDs)
		}
	}
	if !foundParent || !foundChild {
		t.Fatalf("descendant selection=%#v", reader.got.organizationScopeIDs)
	}
	var bundle metricview.DomainBundle
	if err := json.Unmarshal(response.Body.Bytes(), &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.ScopeKind != "ORGANIZATION_SCOPE" || bundle.ScopeID != parent.ID {
		t.Fatalf("bundle=%#v", bundle)
	}
}

func TestDomainMetricsRejectForbiddenOrganizationScopeBeforeMetricRead(t *testing.T) {
	entity := runtimecontext.ScopeNode{ID: "entity-a", Name: "Entity A", Kind: runtimecontext.ScopeKindLegalEntity}
	reader := &scopedDomainReaderStub{}
	api := &API{deps: Dependencies{
		DomainMetrics: reader,
		RuntimeContext: scopeContextResolverStub{hierarchy: runtimecontext.ScopeHierarchy{
			State: runtimecontext.HierarchyComplete, Current: entity, LegalEntities: []runtimecontext.ScopeNode{entity},
		}},
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/domain?organization_scope_id=scope-forbidden", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: entity.ID, PrincipalID: "cro-1", ExpiresAt: time.Now().Add(time.Hour),
	}))
	response := httptest.NewRecorder()

	api.domainMetrics(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.got.organizationScopeID != "" {
		t.Fatalf("forbidden request reached metric reader: %#v", reader.got)
	}
}
