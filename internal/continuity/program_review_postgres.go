//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/federation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) LatestProgramReview(ctx context.Context, tenant, programID, principalID string) (*ProgramReviewCheckpoint, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT prc.id::text,t.slug,prc.program_id::text,prc.principal_id::text,
		       prc.program_version,prc.projection_version,prc.accepted_at,prc.visible_open_matter_count,prc.visible_projection_version,prc.authority_source_type,prc.authority_source_id::text,prc.authority_rule_id,prc.authority_policy_version
		FROM program_review_checkpoints prc
		JOIN tenants t ON t.id=prc.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND prc.program_id=$2::uuid
		  AND prc.principal_id=$3::uuid
		ORDER BY prc.accepted_at DESC,prc.id DESC
		LIMIT 1`, tenant, programID, principalID)
	value, err := scanProgramReviewCheckpoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *PostgresRepository) RecordProgramReview(ctx context.Context, checkpoint ProgramReviewCheckpoint, event Event) (ProgramReviewCheckpoint, error) {
	actor, ok := identity.FromContext(ctx)
	if !ok || actor.Valid(time.Now()) != nil || actor.Kind != "PERSON" || actor.TenantID != checkpoint.TenantID || actor.PrincipalID != checkpoint.PrincipalID || strings.TrimSpace(actor.LegalEntityID) == "" {
		return ProgramReviewCheckpoint{}, ErrNotFound
	}
	if event.ID == "" || event.TenantID != checkpoint.TenantID || event.AggregateType != "PROGRAM_REVIEW" || event.AggregateID != checkpoint.ID || event.AggregateVersion != 1 || event.Type != EventProgramReviewAccepted || event.ActorType != ActorPerson || event.ActorID != actor.PrincipalID {
		return ProgramReviewCheckpoint{}, ErrNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	defer func() {
		rollbackCtx, stop := context.WithTimeout(context.Background(), time.Second)
		defer stop()
		_ = tx.Rollback(rollbackCtx)
	}()
	session, err := federation.LockVerifiedOIDCSession(ctx, tx, actor)
	if err != nil {
		if errors.Is(err, federation.ErrOIDCSessionProofUnavailable) {
			return ProgramReviewCheckpoint{}, ErrNotFound
		}
		return ProgramReviewCheckpoint{}, err
	}

	var programVersion, projectedProgramVersion, projectionVersion int64
	var tenantID, entityID string
	err = tx.QueryRow(ctx, `
		SELECT p.version,
		       COALESCE(pss.program_version,0),
		       COALESCE(pss.projection_version,0), p.tenant_id::text, p.legal_entity_id::text
		FROM programs p
		JOIN tenants t ON t.id=p.tenant_id
		LEFT JOIN LATERAL (
			SELECT program_version,projection_version
			FROM program_state_snapshots
			WHERE tenant_id=p.tenant_id AND program_id=p.id
			ORDER BY generated_at DESC,projection_version DESC
			LIMIT 1
		) pss ON true
		WHERE t.id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid END FROM tenants WHERE id::text=$1 OR slug=$1) AND p.id=$2::uuid
		FOR UPDATE OF p`, checkpoint.TenantID, checkpoint.ProgramID).Scan(&programVersion, &projectedProgramVersion, &projectionVersion, &tenantID, &entityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgramReviewCheckpoint{}, ErrNotFound
	}
	if err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	if programVersion != checkpoint.ProgramVersion || projectedProgramVersion != checkpoint.ProgramVersion || projectionVersion != checkpoint.ProjectionVersion {
		return ProgramReviewCheckpoint{}, ErrVersionConflict
	}
	route, err := authority.PostgresCurrentRouteRequest(authority.ResolveInput{TenantID: tenantID, LegalEntityID: entityID, ObjectType: "PROGRAM", ObjectID: checkpoint.ProgramID, Responsibility: authority.ResponsibilityReviewer, DecisionType: "program.review.accept", Materiality: 3})
	if err != nil {
		return ProgramReviewCheckpoint{}, ErrNotFound
	}

	row := tx.QueryRow(ctx, authority.PostgresCurrentRouteCTEs()+`
		INSERT INTO program_review_checkpoints(
			id,tenant_id,program_id,principal_id,program_version,projection_version,accepted_at,visible_projection_version,visible_open_matter_count,authority_source_type,authority_source_id,authority_rule_id,authority_policy_version
		) SELECT DISTINCT $2::uuid,p.tenant_id,p.id,$5::uuid,$6::bigint,$7::bigint,statement_timestamp(),$9::bigint,$8::bigint,
		CASE WHEN left(s.source_id,11)='assignment:' THEN 'ASSIGNMENT' ELSE 'POLICY' END,
		(CASE WHEN left(s.source_id,11)='assignment:' THEN substring(s.source_id FROM 12) ELSE s.source_id END)::uuid,s.rule_id,s.policy_version
		FROM programs p JOIN tenants t ON t.id=p.tenant_id
		JOIN selected s ON s.request_index=0 AND NOT s.ambiguous
		JOIN effective e ON e.request_index=s.request_index AND e.principal_id=$5::uuid
		JOIN principals reviewer ON reviewer.id=$5::uuid AND reviewer.tenant_id=p.tenant_id AND reviewer.kind='PERSON' AND reviewer.status='ACTIVE' AND reviewer.valid_from<=statement_timestamp() AND (reviewer.valid_until IS NULL OR reviewer.valid_until>statement_timestamp())
		JOIN legal_entities entity ON entity.id=p.legal_entity_id AND entity.tenant_id=p.tenant_id AND entity.valid_from<=statement_timestamp() AND (entity.valid_until IS NULL OR entity.valid_until>statement_timestamp())
		WHERE current_setting('transaction_isolation')='read committed' AND t.id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid END FROM tenants WHERE id::text=$3 OR slug=$3) AND p.id=$4::uuid
		AND ($10='*' OR entity.id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid END FROM legal_entities WHERE tenant_id=p.tenant_id AND (id::text=$10 OR code=$10) AND valid_from<=statement_timestamp() AND (valid_until IS NULL OR valid_until>statement_timestamp())))
		AND $11::timestamptz>statement_timestamp() AND $11::timestamptz>clock_timestamp() AND $12::timestamptz<=statement_timestamp()+interval '1 minute'
		AND EXISTS(SELECT 1 FROM web_sessions staff_session WHERE staff_session.token=$13 AND staff_session.data=$14::bytea AND staff_session.expiry>statement_timestamp() AND staff_session.expiry>clock_timestamp())
		AND `+visibleMatterScopeSQL("p.scope", "$5::text")+`
		AND p.version=$6 AND EXISTS(SELECT 1 FROM LATERAL (SELECT program_version,projection_version FROM program_state_snapshots WHERE tenant_id=p.tenant_id AND program_id=p.id ORDER BY generated_at DESC,projection_version DESC LIMIT 1) current_state WHERE current_state.program_version=$6 AND current_state.projection_version=$7)
		AND $8::bigint=`+visibleProgramMatterCountSQL("p", "$5::text")+`
		ON CONFLICT(tenant_id,program_id,principal_id,program_version,projection_version,visible_projection_version) WHERE authority_source_id IS NOT NULL DO NOTHING
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),program_id::text,principal_id::text,program_version,projection_version,accepted_at,visible_open_matter_count,visible_projection_version,authority_source_type,authority_source_id::text,authority_rule_id,authority_policy_version`,
		string(route), checkpoint.ID, checkpoint.TenantID, checkpoint.ProgramID, checkpoint.PrincipalID, checkpoint.ProgramVersion, checkpoint.ProjectionVersion, checkpoint.visibleOpenMatterCount, checkpoint.visibleProjectionVersion, actor.LegalEntityID, actor.ExpiresAt, actor.IssuedAt, session.TokenHash, session.Data)
	inserted, scanErr := scanProgramReviewCheckpoint(row)
	if scanErr == nil {
		// Record the same database write-boundary time in the immutable receipt,
		// continuity event and outbox payload after any lock wait.
		event.OccurredAt = inserted.AcceptedAt
		checkpoint.AcceptedAt = inserted.AcceptedAt
		checkpoint.authoritySourceType, checkpoint.authoritySourceID = inserted.authoritySourceType, inserted.authoritySourceID
		checkpoint.authorityRuleID, checkpoint.authorityPolicyVersion = inserted.authorityRuleID, inserted.authorityPolicyVersion
		event.Payload, err = json.Marshal(programReviewEventValue(checkpoint))
		if err != nil {
			return ProgramReviewCheckpoint{}, err
		}
		if err := insertContinuityEvent(ctx, tx, event); err != nil {
			return ProgramReviewCheckpoint{}, err
		}
		if err := insertOutbox(ctx, tx, event); err != nil {
			return ProgramReviewCheckpoint{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ProgramReviewCheckpoint{}, err
		}
		return inserted, nil
	}
	if !errors.Is(scanErr, pgx.ErrNoRows) {
		if isForeignKeyViolation(scanErr) {
			return ProgramReviewCheckpoint{}, ErrNotFound
		}
		return ProgramReviewCheckpoint{}, scanErr
	}

	row = tx.QueryRow(ctx, authority.PostgresCurrentRouteCTEs()+`
		SELECT prc.id::text,t.slug,prc.program_id::text,prc.principal_id::text,
		       prc.program_version,prc.projection_version,prc.accepted_at,prc.visible_open_matter_count,prc.visible_projection_version,prc.authority_source_type,prc.authority_source_id::text,prc.authority_rule_id,prc.authority_policy_version
		FROM program_review_checkpoints prc
		JOIN tenants t ON t.id=prc.tenant_id
		JOIN programs p ON p.id=prc.program_id AND p.tenant_id=prc.tenant_id
		JOIN selected s ON s.request_index=0 AND NOT s.ambiguous
		JOIN effective e ON e.request_index=s.request_index AND e.principal_id=$4::uuid
		JOIN principals reviewer ON reviewer.id=$4::uuid AND reviewer.tenant_id=p.tenant_id AND reviewer.kind='PERSON' AND reviewer.status='ACTIVE' AND reviewer.valid_from<=statement_timestamp() AND (reviewer.valid_until IS NULL OR reviewer.valid_until>statement_timestamp())
		JOIN legal_entities entity ON entity.id=p.legal_entity_id AND entity.tenant_id=p.tenant_id AND entity.valid_from<=statement_timestamp() AND (entity.valid_until IS NULL OR entity.valid_until>statement_timestamp())
		WHERE current_setting('transaction_isolation')='read committed' AND t.id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid END FROM tenants WHERE id::text=$2 OR slug=$2)
		  AND prc.program_id=$3::uuid AND prc.principal_id=$4::uuid
		  AND prc.program_version=$5 AND prc.projection_version=$6 AND prc.visible_projection_version=$8 AND prc.authority_source_id IS NOT NULL
		  AND ($9='*' OR entity.id=(SELECT CASE WHEN count(*)=1 THEN min(id::text)::uuid END FROM legal_entities WHERE tenant_id=p.tenant_id AND (id::text=$9 OR code=$9) AND valid_from<=statement_timestamp() AND (valid_until IS NULL OR valid_until>statement_timestamp())))
		  AND $10::timestamptz>statement_timestamp() AND $10::timestamptz>clock_timestamp() AND $11::timestamptz<=statement_timestamp()+interval '1 minute'
		  AND EXISTS(SELECT 1 FROM web_sessions staff_session WHERE staff_session.token=$12 AND staff_session.data=$13::bytea AND staff_session.expiry>statement_timestamp() AND staff_session.expiry>clock_timestamp())
		  AND `+visibleMatterScopeSQL("p.scope", "$4::text")+`
		  AND p.version=$5 AND EXISTS(SELECT 1 FROM LATERAL (SELECT program_version,projection_version FROM program_state_snapshots WHERE tenant_id=p.tenant_id AND program_id=p.id ORDER BY generated_at DESC,projection_version DESC LIMIT 1) current_state WHERE current_state.program_version=$5 AND current_state.projection_version=$6)
		  AND $7::bigint=`+visibleProgramMatterCountSQL("p", "$4::text"),
		string(route), checkpoint.TenantID, checkpoint.ProgramID, checkpoint.PrincipalID, checkpoint.ProgramVersion, checkpoint.ProjectionVersion, checkpoint.visibleOpenMatterCount, checkpoint.visibleProjectionVersion, actor.LegalEntityID, actor.ExpiresAt, actor.IssuedAt, session.TokenHash, session.Data)
	existing, err := scanProgramReviewCheckpoint(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProgramReviewCheckpoint{}, ErrVersionConflict
	}
	if err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ProgramReviewCheckpoint{}, err
	}
	return existing, nil
}

func (r *PostgresRepository) ProgramStateVersion(ctx context.Context, tenant, programID string, projectionVersion int64) (*ProgramStateSnapshot, error) {
	var state ProgramStateSnapshot
	var dimensions, reasons json.RawMessage
	err := r.pool.QueryRow(ctx, `
		SELECT pss.id::text,t.slug,pss.program_id::text,pss.overall_state,pss.dimensions,pss.reasons,
		       pss.open_matter_count,pss.trigger_type,pss.trigger_id,pss.generated_at,pss.program_version,pss.projection_version
		FROM program_state_snapshots pss
		JOIN tenants t ON t.id=pss.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND pss.program_id=$2::uuid
		  AND pss.projection_version=$3`, tenant, programID, projectionVersion).Scan(
		&state.ID, &state.TenantID, &state.ProgramID, &state.Overall, &dimensions, &reasons,
		&state.OpenMatterCount, &state.TriggerType, &state.TriggerID, &state.GeneratedAt, &state.ProgramVersion, &state.ProjectionVersion,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dimensions, &state.Dimensions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(reasons, &state.Reasons); err != nil {
		return nil, err
	}
	return &state, nil
}

type programReviewScanner interface {
	Scan(...any) error
}

func scanProgramReviewCheckpoint(row programReviewScanner) (ProgramReviewCheckpoint, error) {
	var value ProgramReviewCheckpoint
	var count *int
	var version *int64
	var sourceType, sourceID, ruleID, policyVersion *string
	err := row.Scan(&value.ID, &value.TenantID, &value.ProgramID, &value.PrincipalID, &value.ProgramVersion, &value.ProjectionVersion, &value.AcceptedAt, &count, &version, &sourceType, &sourceID, &ruleID, &policyVersion)
	if count != nil && version != nil {
		value.visibleOpenMatterCount = *count
		value.visibleProjectionVersion = *version
	}
	if sourceType != nil && sourceID != nil && ruleID != nil && policyVersion != nil {
		value.authoritySourceType, value.authoritySourceID = *sourceType, *sourceID
		value.authorityRuleID, value.authorityPolicyVersion = *ruleID, *policyVersion
		value.visibleReceiptPresent = count != nil && version != nil
	}
	return value, err
}
