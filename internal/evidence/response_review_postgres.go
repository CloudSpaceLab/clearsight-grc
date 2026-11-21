//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

// responseReviewTenant verifies the supplied scope resolves to exactly one tenant.
// Its result is used only to adapt a scoped record for existing route callbacks;
// stored records and the atomic decision retain the canonical UUID.
func responseReviewTenant(ctx context.Context, tx pgx.Tx, tenant string) (string, error) {
	var canonical *string
	if err := tx.QueryRow(ctx, `SELECT CASE WHEN count(*)=1 THEN min(id::text) END FROM tenants WHERE id::text=$1 OR slug=$1`, tenant).Scan(&canonical); err != nil {
		return "", err
	}
	if canonical == nil || *canonical == "" {
		return "", ErrAssessmentForbidden
	}
	return *canonical, nil
}

func responseReviewAuthorityMaterial(m assessmentMaterial, canonical, tenant string) (assessmentMaterial, bool) {
	if m.Summary.TenantID != canonical {
		return assessmentMaterial{}, false
	}
	m.Summary.TenantID = tenant
	return m, true
}

func responseReviewScopedAuthorizer(authorize assessmentReadAuthorizer, canonical, tenant string) assessmentReadAuthorizer {
	return func(ctx context.Context, m assessmentMaterial) bool {
		scoped, ok := responseReviewAuthorityMaterial(m, canonical, tenant)
		return ok && authorize != nil && authorize(ctx, scoped)
	}
}

