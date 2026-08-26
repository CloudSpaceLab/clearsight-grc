package thirdparty

import (
	"context"
	"time"
)

func (r *MemoryAssessmentRepository) RelationshipExists(ctx context.Context, scope Scope, relationshipID string) (bool, error) {
	if _, err := r.GetRelationship(ctx, scope, relationshipID); err != nil {
		if err == ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *MemoryAssessmentRepository) TargetAvailable(ctx context.Context, scope Scope, targetType LinkTargetType, targetID string) (bool, error) {
	return r.relationshipLinkRepo.TargetAvailable(ctx, scope, targetType, targetID)
}

func (r *MemoryAssessmentRepository) CreateRelationshipLink(ctx context.Context, value RelationshipLink) (RelationshipLink, error) {
	return r.relationshipLinkRepo.CreateRelationshipLink(ctx, value)
}

func (r *MemoryAssessmentRepository) GetRelationshipLink(ctx context.Context, scope Scope, linkID string) (RelationshipLink, error) {
	return r.relationshipLinkRepo.GetRelationshipLink(ctx, scope, linkID)
}

func (r *MemoryAssessmentRepository) EndRelationshipLink(ctx context.Context, scope Scope, linkID string, expectedVersion int64, actorID, reason string, at time.Time) (RelationshipLink, error) {
	return r.relationshipLinkRepo.EndRelationshipLink(ctx, scope, linkID, expectedVersion, actorID, reason, at)
}

func (r *MemoryAssessmentRepository) ListRelationshipLinks(ctx context.Context, scope Scope, input RelationshipLinkListInput) (RelationshipLinkPage, error) {
	return r.relationshipLinkRepo.ListRelationshipLinks(ctx, scope, input)
}

var _ RelationshipLinkRepository = (*MemoryAssessmentRepository)(nil)
