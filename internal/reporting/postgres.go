//go:build postgres

package reporting

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	platformid "github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxRepositoryListRows = 500
	maxClaimWorkerIDRunes = 200
)

var reportFailureCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{0,99}$`)

// PostgresRepository is the durable report state boundary. Every current-row
// change, revision decision, and outbox event is committed in one transaction;
// report reads are tenant and legal-entity scoped before any bounded limit.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

const definitionProjection = `d.id::text,d.tenant_id::text,d.legal_entity_id::text,d.code,d.name,d.description,
	d.dataset,d.scope_kind,COALESCE(d.scope_ref::text,''),d.format,d.filter,d.status,d.current_version,
	d.checksum,d.maker_id::text,COALESCE(d.checker_id::text,''),COALESCE(d.reviewer_id::text,''),
	d.reviewer_note,d.effective_from,d.effective_until,d.submitted_at,d.approved_at,d.retired_at,
	d.created_at,d.updated_at,d.version`

const runProjection = `r.id::text,r.tenant_id::text,r.legal_entity_id::text,r.definition_id::text,
	r.definition_version,r.definition_code,r.definition_checksum,r.scope_kind,COALESCE(r.scope_ref::text,''),
	r.requested_by_ref,r.as_of,r.source_boundary,r.filter,r.dataset,r.format,r.status,r.attempt_count,
	r.row_count,COALESCE(r.data_object_key,''),COALESCE(r.data_sha256,''),
	COALESCE(r.manifest_object_key,''),COALESCE(r.manifest_sha256,''),COALESCE(r.failure_code,''),
	r.created_at,r.completed_at,r.expires_at`

type reportingRowScanner interface {
	Scan(...any) error
}

func (r *PostgresRepository) CreateDefinition(ctx context.Context, scope ReportScope, definition ReportDefinition, revision ReportDefinitionRevision) (ReportDefinition, error) {
	if err := r.validateInput(ctx, scope, definition.ID); err != nil {
		return ReportDefinition{}, err
	}
	if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return ReportDefinition{}, ErrNotFound
	}
	if err := validateDefinitionForCreate(definition); err != nil {
		return ReportDefinition{}, err
	}
	if definition.Status != DefinitionDraft || definition.Version != 1 || definition.CurrentVersion != 1 ||
		definition.StoredChecksum == "" || definition.StoredChecksum != definition.Checksum() {
		return ReportDefinition{}, ErrInvalid
	}
	if err := validateDefinitionRevision(revision, definition); err != nil {
		return ReportDefinition{}, err
	}
	filter, err := json.Marshal(definition.Filter)
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("encode report definition filter: %w", err)
	}
	revisionFilter, err := json.Marshal(revision.Filter)
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("encode report definition revision filter: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("begin report definition creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `INSERT INTO report_definitions (
		id,tenant_id,legal_entity_id,code,name,description,dataset,scope_kind,scope_ref,format,filter,
		status,current_version,checksum,maker_id,checker_id,reviewer_id,reviewer_note,effective_from,
		effective_until,submitted_at,approved_at,retired_at,created_at,updated_at,version
	) VALUES (
		$1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8,$9::uuid,$10,$11::jsonb,
		$12,$13,$14,$15::uuid,$16::uuid,$17::uuid,$18,$19,$20,$21,$22,$23,$24,$25,$26
	)`, definition.ID, definition.TenantID, definition.LegalEntityID, definition.Code, definition.Name,
		definition.Description, definition.Dataset, definition.ScopeKind, nullableText(definition.ScopeRef), definition.Format,
		filter, definition.Status, definition.CurrentVersion, definition.StoredChecksum, definition.MakerID,
		nullableText(definition.CheckerID), nullableText(definition.ReviewerID), definition.ReviewerNote,
		definition.EffectiveFrom, definition.EffectiveUntil, definition.SubmittedAt, definition.ApprovedAt,
		definition.RetiredAt, definition.CreatedAt, definition.UpdatedAt, definition.Version)
	if err != nil {
		if isPostgresUniqueViolation(err) {
			return ReportDefinition{}, ErrConflict
		}
		return ReportDefinition{}, fmt.Errorf("insert current report definition: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ReportDefinition{}, ErrConflict
	}

	tag, err = tx.Exec(ctx, `INSERT INTO report_definition_revisions (
		definition_id,tenant_id,legal_entity_id,version,base_version,dataset,scope_kind,scope_ref,format,
		filter,checksum,maker_id,created_at,decision,decision_note
	) VALUES ($1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,$8::uuid,$9,$10::jsonb,$11,$12::uuid,$13,$14,$15)`,
		revision.DefinitionID, revision.TenantID, revision.LegalEntityID, revision.Version, revision.BaseVersion,
		revision.Dataset, revision.ScopeKind, nullableText(revision.ScopeRef), revision.Format, revisionFilter,
		revision.Checksum, revision.MakerID, revision.CreatedAt, revision.Decision, revision.DecisionNote)
	if err != nil {
		if isPostgresUniqueViolation(err) {
			return ReportDefinition{}, ErrConflict
		}
		return ReportDefinition{}, fmt.Errorf("insert report definition revision: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ReportDefinition{}, ErrConflict
	}
	if err := insertReportOutbox(ctx, tx, scope.TenantID, "REPORT_DEFINITION", definition.ID,
		"ReportDefinitionProposed", definition.CreatedAt, map[string]any{
			"legal_entity_id": scope.LegalEntityID, "version": definition.Version, "maker_id": definition.MakerID,
		}); err != nil {
		return ReportDefinition{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReportDefinition{}, fmt.Errorf("commit report definition creation: %w", err)
	}
	definition.Effective = false
	return definition, nil
}

func (r *PostgresRepository) GetDefinition(ctx context.Context, scope ReportScope, id string) (ReportDefinition, error) {
	if err := r.validateInput(ctx, scope, strings.TrimSpace(id)); err != nil {
		return ReportDefinition{}, err
	}
	definition, err := scanDefinition(r.pool.QueryRow(ctx, `SELECT `+definitionProjection+`
		FROM report_definitions d WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid AND d.id=$3::uuid`,
		scope.TenantID, scope.LegalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportDefinition{}, ErrNotFound
	}
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("read current report definition: %w", err)
	}
	definition.Effective = definitionIsEffective(definition, time.Now().UTC())
	return definition, nil
}

func (r *PostgresRepository) GetDefinitionByCode(ctx context.Context, scope ReportScope, code string) (ReportDefinition, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if err := r.validateInput(ctx, scope, code); err != nil {
		return ReportDefinition{}, err
	}
	definition, err := scanDefinition(r.pool.QueryRow(ctx, `SELECT `+definitionProjection+`
		FROM report_definitions d WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid AND d.code=$3
		ORDER BY (d.status='RETIRED')::int,d.created_at DESC,d.id DESC LIMIT 1`, scope.TenantID, scope.LegalEntityID, code))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportDefinition{}, ErrNotFound
	}
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("read report definition by code: %w", err)
	}
	definition.Effective = definitionIsEffective(definition, time.Now().UTC())
	return definition, nil
}

func (r *PostgresRepository) ListDefinitions(ctx context.Context, scope ReportScope, includeRetired bool) ([]ReportDefinition, error) {
	if err := r.validateInput(ctx, scope, "list"); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+definitionProjection+`
		FROM report_definitions d WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid
		  AND ($3 OR d.status<>'RETIRED')
		ORDER BY d.created_at DESC,d.id DESC LIMIT $4`, scope.TenantID, scope.LegalEntityID, includeRetired, maxRepositoryListRows)
	if err != nil {
		return nil, fmt.Errorf("list report definitions: %w", err)
	}
	defer rows.Close()
	values := make([]ReportDefinition, 0)
	for rows.Next() {
		definition, err := scanDefinition(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report definition list: %w", err)
		}
		definition.Effective = definitionIsEffective(definition, time.Now().UTC())
		values = append(values, definition)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list report definitions: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) ListDefinitionHistory(ctx context.Context, scope ReportScope, id string) ([]ReportDefinitionRevision, error) {
	if err := r.validateInput(ctx, scope, strings.TrimSpace(id)); err != nil {
		return nil, err
	}
	if _, err := r.GetDefinition(ctx, scope, id); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT definition_id::text,tenant_id::text,legal_entity_id::text,version,
		base_version,dataset,scope_kind,COALESCE(scope_ref::text,''),format,filter,checksum,maker_id::text,
		created_at,COALESCE(reviewed_by::text,''),reviewed_at,COALESCE(approved_by::text,''),approved_at,
		decision,decision_note
		FROM report_definition_revisions
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND definition_id=$3::uuid
		ORDER BY version LIMIT $4`, scope.TenantID, scope.LegalEntityID, id, maxRepositoryListRows)
	if err != nil {
		return nil, fmt.Errorf("list report definition history: %w", err)
	}
	defer rows.Close()
	values := make([]ReportDefinitionRevision, 0)
	for rows.Next() {
		value, err := scanDefinitionRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report definition revision: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list report definition history: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) TransitionDefinition(ctx context.Context, scope ReportScope, id string, expectedVersion int64, next DefinitionStatus, decision DecisionRecord) (ReportDefinition, error) {
	id = strings.TrimSpace(id)
	if err := r.validateInput(ctx, scope, id); err != nil {
		return ReportDefinition{}, err
	}
	if expectedVersion <= 0 || !isUUID(strings.TrimSpace(decision.ActorID)) || decision.ChecksumSeen == "" || decision.Timestamp.IsZero() {
		return ReportDefinition{}, ErrInvalid
	}
	decision.ActorID = strings.ToLower(strings.TrimSpace(decision.ActorID))
	decision.Action = strings.ToUpper(strings.TrimSpace(decision.Action))
	decision.Note = strings.TrimSpace(decision.Note)
	decision.Timestamp = decision.Timestamp.UTC()
	if len([]rune(decision.Note)) > maxDecisionNote {
		return ReportDefinition{}, ErrInvalid
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("begin report definition transition: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanDefinition(tx.QueryRow(ctx, `SELECT `+definitionProjection+`
		FROM report_definitions d
		WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid AND d.id=$3::uuid
		FOR UPDATE`, scope.TenantID, scope.LegalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportDefinition{}, ErrNotFound
	}
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("lock current report definition: %w", err)
	}
	if current.Version != expectedVersion || decision.ChecksumSeen != current.StoredChecksum {
		return ReportDefinition{}, ErrConflict
	}
	if err := ValidateTransitionForWrite(current, next); err != nil {
		return ReportDefinition{}, err
	}
	if err := validateDefinitionDecision(current, next, decision); err != nil {
		return ReportDefinition{}, err
	}

	transitioned := applyDefinitionDecision(current, next, decision)
	tag, err := tx.Exec(ctx, `UPDATE report_definitions SET
		status=$4,reviewer_id=$5,reviewer_note=$6,submitted_at=$7,approved_at=$8,retired_at=$9,
		effective_from=$10,updated_at=$11,version=version+1,checker_id=$12
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$13`,
		scope.TenantID, scope.LegalEntityID, id, next, nullableText(transitioned.ReviewerID),
		transitioned.ReviewerNote, transitioned.SubmittedAt, transitioned.ApprovedAt, transitioned.RetiredAt,
		transitioned.EffectiveFrom, decision.Timestamp, nullableText(transitioned.CheckerID), expectedVersion)
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("update current report definition: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ReportDefinition{}, ErrConflict
	}

	tag, err = tx.Exec(ctx, revisionDecisionSQL(next, decision), revisionDecisionArguments(current, next, decision)...)
	if err != nil {
		return ReportDefinition{}, fmt.Errorf("record report definition revision decision: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ReportDefinition{}, ErrConflict
	}
	if err := insertReportOutbox(ctx, tx, scope.TenantID, "REPORT_DEFINITION", id,
		"ReportDefinitionStateChanged", decision.Timestamp, map[string]any{
			"from": current.Status, "to": next, "action": decision.Action, "actor_id": decision.ActorID,
			"note": decision.Note, "checksum_seen": decision.ChecksumSeen, "version": expectedVersion + 1,
			"legal_entity_id": scope.LegalEntityID,
		}); err != nil {
		return ReportDefinition{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReportDefinition{}, fmt.Errorf("commit report definition transition: %w", err)
	}
	transitioned.Version = expectedVersion + 1
	transitioned.UpdatedAt = decision.Timestamp
	transitioned.Effective = definitionIsEffective(transitioned, time.Now().UTC())
	return transitioned, nil
}

func (r *PostgresRepository) CreateRun(ctx context.Context, scope ReportScope, run ReportRun) (ReportRun, error) {
	if err := r.validateInput(ctx, scope, run.ID); err != nil {
		return ReportRun{}, err
	}
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if err := validateReportRunIdentity(run); err != nil {
		return ReportRun{}, err
	}
	if err := validateSourceBoundary(run.SourceBoundary); err != nil {
		return ReportRun{}, err
	}
	if run.Status != RunQueued || run.AttemptCount != 0 || run.RowCount != 0 || run.CompletedAt != nil ||
		run.DataObjectKey != "" || run.DataSHA256 != "" || run.ManifestObjectKey != "" || run.ManifestSHA256 != "" || run.FailureCode != "" {
		return ReportRun{}, ErrInvalid
	}
	filter, err := json.Marshal(run.Filter)
	if err != nil {
		return ReportRun{}, fmt.Errorf("encode report run filter: %w", err)
	}
	boundary, err := json.Marshal(run.SourceBoundary)
	if err != nil {
		return ReportRun{}, fmt.Errorf("encode report source boundary: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReportRun{}, fmt.Errorf("begin report run creation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	definition, err := scanDefinition(tx.QueryRow(ctx, `SELECT `+definitionProjection+`
		FROM report_definitions d WHERE d.tenant_id=$1::uuid AND d.legal_entity_id=$2::uuid AND d.id=$3::uuid
		FOR SHARE`, scope.TenantID, scope.LegalEntityID, run.DefinitionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportRun{}, ErrNotFound
	}
	if err != nil {
		return ReportRun{}, fmt.Errorf("lock report definition snapshot: %w", err)
	}
	if definition.Status != DefinitionActive || definition.CurrentVersion != run.DefinitionVersion ||
		definition.StoredChecksum != run.DefinitionChecksum || definition.Dataset != run.Dataset ||
		definition.ScopeKind != run.ScopeKind || definition.ScopeRef != run.ScopeRef || definition.Format != run.Format {
		return ReportRun{}, ErrConflict
	}
	tag, err := tx.Exec(ctx, `INSERT INTO report_runs (
		id,tenant_id,legal_entity_id,definition_id,definition_version,definition_code,definition_checksum,
		scope_kind,scope_ref,requested_by_ref,as_of,source_boundary,filter,dataset,format,status,attempt_count,
		row_count,created_at,expires_at
	) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9::uuid,$10,$11,$12::jsonb,$13::jsonb,
		$14,$15,$16,0,0,$17,$18)`, run.ID, run.TenantID, run.LegalEntityID, run.DefinitionID,
		run.DefinitionVersion, run.DefinitionCode, run.DefinitionChecksum, run.ScopeKind, nullableText(run.ScopeRef),
		run.RequestedByRef, run.AsOf, boundary, filter, run.Dataset, run.Format, run.Status, run.CreatedAt, run.ExpiresAt)
	if err != nil {
		if isPostgresUniqueViolation(err) {
			return ReportRun{}, ErrConflict
		}
		return ReportRun{}, fmt.Errorf("insert report run: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ReportRun{}, ErrConflict
	}
	if err := insertReportOutbox(ctx, tx, scope.TenantID, "REPORT_RUN", run.ID, "ReportRunQueued", run.CreatedAt, map[string]any{
		"definition_id": run.DefinitionID, "definition_version": run.DefinitionVersion, "dataset": run.Dataset,
		"scope_kind": run.ScopeKind, "scope_ref": run.ScopeRef, "legal_entity_id": scope.LegalEntityID,
	}); err != nil {
		return ReportRun{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReportRun{}, fmt.Errorf("commit report run creation: %w", err)
	}
	return run, nil
}

func (r *PostgresRepository) GetRun(ctx context.Context, scope ReportScope, id string) (ReportRun, error) {
	if err := r.validateInput(ctx, scope, strings.TrimSpace(id)); err != nil {
		return ReportRun{}, err
	}
	run, err := scanRun(r.pool.QueryRow(ctx, `SELECT `+runProjection+`
		FROM report_runs r WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid`,
		scope.TenantID, scope.LegalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportRun{}, ErrNotFound
	}
	if err != nil {
		return ReportRun{}, fmt.Errorf("read report run: %w", err)
	}
	return run, nil
}

func (r *PostgresRepository) ListRuns(ctx context.Context, scope ReportScope, definitionID string, limit int) ([]ReportRun, error) {
	definitionID = strings.TrimSpace(definitionID)
	if definitionID != "" && !isUUID(definitionID) {
		return nil, ErrInvalid
	}
	if err := r.validateInput(ctx, scope, "list"); err != nil {
		return nil, err
	}
	limit = boundedRepositoryLimit(limit, 50)
	rows, err := r.pool.Query(ctx, `SELECT `+runProjection+`
		FROM report_runs r WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid
		  AND ($3='' OR r.definition_id=NULLIF($3,'')::uuid)
		ORDER BY r.created_at DESC,r.id DESC LIMIT $4`, scope.TenantID, scope.LegalEntityID, definitionID, limit)
	if err != nil {
		return nil, fmt.Errorf("list report runs: %w", err)
	}
	defer rows.Close()
	values := make([]ReportRun, 0, limit)
	for rows.Next() {
		value, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan report run list: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list report runs: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) ClaimQueuedRuns(ctx context.Context, scope ReportScope, workerID string, limit int) ([]ReportRun, error) {
	workerID = strings.TrimSpace(workerID)
	if err := r.validateInput(ctx, scope, workerID); err != nil {
		return nil, err
	}
	if workerID == "" || len([]rune(workerID)) > maxClaimWorkerIDRunes || limit <= 0 {
		return nil, ErrInvalid
	}
	if limit > reportClaimBatch {
		limit = reportClaimBatch
	}
	rows, err := r.pool.Query(ctx, `WITH worker_gate AS MATERIALIZED (
		SELECT pg_try_advisory_xact_lock(hashtextextended('clearsight:report-run-claim:'||$1,0)) AS locked
	), candidates AS MATERIALIZED (
		SELECT r.id FROM report_runs r CROSS JOIN worker_gate g
		WHERE g.locked AND r.tenant_id=$2::uuid AND r.legal_entity_id=$3::uuid AND r.status='QUEUED'
		ORDER BY r.created_at,r.id LIMIT $4 FOR UPDATE OF r SKIP LOCKED
	), claimed AS (
		UPDATE report_runs r SET status='RUNNING',attempt_count=r.attempt_count+1
		FROM candidates c WHERE r.id=c.id AND r.status='QUEUED'
		RETURNING r.*
	), events AS (
		INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		SELECT uuidv7(),r.tenant_id,'REPORT_RUN',r.id,'ReportRunClaimed',
			jsonb_build_object('worker_id',$1::text,'attempt_count',r.attempt_count,'dataset',r.dataset,
				'legal_entity_id',r.legal_entity_id::text),clock_timestamp(),clock_timestamp())
		FROM claimed r RETURNING r.aggregate_id
	)
	SELECT `+runProjection+` FROM claimed r
	WHERE EXISTS (SELECT 1 FROM events e WHERE e.aggregate_id=r.id)
	ORDER BY r.created_at,r.id`, workerID, scope.TenantID, scope.LegalEntityID, limit)
	if err != nil {
		return nil, fmt.Errorf("claim queued report runs: %w", err)
	}
	defer rows.Close()
	values := make([]ReportRun, 0, limit)
	for rows.Next() {
		value, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan claimed report run: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("claim queued report runs: %w", err)
	}
	return values, nil
}

func (r *PostgresRepository) CompleteRun(ctx context.Context, scope ReportScope, run ReportRun) (ReportRun, error) {
	if err := r.validateInput(ctx, scope, run.ID); err != nil {
		return ReportRun{}, err
	}
	if run.TenantID != scope.TenantID || run.LegalEntityID != scope.LegalEntityID {
		return ReportRun{}, ErrNotFound
	}
	if err := validateReadyRunForWrite(run); err != nil {
		return ReportRun{}, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReportRun{}, fmt.Errorf("begin report run completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanRun(tx.QueryRow(ctx, `SELECT `+runProjection+`
		FROM report_runs r WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid
		FOR UPDATE`, scope.TenantID, scope.LegalEntityID, run.ID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportRun{}, ErrNotFound
	}
	if err != nil {
		return ReportRun{}, fmt.Errorf("lock report run completion: %w", err)
	}
	if current.Status != RunRunning || current.AttemptCount != run.AttemptCount || current.DefinitionVersion != run.DefinitionVersion ||
		current.DefinitionChecksum != run.DefinitionChecksum {
		return ReportRun{}, ErrConflict
	}
	tag, err := tx.Exec(ctx, `UPDATE report_runs SET status='READY',row_count=$4,data_object_key=$5,
		data_sha256=$6,manifest_object_key=$7,manifest_sha256=$8,completed_at=$9,failure_code=NULL
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='RUNNING' AND attempt_count=$10`,
		scope.TenantID, scope.LegalEntityID, run.ID, run.RowCount, run.DataObjectKey, run.DataSHA256,
		run.ManifestObjectKey, run.ManifestSHA256, run.CompletedAt, run.AttemptCount)
	if err != nil {
		return ReportRun{}, fmt.Errorf("complete report run: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ReportRun{}, ErrConflict
	}
	if err := insertReportOutbox(ctx, tx, scope.TenantID, "REPORT_RUN", run.ID, "ReportRunCompleted", *run.CompletedAt, map[string]any{
		"row_count": run.RowCount, "data_sha256": run.DataSHA256, "legal_entity_id": scope.LegalEntityID,
	}); err != nil {
		return ReportRun{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReportRun{}, fmt.Errorf("commit report run completion: %w", err)
	}
	return run, nil
}

func (r *PostgresRepository) FailRun(ctx context.Context, scope ReportScope, id, failureCode string) (ReportRun, error) {
	id = strings.TrimSpace(id)
	failureCode = strings.ToLower(strings.TrimSpace(failureCode))
	if err := r.validateInput(ctx, scope, id); err != nil {
		return ReportRun{}, err
	}
	if !reportFailureCodePattern.MatchString(failureCode) {
		return ReportRun{}, ErrInvalid
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ReportRun{}, fmt.Errorf("begin report run failure: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := scanRun(tx.QueryRow(ctx, `SELECT `+runProjection+`
		FROM report_runs r WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid
		FOR UPDATE`, scope.TenantID, scope.LegalEntityID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return ReportRun{}, ErrNotFound
	}
	if err != nil {
		return ReportRun{}, fmt.Errorf("lock failed report run: %w", err)
	}
	if current.Status != RunRunning {
		return ReportRun{}, ErrConflict
	}
	completedAt := time.Now().UTC()
	tag, err := tx.Exec(ctx, `UPDATE report_runs SET status='FAILED',failure_code=$4,row_count=0,
		data_object_key=NULL,data_sha256=NULL,manifest_object_key=NULL,manifest_sha256=NULL,completed_at=$5
		WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND status='RUNNING'`,
		scope.TenantID, scope.LegalEntityID, id, failureCode, completedAt)
	if err != nil {
		return ReportRun{}, fmt.Errorf("fail report run: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ReportRun{}, ErrConflict
	}
	if err := insertReportOutbox(ctx, tx, scope.TenantID, "REPORT_RUN", id, "ReportRunFailed", completedAt, map[string]any{
		"failure_code": failureCode, "legal_entity_id": scope.LegalEntityID,
	}); err != nil {
		return ReportRun{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ReportRun{}, fmt.Errorf("commit report run failure: %w", err)
	}
	current.Status = RunFailed
	current.FailureCode = failureCode
	current.RowCount = 0
	current.DataObjectKey = ""
	current.DataSHA256 = ""
	current.ManifestObjectKey = ""
	current.ManifestSHA256 = ""
	current.CompletedAt = &completedAt
	return current, nil
}

func (r *PostgresRepository) RecordRunDownload(ctx context.Context, scope ReportScope, id, downloadedBy string) error {
	id = strings.TrimSpace(id)
	downloadedBy = strings.TrimSpace(downloadedBy)
	if err := r.validateInput(ctx, scope, id); err != nil {
		return err
	}
	if downloadedBy == "" {
		return ErrInvalid
	}
	tag, err := r.pool.Exec(ctx, `INSERT INTO outbox_events
		(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		SELECT uuidv7(),r.tenant_id,'REPORT_RUN',r.id,'ReportRunDownloaded',
			jsonb_build_object('downloaded_by',$3::text,'legal_entity_id',r.legal_entity_id::text),
			clock_timestamp(),clock_timestamp()
		FROM report_runs r WHERE r.tenant_id=$1::uuid AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid
		  AND r.status='READY' AND r.expires_at>clock_timestamp()`, scope.TenantID, scope.LegalEntityID, id, downloadedBy)
	if err != nil {
		return fmt.Errorf("record report run download: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CaptureSourceBoundary(ctx context.Context, scope ReportScope, definition ReportDefinition) (SourceBoundary, error) {
	if err := r.validateInput(ctx, scope, definition.ID); err != nil {
		return SourceBoundary{}, err
	}
	if definition.TenantID != scope.TenantID || definition.LegalEntityID != scope.LegalEntityID {
		return SourceBoundary{}, ErrNotFound
	}
	if !validReportDataset(definition.Dataset) || !validReportScope(definition.ScopeKind, definition.ScopeRef) {
		return SourceBoundary{}, ErrInvalid
	}
	_, predicate, ok := reportDatasetFragments(definition.Dataset)
	if !ok {
		return SourceBoundary{}, ErrInvalid
	}
	filter, filterArgs, err := ReportFilterSQL(definition.Filter, 5)
	if err != nil {
		return SourceBoundary{}, err
	}
	var generatedAt time.Time
	var projectionVersion string
	var sourceHighWater time.Time
	if err := r.pool.QueryRow(ctx, `SELECT generated_at,projection_version,source_high_water
		FROM ropa_register_summary WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid`, scope.TenantID, scope.LegalEntityID).
		Scan(&generatedAt, &projectionVersion, &sourceHighWater); errors.Is(err, pgx.ErrNoRows) {
		return SourceBoundary{}, ErrNotFound
	} else if err != nil {
		return SourceBoundary{}, fmt.Errorf("read report source projection boundary: %w", err)
	}
	arguments := []any{scope.TenantID, scope.LegalEntityID, nullableText(definition.ScopeRef), sourceHighWater}
	arguments = append(arguments, filterArgs...)
	var population int
	if err := r.pool.QueryRow(ctx, `SELECT count(*)::integer FROM ropa_processing_activities a
		WHERE a.tenant_id=$1::uuid AND a.legal_entity_id=$2::uuid
		  AND ($3='' OR a.program_id=$3::uuid OR a.matter_id=$3::uuid)
		  AND a.updated_at<=$4::timestamptz AND `+predicate+` AND (`+filter+`)`, arguments...).Scan(&population); err != nil {
		return SourceBoundary{}, fmt.Errorf("count report source population: %w", err)
	}
	return SourceBoundary{
		CapturedAt: generatedAt.UTC(), ProjectionVersion: projectionVersion,
		SourceHighWater: map[string]time.Time{"processing_activities": sourceHighWater.UTC()},
		Population:      population, PopulationComplete: true,
	}, nil
}

func (r *PostgresRepository) validateInput(ctx context.Context, scope ReportScope, id string) error {
	if r == nil || r.pool == nil || ctx == nil {
		return ErrInvalid
	}
	if err := validateReportScope(scope); err != nil {
		return err
	}
	if id == "" {
		return ErrInvalid
	}
	return nil
}

func scanDefinition(row reportingRowScanner) (ReportDefinition, error) {
	var value ReportDefinition
	var filter []byte
	if err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.Code, &value.Name, &value.Description,
		&value.Dataset, &value.ScopeKind, &value.ScopeRef, &value.Format, &filter, &value.Status,
		&value.CurrentVersion, &value.StoredChecksum, &value.MakerID, &value.CheckerID, &value.ReviewerID,
		&value.ReviewerNote, &value.EffectiveFrom, &value.EffectiveUntil, &value.SubmittedAt, &value.ApprovedAt,
		&value.RetiredAt, &value.CreatedAt, &value.UpdatedAt, &value.Version); err != nil {
		return ReportDefinition{}, err
	}
	if err := json.Unmarshal(filter, &value.Filter); err != nil {
		return ReportDefinition{}, fmt.Errorf("decode report definition filter: %w", err)
	}
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	return value, nil
}

func scanDefinitionRevision(row reportingRowScanner) (ReportDefinitionRevision, error) {
	var value ReportDefinitionRevision
	var filter []byte
	if err := row.Scan(&value.DefinitionID, &value.TenantID, &value.LegalEntityID, &value.Version, &value.BaseVersion,
		&value.Dataset, &value.ScopeKind, &value.ScopeRef, &value.Format, &filter, &value.Checksum, &value.MakerID,
		&value.CreatedAt, &value.ReviewedBy, &value.ReviewedAt, &value.ApprovedBy, &value.ApprovedAt,
		&value.Decision, &value.DecisionNote); err != nil {
		return ReportDefinitionRevision{}, err
	}
	if err := json.Unmarshal(filter, &value.Filter); err != nil {
		return ReportDefinitionRevision{}, fmt.Errorf("decode report definition revision filter: %w", err)
	}
	return value, nil
}

func scanRun(row reportingRowScanner) (ReportRun, error) {
	var value ReportRun
	var boundary, filter []byte
	if err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.DefinitionID,
		&value.DefinitionVersion, &value.DefinitionCode, &value.DefinitionChecksum, &value.ScopeKind, &value.ScopeRef,
		&value.RequestedByRef, &value.AsOf, &boundary, &filter, &value.Dataset, &value.Format, &value.Status,
		&value.AttemptCount, &value.RowCount, &value.DataObjectKey, &value.DataSHA256, &value.ManifestObjectKey,
		&value.ManifestSHA256, &value.FailureCode, &value.CreatedAt, &value.CompletedAt, &value.ExpiresAt); err != nil {
		return ReportRun{}, err
	}
	if err := json.Unmarshal(boundary, &value.SourceBoundary); err != nil {
		return ReportRun{}, fmt.Errorf("decode report source boundary: %w", err)
	}
	if err := json.Unmarshal(filter, &value.Filter); err != nil {
		return ReportRun{}, fmt.Errorf("decode report run filter: %w", err)
	}
	value.AsOf = value.AsOf.UTC()
	value.CreatedAt = value.CreatedAt.UTC()
	value.ExpiresAt = value.ExpiresAt.UTC()
	return value, nil
}

func validateDefinitionRevision(revision ReportDefinitionRevision, definition ReportDefinition) error {
	if revision.DefinitionID != definition.ID || revision.TenantID != definition.TenantID || revision.LegalEntityID != definition.LegalEntityID ||
		revision.Version != definition.CurrentVersion || revision.BaseVersion != 0 || revision.Dataset != definition.Dataset ||
		revision.ScopeKind != definition.ScopeKind || revision.ScopeRef != definition.ScopeRef || revision.Format != definition.Format ||
		revision.Checksum != definition.StoredChecksum || revision.MakerID != definition.MakerID || revision.Decision != "PROPOSED" {
		return ErrInvalid
	}
	return nil
}

func validateDefinitionDecision(current ReportDefinition, next DefinitionStatus, decision DecisionRecord) error {
	expected := DefinitionStatus("")
	switch decision.Action {
	case DecisionSubmit:
		expected = DefinitionPendingReview
	case DecisionReview:
		expected = DefinitionReviewed
	case DecisionActivate:
		expected = DefinitionActive
	case DecisionReject, DecisionRetire:
		expected = DefinitionRetired
	default:
		return ErrInvalid
	}
	if expected != next || decision.Timestamp.Before(current.UpdatedAt) {
		return ErrInvalid
	}
	if decision.Action == DecisionSubmit && decision.ActorID != current.MakerID {
		return ErrClosureBlocked
	}
	if (decision.Action == DecisionReview || decision.Action == DecisionReject) && decision.ActorID == current.MakerID {
		return ErrClosureBlocked
	}
	if decision.Action == DecisionActivate && (decision.ActorID == current.MakerID || decision.ActorID == current.ReviewerID) {
		return ErrClosureBlocked
	}
	return nil
}

func applyDefinitionDecision(current ReportDefinition, next DefinitionStatus, decision DecisionRecord) ReportDefinition {
	result := current
	result.Status = next
	switch decision.Action {
	case DecisionSubmit:
		result.SubmittedAt = timePtrCopy(decision.Timestamp)
	case DecisionReview:
		result.ReviewerID = decision.ActorID
		result.ReviewerNote = decision.Note
	case DecisionActivate:
		result.CheckerID = decision.ActorID
		result.ApprovedAt = timePtrCopy(decision.Timestamp)
		if decision.EffectiveFrom == nil {
			result.EffectiveFrom = timePtrCopy(decision.Timestamp)
		} else {
			result.EffectiveFrom = timePtrCopy(decision.EffectiveFrom.UTC())
		}
	case DecisionReject, DecisionRetire:
		result.RetiredAt = timePtrCopy(decision.Timestamp)
	}
	return result
}

func revisionDecisionSQL(next DefinitionStatus, decision DecisionRecord) string {
	switch decision.Action {
	case DecisionSubmit:
		return `UPDATE report_definition_revisions SET decision_note=$5
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND definition_id=$3::uuid AND version=$4`
	case DecisionReview:
		return `UPDATE report_definition_revisions SET decision='REVIEWED',reviewed_by=$5::uuid,
			reviewed_at=$6,decision_note=$7
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND definition_id=$3::uuid AND version=$4`
	case DecisionActivate:
		return `UPDATE report_definition_revisions SET decision='APPROVED',approved_by=$5::uuid,
			approved_at=$6,decision_note=$7
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND definition_id=$3::uuid AND version=$4`
	case DecisionReject:
		return `UPDATE report_definition_revisions SET decision='REJECTED',
			reviewed_by=COALESCE(reviewed_by,$5::uuid),reviewed_at=COALESCE(reviewed_at,$6),decision_note=$7
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND definition_id=$3::uuid AND version=$4`
	case DecisionRetire:
		return `UPDATE report_definition_revisions SET decision='RETIRED',decision_note=$5
			WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND definition_id=$3::uuid AND version=$4`
	default:
		return `SELECT 1 WHERE false`
	}
}

func revisionDecisionArguments(definition ReportDefinition, next DefinitionStatus, decision DecisionRecord) []any {
	switch decision.Action {
	case DecisionSubmit, DecisionRetire:
		return []any{definition.TenantID, definition.LegalEntityID, definition.ID, definition.CurrentVersion, decision.Note}
	case DecisionReview, DecisionActivate, DecisionReject:
		return []any{definition.TenantID, definition.LegalEntityID, definition.ID, definition.CurrentVersion, decision.ActorID, decision.Timestamp, decision.Note}
	default:
		return nil
	}
}

func insertReportOutbox(ctx context.Context, tx pgx.Tx, tenantID, aggregateType, aggregateID, eventType string, occurredAt time.Time, payload map[string]any) error {
	id, err := platformid.NewUUIDv7()
	if err != nil {
		return fmt.Errorf("create report outbox id: %w", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode report outbox payload: %w", err)
	}
	tag, err := tx.Exec(ctx, `INSERT INTO outbox_events
		(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at)
		VALUES($1::uuid,$2::uuid,$3,$4::uuid,$5,$6::jsonb,$7,$7)`, id, tenantID, aggregateType, aggregateID, eventType, body, occurredAt.UTC())
	if err != nil {
		return fmt.Errorf("insert report outbox event: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	return nil
}

func isPostgresUniqueViolation(err error) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) && postgresError.Code == "23505"
}

func nullableText(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func timePtrCopy(value time.Time) *time.Time {
	result := value.UTC()
	return &result
}

func boundedRepositoryLimit(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value > maxRepositoryListRows {
		return maxRepositoryListRows
	}
	return value
}

var _ Repository = (*PostgresRepository)(nil)
