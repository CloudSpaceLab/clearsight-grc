//go:build postgres

package evidence

import (
	"context"
	"errors"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) ProgramSubmissionReview(ctx context.Context, actor identity.Actor, programID, reviewID string) (SubmissionReview, error) {
	value, err := scanSubmissionReview(r.pool.QueryRow(ctx, `SELECT `+submissionReviewColumns+`
 FROM capture_submission_reviews review
 JOIN capture_requests er ON er.id=review.request_id AND er.tenant_id=review.tenant_id AND er.legal_entity_id=review.legal_entity_id
 JOIN tenants t ON t.id=er.tenant_id
 WHERE `+programAssessmentReadScope+` AND review.id::text=$5`, actor.TenantID, actor.LegalEntityID, programID, actor.PrincipalID, reviewID))
	if errors.Is(err, pgx.ErrNoRows) {
		return SubmissionReview{}, ErrNotFound
	}
	return value, err
}
