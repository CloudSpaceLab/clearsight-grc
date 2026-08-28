package httpapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/aigovernance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentcoverage"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/governance"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/onboarding"
	"github.com/CloudSpaceLab/clearsight-grc/internal/operations"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

type Dependencies struct {
	Logger                           *slog.Logger
	AllowedOrigin                    string
	Mode                             string
	DemoMode                         bool
	IdentityMode                     string
	OIDCIssuer                       string
	Identity                         identity.Authenticator
	Federation                       *federation.Service
	SCIM                             http.Handler
	Access                           access.Resolver
	AccessAdmin                      access.Administrator
	CommandGuard                     *commandauth.Guard
	Authority                        authority.Service
	Governance                       *governance.Service
	Evidence                         *evidence.Service
	FormDistributions                *evidence.DistributionService
	FormDistributionAccess           *evidence.DistributionAccessService
	FormCommunications               *evidence.CommunicationService
	FormCommunicationBrands          *evidence.CommunicationBrandService
	FormCommunicationTestDelivery    *evidence.InvitationDeliveryService
	Monitoring                       *monitoring.Service
	ThirdParty                       *thirdparty.Service
	VendorBrands                     *thirdparty.VendorBrandService
	ThirdPartyRelationshipLinks      *thirdparty.RelationshipLinkService
	ThirdPartyWork                   *thirdparty.VendorWorkService
	ThirdPartyAssessments            *thirdparty.AssessmentService
	ThirdPartyAssessmentReviews      *thirdparty.AssessmentReviewService
	ThirdPartyAssessmentRequests     *thirdparty.AssessmentRequestService
	ThirdPartyAssessmentDeficiencies *thirdparty.AssessmentDeficiencyService
	ThirdPartyAssessmentSetup        interface {
		Maintain(context.Context, time.Time, int) (int, error)
	}
	SourceCatalog    *sourceaccess.CatalogService
	DocumentImports  *documentimport.Service
	Coverage         *documentcoverage.Service
	Continuity       *continuity.Service
	Today            *today.Service
	Workflow         *workflow.Service
	Onboarding       *onboarding.Service
	Autonomy         *autonomy.Service
	AIGovernance     *aigovernance.Service
	BankVerticals    *bankverticals.Service
	BackgroundJobs   *operations.Service
	MaxArtifactBytes int64
}

type API struct{ deps Dependencies }

func New(deps Dependencies) http.Handler {
	api := &API{deps: deps}
	mux := http.NewServeMux()
	api.registerFederationRoutes(mux)
	api.registerProductionRoutes(mux)
	appHandler := httpx.Chain(
		mux,
		httpx.CORS(deps.AllowedOrigin),
		httpx.RequestID,
		httpx.SecurityHeaders,
		identity.Middleware(deps.Identity, deps.Logger),
		httpx.Recover(deps.Logger),
		httpx.AccessLog(deps.Logger),
	)
	if deps.Federation != nil {
		appHandler = deps.Federation.Middleware(appHandler)
	}
	protection := http.NewCrossOriginProtection()
	if deps.AllowedOrigin != "" {
		if err := protection.AddTrustedOrigin(deps.AllowedOrigin); err != nil {
			panic(fmt.Errorf("configure trusted origin: %w", err))
		}
	}
	appHandler = protection.Handler(appHandler)

	if deps.SCIM == nil {
		return appHandler
	}

	// SCIM is a machine-to-machine protocol edge authenticated by its own
	// tenant-scoped bearer token. Browser session, CORS and actor middleware do
	// not apply here; all post-provisioning application access still flows
	// through the normal identity/permission/authority stack above.
	scimHandler := httpx.Chain(
		deps.SCIM,
		httpx.RequestID,
		httpx.SecurityHeaders,
		httpx.Recover(deps.Logger),
		httpx.AccessLog(deps.Logger),
	)
	root := http.NewServeMux()
	root.Handle("/scim/v2/", scimHandler)
	root.Handle("/", appHandler)
	return root
}
