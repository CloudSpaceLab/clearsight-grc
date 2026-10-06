package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

func TestProgramSummaryEndpointIsBoundedAndCursorBased(t *testing.T) {
	handler := continuityTestHandler()
	for index, code := range []string{"ALPHA", "BETA"} {
		body := []byte(`{"tenant_id":"bank","code":"` + code + `","name":"Program ` + code + `","type":"ASSURANCE","owning_function":"Control Assurance","owner_candidate_id":"owner","approval_authority_candidate_id":"approver","scope":{},"effective_from":"2026-08-05T10:00:00Z"}`)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(body)))
		if response.Code != http.StatusCreated {
			t.Fatalf("create program %d: %d %s", index, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/program-summaries?tenant_id=bank&limit=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", response.Code, response.Body.String())
	}
	var page continuity.ProgramSummaryPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.NextCursor == "" {
		t.Fatalf("expected bounded page and cursor: %#v", page)
	}
	if page.Items[0].Program.Name == "" || page.Items[0].StateLabel == "" {
		t.Fatalf("summary lacks operating labels: %#v", page.Items[0])
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/api/v1/program-summaries?tenant_id=bank&cursor=invalid", nil))
	if invalid.Code != http.StatusBadRequest || !bytes.Contains(invalid.Body.Bytes(), []byte("page cursor")) {
		t.Fatalf("expected plain invalid cursor response, got %d %s", invalid.Code, invalid.Body.String())
	}
}

func TestMatterSummaryEndpointUsesOperationalLabels(t *testing.T) {
	handler := continuityTestHandler()
	body := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Confirm access review owners","summary":"Four accounts need a current owner.","scope":{},"known_facts":{"accounts":4},"missing_facts":[],"contradictions":[]}`)
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(body)))
	if created.Code != http.StatusCreated {
		t.Fatalf("create matter: %d %s", created.Code, created.Body.String())
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/matter-summaries?tenant_id=bank&status=OPEN", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("summary status=%d body=%s", response.Code, response.Body.String())
	}
	var page continuity.MatterSummaryPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].TypeLabel != "Control gap" || page.Items[0].NextAction != "Start initial review" {
		t.Fatalf("unexpected matter summary: %#v", page.Items)
	}
}

func TestMatterSummaryReadAddsAuthorizedOwnerAndAreaLabels(t *testing.T) {
	resolver := &assessmentLabelResolver{resolution: access.Resolution{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "owner-1", DisplayName: "Hakeem Adeyemi", Kind: "PERSON",
	}}
	api := &API{deps: Dependencies{
		Access: resolver,
		RuntimeContext: scopeContextResolverStub{hierarchy: runtimecontext.ScopeHierarchy{
			OrganizationScopes: []runtimecontext.ScopeNode{{
				ID: "scope-payments", Name: "Payments Operations", Kind: runtimecontext.ScopeKindDepartment,
				DepartmentPath: []string{"Operations", "Payments"}, Filterable: true,
			}},
		}},
	}}
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "role-cro"}
	page := continuity.MatterSummaryPage{Items: []continuity.MatterSummary{{Matter: continuity.Matter{
		ID: "matter-1", TenantID: "bank", LegalEntityID: "bank-ng", OrganizationScopeID: "scope-payments", OwnerPrincipalID: "owner-1",
	}}}}

	result := api.matterSummaryPageRead(t.Context(), actor, page)
	if len(result.Items) != 1 {
		t.Fatalf("items=%#v", result.Items)
	}
	if result.Items[0].OwnerDisplayName != "Hakeem Adeyemi" {
		t.Fatalf("owner label=%q", result.Items[0].OwnerDisplayName)
	}
	if result.Items[0].OrganizationScopeLabel != "Operations / Payments" {
		t.Fatalf("organization scope label=%q", result.Items[0].OrganizationScopeLabel)
	}
	if len(resolver.calls) != 1 || resolver.calls[0] != "bank:bank-ng:owner-1" {
		t.Fatalf("owner lookups=%v", resolver.calls)
	}
}

