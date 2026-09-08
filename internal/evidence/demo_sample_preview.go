package evidence

import (
	"context"
	"io"

	"github.com/CloudSpaceLab/clearsight-grc/internal/demodocuments"
)

// ConfigureDemoSamplePreview composes the two demo read boundaries at startup.
// Call only with validated non-production DemoMode configuration, before serving.
func ConfigureDemoSamplePreview(capture *Service, distributions *DistributionService, enabled bool) {
	if capture != nil {
		capture.demoSamplePreview = enabled
	}
	if distributions != nil {
		distributions.demoSamplePreview = enabled
	}
}

// OpenDemoSampleArtifact opens only the original stored bytes of an exact
// fictional sample. It neither records a scan nor changes artifact availability.
func (s *Service) OpenDemoSampleArtifact(ctx context.Context, tenant, requestID, artifactID string) (Artifact, io.ReadCloser, error) {
	if s == nil || !s.demoSamplePreview {
		return Artifact{}, nil, ErrNotFound
	}
	artifact, err := s.GetArtifact(ctx, tenant, requestID, artifactID)
	if err != nil || artifact.Status != ArtifactStoredUnscanned || !demodocuments.Matches(artifact.FileName, artifact.MediaType, artifact.SHA256, artifact.SizeBytes) {
		return Artifact{}, nil, ErrNotFound
	}
	return s.openVerifiedArtifact(ctx, artifact)
}
