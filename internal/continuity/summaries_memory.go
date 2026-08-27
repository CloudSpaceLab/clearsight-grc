package continuity

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func (r *MemoryRepository) ListProgramSummaries(ctx context.Context, tenant string, query SummaryQuery) (ProgramSummaryPage, error) {
	cursor, err := decodeProgramSummaryCursor(query.Cursor)
	if err != nil {
		return ProgramSummaryPage{}, err
	}
	search := strings.ToLower(query.Search)
	actor, enforceVisibility := identity.FromContext(ctx)
	if enforceVisibility && actor.TenantID != tenant {
		return ProgramSummaryPage{GeneratedAt: time.Now().UTC()}, nil
	}
	r.mu.RLock()
	values := make([]ProgramSummary, 0, len(r.programs[tenant]))
	for _, aggregate := range r.programs[tenant] {
		if !r.visibleLegalEntity(ctx, aggregate.Program.TenantID, aggregate.Program.LegalEntityID) {
			continue
		}
		aggregate = cloneProgramAggregate(aggregate)
		if enforceVisibility && aggregate.CurrentState != nil {
			visible := r.visibleProgramMatterVisibilityForProgramLocked(tenant, aggregate.Program.ID, actor.PrincipalID)
			state := programStateForVisibleMatters(*aggregate.CurrentState, visible.OpenCount)
			state.GeneratedAt = actorVisibleProgramStateTime(aggregate.Program.UpdatedAt, visible.LatestAt)
			aggregate.CurrentState = &state
		}
		value := summarizeProgram(aggregate)
		if query.Status != "" && string(value.Program.Status) != query.Status {
			continue
		}
		if query.OverallState != "" && string(value.OverallState) != query.OverallState {
			continue
		}
		if query.Jurisdiction != "" && !strings.EqualFold(value.Program.Jurisdiction, query.Jurisdiction) {
			continue
		}
		if query.AssignedToMe && value.Program.OwnerPrincipalID != query.principalID {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(value.Program.Code+" "+value.Program.Name+" "+value.Program.OwningFunction+" "+value.Program.Jurisdiction), search) {
			continue
		}
		values = append(values, value)
	}
	r.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool {
		left, right := values[i], values[j]
		leftRank, rightRank := programStatusRank(left.Program.Status), programStatusRank(right.Program.Status)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if !left.Program.UpdatedAt.Equal(right.Program.UpdatedAt) {
			return left.Program.UpdatedAt.After(right.Program.UpdatedAt)
		}
		return left.Program.ID > right.Program.ID
	})
	if cursor.ID != "" {
		filtered := values[:0]
		for _, value := range values {
			rank := programStatusRank(value.Program.Status)
			if rank > cursor.Rank || (rank == cursor.Rank && (value.Program.UpdatedAt.Before(cursor.UpdatedAt) || (value.Program.UpdatedAt.Equal(cursor.UpdatedAt) && value.Program.ID < cursor.ID))) {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	limit := boundedLimit(query.Limit)
	page := ProgramSummaryPage{GeneratedAt: time.Now().UTC()}
	if len(values) > limit {
		last := values[limit-1]
		page.NextCursor, err = encodeSummaryCursor(programSummaryCursor{Rank: programStatusRank(last.Program.Status), UpdatedAt: last.Program.UpdatedAt, ID: last.Program.ID})
		if err != nil {
			return ProgramSummaryPage{}, err
		}
		values = values[:limit]
	}
	page.Items = values
	return page, nil
}

func (r *MemoryRepository) visibleProgramMatterVisibilityForProgramLocked(tenant, programID, principalID string) programMatterVisibility {
	targets := map[string]struct{}{programID: {}}
	visibility := map[string]programMatterVisibility{}
	for _, aggregate := range r.matters[tenant] {
		collectVisibleMatterPrograms(aggregate, aggregate.Matter.UpdatedAt, principalID, targets, visibility)
	}
	return visibility[programID]
}

func (r *MemoryRepository) ListMatterSummaries(ctx context.Context, tenant string, query SummaryQuery) (MatterSummaryPage, error) {
	cursor, err := decodeMatterSummaryCursor(query.Cursor)
	if err != nil {
		return MatterSummaryPage{}, err
	}
	search := strings.ToLower(query.Search)
	actor, enforceVisibility := identity.FromContext(ctx)
	r.mu.RLock()
	values := make([]MatterSummary, 0, len(r.matters[tenant]))
	for _, aggregate := range r.matters[tenant] {
		if !r.visibleLegalEntity(ctx, aggregate.Matter.TenantID, aggregate.Matter.LegalEntityID) || (enforceVisibility && (aggregate.Matter.TenantID != actor.TenantID || !MatterVisibleTo(aggregate.Matter, actor.PrincipalID))) {
			continue
		}
		if query.Status == "OPEN" && (aggregate.Matter.Status == MatterClosed || aggregate.Matter.Status == MatterCancelled) {
			continue
		}
		if query.Status != "" && query.Status != "OPEN" && string(aggregate.Matter.Status) != query.Status {
			continue
		}
		if query.ProgramID != "" {
			linked := false
			for _, link := range aggregate.Links {
				if link.ProgramID == query.ProgramID {
					linked = true
					break
				}
			}
			if !linked {
				continue
			}
		}
		if query.MatterType != "" && string(aggregate.Matter.Type) != query.MatterType {
			continue
		}
		if query.Priority > 0 && aggregate.Matter.Priority != query.Priority {
			continue
		}
		if query.AssignedToMe && aggregate.Matter.OwnerPrincipalID != query.principalID {
			continue
		}
		if !matterMatchesDueCondition(aggregate.Matter, query.DueCondition, query.asOf) {
			continue
		}
		value := summarizeMatter(cloneMatterAggregate(aggregate))
		if search != "" && !strings.Contains(strings.ToLower(value.Matter.Reference+" "+value.Matter.Title+" "+value.Matter.Summary+" "+value.TypeLabel), search) {
			continue
		}
		values = append(values, value)
	}
	r.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool {
		left, right := values[i].Matter, values[j].Matter
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		return left.ID > right.ID
	})
	if cursor.ID != "" {
		filtered := values[:0]
		for _, value := range values {
			matter := value.Matter
			if matter.Priority < cursor.Priority || (matter.Priority == cursor.Priority && (matter.UpdatedAt.Before(cursor.UpdatedAt) || (matter.UpdatedAt.Equal(cursor.UpdatedAt) && matter.ID < cursor.ID))) {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	limit := boundedLimit(query.Limit)
	page := MatterSummaryPage{GeneratedAt: time.Now().UTC()}
	if len(values) > limit {
		last := values[limit-1].Matter
		page.NextCursor, err = encodeSummaryCursor(matterSummaryCursor{Priority: last.Priority, UpdatedAt: last.UpdatedAt, ID: last.ID})
		if err != nil {
			return MatterSummaryPage{}, err
		}
		values = values[:limit]
	}
	page.Items = values
	return page, nil
}

func matterMatchesDueCondition(matter Matter, condition string, asOf time.Time) bool {
	if condition == "" {
		return true
	}
	if condition == "NO_DUE_DATE" {
		return matter.DueAt == nil
	}
	if matter.DueAt == nil || matter.Status == MatterClosed || matter.Status == MatterCancelled {
		return false
	}
	due := matter.DueAt.UTC()
	switch condition {
	case "OVERDUE":
		return due.Before(asOf)
	case "DUE_7_DAYS":
		return !due.Before(asOf) && !due.After(asOf.AddDate(0, 0, 7))
	case "DUE_30_DAYS":
		return !due.Before(asOf) && !due.After(asOf.AddDate(0, 0, 30))
	default:
		return false
	}
}
