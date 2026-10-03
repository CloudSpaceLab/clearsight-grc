package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oploss"
)

func TestOperationalLossRoutesUseGovernedAuthorityContracts(t *testing.T) {
	want := map[string]bool{
		"POST /api/v1/losses":                   true,
		"POST /api/v1/losses/{id}":              false,
		"POST /api/v1/losses/{id}/recoveries":   false,
		"POST /api/v1/losses/{id}/intervention": false,
	}
	for _, route := range (&API{}).operationalLossRoutes() {
		key := route.Method + " " + route.Path
		bindEntity, ok := want[key]
		if !ok {
			if route.Class != routeAuthenticatedRead {
				t.Fatalf("unexpected loss route: %#v", route)
			}
			continue
		}
		if route.Class != routeMaterialCommand || route.Command == nil {
			t.Fatalf("%s is not a material command: %#v", key, route)
		}
		policy := route.Command.Policy
		if policy.ObjectType != "OPERATIONAL_LOSS" ||
			policy.Responsibility != authority.ResponsibilityOwner ||
			policy.Materiality != 3 ||
			policy.BindLegalEntity != bindEntity ||
			policy.ActorField != noActorField {
			t.Fatalf("%s policy = %#v", key, policy)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing loss routes: %#v", want)
	}
}

func TestOperationalLossHTTPBindsVerifiedScopeAndActor(t *testing.T) {
	service := oploss.NewService(oploss.NewMemoryRepository())
	now := time.Date(2026, 10, 3, 19, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	handler := New(Dependencies{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:        identity.NewDevelopmentAuthenticator("bank", "loss-owner", "entity-a"),
		OperationalLoss: service,
	})

	createBody := `{
		"code":"LOSS-001",
		"title":"Duplicate settlement",
		"event_type":"EXECUTION_DELIVERY_PROCESS_MANAGEMENT",
		"cause":"Duplicate settlement instruction.",
		"description":"Duplicate settlement completed before correction.",
		"gross_amount_minor":500000000,
		"currency":"NGN",
		"occurred_at":"2026-10-03T17:00:00Z",
		"discovered_at":"2026-10-03T18:00:00Z",
		"owner_principal_id":"spoofed-owner",
		"matter_id":"client-selected-matter",
		"actor_id":"spoofed-actor"
	}`
	create := httptest.NewRecorder()
	handler.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/v1/losses", strings.NewReader(createBody)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created oploss.Loss
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.TenantID != "bank" || created.LegalEntityID != "entity-a" || created.OwnerPrincipalID != "loss-owner" || created.MatterID != "" {
		t.Fatalf("created scope/owner/intervention = %#v", created)
	}

	now = now.Add(time.Minute)
	recovery := httptest.NewRecorder()
	recoveryBody := `{
		"expected_version":1,
		"kind":"RECOVERY",
		"amount_minor":200000000,
		"reference":"Insurer settlement",
		"recovered_at":"2026-10-03T19:01:00Z",
		"actor_id":"spoofed-actor"
	}`
	handler.ServeHTTP(recovery, httptest.NewRequest(
		http.MethodPost,
		"/api/v1/losses/"+created.ID+"/recoveries",
		strings.NewReader(recoveryBody),
	))
	if recovery.Code != http.StatusCreated {
		t.Fatalf("recovery status=%d body=%s", recovery.Code, recovery.Body.String())
	}
	var response struct {
		Loss     oploss.Loss     `json:"loss"`
		Recovery oploss.Recovery `json:"recovery"`
	}
	if err := json.Unmarshal(recovery.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Loss.Version != 2 || response.Recovery.ActorID != "loss-owner" {
		t.Fatalf("recovery = %#v", response)
	}
}

func TestOperationalLossHTTPRejectsCrossEntityRead(t *testing.T) {
	service := oploss.NewService(oploss.NewMemoryRepository())
	service.Now = func() time.Time { return time.Date(2026, 10, 3, 19, 0, 0, 0, time.UTC) }
	value, err := service.Create(t.Context(), oploss.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-SCOPED", Title: "Scoped loss",
		EventType: oploss.EventOther, Cause: "Scoped event.", GrossAmountMinor: 10000, Currency: "NGN",
		OccurredAt: time.Date(2026, 10, 3, 17, 0, 0, 0, time.UTC),
		DiscoveredAt: time.Date(2026, 10, 3, 18, 0, 0, 0, time.UTC),
		OwnerPrincipalID: "owner-a", ActorID: "owner-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := New(Dependencies{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		Identity:        identity.NewDevelopmentAuthenticator("bank", "owner-b", "entity-b"),
		OperationalLoss: service,
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/losses/"+value.ID, nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestOperationalLossInterventionCreatesAndReusesCanonicalMatter(t *testing.T) {
	losses := oploss.NewService(oploss.NewMemoryRepository())
	now := time.Date(2026, 10, 3, 19, 0, 0, 0, time.UTC)
	losses.Now = func() time.Time { return now }
	created, err := losses.Create(t.Context(), oploss.CreateInput{
		TenantID: "bank", LegalEntityID: "entity-a", Code: "LOSS-MATERIAL", Title: "Material settlement loss",
		EventType: oploss.EventExecutionDeliveryProcess, Cause: "Settlement control failed.",
		Description: "A duplicate settlement caused a material operational loss.",
		GrossAmountMinor: 500000000, Currency: "NGN",
		OccurredAt: now.Add(-2 * time.Hour), DiscoveredAt: now.Add(-time.Hour),
		OwnerPrincipalID: "owner-1", ActorID: "owner-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	matters := continuity.NewService(continuity.NewMemoryRepository())
	api := &API{deps: Dependencies{OperationalLoss: losses, Continuity: matters}}

	open := func(expected int64) struct {
		Loss   oploss.Loss      `json:"loss"`
		Matter continuity.Matter `json:"matter"`
	} {
		t.Helper()
		response := httptest.NewRecorder()
		request := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/losses/"+created.ID+"/intervention",
			strings.NewReader(fmt.Sprintf(`{"expected_version":%d}`, expected)),
		)
		request.SetPathValue("id", created.ID)
		request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
			TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "owner-1",
		}))
		api.openOperationalLossIntervention(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("intervention status=%d body=%s", response.Code, response.Body.String())
		}
		var payload struct {
			Loss   oploss.Loss      `json:"loss"`
			Matter continuity.Matter `json:"matter"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}

	first := open(1)
	if first.Loss.Version != 2 || first.Loss.MatterID == "" || first.Loss.MatterID != first.Matter.ID ||
		first.Matter.Type != continuity.MatterOperationalLoss ||
		first.Matter.TriggerKey != "operational-loss:"+created.ID {
		t.Fatalf("first intervention=%#v", first)
	}
	second := open(2)
	if second.Loss.Version != 2 || second.Matter.ID != first.Matter.ID {
		t.Fatalf("intervention was not idempotent: first=%#v second=%#v", first, second)
	}
}
