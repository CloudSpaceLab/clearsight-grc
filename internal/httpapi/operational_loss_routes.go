package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) operationalLossRoutes() []routeSpec {
	return []routeSpec{
		withPermission(read("/api/v1/losses", a.listOperationalLosses), identity.PermissionOversightRead),
		withPermission(read("/api/v1/losses/{id}", a.getOperationalLoss), identity.PermissionOversightRead),
		material("/api/v1/losses", "loss.create", a.createOperationalLoss, commandPolicy{
			ObjectType: "OPERATIONAL_LOSS", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, BindLegalEntity: true, ActorField: noActorField,
		}),
		material("/api/v1/losses/{id}", "loss.update", a.updateOperationalLoss, commandPolicy{
			ObjectType: "OPERATIONAL_LOSS", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/losses/{id}/intervention", "loss.intervention.open", a.openOperationalLossIntervention, commandPolicy{
			ObjectType: "OPERATIONAL_LOSS", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/losses/{id}/recoveries", "loss.recovery.record", a.recordOperationalLossRecovery, commandPolicy{
			ObjectType: "OPERATIONAL_LOSS", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
	}
}
