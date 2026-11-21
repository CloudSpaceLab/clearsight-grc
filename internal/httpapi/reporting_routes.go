package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) reportingRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/reports/filter-fields", a.listReportFilterFields),
		withPermission(read("/api/v1/reports/owners", a.listReportOwners), identity.PermissionPrivacyRead),
		withPermission(read("/api/v1/reports/definitions", a.listReportDefinitions), identity.PermissionPrivacyRead),
		read("/api/v1/reports/matter-board-brief", a.getMatterBoardBriefAvailability),
		withPermission(material("/api/v1/reports/definitions", "report.definition.propose", a.proposeReportDefinition, commandPolicy{
			ObjectType: "REPORT_DEFINITION", Responsibility: authority.ResponsibilityProposer,
			Materiality: 4, BindLegalEntity: true, ActorField: "maker_id",
		}), identity.PermissionPrivacyRead),
		withPermission(read("/api/v1/reports/definitions/{id}", a.getReportDefinition), identity.PermissionPrivacyRead),
		withPermission(read("/api/v1/reports/definitions/{id}/history", a.getReportDefinitionHistory), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/reports/definitions/{id}/submit", "report.definition.submit", a.reportDefinitionAction("submit"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityProposer, Materiality: 4,
		}), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/reports/definitions/{id}/review", "report.definition.review", a.reportDefinitionAction("review"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer, Materiality: 4,
		}), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/reports/definitions/{id}/activate", "report.definition.activate", a.reportDefinitionAction("activate"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5,
		}), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/reports/definitions/{id}/reject", "report.definition.reject", a.reportDefinitionAction("reject"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer, Materiality: 4,
		}), identity.PermissionPrivacyRead),
		withPermission(material("/api/v1/reports/definitions/{id}/retire", "report.definition.retire", a.reportDefinitionAction("retire"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5,
		}), identity.PermissionPrivacyRead),
		withPermission(read("/api/v1/reports/runs", a.listReportRuns), identity.PermissionPrivacyRead),
		material("/api/v1/reports/runs", "report.run.create", a.createReportRun, commandPolicy{
			ObjectType: "REPORT_RUN", Responsibility: authority.ResponsibilityPerformer, Materiality: 3, BindLegalEntity: true,
		}),
		withPermission(read("/api/v1/reports/runs/{id}", a.getReportRun), identity.PermissionPrivacyRead),
		read("/api/v1/reports/runs/{id}/download", a.downloadReportRun),
	}
}
