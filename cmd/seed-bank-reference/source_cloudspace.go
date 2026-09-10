//go:build postgres

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/bankverticals"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/config"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
	"github.com/jackc/pgx/v5/pgxpool"
)

func validateCloudspaceManualTarget(target thirdparty.Aggregate) error {
	if !strings.EqualFold(strings.TrimSpace(target.Vendor.LegalName), "Cloudspace Technologies Ltd") || !strings.EqualFold(strings.TrimSpace(target.Relationship.ServiceName), "OEM") {
		return fmt.Errorf("manual Cloudspace sample requires the exact Cloudspace Technologies Ltd / OEM relationship")
	}
	return nil
}

// installCloudspaceSample never installs the generic operating population. The
// operator supplies an exact relationship; all reads and writes retain its scope.
func installCloudspaceSample(ctx context.Context, cfg config.Config, pool *pgxpool.Pool, seed bankverticals.SeedConfig, relationshipID string) (operatingFormSampleReceipt, error) {
	result := operatingFormSampleReceipt{States: map[string]string{}, Scores: map[string]any{}}
	if cfg.Environment == "production" || seed.TenantID != "00000000-0000-4000-8000-000000000001" || seed.LegalEntityID != "00000000-0000-4000-8000-000000000002" {
		return result, fmt.Errorf("manual source samples require the nonproduction Clear Bank demo scope")
	}
	var scopeValid bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants t JOIN legal_entities le ON le.tenant_id=t.id WHERE t.id=$1::uuid AND t.slug='clearsight-demo' AND le.id=$2::uuid)`, seed.TenantID, seed.LegalEntityID).Scan(&scopeValid); err != nil {
		return result, err
	}
	if !scopeValid {
		return result, fmt.Errorf("manual source sample demo scope is missing")
	}
	seed.Now = time.Now().UTC()
	repo := thirdparty.NewPostgresRepository(pool)
	target, err := repo.GetRelationship(ctx, thirdparty.Scope{TenantID: seed.TenantID, LegalEntityID: seed.LegalEntityID}, relationshipID)
	if err != nil {
		return result, err
	}
	if err = validateCloudspaceManualTarget(target); err != nil {
		return result, err
	}
	var formID string
	var version int64
	var matchingForms int
	if err = pool.QueryRow(ctx, `WITH candidates AS MATERIALIZED (
	 SELECT f.id,f.version,EXISTS(SELECT 1 FROM capture_form_distributions d
	  WHERE d.tenant_id=f.tenant_id AND d.legal_entity_id=f.legal_entity_id
	  AND d.subject_type='VENDOR_RELATIONSHIP' AND d.subject_id=$3::uuid
	  AND d.form_template_id=f.id AND d.status IN ('OPEN','LOCKED','SUPERSEDED')) AS assigned
	 FROM monitoring_form_templates f WHERE f.tenant_id=$1::uuid AND f.legal_entity_id=$2::uuid
	 AND f.code='VENDOR-DUE-DILIGENCE' AND f.status='ACTIVE' AND f.is_current
	) SELECT id::text,version,count(*) OVER() FROM candidates
	WHERE assigned OR NOT EXISTS(SELECT 1 FROM candidates WHERE assigned)
	ORDER BY id LIMIT 1`, seed.TenantID, seed.LegalEntityID, relationshipID).Scan(&formID, &version, &matchingForms); err != nil {
		return result, err
	}
	if matchingForms != 1 {
		return result, fmt.Errorf("manual Cloudspace form is ambiguous: %d active vendor due-diligence forms", matchingForms)
	}
	monitoringRepo := monitoring.NewPostgresRepository(pool)
	form, err := monitoringRepo.ReusableFormRevision(ctx, seed.TenantID, seed.LegalEntityID, formID, version)
	if err != nil {
		return result, err
	}
	evidenceRepo := evidence.NewPostgresRepository(pool)
	keyring, err := evidence.NewRecipientKeyring(cfg.RecipientSecurity.ActiveKeyID, cfg.RecipientSecurity.Keyring)
	if err != nil {
		return result, err
	}
	store := evidence.NewPostgresDistributionStore(evidenceRepo, keyring)
	distributions := evidence.NewDistributionService(store)
	access, err := evidence.NewDistributionAccessService(store, keyring, nil, cfg.RecipientSecurity.AccessHMACKey, cfg.CaptureSessionTTL)
	if err != nil {
		return result, err
	}
	for _, spec := range operatingFormSampleSpecs() {
		if spec.key != "cloudspace-oem-risk-register" {
			continue
		}
		state, score, err := seedOperatingFormSample(ctx, pool, distributions, access, evidenceRepo, monitoringRepo, seed, form, target.Relationship.ID, spec, seed.Now)
		if err != nil {
			return result, err
		}
		result.States[spec.key] = state
		if score != nil {
			result.Scores[spec.key] = score
		}
		return result, nil
	}
	return result, fmt.Errorf("Cloudspace sample definition is missing")
}
