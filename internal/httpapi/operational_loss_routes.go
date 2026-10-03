package httpapi

import "github.com/CloudSpaceLab/clearsight-grc/internal/authority"

func (a *API) operationalLossRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/losses", a.listOperationalLosses),
		read("/api/v1/losses/{id}", a.getOperationalLoss),
		material("/api/v1/losses", "loss.create", a.createOperationalLoss, commandPolicy{
			ObjectType: "OPERATIONAL_LOSS", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, BindLegalEntity: true, ActorField: noActorField,
		}),
		material("/api/v1/losses/{id}", "loss.update", a.updateOperationalLoss, commandPolicy{
			ObjectType: "OPERATIONAL_LOSS", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/losses/{id}/intervention", "loss.intervention.open", a.openOperationalLossIntervention, commandPolicy{\n\t\t\tObjectType: "OPERATIONAL_LOSS", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,\n\t\t\tMateriality: 3, ActorField: noActorField,\n\t\t}),\n		material("/api/v1/losses/{id}/recoveries", "loss.recovery.record", a.recordOperationalLossRecovery, commandPolicy{
			ObjectType: "OPERATIONAL_LOSS", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
	}
}
