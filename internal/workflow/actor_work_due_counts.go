package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrActorWorkCountsUnavailable is returned instead of manufacturing zero
// totals when an authorized source cannot provide complete counts.
var ErrActorWorkCountsUnavailable = errors.New("actor work counts are unavailable")

// ActorWorkDueCounts describes the CURRENT authorized workflow assignment
// population only. It is not a tenant-wide Work or completed-outcome count.
type ActorWorkDueCounts struct {
	Total       int64     `json:"total"`
	Overdue     int64     `json:"overdue"`
	DueSoon     int64     `json:"due_soon"`
	Undated     int64     `json:"undated"`
	AsOf        time.Time `json:"as_of"`
	Coverage    string    `json:"coverage"`
	CountBasis  string    `json:"count_basis"`
}

type actorWorkDueCountsRepository interface {
	CountActorWorkDue(context.Context, ListFilter, time.Time) (ActorWorkDueCounts, error)
}

// ReadActorWorkDueCounts counts the same source-authorized workflow and
// proposal records as ListActorWorkPage, without applying its page LIMIT.
// Non-workflow recovery, identity, background-job, Team and Completed sources
// are excluded and must not be implied by these numbers.
func (s *Service) ReadActorWorkDueCounts(ctx context.Context, tenantID, legalEntityID, principalID string) (ActorWorkDueCounts, error) {
	if s == nil {
		return ActorWorkDueCounts{}, ErrActorWorkCountsUnavailable
	}
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	principalID = strings.TrimSpace(principalID)
	if tenantID == "" || legalEntityID == "" || principalID == "" {
		return ActorWorkDueCounts{}, fmt.Errorf("actor work requires verified tenant, entity and principal")
	}
	repo, ok := s.repo.(actorWorkDueCountsRepository)
	if !ok {
		return ActorWorkDueCounts{}, ErrActorWorkCountsUnavailable
	}
	asOf := time.Now().UTC().Truncate(time.Microsecond)
	counts, err := repo.CountActorWorkDue(ctx, ListFilter{
		TenantID: tenantID, LegalEntityID: legalEntityID, PrincipalID: principalID,
		VisibleActorWorkOnly: true, ActiveOnly: true, UpdatedThrough: &asOf,
	}, asOf)
	if err != nil {
		return ActorWorkDueCounts{}, err
	}
	counts.AsOf = asOf
	counts.Coverage = "WORKFLOW_ASSIGNMENTS_ONLY"
	counts.CountBasis = "SOURCE_SNAPSHOT"
	return counts, nil
}
