package httpapi

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5/pgconn"
	"net/http/httptest"
	"strings"
	"testing"
)

type starterHandlerFixture struct {
	testFrameworkEditionReader
	actor string
}

func (s *starterHandlerFixture) StarterRequirements(context.Context, string, string, string) ([]continuity.StarterRequirement, error) {
	return nil, nil
}
func (s *starterHandlerFixture) UpdateStarterRequirement(ctx context.Context, _ string, _ continuity.StarterRequirement, _ string) (int64, error) {
	actor, _ := identity.Require(ctx)
	s.actor = actor.PrincipalID
	return 2, nil
}
func TestStarterMaintenanceAcceptsServerBoundScopeFields(t *testing.T) {
	repo := &starterHandlerFixture{}
	a := &API{deps: Dependencies{FrameworkCatalog: repo}}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"tenant_id":"tenant","legal_entity_id":"entity","rationale":"Reviewed group link","requirement":{"code":"ACCESS","title":"Access review","statement":"Review administrator access","version":1,"form_template_id":"form","form_template_version":3,"group_id":"access"}}`))
	r.SetPathValue("code", "ISO_27001_2022")
	r.SetPathValue("requirement_code", "ACCESS")
	r = r.WithContext(identity.WithActor(r.Context(), identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "verified-owner"}))
	w := httptest.NewRecorder()
	a.updateFrameworkStarterRequirement(w, r)
	if w.Code != 200 || repo.actor != "verified-owner" {
		t.Fatalf("scope-bound request failed: %d %s", w.Code, w.Body.String())
	}
}
func TestFrameworkAdoptionDoesNotExposeDatabaseErrors(t *testing.T) {
	a := &API{}
	w := httptest.NewRecorder()
	a.writeFrameworkAdoptionResult(w, httptest.NewRequest("POST", "/", nil), continuity.ProgramAggregate{}, &pgconn.PgError{Code: "23514", Message: "private SQL detail", ConstraintName: "program_requirements_modality_check"})
	if w.Code != 500 || strings.Contains(w.Body.String(), "private SQL") || strings.Contains(w.Body.String(), "program_requirements") {
		t.Fatalf("database details leaked: %s", w.Body.String())
	}
}
