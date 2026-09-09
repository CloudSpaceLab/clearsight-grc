package thirdparty

import (
	"context"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"testing"
	"time"
)

type collectionSourcesStub struct{ value evidence.DocumentOccurrence }

func (s collectionSourcesStub) ListDocuments(_ context.Context, q evidence.DocumentQuery) (evidence.DocumentPage, error) {
	if q.RelationshipID != s.value.RelationshipID || q.SubmissionID != s.value.SubmissionID || q.ArtifactID != s.value.ArtifactID || q.FieldID != s.value.FieldID {
		return evidence.DocumentPage{}, evidence.ErrNotFound
	}
	return evidence.DocumentPage{Items: []evidence.DocumentOccurrence{s.value}}, nil
}
func (s *assessmentReviewEvidenceStub) MutateCollectionRequest(_ context.Context, tenant, requestID string, expected int64, _ *evidence.CollectionResolution, apply func(*evidence.Request) error) (evidence.Request, error) {
	request := s.requests[requestID]
	if request.Version != expected {
		return evidence.Request{}, evidence.ErrVersionConflict
	}
	request.Fields = append([]evidence.Field(nil), request.Fields...)
	if err := apply(&request); err != nil {
		return evidence.Request{}, err
	}
	s.requests[requestID] = request
	return request, nil
}

