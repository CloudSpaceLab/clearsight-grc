//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetResponseApplicationReceipt(ctx context.Context, scope Scope, assessmentID, responseRevisionID string) (ResponseApplicationReceipt, error) {
	value, err := scanResponseApplicationReceipt(r.pool.QueryRow(ctx, `
		SELECT receipt.id::text,receipt.assessment_id::text,COALESCE(receipt.distribution_id::text,''),receipt.response_revision_id::text,
		       receipt.vendor_id::text,receipt.actor_principal_id::text,receipt.accepted_field_ids,receipt.rejected_field_ids,receipt.decisions,
		       receipt.prior_vendor_version,receipt.result_vendor_version,receipt.result_assessment_version,receipt.applied_at
		FROM third_party_response_application_receipts receipt
		JOIN tenants tenant ON tenant.id=receipt.tenant_id
		WHERE (tenant.id::text=$1 OR tenant.slug=$1) AND receipt.legal_entity_id::text=$2
		  AND receipt.assessment_id::text=$3 AND receipt.response_revision_id::text=$4`, scope.TenantID, scope.LegalEntityID, assessmentID, responseRevisionID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseApplicationReceipt{}, ErrNotFound
	}
	if err != nil {
		return ResponseApplicationReceipt{}, fmt.Errorf("get response application receipt: %w", err)
	}
	return value, nil
}

