package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

type Dependencies struct {
	Logger           *slog.Logger
	AllowedOrigin    string
	Mode             string
	DemoMode         bool
	Identity         identity.Authenticator
	Federation       *federation.Service
	CommandGuard     *commandauth.Guard
	Authority        authority.Service
	Governance       *governance.Service
	Evidence         *evidence.Service
	DocumentImports  *documentimport.Service
	Continuity       *continuity.Service
	Today            *today.Service
	Workflow         *workflow.Service
	Onboarding       *onboarding.Service
	Autonomy         *autonomy.Service
	BankVerticals    *bankverticals.Service
	BackgroundJobs   *operations.Service
	MaxArtifactBytes int64
}

type API struct{ deps Dependencies }

func New(deps Dependencies) http.Handler {
	api := &API{deps: deps}
	mux := http.NewServeMux()
	api.registerFederationRoutes(mux)
	api.registerRoutes(mux)
	handler := httpx.Chain(
		mux,
		httpx.CORS(deps.AllowedOrigin),
		httpx.RequestID,
		httpx.SecurityHeaders,
		identity.Middleware(deps.Identity, deps.Logger),
		httpx.Recover(deps.Logger),
		httpx.AccessLog(deps.Logger),
	)
	if deps.Federation != nil {
		handler = deps.Federation.Middleware(handler)
	}
	protection := http.NewCrossOriginProtection()
	if deps.AllowedOrigin != "" {
		if err := protection.AddTrustedOrigin(deps.AllowedOrigin); err != nil {
			panic(fmt.Errorf("configure trusted origin: %w", err))
		}
	}
	return protection.Handler(handler)
}
