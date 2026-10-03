package oversight

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var (
	ErrGroupUnavailable = errors.New("group oversight is unavailable")
	ErrGroupForbidden   = errors.New("group oversight is not authorized")
)

type GroupProjectionRepository interface {
	LatestGroup(context.Context, string) (GroupProjection, error)
}

type GroupService struct {
	repository GroupProjectionRepository
	access     access.LegalEntityAccessResolver
	Now        func() time.Time
	StaleAfter time.Duration
}

func NewGroupService(repository GroupProjectionRepository, resolver access.LegalEntityAccessResolver) *GroupService {
	return &GroupService{repository: repository, access: resolver, Now: time.Now, StaleAfter: 15 * time.Minute}
}

func (s *GroupService) Get(ctx context.Context, actor identity.Actor) (GroupSnapshot, error) {
	if s == nil || s.repository == nil || s.access == nil || ctx == nil ||
		strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.PrincipalID) == "" {
		return GroupSnapshot{}, ErrGroupUnavailable
	}
	if !identity.HasPermission(actor, identity.PermissionOversightRead) {
		return GroupSnapshot{}, ErrGroupForbidden
	}
	projection, err := s.repository.LatestGroup(ctx, actor.TenantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return GroupSnapshot{}, ErrGroupUnavailable
		}
		return GroupSnapshot{}, err
	}
	entityIDs := make([]string, 0, len(projection.Children))
	for _, child := range projection.Children {
		if strings.TrimSpace(child.LegalEntityID) != "" {
			entityIDs = append(entityIDs, child.LegalEntityID)
		}
	}
	resolved, err := s.access.ResolveLegalEntityAccess(ctx, actor.TenantID, actor.PrincipalID, entityIDs)
	if err != nil {
		return GroupSnapshot{}, err
	}
	allowed := make(map[string]struct{}, len(resolved))
	for _, value := range resolved {
		if slices.Contains(identity.NormalizePermissionCodes(value.PermissionCodes), identity.PermissionOversightRead) {
			allowed[value.LegalEntityID] = struct{}{}
		}
	}

	value := GroupSnapshot{
		RevisionID: projection.ID, GeneratedAt: projection.GeneratedAt, ProjectionVersion: projection.ProjectionVersion,
		Freshness: FreshnessCurrent, Children: make([]GroupChildSummary, 0, len(allowed)),
	}
	excludedKnown, unknownKnown := true, true
	excludedTotal, unknownTotal := 0, 0
	for _, child := range projection.Children {
		if _, ok := allowed[child.LegalEntityID]; !ok {
			continue
		}
		value.Coverage.AuthorizedChildren++
		summary := GroupChildSummary{
			LegalEntityID: child.LegalEntityID, LegalEntityCode: child.LegalEntityCode, LegalEntityName: child.LegalEntityName,
			Jurisdiction: child.Jurisdiction, State: child.State, ChildSnapshotID: child.ChildSnapshotID,
			ChildGeneratedAt: child.ChildGeneratedAt, ChildProjectionVersion: child.ChildProjectionVersion,
			Coverage: child.Coverage, Counts: child.Counts, SourceHighWater: child.SourceHighWater,
		}
		value.Children = append(value.Children, summary)
		switch child.State {
		case GroupChildMissing:
			value.Coverage.MissingChildren++
			excludedKnown, unknownKnown = false, false
			continue
		case GroupChildStale:
			value.Coverage.StaleChildren++
		case GroupChildAvailable:
		default:
			return GroupSnapshot{}, ErrGroupUnavailable
		}
		value.Coverage.IncludedChildren++
		value.RecordCoverage.Population += child.Coverage.Population
		if child.Coverage.Excluded == nil {
			excludedKnown = false
		} else {
			excludedTotal += *child.Coverage.Excluded
		}
		if child.Coverage.Unknown == nil {
			unknownKnown = false
		} else {
			unknownTotal += *child.Coverage.Unknown
		}
		addCounts(&value.Counts, child.Counts)
	}
	if value.Coverage.AuthorizedChildren < 2 {
		return GroupSnapshot{}, ErrGroupForbidden
	}
	value.Coverage.Complete = value.Coverage.MissingChildren == 0 && value.Coverage.StaleChildren == 0
	if excludedKnown {
		value.RecordCoverage.Excluded = intPtr(excludedTotal)
	}
	if unknownKnown {
		value.RecordCoverage.Unknown = intPtr(unknownTotal)
	}
	now := s.Now().UTC()
	if !value.Coverage.Complete || projection.ProjectionVersion != GroupProjectionVersion ||
		projection.GeneratedAt.IsZero() || now.Sub(projection.GeneratedAt) > s.StaleAfter {
		value.Freshness = FreshnessStale
	}
	return value, nil
}

func addCounts(target *Counts, value Counts) {
	target.CriticalHigh += value.CriticalHigh
	target.Overdue += value.Overdue
	target.DueSoon += value.DueSoon
	target.RoutingFailures += value.RoutingFailures
	target.Unassigned += value.Unassigned
	target.OutcomeFailures += value.OutcomeFailures
}

func intPtr(value int) *int { return &value }
