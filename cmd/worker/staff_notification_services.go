//go:build postgres

package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/attention"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
	"github.com/jackc/pgx/v5/pgxpool"
)

func buildStaffNotificationWorker(cfg config.Config, repository *workflow.PostgresRepository, targets workflow.AssignmentNotificationTargetResolver) (workflowruntime.Publisher, error) {
	delivery, applicationURL, err := buildGovernedEmailDelivery(cfg)
	if err != nil || delivery == nil {
		return nil, err
	}
	return workflow.NewAssignmentNotificationConsumer(repository, delivery, applicationURL, targets)
}

func buildAttentionEmailWorker(cfg config.Config, pool *pgxpool.Pool) (workflowruntime.Publisher, error) {
	delivery, applicationURL, err := buildGovernedEmailDelivery(cfg)
	if err != nil || delivery == nil {
		return nil, err
	}
	return attention.NewCriticalEmailConsumer(attention.NewCriticalEmailPostgresRepository(pool), delivery, applicationURL), nil
}

func buildDailyDigestWorker(cfg config.Config, pool *pgxpool.Pool) (*attention.DigestMaintainer, error) {
	delivery, applicationURL, err := buildGovernedEmailDelivery(cfg)
	if err != nil || delivery == nil {
		return nil, err
	}
	return attention.NewDigestMaintainer(attention.NewDigestPostgresRepository(pool), delivery, applicationURL), nil
}

func buildGovernedEmailDelivery(cfg config.Config) (*evidence.InvitationDeliveryService, string, error) {
	smtpConfig, err := config.LoadSMTPConfig(cfg.Environment)
	if err != nil {
		return nil, "", err
	}
	if !smtpConfig.Enabled {
		return nil, "", nil
	}
	applicationURL := strings.TrimSpace(cfg.AllowedOrigin)
	parsed, parseErr := url.Parse(applicationURL)
	if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		if !strings.EqualFold(cfg.Environment, "production") {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("governed email delivery requires a secure CLEARSIGHT_ALLOWED_ORIGIN")
	}
	adapter, err := evidence.NewSMTPDelivery(evidence.SMTPDeliveryConfig{
		Host: smtpConfig.Host, Port: smtpConfig.Port, Username: smtpConfig.Username,
		SecretRef: smtpConfig.SecretRef, FromAddress: smtpConfig.FromAddress,
		TLSMode: evidence.SMTPTLSMode(smtpConfig.TLSMode), Environment: cfg.Environment,
	}, config.EnvironmentSecretResolver{})
	if err != nil {
		return nil, "", err
	}
	return evidence.NewInvitationDeliveryService(adapter), applicationURL, nil
}
