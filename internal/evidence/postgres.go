//go:build postgres

package evidence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateSource(ctx context.Context, value Source) (Source, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `INSERT INTO evidence_sources(id,tenant_id,legal_entity_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version,created_at,updated_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),(SELECT id FROM legal_entities WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND (id::text=$3 OR code=$3) LIMIT 1),$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10,$11,$12,$13,$13) RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),code,name,source_type,authority_class,COALESCE(owner_principal_id::text,''),expected_freshness_minutes,last_observed_at,last_success_at,health,status,version,created_at,updated_at`, value.ID, value.TenantID, value.LegalEntityID, value.Code, value.Name, value.Type, value.AuthorityClass, value.OwnerPrincipalID, value.ExpectedFreshnessMinutes, value.Health, value.Status, value.Version, value.CreatedAt)
	created, err := scanSource(row)
	if err != nil {
		return Source{}, fmt.Errorf("create evidence source: %w", err)
	}
	endpoint := strings.TrimSpace(value.Endpoint)
	if endpoint != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO source_connections(
				tenant_id,source_id,code,name,adapter_kind,adapter_version,secret_ref,definition,
				declared_capabilities,verified_capabilities,owner_principal_id,status,is_current,
				effective_from,version,created_by,created_at,updated_at
			) VALUES (
				(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),$2::uuid,$3,$4,$5,$6,'',
				jsonb_build_object('endpoint',$7::text),'[]'::jsonb,'[]'::jsonb,NULLIF($8,'')::uuid,
				'ACTIVE',true,$9,1,NULLIF($8,'')::uuid,$9,$9
			)`, value.TenantID, created.ID, sourceaccess.ReferenceConnectionCode, sourceaccess.ReferenceConnectionName, sourceaccess.AdapterReference, sourceaccess.ReferenceAdapterVersion, endpoint, value.OwnerPrincipalID, value.CreatedAt); err != nil {
			return Source{}, fmt.Errorf("create source reference connection: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Source{}, err
	}
	return created, nil
}

func (r *PostgresRepository) ListSources(ctx context.Context, tenant string, limit int) ([]Source, error) {
	rows, err := r.pool.Query(ctx, `SELECT es.id::text,t.id::text,COALESCE(es.legal_entity_id::text,''),es.code,es.name,es.source_type,es.authority_class,COALESCE(es.owner_principal_id::text,''),es.expected_freshness_minutes,es.last_observed_at,es.last_success_at,es.health,es.status,es.version,es.created_at,es.updated_at FROM evidence_sources es JOIN tenants t ON t.id=es.tenant_id WHERE (t.id::text=$1 OR t.slug=$1) ORDER BY es.name,es.id LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, fmt.Errorf("list evidence sources: %w", err)
	}
	defer rows.Close()
	values := []Source{}
	for rows.Next() {
		value, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) RecordSourceObservation(ctx context.Context, observation SourceObservation, health SourceHealth) (Source, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Source{}, err
	}
	defer tx.Rollback(ctx)
	current, err := scanSource(tx.QueryRow(ctx, `SELECT es.id::text,t.id::text,COALESCE(es.legal_entity_id::text,''),es.code,es.name,es.source_type,es.authority_class,COALESCE(es.owner_principal_id::text,''),es.expected_freshness_minutes,es.last_observed_at,es.last_success_at,es.health,es.status,es.version,es.created_at,es.updated_at FROM evidence_sources es JOIN tenants t ON t.id=es.tenant_id WHERE es.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2) FOR UPDATE`, observation.SourceID, observation.TenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Source{}, ErrNotFound
	}
	if err != nil {
		return Source{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO source_observations(id,tenant_id,source_id,observed_at,success,unavailable,latency_ms,detail,recorded_by) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid)`, observation.ID, observation.TenantID, observation.SourceID, observation.ObservedAt, observation.Success, observation.Unavailable, observation.LatencyMS, observation.Detail, observation.RecordedBy)
	if err != nil {
		return Source{}, err
	}
	row := tx.QueryRow(ctx, `UPDATE evidence_sources SET last_observed_at=$3,last_success_at=CASE WHEN $4 THEN $3 ELSE last_success_at END,health=$5,version=version+1,updated_at=$3 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),code,name,source_type,authority_class,COALESCE(owner_principal_id::text,''),expected_freshness_minutes,last_observed_at,last_success_at,health,status,version,created_at,updated_at`, observation.SourceID, observation.TenantID, observation.ObservedAt, observation.Success, health)
	updated, err := scanSource(row)
	if err != nil {
		return Source{}, err
	}
	if current.Health != updated.Health {
		_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'EVIDENCE_SOURCE',$2::uuid,'SourceHealthChanged',jsonb_build_object('from',$3::text,'to',$4::text,'source_code',$5::text),$6,$6,$6)`, observation.TenantID, observation.SourceID, current.Health, updated.Health, updated.Code, observation.ObservedAt)
		if err != nil {
			return Source{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return Source{}, err
	}
	return updated, nil
}

