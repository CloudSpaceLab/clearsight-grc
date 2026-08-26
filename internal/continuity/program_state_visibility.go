package continuity

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type programMatterVisibility struct {
	OpenCount int
	LatestAt  time.Time
}

type programMatterVisibilityRepository interface {
	VisibleProgramMatterVisibility(context.Context, string, []string, string, *time.Time) (map[string]programMatterVisibility, error)
}

// ProgramForPrincipal returns an actor-facing Program projection without
// changing canonical Program state. Matter-derived status is rebuilt from only
// the open Matters visible to the supplied principal; every non-Matter state
// dimension remains unchanged.
func (s *Service) ProgramForPrincipal(ctx context.Context, value ProgramAggregate, principalID string, at *time.Time) (ProgramAggregate, error) {
	values, err := s.ProgramsForPrincipal(ctx, []ProgramAggregate{value}, principalID, at)
	if err != nil {
		return ProgramAggregate{}, err
	}
	if len(values) != 1 {
		return ProgramAggregate{}, ErrNotFound
	}
	return values[0], nil
}

// ProgramsForPrincipal applies the same actor-safe Program projection in one
// bounded repository call so list reads do not degrade into N+1 Matter reads.
func (s *Service) ProgramsForPrincipal(ctx context.Context, values []ProgramAggregate, principalID string, at *time.Time) ([]ProgramAggregate, error) {
	principalID = strings.TrimSpace(principalID)
	if principalID == "" {
		return nil, fmt.Errorf("principal_id is required")
	}
	if len(values) == 0 {
		return []ProgramAggregate{}, nil
	}

	tenant := strings.TrimSpace(values[0].Program.TenantID)
	if tenant == "" {
		return nil, fmt.Errorf("Program tenant_id is required")
	}
	ids := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value.Program.TenantID) != tenant || strings.TrimSpace(value.Program.ID) == "" {
			return nil, fmt.Errorf("Programs must belong to one tenant and have ids")
		}
		ids = append(ids, value.Program.ID)
	}

	repo, ok := s.repo.(programMatterVisibilityRepository)
	if !ok {
		return nil, fmt.Errorf("Program Matter visibility is unavailable")
	}
	visibility, err := repo.VisibleProgramMatterVisibility(ctx, tenant, ids, principalID, at)
	if err != nil {
		return nil, err
	}

	result := make([]ProgramAggregate, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Triggers = programTriggersForPrincipal(value.Triggers, principalID)
		if value.CurrentState == nil {
			continue
		}
		visible := visibility[value.Program.ID]
		state := programStateForVisibleMatters(*value.CurrentState, visible.OpenCount)
		state.GeneratedAt = actorVisibleProgramStateTime(value.Program.UpdatedAt, visible.LatestAt)
		result[index].CurrentState = &state
		result[index] = decorateProgram(result[index])
	}
	return result, nil
}

func (s *Service) programStateForPrincipal(ctx context.Context, tenant, programID, principalID string, state ProgramStateSnapshot, at *time.Time) (ProgramStateSnapshot, error) {
	repo, ok := s.repo.(programMatterVisibilityRepository)
	if !ok {
		return ProgramStateSnapshot{}, fmt.Errorf("Program Matter visibility is unavailable")
	}
	visibility, err := repo.VisibleProgramMatterVisibility(ctx, tenant, []string{programID}, principalID, at)
	if err != nil {
		return ProgramStateSnapshot{}, err
	}
	return programStateForVisibleMatters(state, visibility[programID].OpenCount), nil
}

func programStateForVisibleMatters(state ProgramStateSnapshot, visibleOpenMatters int) ProgramStateSnapshot {
	if visibleOpenMatters < 0 {
		visibleOpenMatters = 0
	}

	reasons := make([]StateReason, 0, len(state.Reasons)+1)
	for _, reason := range state.Reasons {
		if strings.EqualFold(strings.TrimSpace(reason.Code), "OPEN_MATTERS") {
			continue
		}
		reasons = append(reasons, reason)
	}
	state.OpenMatterCount = visibleOpenMatters
	state.Dimensions.Exception = StateCurrent
	if visibleOpenMatters > 0 {
		state.Dimensions.Exception = StateAtRisk
		reasons = append(reasons, StateReason{Code: "OPEN_MATTERS", Summary: fmt.Sprintf("%d open issue(s) or change(s) affect this program.", visibleOpenMatters)})
	}
	sortStateReasons(reasons)
	state.Reasons = reasons
	state.Overall = chooseOverallState(state.Dimensions)

	// Snapshot identity and trigger metadata are internal projection provenance
	// and can change because of a Matter this actor cannot see. The actor-facing
	// version is instead a stable fingerprint of visible Program state.
	state.ID = ""
	state.TriggerType = ""
	state.TriggerID = ""
	state.ProjectionVersion = visibleProgramStateVersion(state)
	return state
}

