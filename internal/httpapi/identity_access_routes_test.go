package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type staticIdentityAuthenticator struct{ actor identity.Actor }

func (a staticIdentityAuthenticator) Authenticate(*http.Request) (identity.Actor, bool, error) {
	return a.actor, true, nil
}

type fakeAccessAdministrator struct {
	created  bool
	boundary access.ProposeLegalEntityDataBoundaryInput
}

func (a *fakeAccessAdministrator) Overview(context.Context, string, string, int) (access.AdminOverview, error) {
	return access.AdminOverview{}, nil
}
func (a *fakeAccessAdministrator) CreateSCIMSource(_ context.Context, _ access.CreateSCIMSourceInput, digest []byte) (access.SCIMSourceSummary, error) {
	a.created = len(digest) == 32
	return access.SCIMSourceSummary{ID: "source-1", Code: "ENTRA", Status: "ACTIVE", SubjectAttribute: "externalId"}, nil
}
func (*fakeAccessAdministrator) RotateSCIMSourceToken(context.Context, string, string, string, []byte) error {
	return nil
}
func (*fakeAccessAdministrator) RevokeSCIMSource(context.Context, string, string, string) error {
	return nil
}
func (*fakeAccessAdministrator) CreateGroupRoleBinding(context.Context, access.CreateGroupRoleBindingInput) (access.GroupRoleBindingSummary, error) {
	return access.GroupRoleBindingSummary{}, nil
}
func (*fakeAccessAdministrator) RetireGroupRoleBinding(context.Context, string, string, string) error {
	return nil
}
func (*fakeAccessAdministrator) ProposeOrganizationScope(context.Context, access.ProposeOrganizationScopeInput) (access.OrganizationScopeRevisionSummary, error) {
	return access.OrganizationScopeRevisionSummary{ID: "revision-1", ScopeID: "scope-1", Status: "PENDING"}, nil
}
func (*fakeAccessAdministrator) ApproveOrganizationScope(context.Context, access.DecideOrganizationScopeInput) error {
	return nil
}
func (*fakeAccessAdministrator) RejectOrganizationScope(context.Context, access.DecideOrganizationScopeInput) error {
	return nil
}
func (*fakeAccessAdministrator) ProposeOrganizationPosition(context.Context, access.ProposeOrganizationPositionInput) (access.OrganizationPositionRevisionSummary, error) {
	return access.OrganizationPositionRevisionSummary{ID: "position-revision-1", PositionID: "position-1", Status: "PENDING"}, nil
}
func (*fakeAccessAdministrator) ApproveOrganizationPosition(context.Context, access.DecideOrganizationPositionInput) error {
	return nil
}
func (*fakeAccessAdministrator) RejectOrganizationPosition(context.Context, access.DecideOrganizationPositionInput) error {
	return nil
}
func (a *fakeAccessAdministrator) ProposeLegalEntityDataBoundary(_ context.Context, input access.ProposeLegalEntityDataBoundaryInput) (access.LegalEntityDataBoundaryRevision, error) {
	a.boundary = input
	return access.LegalEntityDataBoundaryRevision{
		ID: "boundary-revision-1", LegalEntityID: input.LegalEntityID, BaseVersion: input.ExpectedVersion,
		ProposedResidencyRegion: input.ResidencyRegion, ProposedDetailTransferMode: input.DetailTransferMode,
		ProposedDestinationRegions: input.AllowedDestinationRegions, MakerID: input.ActorID, Status: "PENDING",
	}, nil
}
func (*fakeAccessAdministrator) ApproveLegalEntityDataBoundary(context.Context, access.DecideLegalEntityDataBoundaryInput) error {
	return nil
}
func (*fakeAccessAdministrator) RejectLegalEntityDataBoundary(context.Context, access.DecideLegalEntityDataBoundaryInput) error {
	return nil
}

func TestIdentityAccessRoutesSeparateReadFromConfigure(t *testing.T) {
	now := time.Now().UTC()
	base := identity.Actor{
		TenantID: "bank", PrincipalID: "principal", LegalEntityID: "bank-ng", Kind: "PERSON",
		AuthenticationMethod: "test", AssuranceLevel: "test", SessionID: "session", IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	admin := &fakeAccessAdministrator{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	readOnly := base
	readOnly.PermissionCodes = []string{identity.PermissionIdentityRead}
	handler := New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: readOnly}, AccessAdmin: admin})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/access/overview", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("identity read should load overview, got %d: %s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/api/v1/access/scim-sources", strings.NewReader(`{"code":"ENTRA","subject_attribute":"externalId"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || admin.created {
		t.Fatalf("identity read must not mutate sources, status=%d created=%v body=%s", response.Code, admin.created, response.Body.String())
	}

	configure := base
	configure.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure}
	handler = New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: configure}, AccessAdmin: admin})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/access/scim-sources", strings.NewReader(`{"code":"ENTRA","subject_attribute":"externalId"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !admin.created {
		t.Fatalf("identity configure should create source, status=%d created=%v body=%s", response.Code, admin.created, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"token":"cs_scim_`) {
		t.Fatalf("create must return a reveal-once provisioning token: %s", response.Body.String())
	}
}

