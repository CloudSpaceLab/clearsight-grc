package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestRiskIndicatorLinkOperationIsVersionedAndAuthorityBound(t *testing.T) {
	svc := risk.NewService(risk.NewMemoryRepository())
	created, err := svc.Create(t.Context(), risk.CreateInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "R-OP", Name: "Service risk", Statement: "Service may fail", Impact: "Payments delayed", Scope: json.RawMessage(`{}`), OwnerPrincipalID: "owner", ActorID: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, actor string
		service     *assignmentAuthorityStub
		want        bool
	}{
		{"allowed", "owner", &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner"}}}}, true},
		{"other actor", "reader", &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner"}}}}, false},
		{"authority unavailable", "owner", nil, false},
		{"route failure", "owner", &assignmentAuthorityStub{err: errors.New("unavailable")}, false},
		{"delegated owner", "delegate", &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner"}, CandidatePrincipals: []authority.Principal{{ID: "delegate"}, {ID: "unrelated"}}, EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "delegate", OriginPrincipalID: "owner"}, {PrincipalID: "unrelated", OriginPrincipalID: "another"}}}}}, true},
		{"unrelated route candidate", "unrelated", &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner"}, CandidatePrincipals: []authority.Principal{{ID: "delegate"}, {ID: "unrelated"}}, EffectiveOrigins: []authority.EffectiveOrigin{{PrincipalID: "delegate", OriginPrincipalID: "owner"}, {PrincipalID: "unrelated", OriginPrincipalID: "another"}}}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps := Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", tc.actor, "entity-a", "CRO"), Risk: svc}
			if tc.service != nil {
				deps.Authority = tc.service
			}
			h := New(deps)
			res := httptest.NewRecorder()
			h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/risks/"+created.ID+"/operations", nil))
			if res.Code != http.StatusOK {
				t.Fatalf("status %d: %s", res.Code, res.Body.String())
			}
			var result struct {
				RiskID      string `json:"risk_id"`
				RiskVersion int64  `json:"risk_version"`
				CanAct      bool   `json:"can_act"`
			}
			if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.RiskID != created.ID || result.RiskVersion != created.Version || result.CanAct != tc.want {
				t.Fatalf("result %#v", result)
			}
		})
	}
}

func TestRiskIndicatorOperationRejectsWrongEntity(t *testing.T) {
	svc := risk.NewService(risk.NewMemoryRepository())
	created, err := svc.Create(t.Context(), risk.CreateInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "R-SCOPE", Name: "Scoped risk", Statement: "Service may fail", Impact: "Payments delayed", Scope: json.RawMessage(`{}`), OwnerPrincipalID: "owner", ActorID: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner", "entity-b", "CRO"), Risk: svc, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner"}}}}})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/risks/"+created.ID+"/operations", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("cross-entity status %d: %s", res.Code, res.Body.String())
	}
}

type riskOperationAuthority struct {
	assignmentAuthorityStub
	input authority.ResolveInput
}

func (s *riskOperationAuthority) Resolve(ctx context.Context, input authority.ResolveInput) (authority.Resolution, error) {
	s.input = input
	return s.assignmentAuthorityStub.Resolve(ctx, input)
}
func TestRiskIndicatorOperationUsesExactMaterialRoute(t *testing.T) {
	svc := risk.NewService(risk.NewMemoryRepository())
	created, err := svc.Create(t.Context(), risk.CreateInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "R-ROUTE", Name: "Route risk", Statement: "Service may fail", Impact: "Payments delayed", Scope: json.RawMessage(`{}`), OwnerPrincipalID: "owner", ActorID: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	auth := &riskOperationAuthority{assignmentAuthorityStub: assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner"}}}}}
	h := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner", "entity-a", "CRO"), Risk: svc, Authority: auth})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/risks/"+created.ID+"/operations", nil))
	if res.Code != http.StatusOK {
		t.Fatal(res.Body.String())
	}
	in := auth.input
	if in.TenantID != "bank" || in.LegalEntityID != "entity-a" || in.ObjectType != "RISK" || in.ObjectID != created.ID || in.DecisionType != "risk.indicator.link" || in.Responsibility != authority.ResponsibilityOwner || in.Materiality != 3 || in.At.IsZero() {
		t.Fatalf("route = %#v", in)
	}
}

func TestRiskIndicatorOperationWithholdsMissingStoredOwner(t *testing.T) {
	svc := risk.NewService(risk.NewMemoryRepository())
	created, err := svc.Create(t.Context(), risk.CreateInput{TenantID: "bank", LegalEntityID: "entity-a", Code: "R-NO-OWNER", Name: "Unassigned risk", Statement: "Service may fail", Impact: "Payments delayed", Scope: json.RawMessage(`{}`), ActorID: "owner"})
	if err != nil {
		t.Fatal(err)
	}
	h := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "owner", "entity-a", "CRO"), Risk: svc, Authority: &assignmentAuthorityStub{resolutions: map[authority.Responsibility]authority.Resolution{authority.ResponsibilityOwner: {Principal: authority.Principal{ID: "owner"}}}}})
	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/risks/"+created.ID+"/operations", nil))
	var result struct {
		CanAct bool `json:"can_act"`
	}
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &result) != nil || result.CanAct {
		t.Fatalf("missing owner response: %d %s", res.Code, res.Body.String())
	}
}
