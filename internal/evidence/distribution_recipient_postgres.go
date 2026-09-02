//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func preparePostgresRecipientAdditions(ctx context.Context, store *PostgresDistributionStore, tx pgx.Tx, distribution FormDistribution, inputs []DistributionRecipientInput, now time.Time) ([]postgresPreparedRecipient, DistributionFormRevision, error) {
	if len(inputs) == 0 {
		return nil, DistributionFormRevision{}, nil
	}
	for _, input := range inputs {
		if err := validateDistributionRecipientInput(input); err != nil {
			return nil, DistributionFormRevision{}, err
		}
	}
	var total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM capture_distribution_recipients WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid`, distribution.TenantID, distribution.ID).Scan(&total); err != nil {
		return nil, DistributionFormRevision{}, err
	}
	if total+len(inputs) > 500 {
		return nil, DistributionFormRevision{}, fmt.Errorf("%w: distribution may contain at most 500 recipients", ErrDistributionInvalid)
	}
	prepared, err := store.prepareRecipients(ctx, CreateDistributionInput{Recipients: inputs}, distribution.ID, distribution.TenantID, distribution.LegalEntityID, now)
	if err != nil {
		return nil, DistributionFormRevision{}, err
	}
	form, err := loadPinnedDistributionForm(ctx, tx, distribution)
	if err != nil {
		return nil, DistributionFormRevision{}, err
	}
	return prepared, form, nil
}

func loadPinnedDistributionForm(ctx context.Context, tx pgx.Tx, distribution FormDistribution) (DistributionFormRevision, error) {
	var form DistributionFormRevision
	var presentationJSON, scoreProfileJSON, sectionsJSON, fieldsJSON []byte
	if err := tx.QueryRow(ctx, `
		SELECT id::text,tenant_id::text,legal_entity_id::text,version,sensitivity,presentation,scoring_mode,score_profile,sections,fields
		FROM monitoring_form_templates
		WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND version=$4
		FOR KEY SHARE`, distribution.FormTemplateID, distribution.TenantID, distribution.LegalEntityID, distribution.FormTemplateVersion).Scan(
		&form.ID, &form.TenantID, &form.LegalEntityID, &form.Version, &form.Sensitivity,
		&presentationJSON, &form.ScoringMode, &scoreProfileJSON, &sectionsJSON, &fieldsJSON,
	); err != nil {
		return DistributionFormRevision{}, fmt.Errorf("load pinned distribution form: %w", err)
	}
	if err := json.Unmarshal(presentationJSON, &form.Presentation); err != nil {
		return DistributionFormRevision{}, err
	}
	if len(scoreProfileJSON) > 0 {
		if err := json.Unmarshal(scoreProfileJSON, &form.ScoreProfile); err != nil {
			return DistributionFormRevision{}, err
		}
	}
	if err := json.Unmarshal(sectionsJSON, &form.Sections); err != nil {
		return DistributionFormRevision{}, err
	}
	if err := json.Unmarshal(fieldsJSON, &form.Fields); err != nil {
		return DistributionFormRevision{}, err
	}
	return form, nil
}

func validatePostgresRecipientRevocations(ctx context.Context, tx pgx.Tx, distribution FormDistribution, revokeIDs []string, additions []postgresPreparedRecipient) (int, error) {
	var exact, remainingTO int
	if len(revokeIDs) > 0 {
		if err := tx.QueryRow(ctx, `
			SELECT count(*)
			FROM capture_distribution_recipients
			WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND id::text=ANY($3::text[])`,
			distribution.TenantID, distribution.ID, revokeIDs).Scan(&exact); err != nil {
			return 0, err
		}
		if exact != len(revokeIDs) {
			return 0, fmt.Errorf("%w: recipient to revoke was not found", ErrDistributionInvalid)
		}
	}
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM capture_distribution_recipients
		WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND role='TO' AND state<>'REVOKED'
		  AND (cardinality($3::text[])=0 OR NOT (id::text=ANY($3::text[])))`,
		distribution.TenantID, distribution.ID, revokeIDs).Scan(&remainingTO); err != nil {
		return 0, err
	}
	for _, recipient := range additions {
		if recipient.safe.Role == RecipientTo {
			remainingTO++
		}
	}
	if remainingTO == 0 {
		return 0, fmt.Errorf("%w: at least one active TO recipient is required", ErrDistributionInvalid)
	}
	if len(revokeIDs) == 0 {
		return 0, nil
	}
	var changed int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM capture_distribution_recipients
		WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND id::text=ANY($3::text[]) AND state<>'REVOKED'`,
		distribution.TenantID, distribution.ID, revokeIDs).Scan(&changed); err != nil {
		return 0, err
	}
	return changed, nil
}

func applyPostgresRecipientAmendment(ctx context.Context, tx pgx.Tx, distribution FormDistribution, prepared []postgresPreparedRecipient, form DistributionFormRevision, revokeIDs []string, now time.Time) error {
	if len(revokeIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE capture_distribution_recipients
			SET state='REVOKED',version=version+1,updated_at=$4
			WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid AND id::text=ANY($3::text[]) AND state<>'REVOKED'`,
			distribution.TenantID, distribution.ID, revokeIDs, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE capture_requests q
			SET status='CANCELLED',version=version+1,updated_at=$4
			FROM capture_distribution_recipients r
			WHERE r.tenant_id=$1::uuid AND r.distribution_id=$2::uuid AND r.id::text=ANY($3::text[])
			  AND r.request_id=q.id AND q.tenant_id=r.tenant_id AND q.status NOT IN ('CANCELLED','EXPIRED')`,
			distribution.TenantID, distribution.ID, revokeIDs, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE capture_access_routes ar
			SET revoked_at=COALESCE(ar.revoked_at,$4)
			FROM capture_distribution_recipients r
			WHERE r.tenant_id=$1::uuid AND r.distribution_id=$2::uuid AND r.id::text=ANY($3::text[])
			  AND ar.tenant_id=r.tenant_id AND ar.distribution_id=r.distribution_id AND ar.recipient_id=r.id`,
			distribution.TenantID, distribution.ID, revokeIDs, now.UTC()); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE capture_distribution_sessions s
			SET revoked_at=COALESCE(s.revoked_at,$4)
			FROM capture_distribution_recipients r
			WHERE r.tenant_id=$1::uuid AND r.distribution_id=$2::uuid AND r.id::text=ANY($3::text[])
			  AND s.tenant_id=r.tenant_id AND s.distribution_id=r.distribution_id AND s.recipient_id=r.id`,
			distribution.TenantID, distribution.ID, revokeIDs, now.UTC()); err != nil {
			return err
		}
	}

	estimatedMinutes := 0
	if len(prepared) > 0 {
		if err := tx.QueryRow(ctx, `
			SELECT estimated_minutes FROM capture_requests
			WHERE tenant_id=$1::uuid AND distribution_id=$2::uuid
			ORDER BY created_at,id LIMIT 1`, distribution.TenantID, distribution.ID).Scan(&estimatedMinutes); err != nil {
			return fmt.Errorf("load pinned distribution effort: %w", err)
		}
	}
	for index := range prepared {
		recipient := &prepared[index]
		if recipient.safe.Role == RecipientTo {
			requestInput := CreateDistributionInput{EstimatedMinutes: estimatedMinutes}
			if err := insertDistributionRequest(ctx, tx, distribution, recipient, form, requestInput, now); err != nil {
				return err
			}
		}
		if err := insertDistributionRecipient(ctx, tx, distribution, recipient); err != nil {
			return err
		}
	}
	return nil
}
