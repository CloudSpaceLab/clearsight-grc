package httpapi

import "github.com/CloudSpaceLab/clearsight-grc/internal/authority"

func (a *API) controlCatalogRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/control-catalog/implementation-links", a.listControlCatalogImplementationLinks),
		material("/api/v1/programs/{id}/control-implementations/{implementation_id}/catalog", "program.control.catalog.promote", a.promoteProgramControlToCatalog, commandPolicy{
			ObjectType: "PROGRAM", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
	}
}
