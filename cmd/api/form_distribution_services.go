package main

import (
	"context"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
)

type formDistributionReader struct{ repo monitoringFormRepository }

func (reader formDistributionReader) GetDistributionFormRevision(ctx context.Context, tenantID, legalEntityID, formID string, version int64) (evidence.DistributionFormRevision, error) {
	form, err := reader.repo.ReusableFormRevision(ctx, tenantID, legalEntityID, formID, version)
	if err != nil {
		return evidence.DistributionFormRevision{}, err
	}
	return evidence.DistributionFormRevision{
		ID: form.ID, TenantID: form.TenantID, LegalEntityID: form.LegalEntityID, Version: form.Version,
		Sensitivity: form.Sensitivity, Presentation: form.Presentation,
		Sections: append([]formcontract.Section(nil), form.Sections...),
		Fields:   append([]formcontract.Field(nil), form.Fields...),
		Active:   form.Status == monitoring.LifecycleActive && form.IsCurrent,
	}, nil
}

func configuredRecipientKeyring(cfg config.Config) (evidence.RecipientKeyring, bool, error) {
	if len(cfg.RecipientSecurity.Keyring) == 0 || cfg.RecipientSecurity.ActiveKeyID == "" {
		return evidence.RecipientKeyring{}, false, nil
	}
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	return keyring, err == nil, err
}

func configuredDistributionAccessService(store evidence.DistributionAccessStore, keyring evidence.RecipientKeyring, hasKeyring bool, cfg config.Config) (*evidence.DistributionAccessService, error) {
	if !hasKeyring || cfg.RecipientSecurity.AccessHMACKey == ([32]byte{}) {
		return nil, nil
	}
	return evidence.NewDistributionAccessService(store, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
}
