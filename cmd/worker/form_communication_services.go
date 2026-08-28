//go:build postgres

package main

import (
	"fmt"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func buildFormCommunicationWorker(cfg config.Config, pool *pgxpool.Pool, evidenceRepo *evidence.PostgresRepository) (*evidence.CommunicationDeliveryWorker, *evidence.CommunicationReminderScheduler, error) {
	smtpConfig, err := config.LoadSMTPConfig(cfg.Environment)
	if err != nil {
		return nil, nil, err
	}
	if !smtpConfig.Enabled {
		if cfg.RecipientSecurity.ExternalDeliveryEnabled {
			return nil, nil, fmt.Errorf("external form delivery requires SMTP configuration")
		}
		return nil, nil, nil
	}
	if len(cfg.RecipientSecurity.Keyring) == 0 || cfg.RecipientSecurity.ActiveKeyID == "" || cfg.RecipientSecurity.AccessHMACKey == ([32]byte{}) {
		return nil, nil, fmt.Errorf("SMTP form delivery requires recipient protection and access HMAC configuration")
	}
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		return nil, nil, err
	}
	distributionStore := evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	access, err := evidence.NewDistributionAccessService(distributionStore, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	communications := evidence.NewCommunicationService(evidence.NewPostgresCommunicationStore(evidenceRepo))
	worker, err := evidence.NewCommunicationDeliveryWorker(
		evidence.NewPostgresCommunicationDeliveryRepository(evidenceRepo),
		communications,
		access,
		evidence.NewInvitationDeliveryService(adapter),
		cfg.CapturePublicBaseURL,
	)
	if err != nil {
		return nil, nil, err
	}
	reminders := evidence.NewCommunicationReminderScheduler(evidence.NewPostgresCommunicationReminderRepository(pool))
	return worker, reminders, nil
}