func TestMatterAggregateReadAddsAuthorizedOwnerAndAreaLabels(t *testing.T) {
	resolver := &assessmentLabelResolver{resolution: access.Resolution{
		TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "owner-1", DisplayName: "Hakeem Adeyemi", Kind: "PERSON",
	}}
	api := &API{deps: Dependencies{
		Access: resolver,
		RuntimeContext: scopeContextResolverStub{hierarchy: runtimecontext.ScopeHierarchy{
			OrganizationScopes: []runtimecontext.ScopeNode{{
				ID: "scope-payments", Name: "Payments Operations", Kind: runtimecontext.ScopeKindDepartment,
				DepartmentPath: []string{"Operations", "Payments"}, Filterable: true,
			}},
		}},
	}}
	aggregate := continuity.MatterAggregate{Matter: continuity.Matter{
		ID: "matter-1", TenantID: "bank", LegalEntityID: "bank-ng", OrganizationScopeID: "scope-payments", OwnerPrincipalID: "owner-1",
	}}

	result := api.matterAggregateRead(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "bank-ng", PrincipalID: "role-cro"}, aggregate)
	if result.OwnerDisplayName != "Hakeem Adeyemi" || result.OrganizationScopeLabel != "Operations / Payments" {
		t.Fatalf("presentation=%+v", result)
	}
}

func TestMatterSummaryEndpointAcceptsExactProgramFilter(t *testing.T) {
	handler := continuityTestHandler()
	programBody := []byte(`{"tenant_id":"bank","code":"FILTER","name":"Filtered Program","type":"ASSURANCE","owning_function":"Control Assurance","owner_candidate_id":"owner","approval_authority_candidate_id":"approver","scope":{},"effective_from":"2026-08-05T10:00:00Z"}`)
	programResponse := httptest.NewRecorder()
	handler.ServeHTTP(programResponse, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(programBody)))
	var program continuity.ProgramAggregate
	if err := json.NewDecoder(programResponse.Body).Decode(&program); err != nil {
		t.Fatal(err)
	}
	linkedBody := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Linked issue","summary":"This issue belongs to the Program.","scope":{},"known_facts":{},"missing_facts":[],"contradictions":[],"program_id":"` + program.Program.ID + `"}`)
	linkedResponse := httptest.NewRecorder()
	handler.ServeHTTP(linkedResponse, httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(linkedBody)))
	unlinkedBody := []byte(`{"tenant_id":"bank","type":"CONTROL_GAP","priority":4,"title":"Other issue","summary":"This issue is unrelated.","scope":{},"known_facts":{},"missing_facts":[],"contradictions":[]}`)
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/matters", bytes.NewReader(unlinkedBody)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/matter-summaries?tenant_id=bank&status=OPEN&program_id="+program.Program.ID, nil))
	var page continuity.MatterSummaryPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Matter.Title != "Linked issue" {
		t.Fatalf("unexpected filtered matters: %#v", page.Items)
	}
}

func TestSummaryEndpointsValidateAndAcceptStructuredFilters(t *testing.T) {
	handler := continuityTestHandler()
	valid := httptest.NewRecorder()
	handler.ServeHTTP(valid, httptest.NewRequest(http.MethodGet, "/api/v1/matter-summaries?tenant_id=bank&matter_type=CONTROL_GAP&priority=4&due=NO_DUE_DATE&assigned_to_me=true", nil))
	if valid.Code != http.StatusOK {
		t.Fatalf("valid filters returned %d: %s", valid.Code, valid.Body.String())
	}
	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/api/v1/program-summaries?tenant_id=bank&overall_state=COMPLIANT", nil))
	if invalid.Code != http.StatusBadRequest || !bytes.Contains(invalid.Body.Bytes(), []byte("overall state")) {
		t.Fatalf("invalid state returned %d: %s", invalid.Code, invalid.Body.String())
	}
}

