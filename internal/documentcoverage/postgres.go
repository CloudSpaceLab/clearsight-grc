//go:build postgres

package documentcoverage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ pool *pgxpool.Pool }

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) BeginVersion(ctx context.Context, value Assessment) (Assessment, error) {
	metrics, _ := json.Marshal(value.Metrics)
	limitations, _ := json.Marshal(value.Limitations)
	row := r.pool.QueryRow(ctx, `
		INSERT INTO document_coverage_assessments(
			id,tenant_id,legal_entity_id,document_id,document_sha256,status,analyzer_version,matcher_version,
			scoring_policy_version,program_snapshot_hash,metrics,limitations,failure_message,assessed_at,created_at,updated_at,version)
		SELECT $2::uuid,t.id,
		       CASE WHEN $3='' THEN NULL ELSE (SELECT le.id FROM legal_entities le WHERE le.tenant_id=t.id AND le.id=$3::uuid) END,
		       $4::uuid,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb,$13,$14,$15,$16,1
		FROM tenants t WHERE t.id::text=$1 OR t.slug=$1
		ON CONFLICT (tenant_id,document_id,document_sha256,analyzer_version,matcher_version,program_snapshot_hash)
		DO UPDATE SET status=CASE WHEN document_coverage_assessments.status IN ('READY','PARTIAL') THEN document_coverage_assessments.status ELSE 'COMPARING' END,
		              updated_at=EXCLUDED.updated_at
		RETURNING id::text,(SELECT slug FROM tenants WHERE id=tenant_id),COALESCE(legal_entity_id::text,''),document_id::text,
		          document_sha256,status,analyzer_version,matcher_version,scoring_policy_version,program_snapshot_hash,
		          metrics,limitations,failure_message,assessed_at,created_at,updated_at,version`,
		value.TenantID, value.ID, value.LegalEntityID, value.DocumentID, value.DocumentSHA256, value.Status,
		value.AnalyzerVersion, value.MatcherVersion, value.ScoringPolicyVersion, value.ProgramSnapshotHash,
		metrics, limitations, value.FailureMessage, value.AssessedAt, value.CreatedAt, value.UpdatedAt)
	started, err := scanAssessmentBase(row)
	if err != nil {
		return Assessment{}, fmt.Errorf("begin document coverage assessment: %w", err)
	}
	return r.loadChildren(ctx, started)
}

func (r *PostgresRepository) CompleteVersion(ctx context.Context, value Assessment, expected int64) (Assessment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Assessment{}, err
	}
	defer tx.Rollback(ctx)
	var version int64
	if err := tx.QueryRow(ctx, `SELECT version FROM document_coverage_assessments WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) FOR UPDATE`, value.ID, value.TenantID).Scan(&version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Assessment{}, ErrNotFound
		}
		return Assessment{}, err
	}
	if version != expected {
		return Assessment{}, ErrVersionConflict
	}
	if _, err := tx.Exec(ctx, `DELETE FROM document_coverage_candidates WHERE assessment_id=$1::uuid`, value.ID); err != nil {
		return Assessment{}, err
	}
	if err := insertAssessmentChildren(ctx, tx, value); err != nil {
		return Assessment{}, err
	}
	metrics, _ := json.Marshal(value.Metrics)
	limitations, _ := json.Marshal(value.Limitations)
	if _, err := tx.Exec(ctx, `
		UPDATE document_coverage_assessments SET status=$3,metrics=$4::jsonb,limitations=$5::jsonb,failure_message=$6,
		       assessed_at=$7,updated_at=$8
		WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND version=$9`,
		value.ID, value.TenantID, value.Status, metrics, limitations, value.FailureMessage, value.AssessedAt, value.UpdatedAt, expected); err != nil {
		return Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Assessment{}, err
	}
	return r.Current(ctx, value.TenantID, value.DocumentID)
}

func (r *PostgresRepository) Current(ctx context.Context, tenant, documentID string) (Assessment, error) {
	row := r.pool.QueryRow(ctx, assessmentSelect+`
		WHERE (t.id::text=$1 OR t.slug=$1) AND a.document_id=$2::uuid
		ORDER BY a.created_at DESC,a.id DESC LIMIT 1`, tenant, documentID)
	value, err := scanAssessmentBase(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Assessment{}, ErrNotFound
	}
	if err != nil {
		return Assessment{}, fmt.Errorf("load document coverage assessment: %w", err)
	}
	return r.loadChildren(ctx, value)
}

