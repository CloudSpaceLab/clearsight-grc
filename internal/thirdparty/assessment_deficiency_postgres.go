//go:build postgres

package thirdparty

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) LinkAssessmentDeficiency(ctx context.Context, record LinkAssessmentDeficiencyRecord) (AssessmentMatterLink, Assessment, error) {
	if !validAssessmentIdentifiers(record.AssessmentID, record.ActorPrincipalID, record.MatterID, record.MatterTriggerKey) || record.ExpectedVersion < 1 || record.LinkedAt.IsZero() {
		return AssessmentMatterLink{}, Assessment{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentMatterLink{}, Assessment{}, fmt.Errorf("begin assessment deficiency link: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return AssessmentMatterLink{}, Assessment{}, err
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return AssessmentMatterLink{}, Assessment{}, err
	}
	var existing AssessmentMatterLink
	err = tx.QueryRow(ctx, `SELECT t.slug,l.legal_entity_id::text,l.assessment_id::text,l.matter_id::text,l.link_kind,l.created_at
		FROM third_party_assessment_matter_links l JOIN tenants t ON t.id=l.tenant_id
		WHERE l.tenant_id=$1::uuid AND l.legal_entity_id::text=$2 AND l.assessment_id=$3::uuid AND l.matter_id=$4::uuid
		FOR UPDATE OF l`, tenantID, record.LegalEntityID, record.AssessmentID, record.MatterID).Scan(&existing.TenantID, &existing.LegalEntityID, &existing.AssessmentID, &existing.MatterID, &existing.Kind, &existing.CreatedAt)
	if err == nil {
		if existing.Kind != AssessmentMatterDeficiency {
			return AssessmentMatterLink{}, Assessment{}, ErrInvalid
		}
		if current.Version == record.ExpectedVersion+1 {
			var replay bool
			replayErr := tx.QueryRow(ctx, `SELECT true FROM third_party_events WHERE tenant_id=$1::uuid AND aggregate_type='THIRD_PARTY_ASSESSMENT' AND aggregate_id=$2::uuid AND aggregate_version=$3 AND event_type='AssessmentDeficiencyLinked' AND payload->>'deficiency_matter_id'=$4`, tenantID, current.ID, current.Version, record.MatterID).Scan(&replay)
			if replayErr == nil && replay {
				if commitErr := tx.Commit(ctx); commitErr != nil {
					return AssessmentMatterLink{}, Assessment{}, commitErr
				}
				return existing, current, nil
			}
			if replayErr != nil && !errors.Is(replayErr, pgx.ErrNoRows) {
				return AssessmentMatterLink{}, Assessment{}, replayErr
			}
		}
		return AssessmentMatterLink{}, Assessment{}, ErrVersionConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AssessmentMatterLink{}, Assessment{}, fmt.Errorf("load assessment deficiency replay: %w", err)
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentMatterLink{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentUnderReview {
		return AssessmentMatterLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	var proof bool
	err = tx.QueryRow(ctx, `SELECT true FROM matters WHERE tenant_id=$1::uuid AND id=$2::uuid AND matter_type='VENDOR_DEFICIENCY' AND trigger_type='VENDOR_ASSESSMENT_DEFICIENCY' AND trigger_id=$3::uuid AND trigger_key=$4`, tenantID, record.MatterID, current.ID, record.MatterTriggerKey).Scan(&proof)
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentMatterLink{}, Assessment{}, ErrNotFound
	}
	if err != nil {
		return AssessmentMatterLink{}, Assessment{}, fmt.Errorf("verify canonical deficiency Matter: %w", err)
	}
	link := AssessmentMatterLink{Scope: record.Scope, AssessmentID: current.ID, MatterID: record.MatterID, Kind: AssessmentMatterDeficiency, CreatedAt: record.LinkedAt.UTC()}
	if _, err = tx.Exec(ctx, `INSERT INTO third_party_assessment_matter_links(tenant_id,legal_entity_id,assessment_id,matter_id,link_kind,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,'DEFICIENCY',$5)`, tenantID, record.LegalEntityID, current.ID, record.MatterID, link.CreatedAt); err != nil {
		return AssessmentMatterLink{}, Assessment{}, fmt.Errorf("link assessment deficiency: %w", err)
	}
	current.Version++
	current.UpdatedAt = link.CreatedAt
	if err = updateAssessment(ctx, tx, tenantID, current); err != nil {
		return AssessmentMatterLink{}, Assessment{}, err
	}
	if err = appendAssessmentDeficiencyEvent(ctx, tx, tenantID, current, record); err != nil {
		return AssessmentMatterLink{}, Assessment{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return AssessmentMatterLink{}, Assessment{}, fmt.Errorf("commit assessment deficiency link: %w", err)
	}
	return link, current, nil
}

func appendAssessmentDeficiencyEvent(ctx context.Context, tx pgx.Tx, tenantID string, assessment Assessment, record LinkAssessmentDeficiencyRecord) error {
	_, err := tx.Exec(ctx, `INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,$4::uuid,'AssessmentDeficiencyLinked',jsonb_build_object('status',$5::text,'relationship_id',$6::text,'deficiency_matter_id',$7::text,'deficiency_trigger_key',$8::text),$9)`, tenantID, assessment.ID, assessment.Version, record.ActorPrincipalID, assessment.Status, assessment.RelationshipID, record.MatterID, record.MatterTriggerKey, assessment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append assessment deficiency event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,'AssessmentDeficiencyLinked',jsonb_build_object('version',$3::bigint,'status',$4::text,'relationship_id',$5::text,'deficiency_matter_id',$6::text,'deficiency_trigger_key',$7::text),$8,$8)`, tenantID, assessment.ID, assessment.Version, assessment.Status, assessment.RelationshipID, record.MatterID, record.MatterTriggerKey, assessment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append assessment deficiency outbox: %w", err)
	}
	return nil
}
