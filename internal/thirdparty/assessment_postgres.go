//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) CreateAssessment(ctx context.Context, record CreateAssessmentRecord) (Assessment, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return Assessment{}, fmt.Errorf("begin assessment start: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return Assessment{}, err
	}
	var relationshipVersion int64
	err = tx.QueryRow(ctx, `
		SELECT version FROM third_party_relationships
		WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND id::text=$3
		FOR UPDATE`, tenantID, record.LegalEntityID, record.RelationshipID).Scan(&relationshipVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, ErrNotFound
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("lock assessment relationship: %w", err)
	}
	existing, err := scanAssessment(tx.QueryRow(ctx, assessmentSelect+`
		WHERE a.tenant_id=$1::uuid AND a.legal_entity_id::text=$2 AND a.stable_episode_key=$3`, tenantID, record.LegalEntityID, record.Assessment.StableEpisodeKey))
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return Assessment{}, fmt.Errorf("commit assessment replay: %w", err)
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, fmt.Errorf("load assessment replay: %w", err)
	}
	if relationshipVersion != record.RelationshipVersion {
		return Assessment{}, ErrVersionConflict
	}
	var templateExists bool
	err = tx.QueryRow(ctx, `
		SELECT true FROM monitoring_form_templates
		WHERE tenant_id=$1::uuid AND id::text=$2 AND version=$3 AND status='ACTIVE' AND is_current
		FOR SHARE`, tenantID, record.Assessment.FormTemplateID, record.Assessment.FormTemplateVersion).Scan(&templateExists)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, ErrNotFound
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("lock assessment form: %w", err)
	}
	assessment := record.Assessment
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_assessments(
			id,tenant_id,legal_entity_id,relationship_id,review_kind,stable_episode_key,status,
			form_template_id,form_template_version,review_due_at,started_by_principal_id,started_at,version,created_at,updated_at
		) VALUES(
			$1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8::uuid,$9,$10,$11::uuid,$12,$13,$14,$15
		)`, assessment.ID, tenantID, record.LegalEntityID, record.RelationshipID, assessment.ReviewKind,
		assessment.StableEpisodeKey, assessment.Status, assessment.FormTemplateID, assessment.FormTemplateVersion,
		assessment.ReviewDueAt, assessment.StartedByPrincipalID, assessment.StartedAt, assessment.Version,
		assessment.CreatedAt, assessment.UpdatedAt)
	if err != nil {
		return Assessment{}, fmt.Errorf("store assessment: %w", err)
	}
	if err := appendAssessmentEvent(ctx, tx, tenantID, assessment, assessment.StartedByPrincipalID, "AssessmentStarted"); err != nil {
		return Assessment{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_assessment_jobs(
			tenant_id,legal_entity_id,assessment_id,job_type,dedupe_key,state,available_at,created_at,updated_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,'SETUP_REVIEW',$4,'READY',$5,$5,$5)`,
		tenantID, record.LegalEntityID, assessment.ID, "assessment-setup:"+assessment.ID, assessment.CreatedAt)
	if err != nil {
		return Assessment{}, fmt.Errorf("store assessment setup job: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Assessment{}, fmt.Errorf("commit assessment start: %w", err)
	}
	assessment.TenantID = record.TenantID
	assessment.LegalEntityID = record.LegalEntityID
	return assessment, nil
}

func (r *PostgresRepository) GetAssessment(ctx context.Context, scope Scope, assessmentID string) (Assessment, error) {
	value, err := scanAssessment(r.pool.QueryRow(ctx, assessmentSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND a.legal_entity_id::text=$2 AND a.id::text=$3`, scope.TenantID, scope.LegalEntityID, assessmentID))
	return value, mapAssessmentReadError(err, "get assessment")
}

func (r *PostgresRepository) GetCurrentAssessment(ctx context.Context, scope Scope, relationshipID string, kind AssessmentReviewKind) (Assessment, error) {
	value, err := scanAssessment(r.pool.QueryRow(ctx, assessmentSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND a.legal_entity_id::text=$2 AND a.relationship_id::text=$3 AND a.review_kind=$4
		ORDER BY a.updated_at DESC,a.id DESC LIMIT 1`, scope.TenantID, scope.LegalEntityID, relationshipID, kind))
	return value, mapAssessmentReadError(err, "get current assessment")
}