func (s *PostgresDistributionStore) ReadResponseReviews(ctx context.Context, tenant, entity, principal, response string, authorize assessmentReadAuthorizer, before int64, limit int) (assessmentMaterial, []ResponseReview, int64, error) {
	tx, err := s.repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	canonical, err := responseReviewTenant(ctx, tx, tenant)
	if err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	m, err := readResponseAssessmentMaterial(ctx, tx, tenant, entity, principal, response, responseReviewScopedAuthorizer(authorize, canonical, tenant), false)
	if err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	var latest int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max(version),0) FROM capture_response_reviews WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND response_revision_id=$3::uuid`, m.Summary.TenantID, entity, response).Scan(&latest); err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	rows, err := tx.Query(ctx, `SELECT id::text,response_revision_id::text,response_revision,version,decision,comment,reviewer_id::text,authority_route,reviewed_at,COALESCE(supersedes_id::text,''),COALESCE(authority_source_type,''),COALESCE(authority_source_id::text,''),COALESCE(authority_rule_id,''),COALESCE(authority_policy_version,'') FROM capture_response_reviews WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND response_revision_id=$3::uuid AND ($4=0 OR version<$4) ORDER BY version DESC LIMIT $5`, m.Summary.TenantID, entity, response, before, limit)
	if err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	items := make([]ResponseReview, 0, limit)
	for rows.Next() {
		var v ResponseReview
		if err = rows.Scan(&v.ID, &v.ResponseID, &v.ResponseRevision, &v.Version, &v.Decision, &v.Comment, &v.ReviewerID, &v.AuthorityRoute, &v.ReviewedAt, &v.SupersedesID, &v.AuthoritySourceType, &v.AuthoritySourceID, &v.AuthorityRuleID, &v.AuthorityPolicyVersion); err != nil {
			rows.Close()
			return assessmentMaterial{}, nil, 0, err
		}
		items = append(items, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return assessmentMaterial{}, nil, 0, err
	}
	return m, items, latest, nil
}

func (s *PostgresDistributionStore) WriteResponseReview(ctx context.Context, tenant, entity, principal, response string, input RecordResponseReviewInput, authorize assessmentReadAuthorizer, apply func(context.Context, assessmentMaterial, *ResponseReview) (ResponseReview, error)) (ResponseReview, error) {
	actor, ok := identity.FromContext(ctx)
	if !ok || actor.TenantID != tenant || actor.PrincipalID != principal || (actor.LegalEntityID != entity && actor.LegalEntityID != "*") || actor.ExpiresAt.IsZero() {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	tx, err := s.repo.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return ResponseReview{}, err
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	canonical, err := responseReviewTenant(ctx, tx, tenant)
	if err != nil {
		return ResponseReview{}, err
	}
	if input.RequestKey != "" {
		_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, canonical+":"+entity+":"+principal+":"+input.RequestKey)
		if err != nil {
			return ResponseReview{}, err
		}
	}
	var current bool
	err = tx.QueryRow(ctx, `SELECT r.is_current AND d.status NOT IN ('REVOKED','SUPERSEDED') FROM capture_response_revisions r JOIN tenants t ON t.id=r.tenant_id JOIN capture_form_distributions d ON d.id=r.distribution_id AND d.tenant_id=r.tenant_id AND d.legal_entity_id=r.legal_entity_id `+completedResponseRequestJoinsSQL()+` WHERE (t.id::text=$1 OR t.slug=$1) AND r.legal_entity_id=$2::uuid AND r.id=$3::uuid AND ((`+completedResponseVisibilitySQL(4, 5)+`) OR (`+vendorAssessmentCandidateSQL()+`)) FOR UPDATE OF r,d`, tenant, entity, response, principal, time.Now().UTC()).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseReview{}, ErrNotFound
	}
	if err != nil {
		return ResponseReview{}, err
	}
	m, err := readResponseAssessmentMaterial(ctx, tx, tenant, entity, principal, response, responseReviewScopedAuthorizer(authorize, canonical, tenant), true)
	if err != nil {
		return ResponseReview{}, err
	}
	if input.RequestKey != "" {
		var exists bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM capture_response_reviews WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND reviewer_id=$3::uuid AND request_key=$4::uuid)`, canonical, entity, principal, input.RequestKey).Scan(&exists)
		if err != nil {
			return ResponseReview{}, err
		}
		if exists {
			return replayResponseReview(ctx, tx, actor, m, input)
		}
	}
	if !current {
		return ResponseReview{}, ErrAssessmentConflict
	}
	var prev ResponseReview
	err = tx.QueryRow(ctx, `SELECT id::text,response_revision_id::text,response_revision,version,decision,comment,reviewer_id::text,authority_route,reviewed_at,COALESCE(supersedes_id::text,'') FROM capture_response_reviews WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND response_revision_id=$3::uuid ORDER BY version DESC LIMIT 1`, m.Summary.TenantID, m.Summary.LegalEntityID, response).Scan(&prev.ID, &prev.ResponseID, &prev.ResponseRevision, &prev.Version, &prev.Decision, &prev.Comment, &prev.ReviewerID, &prev.AuthorityRoute, &prev.ReviewedAt, &prev.SupersedesID)
	var previous *ResponseReview
	if err == nil {
		previous = &prev
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return ResponseReview{}, err
	}
	scoped, ok := responseReviewAuthorityMaterial(m, canonical, tenant)
	if !ok {
		return ResponseReview{}, ErrAssessmentForbidden
	}
	review, err := apply(ctx, scoped, previous)
	if err != nil {
		return ResponseReview{}, err
	}
	review, err = insertCurrentResponseReview(ctx, tx, actor, m, review, input.RequestKey)
	if err != nil {
		return ResponseReview{}, err
	}
	event := responseReviewEvent(m, review)
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return ResponseReview{}, err
	}
	eventType := fmt.Sprintf("FORM_RESPONSE_REVIEWED_%s_%d", response, review.Version)
	_, err = tx.Exec(ctx, `INSERT INTO capture_distribution_events(tenant_id,legal_entity_id,distribution_id,distribution_version,event_type,payload,actor_id,occurred_at) SELECT $1::uuid,$2::uuid,d.id,d.version,$4,$5::jsonb,$6::uuid,$7 FROM capture_form_distributions d WHERE d.id=$3::uuid AND d.tenant_id=$1::uuid`, m.Summary.TenantID, m.Summary.LegalEntityID, m.Revision.DistributionID, eventType, string(payload), principal, review.ReviewedAt)
	if err != nil {
		return ResponseReview{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at,next_attempt_at) VALUES($1::uuid,'FORM_RESPONSE_REVIEW',$2::uuid,'FORM_RESPONSE_REVIEWED',$3::jsonb,$4,$4,$4)`, m.Summary.TenantID, response, string(payload), review.ReviewedAt)
	if err != nil {
		return ResponseReview{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ResponseReview{}, err
	}
	return review, nil
}
