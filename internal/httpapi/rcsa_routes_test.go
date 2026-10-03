package httpapi

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func TestRCSARoutesUseGovernedCreateAuthority(t *testing.T) {
	seenRead := false
	seenCreate := false
	for _, route := range (&API{}).rcsaRoutes() {
		switch route.Method + " " + route.Path {
		case "GET /api/v1/rcsa/cycles/{id}":
			seenRead = route.Class == routeAuthenticatedRead
		case "POST /api/v1/rcsa/cycles":
			if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != "rcsa.cycle.create" {
				t.Fatalf("create route = %#v", route)
			}
			policy := route.Command.Policy
			if policy.ObjectType != "RCSA_CYCLE" || policy.Responsibility != authority.ResponsibilityOwner ||
				policy.Materiality != 3 || !policy.BindLegalEntity || policy.ActorField != noActorField {
				t.Fatalf("create policy = %#v", policy)
			}
			seenCreate = true
		default:
			t.Fatalf("unexpected RCSA route: %#v", route)
		}
	}
	if !seenRead || !seenCreate {
		t.Fatalf("route coverage read=%v create=%v", seenRead, seenCreate)
	}
}
