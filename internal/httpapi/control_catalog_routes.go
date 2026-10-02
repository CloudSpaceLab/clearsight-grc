package httpapi

import "github.com/CloudSpaceLab/clearsight-grc/internal/authority"

func (a *API) controlCatalogRoutes() []routeSpec {
	return []routeSpec{
		material("/api/v1/programs/{id}/control-implementations/{implementation_id}/catalog", "program.control.catalog.promote", a.promoteProgramControlToCatalog, commandPolicy{
			ObjectType: "PROGRAM", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/programs/{id}/control-implementations/{implementation_id}/catalog/{definition_id}", "program.control.catalog.reuse", a.reuseProgramControlDefinition, commandPolicy{
			ObjectType: "PROGRAM", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
	}
}
