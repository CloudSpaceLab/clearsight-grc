package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
)

// OpenArtifact returns one exact available artifact and its content. Storage
// keys remain internal to the evidence service.
func (s *Service) OpenArtifact(ctx context.Context, tenant, requestID, artifactID string) (Artifact, io.ReadCloser, error) {
	artifact, err := s.GetArtifact(ctx, tenant, requestID, artifactID)
	if err != nil || artifact.Status != ArtifactAvailable {
		return Artifact{}, nil, ErrNotFound
	}
	return s.openVerifiedArtifact(ctx, artifact)
}

func (s *Service) openVerifiedArtifact(ctx context.Context, artifact Artifact) (Artifact, io.ReadCloser, error) {
	if s.store == nil || artifact.SizeBytes < 1 || artifact.SizeBytes > s.maxArtifactBytes {
		return Artifact{}, nil, ErrNotFound
	}
	reader, err := s.store.Open(ctx, artifact.StorageKey)
	if err != nil {
		return Artifact{}, nil, ErrNotFound
	}
	defer reader.Close()
	// The development store can be overwritten. Buffer within the configured
	// upload bound and verify the complete object before exposing any bytes.
	content, err := io.ReadAll(io.LimitReader(reader, artifact.SizeBytes+1))
	if err != nil || ctx.Err() != nil || int64(len(content)) != artifact.SizeBytes {
		return Artifact{}, nil, ErrNotFound
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != artifact.SHA256 {
		return Artifact{}, nil, ErrNotFound
	}
	artifact.StorageKey = ""
	return artifact, io.NopCloser(bytes.NewReader(content)), nil
}
