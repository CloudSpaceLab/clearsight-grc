//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func (r *PostgresRepository) WriteAssessmentCollection(ctx context.Context, record assessmentCollectionRecord, _ collectionRequestWriter) (Assessment, evidence.Request, evidence.CollectionResolution, error) {
	fail := func(err error) (Assessment, evidence.Request, evidence.CollectionResolution, error) {
		return Assessment{}, evidence.Request{}, evidence.CollectionResolution{}, err
	}
	if record.Authorize == nil {
		return fail(ErrAssessmentAuthorityUnavailable)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fail(err)
	}
	defer tx.Rollback(ctx)
	ctx = authority.WithPostgresTransaction(ctx, tx)
	tenant, err := resolveTenant(ctx, tx, record.TenantID)
	if err != nil {
		return fail(err)
	}
	current, err := lockAssessment(ctx, tx, tenant, record.LegalEntityID, record.AssessmentID)
	if err != nil {
		return fail(err)
	}
	if current.Version != record.ExpectedVersion {
		return fail(ErrVersionConflict)
	}
	if err = record.Authorize(ctx); err != nil {
		return fail(err)
	}
	request, err := evidence.ReadCollectionRequestPostgres(ctx, tx, tenant, record.LegalEntityID, record.RequestID)
	if err != nil {
		return fail(err)
	}
	source := record.Resolution.Source
	exact, err := evidence.ReadCollectionSourcePostgres(ctx, tx, evidence.DocumentQuery{TenantID: tenant, LegalEntityID: record.LegalEntityID, PrincipalID: record.ActorPrincipalID, RelationshipID: current.RelationshipID, SubmissionID: source.SubmissionID, FieldID: source.FieldID, ArtifactID: source.ArtifactID, ResponseRevisionID: source.ResponseRevisionID, CurrentOnly: !record.Review, Limit: 1})
	if err != nil {
		return fail(err)
	}
	if exact.SubmissionChannel != "MAGIC_LINK" || exact.ID != source.ID || exact.SHA256 != source.SHA256 || exact.SizeBytes != source.SizeBytes || exact.ArtifactStatus != evidence.ArtifactAvailable {
		return fail(ErrAssessmentCompletionBlocked)
	}
	record.Resolution.Source = exact
	record.Resolution.SourceArtifactRequestID = exact.ArtifactRequestID
	// Fence source review and currency changes in the same command transaction.
	probe := request
	probe.Fields = []evidence.Field{{ID: record.FieldID, Type: "vendor_document", CollectionResolution: &record.Resolution}}
	probe, err = evidence.RefreshCollectionReviewsPostgres(ctx, tx, probe)
	if err != nil {
		return fail(err)
	}
	record.Resolution = *probe.Fields[0].CollectionResolution
	receipt, err := applyCollectionRecord(&current, &request, record)
	if err != nil {
		return fail(err)
	}
	fields, err := json.Marshal(request.Fields)
	if err != nil {
		return fail(err)
	}
	tag, err := tx.Exec(ctx, `UPDATE capture_requests SET fields=$4::jsonb,version=version+1,updated_at=$5 WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$6`, tenant, record.LegalEntityID, request.ID, string(fields), record.At, record.ExpectedRequestVersion)
	if err != nil {
		return fail(err)
	}
	if tag.RowsAffected() != 1 {
		return fail(ErrVersionConflict)
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return fail(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO capture_field_collection_resolutions(id,tenant_id,legal_entity_id,assessment_id,assessment_version,request_id,request_version,field_id,version,source_submission_id,source_field_id,source_artifact_id,actor_principal_id,receipt,created_at,source_request_id,source_artifact_request_id) VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6::uuid,$7,$8,$9,$10::uuid,$11,$12::uuid,$13::uuid,$14::jsonb,$15,$16::uuid,$17::uuid)`, receipt.ID, tenant, record.LegalEntityID, current.ID, current.Version, request.ID, request.Version, record.FieldID, receipt.Version, receipt.Source.SubmissionID, receipt.Source.FieldID, receipt.Source.ArtifactID, record.ActorPrincipalID, string(raw), record.At, receipt.Source.RequestID, receipt.SourceArtifactRequestID)
	if err != nil {
		return fail(err)
	}
	if err = updateAssessment(ctx, tx, tenant, current); err != nil {
		return fail(err)
	}
	eventType := "AssessmentDocumentReconciled"
	if record.Review {
		eventType = "AssessmentReusedDocumentReviewed"
	}
	var eventID string
	err = tx.QueryRow(ctx, `INSERT INTO third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,actor_principal_id,event_type,payload,occurred_at) VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,$4::uuid,$5,jsonb_build_object('status',$6::text,'request_id',$7::text,'collection_resolution_id',$8::text,'field_id',$9::text),$10) RETURNING id::text`, tenant, current.ID, current.Version, record.ActorPrincipalID, eventType, current.Status, request.ID, receipt.ID, record.FieldID, record.At).Scan(&eventID)
	if err != nil {
		return fail(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,payload,occurred_at,available_at) VALUES($1::uuid,'THIRD_PARTY_ASSESSMENT',$2::uuid,$3,jsonb_build_object('version',$4::bigint,'request_id',$5::text,'collection_resolution_id',$6::text),$7,$7)`, tenant, current.ID, eventType, current.Version, request.ID, receipt.ID, record.At)
	if err != nil {
		return fail(err)
	}
	if err = r.commitThirdPartyEvents(ctx, tx, assessmentCommitProof(eventID, current, eventType)); err != nil {
		return fail(err)
	}
	return current, request, receipt, nil
}
