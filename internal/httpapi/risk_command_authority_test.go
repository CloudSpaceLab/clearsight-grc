package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

func TestStoredRiskOwnerOrDelegateRequiredForUpdate(t *testing.T) {
	service := risk.NewService(risk.NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) }
	created, err := service.Create(t.Context(), risk.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RISK-OWNER", Name: "Network resilience",
		Statement: "Critical service may exceed its recovery tolerance.", Impact: "Customer service disruption.",
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "owner-1", DisplayName: "Risk owner"},
		CandidatePrincipals: []authority.Principal{
			{ID: "owner-1", DisplayName: "Risk owner"},
			{ID: "owner-delegate", DisplayName: "Acting risk owner"},
			{ID: "other-owner", DisplayName: "Another eligible owner"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "owner-1", OriginPrincipalID: "owner-1"},
			{PrincipalID: "owner-delegate", OriginPrincipalID: "owner-1"},
			{PrincipalID: "other-owner", OriginPrincipalID: "other-owner"},
		},
	}}
	api := &API{deps: Dependencies{Risk: service, Authority: resolver}}

	check := func(actorID string) error {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/risks/"+created.ID, nil)
		request.SetPathValue("id", created.ID)
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID,
		}))
		_, err := api.lifecycleCommandPolicy(
			request.Context(),
			request,
			"bank",
			"risk.update",
			map[string]any{},
			commandPolicy{ObjectType: "RISK", Responsibility: authority.ResponsibilityOwner, Materiality: 3},
		)
		return err
	}

	if err := check("owner-delegate"); err != nil {
		t.Fatalf("stored Risk owner's delegate was rejected: %v", err)
	}
	if err := check("other-owner"); !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("unassigned Risk owner candidate was not rejected: %v", err)
	}
}

func TestRiskCommandLifecycleDoesNotRevealCrossEntityRecord(t *testing.T) {
	service := risk.NewService(risk.NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC) }
	created, err := service.Create(t.Context(), risk.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "RISK-SCOPED", Name: "Scoped risk",
		Statement: "A scoped risk exists.", Impact: "Material impact.", OwnerPrincipalID: "owner-a", ActorID: "owner-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{Risk: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/risks/"+created.ID+"/assessments", nil)
	request.SetPathValue("id", created.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-b", PrincipalID: "reviewer-b",
	}))
	_, err = api.lifecycleCommandPolicy(
		request.Context(),
		request,
		"bank",
		"risk.assess",
		map[string]any{},
		commandPolicy{ObjectType: "RISK", Responsibility: authority.ResponsibilityReviewer, Materiality: 3},
	)
	if !errors.Is(err, risk.ErrNotFound) {
		t.Fatalf("cross-entity Risk command error = %v", err)
	}
}