func TestDataBoundaryChangeRequiresGovernedCurrentLegalEntity(t *testing.T) {
	now := time.Now().UTC()
	base := identity.Actor{
		TenantID: "bank", PrincipalID: "principal", LegalEntityID: "bank-ng", Kind: "PERSON",
		AuthenticationMethod: "test", AssuranceLevel: "test", SessionID: "session", IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	admin := &fakeAccessAdministrator{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	identityOnly := base
	identityOnly.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure}
	handler := New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: identityOnly}, AccessAdmin: admin})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/access/legal-entity-data-boundary-revisions", strings.NewReader(`{
		"tenant_id":"spoofed","legal_entity_id":"bank-gh",
		"residency_region":"NG","detail_transfer_mode":"AGGREGATE_ONLY","expected_version":0
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("identity configure alone must not mutate data boundary, got %d: %s", response.Code, response.Body.String())
	}

	governed := base
	governed.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure, identity.PermissionConfigWrite}
	handler = New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: governed}, AccessAdmin: admin})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/access/legal-entity-data-boundary-revisions", strings.NewReader(`{
		"tenant_id":"spoofed","legal_entity_id":"bank-gh",
		"residency_region":"NG","detail_transfer_mode":"AGGREGATE_ONLY","expected_version":0
	}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("governed data-boundary proposal status=%d body=%s", response.Code, response.Body.String())
	}
	if admin.boundary.TenantID != "bank" || admin.boundary.LegalEntityID != "bank-ng" || admin.boundary.ActorID != "principal" {
		t.Fatalf("data-boundary scope/actor was not server-bound: %#v", admin.boundary)
	}
}

func TestEscalationGuardMutationRequiresIdentityAndGovernanceConfigure(t *testing.T) {
	now := time.Now().UTC()
	base := identity.Actor{
		TenantID: "bank", PrincipalID: "principal", LegalEntityID: "bank-ng", Kind: "PERSON",
		AuthenticationMethod: "test", AssuranceLevel: "test", SessionID: "session", IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	governanceService := governance.NewService(governance.NewMemoryRepository())

	identityOnly := base
	identityOnly.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure}
	handler := New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: identityOnly}, Governance: governanceService})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/access/escalation-guard-revisions", strings.NewReader(`{"policy_id":"policy","sequence_id":"overdue","step_index":0,"expected_policy_version":1}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("identity configure without governance configure must not mutate escalation policy, got %d: %s", response.Code, response.Body.String())
	}

	governed := base
	governed.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure, identity.PermissionConfigWrite}
	handler = New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: governed}, Governance: governanceService})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/access/escalation-guard-revisions", strings.NewReader(`{"policy_id":"policy","sequence_id":"overdue","step_index":0,"expected_policy_version":1}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code == http.StatusForbidden {
		t.Fatalf("actor with both configuration permissions should reach governed service validation, got %d: %s", response.Code, response.Body.String())
	}
}

func TestOrganizationPositionChangesRequireIdentityAndGovernanceConfigure(t *testing.T) {
	now := time.Now().UTC()
	base := identity.Actor{
		TenantID: "bank", PrincipalID: "principal", LegalEntityID: "bank-ng", Kind: "PERSON",
		AuthenticationMethod: "test", AssuranceLevel: "test", SessionID: "session", IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	admin := &fakeAccessAdministrator{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	identityOnly := base
	identityOnly.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure}
	handler := New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: identityOnly}, AccessAdmin: admin})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/access/organization-position-revisions", strings.NewReader(`{"operation":"CREATE","code":"RISK_MANAGER","title":"Risk Manager"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("identity configure alone must not mutate organization positions, got %d: %s", response.Code, response.Body.String())
	}

	governed := base
	governed.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure, identity.PermissionConfigWrite}
	handler = New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: governed}, AccessAdmin: admin})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/access/organization-position-revisions", strings.NewReader(`{"operation":"CREATE","code":"RISK_MANAGER","title":"Risk Manager"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("governed position proposal status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestOrganizationScopeChangesRequireIdentityAndGovernanceConfigure(t *testing.T) {
	now := time.Now().UTC()
	base := identity.Actor{
		TenantID: "bank", PrincipalID: "principal", LegalEntityID: "bank-ng", Kind: "PERSON",
		AuthenticationMethod: "test", AssuranceLevel: "test", SessionID: "session", IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	}
	admin := &fakeAccessAdministrator{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	identityOnly := base
	identityOnly.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure}
	handler := New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: identityOnly}, AccessAdmin: admin})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/access/organization-scope-revisions", strings.NewReader(`{"operation":"CREATE","code":"RISK","name":"Risk","kind":"DEPARTMENT"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("identity configure alone must not mutate organization scope, got %d: %s", response.Code, response.Body.String())
	}

	governed := base
	governed.PermissionCodes = []string{identity.PermissionIdentityRead, identity.PermissionIdentityConfigure, identity.PermissionConfigWrite}
	handler = New(Dependencies{Logger: logger, Identity: staticIdentityAuthenticator{actor: governed}, AccessAdmin: admin})
	request = httptest.NewRequest(http.MethodPost, "/api/v1/access/organization-scope-revisions", strings.NewReader(`{"operation":"CREATE","code":"RISK","name":"Risk","kind":"DEPARTMENT"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("governed organization proposal status=%d body=%s", response.Code, response.Body.String())
	}
}
