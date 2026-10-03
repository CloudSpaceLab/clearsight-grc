package httpapi

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func TestRCSARoutesUseGovernedAuthorityContracts(t *testing.T) {
	want := map[string]string{
		"POST /api/v1/rcsa/cycles":                              "rcsa.cycle.create",
		"POST /api/v1/rcsa/cycles/{id}/first-line-distribution": "rcsa.first-line.bind",
		"POST /api/v1/rcsa/cycles/{id}/first-line-complete":     "rcsa.first-line.complete",
	}
	seenRead := false
	for _, route := range (&API{}).rcsaRoutes() {
		key := route.Method + " " + route.Path
		if key == "GET /api/v1/rcsa/cycles/{id}" {
			seenRead = route.Class == routeAuthenticatedRead
			continue
		}
		command, ok := want[key]
		if !ok {
			t.Fatalf("unexpected RCSA route: %#v", route)
		}
		if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != command {
			t.Fatalf("%s command = %#v", key, route)
		}
		policy := route.Command.Policy
		if policy.ObjectType != "RCSA_CYCLE" || policy.Responsibility != authority.ResponsibilityOwner ||
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
