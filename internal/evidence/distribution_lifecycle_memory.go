package evidence

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

func (s *MemoryDistributionStore) AmendDistribution(_ context.Context, tenantID, legalEntityID, distributionID string, input AmendDistributionInput, now time.Time) (AmendDistributionResult, error) {
	if s == nil || s.repo == nil {
		return AmendDistributionResult{}, ErrDistributionInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repo.mu.Lock()
	defer s.repo.mu.Unlock()

	current, ok := s.distributions[distributionID]
	if !ok || current.TenantID != tenantID || current.LegalEntityID != legalEntityID {
		return AmendDistributionResult{}, ErrNotFound
	}
	if current.Version != input.ExpectedVersion {
		return AmendDistributionResult{}, ErrDistributionConflict
	}
	if current.Status == DistributionRevoked || current.Status == DistributionCompleted || current.Status == DistributionSuperseded {
		return AmendDistributionResult{}, ErrDistributionConflict
	}

	next := cloneDistribution(current)
	impact := DistributionImpact{CurrentVersion: current.Version, NextVersion: current.Version + 1, EffectiveDeadline: current.Deadline, EffectiveRouteExpiry: current.RouteExpiresAt, AffectedRecipients: len(s.recipients[distributionID])}
	if input.Deadline != nil {
		deadline := input.Deadline.UTC()
		if !deadline.After(now) {
			return AmendDistributionResult{}, fmt.Errorf("%w: deadline must be in the future", ErrDistributionInvalid)
		}
		next.Deadline = deadline
		impact.DeadlineChanged = !deadline.Equal(current.Deadline)
	}
	if input.RouteExpiresAt != nil {
		expiresAt := input.RouteExpiresAt.UTC()
		if !expiresAt.After(now) {
			return AmendDistributionResult{}, fmt.Errorf("%w: route expiry must be in the future", ErrDistributionInvalid)
		}
		next.RouteExpiresAt = expiresAt
		impact.RouteExpiryChanged = !expiresAt.Equal(current.RouteExpiresAt)
	}
	if next.RouteExpiresAt.After(next.Deadline) {
		next.RouteExpiresAt = next.Deadline
		impact.RouteExpiryChanged = true
	}
	if input.ReminderPolicy != nil {
		policy := cloneAnyMap(*input.ReminderPolicy)
		impact.ReminderPolicyChanged = !reflect.DeepEqual(policy, current.ReminderPolicy)
		next.ReminderPolicy = policy
	}
	if !impact.DeadlineChanged && !impact.RouteExpiryChanged && !impact.ReminderPolicyChanged {
		return AmendDistributionResult{Bundle: bundleFromMemory(current, s.recipients[distributionID], s.workspaces[distributionID]), Impact: impact}, nil
	}

	next.Version++
	next.UpdatedAt = now.UTC()
	impact.EffectiveDeadline = next.Deadline
	impact.EffectiveRouteExpiry = next.RouteExpiresAt
	s.distributions[distributionID] = cloneDistribution(next)
	for _, recipient := range s.recipients[distributionID] {
		if recipient.safe.RequestID == "" {
			continue
		}
		request, exists := s.repo.requests[recipient.safe.RequestID]
		if !exists {
			continue
		}
		request.Deadline = next.Deadline
		request.Version++
		request.UpdatedAt = now.UTC()
		s.repo.requests[request.ID] = request
	}
	event := distributionEvent{DistributionID: next.ID, Version: next.Version, EventType: "FORM_DISTRIBUTION_AMENDED", ActorID: next.CreatedBy, OccurredAt: now.UTC()}
	s.events = append(s.events, event)
	s.outbox = append(s.outbox, event)
	return AmendDistributionResult{Bundle: bundleFromMemory(next, s.recipients[distributionID], s.workspaces[distributionID]), Impact: impact}, nil
}

func (s *MemoryDistributionStore) TransitionDistribution(_ context.Context, tenantID, legalEntityID, distributionID string, input TransitionDistributionInput, now time.Time) (DistributionBundle, error) {
	if s == nil || s.repo == nil {
		return DistributionBundle{}, ErrDistributionInvalid
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repo.mu.Lock()
	defer s.repo.mu.Unlock()
	current, ok := s.distributions[distributionID]
	if !ok || current.TenantID != tenantID || current.LegalEntityID != legalEntityID {
		return DistributionBundle{}, ErrNotFound
	}
	if current.Version != input.ExpectedVersion || !validDistributionTransition(current.Status, input.To, now.UTC(), current.Deadline) {
		return DistributionBundle{}, ErrDistributionConflict
	}
	workspace := s.workspaces[distributionID]
	current.Status = input.To
	current.Version++
	current.UpdatedAt = now.UTC()
	switch input.To {
	case DistributionLocked:
		workspace.Status = ResponseWorkspaceLocked
	case DistributionOpen:
		workspace.Status = ResponseWorkspaceOpen
	case DistributionRevoked:
		workspace.Status = ResponseWorkspaceRevoked
		recipients := s.recipients[distributionID]
		for index := range recipients {
			recipients[index].safe.State = DistributionRecipientRevoked
			recipients[index].safe.Version++
			recipients[index].safe.UpdatedAt = now.UTC()
			if requestID := recipients[index].safe.RequestID; requestID != "" {
				request := s.repo.requests[requestID]
				request.Status = RequestCancelled
				request.Version++
				request.UpdatedAt = now.UTC()
				s.repo.requests[requestID] = request
			}
		}
		s.recipients[distributionID] = recipients
	}
	workspace.Version++
	workspace.UpdatedAt = now.UTC()
	s.distributions[distributionID] = current
	s.workspaces[distributionID] = workspace
	event := distributionEvent{DistributionID: current.ID, Version: current.Version, EventType: "FORM_DISTRIBUTION_" + string(input.To), ActorID: input.ActorID, OccurredAt: now.UTC()}
	s.events = append(s.events, event)
	s.outbox = append(s.outbox, event)
	return bundleFromMemory(current, s.recipients[distributionID], workspace), nil
}