func (r *PostgresRepository) ListAssessments(ctx context.Context, filter AssessmentListFilter) (AssessmentPage, error) {
	limit := filter.Limit
	if limit < 1 || limit > 100 {
		limit = 50
	}
	args := []any{filter.TenantID, filter.LegalEntityID, filter.Status, limit + 1}
	cursorClause := ""
	if filter.Cursor != "" {
		cursorTime, cursorID, err := decodeCursor(filter.Cursor)
		if err != nil {
			return AssessmentPage{}, ErrInvalid
		}
		args = append(args, cursorTime, cursorID)
		cursorClause = " AND (a.updated_at,a.id) < ($5,$6::uuid)"
	}
	rows, err := r.pool.Query(ctx, assessmentSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND a.legal_entity_id::text=$2 AND ($3='' OR a.status=$3)`+cursorClause+`
		ORDER BY a.updated_at DESC,a.id DESC LIMIT $4`, args...)
	if err != nil {
		return AssessmentPage{}, fmt.Errorf("list assessments: %w", err)
	}
	defer rows.Close()
	items := make([]Assessment, 0, limit+1)
	for rows.Next() {
		value, err := scanAssessment(rows)
		if err != nil {
			return AssessmentPage{}, err
		}
		items = append(items, value)
	}
	if err := rows.Err(); err != nil {
		return AssessmentPage{}, err
	}
	page := AssessmentPage{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeCursor(last.UpdatedAt, last.ID)
	}
	return page, nil
}

func (r *PostgresRepository) ListAssessmentEvents(ctx context.Context, scope Scope, assessmentID string, throughVersion int64) ([]AssessmentEvent, error) {
	if throughVersion < 1 {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `
		SELECT e.id::text,t.slug,e.aggregate_id::text,e.aggregate_version,e.event_type,e.payload,e.occurred_at
		FROM third_party_events e
		JOIN tenants t ON t.id=e.tenant_id
		JOIN third_party_assessments a ON a.id=e.aggregate_id AND a.tenant_id=e.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND a.legal_entity_id::text=$2
		  AND e.aggregate_type='THIRD_PARTY_ASSESSMENT' AND e.aggregate_id::text=$3 AND e.aggregate_version<=$4
		ORDER BY e.aggregate_version,e.id`, scope.TenantID, scope.LegalEntityID, assessmentID, throughVersion)
	if err != nil {
		return nil, fmt.Errorf("list assessment events: %w", err)
	}
	defer rows.Close()
	events := []AssessmentEvent{}
	for rows.Next() {
		var value AssessmentEvent
		var payload []byte
		if err := rows.Scan(&value.ID, &value.TenantID, &value.AssessmentID, &value.AssessmentVersion, &value.Type, &payload, &value.OccurredAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &value.Payload); err != nil {
			return nil, err
		}
		events = append(events, value)
	}
	return events, rows.Err()
}

func (r *PostgresRepository) TransitionAssessment(ctx context.Context, record AssessmentTransitionRecord) (Assessment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Assessment{}, fmt.Errorf("begin assessment transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return Assessment{}, err
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.ID)
	if err != nil {
		return Assessment{}, err
	}
	if current.Version != record.ExpectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	if !containsAssessmentStatus(record.From, current.Status) {
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	eventType := ""
	current.Status = record.To
	current.UpdatedAt = record.At.UTC()
	current.Version++
	switch record.To {
	case AssessmentUnderReview:
		if !validAssessmentIdentifier(record.ActorPrincipalID) || current.SubmissionID == "" {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		at := record.At.UTC()
		current.ReviewStartedAt = &at
		current.ReviewerPrincipalID = record.ActorPrincipalID
		eventType = "AssessmentReviewStarted"
	case AssessmentCompleted:
		if !validAssessmentIdentifier(record.ActorPrincipalID) || !validAssessmentConclusion(record.Conclusion) || strings.TrimSpace(record.ConclusionRationale) == "" {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		if err := verifyPostgresAssessmentCompletionReady(ctx, tx, tenantID, current); err != nil {
			return Assessment{}, err
		}
		at := record.At.UTC()
		current.CompletedAt = &at
		current.ReviewerPrincipalID = record.ActorPrincipalID
		current.Conclusion = record.Conclusion
		current.ConclusionUncertainty = record.ConclusionUncertainty
		current.ConclusionRationale = record.ConclusionRationale
		current.NextReviewRecommendedAt = cloneAssessmentTime(record.NextReviewRecommendedAt)
		eventType = "AssessmentCompleted"
	case AssessmentCancelled:
		if strings.TrimSpace(record.CancellationReason) == "" {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		current.CancellationReason = record.CancellationReason
		eventType = "AssessmentCancelled"
	default:
		return Assessment{}, ErrInvalidAssessmentTransition
	}
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return Assessment{}, err
	}
	if err := appendAssessmentEvent(ctx, tx, tenantID, current, record.ActorPrincipalID, eventType); err != nil {
		return Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Assessment{}, fmt.Errorf("commit assessment transition: %w", err)
	}
	return current, nil
}

func (r *PostgresRepository) ApplyAssessmentReaction(ctx context.Context, record AssessmentReactionRecord) (Assessment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Assessment{}, fmt.Errorf("begin assessment reaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return Assessment{}, err
	}
	if replay, found, err := assessmentReactionReplay(ctx, tx, tenantID, record); err != nil {
		return Assessment{}, err
	} else if found {
		return replay, nil
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return Assessment{}, err
	}
	if replay, found, err := assessmentReactionReplay(ctx, tx, tenantID, record); err != nil {
		return Assessment{}, err
	} else if found {
		return replay, nil
	}
	if current.Version != record.ExpectedVersion {
		return Assessment{}, ErrVersionConflict
	}
	eventType := ""
	switch record.Kind {
	case AssessmentReactionSetupCompleted:
		if current.Status != AssessmentSetupPending {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		var jobAssessmentID string
		err = tx.QueryRow(ctx, `
			SELECT assessment_id::text FROM third_party_assessment_jobs
			WHERE id::text=$1 AND tenant_id=$2::uuid AND legal_entity_id::text=$3 AND assessment_id=$4::uuid
			FOR UPDATE`, record.JobID, tenantID, record.LegalEntityID, current.ID).Scan(&jobAssessmentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return Assessment{}, ErrNotFound
		}
		if err != nil {
			return Assessment{}, fmt.Errorf("lock assessment setup job: %w", err)
		}
		canonical, linkErr := ensureAssessmentMatterRelationshipLink(ctx, tx, tenantID, current, record.MatterID, AssessmentMatterReview, current.StartedByPrincipalID, record.At)
		if linkErr != nil {
			return Assessment{}, linkErr
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO third_party_assessment_matter_links(tenant_id,legal_entity_id,assessment_id,matter_id,relationship_link_id,link_kind,created_at)
			VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,'REVIEW',$6)
			ON CONFLICT (tenant_id,assessment_id,matter_id) DO NOTHING`, tenantID, record.LegalEntityID, current.ID, record.MatterID, canonical.ID, record.At)
		if err != nil {
			return Assessment{}, fmt.Errorf("link assessment review matter: %w", err)
		}
		_, err = tx.Exec(ctx, `
			UPDATE third_party_assessment_jobs SET state='COMPLETED',lease_token=NULL,lease_expires_at=NULL,updated_at=$4
			WHERE id::text=$1 AND tenant_id=$2::uuid AND assessment_id=$3::uuid`, record.JobID, tenantID, current.ID, record.At)
		if err != nil {
			return Assessment{}, fmt.Errorf("complete assessment setup job: %w", err)
		}
		current.Status = AssessmentReadyToSend
		current.ReviewMatterID = record.MatterID
		eventType = "AssessmentSetupCompleted"
	case AssessmentReactionSubmitted:
		if current.Status != AssessmentCollecting || current.CurrentRequestID != record.RequestID {
			return Assessment{}, ErrInvalidAssessmentTransition
		}
		var exists bool
		err = tx.QueryRow(ctx, `
			SELECT true FROM capture_submissions
			WHERE id::text=$1 AND tenant_id=$2::uuid AND request_id::text=$3`, record.SubmissionID, tenantID, record.RequestID).Scan(&exists)
		if errors.Is(err, pgx.ErrNoRows) {
			return Assessment{}, ErrNotFound
		}
		if err != nil {
			return Assessment{}, fmt.Errorf("verify assessment submission: %w", err)
		}
		at := record.At.UTC()
		current.Status = AssessmentSubmitted
		current.SubmissionID = record.SubmissionID
		current.SubmittedAt = &at
		eventType = "AssessmentSubmitted"
	default:
		return Assessment{}, ErrInvalid
	}
	current.Version++
	current.UpdatedAt = record.At.UTC()
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return Assessment{}, err
	}
	if err := appendAssessmentEvent(ctx, tx, tenantID, current, "", eventType); err != nil {
		return Assessment{}, err
	}
	snapshot, err := json.Marshal(current)
	if err != nil {
		return Assessment{}, err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_assessment_reactions(
			tenant_id,legal_entity_id,assessment_id,reaction_kind,causation_id,job_id,event_id,matter_id,request_id,submission_id,
			resulting_version,result_snapshot,applied_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,NULLIF($6,'')::uuid,NULLIF($7,'')::uuid,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11,$12::jsonb,$13)`,
		tenantID, record.LegalEntityID, current.ID, record.Kind, record.CausationID, record.JobID, record.EventID,
		record.MatterID, record.RequestID, record.SubmissionID, current.Version, string(snapshot), record.At)
	if err != nil {
		return Assessment{}, fmt.Errorf("store assessment reaction receipt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Assessment{}, fmt.Errorf("commit assessment reaction: %w", err)
	}
	return current, nil
}

func (r *PostgresRepository) PrepareAssessmentRequest(ctx context.Context, record PrepareAssessmentRequestRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifier(record.ActorPrincipalID) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("begin assessment request preparation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	existing, err := scanAssessmentRequestLink(tx.QueryRow(ctx, assessmentRequestLinkSelect+`
		WHERE l.tenant_id=$1::uuid AND l.assessment_id=$2::uuid AND l.origin_type=$3 AND l.origin_id::text=$4 AND l.origin_sequence=$5`,
		tenantID, record.AssessmentID, record.OriginType, record.OriginID, record.OriginSequence))
	if err == nil {
		if existing.RequestID != record.RequestID || existing.Purpose != record.Purpose {
			return AssessmentRequestLink{}, Assessment{}, ErrInvalid
		}
		if current.Version != record.ExpectedVersion {
			return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
		}
		if (existing.Sequence == 1 && current.Status != AssessmentReadyToSend) || (existing.Sequence > 1 && current.Status != AssessmentUnderReview) {
			return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
		}
		if err := tx.Commit(ctx); err != nil {
			return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("commit assessment request preparation replay: %w", err)
		}
		return existing, current, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("load assessment request preparation replay: %w", err)
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	var sequence int
	if err := tx.QueryRow(ctx, `SELECT count(*)+1 FROM third_party_assessment_request_links WHERE tenant_id=$1::uuid AND assessment_id=$2::uuid`, tenantID, current.ID).Scan(&sequence); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if record.OriginType != AssessmentRequestOrigin || record.OriginID != current.ID || record.OriginSequence != sequence || !validAssessmentRequestPurpose(record.Purpose) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	if (sequence == 1 && (record.Purpose != AssessmentRequestInitial || current.Status != AssessmentReadyToSend || current.CurrentRequestID != "")) ||
		(sequence > 1 && (record.Purpose != AssessmentRequestClarification || current.Status != AssessmentUnderReview || current.CurrentRequestID == "")) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	var proof bool
	err = tx.QueryRow(ctx, `
		SELECT true FROM capture_requests r
		WHERE r.tenant_id=$1::uuid AND r.id::text=$2 AND r.origin_type=$3 AND r.origin_id=$4 AND r.origin_version=$5`,
		tenantID, record.RequestID, record.OriginType, record.OriginID, record.OriginSequence).Scan(&proof)
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentRequestLink{}, Assessment{}, ErrNotFound
	}
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("verify prepared assessment request: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE third_party_assessment_request_links SET is_current=false WHERE tenant_id=$1::uuid AND assessment_id=$2::uuid AND is_current`, tenantID, current.ID); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	link := AssessmentRequestLink{
		TenantID: current.TenantID, LegalEntityID: current.LegalEntityID, AssessmentID: current.ID,
		RequestID: record.RequestID, Purpose: record.Purpose, Sequence: sequence, OriginType: record.OriginType,
		OriginID: record.OriginID, OriginSequence: record.OriginSequence, CreatedAt: record.PreparedAt.UTC(),
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO third_party_assessment_request_links(
			tenant_id,legal_entity_id,assessment_id,request_id,purpose,sequence,origin_type,origin_id,origin_sequence,invitation_id,is_current,created_at
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8::uuid,$9,NULL,true,$10)`, tenantID, record.LegalEntityID,
		current.ID, record.RequestID, record.Purpose, sequence, record.OriginType, record.OriginID, record.OriginSequence, record.PreparedAt)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("prepare assessment request link: %w", err)
	}
	current.CurrentRequestID = record.RequestID
	current.Version++
	current.UpdatedAt = record.PreparedAt.UTC()
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if err := appendAssessmentEvent(ctx, tx, tenantID, current, record.ActorPrincipalID, "AssessmentRequestPrepared"); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("commit assessment request preparation: %w", err)
	}
	return link, current, nil
}

func (r *PostgresRepository) RecordRequestIssued(ctx context.Context, record RecordRequestIssuedRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifier(record.ActorPrincipalID) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("begin assessment request issue: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	existing, err := scanAssessmentRequestLink(tx.QueryRow(ctx, assessmentRequestLinkSelect+`
		WHERE l.tenant_id=$1::uuid AND l.assessment_id=$2::uuid AND l.origin_type=$3 AND l.origin_id::text=$4 AND l.origin_sequence=$5`,
		tenantID, record.AssessmentID, record.OriginType, record.OriginID, record.OriginSequence))
	if err == nil {
		if existing.RequestID != record.RequestID || existing.Purpose != record.Purpose || (existing.InvitationID != "" && existing.InvitationID != record.InvitationID) {
			return AssessmentRequestLink{}, Assessment{}, ErrInvalid
		}
		if existing.InvitationID == record.InvitationID {
			return existing, current, nil
		}
		if current.Version != record.ExpectedVersion {
			return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
		}
		if (existing.Sequence == 1 && current.Status != AssessmentReadyToSend) || (existing.Sequence > 1 && current.Status != AssessmentUnderReview) {
			return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
		}
		var proof bool
		err = tx.QueryRow(ctx, `
			SELECT true FROM capture_invitations i
			WHERE i.tenant_id=$1::uuid AND i.request_id=$2::uuid AND i.id::text=$3`, tenantID, record.RequestID, record.InvitationID).Scan(&proof)
		if errors.Is(err, pgx.ErrNoRows) {
			return AssessmentRequestLink{}, Assessment{}, ErrNotFound
		}
		if err != nil {
			return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("verify prepared request invitation: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE third_party_assessment_request_links SET invitation_id=$4::uuid WHERE tenant_id=$1::uuid AND assessment_id=$2::uuid AND sequence=$3`, tenantID, current.ID, existing.Sequence, record.InvitationID); err != nil {
			return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("finalize assessment request link: %w", err)
		}
		existing.InvitationID = record.InvitationID
		current.CurrentRequestID = record.RequestID
		current.SubmissionID = ""
		current.Status = AssessmentCollecting
		current.Version++
		current.UpdatedAt = record.IssuedAt.UTC()
		if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
			return AssessmentRequestLink{}, Assessment{}, err
		}
		if err := appendAssessmentEvent(ctx, tx, tenantID, current, record.ActorPrincipalID, "AssessmentRequestIssued"); err != nil {
			return AssessmentRequestLink{}, Assessment{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("commit assessment request finalization: %w", err)
		}
		return existing, current, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("load assessment request replay: %w", err)
	}
	return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
}