func TestReconcileCollectionKeepsVendorSubmissionAbsentAndBankReviewPending(t *testing.T) {
	review, actor, assessment, capture := assessmentReviewFixture(t)
	repo := review.links.(*MemoryAssessmentRepository)
	assessment.Status = AssessmentCollecting
	assessment.SubmissionID = ""
	assessment.SubmittedAt = nil
	assessment.Conclusion = ""
	assessment.ConclusionRationale = ""
	repo.assessments[assessment.ID] = assessment
	request := capture.requests[assessment.CurrentRequestID]
	request.LegalEntityID = actor.LegalEntityID
	request.Status = evidence.RequestReady
	request.Fields = request.Fields[2:3]
	capture.requests[request.ID] = request
	review.assessments.guard = newAssessmentGuard()
	review.assessments.now = func() time.Time { return assessment.UpdatedAt }
	source := evidence.DocumentOccurrence{ID: "old:submission-old:report:artifact-1", SubmissionChannel: "MAGIC_LINK", ArtifactID: "artifact-1", RequestID: "old-request", ArtifactRequestID: "old-request", SubmissionID: "submission-old", FieldID: "report", RelationshipID: assessment.RelationshipID, FileName: "Report.pdf", FieldLabel: "Assurance report", ArtifactStatus: evidence.ArtifactAvailable, Current: true, SizeBytes: 100, SHA256: "digest"}
	source.MediaType = "application/pdf"
	review.ConfigureCollectionSources(collectionSourcesStub{source}, nil)
	capture.artifacts[source.ArtifactID] = evidence.Artifact{ID: source.ArtifactID, TenantID: actor.TenantID, RequestID: source.ArtifactRequestID, Status: source.ArtifactStatus, SHA256: source.SHA256, SizeBytes: source.SizeBytes}
	input := ReconcileAssessmentCollectionInput{ExpectedVersion: assessment.Version, RequestID: request.ID, ExpectedRequestVersion: request.Version, SourceSubmissionID: source.SubmissionID, SourceFieldID: source.FieldID, SourceArtifactID: source.ArtifactID, Rationale: "This submitted report covers the requested service."}
	result, err := review.ReconcileCollection(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, "assurance_report", input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Collection.VendorPendingCount != 0 || result.Collection.BankPendingCount != 1 || result.Receipt.BankReviewState != "PENDING" {
		t.Fatalf("incorrect collection result: %#v", result)
	}
	current := repo.assessments[assessment.ID]
	if current.SubmissionID != "" || current.SubmittedAt != nil || len(capture.submissions) != 1 {
		t.Fatal("bank reuse manufactured vendor submission")
	}
	if current.Status != AssessmentUnderReview {
		t.Fatalf("all-held review cannot proceed: %s", current.Status)
	}
	if err := review.CheckAssessmentCompletion(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID); !errors.Is(err, ErrAssessmentCompletionBlocked) {
		t.Fatalf("unreviewed reused document allowed completion: %v", err)
	}
	accepted, err := review.ReviewDocument(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, source.ArtifactID, ReviewAssessmentDocumentInput{ExpectedVersion: current.Version, Decision: AssessmentDocumentValidate, DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Assessment.SubmissionID != "" {
		t.Fatal("bank review manufactured vendor submission")
	}
	if err := review.CheckAssessmentCompletion(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID); err != nil {
		t.Fatalf("accepted reused document remains blocked: %v", err)
	}
	if _, err := review.ReconcileCollection(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, "assurance_report", input); !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale reconcile: %v", err)
	}
}

func TestCollectionDocumentReviewDoesNotTreatUnknownOrExpiredAsAccepted(t *testing.T) {
	for _, status := range []AssessmentDocumentStatus{AssessmentDocumentSubmitted, AssessmentDocumentExpired, AssessmentDocumentStatus("UNAVAILABLE")} {
		row := AssessmentCollection{BankPendingCount: 1, Fields: []AssessmentCollectionField{{FieldID: "report", Type: "vendor_document", Required: true, CollectionState: "RECEIVED", BankReviewState: "PENDING"}}}
		answers := map[string]formcontract.AnswerValue{"report": {Document: &formcontract.DocumentAnswer{ArtifactID: "artifact"}}}
		applyCollectionDocumentReviews(&row, "request", answers, []AssessmentDocument{{RequestID: "request", ArtifactID: "artifact", Status: status}, {RequestID: "request", ArtifactID: "artifact", Status: status}}, time.Now())
		if row.AcceptedDocumentCount != 0 || row.BankPendingCount < 0 {
			t.Fatalf("false accepted count for %s: %+v", status, row)
		}
		if status == AssessmentDocumentExpired {
			if !row.Fields[0].VendorActionRequired || row.VendorPendingCount != 1 {
				t.Fatalf("expired document did not require replacement: %+v", row)
			}
		} else if row.Fields[0].BankReviewState != "PENDING" || row.BankPendingCount != 1 {
			t.Fatalf("unknown review cleared pending: %+v", row)
		}
	}
}

func TestCollectionQuarantinedSubmittedDocumentRequiresReplacement(t *testing.T) {
	review, actor, assessment, capture := assessmentReviewFixture(t)
	request := capture.requests[assessment.CurrentRequestID]
	request.LegalEntityID = actor.LegalEntityID
	capture.requests[request.ID] = request
	artifact := capture.artifacts["artifact-1"]
	artifact.Status = evidence.ArtifactQuarantined
	capture.artifacts[artifact.ID] = artifact
	collection, err := review.GetCollection(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range collection.Fields {
		if field.FieldID == "assurance_report" {
			if field.CollectionState != "MISSING" || field.BankReviewState == "VALIDATED" || !field.VendorActionRequired {
				t.Fatalf("quarantined submission looked accepted: %+v", field)
			}
			return
		}
	}
	t.Fatal("assurance requirement missing")
}

func TestReusedDocumentReviewTargetsRequirementWhenArtifactIsShared(t *testing.T) {
	review, actor, assessment, capture := assessmentReviewFixture(t)
	review.assessments.guard = newAssessmentGuard()
	assessment.Status = AssessmentUnderReview
	assessment.SubmissionID = ""
	assessment.SubmittedAt = nil
	at := assessment.UpdatedAt
	assessment.CollectionCompletedAt = &at
	assessment.ReviewStartedAt = &at
	review.assessments.now = func() time.Time { return at }
	repo := review.links.(*MemoryAssessmentRepository)
	repo.assessments[assessment.ID] = assessment
	request := capture.requests[assessment.CurrentRequestID]
	request.LegalEntityID = actor.LegalEntityID
	request.Fields = nil
	source := evidence.DocumentOccurrence{ID: "source", SubmissionChannel: "MAGIC_LINK", ArtifactID: "artifact-1", RequestID: "old-request", ArtifactRequestID: "old-request", SubmissionID: "submission-old", FieldID: "report", RelationshipID: assessment.RelationshipID, FileName: "report.pdf", MediaType: "application/pdf", ArtifactStatus: evidence.ArtifactAvailable, Current: true, SizeBytes: 100, SHA256: "digest"}
	for _, fieldID := range []string{"assurance_report", "resilience_report"} {
		receipt := evidence.CollectionResolution{ID: fieldID, Version: 1, Source: source, SourceArtifactRequestID: source.RequestID, BankReviewState: "PENDING"}
		request.Fields = append(request.Fields, evidence.Field{ID: fieldID, Label: fieldID, Type: "vendor_document", Required: true, CollectionResolution: &receipt})
	}
	capture.requests[request.ID] = request
	capture.artifacts[source.ArtifactID] = evidence.Artifact{ID: source.ArtifactID, TenantID: actor.TenantID, RequestID: source.RequestID, Status: source.ArtifactStatus, SHA256: source.SHA256, SizeBytes: 100}
	ctx := assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID)
	input := ReviewAssessmentDocumentInput{ExpectedVersion: assessment.Version, Decision: AssessmentDocumentValidate, DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated}
	if _, err := review.ReviewDocument(ctx, actor, assessment.ID, source.ArtifactID, input); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ambiguous artifact review accepted: %v", err)
	}
	for _, fieldID := range []string{"assurance_report", "resilience_report"} {
		input.FieldID = fieldID
		result, err := review.ReviewDocument(ctx, actor, assessment.ID, source.ArtifactID, input)
		if err != nil {
			t.Fatal(err)
		}
		input.ExpectedVersion = result.Assessment.Version
	}
	checklist, err := review.GetCollection(ctx, actor, assessment.ID)
	if err != nil || checklist.AcceptedDocumentCount != 2 || checklist.BankPendingCount != 0 {
		t.Fatalf("independent requirement review failed: %+v %v", checklist, err)
	}
}

func (s *assessmentEvidenceStub) Prepare(ctx context.Context, input evidence.WorkflowDistributionDispatchInput) (evidence.WorkflowDistributionDispatch, error) {
	request, err := s.CreateRequest(ctx, input.Request)
	return evidence.WorkflowDistributionDispatch{Request: request}, err
}

func TestPrepareAssessmentRequestCreatesNoInvitationAndSendReusesRequest(t *testing.T) {
	assessments, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessments, mustStartAssessment(t, assessments, relationship))
	capture := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	service, err := NewAssessmentRequestService(assessments, repo, capture, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/respond", "production")
	if err != nil {
		t.Fatal(err)
	}
	input := PrepareAssessmentRequestInput{ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: assessments.now().Add(24 * time.Hour)}
	prepared, err := service.PrepareRequest(assessmentContext(), assessmentActor(), assessment.ID, input)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Assessment.Status != AssessmentReadyToSend || prepared.Assessment.CurrentRequestID == "" || prepared.Invitation != nil || capture.dispatched != 0 {
		t.Fatalf("preparation sent request: %#v", prepared)
	}
	_, err = service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{ExpectedVersion: prepared.Assessment.Version, Audience: input.Audience, Deadline: input.Deadline, InvitationTTLMinutes: 60})
	if err != nil {
		t.Fatal(err)
	}
	if len(capture.created) != 1 {
		t.Fatalf("send duplicated prepared request: %d", len(capture.created))
	}
}
