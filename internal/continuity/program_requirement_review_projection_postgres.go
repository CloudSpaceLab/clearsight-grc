//go:build postgres

package continuity

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
)

// applyRequirementReviewProjection runs in the same transaction as the
// Program event and outbox. A failed status update aborts the entire decision.
func applyRequirementReviewProjection(ctx context.Context, tx pgx.Tx, event Event) error {
	if event.Type == EventRequirementReviewProposed {
		var v RequirementReview
		if err := json.Unmarshal(event.Payload, &v); err != nil {
			return err
		}
		result, err := tx.Exec(ctx, `INSERT INTO program_requirement_reviews(id,tenant_id,program_id,requirement_id,proposed_by,proposal_rationale,proposed_at,status,version)
   SELECT $1::uuid,t.id,p.id,$4::uuid,$5::uuid,$6,$7,$8,$9 FROM programs p JOIN tenants t ON t.id=p.tenant_id
   WHERE p.id=$2::uuid AND (t.id::text=$3 OR t.slug=$3)`, v.ID, v.ProgramID, v.TenantID, v.RequirementID, v.ProposedBy, v.ProposalRationale, v.ProposedAt, v.Status, v.Version)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrNotFound
		}
		return nil
	}
	var v requirementReviewDecisionEvent
	if err := json.Unmarshal(event.Payload, &v); err != nil {
		return err
	}
	a := v.Review
	result, err := tx.Exec(ctx, `UPDATE program_requirement_reviews r SET status=$4,reviewed_by=$5::uuid,review_rationale=$6,reviewed_at=$7,version=$8
 FROM tenants t WHERE r.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND r.program_id=$2::uuid AND r.id=$3::uuid AND r.status='PENDING' AND r.version=$9`, a.TenantID, a.ProgramID, a.ID, a.Status, a.ReviewedBy, a.ReviewRationale, a.ReviewedAt, a.Version, a.Version-1)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrVersionConflict
	}
	if event.Type == EventRequirementReviewApproved {
		q := v.Requirement
		result, err = tx.Exec(ctx, `UPDATE program_requirements r SET status='APPROVED',effective_from=$4,version=$5 FROM tenants t
   WHERE r.tenant_id=t.id AND (t.id::text=$1 OR t.slug=$1) AND r.program_id=$2::uuid AND r.id=$3::uuid AND r.status='DRAFT' AND r.version=$6`, q.TenantID, q.ProgramID, q.ID, q.EffectiveFrom, q.Version, q.Version-1)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return ErrVersionConflict
		}
	}
	return nil
}
