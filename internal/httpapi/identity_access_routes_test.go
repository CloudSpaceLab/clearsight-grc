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

type fakeAccessAdministrator struct{ created bool }

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
