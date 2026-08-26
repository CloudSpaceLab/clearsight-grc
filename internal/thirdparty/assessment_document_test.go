package thirdparty

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type postCommitAssessmentReviewReadFailure struct {
	*MemoryAssessmentRepository
	failReads bool
}

func (r *postCommitAssessmentReviewReadFailure) ReviewAssessmentDocument(ctx context.Context, input AssessmentDocumentReviewRecord) (AssessmentDocument, Assessment, error) {
	document, assessment, err := r.MemoryAssessmentRepository.ReviewAssessmentDocument(ctx, input)
	if err == nil {
		r.failReads = true
	}
	return document, assessment, err
}

func (r *postCommitAssessmentReviewReadFailure) ListAssessmentRequestLinks(ctx context.Context, scope Scope, assessmentID string) ([]AssessmentRequestLink, error) {
	if r.failReads {
		return nil, errors.New("review projection unavailable")
	}
	return r.MemoryAssessmentRepository.ListAssessmentRequestLinks(ctx, scope, assessmentID)
}

func TestReviewAssessmentDocumentUsesExactCurrentSubmissionAndRecordsMaterialDecision(t *testing.T) {
	service, actor, assessment, _ := assessmentReviewFixture(t)
	prepareAssessmentDocumentReviewFixture(service, assessment)

	view, err := service.ReviewDocument(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, "artifact-1", ReviewAssessmentDocumentInput{
		ExpectedVersion: assessment.Version, Decision: AssessmentDocumentValidate,
		DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated, ValidUntil: "2027-05-31",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Assessment.Version != assessment.Version+1 || len(view.Documents) != 1 {
		t.Fatalf("unexpected refreshed review %#v", view)
	}
	document := view.Documents[0]
	if document.Status != string(AssessmentDocumentValidated) || document.EvidenceClass != AssessmentEvidenceClass(AssessmentDocumentBankValidated) || document.Reference != "SOC2-2026" || document.IssuedBy != "Independent auditor" || document.IssuedOn != "2026-06-01" || document.ExpiresOn != "2027-05-31" {
		t.Fatalf("review decision did not retain typed submitted metadata: %#v", document)
	}
	repository := service.links.(*MemoryAssessmentRepository)
	stored := repository.assessmentDocuments[assessment.ID]["artifact-1"]
	if stored.Version != 1 || stored.ValidatedByPrincipalID != actor.PrincipalID || stored.Status != AssessmentDocumentValidated {
		t.Fatalf("unexpected authoritative document %#v", stored)
	}
	if event := repository.assessmentEvents[len(repository.assessmentEvents)-1]; event.Type != "AssessmentDocumentValidated" || event.ActorPrincipalID != actor.PrincipalID || event.AssessmentVersion != assessment.Version+1 {
		t.Fatalf("material document event was not recorded with the assessment version: %#v", event)
	}
	if event := repository.assessmentOutbox[len(repository.assessmentOutbox)-1]; event.Type != "AssessmentDocumentValidated" || event.ActorPrincipalID != "" {
		t.Fatalf("safe document outbox event was not recorded: %#v", event)
	}
}

func TestReviewAssessmentDocumentReturnsCommittedDecisionWhenReviewRefreshFails(t *testing.T) {
	service, actor, assessment, _ := assessmentReviewFixture(t)
	prepareAssessmentDocumentReviewFixture(service, assessment)
	base := service.links.(*MemoryAssessmentRepository)
	service.links = &postCommitAssessmentReviewReadFailure{MemoryAssessmentRepository: base}

	view, err := service.ReviewDocument(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, "artifact-1", ReviewAssessmentDocumentInput{
		ExpectedVersion: assessment.Version, Decision: AssessmentDocumentReject,
		DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentVendorSupplied,
	})
	if err != nil {
		t.Fatalf("committed document review returned an error: %v", err)
	}
	if view.Assessment.Version != assessment.Version+1 || view.Documents[0].Status != string(AssessmentDocumentRejected) {
		t.Fatalf("fallback review = %#v", view)
	}
}

func TestReviewAssessmentDocumentFailsClosedForWrongArtifactScopeAndUnavailableValidation(t *testing.T) {
	t.Run("artifact from another request", func(t *testing.T) {
		service, actor, assessment, reader := assessmentReviewFixture(t)
		prepareAssessmentDocumentReviewFixture(service, assessment)
		reader.artifacts["other-artifact"] = evidence.Artifact{ID: "other-artifact", TenantID: actor.TenantID, RequestID: "other-request", SubmissionID: assessment.SubmissionID, FileName: "other.pdf", MediaType: "application/pdf", SizeBytes: 10, Status: evidence.ArtifactAvailable}
		_, err := service.ReviewDocument(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, "other-artifact", ReviewAssessmentDocumentInput{
			ExpectedVersion: assessment.Version, Decision: AssessmentDocumentValidate, DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated,
		})
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected exact current request scope failure, got %v", err)
		}
	})

	t.Run("artifact is quarantined", func(t *testing.T) {
		service, actor, assessment, reader := assessmentReviewFixture(t)
		prepareAssessmentDocumentReviewFixture(service, assessment)
		artifact := reader.artifacts["artifact-1"]
		artifact.Status = evidence.ArtifactQuarantined
		reader.artifacts[artifact.ID] = artifact
		_, err := service.ReviewDocument(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, artifact.ID, ReviewAssessmentDocumentInput{
			ExpectedVersion: assessment.Version, Decision: AssessmentDocumentValidate, DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated,
		})
		if !errors.Is(err, ErrAssessmentCompletionBlocked) {
			t.Fatalf("expected unavailable artifact validation to be blocked, got %v", err)
		}
	})
}

func TestAssessmentCompletionFailsClosedUntilRequiredResponseAndDocumentReviewAreResolved(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*AssessmentReviewService, *assessmentReviewEvidenceStub)
	}{
		{name: "required answer missing", mutate: func(_ *AssessmentReviewService, reader *assessmentReviewEvidenceStub) {
			submission := reader.submissions["submission-1"]
			delete(submission.Answers, "access_control")
			delete(submission.AnswerProvenance, "access_control")
			reader.submissions[submission.ID] = submission
		}},
		{name: "artifact not available", mutate: func(_ *AssessmentReviewService, reader *assessmentReviewEvidenceStub) {
			artifact := reader.artifacts["artifact-2"]
			artifact.Status = evidence.ArtifactQuarantined
			reader.artifacts[artifact.ID] = artifact
		}},
		{name: "required vendor document unreviewed", mutate: func(_ *AssessmentReviewService, _ *assessmentReviewEvidenceStub) {}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, actor, assessment, reader := assessmentReviewFixture(t)
			prepareAssessmentDocumentReviewFixture(service, assessment)
			test.mutate(service, reader)
			service.assessments.ConfigureCompletionReadiness(service)
			_, err := service.assessments.CompleteAssessment(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, CompleteAssessmentInput{
				ExpectedVersion: assessment.Version, Conclusion: AssessmentUnsatisfactory, Rationale: "The required review conditions are not resolved.",
			})
			if !errors.Is(err, ErrAssessmentCompletionBlocked) {
				t.Fatalf("expected completion readiness block, got %v", err)
			}
		})
	}
}