func (r *PostgresRepository) GetCurrentAssessmentRequestLink(ctx context.Context, scope Scope, assessmentID string) (AssessmentRequestLink, error) {
	value, err := scanAssessmentRequestLink(r.pool.QueryRow(ctx, assessmentRequestLinkSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND l.legal_entity_id::text=$2 AND l.assessment_id::text=$3 AND l.is_current`,
		scope.TenantID, scope.LegalEntityID, assessmentID))
	return value, mapAssessmentReadError(err, "get current assessment request")
}

func (r *PostgresRepository) PrepareRequestReissue(ctx context.Context, record PrepareRequestReissueRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifiers(record.ActorPrincipalID, record.RequestID) || (record.ExpectedInvitationID != "" && !validAssessmentIdentifier(record.ExpectedInvitationID)) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("begin assessment request reissue preparation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentCollecting || current.CurrentRequestID != record.RequestID {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	link, err := scanAssessmentRequestLink(tx.QueryRow(ctx, assessmentRequestLinkSelect+`
		WHERE l.tenant_id=$1::uuid AND l.legal_entity_id::text=$2 AND l.assessment_id=$3::uuid AND l.is_current FOR UPDATE OF l`,
		tenantID, record.LegalEntityID, record.AssessmentID))
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, mapAssessmentReadError(err, "lock current assessment request")
	}
	if link.RequestID != record.RequestID || link.InvitationID != record.ExpectedInvitationID {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	if _, err := tx.Exec(ctx, `
		UPDATE third_party_assessment_request_links SET invitation_id=NULL
		WHERE tenant_id=$1::uuid AND assessment_id=$2::uuid AND sequence=$3`,
		tenantID, current.ID, link.Sequence); err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("prepare assessment replacement invitation: %w", err)
	}
	link.InvitationID = ""
	current.Version++
	current.UpdatedAt = record.PreparedAt.UTC()
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if err := appendAssessmentEvent(ctx, tx, tenantID, current, record.ActorPrincipalID, "AssessmentRequestReissuePrepared"); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("commit assessment request reissue preparation: %w", err)
	}
	return link, current, nil
}

func (r *PostgresRepository) FinalizeRequestReissue(ctx context.Context, record FinalizeRequestReissueRecord) (AssessmentRequestLink, Assessment, error) {
	if !validAssessmentIdentifiers(record.ActorPrincipalID, record.RequestID, record.InvitationID) {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("begin assessment request reissue finalization: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentCollecting || current.CurrentRequestID != record.RequestID {
		return AssessmentRequestLink{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	link, err := scanAssessmentRequestLink(tx.QueryRow(ctx, assessmentRequestLinkSelect+`
		WHERE l.tenant_id=$1::uuid AND l.legal_entity_id::text=$2 AND l.assessment_id=$3::uuid AND l.is_current FOR UPDATE OF l`,
		tenantID, record.LegalEntityID, record.AssessmentID))
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, mapAssessmentReadError(err, "lock prepared assessment request reissue")
	}
	if link.RequestID != record.RequestID || link.InvitationID != "" {
		return AssessmentRequestLink{}, Assessment{}, ErrVersionConflict
	}
	var proof bool
	err = tx.QueryRow(ctx, `
		SELECT true FROM capture_invitations i
		WHERE i.tenant_id=$1::uuid AND i.request_id=$2::uuid AND i.id::text=$3 AND i.revoked_at IS NULL`,
		tenantID, record.RequestID, record.InvitationID).Scan(&proof)
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentRequestLink{}, Assessment{}, ErrNotFound
	}
	if err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("verify replacement invitation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE third_party_assessment_request_links SET invitation_id=$4::uuid
		WHERE tenant_id=$1::uuid AND assessment_id=$2::uuid AND sequence=$3`,
		tenantID, current.ID, link.Sequence, record.InvitationID); err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("finalize assessment replacement invitation: %w", err)
	}
	link.InvitationID = record.InvitationID
	current.Version++
	current.UpdatedAt = record.ReissuedAt.UTC()
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if err := appendAssessmentEvent(ctx, tx, tenantID, current, record.ActorPrincipalID, "AssessmentRequestReissued"); err != nil {
		return AssessmentRequestLink{}, Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AssessmentRequestLink{}, Assessment{}, fmt.Errorf("commit assessment request reissue finalization: %w", err)
	}
	return link, current, nil
}

