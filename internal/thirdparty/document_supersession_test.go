package thirdparty

import (
	"context"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

func TestApplyAssessmentResponseSupersedesExactValidatedDocument(t *testing.T) {
	service, review, repository, actor, assessment := assessmentApplicationFixture(t)
	reader := review.evidence.(*assessmentReviewEvidenceStub)
	request := reader.requests["request-1"]
	request.Fields = []evidence.Field{{ID: "certificate", SectionID: "identity", Label: "Certificate of operation", Type: string(formcontract.TypeVendorDocument), Required: true, CollectionIntent: formcontract.IntentReplaceHeldDocument, RecordTarget: &formcontract.RecordTarget{Key: "VENDOR.DOCUMENT.CERTIFICATE_OF_OPERATION", RequiredSubjectType: "VENDOR_RELATIONSHIP"}, RecordBaseline: &evidence.RecordBaseline{TargetKey: "VENDOR.DOCUMENT.CERTIFICATE_OF_OPERATION", SubjectType: "VENDOR_RELATIONSHIP", SubjectID: assessment.RelationshipID, RecordID: "old-document", RecordVersion: 4, DisplayValue: "Certificate of operation · CAC-10"}}}
	reader.requests["request-1"] = request
	documentAnswer := formcontract.DocumentAnswer{ArtifactID: "artifact-1", DocumentType: "CERTIFICATE_OF_OPERATION", Reference: "CAC-11", IssuedBy: "CAC"}
	submission := reader.submissions["submission-1"]
	submission.Answers = map[string]formcontract.AnswerValue{"certificate": {Document: &documentAnswer}}
	submission.AnswerProvenance = map[string]evidence.AnswerProvenance{"certificate": {Origin: evidence.AnswerRespondentCorrected}}
	reader.submissions["submission-1"] = submission
	repository.assessmentDocuments["prior-assessment"] = map[string]AssessmentDocument{"old-artifact": {ID: "old-document", Scope: Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, RelationshipID: assessment.RelationshipID, AssessmentID: "prior-assessment", RequestID: "prior-request", ArtifactID: "old-artifact", DocumentType: "CERTIFICATE_OF_OPERATION", Reference: "CAC-10", Status: AssessmentDocumentValidated, EvidenceClass: AssessmentDocumentBankValidated, Version: 4}}
	repository.assessmentDocuments[assessment.ID] = map[string]AssessmentDocument{"artifact-1": {ID: "new-document", Scope: Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, RelationshipID: assessment.RelationshipID, AssessmentID: assessment.ID, RequestID: assessment.CurrentRequestID, ArtifactID: "artifact-1", DocumentType: "CERTIFICATE_OF_OPERATION", Reference: "CAC-11", Status: AssessmentDocumentValidated, EvidenceClass: AssessmentDocumentBankValidated, Version: 1}}

	result, err := service.ApplyResponse(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), Actor{}, assessment.ID, "submission-1", ApplyAssessmentResponseInput{ExpectedAssessmentVersion: assessment.Version, ExpectedSubmissionRevision: 1, Decisions: []FieldApplicationDecision{{FieldID: "certificate", Decision: FieldApplicationAccept, Rationale: "The replacement artifact was validated."}}})
	if err != nil {
		t.Fatal(err)
	}
	old := repository.assessmentDocuments["prior-assessment"]["old-artifact"]
	current := repository.assessmentDocuments[assessment.ID]["artifact-1"]
	if old.Status != AssessmentDocumentSuperseded || old.Version != 5 || current.SupersedesDocumentID != old.ID || current.Version != 2 || result.Receipt.ResultVendorVersion != 1 {
		t.Fatalf("supersession old=%#v current=%#v receipt=%#v", old, current, result.Receipt)
	}
	if values, err := repository.CurrentRelationshipDocuments(context.Background(), Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, assessment.RelationshipID, "CERTIFICATE_OF_OPERATION"); err != nil || len(values) != 1 || values[0].ID != "new-document" {
		t.Fatalf("current document lookup = %#v err=%v", values, err)
	}
}
