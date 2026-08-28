package httpapi

import (
	"net/http"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestFormCommunicationRoutesAreGovernedAndComplete(t *testing.T) {
	t.Parallel()

	routes := (&API{}).formCommunicationRoutes()
	if len(routes) != 15 {
		t.Fatalf("communication route count = %d, want 15", len(routes))
	}
	seen := map[string]routeSpec{}
	for _, route := range routes {
		seen[route.Method+" "+route.Path] = route
	}
	for _, key := range []string{
		"GET /api/v1/forms/communications/profiles",
		"GET /api/v1/forms/communications/profiles/{version}",
		"GET /api/v1/forms/communications/templates/{action}/{locale}/revisions/{version}",
		"POST /api/v1/forms/communications/templates/{action}/{locale}/revisions/{version}/preview",
		"POST /api/v1/forms/communications/templates/{action}/{locale}/revisions/{version}/test-send",
		"GET /api/v1/forms/communications/brand-assets",
	} {
		if _, ok := seen[key]; !ok {
			t.Fatalf("missing communication route %s", key)
		}
	}

	for _, route := range routes {
		if route.Method == http.MethodGet {
			if route.Class != routeAuthenticatedRead || route.Permission != identity.PermissionConfigRead {
				t.Fatalf("read route %s %s is not config-read guarded: %#v", route.Method, route.Path, route)
			}
			continue
		}
		if route.Path[len(route.Path)-8:] == "/preview" || route.Path[len(route.Path)-7:] == "/impact" {
			if route.Class != routeAuthenticatedWrite || route.Permission != identity.PermissionConfigRead {
				t.Fatalf("preview/impact route is not bounded config read: %#v", route)
			}
			continue
		}
		if route.Class != routeMaterialCommand || route.Permission != identity.PermissionConfigWrite || route.Command == nil || route.Command.Policy.ActorField != noActorField || !route.Command.Policy.BindLegalEntity {
			t.Fatalf("mutation route is not a verified material config command: %#v", route)
		}
	}
}
