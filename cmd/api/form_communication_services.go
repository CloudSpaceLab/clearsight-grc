package main

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
)

func configuredCommunicationDelivery(cfg config.Config) (*evidence.InvitationDeliveryService, error) {
	smtpConfig, err := config.LoadSMTPConfig(cfg.Environment)
	if err != nil {
		return nil, err
	}
	if !smtpConfig.Enabled {
		return nil, nil
	}
	adapter, err := evidence.NewSMTPDelivery(evidence.SMTPDeliveryConfig{
		Host:        smtpConfig.Host,
		Port:        smtpConfig.Port,
		Username:    smtpConfig.Username,
		SecretRef:   smtpConfig.SecretRef,
		FromAddress: smtpConfig.FromAddress,
		TLSMode:     evidence.SMTPTLSMode(smtpConfig.TLSMode),
		Environment: cfg.Environment,
	}, config.EnvironmentSecretResolver{})
	if err != nil {
		return nil, err
	}
	return evidence.NewInvitationDeliveryService(adapter), nil
}
