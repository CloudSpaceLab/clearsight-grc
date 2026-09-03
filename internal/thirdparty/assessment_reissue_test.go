package thirdparty

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestReissueAssessmentRequestKeepsPriorMagicLinkUntilItsExpiry(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	capture := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	capture.resumeReplacement = true
	service, err := NewAssessmentRequestService(assessmentService, repo, capture, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/capture", "production")
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ReissueRequest(assessmentContext(), assessmentActor(), assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: first.Assessment.Version, Audience: "security@vendor.example", InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(capture.revoked) != 0 {
		t.Fatalf("reissue silently revoked prior magic links: %#v", capture.revoked)
	}
	if second.Invitation == nil || second.Invitation.InvitationID != "invitation-2" || !strings.Contains(second.CaptureURL, "#form_access=replacement-token") {
		t.Fatalf("canonical replacement = %#v", second)
	}
	link, err := repo.GetCurrentAssessmentRequestLink(context.Background(), scopeFromVerified(), assessment.ID)
	if err != nil || link.InvitationID != "invitation-2" || link.RequestID != first.Request.ID {
		t.Fatalf("stored replacement link = (%#v, %v)", link, err)
	}
}

func TestReissueAssessmentRequestDoesNotDependOnPriorMagicLinkRevocation(t *testing.T) {
	assessmentService, repo, relationship := newAssessmentServiceFixture(t, newAssessmentGuard())
	assessment := mustReadyAssessment(t, assessmentService, mustStartAssessment(t, assessmentService, relationship))
	capture := &assessmentEvidenceStub{repo: repo, assessmentID: assessment.ID}
	capture.resumeReplacement = true
	service, err := NewAssessmentRequestService(assessmentService, repo, capture, assessmentFormReaderStub{form: activeAssessmentForm()}, nil, "https://capture.example.test/capture", "production")
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.SendRequest(assessmentContext(), assessmentActor(), assessment.ID, SendAssessmentRequestInput{
		ExpectedVersion: assessment.Version, Audience: "security@vendor.example", Deadline: time.Date(2026, 9, 8, 10, 0, 0, 0, time.UTC), InvitationTTLMinutes: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	capture.revokeErr = evidence.ErrDistributionAccessUnavailable
	if _, err := service.ReissueRequest(assessmentContext(), assessmentActor(), assessment.ID, ReissueAssessmentRequestInput{
		ExpectedVersion: first.Assessment.Version, Audience: "security@vendor.example", InvitationTTLMinutes: 60,
	}); err != nil {
		t.Fatalf("reissue depended on revoking the prior unexpired magic link: %v", err)
	}
	if len(capture.revoked) != 0 {
		t.Fatalf("reissue called route revocation: %#v", capture.revoked)
	}
}
