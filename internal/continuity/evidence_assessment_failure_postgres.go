//go:build postgres

package continuity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) RecordEvidenceAssessmentWithFailure(ctx context.Context, bundle EvidenceAssessmentFailureBundle) (EvidenceAssessmentFailureResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	defer tx.Rollback(ctx)

	var currentVersion int64
	var programEntity string
	enforce, actorTenant, actorEntity := postgresActorScope(ctx)
	err = tx.QueryRow(ctx, `SELECT p.version,p.legal_entity_id::text FROM programs p JOIN tenants t ON t.id=p.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND p.id=$2::uuid AND p.legal_entity_id IS NOT NULL
		  AND (NOT $3 OR ((t.id::text=$4 OR t.slug=$4) AND ($5='*' OR p.legal_entity_id=(SELECT le.id FROM legal_entities le WHERE le.tenant_id=p.tenant_id AND (le.id::text=$5 OR le.code=$5) AND le.valid_from<=clock_timestamp() AND (le.valid_until IS NULL OR clock_timestamp()<le.valid_until) ORDER BY le.valid_from DESC,le.id LIMIT 1))))
		FOR UPDATE`, bundle.TenantID, bundle.ProgramID, enforce, actorTenant, actorEntity).Scan(&currentVersion, &programEntity)
	if errors.Is(err, pgx.ErrNoRows) {
		return EvidenceAssessmentFailureResult{}, ErrNotFound
	}
	if err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	if currentVersion != bundle.ExpectedVersion || bundle.ProgramEvent.AggregateVersion != bundle.ExpectedVersion+1 {
		return EvidenceAssessmentFailureResult{}, ErrVersionConflict
	}
	if bundle.Matter.LegalEntityID == "" || bundle.Matter.LegalEntityID != programEntity || bundle.Link.ProgramID != bundle.ProgramID {
		return EvidenceAssessmentFailureResult{}, ErrNotFound
	}
	if err = applyProgramProjection(ctx, tx, bundle.ProgramEvent); err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	tag, err := tx.Exec(ctx, `UPDATE programs SET version=$3,updated_at=$4 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid AND version=$5`, bundle.TenantID, bundle.ProgramID, bundle.ProgramEvent.AggregateVersion, bundle.ProgramEvent.OccurredAt, bundle.ExpectedVersion)
	if err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	if tag.RowsAffected() != 1 {
		return EvidenceAssessmentFailureResult{}, ErrVersionConflict
	}
	if err = insertContinuityEvent(ctx, tx, bundle.ProgramEvent); err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	if err = insertOutbox(ctx, tx, bundle.ProgramEvent); err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}

	result := EvidenceAssessmentFailureResult{}
	err = tx.QueryRow(ctx, `SELECT m.id::text,t.slug,m.legal_entity_id::text,m.reference,m.matter_type,m.status,m.priority,m.title,m.summary,m.scope,m.source_type,COALESCE(m.source_id::text,''),m.trigger_type,COALESCE(m.trigger_id::text,''),m.trigger_key,m.known_facts,m.missing_facts,m.contradictions,COALESCE(m.owner_principal_id::text,''),m.required_authority,m.due_at,m.closed_at,m.closure_reason,m.reopen_count,m.created_at,m.updated_at,m.version
		FROM matters m JOIN tenants t ON t.id=m.tenant_id JOIN matter_links ml ON ml.tenant_id=m.tenant_id AND ml.matter_id=m.id
		WHERE (t.id::text=$1 OR t.slug=$1) AND m.trigger_key=$2 AND ml.program_id=$3::uuid AND m.legal_entity_id=$4::uuid AND m.status NOT IN ('CLOSED','CANCELLED')
		ORDER BY m.created_at DESC LIMIT 1`, bundle.TenantID, bundle.Matter.TriggerKey, bundle.ProgramID, programEntity).Scan(
		&result.Matter.ID, &result.Matter.TenantID, &result.Matter.LegalEntityID, &result.Matter.Reference, &result.Matter.Type, &result.Matter.Status,
		&result.Matter.Priority, &result.Matter.Title, &result.Matter.Summary, &result.Matter.Scope, &result.Matter.SourceType, &result.Matter.SourceID,
		&result.Matter.TriggerType, &result.Matter.TriggerID, &result.Matter.TriggerKey, &result.Matter.KnownFacts, &result.Matter.MissingFacts,
		&result.Matter.Contradictions, &result.Matter.OwnerPrincipalID, &result.Matter.RequiredAuthority, &result.Matter.DueAt, &result.Matter.ClosedAt,
		&result.Matter.ClosureReason, &result.Matter.ReopenCount, &result.Matter.CreatedAt, &result.Matter.UpdatedAt, &result.Matter.Version)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return EvidenceAssessmentFailureResult{}, err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		matter := bundle.Matter
		_, err = tx.Exec(ctx, `INSERT INTO matters(id,tenant_id,reference,matter_type,status,priority,title,summary,scope,source_type,source_id,trigger_type,trigger_id,trigger_key,known_facts,missing_facts,contradictions,owner_principal_id,required_authority,due_at,closed_at,closure_reason,reopen_count,created_at,updated_at,version,legal_entity_id)
			VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid,$12,NULLIF($13,'')::uuid,$14,$15,$16,$17,NULLIF($18,'')::uuid,$19,$20,$21,$22,$23,$24,$24,$25,$26::uuid)`,
			matter.ID, matter.TenantID, matter.Reference, matter.Type, matter.Status, matter.Priority, matter.Title, matter.Summary, rawJSON(matter.Scope, `{}`),
			matter.SourceType, matter.SourceID, matter.TriggerType, matter.TriggerID, matter.TriggerKey, rawJSON(matter.KnownFacts, `{}`), rawJSON(matter.MissingFacts, `[]`),
			rawJSON(matter.Contradictions, `[]`), matter.OwnerPrincipalID, matter.RequiredAuthority, matter.DueAt, matter.ClosedAt, matter.ClosureReason,
			matter.ReopenCount, matter.CreatedAt, matter.Version, matter.LegalEntityID)
		if err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		if err = insertContinuityEvent(ctx, tx, bundle.MatterEvent); err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		if err = insertOutbox(ctx, tx, bundle.MatterEvent); err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		if err = applyMatterProjection(ctx, tx, bundle.LinkEvent); err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE matters SET version=$3,updated_at=$4 WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1) AND id=$2::uuid`, matter.TenantID, matter.ID, bundle.LinkEvent.AggregateVersion, bundle.LinkEvent.OccurredAt); err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		if err = insertContinuityEvent(ctx, tx, bundle.LinkEvent); err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		if err = insertOutbox(ctx, tx, bundle.LinkEvent); err != nil {
			return EvidenceAssessmentFailureResult{}, err
		}
		matter.Version = bundle.LinkEvent.AggregateVersion
		matter.UpdatedAt = bundle.LinkEvent.OccurredAt
		result.Matter = matter
		result.MatterCreated = true
	}
	if _, err = queueProgramStateTx(ctx, tx, bundle.TenantID, bundle.ProgramID, bundle.ProgramEvent.AggregateVersion, EventEvidenceAssessmentRecorded, bundle.ProgramEvent.ID, bundle.ProgramEvent.ActorID, bundle.ProgramEvent.OccurredAt); err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return EvidenceAssessmentFailureResult{}, err
	}
	return result, nil
}

var _ EvidenceAssessmentFailureRepository = (*PostgresRepository)(nil)
