package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (a *API) reportingRoutes() []routeSpec {
	return []routeSpec{
		read("/api/v1/ropa/reports/filter-fields", a.listReportFilterFields),
		read("/api/v1/ropa/reports/definitions", a.listReportDefinitions),
		material("/api/v1/ropa/reports/definitions", "report.definition.propose", a.proposeReportDefinition, commandPolicy{
			ObjectType: "REPORT_DEFINITION", Responsibility: authority.ResponsibilityProposer,
			Materiality: 4, BindLegalEntity: true, ActorField: "maker_id",
		}),
		read("/api/v1/ropa/reports/definitions/{id}", a.getReportDefinition),
		read("/api/v1/ropa/reports/definitions/{id}/history", a.getReportDefinitionHistory),
		material("/api/v1/ropa/reports/definitions/{id}/submit", "report.definition.submit", a.reportDefinitionAction("submit"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityProposer, Materiality: 4,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/review", "report.definition.review", a.reportDefinitionAction("review"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer, Materiality: 4,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/activate", "report.definition.activate", a.reportDefinitionAction("activate"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/reject", "report.definition.reject", a.reportDefinitionAction("reject"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer, Materiality: 4,
		}),
		material("/api/v1/ropa/reports/definitions/{id}/retire", "report.definition.retire", a.reportDefinitionAction("retire"), commandPolicy{
			ObjectType: "REPORT_DEFINITION", ObjectIDPath: "id", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5,
		}),
		read("/api/v1/ropa/reports/runs", a.listReportRuns),
		material("/api/v1/ropa/reports/runs", "report.run.create", a.createReportRun, commandPolicy{
			ObjectType: "REPORT_RUN", Responsibility: authority.ResponsibilityPerformer, Materiality: 3, BindLegalEntity: true,
		}),
		read("/api/v1/ropa/reports/runs/{id}", a.getReportRun),
		withPermission(read("/api/v1/ropa/reports/runs/{id}/download", a.downloadReportRun), identity.PermissionReportDownload),
	}
}
