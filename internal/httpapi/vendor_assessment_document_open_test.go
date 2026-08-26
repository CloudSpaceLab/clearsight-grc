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
