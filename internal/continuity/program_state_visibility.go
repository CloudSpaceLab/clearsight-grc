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

type programMatterVisibilityRepository interface {
	VisibleOpenMatterCounts(context.Context, string, []string, string, *time.Time) (map[string]int, error)
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
// bounded repository call so list reads do not degrade into N+1 Matter counts.
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

	counts := make(map[string]int, len(ids))
	if repo, ok := s.repo.(programMatterVisibilityRepository); ok {
		var err error
		counts, err = repo.VisibleOpenMatterCounts(ctx, tenant, ids, principalID, at)
		if err != nil {
			return nil, err
		}
	}

	result := make([]ProgramAggregate, len(values))
	for index, value := range values {
		result[index] = value
		if value.CurrentState == nil {
			continue
		}
		state := programStateForVisibleMatters(*value.CurrentState, counts[value.Program.ID])
		// Canonical projection time can advance solely because of a Matter the
		// actor cannot see. Program updated_at is canonical actor-visible state,
		// so use it as the stable presentation timestamp instead.
		state.GeneratedAt = value.Program.UpdatedAt.UTC()
		result[index].CurrentState = &state
		result[index] = decorateProgram(result[index])
	}
	return result, nil
}

func (s *Service) programStateForPrincipal(ctx context.Context, tenant, programID, principalID string, state ProgramStateSnapshot, at *time.Time) (ProgramStateSnapshot, error) {
	count := 0
	if repo, ok := s.repo.(programMatterVisibilityRepository); ok {
		counts, err := repo.VisibleOpenMatterCounts(ctx, tenant, []string{programID}, principalID, at)
		if err != nil {
			return ProgramStateSnapshot{}, err
		}
		count = counts[programID]
	}
	return programStateForVisibleMatters(state, count), nil
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
	// version is instead a stable fingerprint of visible semantic state.
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

func (r *MemoryRepository) VisibleOpenMatterCounts(_ context.Context, tenant string, programIDs []string, principalID string, at *time.Time) (map[string]int, error) {
	targets := make(map[string]struct{}, len(programIDs))
	for _, programID := range programIDs {
		targets[programID] = struct{}{}
	}
	counts := make(map[string]int, len(programIDs))

	r.mu.RLock()
	defer r.mu.RUnlock()
	if at == nil {
		for _, aggregate := range r.matters[tenant] {
			countVisibleMatterPrograms(aggregate, principalID, targets, counts)
		}
		return counts, nil
	}

	for _, events := range r.matterEvents[tenant] {
		historical, err := reconstructMatter(filterEvents(events, at))
		if err == ErrNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		countVisibleMatterPrograms(historical, principalID, targets, counts)
	}
	return counts, nil
}

func countVisibleMatterPrograms(aggregate MatterAggregate, principalID string, targets map[string]struct{}, counts map[string]int) {
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
		counts[link.ProgramID]++
	}
}

var _ programMatterVisibilityRepository = (*MemoryRepository)(nil)
