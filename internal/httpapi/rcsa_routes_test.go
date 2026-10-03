package httpapi

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func TestRCSARoutesUseGovernedAuthorityContracts(t *testing.T) {
	want := map[string]struct {
		command        string
		responsibility authority.Responsibility
	}{
		"POST /api/v1/rcsa/cycles":                              {"rcsa.cycle.create", authority.ResponsibilityOwner},
		"POST /api/v1/rcsa/cycles/{id}/first-line-distribution": {"rcsa.first-line.bind", authority.ResponsibilityOwner},
		"POST /api/v1/rcsa/cycles/{id}/first-line-complete":     {"rcsa.first-line.complete", authority.ResponsibilityOwner},
		"POST /api/v1/rcsa/cycles/{id}/challenge/start":         {"rcsa.challenge.start", authority.ResponsibilityReviewer},
		"POST /api/v1/rcsa/cycles/{id}/challenge/complete":      {"rcsa.challenge.complete", authority.ResponsibilityReviewer},
	}
	seenRead := false
	for _, route := range (&API{}).rcsaRoutes() {
		key := route.Method + " " + route.Path
		if key == "GET /api/v1/rcsa/cycles/{id}" {
			seenRead = route.Class == routeAuthenticatedRead
			continue
		}
		expected, ok := want[key]
		if !ok {
			t.Fatalf("unexpected RCSA route: %#v", route)
		}
		if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != expected.command {
			t.Fatalf("%s command = %#v", key, route)
		}
		policy := route.Command.Policy
		if policy.ObjectType != "RCSA_CYCLE" || policy.Responsibility != expected.responsibility ||
			policy.Materiality != 3 || policy.ActorField != noActorField {
			t.Fatalf("%s policy = %#v", key, policy)
		}
		if key == "POST /api/v1/rcsa/cycles" {
			if !policy.BindLegalEntity {
				t.Fatalf("create policy does not bind legal entity: %#v", policy)
			}
		} else if policy.ObjectIDPath != "id" || policy.BindLegalEntity {
			t.Fatalf("existing-cycle policy = %#v", policy)
		}
		delete(want, key)
	}
	if !seenRead || len(want) != 0 {
		t.Fatalf("route coverage read=%v missing=%#v", seenRead, want)
	}
}
