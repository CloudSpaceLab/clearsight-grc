package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

type Dependencies struct {
	Logger        *slog.Logger
	AllowedOrigin string
	Mode          string
	Authority     authority.Service
	Capture       *capture.Service
	Invitations   *capture.InvitationService
	Today         *today.Service
	Workflow      *workflow.Service
	Onboarding    *onboarding.Service
	Autonomy      *autonomy.Service
}
type API struct{ deps Dependencies }

func New(deps Dependencies) http.Handler {
	api := &API{deps: deps}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", api.live)
	mux.HandleFunc("GET /health/ready", api.ready)
	mux.HandleFunc("GET /api/v1/context", api.context)
	mux.HandleFunc("GET /api/v1/today", api.today)
	mux.HandleFunc("POST /api/v1/authority/resolve", api.resolveAuthority)
	mux.HandleFunc("POST /api/v1/authority/simulate", api.simulateAuthority)
	mux.HandleFunc("GET /api/v1/authority/integrity", api.authorityIntegrity)
	mux.HandleFunc("GET /api/v1/authority/policies", api.authorityPolicies)
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
	return httpx.Chain(mux, httpx.CORS(deps.AllowedOrigin), httpx.RequestID, httpx.SecurityHeaders, httpx.Recover(deps.Logger), httpx.AccessLog(deps.Logger))
}
