//go:build postgres

package evidence

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) RefreshCollectionRequestReviews(ctx context.Context, request Request) (Request, error) {
	needed := false
	for _, field := range request.Fields {
		if field.CollectionResolution != nil {
			needed = true
			break
		}
	}
	if !needed {
		return request, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Request{}, err
	}
	defer tx.Rollback(ctx)
	request, err = RefreshCollectionReviewsPostgres(ctx, tx, request)
	if err != nil {
		return Request{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Request{}, err
	}
	return request, nil
}

// RefreshCollectionReviewsPostgres locks the selected source assessment and
// exact document review, including the case where a first review is still due.
// Its caller supplies the request's own transaction for submission consumption.
func RefreshCollectionReviewsPostgres(ctx context.Context, tx pgx.Tx, request Request) (Request, error) {
	request.Fields = cloneFields(request.Fields)
	for i := range request.Fields {
		r := request.Fields[i].CollectionResolution
		if r == nil {
			continue
		}
		if err := refreshCollectionSourceCurrencyPostgres(ctx, tx, request, r); err != nil {
			return Request{}, err
		}
		if r.Source.AssessmentID == "" {
			continue
		}
		var assessmentID string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM third_party_assessments WHERE id=$1::uuid AND tenant_id=$2::uuid AND legal_entity_id=$3::uuid AND relationship_id::text=$4 FOR SHARE`, r.Source.AssessmentID, request.TenantID, request.LegalEntityID, r.Source.RelationshipID).Scan(&assessmentID); err != nil {
			return Request{}, err
		}
		var review DocumentReview
		var expiry string
		err := tx.QueryRow(ctx, `SELECT id::text,status,COALESCE(validated_by_principal_id::text,''),validated_at,COALESCE(expires_on::text,'')
		 FROM third_party_documents WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND assessment_id=$3::uuid AND request_id=$4::uuid AND artifact_id=$5::uuid FOR SHARE`, request.TenantID, request.LegalEntityID, r.Source.AssessmentID, r.Source.RequestID, r.Source.ArtifactID).Scan(&review.ID, &review.Status, &review.ReviewedBy, &review.ReviewedAt, &expiry)
		if errors.Is(err, pgx.ErrNoRows) && r.Source.Review == nil {
			continue
		}
		if err != nil {
			return Request{}, err
		}
		review.Source = "VENDOR_ASSESSMENT"
		r.Source.Review = &review
		if expiry != "" {
			r.Source.ExpiresOn = expiry
		}
	}
	return request, nil
}

// Keep the exact source's currency population stable while a receipt is consumed.
func refreshCollectionSourceCurrencyPostgres(ctx context.Context, tx pgx.Tx, request Request, receipt *CollectionResolution) error {
	var locked string
	err := tx.QueryRow(ctx, `SELECT id::text FROM capture_requests WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid FOR SHARE`, request.TenantID, request.LegalEntityID, receipt.Source.RequestID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		receipt.Source.Current = false
		return nil
	}
	if err != nil {
		return err
	}
	if receipt.Source.AssessmentID != "" {
		err = tx.QueryRow(ctx, `SELECT id::text FROM third_party_assessments WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid FOR SHARE`, request.TenantID, request.LegalEntityID, receipt.Source.AssessmentID).Scan(&locked)
	} else if receipt.Source.WorkRequestID != "" {
		err = tx.QueryRow(ctx, `SELECT id::text FROM third_party_work_requests WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid FOR SHARE`, request.TenantID, request.LegalEntityID, receipt.Source.WorkRequestID).Scan(&locked)
	} else if receipt.Source.DistributionID != "" {
		err = tx.QueryRow(ctx, `SELECT id::text FROM capture_form_distributions WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid FOR SHARE`, request.TenantID, request.LegalEntityID, receipt.Source.DistributionID).Scan(&locked)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		receipt.Source.Current = false
		return nil
	}
	if err != nil {
		return err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	return tx.QueryRow(ctx, `SELECT `+collectionSourceCurrentSQL("$3::jsonb", "$1::uuid", "$2::uuid"), request.TenantID, request.LegalEntityID, raw).Scan(&receipt.Source.Current)
}

// This purpose-bound predicate reads only the occurrence already named by the
// saved receipt. Reuse the inventory currency rules, including omitted fields
// in later workflow submissions, without granting respondent inventory access.
func collectionSourceCurrentSQL(receipt, tenant, entity string) string {
	return `EXISTS (
	 SELECT 1 FROM (SELECT ` + receipt + ` AS receipt, ` + tenant + ` AS tenant_id, ` + entity + ` AS legal_entity_id) collection_scope
	 CROSS JOIN LATERAL (
	 SELECT 1 ` + documentSubmissionJoinsSQL() + `
	 JOIN LATERAL jsonb_array_elements(req.fields) field ON field->>'id'=collection_scope.receipt->'source'->>'field_id'
	 WHERE req.tenant_id=collection_scope.tenant_id AND req.legal_entity_id=collection_scope.legal_entity_id
	 AND req.id=(collection_scope.receipt->'source'->>'request_id')::uuid
	 AND submission.id=(collection_scope.receipt->'source'->>'submission_id')::uuid
	 AND req.subject_id=collection_scope.receipt->'source'->>'relationship_id'
	 AND COALESCE(r.id::text,'')=COALESCE(collection_scope.receipt->'source'->>'response_revision_id','')
	 AND CASE req.origin_type
	 WHEN 'THIRD_PARTY_ASSESSMENT' THEN assessment.id::text=collection_scope.receipt->'source'->>'assessment_id'
	 WHEN 'THIRD_PARTY_WORK' THEN work.id::text=collection_scope.receipt->'source'->>'work_request_id'
	 ELSE r.id IS NOT NULL END
	 AND ` + documentRevisionScopeSQL() + ` AND ` + documentCurrentSQL() + `
	 ) current_collection_source
	)`
}
