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
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/httpapi"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
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
	handler := httpapi.New(httpapi.Dependencies{
		Logger: logger, AllowedOrigin: cfg.AllowedOrigin, Mode: services.Mode, DemoMode: cfg.DemoMode,
		Identity: authenticator, Federation: federationService, CommandGuard: guard, Authority: services.Authority, Governance: services.Governance,
		Evidence: services.Evidence, DocumentImports: services.DocumentImports,
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
			Issuer: cfg.OIDCIssuer, ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret, RedirectURL: cfg.OIDCRedirectURL,
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
		return identity.NewDevelopmentAuthenticator(cfg.DemoTenantID, cfg.DemoPrincipalID, cfg.DemoLegalEntityID, cfg.DemoRoleCodes...), nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported identity mode %q", cfg.IdentityMode)
	}
}