func (r *PostgresRepository) EvaluateSourceHealth(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `WITH due AS (SELECT id,tenant_id,health AS from_health,code FROM evidence_sources WHERE status='ACTIVE' AND last_success_at IS NOT NULL AND health NOT IN ('STALE','UNAVAILABLE') AND last_success_at + make_interval(mins=>expected_freshness_minutes) <= $1 ORDER BY last_success_at,id LIMIT $2 FOR UPDATE SKIP LOCKED), updated AS (UPDATE evidence_sources es SET health='STALE',version=version+1,updated_at=$1 FROM due WHERE es.id=due.id RETURNING es.id,es.tenant_id) SELECT u.id::text,u.tenant_id::text,d.from_health,d.code FROM updated u JOIN due d ON d.id=u.id`, now, limit)
	if err != nil {
		return 0, err
	}
	type changed struct{ id, tenant, from, code string }
	values := []changed{}
	for rows.Next() {
		var value changed
		if err := rows.Scan(&value.id, &value.tenant, &value.from, &value.code); err != nil {
			rows.Close()
			return 0, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	for _, value := range values {
		if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'EVIDENCE_SOURCE',$2::uuid,'SourceHealthChanged',jsonb_build_object('from',$3::text,'to','STALE','source_code',$4::text),$5,$5,$5)`, value.tenant, value.id, value.from, value.code, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(values), nil
}

func (r *PostgresRepository) ExpireRequests(ctx context.Context, now time.Time, limit int) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id::text,tenant_id::text FROM capture_requests WHERE status IN ('READY','IN_PROGRESS') AND deadline <= $1 ORDER BY deadline,id LIMIT $2 FOR UPDATE SKIP LOCKED`, now, limit)
	if err != nil {
		return 0, err
	}
	type due struct{ id, tenant string }
	values := []due{}
	for rows.Next() {
		var value due
		if err := rows.Scan(&value.id, &value.tenant); err != nil {
			rows.Close()
			return 0, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	for _, value := range values {
		if _, err := tx.Exec(ctx, `UPDATE capture_requests SET status='EXPIRED',version=version+1,updated_at=$2 WHERE id=$1::uuid AND status IN ('READY','IN_PROGRESS')`, value.id, now); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'EVIDENCE_REQUEST',$2::uuid,'EvidenceRequestExpired','{}'::jsonb,$3,$3,$3)`, value.tenant, value.id, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(values), nil
}

func (r *PostgresRepository) CreateRequest(ctx context.Context, value Request) (Request, error) {
	if !value.Deadline.After(value.CreatedAt) {
		return Request{}, ErrRequestClosed
	}
	facts, err := json.Marshal(value.KnownFacts)
	if err != nil {
		return Request{}, err
	}
	presentation, err := json.Marshal(value.Presentation)
	if err != nil {
		return Request{}, err
	}
	sections, err := json.Marshal(value.Sections)
	if err != nil {
		return Request{}, err
	}
	fields, err := json.Marshal(value.Fields)
	if err != nil {
		return Request{}, err
	}
	sourceBindingValues := value.SourceBindings
	if sourceBindingValues == nil {
		sourceBindingValues = []RequestBindingReference{}
	}
	sourceBindings, err := json.Marshal(sourceBindingValues)
	if err != nil {
		return Request{}, err
	}
	row := r.pool.QueryRow(ctx, `INSERT INTO capture_requests(
		id,tenant_id,subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,estimated_minutes,deadline,
		known_facts,presentation,sections,fields,source_bindings,form_template_id,form_template_version,collection_period_start,collection_period_end,
		origin_type,origin_id,origin_version,status,created_by,version,created_at,updated_at
	) VALUES(
		$1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3,$4,$5,$6,$7,$8,$9,$10,$11,
		$12::jsonb,$13::jsonb,$14::jsonb,$15::jsonb,$16::jsonb,NULLIF($17,'')::uuid,NULLIF($18,0),$19,$20,
		NULLIF($21,''),NULLIF($22,''),NULLIF($23,0),$24,NULLIF($25,'')::uuid,$26,$27,$27
	) RETURNING `+requestReturningColumns,
		value.ID, value.TenantID, value.SubjectType, value.SubjectID, value.Title, value.Purpose, value.WhyYou, value.Sensitivity, value.AudienceType, value.EstimatedMinutes, value.Deadline,
		string(facts), string(presentation), string(sections), string(fields), string(sourceBindings), value.FormTemplateID, value.FormTemplateVersion, value.CollectionPeriodStart, value.CollectionPeriodEnd,
		value.Origin.Type, value.Origin.ID, value.Origin.Version, value.Status, value.CreatedBy, value.Version, value.CreatedAt)
	created, err := scanRequest(row)
	if err != nil {
		return Request{}, fmt.Errorf("create evidence request: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) ListRequests(ctx context.Context, tenant string, limit int) ([]Request, error) {
	rows, err := r.pool.Query(ctx, requestSelect+` WHERE (t.id::text=$1 OR t.slug=$1) ORDER BY er.deadline,er.id LIMIT $2`, tenant, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []Request{}
	for rows.Next() {
		value, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) GetRequest(ctx context.Context, tenant, id string) (Request, error) {
	value, err := scanRequest(r.pool.QueryRow(ctx, requestSelect+` WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2)`, id, tenant))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	return value, err
}

func (r *PostgresRepository) GetRequestByOrigin(ctx context.Context, tenant string, origin RequestOrigin) (Request, error) {
	origin = origin.normalized()
	value, err := scanRequest(r.pool.QueryRow(ctx, requestSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1)
		  AND er.origin_type=$2 AND er.origin_id=$3 AND er.origin_version=$4`, tenant, origin.Type, origin.ID, origin.Version))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	return value, err
}

