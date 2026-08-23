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
}

func (a *API) registerRoutes(mux *http.ServeMux) {
	routes := a.routes()
	if err := validateRoutes(routes); err != nil {
		panic(err)
	}
	for _, spec := range routes {
		handler := spec.Handler
		if spec.Command != nil {
			handler = a.command(spec.Command.Name, spec.Command.Policy, handler)
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
		withPermission(write(http.MethodPost, "/api/v1/governance/policies", a.createGovernancePolicy, bindJSONIdentity(false, "maker_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/governance/policies/{id}/submit", a.governancePolicyAction("submit"), bindJSONIdentity(false, "actor_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/governance/policies/{id}/approve", a.governancePolicyAction("approve"), bindJSONIdentity(false, "actor_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/governance/policies/{id}/reject", a.governancePolicyAction("reject"), bindJSONIdentity(false, "actor_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/governance/policies/{id}/retire", a.governancePolicyAction("retire"), bindJSONIdentity(false, "actor_id")), identity.PermissionConfigWrite),
		withPermission(read("/api/v1/governance/delegations", a.listGovernanceDelegations), identity.PermissionConfigRead),
		withPermission(write(http.MethodPost, "/api/v1/governance/delegations", a.createGovernanceDelegation, bindJSONIdentity(false, "maker_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/governance/delegations/{id}/submit", a.governanceDelegationAction("submit"), bindJSONIdentity(false, "actor_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/governance/delegations/{id}/approve", a.governanceDelegationAction("approve"), bindJSONIdentity(false, "actor_id")), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/governance/delegations/{id}/revoke", a.governanceDelegationAction("revoke"), bindJSONIdentity(false, "actor_id")), identity.PermissionConfigWrite),

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
		material("/api/v1/programs", "program.create", a.createProgram, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2, BindLegalEntity: true}),
		read("/api/v1/programs/{id}", a.getProgram),
		read("/api/v1/programs/{id}/history", a.getProgramHistory),
		read("/api/v1/programs/{id}/review-digest", a.getProgramReviewDigest),
		write(http.MethodPost, "/api/v1/programs/{id}/reviews", a.acceptProgramReview, nil),
		material("/api/v1/programs/{id}/transition", "program.transition", a.transitionProgram, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3}),
		material("/api/v1/programs/{id}/requirements", "program.requirement.add", a.addProgramRequirement, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/applicability", "program.applicability.decide", a.determineProgramApplicability, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 3, ActorField: "approved_by"}),
		material("/api/v1/programs/{id}/control-objectives", "program.safeguard.define", a.addProgramControlObjective, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/control-implementations", "program.safeguard.define", a.addProgramControlImplementation, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/control-links", "program.safeguard.define", a.linkProgramRequirementControl, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/evidence-contracts", "program.evidence.define", a.addProgramEvidenceContract, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/programs/{id}/evidence-assessments", "program.evidence.assess", a.recordProgramEvidenceAssessment, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityReviewer, Materiality: 3, ActorField: "assessed_by"}),
		materialService("/api/v1/programs/{id}/triggers", "program.trigger.ingest", a.applyProgramTrigger, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityPerformer, Materiality: 2}),

		read("/api/v1/form-templates", a.listFormTemplates),
		withPermission(write(http.MethodPost, "/api/v1/form-templates", a.createFormTemplate, nil), identity.PermissionConfigWrite),
		withPermission(write(http.MethodPost, "/api/v1/form-templates/{id}/transition", a.transitionFormTemplate, nil), identity.PermissionConfigWrite),
		material("/api/v1/form-templates/{id}/collections", "monitoring.collection.start", a.startFormCollection, commandPolicy{ObjectType: "PROGRAM", ObjectIDField: "program_id", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),
		read("/api/v1/programs/{id}/monitoring-checks", a.listMonitoringChecks),
		material("/api/v1/programs/{id}/monitoring-checks", "monitoring.check.create", a.createMonitoringCheck, commandPolicy{ObjectType: "PROGRAM", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),
		material("/api/v1/monitoring-checks/{id}/transition", "monitoring.check.transition", a.transitionMonitoringCheck, commandPolicy{ObjectType: "PROGRAM", ObjectIDField: "program_id", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),
		material("/api/v1/monitoring-checks/{id}/evaluate-source", "monitoring.source.evaluate", a.evaluateMonitoringSource, commandPolicy{ObjectType: "PROGRAM", ObjectIDField: "program_id", Responsibility: authority.ResponsibilityOwner, Materiality: 2, ActorField: noActorField}),
		read("/api/v1/monitoring-checks/{id}/results", a.listMonitoringResults),

		read("/api/v1/matter-summaries", a.listMatterSummaries),
		read("/api/v1/matters", a.listMatters),
		material("/api/v1/matters", "matter.create", a.createMatter, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		read("/api/v1/matters/{id}", a.getMatter),
		read("/api/v1/matters/{id}/history", a.getMatterHistory),
		material("/api/v1/matters/{id}/transition", "matter.transition", a.transitionMatter, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
		material("/api/v1/matters/{id}/links", "matter.link", a.addMatterLink, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/decisions", "matter.decision.record", a.addMatterDecision, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityAuthorizer, Materiality: 4, ActorField: "authority_principal_id"}),
		material("/api/v1/matters/{id}/actions", "matter.action.add", a.addMatterAction, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/actions/{action_id}/transition", "matter.action.transition", a.transitionMatterAction, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 2}),
		material("/api/v1/matters/{id}/verification-contracts", "matter.outcome.define", a.addMatterVerificationContract, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 3}),
		material("/api/v1/matters/{id}/verification-results", "matter.outcome.record", a.recordMatterVerificationResult, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityReviewer, Materiality: 4, ActorField: "reviewer_principal_id"}),
		material("/api/v1/matters/{id}/responses", "matter.response.add", a.addMatterResponse, commandPolicy{ObjectType: "MATTER", Responsibility: authority.ResponsibilityOwner, Materiality: 3}),
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
		read("/api/v1/evidence/requests", a.listVisibleEvidenceRequests),
		write(http.MethodPost, "/api/v1/evidence/requests", a.createEvidenceRequest, bindJSONIdentity(false, "created_by")),
		read("/api/v1/evidence/requests/{id}", a.getEvidenceRequest),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/wrong-recipient", a.declareEvidenceWrongRecipient, bindJSONIdentity(false, "actor_principal_id")),
		write(http.MethodPut, "/api/v1/evidence/requests/{id}/recipient", a.reassignEvidenceRecipient, bindJSONIdentity(false, "actor_principal_id")),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/submissions", a.submitEvidenceRequest, bindJSONIdentity(false, "submitted_by")),
		write(http.MethodPost, "/api/v1/evidence/requests/{id}/invitations", a.issueEvidenceInvitation, bindJSONIdentity(false, "created_by")),
		capability(http.MethodPost, "/api/v1/evidence/invitations/redeem", a.redeemEvidenceInvitation),
		write(http.MethodPost, "/api/v1/evidence/invitations/{id}/revoke", a.revokeEvidenceInvitation, nil),
		capability(http.MethodGet, "/api/v1/evidence/session", a.getEvidenceSession),
		capability(http.MethodPost, "/api/v1/evidence/session/submissions", a.submitEvidenceSession),
		write(http.MethodPost, "/api/v1/evidence/sessions/{id}/revoke", a.revokeEvidenceSession, nil),
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
		withPermission(write(http.MethodPost, "/api/v1/access/scim-sources/{id}/revoke", a.revokeSCIMSourceToken, nil), identity.PermissionIdentityConfigure),
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
	return routeSpec{Method: http.MethodPost, Path: path, Class: routeMaterialCommand, Handler: handler, Command: &routeCommand{Name: name, Policy: policy}}
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
