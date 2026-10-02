package httpapi

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func TestControlCatalogRoutesUseProgramOwnerAuthority(t *testing.T) {
	seenRead := false
	seenCommand := false
	for _, route := range (&API{}).controlCatalogRoutes() {
		switch route.Method + " " + route.Path {
		case "GET /api/v1/control-catalog/implementation-links":
			seenRead = route.Class == routeAuthenticatedRead
		case "POST /api/v1/programs/{id}/control-implementations/{implementation_id}/catalog":
			if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != "program.control.catalog.promote" {
				t.Fatalf("catalog promotion command contract = %#v", route)
			}
			policy := route.Command.Policy
			if policy.ObjectType != "PROGRAM" || policy.ObjectIDPath != "id" ||
				policy.Responsibility != authority.ResponsibilityOwner || policy.Materiality != 3 ||
				policy.BindLegalEntity || policy.ActorField != noActorField {
				t.Fatalf("catalog promotion policy = %#v", policy)
			}
			seenCommand = true
		default:
			t.Fatalf("unexpected control catalog route: %#v", route)
		}
	}
	if !seenRead || !seenCommand {
		t.Fatalf("control catalog route coverage read=%v command=%v", seenRead, seenCommand)
	}
}