func (r *PostgresRepository) Submit(ctx context.Context, submission Submission) (SubmissionReceipt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	defer tx.Rollback(ctx)
	request, err := scanRequest(tx.QueryRow(ctx, requestSelect+` WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2) FOR UPDATE`, submission.RequestID, submission.TenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return SubmissionReceipt{}, ErrNotFound
	}
	if err != nil {
		return SubmissionReceipt{}, err
	}
	if !requestOpenAt(request, submission.SubmittedAt) {
		return SubmissionReceipt{}, ErrRequestClosed
	}
	if request.Version != submission.ExpectedVersion {
		return SubmissionReceipt{}, ErrVersionConflict
	}
	answers, err := json.Marshal(submission.Answers)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	answerProvenanceValues := submission.AnswerProvenance
	if answerProvenanceValues == nil {
		answerProvenanceValues = map[string]AnswerProvenance{}
	}
	answerProvenance, err := json.Marshal(answerProvenanceValues)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO capture_submissions(id,tenant_id,request_id,session_id,submitted_by,channel,answers,answer_provenance,submitted_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,NULLIF($5,'')::uuid,$6,$7::jsonb,$8::jsonb,$9)`, submission.ID, submission.TenantID, submission.RequestID, submission.SessionID, submission.SubmittedBy, submission.Channel, string(answers), string(answerProvenance), submission.SubmittedAt)
	if err != nil {
		return SubmissionReceipt{}, err
	}
	var version int64
	if err := tx.QueryRow(ctx, `UPDATE capture_requests SET status='SUBMITTED',version=version+1,updated_at=$3 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) RETURNING version`, submission.RequestID, submission.TenantID, submission.SubmittedAt).Scan(&version); err != nil {
		return SubmissionReceipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES((SELECT id FROM tenants WHERE id::text=$1 OR slug=$1),'EVIDENCE_REQUEST',$2::uuid,'EvidenceResponseSubmitted',jsonb_build_object('submission_id',$3::text,'channel',$4::text),$5,$5,$5)`, submission.TenantID, submission.RequestID, submission.ID, submission.Channel, submission.SubmittedAt); err != nil {
		return SubmissionReceipt{}, err
	}
	if submission.SessionID != "" {
		if _, err = tx.Exec(ctx, `DELETE FROM capture_response_drafts
			WHERE tenant_id=(SELECT id FROM tenants WHERE id::text=$1 OR slug=$1)
			  AND request_id=$2::uuid AND session_id=$3::uuid`, submission.TenantID, submission.RequestID, submission.SessionID); err != nil {
			return SubmissionReceipt{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return SubmissionReceipt{}, err
	}
	return SubmissionReceipt{SubmissionID: submission.ID, RequestID: submission.RequestID, Status: RequestSubmitted, SubmittedAt: submission.SubmittedAt, Version: version}, nil
}

func (r *PostgresRepository) GetSubmission(ctx context.Context, tenant, id string) (Submission, error) {
	var value Submission
	var answers, provenance []byte
	err := r.pool.QueryRow(ctx, `
		SELECT s.id::text,s.tenant_id::text,s.request_id::text,COALESCE(s.session_id::text,''),COALESCE(s.submitted_by::text,''),s.channel,s.answers,s.answer_provenance,s.submitted_at
		FROM capture_submissions s JOIN tenants t ON t.id=s.tenant_id
		WHERE (t.id::text=$1 OR t.slug=$1) AND s.id=$2::uuid`, tenant, id).Scan(
		&value.ID, &value.TenantID, &value.RequestID, &value.SessionID, &value.SubmittedBy, &value.Channel, &answers, &provenance, &value.SubmittedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Submission{}, ErrNotFound
	}
	if err != nil {
		return Submission{}, err
	}
	if err := json.Unmarshal(answers, &value.Answers); err != nil {
		return Submission{}, err
	}
	if err := json.Unmarshal(provenance, &value.AnswerProvenance); err != nil {
		return Submission{}, err
	}
	return value, nil
}

func (r *PostgresRepository) CreateInvitation(ctx context.Context, value Invitation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	request, err := scanRequest(tx.QueryRow(ctx, requestSelect+` WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2) FOR UPDATE`, value.RequestID, value.TenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if !requestOpenAt(request, value.CreatedAt) || !value.ExpiresAt.After(value.CreatedAt) || value.ExpiresAt.After(request.Deadline) {
		return ErrRequestClosed
	}
	if _, err := tx.Exec(ctx, `INSERT INTO capture_invitations(id,tenant_id,request_id,token_hash,audience_hash,audience_hint,purpose,expires_at,max_redemptions,created_by,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,$4,$5,$6,$7,$8,$9,NULLIF($10,'')::uuid,$11)`, value.ID, value.TenantID, value.RequestID, value.TokenHash, value.AudienceHash, value.AudienceHint, value.Purpose, value.ExpiresAt, value.MaxRedemptions, value.CreatedBy, value.CreatedAt); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *PostgresRepository) RedeemInvitation(ctx context.Context, input RedeemInput) (Session, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback(ctx)
	var invitationID, tenantID, requestID, audienceHint string
	var expiresAt, requestDeadline time.Time
	var requestStatus RequestStatus
	var maxRedemptions, redemptions int
	var revoked sql.NullTime
	err = tx.QueryRow(ctx, `SELECT ei.id::text,ei.tenant_id::text,ei.request_id::text,ei.audience_hint,ei.expires_at,ei.max_redemptions,ei.redemptions,ei.revoked_at,er.deadline,er.status FROM capture_invitations ei JOIN capture_requests er ON er.id=ei.request_id AND er.tenant_id=ei.tenant_id WHERE ei.token_hash=$1 AND ei.audience_hash=$2 FOR UPDATE`, input.InvitationTokenHash, input.AudienceHash).Scan(&invitationID, &tenantID, &requestID, &audienceHint, &expiresAt, &maxRedemptions, &redemptions, &revoked, &requestDeadline, &requestStatus)
	if errors.Is(err, pgx.ErrNoRows) || revoked.Valid || redemptions >= maxRedemptions || !input.Now.Before(expiresAt) || (requestStatus != RequestReady && requestStatus != RequestInProgress) || !input.Now.Before(requestDeadline) {
		return Session{}, ErrInvitationInvalid
	}
	if err != nil {
		return Session{}, err
	}
	sessionExpires := input.SessionExpiresAt
	if sessionExpires.After(expiresAt) {
		sessionExpires = expiresAt
	}
	if sessionExpires.After(requestDeadline) {
		sessionExpires = requestDeadline
	}
	if !sessionExpires.After(input.Now) {
		return Session{}, ErrInvitationInvalid
	}
	if _, err := tx.Exec(ctx, `UPDATE capture_invitations SET redemptions=redemptions+1,last_redeemed_at=$2 WHERE id=$1::uuid`, invitationID, input.Now); err != nil {
		return Session{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO capture_sessions(id,tenant_id,request_id,invitation_id,token_hash,audience_hint,expires_at,created_at) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8)`, input.SessionID, tenantID, requestID, invitationID, input.SessionTokenHash, audienceHint, sessionExpires, input.Now); err != nil {
		return Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Session{}, err
	}
	return Session{ID: input.SessionID, TenantID: tenantID, RequestID: requestID, AudienceHint: audienceHint, TokenHash: input.SessionTokenHash, ExpiresAt: sessionExpires, CreatedAt: input.Now}, nil
}

func (r *PostgresRepository) SessionByTokenHash(ctx context.Context, hash []byte, now time.Time) (Session, error) {
	var value Session
	var revoked sql.NullTime
	err := r.pool.QueryRow(ctx, `SELECT es.id::text,es.tenant_id::text,es.request_id::text,es.audience_hint,es.token_hash,es.expires_at,es.revoked_at,es.created_at FROM capture_sessions es WHERE es.token_hash=$1 AND es.expires_at>$2`, hash, now).Scan(&value.ID, &value.TenantID, &value.RequestID, &value.AudienceHint, &value.TokenHash, &value.ExpiresAt, &revoked, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) || revoked.Valid {
		return Session{}, ErrSessionInvalid
	}
	return value, err
}

func (r *PostgresRepository) RevokeInvitation(ctx context.Context, tenant, id string, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE capture_invitations SET revoked_at=COALESCE(revoked_at,$3) WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)`, id, tenant, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) RevokeSession(ctx context.Context, tenant, id string, now time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE capture_sessions SET revoked_at=COALESCE(revoked_at,$3) WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2)`, id, tenant, now)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateArtifact(ctx context.Context, value Artifact) (Artifact, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Artifact{}, err
	}
	defer tx.Rollback(ctx)
	request, err := scanRequest(tx.QueryRow(ctx, requestSelect+` WHERE er.id=$1::uuid AND (t.id::text=$2 OR t.slug=$2) FOR UPDATE`, value.RequestID, value.TenantID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Artifact{}, ErrNotFound
	}
	if err != nil {
		return Artifact{}, err
	}
	if !requestOpenAt(request, value.CreatedAt) {
		return Artifact{}, ErrRequestClosed
	}
	row := tx.QueryRow(ctx, `INSERT INTO capture_artifacts(id,tenant_id,request_id,submission_id,file_name,media_type,size_bytes,sha256,storage_key,status,created_by,created_at) VALUES($1::uuid,(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2),$3::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,NULLIF($11,'')::uuid,$12) RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),request_id::text,COALESCE(submission_id::text,''),file_name,media_type,size_bytes,sha256,storage_key,status,COALESCE(created_by::text,''),created_at`, value.ID, value.TenantID, value.RequestID, value.SubmissionID, value.FileName, value.MediaType, value.SizeBytes, value.SHA256, value.StorageKey, value.Status, value.CreatedBy, value.CreatedAt)
	var created Artifact
	if err := row.Scan(&created.ID, &created.TenantID, &created.RequestID, &created.SubmissionID, &created.FileName, &created.MediaType, &created.SizeBytes, &created.SHA256, &created.StorageKey, &created.Status, &created.CreatedBy, &created.CreatedAt); err != nil {
		return Artifact{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Artifact{}, err
	}
	return created, nil
}

const requestReturningColumns = `id::text,(SELECT slug FROM tenants WHERE id=tenant_id),subject_type,subject_id,title,purpose,why_you,sensitivity,audience_type,
	estimated_minutes,deadline,known_facts,presentation,sections,fields,source_bindings,
	COALESCE(form_template_id::text,''),COALESCE(form_template_version,0),collection_period_start,collection_period_end,
	COALESCE(origin_type,''),COALESCE(origin_id,''),COALESCE(origin_version,0),status,COALESCE(created_by::text,''),version,created_at,updated_at`

const requestProjection = `er.id::text,t.id::text,er.subject_type,er.subject_id,er.title,er.purpose,er.why_you,er.sensitivity,er.audience_type,
	er.estimated_minutes,er.deadline,er.known_facts,er.presentation,er.sections,er.fields,er.source_bindings,
	COALESCE(er.form_template_id::text,''),COALESCE(er.form_template_version,0),er.collection_period_start,er.collection_period_end,
	COALESCE(er.origin_type,''),COALESCE(er.origin_id,''),COALESCE(er.origin_version,0),er.status,COALESCE(er.created_by::text,''),er.version,er.created_at,er.updated_at`

const requestSelect = `SELECT ` + requestProjection + ` FROM capture_requests er JOIN tenants t ON t.id=er.tenant_id`

type scanner interface{ Scan(...any) error }

func scanSource(row scanner) (Source, error) {
	var value Source
	var observed, success sql.NullTime
	if err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.Code, &value.Name, &value.Type, &value.AuthorityClass, &value.OwnerPrincipalID, &value.ExpectedFreshnessMinutes, &observed, &success, &value.Health, &value.Status, &value.Version, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return Source{}, err
	}
	if observed.Valid {
		value.LastObservedAt = pointerTime(observed.Time)
	}
	if success.Valid {
		value.LastSuccessAt = pointerTime(success.Time)
	}
	return value, nil
}

func scanRequest(row scanner) (Request, error) {
	var value Request
	var facts, presentation, sections, fields, sourceBindings []byte
	if err := row.Scan(
		&value.ID, &value.TenantID, &value.SubjectType, &value.SubjectID, &value.Title, &value.Purpose, &value.WhyYou, &value.Sensitivity, &value.AudienceType,
		&value.EstimatedMinutes, &value.Deadline, &facts, &presentation, &sections, &fields, &sourceBindings,
		&value.FormTemplateID, &value.FormTemplateVersion, &value.CollectionPeriodStart, &value.CollectionPeriodEnd,
		&value.Origin.Type, &value.Origin.ID, &value.Origin.Version, &value.Status, &value.CreatedBy, &value.Version, &value.CreatedAt, &value.UpdatedAt,
	); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(facts, &value.KnownFacts); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(presentation, &value.Presentation); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(sections, &value.Sections); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(fields, &value.Fields); err != nil {
		return Request{}, err
	}
	if err := json.Unmarshal(sourceBindings, &value.SourceBindings); err != nil {
		return Request{}, err
	}
	return value, nil
}

var _ OriginRequestStore = (*PostgresRepository)(nil)
