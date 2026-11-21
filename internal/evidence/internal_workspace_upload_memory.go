package evidence

import (
	"context"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (r *MemoryRepository) createInternalWorkspaceArtifact(ctx context.Context, artifact Artifact) (Artifact, error) {
	actor, err := identity.Require(ctx)
	if err != nil || r.collectionCurrency == nil || actor.PrincipalID != artifact.CreatedBy || actor.TenantID != artifact.TenantID || artifact.SubmissionID != "" {
		return Artifact{}, ErrWorkspaceUnavailable
	}
	distributions := r.collectionCurrency
	distributions.mu.Lock()
	defer distributions.mu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	commitTime := time.Now().UTC()
	request, workspace, recipient, ok := internalMemoryAccessLocked(ctx, distributions, actor, artifact.RequestID, commitTime)
	if !ok {
		return Artifact{}, ErrWorkspaceUnavailable
	}
	route, err := authorizeInternalWorkspaceLocked(ctx, actor, request)
	if err != nil {
		return Artifact{}, err
	}
	created, err := r.createArtifactLocked(artifact)
	if err != nil {
		return Artifact{}, err
	}
	event := distributionEvent{DistributionID: workspace.DistributionID, Version: workspace.Version, EventType: fmt.Sprintf("FORM_RESPONSE_DOCUMENT_UPLOADED_%s", artifact.ID), ActorID: actor.PrincipalID, OccurredAt: commitTime, Payload: map[string]any{"authority_route": route, "artifact_id": artifact.ID, "request_id": request.ID, "recipient_id": recipient.ID}}
	distributions.events = append(distributions.events, event)
	distributions.outbox = append(distributions.outbox, event)
	return created, nil
}