func visibleProgramStateVersion(state ProgramStateSnapshot) int64 {
	payload, _ := json.Marshal(struct {
		ProgramVersion  int64                `json:"program_version"`
		Overall         ProgramState         `json:"overall"`
		Dimensions      ComplianceDimensions `json:"dimensions"`
		Reasons         []StateReason        `json:"reasons"`
		OpenMatterCount int                  `json:"open_matter_count"`
	}{
		ProgramVersion:  state.ProgramVersion,
		Overall:         state.Overall,
		Dimensions:      state.Dimensions,
		Reasons:         state.Reasons,
		OpenMatterCount: state.OpenMatterCount,
	})
	sum := sha256.Sum256(payload)
	// Keep this within JavaScript Number.MAX_SAFE_INTEGER so the browser can
	// round-trip optimistic review versions without precision loss.
	version := int64(binary.BigEndian.Uint64(sum[:8]) & 0x001fffffffffffff)
	if version == 0 {
		return 1
	}
	return version
}

func actorVisibleProgramStateTime(programUpdatedAt, latestVisibleMatterAt time.Time) time.Time {
	value := programUpdatedAt.UTC()
	if latestVisibleMatterAt.After(value) {
		value = latestVisibleMatterAt.UTC()
	}
	return value
}

func programTriggersForPrincipal(values []Trigger, principalID string) []Trigger {
	if len(values) == 0 {
		return []Trigger{}
	}
	result := make([]Trigger, len(values))
	for index, value := range values {
		result[index] = programTriggerForPrincipal(value, principalID)
	}
	return result
}

func programTriggerForPrincipal(value Trigger, principalID string) Trigger {
	_, _, _, createsMatter := matterForTrigger(value)
	if !createsMatter || MatterVisibleTo(Matter{TenantID: value.TenantID, Scope: value.Payload}, principalID) {
		value.Payload = append(json.RawMessage(nil), value.Payload...)
		return value
	}

	// A material Program trigger remains a Program-level fact, but its payload
	// is also used as the generated Matter's scope. Preserve the event type/time
	// and source while suppressing fields that can disclose the restricted case.
	value.Payload = json.RawMessage(`{}`)
	value.DedupeKey = ""
	value.SubjectType = ""
	value.SubjectID = ""
	value.ActorID = ""
	return value
}

func programReviewEventsForPrincipal(events []Event, principalID string) ([]Event, error) {
	result := append([]Event(nil), events...)
	for index := range result {
		if result[index].Type != EventProgramTriggerRecorded {
			continue
		}
		var trigger Trigger
		if err := json.Unmarshal(result[index].Payload, &trigger); err != nil {
			return nil, fmt.Errorf("decode Program trigger for actor projection: %w", err)
		}
		trigger = programTriggerForPrincipal(trigger, principalID)
		payload, err := json.Marshal(trigger)
		if err != nil {
			return nil, fmt.Errorf("encode Program trigger for actor projection: %w", err)
		}
		result[index].Payload = payload
	}
	return result, nil
}

func (r *MemoryRepository) VisibleProgramMatterVisibility(_ context.Context, tenant string, programIDs []string, principalID string, at *time.Time) (map[string]programMatterVisibility, error) {
	targets := make(map[string]struct{}, len(programIDs))
	for _, programID := range programIDs {
		targets[programID] = struct{}{}
	}
	visibility := make(map[string]programMatterVisibility, len(programIDs))

	r.mu.RLock()
	defer r.mu.RUnlock()
	if at == nil {
		for _, aggregate := range r.matters[tenant] {
			collectVisibleMatterPrograms(aggregate, aggregate.Matter.UpdatedAt, principalID, targets, visibility)
		}
		return visibility, nil
	}

	for _, events := range r.matterEvents[tenant] {
		filtered := filterEvents(events, at)
		historical, err := reconstructMatter(filtered)
		if err == ErrNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		activityAt := historical.Matter.UpdatedAt
		if len(filtered) > 0 {
			activityAt = filtered[len(filtered)-1].OccurredAt
		}
		collectVisibleMatterPrograms(historical, activityAt, principalID, targets, visibility)
	}
	return visibility, nil
}

func collectVisibleMatterPrograms(aggregate MatterAggregate, activityAt time.Time, principalID string, targets map[string]struct{}, visibility map[string]programMatterVisibility) {
	if aggregate.Matter.Status == MatterClosed || aggregate.Matter.Status == MatterCancelled || !MatterVisibleTo(aggregate.Matter, principalID) {
		return
	}
	seen := make(map[string]struct{})
	for _, link := range aggregate.Links {
		if _, wanted := targets[link.ProgramID]; !wanted {
			continue
		}
		if _, duplicate := seen[link.ProgramID]; duplicate {
			continue
		}
		seen[link.ProgramID] = struct{}{}
		value := visibility[link.ProgramID]
		value.OpenCount++
		if activityAt.After(value.LatestAt) {
			value.LatestAt = activityAt.UTC()
		}
		visibility[link.ProgramID] = value
	}
}

var _ programMatterVisibilityRepository = (*MemoryRepository)(nil)
