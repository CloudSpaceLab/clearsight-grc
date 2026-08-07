package continuity

import (
	"context"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

// AddResponsePackageLifecycle is the production response-preparation command.
// The verified actor is retained in the canonical payload as the preparer.
func (s *Service) AddResponsePackageLifecycle(ctx context.Context, input AddResponsePackageInput) (MatterAggregate, error) {
	if _, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion); err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.Purpose) == "" || strings.TrimSpace(input.Audience) == "" || strings.TrimSpace(input.ActorID) == "" {
		return MatterAggregate{}, fmt.Errorf("purpose, audience and verified actor are required")
	}
	manifest, err := normalizedJSON(input.Manifest, `[]`)
	if err != nil {
		return MatterAggregate{}, err
	}
	valueID, err := id.NewUUIDv7()
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	value := ResponsePackage{
		ID: valueID, TenantID: input.TenantID, MatterID: input.MatterID,
		Purpose: strings.TrimSpace(input.Purpose), Audience: strings.TrimSpace(input.Audience),
		Status: ResponseDraft, Manifest: manifest, PreparedBy: strings.TrimSpace(input.ActorID),
		CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if err = s.applyMatterValue(ctx, input.TenantID, input.MatterID, input.ExpectedVersion, EventResponsePackageAdded, value, input.ActorID); err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, input.TenantID, input.MatterID)
}

// TransitionResponseLifecycle persists the verified actor for every response
// lifecycle transition instead of retaining only approval timestamps.
func (s *Service) TransitionResponseLifecycle(ctx context.Context, input TransitionResponseInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	var response *ResponsePackage
	for index := range aggregate.ResponsePackages {
		if aggregate.ResponsePackages[index].ID == input.ResponseID {
			response = &aggregate.ResponsePackages[index]
			break
		}
	}
	if response == nil {
		return MatterAggregate{}, ErrNotFound
	}
	if !allowedResponseTransition(response.Status, input.To) {
		return MatterAggregate{}, ErrInvalidState
	}
	if strings.TrimSpace(input.ActorID) == "" {
		return MatterAggregate{}, fmt.Errorf("verified actor is required for a response lifecycle transition")
	}
	response.Status = input.To
	response.UpdatedAt = s.now().UTC()
	response.Version++
	setResponseActor(response, input.To, input.ActorID)
	switch input.To {
	case ResponseTransmitted:
		value := response.UpdatedAt
		response.TransmittedAt = &value
	case ResponseAcknowledged:
		value := response.UpdatedAt
		response.AcknowledgedAt = &value
	}
	if err = s.applyMatterValue(ctx, input.TenantID, input.MatterID, input.ExpectedVersion, EventResponsePackageStateChanged, *response, input.ActorID); err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, input.TenantID, input.MatterID)
}
