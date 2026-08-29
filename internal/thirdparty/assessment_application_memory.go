package thirdparty

import (
	"context"
)

func applicationReceiptKey(assessmentID, responseRevisionID string) string {
	return assessmentID + "\x00" + responseRevisionID
}

func (r *MemoryAssessmentRepository) GetResponseApplicationReceipt(_ context.Context, scope Scope, assessmentID, responseRevisionID string) (ResponseApplicationReceipt, error) {
	r.assessmentMu.RLock()
	defer r.assessmentMu.RUnlock()
	receipt, ok := r.applicationReceipts[applicationReceiptKey(assessmentID, responseRevisionID)]
	assessment, assessmentOK := r.assessments[assessmentID]
	if !ok || !assessmentOK || assessment.TenantID != scope.TenantID || assessment.LegalEntityID != scope.LegalEntityID {
		return ResponseApplicationReceipt{}, ErrNotFound
	}
	return cloneApplicationReceipt(receipt), nil
}

func (r *MemoryAssessmentRepository) ApplyAssessmentResponse(_ context.Context, record AssessmentApplicationRecord) (ResponseApplicationReceipt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assessmentMu.Lock()
	defer r.assessmentMu.Unlock()
	key := applicationReceiptKey(record.AssessmentID, record.ResponseRevisionID)
	if receipt, exists := r.applicationReceipts[key]; exists {
		return cloneApplicationReceipt(receipt), nil
	}
	assessment, ok := r.assessments[record.AssessmentID]
	if !ok || assessment.TenantID != record.TenantID || assessment.LegalEntityID != record.LegalEntityID {
		return ResponseApplicationReceipt{}, ErrNotFound
	}
	if assessment.Version != record.ExpectedAssessmentVersion || assessment.Status != AssessmentUnderReview || assessment.SubmissionID != record.ResponseRevisionID {
		return ResponseApplicationReceipt{}, ErrVersionConflict
	}
	relationship, ok := r.relationships[assessment.RelationshipID]
	if !ok || relationship.TenantID != record.TenantID || relationship.LegalEntityID != record.LegalEntityID {
		return ResponseApplicationReceipt{}, ErrNotFound
	}
	current, ok := r.vendors[relationship.VendorID]
	if !ok || current.ID != record.Vendor.ID || current.Version != record.PriorVendorVersion {
		return ResponseApplicationReceipt{}, ErrVersionConflict
	}
	updated := current
	if record.IdentityChanged {
		updated.LegalName, updated.TradingName, updated.RegistrationRef, updated.Jurisdiction = record.Vendor.LegalName, record.Vendor.TradingName, record.Vendor.RegistrationRef, record.Vendor.Jurisdiction
		updated.RegisteredAddress, updated.WebsiteDomain = record.Vendor.RegisteredAddress, record.Vendor.WebsiteDomain
		updated.Version++
		updated.UpdatedAt = record.AppliedAt.UTC()
	}
	type validatedReplacement struct {
		input             DocumentReplacementApplication
		priorAssessmentID string
		priorArtifactID   string
		prior             AssessmentDocument
		replacement       AssessmentDocument
	}
	validated := make([]validatedReplacement, 0, len(record.DocumentReplacements))
	for _, replacement := range record.DocumentReplacements {
		var prior AssessmentDocument
		priorAssessmentID, priorArtifactID := "", ""
		currentCount := 0
		for assessmentID, documents := range r.assessmentDocuments {
			for artifactID, value := range documents {
				if value.ID != replacement.ReplacementID && value.RelationshipID == assessment.RelationshipID && value.DocumentType == replacement.DocumentType && (value.Status == AssessmentDocumentValidated || value.Status == AssessmentDocumentExpired) {
					currentCount++
				}
				if value.ID == replacement.PriorDocumentID {
					prior, priorAssessmentID, priorArtifactID = value, assessmentID, artifactID
				}
			}
		}
		replacementDocuments := r.assessmentDocuments[record.AssessmentID]
		newDocument, exists := replacementDocuments[replacement.ReplacementArtifactID]
		if priorAssessmentID == "" || currentCount != 1 || !exists || prior.ID == newDocument.ID || prior.RelationshipID != assessment.RelationshipID || prior.DocumentType != replacement.DocumentType || prior.Version != replacement.PriorVersion || (prior.Status != AssessmentDocumentValidated && prior.Status != AssessmentDocumentExpired) || newDocument.ID != replacement.ReplacementID || newDocument.DocumentType != replacement.DocumentType || newDocument.Status != AssessmentDocumentValidated {
			return ResponseApplicationReceipt{}, ErrVersionConflict
		}
		validated = append(validated, validatedReplacement{input: replacement, priorAssessmentID: priorAssessmentID, priorArtifactID: priorArtifactID, prior: prior, replacement: newDocument})
	}
	for _, item := range validated {
		priorDocuments := r.assessmentDocuments[item.priorAssessmentID]
		prior := item.prior
		prior.Status = AssessmentDocumentSuperseded
		prior.Version++
		prior.UpdatedAt = record.AppliedAt.UTC()
		priorDocuments[item.priorArtifactID] = prior
		r.assessmentDocuments[item.priorAssessmentID] = priorDocuments
		replacementDocuments := r.assessmentDocuments[record.AssessmentID]
		newDocument := item.replacement
		newDocument.SupersedesDocumentID = prior.ID
		newDocument.Version++
		newDocument.UpdatedAt = record.AppliedAt.UTC()
		replacementDocuments[item.input.ReplacementArtifactID] = newDocument
		r.assessmentDocuments[record.AssessmentID] = replacementDocuments
	}
	if record.IdentityChanged {
		r.vendors[updated.ID] = updated
		r.appendVendorIdentityAudit(updated, record.ActorPrincipalID, VendorIdentityUpdatedEvent)
	}
	assessment.Version++
	assessment.UpdatedAt = record.AppliedAt.UTC()
	r.assessments[assessment.ID] = assessment
	r.appendMemoryAssessmentAudit(assessment, record.ActorPrincipalID, "AssessmentResponseApplied")
	receipt := ResponseApplicationReceipt{ID: record.ReceiptID, AssessmentID: assessment.ID, ResponseRevisionID: record.ResponseRevisionID, VendorID: current.ID, ActorPrincipalID: record.ActorPrincipalID, AcceptedFieldIDs: append([]string(nil), record.AcceptedFieldIDs...), RejectedFieldIDs: append([]string(nil), record.RejectedFieldIDs...), Decisions: append([]FieldApplicationDecision(nil), record.Decisions...), PriorVendorVersion: current.Version, ResultVendorVersion: updated.Version, ResultAssessmentVersion: assessment.Version, AppliedAt: record.AppliedAt.UTC()}
	r.applicationReceipts[key] = receipt
	return cloneApplicationReceipt(receipt), nil
}

func cloneApplicationReceipt(value ResponseApplicationReceipt) ResponseApplicationReceipt {
	value.AcceptedFieldIDs = append([]string(nil), value.AcceptedFieldIDs...)
	value.RejectedFieldIDs = append([]string(nil), value.RejectedFieldIDs...)
	value.Decisions = append([]FieldApplicationDecision(nil), value.Decisions...)
	return value
}
