package continuity

import (
	"context"
	"fmt"
	"strings"
)

type RetireRequirementControlLinkInput struct {
	TenantID        string `json:"tenant_id"`
	ProgramID       string `json:"program_id"`
	LinkID          string `json:"link_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id,omitempty"`
	Rationale       string `json:"rationale"`
}

type RetireMatterLinkInput struct {
	TenantID        string `json:"tenant_id"`
	MatterID        string `json:"matter_id"`
	LinkID          string `json:"link_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id,omitempty"`
	Rationale       string `json:"rationale"`
}

func (s *Service) RetireRequirementControlLink(ctx context.Context, input RetireRequirementControlLinkInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	link, ok := requirementControlLinkByID(aggregate.RequirementControlLinks, strings.TrimSpace(input.LinkID))
	if !ok {
		return ProgramAggregate{}, ErrNotFound
	}
	actorID, rationale := strings.TrimSpace(input.ActorID), strings.TrimSpace(input.Rationale)
	if actorID == "" || rationale == "" {
		return ProgramAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	retiredAt := s.now().UTC()
	link.RetiredAt = &retiredAt
	link.RetiredBy = actorID
	link.RetirementReason = rationale
	if err = s.applyProgramValue(ctx, input.TenantID, input.ProgramID, aggregate.Program.Version, EventRequirementControlLinkRetired, link, actorID); err != nil {
		return ProgramAggregate{}, err
	}
	aggregate.RequirementControlLinks = removeRequirementControlLink(aggregate.RequirementControlLinks, link.ID)
	aggregate.Program.Version++
	aggregate.Program.UpdatedAt = retiredAt
	_ = s.requestProgramRefresh(ctx, input.TenantID, input.ProgramID, EventRequirementControlLinkRetired, link.ID, actorID)
	return decorateProgram(aggregate), nil
}

func (s *Service) RetireMatterLink(ctx context.Context, input RetireMatterLinkInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	link, ok := matterLinkByID(aggregate.Links, strings.TrimSpace(input.LinkID))
	if !ok {
		return MatterAggregate{}, ErrNotFound
	}
	actorID, rationale := strings.TrimSpace(input.ActorID), strings.TrimSpace(input.Rationale)
	if actorID == "" || rationale == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	retiredAt := s.now().UTC()
	link.RetiredAt = &retiredAt
	link.RetiredBy = actorID
	link.RetirementReason = rationale
	if err = s.applyMatterValue(ctx, input.TenantID, input.MatterID, aggregate.Matter.Version, EventMatterLinkRetired, link, actorID); err != nil {
		return MatterAggregate{}, err
	}
	aggregate.Links = removeMatterLink(aggregate.Links, link.ID)
	aggregate.Matter.Version++
	aggregate.Matter.UpdatedAt = retiredAt
	aggregate.Closure = assessClosure(aggregate)
	_ = s.requestProgramRefresh(ctx, input.TenantID, link.ProgramID, EventMatterLinkRetired, input.MatterID, actorID)
	return decorateMatter(aggregate), nil
}

func requirementControlLinkByID(values []RequirementControlLink, linkID string) (RequirementControlLink, bool) {
	for _, value := range values {
		if value.ID == linkID {
			return value, true
		}
	}
	return RequirementControlLink{}, false
}

func matterLinkByID(values []MatterLink, linkID string) (MatterLink, bool) {
	for _, value := range values {
		if value.ID == linkID {
			return value, true
		}
	}
	return MatterLink{}, false
}

func removeRequirementControlLink(values []RequirementControlLink, linkID string) []RequirementControlLink {
	result := make([]RequirementControlLink, 0, len(values))
	for _, value := range values {
		if value.ID != linkID {
			result = append(result, value)
		}
	}
	return result
}

func removeMatterLink(values []MatterLink, linkID string) []MatterLink {
	result := make([]MatterLink, 0, len(values))
	for _, value := range values {
		if value.ID != linkID {
			result = append(result, value)
		}
	}
	return result
}
