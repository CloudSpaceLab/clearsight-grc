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
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
)

func TestStoredOperationalLossOwnerOrDelegateRequiredForChange(t *testing.T) {
	service := oploss.NewService(oploss.NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 3, 19, 0, 0, 0, time.UTC) }
	created, err := service.Create(t.Context(), oploss.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-OWNER", Title: "Settlement loss",
		EventType: oploss.EventExecutionDeliveryProcess, Cause: "Duplicate settlement.",
		GrossAmountMinor: 100000, Currency: "NGN",
		OccurredAt: time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC),
		DiscoveredAt: time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolver := fixedProgramAuthority{resolution: authority.Resolution{
		Principal: authority.Principal{ID: "owner-1", DisplayName: "Loss owner"},
		CandidatePrincipals: []authority.Principal{
			{ID: "owner-1", DisplayName: "Loss owner"},
			{ID: "owner-delegate", DisplayName: "Acting loss owner"},
			{ID: "other-owner", DisplayName: "Another eligible owner"},
		},
		EffectiveOrigins: []authority.EffectiveOrigin{
			{PrincipalID: "owner-1", OriginPrincipalID: "owner-1"},
			{PrincipalID: "owner-delegate", OriginPrincipalID: "owner-1"},
			{PrincipalID: "other-owner", OriginPrincipalID: "other-owner"},
		},
	}}
	api := &API{deps: Dependencies{OperationalLoss: service, Authority: resolver}}

	check := func(actorID, command string) error {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/losses/"+created.ID, nil)
		request.SetPathValue("id", created.ID)
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: actorID,
		}))
		_, err := api.lifecycleCommandPolicy(
			request.Context(),
			request,
			"bank",
			command,
			map[string]any{},
			commandPolicy{ObjectType: "OPERATIONAL_LOSS", Responsibility: authority.ResponsibilityOwner, Materiality: 3},
		)
		return err
	}

	if err := check("owner-delegate", "loss.update"); err != nil {
		t.Fatalf("stored loss owner's delegate was rejected: %v", err)
	}
	if err := check("other-owner", "loss.recovery.record"); !errors.Is(err, commandauth.ErrNotAuthorized) {
		t.Fatalf("unassigned loss owner candidate was not rejected: %v", err)
	}
}

func TestOperationalLossCommandLifecycleDoesNotRevealCrossEntityRecord(t *testing.T) {
	service := oploss.NewService(oploss.NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 3, 19, 0, 0, 0, time.UTC) }
	created, err := service.Create(t.Context(), oploss.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-SCOPED", Title: "Scoped loss",
		EventType: oploss.EventOther, Cause: "Scoped event.", GrossAmountMinor: 10000, Currency: "NGN",
		OccurredAt: time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC),
		DiscoveredAt: time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC),
		OwnerPrincipalID: "owner-a", ActorID: "owner-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	api := &API{deps: Dependencies{OperationalLoss: service}}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/losses/"+created.ID, nil)
	request.SetPathValue("id", created.ID)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-b", PrincipalID: "owner-b",
	}))
	_, err = api.lifecycleCommandPolicy(
		request.Context(),
		request,
		"bank",
		"loss.update",
		map[string]any{},
		commandPolicy{ObjectType: "OPERATIONAL_LOSS", Responsibility: authority.ResponsibilityOwner, Materiality: 3},
	)
	if !errors.Is(err, oploss.ErrNotFound) {
		t.Fatalf("cross-entity loss command error=%v", err)
	}
}