func (r *PostgresRepository) ApplyAssessmentResponse(ctx context.Context, record AssessmentApplicationRecord) (ResponseApplicationReceipt, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ResponseApplicationReceipt{}, fmt.Errorf("begin response application: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tenantID, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return ResponseApplicationReceipt{}, err
	}
	if existing, existingErr := scanResponseApplicationReceipt(tx.QueryRow(ctx, `
		SELECT id::text,assessment_id::text,COALESCE(distribution_id::text,''),response_revision_id::text,vendor_id::text,actor_principal_id::text,
		       accepted_field_ids,rejected_field_ids,decisions,prior_vendor_version,result_vendor_version,result_assessment_version,applied_at
		FROM third_party_response_application_receipts
		WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND assessment_id::text=$3 AND response_revision_id::text=$4
		FOR SHARE`, tenantID, record.LegalEntityID, record.AssessmentID, record.ResponseRevisionID)); existingErr == nil {
		return existing, nil
	} else if !errors.Is(existingErr, pgx.ErrNoRows) {
		return ResponseApplicationReceipt{}, existingErr
	}
	assessment, err := lockAssessment(ctx, tx, tenantID, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return ResponseApplicationReceipt{}, err
	}
	if assessment.Version != record.ExpectedAssessmentVersion || assessment.Status != AssessmentUnderReview || assessment.SubmissionID != record.ResponseRevisionID || record.ExpectedSubmissionRevision != 1 {
		return ResponseApplicationReceipt{}, ErrVersionConflict
	}
	aggregate, err := scanAggregate(tx.QueryRow(ctx, relationshipSelect+`
		WHERE t.id=$1::uuid AND r.legal_entity_id::text=$2 AND r.id::text=$3
		FOR UPDATE OF p,r`, tenantID, record.LegalEntityID, assessment.RelationshipID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ResponseApplicationReceipt{}, ErrNotFound
	}
	if err != nil {
		return ResponseApplicationReceipt{}, fmt.Errorf("lock response application vendor: %w", err)
	}
	if aggregate.Vendor.ID != record.Vendor.ID || aggregate.Vendor.Version != record.PriorVendorVersion {
		return ResponseApplicationReceipt{}, ErrVersionConflict
	}
	updated := aggregate.Vendor
	if record.IdentityChanged {
		updated, err = updateVendorIdentityTx(ctx, tx, tenantID, UpdateVendorIdentityRecord{Scope: record.Scope, ID: aggregate.Vendor.ID, ExpectedVersion: record.PriorVendorVersion, Vendor: record.Vendor, ActorID: record.ActorPrincipalID})
		if err != nil {
			return ResponseApplicationReceipt{}, err
		}
	}
	for _, replacement := range record.DocumentReplacements {
		var currentCount int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM third_party_documents
			WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND relationship_id=$3::uuid AND document_type=$4
			  AND id<>$5::uuid AND status IN ('VALIDATED','EXPIRED')`, tenantID, record.LegalEntityID, assessment.RelationshipID, replacement.DocumentType, replacement.ReplacementID).Scan(&currentCount); err != nil {
			return ResponseApplicationReceipt{}, err
		}
		if currentCount != 1 {
			return ResponseApplicationReceipt{}, ErrVersionConflict
		}
		var priorVersion, replacementVersion int64
		var priorStatus, replacementStatus, artifactStatus string
		err = tx.QueryRow(ctx, `SELECT version,status FROM third_party_documents
			WHERE tenant_id=$1::uuid AND legal_entity_id::text=$2 AND relationship_id=$3::uuid AND id::text=$4 AND document_type=$5
			FOR UPDATE`, tenantID, record.LegalEntityID, assessment.RelationshipID, replacement.PriorDocumentID, replacement.DocumentType).Scan(&priorVersion, &priorStatus)
		if errors.Is(err, pgx.ErrNoRows) || priorVersion != replacement.PriorVersion || (priorStatus != string(AssessmentDocumentValidated) && priorStatus != string(AssessmentDocumentExpired)) {
			return ResponseApplicationReceipt{}, ErrVersionConflict
		}
		if err != nil {
			return ResponseApplicationReceipt{}, err
		}
		err = tx.QueryRow(ctx, `SELECT d.version,d.status,a.status FROM third_party_documents d
			JOIN capture_artifacts a ON a.tenant_id=d.tenant_id AND a.id=d.artifact_id AND a.request_id=d.request_id
			WHERE d.tenant_id=$1::uuid AND d.legal_entity_id::text=$2 AND d.relationship_id=$3::uuid AND d.assessment_id=$4::uuid
			  AND d.id::text=$5 AND d.artifact_id::text=$6 AND d.document_type=$7
			FOR UPDATE OF d,a`, tenantID, record.LegalEntityID, assessment.RelationshipID, assessment.ID, replacement.ReplacementID, replacement.ReplacementArtifactID, replacement.DocumentType).Scan(&replacementVersion, &replacementStatus, &artifactStatus)
		if errors.Is(err, pgx.ErrNoRows) || replacementStatus != string(AssessmentDocumentValidated) || artifactStatus != "AVAILABLE" || replacement.PriorDocumentID == replacement.ReplacementID {
			return ResponseApplicationReceipt{}, ErrVersionConflict
		}
		if err != nil {
			return ResponseApplicationReceipt{}, err
		}
		priorResult, err := tx.Exec(ctx, `UPDATE third_party_documents SET status='SUPERSEDED',version=version+1,updated_at=$4
			WHERE tenant_id=$1::uuid AND id=$2::uuid AND version=$3`, tenantID, replacement.PriorDocumentID, priorVersion, record.AppliedAt)
		if err != nil {
			return ResponseApplicationReceipt{}, err
		}
		if priorResult.RowsAffected() != 1 {
			return ResponseApplicationReceipt{}, ErrVersionConflict
		}
		replacementResult, err := tx.Exec(ctx, `UPDATE third_party_documents SET supersedes_document_id=$2::uuid,version=version+1,updated_at=$4
			WHERE tenant_id=$1::uuid AND id=$3::uuid AND version=$5`, tenantID, replacement.PriorDocumentID, replacement.ReplacementID, record.AppliedAt, replacementVersion)
		if err != nil {
			return ResponseApplicationReceipt{}, err
		}
		if replacementResult.RowsAffected() != 1 {
			return ResponseApplicationReceipt{}, ErrVersionConflict
		}
	}
	assessment.Version++
	assessment.UpdatedAt = record.AppliedAt.UTC()
	if err := updateAssessment(ctx, tx, tenantID, assessment); err != nil {
		return ResponseApplicationReceipt{}, err
	}
	acceptedJSON, _ := json.Marshal(record.AcceptedFieldIDs)
	rejectedJSON, _ := json.Marshal(record.RejectedFieldIDs)
	decisionsJSON, _ := json.Marshal(record.Decisions)
	receipt := ResponseApplicationReceipt{ID: record.ReceiptID, AssessmentID: assessment.ID, ResponseRevisionID: record.ResponseRevisionID, VendorID: aggregate.Vendor.ID, ActorPrincipalID: record.ActorPrincipalID, AcceptedFieldIDs: append([]string(nil), record.AcceptedFieldIDs...), RejectedFieldIDs: append([]string(nil), record.RejectedFieldIDs...), Decisions: append([]FieldApplicationDecision(nil), record.Decisions...), PriorVendorVersion: aggregate.Vendor.Version, ResultVendorVersion: updated.Version, ResultAssessmentVersion: assessment.Version, AppliedAt: record.AppliedAt.UTC()}
	_, err = tx.Exec(ctx, `INSERT INTO third_party_response_application_receipts(
		id,tenant_id,legal_entity_id,assessment_id,response_revision_id,vendor_id,actor_principal_id,accepted_field_ids,rejected_field_ids,decisions,
		prior_vendor_version,result_vendor_version,result_assessment_version,applied_at
	) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7::uuid,$8::jsonb,$9::jsonb,$10::jsonb,$11,$12,$13,$14)`,
		receipt.ID, tenantID, record.LegalEntityID, receipt.AssessmentID, receipt.ResponseRevisionID, receipt.VendorID, receipt.ActorPrincipalID,
		string(acceptedJSON), string(rejectedJSON), string(decisionsJSON), receipt.PriorVendorVersion, receipt.ResultVendorVersion, receipt.ResultAssessmentVersion, receipt.AppliedAt)
	if err != nil {
		return ResponseApplicationReceipt{}, fmt.Errorf("store response application receipt: %w", err)
	}
	if _, err := appendAssessmentEvent(ctx, tx, tenantID, assessment, record.ActorPrincipalID, "AssessmentResponseApplied"); err != nil {
		return ResponseApplicationReceipt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		commitErr := fmt.Errorf("commit response application: %w", err)
		probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		confirmed, probeErr := r.GetResponseApplicationReceipt(probeCtx, record.Scope, record.AssessmentID, record.ResponseRevisionID)
		if probeErr == nil && confirmed.ID == receipt.ID {
			return confirmed, nil
		}
		if probeErr != nil && !errors.Is(probeErr, ErrNotFound) {
			return ResponseApplicationReceipt{}, errors.Join(commitErr, probeErr)
		}
		return ResponseApplicationReceipt{}, commitErr
	}
	return receipt, nil
}

func scanResponseApplicationReceipt(row rowScanner) (ResponseApplicationReceipt, error) {
	var value ResponseApplicationReceipt
	var acceptedJSON, rejectedJSON, decisionsJSON []byte
	err := row.Scan(&value.ID, &value.AssessmentID, &value.DistributionID, &value.ResponseRevisionID, &value.VendorID, &value.ActorPrincipalID,
		&acceptedJSON, &rejectedJSON, &decisionsJSON, &value.PriorVendorVersion, &value.ResultVendorVersion, &value.ResultAssessmentVersion, &value.AppliedAt)
	if err == nil {
		if err = json.Unmarshal(acceptedJSON, &value.AcceptedFieldIDs); err == nil {
			if err = json.Unmarshal(rejectedJSON, &value.RejectedFieldIDs); err == nil {
				err = json.Unmarshal(decisionsJSON, &value.Decisions)
			}
		}
	}
	return value, err
}
