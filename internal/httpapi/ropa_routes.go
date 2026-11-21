package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) ropaRoutes() []routeSpec {
	return []routeSpec{
		withPermission(read("/api/v1/ropa/dashboard", a.getRopaDashboard), identity.PermissionPrivacyRead),
		withPermission(read("/api/v1/ropa/processing-activities", a.listRopaProcessingActivities), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/ropa/processing-activities", "ropa.processing_activity.create", a.createRopaProcessingActivity, commandPolicy{
			ObjectType:      "PROCESSING_ACTIVITY",
			Responsibility:  authority.ResponsibilityOwner,
			Materiality:     3,
			BindLegalEntity: true,
		}), identity.PermissionPrivacyRead),
		withPermission(read("/api/v1/ropa/processing-activities/{id}", a.getRopaProcessingActivity), identity.PermissionPrivacyRead),
		withPermission(read("/api/v1/ropa/processing-activities/{id}/history", a.getRopaProcessingActivityHistory), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/ropa/processing-activities/{id}", "ropa.processing_activity.update", a.updateRopaProcessingActivity, commandPolicy{
			ObjectType:     "PROCESSING_ACTIVITY",
			ObjectIDPath:   "id",
			Responsibility: authority.ResponsibilityOwner,
			Materiality:    3,
		}), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/ropa/processing-activities/{id}/transition", "ropa.processing_activity.transition", a.transitionRopaProcessingActivity, commandPolicy{
			ObjectType:     "PROCESSING_ACTIVITY",
			ObjectIDPath:   "id",
			Responsibility: authority.ResponsibilityAuthorizer,
			Materiality:    4,
		}), identity.PermissionPrivacyRead),
	}
}
