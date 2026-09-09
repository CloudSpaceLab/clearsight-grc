//go:build postgres

package thirdparty

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/jackc/pgx/v5"
)

// Reconstruct the same visible merged response used by Response, with source
// requests, submissions and artifacts fenced by the accepting transaction.
func (r *PostgresRepository) verifyVendorWorkArtifacts(ctx context.Context, tx pgx.Tx, tenant string, work VendorWorkRequest) ([]string, error) {
	var presentation, sections, fields []byte
	if err := tx.QueryRow(ctx, `SELECT presentation,sections,fields FROM monitoring_form_templates WHERE tenant_id=$1::uuid AND legal_entity_id=$2::uuid AND id=$3::uuid AND version=$4 FOR SHARE`, tenant, work.LegalEntityID, work.FormTemplateID, work.FormTemplateVersion).Scan(&presentation, &sections, &fields); err != nil {
		return nil, ErrVendorWorkAcceptanceBlocked
	}
	var contract formcontract.Contract
	if json.Unmarshal(presentation, &contract.Presentation) != nil || json.Unmarshal(sections, &contract.Sections) != nil || json.Unmarshal(fields, &contract.Fields) != nil {
		return nil, ErrVendorWorkAcceptanceBlocked
	}
	contract, err := formcontract.Normalize(contract)
	if err != nil {
		return nil, ErrVendorWorkAcceptanceBlocked
	}
	rows, err := tx.Query(ctx, `SELECT req.id::text,s.id::text,req.fields,s.answers
 FROM third_party_work_capture_links l
 JOIN capture_requests req ON req.tenant_id=l.tenant_id AND req.id=l.request_id AND req.legal_entity_id=l.legal_entity_id AND req.origin_type=$4 AND req.origin_id=l.work_request_id::text AND req.origin_version=l.origin_version
 JOIN capture_submissions s ON s.tenant_id=l.tenant_id AND s.id=l.submission_id AND s.request_id=req.id
 WHERE l.tenant_id=$1::uuid AND l.legal_entity_id=$2::uuid AND l.work_request_id=$3::uuid
 ORDER BY l.sequence LIMIT 101 FOR SHARE OF l,req,s`, tenant, work.LegalEntityID, work.ID, VendorWorkOrigin)
	if err != nil {
		return nil, err
	}
	type source struct{ request, submission string }
	answers := map[string]formcontract.AnswerValue{}
	sources := map[string]source{}
	count := 0
	currentFound := false
	for rows.Next() {
		count++
		var item source
		var requestFields, submitted []byte
		if err = rows.Scan(&item.request, &item.submission, &requestFields, &submitted); err != nil {
			rows.Close()
			return nil, err
		}
		var captureFields []evidence.Field
		var captureAnswers map[string]formcontract.AnswerValue
		if json.Unmarshal(requestFields, &captureFields) != nil || json.Unmarshal(submitted, &captureAnswers) != nil {
			rows.Close()
			return nil, ErrVendorWorkAcceptanceBlocked
		}
		for _, field := range captureFields {
			delete(answers, field.ID)
			delete(sources, field.ID)
		}
		for fieldID, answer := range captureAnswers {
			answers[fieldID] = answer
			sources[fieldID] = item
		}
		if item.request == work.CurrentRequestID && item.submission == work.SubmissionID {
			currentFound = true
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if count > 100 || !currentFound {
		return nil, ErrVendorWorkAcceptanceBlocked
	}
	visible, err := formcontract.VisibleFields(contract, answers)
	if err != nil {
		return nil, ErrVendorWorkAcceptanceBlocked
	}
	unscanned := []string{}
	checked := map[string]bool{}
	for _, field := range visible {
		answer, ok := answers[field.ID]
		if !ok || !reviewArtifactField(field.Type) {
			continue
		}
		item, ok := sources[field.ID]
		if !ok {
			return nil, ErrVendorWorkAcceptanceBlocked
		}
		for _, artifactID := range reviewArtifactIDs(answer) {
			if checked[artifactID] {
				continue
			}
			checked[artifactID] = true
			if len(checked) > assessmentReviewMaxArtifacts || !validAssessmentIdentifier(artifactID) {
				return nil, ErrVendorWorkAcceptanceBlocked
			}
			var status evidence.ArtifactStatus
			err = tx.QueryRow(ctx, `SELECT status FROM capture_artifacts WHERE tenant_id=$1::uuid AND request_id=$2::uuid AND submission_id=$3::uuid AND id=$4::uuid FOR SHARE`, tenant, item.request, item.submission, artifactID).Scan(&status)
			if err != nil || !r.artifactUseAllowed(status) {
				return nil, ErrVendorWorkAcceptanceBlocked
			}
			if r.demoUnscannedAllowed(status) {
				unscanned = append(unscanned, artifactID)
			}
		}
	}
	return unscanned, nil
}
