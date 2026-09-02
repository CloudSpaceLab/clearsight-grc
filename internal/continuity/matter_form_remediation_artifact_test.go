package continuity

import (
	"context"
	"errors"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
)

type matterFormArtifactReader struct {
	artifacts map[string]evidence.Artifact
}

func (r matterFormArtifactReader) GetRequestByOrigin(context.Context, string, evidence.RequestOrigin) (evidence.Request, error) {
	return evidence.Request{}, evidence.ErrNotFound
}

func (r matterFormArtifactReader) GetSubmission(context.Context, string, string) (evidence.Submission, error) {
	return evidence.Submission{}, evidence.ErrNotFound
}

func (r matterFormArtifactReader) GetArtifact(_ context.Context, tenant, requestID, artifactID string) (evidence.Artifact, error) {
	artifact, ok := r.artifacts[artifactID]
	if !ok || artifact.TenantID != tenant || artifact.RequestID != requestID {
		return evidence.Artifact{}, evidence.ErrNotFound
	}
	return artifact, nil
}

func TestMappedMatterFormArtifactsMustBeAvailableBeforeApplication(t *testing.T) {
	request := evidence.Request{ID: "request-1", TenantID: "bank"}
	binding := MatterFormRemediationBinding{Mappings: []MatterFormFieldMapping{{FieldID: "certificate", MissingItem: "Certificate", FactKey: "certificate"}}}
	answers := map[string]formcontract.AnswerValue{"certificate": {ArtifactIDs: []string{"artifact-1"}}}

	for _, status := range []evidence.ArtifactStatus{evidence.ArtifactStoredUnscanned, evidence.ArtifactQuarantined} {
		reader := matterFormArtifactReader{artifacts: map[string]evidence.Artifact{"artifact-1": {ID: "artifact-1", TenantID: "bank", RequestID: "request-1", Status: status}}}
		if err := mappedMatterFormArtifactsAvailable(t.Context(), reader, request, binding, answers); !errors.Is(err, ErrMatterFormResponseRejected) {
			t.Fatalf("artifact status %s error = %v", status, err)
		}
	}

	reader := matterFormArtifactReader{artifacts: map[string]evidence.Artifact{"artifact-1": {ID: "artifact-1", TenantID: "bank", RequestID: "request-1", Status: evidence.ArtifactAvailable}}}
	if err := mappedMatterFormArtifactsAvailable(t.Context(), reader, request, binding, answers); err != nil {
		t.Fatalf("available artifact rejected: %v", err)
	}
}
