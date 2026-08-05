package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

type Dependencies struct {
	Logger           *slog.Logger
	AllowedOrigin    string
	Mode             string
	Identity         identity.Authenticator
	CommandGuard     *commandauth.Guard
	Authority        authority.Service
	Governance       *governance.Service
	Capture          *capture.Service
	Invitations      *capture.InvitationService
	Evidence         *evidence.Service
	Continuity       *continuity.Service
	Today            *today.Service
	Workflow         *workflow.Service
	Onboarding       *onboarding.Service
	Autonomy         *autonomy.Service
	BankVerticals    *bankverticals.Service
	MaxArtifactBytes int64
}
type API struct{ deps Dependencies }

func New(deps Dependencies) http.Handler {
	api := &API{deps: deps}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", api.live)
	mux.HandleFunc("GET /health/ready", api.ready)
	mux.HandleFunc("GET /api/v1/context", api.context)
	mux.HandleFunc("GET /api/v1/today", api.today)
	mux.HandleFunc("GET /api/v1/bank-journeys", api.listBankJourneys)
	mux.HandleFunc("POST /api/v1/authority/resolve", api.resolveAuthority)
	mux.HandleFunc("POST /api/v1/authority/simulate", api.simulateAuthority)
	mux.HandleFunc("GET /api/v1/authority/integrity", api.authorityIntegrity)
	mux.HandleFunc("GET /api/v1/authority/policies", api.authorityPolicies)
	mux.HandleFunc("GET /api/v1/governance/policies", api.listGovernancePolicies)
	mux.HandleFunc("POST /api/v1/governance/policies", api.createGovernancePolicy)
	mux.HandleFunc("POST /api/v1/governance/policies/{id}/{action}", api.transitionGovernancePolicy)
	mux.HandleFunc("GET /api/v1/governance/delegations", api.listGovernanceDelegations)
	mux.HandleFunc("POST /api/v1/governance/delegations", api.createGovernanceDelegation)
	mux.HandleFunc("POST /api/v1/governance/delegations/{id}/{action}", api.transitionGovernanceDelegation)

	mux.HandleFunc("GET /api/v1/program-summaries", api.listProgramSummaries)
	mux.HandleFunc("GET /api/v1/programs", api.listPrograms)
	mux.HandleFunc("POST /api/v1/programs", api.command("program.create", api.createProgram))
	mux.HandleFunc("GET /api/v1/programs/{id}", api.getProgram)
	mux.HandleFunc("GET /api/v1/programs/{id}/history", api.getProgramHistory)
	mux.HandleFunc("POST /api/v1/programs/{id}/transition", api.command("program.transition", api.transitionProgram))
	mux.HandleFunc("POST /api/v1/programs/{id}/requirements", api.command("program.requirement.add", api.addProgramRequirement))
	mux.HandleFunc("POST /api/v1/programs/{id}/applicability", api.command("program.applicability.decide", api.determineProgramApplicability))
	mux.HandleFunc("POST /api/v1/programs/{id}/control-objectives", api.command("program.safeguard.define", api.addProgramControlObjective))
	mux.HandleFunc("POST /api/v1/programs/{id}/control-implementations", api.command("program.safeguard.define", api.addProgramControlImplementation))
	mux.HandleFunc("POST /api/v1/programs/{id}/control-links", api.command("program.safeguard.define", api.linkProgramRequirementControl))
	mux.HandleFunc("POST /api/v1/programs/{id}/evidence-contracts", api.command("program.evidence.define", api.addProgramEvidenceContract))
	mux.HandleFunc("POST /api/v1/programs/{id}/evidence-assessments", api.command("program.evidence.assess", api.recordProgramEvidenceAssessment))
	mux.HandleFunc("POST /api/v1/programs/{id}/triggers", api.command("program.trigger.ingest", api.applyProgramTrigger))
	mux.HandleFunc("GET /api/v1/matter-summaries", api.listMatterSummaries)
	mux.HandleFunc("GET /api/v1/matters", api.listMatters)
	mux.HandleFunc("POST /api/v1/matters", api.command("matter.create", api.createMatter))
	mux.HandleFunc("GET /api/v1/matters/{id}", api.getMatter)
	mux.HandleFunc("GET /api/v1/matters/{id}/history", api.getMatterHistory)
	mux.HandleFunc("POST /api/v1/matters/{id}/transition", api.command("matter.transition", api.transitionMatter))
	mux.HandleFunc("POST /api/v1/matters/{id}/links", api.command("matter.link", api.addMatterLink))
	mux.HandleFunc("POST /api/v1/matters/{id}/decisions", api.command("matter.decision.record", api.addMatterDecision))
	mux.HandleFunc("POST /api/v1/matters/{id}/actions", api.command("matter.action.add", api.addMatterAction))
	mux.HandleFunc("POST /api/v1/matters/{id}/actions/{action_id}/transition", api.command("matter.action.transition", api.transitionMatterAction))
	mux.HandleFunc("POST /api/v1/matters/{id}/verification-contracts", api.command("matter.outcome.define", api.addMatterVerificationContract))
	mux.HandleFunc("POST /api/v1/matters/{id}/verification-results", api.command("matter.outcome.record", api.recordMatterVerificationResult))
	mux.HandleFunc("POST /api/v1/matters/{id}/responses", api.command("matter.response.add", api.addMatterResponse))
	mux.HandleFunc("POST /api/v1/matters/{id}/responses/{response_id}/transition", api.command("matter.response.transition", api.transitionMatterResponse))

	mux.HandleFunc("GET /api/v1/operations/projections", api.projectionHealth)
	mux.HandleFunc("POST /api/v1/operations/projections/reconcile", api.command("projection.reconcile", api.reconcileProgramState))
	mux.HandleFunc("POST /api/v1/operations/projections/rebuild", api.command("projection.rebuild", api.rebuildProgramState))

	mux.HandleFunc("GET /api/v1/evidence/sources", api.listEvidenceSources)
	mux.HandleFunc("POST /api/v1/evidence/sources", api.createEvidenceSource)
	mux.HandleFunc("POST /api/v1/evidence/sources/{id}/observations", api.recordEvidenceSourceObservation)
	mux.HandleFunc("GET /api/v1/evidence/requests", api.listEvidenceRequests)
	mux.HandleFunc("POST /api/v1/evidence/requests", api.createEvidenceRequest)
	mux.HandleFunc("GET /api/v1/evidence/requests/{id}", api.getEvidenceRequest)
	mux.HandleFunc("POST /api/v1/evidence/requests/{id}/submissions", api.submitEvidenceRequest)
	mux.HandleFunc("POST /api/v1/evidence/requests/{id}/invitations", api.issueEvidenceInvitation)
	mux.HandleFunc("POST /api/v1/evidence/invitations/redeem", api.redeemEvidenceInvitation)
	mux.HandleFunc("POST /api/v1/evidence/invitations/{id}/revoke", api.revokeEvidenceInvitation)
	mux.HandleFunc("GET /api/v1/evidence/session", api.getEvidenceSession)
	mux.HandleFunc("POST /api/v1/evidence/session/submissions", api.submitEvidenceSession)
	mux.HandleFunc("POST /api/v1/evidence/sessions/{id}/revoke", api.revokeEvidenceSession)
	mux.HandleFunc("POST /api/v1/evidence/artifacts", api.uploadEvidenceArtifact)
	mux.HandleFunc("GET /api/v1/requests/{id}", api.getCaptureRequest)
	mux.HandleFunc("POST /api/v1/requests/{id}/submit", api.submitCaptureRequest)
	mux.HandleFunc("POST /api/v1/invitations/redeem", api.redeemInvitation)
	mux.HandleFunc("GET /api/v1/workflow/tasks", api.listWorkflowTasks)
	mux.HandleFunc("POST /api/v1/workflow/tasks", api.createWorkflowTask)
	mux.HandleFunc("POST /api/v1/workflow/tasks/{id}/transition", api.transitionWorkflowTask)
	mux.HandleFunc("GET /api/v1/onboarding/guide", api.onboardingGuide)
	mux.HandleFunc("GET /api/v1/onboarding/state", api.onboardingState)
	mux.HandleFunc("PUT /api/v1/onboarding/state", api.updateOnboardingState)
	mux.HandleFunc("GET /api/v1/compliance/readiness", api.readiness)
	mux.HandleFunc("POST /api/v1/compliance/signals", api.ingestSignal)
	return httpx.Chain(mux, httpx.CORS(deps.AllowedOrigin), httpx.RequestID, httpx.SecurityHeaders, identity.Middleware(deps.Identity, deps.Logger), httpx.Recover(deps.Logger), httpx.AccessLog(deps.Logger))
}
