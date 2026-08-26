package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/httpapi"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	services, err := buildServices(ctx, cfg, logger)
	if err != nil {
		logger.Error("service initialization failed", "error", err)
		os.Exit(1)
	}
	defer services.Close()
	authenticator, federationService, err := buildIdentity(ctx, cfg, services)
	if err != nil {
		logger.Error("identity initialization failed", "error", err)
		os.Exit(1)
	}
	guard, err := commandauth.New(services.Authority, commandauth.Mode(cfg.CommandAuthorizationMode), logger)
	if err != nil {
		logger.Error("command authorization initialization failed", "error", err)
		os.Exit(1)
	}
	services.ThirdParty.ConfigureIdentityAuthority(guard)
	vendorBrandService := thirdparty.NewVendorBrandService(services.ThirdPartyBrandRepo, services.ObjectStore, guard)
	vendorBrandService.ConfigureDiscoveryEnabled(cfg.VendorBrandDiscoveryEnabled)
	services.ThirdParty.ConfigureVendorBrands(vendorBrandService)
	assessmentService := thirdparty.NewAssessmentService(services.ThirdPartyAssessmentRepo, guard)
	assessmentService.ConfigureCancellationRevoker(services.Evidence)
	assessmentMatterReader := thirdparty.NewCanonicalAssessmentReviewMatterReader(services.ThirdPartyAssessmentRepo, services.Continuity)
	assessmentReviewService := thirdparty.NewAssessmentReviewService(assessmentService, services.ThirdPartyAssessmentRepo, services.Evidence, assessmentMatterReader)
	assessmentReviewService.ConfigureAuthority(services.Authority)
	assessmentService.ConfigureCompletionReadiness(assessmentReviewService)
	assessmentDeficiencyService := thirdparty.NewAssessmentDeficiencyService(assessmentService, services.ThirdPartyAssessmentRepo, services.Continuity)
	assessmentRequestService, err := thirdparty.NewAssessmentRequestService(
		assessmentService, services.ThirdPartyAssessmentRepo, services.Evidence, services.MonitoringRepo,
		evidence.NewInvitationDeliveryService(nil), cfg.CapturePublicBaseURL, cfg.Environment,
	)
	if err != nil {
		logger.Error("vendor due-diligence initialization failed", "error", err)
		os.Exit(1)
	}
	vendorWorkService, err := thirdparty.NewVendorWorkService(
		services.ThirdPartyWorkRepo, services.ThirdPartyRelationshipLinkRepo, services.Evidence, services.MonitoringRepo,
		evidence.NewInvitationDeliveryService(nil), cfg.CapturePublicBaseURL, cfg.Environment,
	)
	if err != nil {
		logger.Error("vendor request initialization failed", "error", err)
		os.Exit(1)
	}
	vendorWorkService.ConfigureRelationshipReader(services.ThirdPartyAssessmentRepo)
	vendorWorkService.ConfigureAuthority(guard)
	vendorWorkService.ConfigureReadAuthority(services.Authority)
	vendorWorkService.ConfigureTargetReader(services.Continuity)
	linkCoordinator := &thirdparty.RelationshipLinkCoordinator{}
	vendorWorkService.ConfigureCoordinator(linkCoordinator)
	services.ThirdPartyRelationshipLinks.ConfigureCoordinator(linkCoordinator)
	services.ThirdPartyRelationshipLinks.ConfigureAuthority(guard)
	services.ThirdPartyRelationshipLinks.ConfigureActiveWorkGuard(services.ThirdPartyWorkRepo)
	services.ThirdPartyRelationshipLinks.ConfigureTargetReader(services.Continuity)
	handler := httpapi.New(httpapi.Dependencies{
		Logger: logger, AllowedOrigin: cfg.AllowedOrigin, Mode: services.Mode, DemoMode: cfg.DemoMode,
		IdentityMode: cfg.IdentityMode, OIDCIssuer: cfg.OIDCIssuer,
		Identity: authenticator, Federation: federationService, SCIM: services.SCIM, AccessAdmin: services.AccessAdmin,
		CommandGuard: guard, Authority: services.Authority, Governance: services.Governance,
		Evidence: services.Evidence, Monitoring: services.Monitoring, ThirdParty: services.ThirdParty, VendorBrands: vendorBrandService, ThirdPartyRelationshipLinks: services.ThirdPartyRelationshipLinks, ThirdPartyWork: vendorWorkService, ThirdPartyAssessments: assessmentService, ThirdPartyAssessmentReviews: assessmentReviewService, ThirdPartyAssessmentRequests: assessmentRequestService, ThirdPartyAssessmentDeficiencies: assessmentDeficiencyService, ThirdPartyAssessmentSetup: services.ThirdPartyAssessmentSetup, SourceCatalog: services.SourceCatalog, DocumentImports: services.DocumentImports, Coverage: services.Coverage,
		Continuity: services.Continuity, Today: services.Today, Workflow: services.Workflow, Onboarding: services.Onboarding,
		Autonomy: services.Autonomy, BankVerticals: services.BankVerticals, BackgroundJobs: services.BackgroundJobs,
		MaxArtifactBytes: cfg.MaxArtifactBytes,
	})
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: cfg.ReadTimeout, WriteTimeout: cfg.WriteTimeout, IdleTimeout: cfg.IdleTimeout, MaxHeaderBytes: 1 << 20}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("api listening", "address", cfg.HTTPAddr, "environment", cfg.Environment, "mode", services.Mode, "demo_mode", cfg.DemoMode, "identity_mode", cfg.IdentityMode, "command_authorization", cfg.CommandAuthorizationMode)
		serverErrors <- server.ListenAndServe()
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-stop:
		logger.Info("shutdown requested", "signal", sig.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		_ = server.Close()
	}
}

func buildIdentity(ctx context.Context, cfg config.Config, services serviceSet) (identity.Authenticator, *federation.Service, error) {
	switch cfg.IdentityMode {
	case "oidc":
		if services.Access == nil || services.SessionStore == nil {
			return nil, nil, fmt.Errorf("OIDC identity mode requires PostgreSQL access and session services")
		}
		service, err := federation.New(ctx, federation.Config{
			Issuer: cfg.OIDCIssuer, ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret,
			RedirectURL: cfg.OIDCRedirectURL, ApplicationURL: cfg.AllowedOrigin,
			SessionLifetime: cfg.OIDCSessionLifetime, IdleTimeout: cfg.OIDCSessionIdleTimeout, SecureCookies: cfg.OIDCSecureCookies,
		}, services.SessionStore, services.Access)
		if err != nil {
			return nil, nil, err
		}
		return service, service, nil
	case "signed":
		authenticator, err := identity.NewSignedAuthenticator(cfg.IdentityHMACSecret, cfg.IdentityMaxSkew)
		return authenticator, nil, err
	case "development":
		if cfg.DemoMode {
			authenticator, err := identity.NewDemoAuthenticator(cfg.DemoTenantID, cfg.DemoPrincipalID, cfg.DemoLegalEntityID)
			return authenticator, nil, err
		}
		return identity.NewDevelopmentAuthenticator(cfg.DemoTenantID, cfg.DemoPrincipalID, cfg.DemoLegalEntityID, cfg.DemoRoleCodes...), nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported identity mode %q", cfg.IdentityMode)
	}
}
