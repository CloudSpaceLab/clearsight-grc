package thirdparty

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func (r *MemoryAssessmentRepository) WriteAssessmentCollection(ctx context.Context, record assessmentCollectionRecord, writer collectionRequestWriter) (Assessment, evidence.Request, evidence.CollectionResolution, error) {
	if writer == nil || record.Authorize == nil {
		return Assessment{}, evidence.Request{}, evidence.CollectionResolution{}, ErrAssessmentReadinessUnavailable
	}
	if err := record.Authorize(ctx); err != nil {
		return Assessment{}, evidence.Request{}, evidence.CollectionResolution{}, err
	}
	var saved Assessment
	var receipt evidence.CollectionResolution
	request, err := writer.MutateCollectionRequest(ctx, record.TenantID, record.RequestID, record.ExpectedRequestVersion, &record.Resolution, func(request *evidence.Request) error {
		r.assessmentMu.Lock()
		defer r.assessmentMu.Unlock()
		current, ok := r.assessments[record.AssessmentID]
		if !ok || current.TenantID != record.TenantID || current.LegalEntityID != record.LegalEntityID {
			return ErrNotFound
		}
		record.Resolution.Source.DemoUnscannedAllowed = r.demoUnscannedAllowed(record.Resolution.Source.ArtifactStatus)
		var applyErr error
		receipt, applyErr = applyCollectionRecord(&current, request, record)
		if applyErr != nil {
			return applyErr
		}
		r.assessments[current.ID] = current
		saved = current
		eventType := "AssessmentDocumentReconciled"
		if record.Review {
			eventType = "AssessmentReusedDocumentReviewed"
		}
		r.appendMemoryAssessmentAudit(current, record.ActorPrincipalID, eventType)
		raw, _ := json.Marshal(receipt)
		r.assessmentEvents[len(r.assessmentEvents)-1].Payload["collection_resolution"] = string(raw)
		return nil
	})
	return saved, request, receipt, err
}
