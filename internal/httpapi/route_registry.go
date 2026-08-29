package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

type routeClass string

const (
	routePublic                    routeClass = "PUBLIC"
	routeDemoOnly                  routeClass = "DEMO_ONLY"
	routeAuthenticatedRead         routeClass = "AUTHENTICATED_READ"
	routeAuthenticatedOperation    routeClass = "AUTHENTICATED_OPERATION"
	routeAuthenticatedWrite        routeClass = "AUTHENTICATED_WRITE"
	routeMaterialCommand           routeClass = "MATERIAL_COMMAND"
	routeCapability                routeClass = "BOUNDED_CAPABILITY"
	routeAuthenticatedOrCapability routeClass = "AUTHENTICATED_OR_CAPABILITY"
)

type routeBinder func(http.ResponseWriter, *http.Request, identity.Actor) bool

type routeCommand struct {
	Name   string
	Policy commandPolicy
}

type routeSpec struct {
	Method     string
	Path       string
	Class      routeClass
	Handler    http.HandlerFunc
	Binder     routeBinder
	Command    *routeCommand
	Permission string
	RawCommand bool
}

func (a *API) registerRoutes(mux *http.ServeMux) {
	routes := a.routes()
	if err := validateRoutes(routes); err != nil {
		panic(err)
	}
	for _, spec := range routes {
		handler := spec.Handler
		if spec.Command != nil {
			if spec.RawCommand {
				handler = a.rawCommand(spec.Command.Name, spec.Command.Policy, handler)
			} else {
				handler = a.command(spec.Command.Name, spec.Command.Policy, handler)
			}
		}
		handler = a.routeAccess(spec, handler)
		mux.HandleFunc(spec.Method+" "+spec.Path, handler)
	}
}

