package httpapi

import "github.com/CloudSpaceLab/clearsight-grc/internal/authority"

func (a *API) rcsaRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/rcsa/cycles/{id}", a.getRCSACycle),
		material("/api/v1/rcsa/cycles", "rcsa.cycle.create", a.createRCSACycle, commandPolicy{
			ObjectType: "RCSA_CYCLE", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, BindLegalEntity: true, ActorField: noActorField,
		}),
		material("/api/v1/rcsa/cycles/{id}/first-line-distribution", "rcsa.first-line.bind", a.bindRCSAFirstLineDistribution, commandPolicy{
			ObjectType: "RCSA_CYCLE", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/rcsa/cycles/{id}/first-line-complete", "rcsa.first-line.complete", a.completeRCSAFirstLine, commandPolicy{
			ObjectType: "RCSA_CYCLE", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
	}
}
