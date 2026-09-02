//go:build postgres

package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	workflowruntime "github.com/CloudSpaceLab/clearsight-grc/internal/runtime"
	"github.com/CloudSpaceLab/clearsight-grc/internal/workflow"
)

func buildStaffNotificationWorker(cfg config.Config, repository *workflow.PostgresRepository, targets workflow.AssignmentNotificationTargetResolver) (workflowruntime.Publisher, error) {
	smtpConfig, err := config.LoadSMTPConfig(cfg.Environment)
	if err != nil {
		return nil, err
	}
	if !smtpConfig.Enabled {
		return nil, nil
	}
	applicationURL := strings.TrimSpace(cfg.AllowedOrigin)
	parsed, parseErr := url.Parse(applicationURL)
	if parseErr != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		if !strings.EqualFold(cfg.Environment, "production") {
			return nil, nil
		}
		return nil, fmt.Errorf("staff assignment email requires a secure CLEARSIGHT_ALLOWED_ORIGIN")
	}
	adapter, err := evidence.NewSMTPDelivery(evidence.SMTPDeliveryConfig{
		Host: smtpConfig.Host, Port: smtpConfig.Port, Username: smtpConfig.Username,
		SecretRef: smtpConfig.SecretRef, FromAddress: smtpConfig.FromAddress,
		TLSMode: evidence.SMTPTLSMode(smtpConfig.TLSMode), Environment: cfg.Environment,
	}, config.EnvironmentSecretResolver{})
	if err != nil {
		return nil, err
	}
	return workflow.NewAssignmentNotificationConsumer(repository, evidence.NewInvitationDeliveryService(adapter), applicationURL, targets)
}
