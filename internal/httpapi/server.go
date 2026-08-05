package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
)

type Dependencies struct {
	Logger        *slog.Logger
	AllowedOrigin string
	Authority     *authority.Resolver
	Capture       *capture.Service
	Invitations   *capture.InvitationService
	Today         *today.Service
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
	mux.HandleFunc("GET /api/v1/requests/{id}", api.getCaptureRequest)
	mux.HandleFunc("POST /api/v1/requests/{id}/submit", api.submitCaptureRequest)
	mux.HandleFunc("POST /api/v1/invitations/redeem", api.redeemInvitation)
	return httpx.Chain(mux, httpx.CORS(deps.AllowedOrigin), httpx.RequestID, httpx.SecurityHeaders, httpx.Recover(deps.Logger), httpx.AccessLog(deps.Logger))
}
