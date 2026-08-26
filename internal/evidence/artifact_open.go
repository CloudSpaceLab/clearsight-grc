package evidence

import (
	"context"
	"io"
)

// OpenArtifact returns one exact available artifact and its content. Storage
// keys remain internal to the evidence service.
func (s *Service) OpenArtifact(ctx context.Context, tenant, requestID, artifactID string) (Artifact, io.ReadCloser, error) {
	artifact, err := s.GetArtifact(ctx, tenant, requestID, artifactID)
	if err != nil || artifact.Status != ArtifactAvailable || s.store == nil {
		return Artifact{}, nil, ErrNotFound
	}
	reader, err := s.store.Open(ctx, artifact.StorageKey)
	if err != nil {
		return Artifact{}, nil, ErrNotFound
	}
	artifact.StorageKey = ""
	return artifact, reader, nil
}
