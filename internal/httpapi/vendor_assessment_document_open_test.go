package httpapi

import (
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/thirdparty"
)

func TestAssessmentDocumentAvailableRequiresExactCurrentRequestAndAvailableArtifact(t *testing.T) {
	view := thirdparty.AssessmentReviewView{
		Assessment: thirdparty.Assessment{CurrentRequestID: "request-1"},
		Requests:   []thirdparty.AssessmentReviewRequest{{RequestID: "request-1"}},
		Response:   &thirdparty.AssessmentReviewResponse{RequestID: "request-1"},
		Documents:  []thirdparty.AssessmentReviewDocument{{ArtifactID: "artifact-1", ArtifactStatus: evidence.ArtifactAvailable}},
	}
	if !assessmentDocumentAvailable(view, "request-1", "artifact-1") {
		t.Fatal("expected exact artifact to be available")
	}
	if assessmentDocumentAvailable(view, "request-other", "artifact-1") {
		t.Fatal("wrong request must not be available")
	}
	view.Documents[0].ArtifactStatus = evidence.ArtifactQuarantined
	if assessmentDocumentAvailable(view, "request-1", "artifact-1") {
		t.Fatal("quarantined artifact must not be available")
	}
}

func TestAssessmentCollectionDocumentRequiresExactReceipt(t *testing.T) {
	view := thirdparty.AssessmentReviewView{Answers: []thirdparty.AssessmentReviewAnswer{{CollectionResolution: &evidence.CollectionResolution{SourceArtifactRequestID: "original", Source: evidence.DocumentOccurrence{ArtifactID: "report", ArtifactStatus: evidence.ArtifactAvailable}}}}}
	if assessmentCollectionDocument(view, "original", "report") == nil {
		t.Fatal("receipt source unavailable")
	}
	if assessmentCollectionDocument(view, "other", "report") != nil || assessmentCollectionDocument(view, "original", "other") != nil {
		t.Fatal("receipt widened document access")
	}
	view.Answers[0].CollectionResolution.Source.ArtifactStatus = evidence.ArtifactQuarantined
	if assessmentCollectionDocument(view, "original", "report") != nil {
		t.Fatal("quarantined source available")
	}
}