func (r *PostgresRepository) Review(ctx context.Context, value Assessment, expected int64) (Assessment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Assessment{}, err
	}
	defer tx.Rollback(ctx)
	var version int64
	if err := tx.QueryRow(ctx, `SELECT version FROM document_coverage_assessments WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) FOR UPDATE`, value.ID, value.TenantID).Scan(&version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Assessment{}, ErrNotFound
		}
		return Assessment{}, err
	}
	if version != expected {
		return Assessment{}, ErrVersionConflict
	}
	for _, candidate := range value.Candidates {
		payload, _ := json.Marshal(candidate)
		if _, err := tx.Exec(ctx, `UPDATE document_coverage_candidates SET classification=$3,candidate=$4::jsonb WHERE assessment_id=$1::uuid AND candidate_id=$2`, value.ID, candidate.ID, candidate.Classification, payload); err != nil {
			return Assessment{}, err
		}
	}
	var existingReviews int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM document_coverage_reviews WHERE assessment_id=$1::uuid`, value.ID).Scan(&existingReviews); err != nil {
		return Assessment{}, err
	}
	if existingReviews > len(value.Reviews) {
		return Assessment{}, fmt.Errorf("stored coverage review history is longer than the submitted assessment")
	}
	for _, review := range value.Reviews[existingReviews:] {
		if _, err := tx.Exec(ctx, `
			INSERT INTO document_coverage_reviews(assessment_id,tenant_id,candidate_id,decision,match_id,reason,reviewer_id,reviewed_at)
			SELECT $1::uuid,a.tenant_id,$2,$3,NULLIF($4,''),$5,$6::uuid,$7 FROM document_coverage_assessments a WHERE a.id=$1::uuid`,
			value.ID, review.CandidateID, review.Decision, review.MatchID, review.Reason, review.ReviewerID, review.ReviewedAt); err != nil {
			return Assessment{}, err
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM document_coverage_suggestions WHERE assessment_id=$1::uuid`, value.ID); err != nil {
		return Assessment{}, err
	}
	if err := insertSuggestions(ctx, tx, value); err != nil {
		return Assessment{}, err
	}
	metrics, _ := json.Marshal(value.Metrics)
	if _, err := tx.Exec(ctx, `UPDATE document_coverage_assessments SET metrics=$3::jsonb,updated_at=$4,version=version+1 WHERE id=$1::uuid AND tenant_id=(SELECT id FROM tenants WHERE id::text=$2 OR slug=$2) AND version=$5`, value.ID, value.TenantID, metrics, value.UpdatedAt, expected); err != nil {
		return Assessment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Assessment{}, err
	}
	return r.Current(ctx, value.TenantID, value.DocumentID)
}

func (r *PostgresRepository) MarkFailed(ctx context.Context, value Assessment, expected int64) (Assessment, error) {
	value.Status = AssessmentFailed
	return r.CompleteVersion(ctx, value, expected)
}

