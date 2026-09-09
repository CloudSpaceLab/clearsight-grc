package thirdparty

import (
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"testing"
)

func configureDemoArtifactsForTest(t *testing.T, target any, enabled bool) {
	t.Helper()
	configurable, ok := target.(interface{ ConfigureDemoUnscannedArtifacts(bool) })
	if !ok {
		t.Fatal("instance does not expose demo artifact policy")
	}
	configurable.ConfigureDemoUnscannedArtifacts(enabled)
}

func TestDemoUnscannedAssessmentDocumentReview(t *testing.T) {
	for _, test := range []struct {
		name                                 string
		enabled, revoked, repositoryDisabled bool
		status                               evidence.ArtifactStatus
		allowed                              bool
	}{
		{name: "default off", status: evidence.ArtifactStoredUnscanned},
		{name: "demo enabled", enabled: true, status: evidence.ArtifactStoredUnscanned, allowed: true},
		{name: "quarantine remains blocked", enabled: true, status: evidence.ArtifactQuarantined},
		{name: "policy revoked", enabled: true, revoked: true, status: evidence.ArtifactStoredUnscanned},
		{name: "repository policy independently off", enabled: true, repositoryDisabled: true, status: evidence.ArtifactStoredUnscanned},
	} {
		t.Run(test.name, func(t *testing.T) {
			service, actor, assessment, reader := assessmentReviewFixture(t)
			prepareAssessmentDocumentReviewFixture(service, assessment)
			repo := service.links.(*MemoryAssessmentRepository)
			if test.enabled {
				configureDemoArtifactsForTest(t, service, true)
				configureDemoArtifactsForTest(t, repo, true)
			}
			if test.revoked {
				configureDemoArtifactsForTest(t, service, false)
				configureDemoArtifactsForTest(t, repo, false)
			}
			if test.repositoryDisabled {
				configureDemoArtifactsForTest(t, repo, false)
			}
			artifact := reader.artifacts["artifact-1"]
			artifact.Status = test.status
			reader.artifacts[artifact.ID] = artifact
			ctx := assessmentContextFor(actor.TenantID, actor.LegalEntityID, actor.PrincipalID)
			view, err := service.ReviewDocument(ctx, actor, assessment.ID, artifact.ID, ReviewAssessmentDocumentInput{ExpectedVersion: assessment.Version, Decision: AssessmentDocumentValidate, DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated})
			if !test.allowed {
				if !errors.Is(err, ErrAssessmentCompletionBlocked) {
					t.Fatalf("review error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if view.Documents[0].ArtifactStatus != evidence.ArtifactStoredUnscanned || !view.Documents[0].DemoUnscannedAllowed {
				t.Fatal("scan status was rewritten")
			}
			if _, err = service.ReviewDocument(ctx, actor, assessment.ID, artifact.ID, ReviewAssessmentDocumentInput{ExpectedVersion: assessment.Version, Decision: AssessmentDocumentValidate, DocumentType: "SOC_2_TYPE_II", EvidenceClass: AssessmentDocumentBankValidated}); !errors.Is(err, ErrVersionConflict) {
				t.Fatalf("stale version review = %v", err)
			}
		})
	}
}
