package sourceaccess

import (
	"context"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgreSQL reservations serialize per legal entity across application
// instances. Only identifiers and outcomes are retained; payloads are never
// supplied to this store.
type PostgresQuestionTestAudit struct{ pool *pgxpool.Pool }

func NewPostgresQuestionTestAudit(pool *pgxpool.Pool) *PostgresQuestionTestAudit {
	return &PostgresQuestionTestAudit{pool: pool}
}

func (s *PostgresQuestionTestAudit) Reserve(ctx context.Context, actor QuestionTestActor, formID string, version int64, fieldID, mode string, when time.Time) (string, error) {
	if s == nil || s.pool == nil || !validQuestionTestAudit(actor, formID, version, fieldID, mode, when) {
		return "", ErrDefinitionInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	// Resolve aliases before taking the rate-limit lock, so slug and UUID
	// identities share the same tenant-wide quota.
	var tenantID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM tenants WHERE id::text=$1 OR slug=$1`, actor.TenantID).Scan(&tenantID); err != nil {
		return "", ErrDefinitionInvalid
	}
	// Database clock controls the window, preventing drift across workers.
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "api-question-test:"+tenantID+":"+actor.LegalEntityID)
	if err != nil {
		return "", err
	}
	var entity, principal int
	err = tx.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE a.principal_id::text=$3)
	 FROM api_question_test_attempts a JOIN tenants t ON t.id=a.tenant_id
	 WHERE t.id::text=$1 AND a.legal_entity_id::text=$2 AND a.attempted_at > clock_timestamp()-interval '1 minute'`, tenantID, actor.LegalEntityID, actor.PrincipalID).Scan(&entity, &principal)
	if err != nil {
		return "", err
	}
	if entity >= 60 || principal >= 10 {
		return "", ErrQuestionTestRateLimit
	}
	attempt, err := id.NewUUIDv7()
	if err != nil {
		return "", err
	}
	result, err := tx.Exec(ctx, `INSERT INTO api_question_test_attempts(id,tenant_id,legal_entity_id,principal_id,form_id,form_version,field_id,mode,authority_rule_id,authority_policy_version)
	 SELECT $1::uuid,t.id,e.id,p.id,$5::uuid,$6,$7,$8,$9,$10 FROM tenants t JOIN legal_entities e ON e.tenant_id=t.id AND e.id::text=$3 JOIN principals p ON p.tenant_id=t.id AND p.id::text=$4 WHERE t.id::text=$2`, attempt, tenantID, actor.LegalEntityID, actor.PrincipalID, formID, version, fieldID, mode, actor.AuthorityRuleID, actor.AuthorityPolicyVersion)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() != 1 {
		return "", ErrDefinitionInvalid
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return attempt, nil
}
func (s *PostgresQuestionTestAudit) Complete(ctx context.Context, actor QuestionTestActor, attemptID string, outcome QuestionTestOutcome, when time.Time) error {
	if s == nil || s.pool == nil || when.IsZero() || (outcome != QuestionTestValid && outcome != QuestionTestInvalid && outcome != QuestionTestUnavailable) {
		return ErrDefinitionInvalid
	}
	result, err := s.pool.Exec(ctx, `INSERT INTO api_question_test_outcomes(attempt_id,outcome)
	 SELECT a.id,$5 FROM api_question_test_attempts a JOIN tenants t ON t.id=a.tenant_id WHERE a.id::text=$1 AND (t.id::text=$2 OR t.slug=$2) AND a.legal_entity_id::text=$3 AND a.principal_id::text=$4`, attemptID, actor.TenantID, actor.LegalEntityID, actor.PrincipalID, outcome)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrDefinitionInvalid
	}
	return nil
}
