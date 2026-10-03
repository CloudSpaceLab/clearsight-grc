package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type scopeContextResolverStub struct {
	display   runtimecontext.DisplayContext
	hierarchy runtimecontext.ScopeHierarchy
}

func (s scopeContextResolverStub) Resolve(context.Context, runtimecontext.Scope) (runtimecontext.DisplayContext, error) {
	return s.display, nil
}

func (s scopeContextResolverStub) ResolveHierarchy(context.Context, runtimecontext.Scope) (runtimecontext.ScopeHierarchy, error) {
	return s.hierarchy, nil
}

type scopeSearchResolverStub struct {
	scopeContextResolverStub
	page runtimecontext.OrganizationScopeSearchPage
}

func (s scopeSearchResolverStub) SearchOrganizationScopes(context.Context, runtimecontext.Scope, string, int) (runtimecontext.OrganizationScopeSearchPage, error) {
	return s.page, nil
}

func TestFederationScopeRouteRejectsInvalidBody(t *testing.T) {
	api := &API{deps: Dependencies{Federation: &federation.Service{}}}
	mux := http.NewServeMux()
	api.registerFederationRoutes(mux)
	now := time.Now().UTC()
	actor := identity.Actor{
		TenantID: "bank-demo", LegalEntityID: "BANK-NG", PrincipalID: "principal-1", Kind: "PERSON",
		AuthenticationMethod: "OIDC", AssuranceLevel: "MFA", SessionID: "session-1",
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}
	request := httptest.NewRequest(http.MethodPost, "/auth/scope", strings.NewReader(`{"legal_entity_id":""}`))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(identity.WithActor(request.Context(), actor))
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "scope_switch_invalid") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestActorContextAdvertisesScopeSwitchOnlyForFederatedMultiEntityContext(t *testing.T) {
	now := time.Now().UTC()
	actor := identity.Actor{
		TenantID: "bank-demo", LegalEntityID: "BANK-NG", PrincipalID: "principal-1", Kind: "PERSON",
		AuthenticationMethod: "OIDC", AssuranceLevel: "MFA", SessionID: "session-1",
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}
	root := runtimecontext.ScopeNode{ID: "tenant-1", Code: "bank-demo", Name: "Clear Bank", Kind: runtimecontext.ScopeKindOrganization}
	current := runtimecontext.ScopeNode{ID: "entity-ng", Code: "BANK-NG", Name: "Clear Bank Nigeria", Kind: runtimecontext.ScopeKindLegalEntity, ParentID: root.ID, Current: true}
	ghana := runtimecontext.ScopeNode{ID: "entity-gh", Code: "BANK-GH", Name: "Clear Bank Ghana", Kind: runtimecontext.ScopeKindLegalEntity, ParentID: root.ID}
	resolver := scopeContextResolverStub{
		display:   runtimecontext.DisplayContext{TenantName: "Clear Bank", LegalEntityName: "Clear Bank Nigeria", PrincipalName: "Risk Officer"},
		hierarchy: runtimecontext.ScopeHierarchy{State: runtimecontext.HierarchyComplete, Root: root, Current: current, LegalEntities: []runtimecontext.ScopeNode{current, ghana}},
	}
	for _, test := range []struct {
		name                        string
		federation                  *federation.Service
		hierarchyState              runtimecontext.HierarchyState
		organizationScopesTruncated bool
		want                        string
	}{
		{name: "federated", federation: &federation.Service{}, hierarchyState: runtimecontext.HierarchyComplete, want: `"scope_switch":true`},
		{name: "federated with truncated areas", federation: &federation.Service{}, hierarchyState: runtimecontext.HierarchyComplete, organizationScopesTruncated: true, want: `"scope_switch":true`},
		{name: "non-federated", federation: nil, hierarchyState: runtimecontext.HierarchyComplete, want: `"scope_switch":false`},
		{name: "truncated legal entities", federation: &federation.Service{}, hierarchyState: runtimecontext.HierarchyTruncated, want: `"scope_switch":false`},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver.hierarchy.State = test.hierarchyState
			resolver.hierarchy.OrganizationScopesTruncated = test.organizationScopesTruncated
			api := &API{deps: Dependencies{
				Logger: slog.Default(), RuntimeContext: resolver, Federation: test.federation,
			}}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/context", nil)
			request = request.WithContext(identity.WithActor(request.Context(), actor))
			response := httptest.NewRecorder()

			api.actorContext(response, request)

			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), test.want) {
				t.Fatalf("status=%d body=%s want=%s", response.Code, response.Body.String(), test.want)
			}
		})
	}
}


func TestActorOrganizationScopeSearchUsesVerifiedScopeAndBoundedQuery(t *testing.T) {
	now := time.Now().UTC()
	actor := identity.Actor{
		TenantID: "bank-demo", LegalEntityID: "BANK-NG", PrincipalID: "principal-1", Kind: "PERSON",
		AuthenticationMethod: "OIDC", AssuranceLevel: "MFA", SessionID: "session-1",
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour),
	}
	resolver := scopeSearchResolverStub{page: runtimecontext.OrganizationScopeSearchPage{Items: []runtimecontext.ScopeNode{{
		ID: "scope-payments", Code: "PAYMENTS", Name: "Payments", Kind: runtimecontext.ScopeKindFunction,
		DepartmentPath: []string{"BANK", "RISK", "PAYMENTS"}, Filterable: true,
	}}}}
	api := &API{deps: Dependencies{RuntimeContext: resolver}}

	valid := httptest.NewRequest(http.MethodGet, "/api/v1/context/organization-scopes?q=payments&limit=20", nil)
	valid = valid.WithContext(identity.WithActor(valid.Context(), actor))
	response := httptest.NewRecorder()
	api.actorOrganizationScopeSearch(response, valid)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "scope-payments") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	invalid := httptest.NewRequest(http.MethodGet, "/api/v1/context/organization-scopes?q=p", nil)
	invalid = invalid.WithContext(identity.WithActor(invalid.Context(), actor))
	invalidResponse := httptest.NewRecorder()
	api.actorOrganizationScopeSearch(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest || !strings.Contains(invalidResponse.Body.String(), "scope_search_invalid") {
		t.Fatalf("invalid status=%d body=%s", invalidResponse.Code, invalidResponse.Body.String())
	}
}
