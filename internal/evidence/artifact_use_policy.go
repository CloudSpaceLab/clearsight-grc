package evidence

// ArtifactUseAllowed does not imply a clean scan or an evidence review decision.
// The exception must come from validated, non-production demo configuration.
func ArtifactUseAllowed(status ArtifactStatus, allowUnscanned bool) bool {
	return status == ArtifactAvailable || (allowUnscanned && status == ArtifactStoredUnscanned)
}

func withArtifactUsePolicy(artifact Artifact, enabled bool) Artifact {
	artifact.DemoUnscannedAllowed = enabled && artifact.Status == ArtifactStoredUnscanned
	return artifact
}

// ConfigureDemoUnscannedArtifacts is a startup-only setting. Call only with
// validated non-production demo configuration, before serving requests.
func (s *Service) ConfigureDemoUnscannedArtifacts(enabled bool) {
	s.demoUnscannedAllowed = enabled
	if repository, ok := s.repo.(interface{ ConfigureDemoUnscannedArtifacts(bool) }); ok {
		repository.ConfigureDemoUnscannedArtifacts(enabled)
	}
}

func (s *DistributionService) ConfigureDemoUnscannedArtifacts(enabled bool) {
	s.demoUnscannedAllowed = enabled
}

func (r *MemoryRepository) ConfigureDemoUnscannedArtifacts(enabled bool) {
	r.demoUnscannedAllowed = enabled
}