func (r *PostgresRepository) ListAssessmentRequestLinks(ctx context.Context, scope Scope, assessmentID string) ([]AssessmentRequestLink, error) {
	rows, err := r.pool.Query(ctx, assessmentRequestLinkSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND l.legal_entity_id::text=$2 AND l.assessment_id::text=$3
		ORDER BY l.sequence,l.request_id`, scope.TenantID, scope.LegalEntityID, assessmentID)
	if err != nil {
		return nil, fmt.Errorf("list assessment requests: %w", err)
	}
	defer rows.Close()
	links := []AssessmentRequestLink{}
	for rows.Next() {
		value, err := scanAssessmentRequestLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, value)
	}
	return links, rows.Err()
}

func (r *PostgresRepository) ListAssessmentMatterLinks(ctx context.Context, scope Scope, assessmentID string, limit int) ([]AssessmentMatterLink, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if !validAssessmentScope(scope) || !validAssessmentIdentifier(assessmentID) || limit < 1 || limit > assessmentReviewMaxMatters+1 {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `
		SELECT t.slug,l.legal_entity_id::text,l.assessment_id::text,l.matter_id::text,l.relationship_link_id::text,l.link_kind,l.created_at
		FROM third_party_assessment_matter_links l
		JOIN third_party_assessments a
		  ON a.id=l.assessment_id AND a.tenant_id=l.tenant_id AND a.legal_entity_id=l.legal_entity_id
		JOIN third_party_relationship_matter_links relationship_link
		  ON relationship_link.id=l.relationship_link_id
		 AND relationship_link.tenant_id=l.tenant_id
		 AND relationship_link.legal_entity_id=l.legal_entity_id
		 AND relationship_link.relationship_id=a.relationship_id
		 AND relationship_link.matter_id=l.matter_id
		JOIN tenants t ON t.id=l.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND l.legal_entity_id::text=$2 AND l.assessment_id::text=$3
		ORDER BY l.created_at,l.matter_id
		LIMIT $4`, scope.TenantID, scope.LegalEntityID, assessmentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list assessment matters: %w", err)
	}
	defer rows.Close()
	values := make([]AssessmentMatterLink, 0)
	for rows.Next() {
		var value AssessmentMatterLink
		if err := rows.Scan(&value.TenantID, &value.LegalEntityID, &value.AssessmentID, &value.MatterID, &value.RelationshipLinkID, &value.Kind, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assessment matter link: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list assessment matters: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) ReviewAssessmentDocument(ctx context.Context, record AssessmentDocumentReviewRecord) (AssessmentDocument, Assessment, error) {
	if !validAssessmentScope(record.Scope) || !validAssessmentIdentifiers(record.AssessmentID, record.ActorPrincipalID, record.Artifact.ID, record.Artifact.RequestID, record.Artifact.SubmissionID) || record.ExpectedVersion < 1 || !validAssessmentDocumentDecision(record.Decision) || !validAssessmentDocumentEvidenceClass(record.EvidenceClass) {
		return AssessmentDocument{}, Assessment{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return AssessmentDocument{}, Assessment{}, fmt.Errorf("begin assessment document review: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return AssessmentDocument{}, Assessment{}, err
	}
	current, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return AssessmentDocument{}, Assessment{}, err
	}
	if current.Version != record.ExpectedVersion {
		return AssessmentDocument{}, Assessment{}, ErrVersionConflict
	}
	if current.Status != AssessmentUnderReview || current.CurrentRequestID != record.Artifact.RequestID || current.SubmissionID != record.Artifact.SubmissionID || current.RelationshipID == "" {
		return AssessmentDocument{}, Assessment{}, ErrInvalidAssessmentTransition
	}
	var artifactStatus evidence.ArtifactStatus
	var submittedType, reference, issuedBy, issuedOn, expiresOn string
	err = tx.QueryRow(ctx, `
		SELECT ar.status,answer->'document'->>'document_type',COALESCE(answer->'document'->>'reference',''),
			COALESCE(answer->'document'->>'issued_by',''),COALESCE(answer->'document'->>'issued_on',''),COALESCE(answer->'document'->>'expires_on','')
		FROM capture_artifacts ar
		JOIN capture_submissions s ON s.id=ar.submission_id AND s.tenant_id=ar.tenant_id AND s.request_id=ar.request_id
		JOIN capture_requests req ON req.id=ar.request_id AND req.tenant_id=ar.tenant_id
		JOIN LATERAL jsonb_array_elements(req.fields) field ON field->>'type'='vendor_document'
		JOIN LATERAL (SELECT s.answers->(field->>'id') AS answer) submitted ON true
		WHERE ar.tenant_id=$1::uuid AND ar.request_id=$2::uuid AND ar.submission_id=$3::uuid AND ar.id=$4::uuid
		  AND req.subject_type='VENDOR_RELATIONSHIP' AND req.subject_id=$5
		  AND req.origin_type=$6 AND req.origin_id=$7 AND req.form_template_id=$8::uuid AND req.form_template_version=$9
		  AND answer->'document'->>'artifact_id'=$4
		FOR SHARE OF ar,req,s`, tenantID, current.CurrentRequestID, current.SubmissionID, record.Artifact.ID, current.RelationshipID,
		AssessmentRequestOrigin, current.ID, current.FormTemplateID, current.FormTemplateVersion).Scan(&artifactStatus, &submittedType, &reference, &issuedBy, &issuedOn, &expiresOn)
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentDocument{}, Assessment{}, ErrNotFound
	}
	if err != nil {
		return AssessmentDocument{}, Assessment{}, fmt.Errorf("verify assessment document artifact: %w", err)
	}
	if artifactStatus != record.Artifact.Status || submittedType != record.Document.DocumentType || reference != record.Document.Reference || issuedBy != record.Document.IssuedBy || issuedOn != record.Document.IssuedOn || expiresOn != record.Document.ExpiresOn {
		return AssessmentDocument{}, Assessment{}, ErrNotFound
	}
	if record.Decision == AssessmentDocumentValidate && artifactStatus != evidence.ArtifactAvailable {
		return AssessmentDocument{}, Assessment{}, ErrAssessmentCompletionBlocked
	}
	issuedDate, err := assessmentDocumentDate(issuedOn)
	if err != nil || (issuedDate != nil && record.ExpiresOn != nil && record.ExpiresOn.Before(*issuedDate)) {
		return AssessmentDocument{}, Assessment{}, ErrInvalid
	}
	status, eventType := AssessmentDocumentValidated, "AssessmentDocumentValidated"
	if record.Decision == AssessmentDocumentReject {
		status, eventType = AssessmentDocumentRejected, "AssessmentDocumentRejected"
	}
	var document AssessmentDocument
	err = tx.QueryRow(ctx, `
		INSERT INTO third_party_documents(
			tenant_id,legal_entity_id,relationship_id,assessment_id,request_id,artifact_id,document_type,reference,issued_by,issued_on,expires_on,
			evidence_class,status,validated_by_principal_id,validated_at,created_at,updated_at,version
		) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9,$10,$11,$12,$13,$14::uuid,$15,$15,$15,1)
		ON CONFLICT (tenant_id,assessment_id,artifact_id) DO UPDATE SET
			document_type=EXCLUDED.document_type,reference=EXCLUDED.reference,issued_by=EXCLUDED.issued_by,issued_on=EXCLUDED.issued_on,
			expires_on=EXCLUDED.expires_on,evidence_class=EXCLUDED.evidence_class,status=EXCLUDED.status,
			validated_by_principal_id=EXCLUDED.validated_by_principal_id,validated_at=EXCLUDED.validated_at,updated_at=EXCLUDED.updated_at,
			version=third_party_documents.version+1
		RETURNING id::text,relationship_id::text,assessment_id::text,request_id::text,artifact_id::text,document_type,reference,issued_by,
			issued_on,expires_on,evidence_class,status,validated_by_principal_id::text,validated_at,version,created_at,updated_at`,
		tenantID, record.LegalEntityID, current.RelationshipID, current.ID, current.CurrentRequestID, record.Artifact.ID,
		record.DocumentType, reference, issuedBy, issuedDate, record.ExpiresOn, record.EvidenceClass, status, record.ActorPrincipalID, record.At.UTC()).Scan(
		&document.ID, &document.RelationshipID, &document.AssessmentID, &document.RequestID, &document.ArtifactID, &document.DocumentType,
		&document.Reference, &document.IssuedBy, &document.IssuedOn, &document.ExpiresOn, &document.EvidenceClass, &document.Status,
		&document.ValidatedByPrincipalID, &document.ValidatedAt, &document.Version, &document.CreatedAt, &document.UpdatedAt,
	)
	if err != nil {
		return AssessmentDocument{}, Assessment{}, fmt.Errorf("store assessment document review: %w", err)
	}
	document.Scope = record.Scope
	current.Version++
	current.UpdatedAt = record.At.UTC()
	current.ReviewerPrincipalID = record.ActorPrincipalID
	if err := updateAssessment(ctx, tx, tenantID, current); err != nil {
		return AssessmentDocument{}, Assessment{}, err
	}
	if err := appendAssessmentDocumentEvent(ctx, tx, tenantID, current, document, record.ActorPrincipalID, eventType); err != nil {
		return AssessmentDocument{}, Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AssessmentDocument{}, Assessment{}, fmt.Errorf("commit assessment document review: %w", err)
	}
	return document, current, nil
}

func (r *PostgresRepository) ListAssessmentDocuments(ctx context.Context, scope Scope, assessmentID string, limit int) ([]AssessmentDocument, error) {
	assessmentID = strings.TrimSpace(assessmentID)
	if !validAssessmentScope(scope) || !validAssessmentIdentifier(assessmentID) || limit < 1 || limit > assessmentReviewMaxArtifacts+1 {
		return nil, ErrInvalid
	}
	rows, err := r.pool.Query(ctx, `
		SELECT d.id::text,t.slug,d.legal_entity_id::text,d.relationship_id::text,d.assessment_id::text,d.request_id::text,d.artifact_id::text,
			d.document_type,d.reference,d.issued_by,d.issued_on,d.expires_on,d.evidence_class,d.status,d.validated_by_principal_id::text,
			d.validated_at,d.version,d.created_at,d.updated_at
		FROM third_party_documents d JOIN tenants t ON t.id=d.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND d.legal_entity_id::text=$2 AND d.assessment_id::text=$3
		ORDER BY d.updated_at,d.id LIMIT $4`, scope.TenantID, scope.LegalEntityID, assessmentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list assessment documents: %w", err)
	}
	defer rows.Close()
	values := make([]AssessmentDocument, 0)
	for rows.Next() {
		var value AssessmentDocument
		if err := rows.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.RelationshipID, &value.AssessmentID, &value.RequestID,
			&value.ArtifactID, &value.DocumentType, &value.Reference, &value.IssuedBy, &value.IssuedOn, &value.ExpiresOn, &value.EvidenceClass,
			&value.Status, &value.ValidatedByPrincipalID, &value.ValidatedAt, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan assessment document: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list assessment documents: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) ResolveAssessmentRequest(ctx context.Context, tenantID string, origin evidence.RequestOrigin, requestID string) (AssessmentSubmissionTarget, error) {
	tenantID, requestID = strings.TrimSpace(tenantID), strings.TrimSpace(requestID)
	origin.Type, origin.ID = strings.ToUpper(strings.TrimSpace(origin.Type)), strings.TrimSpace(origin.ID)
	if tenantID == "" || requestID == "" || origin.Type != AssessmentRequestOrigin || !validAssessmentIdentifier(origin.ID) || origin.Version < 1 {
		return AssessmentSubmissionTarget{}, ErrNotFound
	}
	var target AssessmentSubmissionTarget
	err := r.pool.QueryRow(ctx, `
		SELECT t.slug,a.legal_entity_id::text,a.id::text,a.version,l.request_id::text
		FROM third_party_assessments a
		JOIN tenants t ON t.id=a.tenant_id
		JOIN third_party_assessment_request_links l
		  ON l.tenant_id=a.tenant_id AND l.legal_entity_id=a.legal_entity_id AND l.assessment_id=a.id
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND a.id::text=$2
		  AND a.status='COLLECTING'
		  AND a.current_request_id=l.request_id
		  AND l.is_current
		  AND l.request_id::text=$3
		  AND l.origin_type=$4
		  AND l.origin_id::text=$2
		  AND l.origin_sequence=$5`, tenantID, origin.ID, requestID, origin.Type, origin.Version).Scan(
		&target.TenantID, &target.LegalEntityID, &target.AssessmentID, &target.AssessmentVersion, &target.RequestID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AssessmentSubmissionTarget{}, ErrNotFound
	}
	if err != nil {
		return AssessmentSubmissionTarget{}, fmt.Errorf("resolve assessment request: %w", err)
	}
	return target, nil
}

const assessmentProjection = `a.id::text,t.slug,a.legal_entity_id::text,a.relationship_id::text,a.review_kind,a.stable_episode_key,a.status,
	a.form_template_id::text,a.form_template_version,COALESCE(a.current_request_id::text,''),COALESCE(a.submission_id::text,''),COALESCE(a.review_matter_id::text,''),
	a.review_due_at,a.started_by_principal_id::text,a.started_at,a.submitted_at,a.review_started_at,a.completed_at,
	COALESCE(a.reviewer_principal_id::text,''),COALESCE(a.conclusion,''),a.conclusion_uncertainty,a.conclusion_rationale,a.next_review_recommended_at,
	a.cancellation_reason,a.version,a.created_at,a.updated_at`

const assessmentSelect = `SELECT ` + assessmentProjection + ` FROM third_party_assessments a JOIN tenants t ON t.id=a.tenant_id `

const assessmentRequestLinkSelect = `SELECT l.tenant_id::text,l.legal_entity_id::text,l.assessment_id::text,l.request_id::text,l.purpose,l.sequence,
	l.origin_type,l.origin_id::text,l.origin_sequence,COALESCE(l.invitation_id::text,''),l.created_at
	FROM third_party_assessment_request_links l JOIN tenants t ON t.id=l.tenant_id `

func scanAssessment(row rowScanner) (Assessment, error) {
	var value Assessment
	err := row.Scan(
		&value.ID, &value.TenantID, &value.LegalEntityID, &value.RelationshipID, &value.ReviewKind, &value.StableEpisodeKey, &value.Status,
		&value.FormTemplateID, &value.FormTemplateVersion, &value.CurrentRequestID, &value.SubmissionID, &value.ReviewMatterID,
		&value.ReviewDueAt, &value.StartedByPrincipalID, &value.StartedAt, &value.SubmittedAt, &value.ReviewStartedAt, &value.CompletedAt,
		&value.ReviewerPrincipalID, &value.Conclusion, &value.ConclusionUncertainty, &value.ConclusionRationale, &value.NextReviewRecommendedAt,
		&value.CancellationReason, &value.Version, &value.CreatedAt, &value.UpdatedAt,
	)
	return value, err
}

func scanAssessmentRequestLink(row rowScanner) (AssessmentRequestLink, error) {
	var value AssessmentRequestLink
	err := row.Scan(&value.TenantID, &value.LegalEntityID, &value.AssessmentID, &value.RequestID, &value.Purpose, &value.Sequence,
		&value.OriginType, &value.OriginID, &value.OriginSequence, &value.InvitationID, &value.CreatedAt)
	return value, err
}

func lockAssessment(ctx context.Context, tx pgx.Tx, tenantID, legalEntityID, assessmentID string) (Assessment, error) {
	value, err := scanAssessment(tx.QueryRow(ctx, assessmentSelect+`
		WHERE a.tenant_id=$1::uuid AND a.legal_entity_id::text=$2 AND a.id::text=$3 FOR UPDATE`, tenantID, legalEntityID, assessmentID))
	return value, mapAssessmentReadError(err, "lock assessment")
}

func updateAssessment(ctx context.Context, tx pgx.Tx, tenantID string, value Assessment) error {
	result, err := tx.Exec(ctx, `
		UPDATE third_party_assessments SET
			status=$4,current_request_id=NULLIF($5,'')::uuid,submission_id=NULLIF($6,'')::uuid,review_matter_id=NULLIF($7,'')::uuid,
			submitted_at=$8,review_started_at=$9,completed_at=$10,reviewer_principal_id=NULLIF($11,'')::uuid,
			conclusion=NULLIF($12,''),conclusion_uncertainty=$13,conclusion_rationale=$14,next_review_recommended_at=$15,
			cancellation_reason=$16,version=$17,updated_at=$18
		WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND id::text=$3`, tenantID, value.LegalEntityID, value.ID,
		value.Status, value.CurrentRequestID, value.SubmissionID, value.ReviewMatterID, value.SubmittedAt, value.ReviewStartedAt, value.CompletedAt,
		value.ReviewerPrincipalID, value.Conclusion, value.ConclusionUncertainty, value.ConclusionRationale,
		value.NextReviewRecommendedAt, value.CancellationReason, value.Version, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update assessment: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrVersionConflict
	}
	return nil
}

func appendAssessmentEvent(ctx context.Context, tx pgx.Tx, tenantID string, value Assessment, actorID, eventType string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,NULLIF($4,'')::uuid,$5,
			jsonb_build_object('status',$6::text,'relationship_id',$7::text,'request_id',$8::text,'matter_id',$9::text,'submission_id',$10::text),$11)`,
		tenantID, value.ID, value.Version, actorID, eventType, value.Status, value.RelationshipID,
		value.CurrentRequestID, value.ReviewMatterID, value.SubmissionID, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append assessment event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,
			jsonb_build_object('version',$4::bigint,'status',$5::text,'relationship_id',$6::text,'request_id',$7::text,'matter_id',$8::text),$9,$9)`,
		tenantID, value.ID, eventType, value.Version, value.Status, value.RelationshipID, value.CurrentRequestID, value.ReviewMatterID, value.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append assessment outbox event: %w", err)
	}
	return nil
}

func appendAssessmentDocumentEvent(ctx context.Context, tx pgx.Tx, tenantID string, assessment Assessment, document AssessmentDocument, actorID, eventType string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,$4::uuid,$5,
			jsonb_build_object('status',$6::text,'relationship_id',$7::text,'request_id',$8::text,'artifact_id',$9::text,'document_id',$10::text,'document_status',$11::text),$12)`,
		tenantID, assessment.ID, assessment.Version, actorID, eventType, assessment.Status, assessment.RelationshipID,
		assessment.CurrentRequestID, document.ArtifactID, document.ID, document.Status, assessment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append assessment document event: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,
			jsonb_build_object('version',$4::bigint,'status',$5::text,'relationship_id',$6::text,'request_id',$7::text,'artifact_id',$8::text,'document_id',$9::text,'document_status',$10::text),$11,$11)`,
		tenantID, assessment.ID, eventType, assessment.Version, assessment.Status, assessment.RelationshipID,
		assessment.CurrentRequestID, document.ArtifactID, document.ID, document.Status, assessment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("append assessment document outbox event: %w", err)
	}
	return nil
}

func verifyPostgresAssessmentCompletionReady(ctx context.Context, tx pgx.Tx, tenantID string, assessment Assessment) error {
	var requestStatus evidence.RequestStatus
	var presentationJSON, sectionsJSON, fieldsJSON, answersJSON []byte
	err := tx.QueryRow(ctx, `
		SELECT req.status,req.presentation,req.sections,req.fields,s.answers
		FROM capture_requests req
		JOIN capture_submissions s ON s.id=$3::uuid AND s.tenant_id=req.tenant_id AND s.request_id=req.id
		WHERE req.tenant_id=$1::uuid AND req.id=$2::uuid
		  AND req.subject_type='VENDOR_RELATIONSHIP' AND req.subject_id=$4
		  AND req.origin_type=$5 AND req.origin_id=$6 AND req.form_template_id=$7::uuid AND req.form_template_version=$8
		FOR SHARE OF req,s`, tenantID, assessment.CurrentRequestID, assessment.SubmissionID, assessment.RelationshipID,
		AssessmentRequestOrigin, assessment.ID, assessment.FormTemplateID, assessment.FormTemplateVersion).Scan(
		&requestStatus, &presentationJSON, &sectionsJSON, &fieldsJSON, &answersJSON,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAssessmentCompletionBlocked
	}
	if err != nil {
		return fmt.Errorf("load assessment completion evidence: %w", err)
	}
	if requestStatus != evidence.RequestSubmitted {
		return ErrAssessmentCompletionBlocked
	}
	var presentation formcontract.Presentation
	var sections []formcontract.Section
	var fields []evidence.Field
	var answers map[string]formcontract.AnswerValue
	if json.Unmarshal(presentationJSON, &presentation) != nil || json.Unmarshal(sectionsJSON, &sections) != nil || json.Unmarshal(fieldsJSON, &fields) != nil || json.Unmarshal(answersJSON, &answers) != nil {
		return ErrAssessmentCompletionBlocked
	}
	contractFields := make([]formcontract.Field, len(fields))
	for index, field := range fields {
		contractFields[index] = formcontract.Field{
			ID: field.ID, SectionID: field.SectionID, Label: field.Label, Type: formcontract.Type(field.Type), Required: field.Required,
			Description: field.Description, Options: append([]string(nil), field.Options...), AcceptedFormats: append([]string(nil), field.AcceptedFormats...),
			Attestation: field.Attestation, Constraints: field.Constraints, Condition: field.Condition, Scoring: field.Scoring,
		}
	}
	visible, err := formcontract.VisibleFields(formcontract.Contract{Presentation: presentation, Sections: sections, Fields: contractFields}, answers)
	if err != nil {
		return ErrAssessmentCompletionBlocked
	}
	artifactIDs := make(map[string]struct{})
	for _, field := range visible {
		answer, answered := answers[field.ID]
		if field.Required && (!answered || !answer.Answered()) {
			return ErrAssessmentCompletionBlocked
		}
		if !answered || !answer.Answered() || !reviewArtifactField(field.Type) {
			continue
		}
		for _, artifactID := range reviewArtifactIDs(answer) {
			if !validAssessmentIdentifier(artifactID) {
				return ErrAssessmentCompletionBlocked
			}
			artifactIDs[artifactID] = struct{}{}
			if len(artifactIDs) > assessmentReviewMaxArtifacts {
				return ErrAssessmentCompletionBlocked
			}
		}
		if field.Required && field.Type == formcontract.TypeVendorDocument {
			if answer.Document == nil {
				return ErrAssessmentCompletionBlocked
			}
			var reviewed bool
			err := tx.QueryRow(ctx, `
				SELECT true FROM third_party_documents
				WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND assessment_id=$3::uuid AND request_id=$4::uuid
				  AND artifact_id=$5::uuid AND status IN ('VALIDATED','REJECTED')
				FOR SHARE`, tenantID, assessment.LegalEntityID, assessment.ID, assessment.CurrentRequestID, answer.Document.ArtifactID).Scan(&reviewed)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAssessmentCompletionBlocked
			}
			if err != nil {
				return fmt.Errorf("verify required assessment document review: %w", err)
			}
		}
	}
	for artifactID := range artifactIDs {
		var status evidence.ArtifactStatus
		err := tx.QueryRow(ctx, `
			SELECT status FROM capture_artifacts
			WHERE tenant_id=$1::uuid AND request_id=$2::uuid AND submission_id=$3::uuid AND id=$4::uuid
			FOR SHARE`, tenantID, assessment.CurrentRequestID, assessment.SubmissionID, artifactID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && status != evidence.ArtifactAvailable) {
			return ErrAssessmentCompletionBlocked
		}
		if err != nil {
			return fmt.Errorf("verify assessment artifact state: %w", err)
		}
	}
	return nil
}

func assessmentReactionReplay(ctx context.Context, tx pgx.Tx, tenantID string, record AssessmentReactionRecord) (Assessment, bool, error) {
	var storedAssessmentID, storedJobID, storedEventID, storedMatterID, storedRequestID, storedSubmissionID string
	var snapshot []byte
	err := tx.QueryRow(ctx, `
		SELECT assessment_id::text,COALESCE(job_id::text,''),COALESCE(event_id::text,''),COALESCE(matter_id::text,''),
			COALESCE(request_id::text,''),COALESCE(submission_id::text,''),result_snapshot
		FROM third_party_assessment_reactions
		WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND reaction_kind=$3 AND causation_id=$4`,
		tenantID, record.LegalEntityID, record.Kind, record.CausationID).Scan(&storedAssessmentID, &storedJobID, &storedEventID,
		&storedMatterID, &storedRequestID, &storedSubmissionID, &snapshot)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, false, nil
	}
	if err != nil {
		return Assessment{}, false, fmt.Errorf("load assessment reaction receipt: %w", err)
	}
	if storedAssessmentID != record.AssessmentID || storedJobID != record.JobID || storedEventID != record.EventID ||
		storedMatterID != record.MatterID || storedRequestID != record.RequestID || storedSubmissionID != record.SubmissionID {
		return Assessment{}, false, ErrInvalid
	}
	var value Assessment
	if err := json.Unmarshal(snapshot, &value); err != nil {
		return Assessment{}, false, fmt.Errorf("decode assessment reaction receipt: %w", err)
	}
	return value, true, nil
}

func mapAssessmentReadError(err error, operation string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	return nil
}

var _ AssessmentRepository = (*PostgresRepository)(nil)
