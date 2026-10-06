package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
)

func (a *API) riskRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/risks", a.listRisks),
		read("/api/v1/risk-indicators", a.listRiskIndicators),
		read("/api/v1/risks/{id}", a.getRisk),
		material("/api/v1/risks", "risk.create", a.createRisk, commandPolicy{
			ObjectType: "RISK", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, BindLegalEntity: true, ActorField: noActorField,
		}),
		material("/api/v1/risks/{id}", "risk.update", a.updateRisk, commandPolicy{
			ObjectType: "RISK", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/risks/{id}/assessments", "risk.assess", a.assessRisk, commandPolicy{
			ObjectType: "RISK", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/risks/{id}/appetite", "risk.appetite.activate", a.activateRiskAppetite, commandPolicy{
			ObjectType: "RISK", ObjectIDPath: "id", Responsibility: authority.ResponsibilityAuthorizer,
			Materiality: 4, ActorField: noActorField,
		}),
		material("/api/v1/risks/{id}/controls", "risk.control.link", a.linkRiskControl, commandPolicy{
			ObjectType: "RISK", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
		material("/api/v1/risks/{id}/indicators", "risk.indicator.link", a.linkRiskIndicator, commandPolicy{
			ObjectType: "RISK", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner,
			Materiality: 3, ActorField: noActorField,
		}),
	}
}