func (r *PostgresRepository) QueueRecompare(ctx context.Context, tenant, documentID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO outbox_events(id,tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at)
		SELECT uuidv7(),t.id,'DOCUMENT_IMPORT',$2::uuid,$3,'{}'::jsonb,clock_timestamp(),clock_timestamp(),clock_timestamp()
		FROM tenants t WHERE (t.id::text=$1 OR t.slug=$1)
		  AND NOT EXISTS (
		      SELECT 1 FROM outbox_events o WHERE o.tenant_id=t.id AND o.aggregate_type='DOCUMENT_IMPORT'
		        AND o.aggregate_id=$2::uuid AND o.event_type=$3 AND o.published_at IS NULL AND o.dead_lettered_at IS NULL
		  )`, tenant, documentID, EventCoverageComparisonRequested)
	if err != nil {
		return fmt.Errorf("queue document coverage comparison: %w", err)
	}
	return nil
}

const assessmentSelect = `
	SELECT a.id::text,t.slug,COALESCE(a.legal_entity_id::text,''),a.document_id::text,a.document_sha256,a.status,
	       a.analyzer_version,a.matcher_version,a.scoring_policy_version,a.program_snapshot_hash,a.metrics,a.limitations,
	       a.failure_message,a.assessed_at,a.created_at,a.updated_at,a.version
	FROM document_coverage_assessments a JOIN tenants t ON t.id=a.tenant_id`

type assessmentScanner interface{ Scan(...any) error }

func scanAssessmentBase(row assessmentScanner) (Assessment, error) {
	var value Assessment
	var metrics, limitations []byte
	err := row.Scan(&value.ID, &value.TenantID, &value.LegalEntityID, &value.DocumentID, &value.DocumentSHA256, &value.Status,
		&value.AnalyzerVersion, &value.MatcherVersion, &value.ScoringPolicyVersion, &value.ProgramSnapshotHash,
		&metrics, &limitations, &value.FailureMessage, &value.AssessedAt, &value.CreatedAt, &value.UpdatedAt, &value.Version)
	if err != nil {
		return Assessment{}, err
	}
	if err := json.Unmarshal(metrics, &value.Metrics); err != nil {
		return Assessment{}, err
	}
	if err := json.Unmarshal(limitations, &value.Limitations); err != nil {
		return Assessment{}, err
	}
	value.Candidates = []Candidate{}
	value.Reviews = []ReviewRecord{}
	value.Suggestions = []Suggestion{}
	return value, nil
}

func (r *PostgresRepository) loadChildren(ctx context.Context, value Assessment) (Assessment, error) {
	rows, err := r.pool.Query(ctx, `SELECT candidate FROM document_coverage_candidates WHERE assessment_id=$1::uuid ORDER BY ordinal,candidate_id`, value.ID)
	if err != nil {
		return Assessment{}, err
	}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return Assessment{}, err
		}
		var candidate Candidate
		if err := json.Unmarshal(raw, &candidate); err != nil {
			rows.Close()
			return Assessment{}, err
		}
		value.Candidates = append(value.Candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return Assessment{}, err
	}
	rows.Close()

	reviewRows, err := r.pool.Query(ctx, `SELECT candidate_id,decision,COALESCE(match_id,''),reason,reviewer_id::text,reviewed_at FROM document_coverage_reviews WHERE assessment_id=$1::uuid ORDER BY reviewed_at,id`, value.ID)
	if err != nil {
		return Assessment{}, err
	}
	for reviewRows.Next() {
		var review ReviewRecord
		if err := reviewRows.Scan(&review.CandidateID, &review.Decision, &review.MatchID, &review.Reason, &review.ReviewerID, &review.ReviewedAt); err != nil {
			reviewRows.Close()
			return Assessment{}, err
		}
		value.Reviews = append(value.Reviews, review)
	}
	if err := reviewRows.Err(); err != nil {
		reviewRows.Close()
		return Assessment{}, err
	}
	reviewRows.Close()

	suggestionRows, err := r.pool.Query(ctx, `SELECT suggestion FROM document_coverage_suggestions WHERE assessment_id=$1::uuid ORDER BY candidate_id,suggestion_id`, value.ID)
	if err != nil {
		return Assessment{}, err
	}
	for suggestionRows.Next() {
		var raw []byte
		if err := suggestionRows.Scan(&raw); err != nil {
			suggestionRows.Close()
			return Assessment{}, err
		}
		var suggestion Suggestion
		if err := json.Unmarshal(raw, &suggestion); err != nil {
			suggestionRows.Close()
			return Assessment{}, err
		}
		value.Suggestions = append(value.Suggestions, suggestion)
	}
	if err := suggestionRows.Err(); err != nil {
		suggestionRows.Close()
		return Assessment{}, err
	}
	suggestionRows.Close()
	return value, nil
}

type coverageExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertAssessmentChildren(ctx context.Context, tx coverageExecer, value Assessment) error {
	for candidateOrdinal, candidate := range value.Candidates {
		payload, _ := json.Marshal(candidate)
		if _, err := tx.Exec(ctx, `
			INSERT INTO document_coverage_candidates(assessment_id,tenant_id,candidate_id,ordinal,fingerprint,eligible,classification,candidate)
			SELECT $1::uuid,a.tenant_id,$2,$3,$4,$5,$6,$7::jsonb FROM document_coverage_assessments a WHERE a.id=$1::uuid`,
			value.ID, candidate.ID, candidateOrdinal, candidate.Fingerprint, candidate.Eligible, candidate.Classification, payload); err != nil {
			return err
		}
		for matchOrdinal, match := range candidate.Matches {
			matchPayload, _ := json.Marshal(match)
			if _, err := tx.Exec(ctx, `
				INSERT INTO document_coverage_matches(assessment_id,tenant_id,candidate_id,match_id,ordinal,target_program_id,target_requirement_id,score,match)
				SELECT $1::uuid,a.tenant_id,$2,$3,$4,$5::uuid,$6::uuid,$7,$8::jsonb FROM document_coverage_assessments a WHERE a.id=$1::uuid`,
				value.ID, candidate.ID, match.ID, matchOrdinal, match.ProgramID, match.RequirementID, match.Score, matchPayload); err != nil {
				return err
			}
		}
	}
	return insertSuggestions(ctx, tx, value)
}

func insertSuggestions(ctx context.Context, tx coverageExecer, value Assessment) error {
	for _, suggestion := range value.Suggestions {
		payload, _ := json.Marshal(suggestion)
		if _, err := tx.Exec(ctx, `
			INSERT INTO document_coverage_suggestions(assessment_id,tenant_id,suggestion_id,candidate_id,suggestion_type,status,suggestion,applied_type,applied_id,failure_message)
			SELECT $1::uuid,a.tenant_id,$2,$3,$4,$5,$6::jsonb,$7,NULLIF($8,'')::uuid,$9 FROM document_coverage_assessments a WHERE a.id=$1::uuid`,
			value.ID, suggestion.ID, suggestion.CandidateID, suggestion.Type, suggestion.Status, payload,
			suggestion.AppliedType, suggestion.AppliedID, suggestion.FailureMessage); err != nil {
			return err
		}
	}
	return nil
}

var _ Repository = (*PostgresRepository)(nil)