func (a *API) routes() []routeSpec {
	routes := []routeSpec{
		public(http.MethodGet, "/health/live", a.live),
		public(http.MethodGet, "/health/ready", a.ready),
		public(http.MethodGet, "/api/v1/session/status", a.sessionStatus),
		read("/api/v1/context", a.actorContext),
		read("/api/v1/today", a.actorToday),

		operation("/api/v1/authority/resolve", a.resolveAuthority, bindJSONIdentity(true)),
		withPermission(operation("/api/v1/authority/simulate", a.simulateAuthority, bindJSONIdentity(true)), identity.PermissionConfigRead),
		withPermission(read("/api/v1/authority/integrity", a.authorityIntegrity), identity.PermissionConfigRead),
		withPermission(read("/api/v1/authority/policies", a.authorityPolicies), identity.PermissionConfigRead),

		withPermission(read("/api/v1/governance/policies", a.listGovernancePolicies), identity.PermissionConfigRead),
		withPermission(material("/api/v1/governance/policies", "governance.policy.create", a.createGovernancePolicy, commandPolicy{ObjectType: "ROUTING_POLICY", Responsibility: authority.ResponsibilityProposer, Materiality: 4, BindLegalEntity: true, ActorField: "maker_id"}), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/policies/{id}/submit", "governance.policy.submit", a.governancePolicyAction("submit"), commandPolicy{ObjectType: "ROUTING_POLICY", Responsibility: authority.ResponsibilityProposer, Materiality: 4, BindLegalEntity: true}), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/policies/{id}/approve", "governance.policy.approve", a.governancePolicyAction("approve"), commandPolicy{ObjectType: "ROUTING_POLICY", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5, BindLegalEntity: true}), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/policies/{id}/reject", "governance.policy.reject", a.governancePolicyAction("reject"), commandPolicy{ObjectType: "ROUTING_POLICY", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5, BindLegalEntity: true}), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/policies/{id}/retire", "governance.policy.retire", a.governancePolicyAction("retire"), commandPolicy{ObjectType: "ROUTING_POLICY", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5, BindLegalEntity: true}), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/governance/delegations", a.listGovernanceDelegations), identity.PermissionConfigRead),
		withPermission(read("/api/v1/governance/delegation-candidates", a.listGovernanceDelegationCandidates), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/delegations", "governance.delegation.create", a.createGovernanceDelegation, commandPolicy{ObjectType: "DELEGATION", Responsibility: authority.ResponsibilityProposer, Materiality: 4, BindLegalEntity: true, ActorField: "maker_id"}), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/delegations/{id}/submit", "governance.delegation.submit", a.governanceDelegationAction("submit"), commandPolicy{ObjectType: "DELEGATION", Responsibility: authority.ResponsibilityProposer, Materiality: 4, BindLegalEntity: true}), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/delegations/{id}/approve", "governance.delegation.approve", a.governanceDelegationAction("approve"), commandPolicy{ObjectType: "DELEGATION", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5, BindLegalEntity: true}), identity.PermissionConfigWrite),
		withPermission(material("/api/v1/governance/delegations/{id}/revoke", "governance.delegation.revoke", a.governanceDelegationAction("revoke"), commandPolicy{ObjectType: "DELEGATION", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5, BindLegalEntity: true}), identity.PermissionConfigWrite),

		withPermission(read("/api/v1/ai-governance/policies", a.listAIGovernancePolicies), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/policies", a.createAIGovernancePolicy, bindJSONIdentity(false, "maker_id")), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/ai-governance/policies/{id}", a.getAIGovernancePolicy), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/policies/{id}/submit", a.aiGovernancePolicyAction("submit"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/policies/{id}/approve", a.aiGovernancePolicyAction("approve"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/policies/{id}/activate", a.aiGovernancePolicyAction("activate"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/policies/{id}/suspend", a.aiGovernancePolicyAction("suspend"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/policies/{id}/retire", a.aiGovernancePolicyAction("retire"), nil), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/ai-governance/workloads", a.listAIGovernanceWorkloads), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/workloads", a.createAIGovernanceWorkload, bindJSONIdentity(false, "maker_id")), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/ai-governance/workloads/{id}", a.getAIGovernanceWorkload), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/workloads/{id}/submit", a.aiGovernanceWorkloadAction("submit"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/workloads/{id}/approve", a.aiGovernanceWorkloadAction("approve"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/workloads/{id}/activate", a.aiGovernanceWorkloadAction("activate"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/workloads/{id}/suspend", a.aiGovernanceWorkloadAction("suspend"), nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/ai-governance/workloads/{id}/retire", a.aiGovernanceWorkloadAction("retire"), nil), identity.PermissionConfigWrite),
		materialService("/api/v1/ai-governance/receipts", "ai.governance.receipt.ingest", a.ingestAIGovernanceReceipt, commandPolicy{ObjectType: "AI_WORKLOAD", Responsibility: authority.ResponsibilityPerformer, Materiality: 1, ActorField: noActorField}),
		material("/api/v1/ai-governance/execution-grants", "ai.governance.grant.create", a.createAIGovernanceGrant, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 5, ActorField: "actor_id"}),

		read("/api/v1/program-summaries", a.listProgramSummaries),
		read("/api/v1/programs", a.listPrograms),
		read("/api/v1/programs/setup-candidates", a.listProgramSetupCandidates),
		material("/api/v1/programs", "program.create", a.createProgram, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2, BindLegalEntity: true}),
		read("/api/v1/programs/{id}", a.getProgram),
		read("/api/v1/programs/{id}/history", a.getProgramHistory),
		read("/api/v1/programs/{id}/operations", a.getProgramOperations),
		read("/api/v1/programs/{id}/review-digest", a.getProgramReviewDigest),
		material("/api/v1/programs/{id}/reviews", "program.review.accept", a.acceptProgramReview, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: noActorField}),
		material("/api/v1/programs/{id}/details", "program.details.update", a.updateProgramDetails, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/assignment", "program.assign", a.assignProgram, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/programs/{id}/approval-authority", "program.approval-authority.assign", a.assignProgramApprovalAuthority, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4, DecisionType: "program.transition"}),
		material("/api/v1/programs/{id}/transition", "program.transition", a.transitionProgram, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3}),
		material("/api/v1/programs/{id}/requirements", "program.requirement.add", a.addProgramRequirement, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/requirements/{requirement_id}/supersede", "program.requirement.supersede", a.supersedeProgramRequirement, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/applicability", "program.applicability.decide", a.determineProgramApplicability, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3, ActorField: "approved_by"}),
		material("/api/v1/programs/{id}/control-objectives", "program.safeguard.define", a.addProgramControlObjective, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/control-implementations", "program.safeguard.define", a.addProgramControlImplementation, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/control-implementations/{implementation_id}/details", "program.safeguard.update", a.reviseProgramControlImplementation, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/control-implementations/{implementation_id}/assignment", "program.safeguard.assign", a.assignProgramControlImplementation, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/programs/{id}/control-implementations/{implementation_id}/transition", "program.safeguard.transition", a.transitionProgramControlImplementation, commandPolicy{ObjectType: "CONTROL_IMPLEMENTATION", ObjectIDPath: "implementation_id", Responsibility: authority.ResponsibilityPerformer, Materiality: 3}),
		material("/api/v1/programs/{id}/control-links", "program.safeguard.define", a.linkProgramRequirementControl, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/control-links/{link_id}/retirement", "program.safeguard.unlink", a.retireProgramRequirementControlLink, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/programs/{id}/evidence-contracts", "program.evidence.define", a.addProgramEvidenceContract, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/evidence-contracts/{contract_id}/revision", "program.evidence.revise", a.reviseProgramEvidenceContract, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/evidence-contracts/{contract_id}/transition", "program.evidence.transition", a.transitionProgramEvidenceContract, commandPolicy{ObjectType: "EVIDENCE_CONTRACT", ObjectIDPath: "contract_id", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/programs/{id}/evidence-contracts/{contract_id}/assessments", "program.evidence.assess", a.recordProgramEvidenceAssessment, commandPolicy{ObjectType: "EVIDENCE_CONTRACT", ObjectIDPath: "contract_id", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: "assessed_by"}),
		materialService("/api/v1/programs/{id}/triggers", "program.trigger.ingest", a.applyProgramTrigger, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityPerformer, Materiality: 2}),

		read("/api/v1/programs/{id}/form-templates", a.listFormTemplates),
		material("/api/v1/programs/{id}/form-templates", "program.monitoring.form.define", a.createFormTemplate, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),
		material("/api/v1/programs/{id}/form-templates/{form_id}/transition", "program.monitoring.form.transition", a.transitionFormTemplate, commandPolicy{ObjectType: "FORM_TEMPLATE", ObjectIDPath: "form_id", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: noActorField}),
		material("/api/v1/programs/{id}/form-templates/{form_id}/collections", "program.monitoring.collect", a.startFormCollection, commandPolicy{ObjectType: "FORM_TEMPLATE", ObjectIDPath: "form_id", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),

		read("/api/v1/vendors", a.listVendorRelationships),
		material("/api/v1/vendors", "thirdparty.relationship.create", a.createVendorRelationship, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 3, BindLegalEntity: true}),
		read("/api/v1/vendors/{id}", a.getVendorRelationship),
		material("/api/v1/vendors/{id}", "thirdparty.relationship.update", a.updateVendorRelationship, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		read("/api/v1/vendor-identities/{vendor_id}", a.getVendorIdentity),
		materialMethod(http.MethodPut, "/api/v1/vendor-identities/{vendor_id}", thirdparty.VendorIdentityUpdateCommand, a.updateVendorIdentity, commandPolicy{ObjectType: thirdparty.VendorIdentityObjectType, Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		read("/api/v1/vendor-identities/{vendor_id}/brand", a.openVendorBrand),
		materialBinary(http.MethodPut, "/api/v1/vendor-identities/{vendor_id}/brand", thirdparty.VendorBrandApproveCommand, a.uploadVendorBrand, commandPolicy{ObjectType: thirdparty.VendorIdentityObjectType, OutcomeObjectType: "VENDOR_BRAND", OutcomePathValue: "vendor_id", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		materialBinary(http.MethodDelete, "/api/v1/vendor-identities/{vendor_id}/brand", thirdparty.VendorBrandRemoveCommand, a.removeVendorBrand, commandPolicy{ObjectType: thirdparty.VendorIdentityObjectType, OutcomeObjectType: "VENDOR_BRAND", OutcomePathValue: "vendor_id", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		read("/api/v1/vendors/{id}/links", a.listVendorRelationshipLinks),
		read("/api/v1/vendor-links", a.listVendorRelationshipLinks),
		material("/api/v1/vendors/{id}/links", "thirdparty.relationship.link", a.linkVendorRelationship, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/vendors/{id}/links/{link_id}/end", "thirdparty.relationship.unlink", a.endVendorRelationshipLink, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		read("/api/v1/vendor-work", a.listVendorWork),
		read("/api/v1/vendor-work/{request_id}", a.getVendorWork),
		read("/api/v1/vendors/{id}/work/{request_id}/response", a.getVendorWorkResponse),
		read("/api/v1/vendors/{id}/work/{request_id}/requests/{capture_request_id}/documents/{artifact_id}/open", a.openVendorWorkDocument),
		material("/api/v1/vendors/{id}/work/prepare", "thirdparty.work.prepare", a.prepareVendorWork, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/vendors/{id}/work/{request_id}/send", "thirdparty.work.send", a.sendVendorWork, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 3, OutcomeObjectType: "VENDOR_WORK_REQUEST", OutcomePathValue: "request_id"}),
		material("/api/v1/vendors/{id}/work/{request_id}/review/start", "thirdparty.work.review", a.startVendorWorkReview, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, OutcomeObjectType: "VENDOR_WORK_REQUEST", OutcomePathValue: "request_id"}),
		material("/api/v1/vendors/{id}/work/{request_id}/changes", "thirdparty.work.request_changes", a.requestVendorWorkChanges, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, OutcomeObjectType: "VENDOR_WORK_REQUEST", OutcomePathValue: "request_id"}),
		material("/api/v1/vendors/{id}/work/{request_id}/accept", "thirdparty.work.accept", a.acceptVendorWork, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, OutcomeObjectType: "VENDOR_WORK_REQUEST", OutcomePathValue: "request_id"}),
		material("/api/v1/vendors/{id}/work/{request_id}/cancel", "thirdparty.work.cancel", a.cancelVendorWork, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 3, OutcomeObjectType: "VENDOR_WORK_REQUEST", OutcomePathValue: "request_id"}),
		material("/api/v1/vendors/{id}/work/{request_id}/retry", "thirdparty.work.retry", a.retryVendorWork, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 3, OutcomeObjectType: "VENDOR_WORK_REQUEST", OutcomePathValue: "request_id"}),
		material("/api/v1/vendors/{id}/assessments", "thirdparty.assessment.start", a.startVendorAssessment, commandPolicy{ObjectType: "VENDOR_RELATIONSHIP", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		read("/api/v1/vendors/{id}/assessments/current", a.getCurrentVendorAssessment),
		material("/api/v1/vendor-assessments/{id}/send-request", "thirdparty.assessment.send_request", a.sendVendorAssessmentRequest, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/vendor-assessments/{id}/reissue-request", "thirdparty.assessment.reissue_request", a.reissueVendorAssessmentRequest, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/vendor-assessments/{id}/setup/retry", thirdparty.AssessmentSetupRetryCommand, a.retryVendorAssessmentSetup, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		read("/api/v1/vendor-assessments/{id}", a.getVendorAssessmentReview),
		read("/api/v1/vendor-assessments/{id}/requests/{request_id}/documents/{artifact_id}/open", a.openVendorAssessmentDocument),
		material("/api/v1/vendor-assessments/{id}/review/start", "thirdparty.assessment.review", a.startVendorAssessmentReview, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/vendor-assessments/{id}/documents/{artifact_id}/validate", thirdparty.AssessmentDocumentReviewCommand, a.reviewVendorAssessmentDocument, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/vendor-assessments/{id}/responses/{revision_id}/apply", thirdparty.AssessmentApplyResponseCommand, a.applyVendorAssessmentResponse, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: noActorField}),
		material("/api/v1/vendor-assessments/{id}/clarifications", thirdparty.AssessmentClarificationCommand, a.requestVendorAssessmentClarification, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/vendor-assessments/{id}/deficiencies", thirdparty.AssessmentDeficiencyCommand, a.createVendorAssessmentDeficiency, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/vendor-assessments/{id}/complete", "thirdparty.assessment.complete", a.completeVendorAssessment, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/vendor-assessments/{id}/cancel", thirdparty.AssessmentCancelCommand, a.cancelVendorAssessment, commandPolicy{ObjectType: "THIRD_PARTY_ASSESSMENT", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),

		read("/api/v1/form-templates", a.listReusableFormTemplates),
		read("/api/v1/forms/templates", a.listLibraryForms),
		material("/api/v1/forms/templates", "forms.template.create", a.createLibraryForm, commandPolicy{ObjectType: "LEGAL_ENTITY", Responsibility: authority.ResponsibilityOwner, Materiality: 2, BindLegalEntity: true, ActorField: noActorField}),
		read("/api/v1/forms/templates/{id}/revisions/{version}", a.getLibraryFormRevision),
		material("/api/v1/forms/templates/{id}/revisions", "forms.template.revise", a.createLibraryFormRevision, commandPolicy{ObjectType: "FORM_TEMPLATE", ObjectIDPath: "id", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),
		material("/api/v1/forms/templates/{id}/transition", "forms.template.transition", a.transitionLibraryForm, commandPolicy{ObjectType: "FORM_TEMPLATE", ObjectIDPath: "id", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: noActorField}),
		read("/api/v1/forms/starter-templates", a.listStarterForms),
		material("/api/v1/forms/starter-templates/{code}/instantiate", "forms.starter.instantiate", a.instantiateStarterForm, commandPolicy{ObjectType: "LEGAL_ENTITY", Responsibility: authority.ResponsibilityOwner, Materiality: 2, BindLegalEntity: true, ActorField: noActorField}),
		read("/api/v1/forms/saved-views", a.listSavedFormViews),
		write(http.MethodPost, "/api/v1/forms/saved-views", a.saveFormView, nil),
		write(http.MethodDelete, "/api/v1/forms/saved-views/{id}", a.deleteSavedFormView, nil),
		read("/api/v1/programs/{id}/monitoring-checks", a.listMonitoringChecks),
		material("/api/v1/programs/{id}/monitoring-checks", "program.monitoring.define", a.createMonitoringCheck, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),
		material("/api/v1/monitoring-checks/{id}/transition", "program.monitoring.transition", a.transitionMonitoringCheck, commandPolicy{ObjectType: "MONITORING_CHECK", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: noActorField}),
		material("/api/v1/monitoring-checks/{id}/evaluate-source", "program.monitoring.evaluate", a.evaluateMonitoringSource, commandPolicy{ObjectType: "MONITORING_CHECK", Responsibility: authority.ResponsibilityPerformer, Materiality: 2, ActorField: noActorField}),
		read("/api/v1/monitoring-checks/{id}/results", a.listMonitoringResults),
		material("/api/v1/monitoring-results/{result_id}/linked-issue", "program.monitoring.issue.create", a.createMonitoringLinkedIssue, commandPolicy{ObjectType: "MONITORING_CHECK", Responsibility: authority.ResponsibilityReviewer, Materiality: 4, ActorField: noActorField}),

		read("/api/v1/matter-summaries", a.listMatterSummaries),
		read("/api/v1/matters", a.listMatters),
		material("/api/v1/matters", "matter.create", a.createMatter, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3, BindLegalEntity: true}),
		read("/api/v1/matters/{id}", a.getMatter),
		read("/api/v1/matters/{id}/history", a.getMatterHistory),
		read("/api/v1/matters/{id}/operations", a.getMatterOperations),
		material("/api/v1/matters/{id}/details", "matter.details.update", a.updateMatterDetails, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/context-changes", "matter.context.change", a.changeMatterContext, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/assignment", "matter.assign", a.assignMatter, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/matters/{id}/transition", "matter.transition", a.transitionMatter, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/matters/{id}/links", "matter.link", a.addMatterLink, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/links/{link_id}/retirement", "matter.unlink", a.retireMatterLink, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/matters/{id}/decisions", "matter.decision.record", a.addMatterDecision, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4, ActorField: "authority_principal_id"}),
		material("/api/v1/matters/{id}/actions", "matter.action.add", a.addMatterAction, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/actions/{action_id}", "matter.action.update", a.updateMatterAction, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/actions/{action_id}/assignment", "matter.action.assign", a.assignMatterAction, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/matters/{id}/actions/{action_id}/transition", "matter.action.transition", a.transitionMatterAction, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/verification-contracts", "matter.outcome.define", a.addMatterVerificationContract, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/matters/{id}/verification-contracts/{contract_id}/supersede", "matter.outcome.supersede", a.supersedeMatterVerificationContract, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/matters/{id}/verification-contracts/{contract_id}/retire", "matter.outcome.retire", a.retireMatterVerificationContract, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/matters/{id}/verification-results", "matter.outcome.record", a.recordMatterVerificationResult, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 4, ActorField: "reviewer_principal_id"}),
		material("/api/v1/matters/{id}/responses", "matter.response.add", a.addMatterResponse, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		read("/api/v1/matters/{id}/responses/{response_id}/history", a.getMatterResponseHistory),
		material("/api/v1/matters/{id}/responses/{response_id}/transition", "matter.response.transition", a.transitionMatterResponse, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilitySignatory, Materiality: 4}),

		withPermission(read("/api/v1/operations/projections", a.projectionHealth), identity.PermissionPlatformOperationsRead),
		withPermission(materialService("/api/v1/operations/projections/reconcile", "projection.reconcile", a.reconcileProgramState, commandPolicy{ObjectType: "PROJECTION", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: noActorField}), identity.PermissionPlatformOperationsWrite),
		withPermission(material("/api/v1/operations/projections/rebuild", "projection.rebuild", a.rebuildProgramState, commandPolicy{ObjectType: "PROJECTION", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4, ActorField: noActorField}), identity.PermissionPlatformOperationsWrite),
		withPermission(read("/api/v1/operations/background-jobs", a.backgroundJobs), identity.PermissionPlatformJobsRead),

		withPermission(read("/api/v1/config/sources/{source_id}/connections", a.listSourceConnections), identity.PermissionConfigRead),
		withPermission(read("/api/v1/config/sources/{source_id}/health", a.sourceScopeHealth), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/config/sources/{source_id}/connections", a.createSourceConnectionDraft, nil), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/config/source-connections/{connection_id}", a.getSourceConnection), identity.PermissionConfigRead),
		withPermission(read("/api/v1/config/source-connections/{connection_id}/views", a.listSourceViews), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/config/source-connections/{connection_id}/views", a.createSourceViewDraft, nil), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/config/source-connections/{connection_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageConnection, "connection_id")), identity.PermissionConfigRead),
		withPermission(read("/api/v1/config/source-views/{view_id}", a.getSourceView), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/config/source-views/{view_id}/inspect", a.inspectSourceView, nil), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/config/source-views/{view_id}/bindings", a.listSourceBindings), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/config/source-views/{view_id}/bindings", a.createSourceBindingDraft, nil), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/config/source-views/{view_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageView, "view_id")), identity.PermissionConfigRead),
		withPermission(read("/api/v1/config/source-bindings/{binding_id}", a.getSourceBinding), identity.PermissionConfigRead),
		withPermission(operation("/api/v1/config/source-bindings/{binding_id}/preview", a.previewSourceBinding, nil), identity.PermissionConfigRead),
		withPermission(read("/api/v1/config/source-bindings/{binding_id}/where-used", a.sourceCatalogWhereUsed(sourceaccess.UsageBinding, "binding_id")), identity.PermissionConfigRead),
		materialService("/api/v1/source-bindings/{id}/events", "source.binding.event.ingest", a.ingestSourceBindingEvent, commandPolicy{ObjectType: "SOURCE_BINDING", Responsibility: authority.ResponsibilityPerformer, Materiality: 2, ActorField: noActorField}),

		read("/api/v1/evidence/sources", a.listEvidenceSources),
		write(http.MethodPost, "/api/v1/evidence/sources", a.createEvidenceSource, bindJSONIdentity(true)),
		write(http.MethodPost, "/api/v1/evidence/sources/{id}/observations", a.recordEvidenceSourceObservation, bindJSONIdentity(false, "recorded_by")),
		read("/api/v1/evidence/requests", a.listEvidenceRequests),
		write(http.MethodPost, "/api/v1/evidence/requests", a.createEvidenceRequest, bindJSONIdentity(true, "created_by")),
		read("/api/v1/evidence/requests/{id}", a.getEvidenceRequest),
		read("/api/v1/evidence/requests/{id}/recipient-candidates", a.listEvidenceRecipientCandidates),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/wrong-recipient", a.declareEvidenceWrongRecipient, bindJSONIdentity(true, "actor_principal_id")),
		write(http.MethodPut, "/api/v1/evidence/requests/{id}/recipient", a.reassignEvidenceRecipient, bindJSONIdentity(true, "actor_principal_id")),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/submissions", a.submitEvidenceRequest, bindJSONIdentity(true, "submitted_by")),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/invitations", a.issueEvidenceInvitation, bindJSONIdentity(true, "created_by")),
		read("/api/v1/evidence/requests/{id}/invitations", a.listEvidenceInvitationMetadata),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/invitations/{invitation_id}/replace", a.replaceEvidenceInvitation, nil),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/invitations/{invitation_id}/revoke", a.revokeEvidenceInvitationAsRequester, nil),
		read("/api/v1/evidence/requests/{id}/sessions", a.listEvidenceActiveSessions),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/sessions/{session_id}/revoke", a.revokeEvidenceSessionAsRequester, nil),
		capability(http.MethodPost, "/api/v1/evidence/invitations/redeem", a.redeemEvidenceInvitation),
		capability(http.MethodGet, "/api/v1/evidence/session", a.getEvidenceSession),
		capability(http.MethodGet, "/api/v1/evidence/session/draft", a.getEvidenceSessionDraft),
		capability(http.MethodPut, "/api/v1/evidence/session/draft", a.saveEvidenceSessionDraft),
		capability(http.MethodPost, "/api/v1/evidence/session/submissions", a.submitEvidenceSession),
		hybridCapability(http.MethodPost, "/api/v1/evidence/artifacts", a.uploadEvidenceArtifact, a.bindArtifactIdentity()),

		read("/api/v1/document-imports", a.listDocumentImports),
		write(http.MethodPost, "/api/v1/document-imports", a.createDocumentImport, nil),
		read("/api/v1/document-imports/{id}", a.getDocumentImport),
		write(http.MethodPost, "/api/v1/document-imports/{id}/proposals/{proposal_id}/review", a.reviewDocumentProposal, nil),
		material("/api/v1/document-imports/{id}/proposals/{proposal_id}/handoff/review", "document.proposal.review", a.reviewDocumentProposalHandoff, commandPolicy{ObjectType: "DOCUMENT_IMPORT", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/document-imports/{id}/proposals/{proposal_id}/handoff/authorize", "document.proposal.authorize", a.authorizeDocumentProposalHandoff, commandPolicy{ObjectType: "DOCUMENT_IMPORT", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4}),
		read("/api/v1/document-imports/{id}/coverage", a.getDocumentCoverage),
		write(http.MethodPost, "/api/v1/document-imports/{id}/coverage/review", a.reviewDocumentCoverage, nil),
		write(http.MethodPost, "/api/v1/document-imports/{id}/coverage/recompare", a.recompareDocumentCoverage, nil),
		write(http.MethodPost, "/api/v1/document-imports/{id}/coverage/suggestions/{suggestion_id}/apply", a.applyDocumentCoverageSuggestion, nil),

		read("/api/v1/workflow/tasks", a.listWorkflowTasks),

		read("/api/v1/onboarding/guide", a.actorOnboardingGuide),
		readBound("/api/v1/onboarding/state", a.onboardingState, bindActorQuery("principal_id")),
		write(http.MethodPut, "/api/v1/onboarding/state", a.updateOnboardingState, bindActorQuery("principal_id")),

		read("/api/v1/compliance/readiness", a.readiness),
		withPermission(read("/api/v1/compliance/automation-policies", a.automationPolicies), identity.PermissionConfigRead),
		write(http.MethodPost, "/api/v1/compliance/signals", a.ingestSignal, bindJSONIdentity(false)),
	}

	routes = append(routes,
		withPermission(read("/api/v1/access/overview", a.identityAccessOverview), identity.PermissionIdentityRead),
		withPermission(write(http.MethodPost, "/api/v1/access/scim-sources", a.createSCIMSource, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/scim-sources/{id}/rotate-token", a.rotateSCIMSourceToken, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/scim-sources/{id}/revoke", a.revokeSCIMSource, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/group-role-bindings", a.createDirectoryGroupRoleBinding, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/group-role-bindings/{id}/retire", a.retireDirectoryGroupRoleBinding, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/escalation-guard-revisions", a.proposeEscalationGuardRevision, nil), identity.PermissionIdentityConfigure),
		withPermission(write(http.MethodPost, "/api/v1/access/escalation-guard-revisions/{policy_id}/{version}/approve", a.approveEscalationGuardRevision, nil), identity.PermissionIdentityConfigure),
		withPermission(operation("/api/v1/access/escalations/preview", a.previewEscalation, nil), identity.PermissionIdentityRead),
	)
	if a.deps.DemoMode {
		if _, ok := a.deps.Identity.(identity.DemoSessionAuthenticator); ok {
			routes = append(routes,
				demoOnly(http.MethodGet, "/api/v1/demo/accounts", a.demoAccounts),
				demoOnly(http.MethodPost, "/api/v1/demo/login", a.demoLogin),
				demoOnly(http.MethodPost, "/api/v1/demo/logout", a.demoLogout),
			)
		}
	}
	if a.deps.DemoMode && a.deps.BankVerticals != nil {
		routes = append(routes, read("/api/v1/bank-journeys", a.listBankJourneys))
	}
	return routes
}

func public(method, path string, handler http.HandlerFunc) routeSpec {
	return routeSpec{Method: method, Path: path, Class: routePublic, Handler: handler}
}
func demoOnly(method, path string, handler http.HandlerFunc) routeSpec {
	return routeSpec{Method: method, Path: path, Class: routeDemoOnly, Handler: handler}
}
func read(path string, handler http.HandlerFunc) routeSpec {
	return routeSpec{Method: http.MethodGet, Path: path, Class: routeAuthenticatedRead, Handler: handler}
}
func readBound(path string, handler http.HandlerFunc, binder routeBinder) routeSpec {
	return routeSpec{Method: http.MethodGet, Path: path, Class: routeAuthenticatedRead, Handler: handler, Binder: binder}
}
func operation(path string, handler http.HandlerFunc, binder routeBinder) routeSpec {
	return routeSpec{Method: http.MethodPost, Path: path, Class: routeAuthenticatedOperation, Handler: handler, Binder: binder}
}
func write(method, path string, handler http.HandlerFunc, binder routeBinder) routeSpec {
	return routeSpec{Method: method, Path: path, Class: routeAuthenticatedWrite, Handler: handler, Binder: binder}
}
func material(path, name string, handler http.HandlerFunc, policy commandPolicy) routeSpec {
	return materialMethod(http.MethodPost, path, name, handler, policy)
}
func materialMethod(method, path, name string, handler http.HandlerFunc, policy commandPolicy) routeSpec {
	return routeSpec{Method: method, Path: path, Class: routeMaterialCommand, Handler: handler, Command: &routeCommand{Name: name, Policy: policy}}
}
func materialBinary(method, path, name string, handler http.HandlerFunc, policy commandPolicy) routeSpec {
	spec := materialMethod(method, path, name, handler, policy)
	spec.RawCommand = true
	return spec
}
func materialService(path, name string, handler http.HandlerFunc, policy commandPolicy) routeSpec {
	policy.AllowService = true
	return material(path, name, handler, policy)
}
func capability(method, path string, handler http.HandlerFunc) routeSpec {
	return routeSpec{Method: method, Path: path, Class: routeCapability, Handler: handler}
}
func hybridCapability(method, path string, handler http.HandlerFunc, binder routeBinder) routeSpec {
	return routeSpec{Method: method, Path: path, Class: routeAuthenticatedOrCapability, Handler: handler, Binder: binder}
}
func withPermission(spec routeSpec, permission string) routeSpec {
	spec.Permission = permission
	return spec
}

func validateRoutes(routes []routeSpec) error {
	seen := make(map[string]struct{}, len(routes))
	for _, route := range routes {
		if route.Method == "" || route.Path == "" || route.Class == "" || route.Handler == nil {
			return fmt.Errorf("invalid route registration for %s %s", route.Method, route.Path)
		}
		key := route.Method + " " + route.Path
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate route registration: %s", key)
		}
		seen[key] = struct{}{}
		if route.Class == routePublic && route.Method != http.MethodGet {
			return fmt.Errorf("public mutating route is prohibited: %s", key)
		}
		if route.Class == routeDemoOnly && !strings.HasPrefix(route.Path, "/api/v1/demo/") {
			return fmt.Errorf("demo-only route must use the demo namespace: %s", key)
		}
		if (route.Class == routePublic || route.Class == routeCapability || route.Class == routeDemoOnly) && route.Permission != "" {
			return fmt.Errorf("public, demo-only or capability route cannot require staff permission: %s", key)
		}
		if route.Class == routeMaterialCommand && route.Command == nil {
			return fmt.Errorf("material route lacks command policy: %s", key)
		}
		if route.Class != routeMaterialCommand && route.Command != nil {
			return fmt.Errorf("non-material route declares command policy: %s", key)
		}
	}
	return nil
}

func (a *API) routeAccess(spec routeSpec, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch spec.Class {
		case routePublic, routeCapability, routeDemoOnly:
			handler(w, r)
			return
		case routeAuthenticatedOrCapability:
			if optionalBearerToken(r) != "" {
				handler(w, r)
				return
			}
		}
		actor, err := identity.Require(r.Context())
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue.")
			return
		}
		if spec.Permission != "" && !identity.HasPermission(actor, spec.Permission) {
			httpx.WriteError(w, http.StatusForbidden, "permission_required", "You do not have permission to use this administrative function.")
			return
		}
		if !bindRouteTenant(w, r, actor) {
			return
		}
		if spec.Binder != nil && !spec.Binder(w, r, actor) {
			return
		}
		handler(w, r)
	}
}

func bindRouteTenant(w http.ResponseWriter, r *http.Request, actor identity.Actor) bool {
	query := r.URL.Query()
	if tenant := strings.TrimSpace(query.Get("tenant_id")); tenant != "" && tenant != actor.TenantID {
		httpx.WriteError(w, http.StatusForbidden, "tenant_not_allowed", "This request is outside your signed-in bank scope.")
		return false
	}
	query.Set("tenant_id", actor.TenantID)
	r.URL.RawQuery = query.Encode()
	return true
}

func bindJSONIdentity(bindLegalEntity bool, actorFields ...string) routeBinder {
	return func(w http.ResponseWriter, r *http.Request, actor identity.Actor) bool {
		payload, _, err := commandPayload(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The request body must be valid JSON.")
			return false
		}
		if !bindPayloadIdentity(w, payload, actor, bindLegalEntity) {
			return false
		}
		for _, field := range actorFields {
			payload[field] = actor.PrincipalID
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The request body could not be processed.")
			return false
		}
		restoreJSONBody(r, raw)
		return true
	}
}

func bindActorQuery(fields ...string) routeBinder {
	return func(w http.ResponseWriter, r *http.Request, actor identity.Actor) bool {
		query := r.URL.Query()
		for _, field := range fields {
			if existing := strings.TrimSpace(query.Get(field)); existing != "" && existing != actor.PrincipalID {
				httpx.WriteError(w, http.StatusForbidden, "principal_not_allowed", "This request is outside your signed-in user scope.")
				return false
			}
			query.Set(field, actor.PrincipalID)
		}
		r.URL.RawQuery = query.Encode()
		return true
	}
}

func (a *API) bindArtifactIdentity() routeBinder {
	return func(w http.ResponseWriter, r *http.Request, actor identity.Actor) bool {
		maximum := a.deps.MaxArtifactBytes
		if maximum <= 0 {
			maximum = 20 << 20
		}
		r.Body = http.MaxBytesReader(w, r.Body, maximum+(1<<20))
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "artifact_invalid", "The upload could not be read or exceeds the allowed size.")
			return false
		}
		if tenant := strings.TrimSpace(r.FormValue("tenant_id")); tenant != "" && tenant != actor.TenantID {
			httpx.WriteError(w, http.StatusForbidden, "tenant_not_allowed", "This upload is outside your signed-in bank scope.")
			return false
		}
		if r.MultipartForm.Value == nil {
			r.MultipartForm.Value = map[string][]string{}
		}
		r.MultipartForm.Value["tenant_id"] = []string{actor.TenantID}
		r.MultipartForm.Value["created_by"] = []string{actor.PrincipalID}
		return true
	}
}

func (a *API) governancePolicyAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("action", action)
		a.transitionGovernancePolicy(w, r)
	}
}

func (a *API) governanceDelegationAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("action", action)
		a.transitionGovernanceDelegation(w, r)
	}
}
