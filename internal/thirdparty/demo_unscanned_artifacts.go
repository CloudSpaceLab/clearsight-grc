package thirdparty

import (
	"context"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"sync/atomic"
)

// demoArtifactPolicy is instance configuration, never a value taken from a
// command or a stored receipt. Its zero value keeps all material checks strict.
type demoArtifactPolicy struct{ demoUnscannedArtifacts atomic.Bool }

func (p *demoArtifactPolicy) ConfigureDemoUnscannedArtifacts(enabled bool) {
	p.demoUnscannedArtifacts.Store(enabled)
}

func (p *demoArtifactPolicy) artifactUseAllowed(status evidence.ArtifactStatus) bool {
	return evidence.ArtifactUseAllowed(status, p.demoUnscannedArtifacts.Load())
}
func (p *demoArtifactPolicy) demoUnscannedAllowed(status evidence.ArtifactStatus) bool {
	return status == evidence.ArtifactStoredUnscanned && p.demoUnscannedArtifacts.Load()
}
func (s *AssessmentReviewService) refreshCollectionArtifacts(ctx context.Context, request evidence.Request) evidence.Request {
	return evidence.RefreshCollectionResolutions(ctx, request, func(ctx context.Context, tenant, requestID, artifactID string) (evidence.Artifact, error) {
		artifact, err := s.evidence.GetArtifact(ctx, tenant, requestID, artifactID)
		artifact.DemoUnscannedAllowed = s.demoUnscannedAllowed(artifact.Status)
		return artifact, err
	}, s.assessments.now())
}
