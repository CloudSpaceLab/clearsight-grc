package httpapi

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func TestControlCatalogRoutesUseProgramOwnerAuthority(t *testing.T) {
	want := map[string]string{
		"POST /api/v1/programs/{id}/control-implementations/{implementation_id}/catalog":                 "program.control.catalog.promote",
		"POST /api/v1/programs/{id}/control-implementations/{implementation_id}/catalog/{definition_id}": "program.control.catalog.reuse",
	}
	for _, route := range (&API{}).controlCatalogRoutes() {
		key := route.Method + " " + route.Path
		command, ok := want[key]
		if !ok {
			t.Fatalf("unexpected control catalog route: %#v", route)
		}
		if route.Class != routeMaterialCommand || route.Command == nil || route.Command.Name != command {
			t.Fatalf("%s command contract = %#v", key, route)
		}
		policy := route.Command.Policy
		if policy.ObjectType != "PROGRAM" || policy.ObjectIDPath != "id" ||
			policy.Responsibility != authority.ResponsibilityOwner || policy.Materiality != 3 ||
			policy.BindLegalEntity || policy.ActorField != noActorField {
			t.Fatalf("%s policy = %#v", key, policy)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing control catalog routes: %#v", want)
	}
}