func TestAssessmentCompletionAllowsReviewedRequiredDocumentWithoutChangingRelationship(t *testing.T) {
	service, actor, assessment, reader := assessmentReviewFixture(t)
	prepareAssessmentDocumentReviewFixture(service, assessment)
	artifact := reader.artifacts["artifact-2"]
	artifact.Status = evidence.ArtifactAvailable
	reader.artifacts[artifact.ID] = artifact
	view, err := service.ReviewDocument(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, "artifact-1", ReviewAssessmentDocumentInput{
		ExpectedVersion: assessment.Version, Decision: AssessmentDocumentReject,
		DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentVendorSupplied,
	})
	if err != nil {
		t.Fatal(err)
	}
	service.assessments.ConfigureCompletionReadiness(service)
	before := service.links.(*MemoryAssessmentRepository).relationships[assessment.RelationshipID]
	completed, err := service.assessments.CompleteAssessment(assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID), actor, assessment.ID, CompleteAssessmentInput{
		ExpectedVersion: view.Assessment.Version, Conclusion: AssessmentUnsatisfactory, Rationale: "The submitted report was reviewed and rejected as insufficient.",
	})
	if err != nil {
		t.Fatal(err)
	}
	after := service.links.(*MemoryAssessmentRepository).relationships[assessment.RelationshipID]
	if completed.Status != AssessmentCompleted || before != after {
		t.Fatalf("assessment completion changed relationship state: completed=%#v before=%#v after=%#v", completed, before, after)
	}
}

func prepareAssessmentDocumentReviewFixture(service *AssessmentReviewService, assessment Assessment) {
	repository := service.links.(*MemoryAssessmentRepository)
	current := repository.assessments[assessment.ID]
	current.Status = AssessmentUnderReview
	current.CompletedAt = nil
	current.Conclusion = ""
	current.ConclusionRationale = ""
	current.ConclusionUncertainty = ""
	repository.assessments[assessment.ID] = current
	service.assessments.guard = newAssessmentGuard()
	service.assessments.now = func() time.Time { return time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC) }
}