func TestProgramSummaryEndpointFiltersAuthorizedOrganizationDescendants(t *testing.T) {
	entity := runtimecontext.ScopeNode{ID: "bank-ng", Name: "Bank Nigeria", Kind: runtimecontext.ScopeKindLegalEntity}
	parent := runtimecontext.ScopeNode{ID: "scope-risk", Name: "Risk", Kind: runtimecontext.ScopeKindDepartment, ParentID: entity.ID, DepartmentPath: []string{"BANK", "RISK"}, Filterable: true}
	child := runtimecontext.ScopeNode{ID: "scope-risk-ops", Name: "Risk Operations", Kind: runtimecontext.ScopeKindDepartment, ParentID: parent.ID, DepartmentPath: []string{"BANK", "RISK", "OPERATIONS"}, Filterable: true}
	sibling := runtimecontext.ScopeNode{ID: "scope-finance", Name: "Finance", Kind: runtimecontext.ScopeKindDepartment, ParentID: entity.ID, DepartmentPath: []string{"BANK", "FINANCE"}, Filterable: true}
	service := continuity.NewService(continuity.NewMemoryRepository())
	handler := New(Dependencies{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AllowedOrigin: "http://localhost:5173",
		Mode:          "test-memory",
		Identity:      identity.NewDevelopmentAuthenticator("bank", "role-cro", "bank-ng"),
		RuntimeContext: scopeContextResolverStub{
			hierarchy: runtimecontext.ScopeHierarchy{
				State: runtimecontext.HierarchyComplete, Current: entity, LegalEntities: []runtimecontext.ScopeNode{entity},
				OrganizationScopes: []runtimecontext.ScopeNode{parent, child, sibling},
			},
		},
		Continuity: service,
		Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{
			authority.ResponsibilityOwner:      {Principal: authority.Principal{ID: "role-cro", DisplayName: "Program creator"}, CandidatePrincipals: []authority.Principal{{ID: "owner", DisplayName: "Program owner"}}},
			authority.ResponsibilityAuthorizer: {Principal: authority.Principal{ID: "approver", DisplayName: "Approval authority"}},
		}},
	})

	for _, program := range []struct {
		code  string
		scope string
	}{
		{code: "PARENT", scope: parent.ID},
		{code: "CHILD", scope: child.ID},
		{code: "SIBLING", scope: sibling.ID},
		{code: "UNATTRIBUTED"},
	} {
		body := []byte(`{"tenant_id":"bank","code":"` + program.code + `","name":"` + program.code + `","type":"ASSURANCE","owning_function":"Risk","owner_candidate_id":"owner","approval_authority_candidate_id":"approver","organization_scope_id":"` + program.scope + `","scope":{},"effective_from":"2026-10-03T10:00:00Z"}`)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(body)))
		if response.Code != http.StatusCreated {
			t.Fatalf("create %s status=%d body=%s", program.code, response.Code, response.Body.String())
		}
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/program-summaries?tenant_id=bank&organization_scope_id="+parent.ID+"&limit=20", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("scoped summary status=%d body=%s", response.Code, response.Body.String())
	}
	var page continuity.ProgramSummaryPage
	if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	if page.OrganizationScopeID != parent.ID || len(page.Items) != 2 {
		t.Fatalf("scoped programs=%#v", page)
	}
	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.Program.Code] = true
	}
	if !seen["PARENT"] || !seen["CHILD"] || seen["SIBLING"] || seen["UNATTRIBUTED"] {
		t.Fatalf("scope membership=%#v", seen)
	}

	forbidden := httptest.NewRecorder()
	handler.ServeHTTP(forbidden, httptest.NewRequest(http.MethodGet, "/api/v1/program-summaries?tenant_id=bank&organization_scope_id=scope-unauthorized", nil))
	if forbidden.Code != http.StatusForbidden || !bytes.Contains(forbidden.Body.Bytes(), []byte("organization_scope_forbidden")) {
		t.Fatalf("forbidden scope status=%d body=%s", forbidden.Code, forbidden.Body.String())
	}

	forbiddenCreate := httptest.NewRecorder()
	body := []byte(`{"tenant_id":"bank","code":"FORBIDDEN","name":"Forbidden","type":"ASSURANCE","owning_function":"Risk","owner_candidate_id":"owner","approval_authority_candidate_id":"approver","organization_scope_id":"scope-unauthorized","scope":{},"effective_from":"2026-10-03T10:00:00Z"}`)
	handler.ServeHTTP(forbiddenCreate, httptest.NewRequest(http.MethodPost, "/api/v1/programs", bytes.NewReader(body)))
	if forbiddenCreate.Code != http.StatusForbidden || !bytes.Contains(forbiddenCreate.Body.Bytes(), []byte("organization_scope_forbidden")) {
		t.Fatalf("forbidden create status=%d body=%s", forbiddenCreate.Code, forbiddenCreate.Body.String())
	}
}
