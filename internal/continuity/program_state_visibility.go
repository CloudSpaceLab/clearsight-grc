package continuity

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type programMatterVisibility struct {
	OpenCount int
	Revision  string
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
		if value.CurrentState == nil {
			continue
		}
		visible := visibility[value.Program.ID]
		state := programStateForVisibleMatters(*value.CurrentState, visible)
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
	return programStateForVisibleMatters(state, visibility[programID]), nil
}

func programStateForVisibleMatters(state ProgramStateSnapshot, visibility programMatterVisibility) ProgramStateSnapshot {
	if visibility.OpenCount < 0 {
		visibility.OpenCount = 0
	}

	reasons := make([]StateReason, 0, len(state.Reasons)+1)
	for _, reason := range state.Reasons {
		if strings.EqualFold(strings.TrimSpace(reason.Code), "OPEN_MATTERS") {
			continue
		}
		reasons = append(reasons, reason)
	}
	state.OpenMatterCount = visibility.OpenCount
	state.Dimensions.Exception = StateCurrent
	if visibility.OpenCount > 0 {
		state.Dimensions.Exception = StateAtRisk
		reasons = append(reasons, StateReason{Code: "OPEN_MATTERS", Summary: fmt.Sprintf("%d open issue(s) or change(s) affect this program.", visibility.OpenCount)})
	}
	sortStateReasons(reasons)
	state.Reasons = reasons
	state.Overall = chooseOverallState(state.Dimensions)

	// Snapshot identity and trigger metadata are internal projection provenance
	// and can change because of a Matter this actor cannot see. The actor-facing
	// version is a stable fingerprint of visible Program semantics plus the
	// identities/versions of visible linked open Matters. The latter makes an
	// open-to-open visible Matter transition invalidate review state without
	// allowing hidden Matter churn to do the same.
	state.ID = ""
	state.TriggerType = ""
	state.TriggerID = ""
	state.ProjectionVersion = visibleProgramStateVersion(state, visibility.Revision)
	return state
}

func visibleProgramStateVersion(state ProgramStateSnapshot, matterRevision string) int64 {
	payload, _ := json.Marshal(struct {
		ProgramVersion  int64                `json:"program_version"`
		Overall         ProgramState         `json:"overall"`
		Dimensions      ComplianceDimensions `json:"dimensions"`
		Reasons         []StateReason        `json:"reasons"`
		OpenMatterCount int                  `json:"open_matter_count"`
		MatterRevision  string               `json:"matter_revision"`
	}{
		ProgramVersion:  state.ProgramVersion,
		Overall:         state.Overall,
		Dimensions:      state.Dimensions,
		Reasons:         state.Reasons,
		OpenMatterCount: state.OpenMatterCount,
		MatterRevision:  matterRevision,
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

func visibleMatterRevision(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	sort.Strings(tokens)
	return visibleMatterRevisionMaterial(strings.Join(tokens, "\n"))
}

func visibleMatterRevisionMaterial(material string) string {
	if material == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(material))
	return fmt.Sprintf("%x", sum[:16])
}

func (r *MemoryRepository) VisibleProgramMatterVisibility(_ context.Context, tenant string, programIDs []string, principalID string, at *time.Time) (map[string]programMatterVisibility, error) {
	targets := make(map[string]struct{}, len(programIDs))
	for _, programID := range programIDs {
		targets[programID] = struct{}{}
	}
	visibility := make(map[string]programMatterVisibility, len(programIDs))
	revisions := make(map[string][]string, len(programIDs))

	r.mu.RLock()
	defer r.mu.RUnlock()
	if at == nil {
		for _, aggregate := range r.matters[tenant] {
			collectVisibleMatterPrograms(aggregate, aggregate.Matter.UpdatedAt, principalID, targets, visibility, revisions)
		}
		finalizeProgramMatterVisibility(visibility, revisions)
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
		collectVisibleMatterPrograms(historical, activityAt, principalID, targets, visibility, revisions)
	}
	finalizeProgramMatterVisibility(visibility, revisions)
	return visibility, nil
}

func collectVisibleMatterPrograms(aggregate MatterAggregate, activityAt time.Time, principalID string, targets map[string]struct{}, visibility map[string]programMatterVisibility, revisions map[string][]string) {
	if aggregate.Matter.Status == MatterClosed || aggregate.Matter.Status == MatterCancelled || !MatterVisibleTo(aggregate.Matter, principalID) {
		return
	}
	token := aggregate.Matter.ID + ":" + strconv.FormatInt(aggregate.Matter.Version, 10)
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
		revisions[link.ProgramID] = append(revisions[link.ProgramID], token)
	}
}

func finalizeProgramMatterVisibility(visibility map[string]programMatterVisibility, revisions map[string][]string) {
	for programID, tokens := range revisions {
		value := visibility[programID]
		value.Revision = visibleMatterRevision(tokens)
		visibility[programID] = value
	}
}

var _ programMatterVisibilityRepository = (*MemoryRepository)(nil)
