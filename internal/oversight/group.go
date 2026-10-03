package oversight

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"
)

var ErrInvalidGroupAggregate = errors.New("group oversight aggregate is invalid")

type GroupEntity struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Jurisdiction string `json:"jurisdiction,omitempty"`
}

type GroupChildState string

const (
	GroupChildCurrent GroupChildState = "CURRENT"
	GroupChildStale   GroupChildState = "STALE"
	GroupChildMissing GroupChildState = "MISSING"
)

type GroupChild struct {
	LegalEntityID string          `json:"legal_entity_id"`
	Code          string          `json:"code"`
	Name          string          `json:"name"`
	Jurisdiction  string          `json:"jurisdiction,omitempty"`
	State         GroupChildState `json:"state"`
	SnapshotID    string          `json:"snapshot_id,omitempty"`
	GeneratedAt   *time.Time      `json:"generated_at,omitempty"`
	PostureAsOf   *time.Time      `json:"posture_as_of,omitempty"`
	Coverage      *Coverage       `json:"coverage,omitempty"`
	Counts        *Counts         `json:"counts,omitempty"`
}

type GroupContributor struct {
	LegalEntityID     string    `json:"legal_entity_id"`
	SnapshotID        string    `json:"snapshot_id"`
	GeneratedAt       time.Time `json:"generated_at"`
	PostureAsOf       time.Time `json:"posture_as_of"`
	ProjectionVersion string    `json:"projection_version"`
}

type GroupCoverage struct {
	AuthorizedChildren  int  `json:"authorized_children"`
	ContributingChildren int  `json:"contributing_children"`
	MissingChildren      int  `json:"missing_children"`
	StaleChildren        int  `json:"stale_children"`
	Population           int  `json:"population"`
	Excluded             *int `json:"excluded,omitempty"`
	Unknown              *int `json:"unknown,omitempty"`
}

type GroupSnapshot struct {
	ScopeID            string             `json:"scope_id"`
	ScopeName          string             `json:"scope_name"`
	ScopeKind          string             `json:"scope_kind"`
	GeneratedAt        time.Time          `json:"generated_at"`
	PostureAsOf        time.Time          `json:"posture_as_of"`
	Freshness          Freshness          `json:"freshness"`
	ProjectionVersion  string             `json:"projection_version"`
	ContributorRevision string            `json:"contributor_revision"`
	Coverage           GroupCoverage      `json:"coverage"`
	Counts             Counts             `json:"counts"`
	Children           []GroupChild       `json:"children"`
	Contributors       []GroupContributor `json:"contributors"`
}

func BuildGroupSnapshot(scopeID, scopeName string, entities []GroupEntity, snapshots []Snapshot) (GroupSnapshot, error) {
	scopeID = strings.TrimSpace(scopeID)
	scopeName = strings.TrimSpace(scopeName)
	if scopeID == "" || scopeName == "" || len(entities) < 2 || len(entities) > 256 {
		return GroupSnapshot{}, ErrInvalidGroupAggregate
	}

	entityByID := make(map[string]GroupEntity, len(entities))
	ordered := make([]GroupEntity, 0, len(entities))
	for _, entity := range entities {
		entity.ID = strings.TrimSpace(entity.ID)
		entity.Code = strings.TrimSpace(entity.Code)
		entity.Name = strings.TrimSpace(entity.Name)
		entity.Jurisdiction = strings.TrimSpace(entity.Jurisdiction)
		if entity.ID == "" || entity.Name == "" {
			return GroupSnapshot{}, ErrInvalidGroupAggregate
		}
		if _, duplicate := entityByID[entity.ID]; duplicate {
			return GroupSnapshot{}, ErrInvalidGroupAggregate
		}
		entityByID[entity.ID] = entity
		ordered = append(ordered, entity)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Name == ordered[j].Name {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Name < ordered[j].Name
	})

	snapshotByEntity := make(map[string]Snapshot, len(snapshots))
	for _, snapshot := range snapshots {
		snapshot.LegalEntityID = strings.TrimSpace(snapshot.LegalEntityID)
		snapshot.SnapshotID = strings.TrimSpace(snapshot.SnapshotID)
		if snapshot.LegalEntityID == "" || snapshot.SnapshotID == "" {
			return GroupSnapshot{}, ErrInvalidGroupAggregate
		}
		if _, allowed := entityByID[snapshot.LegalEntityID]; !allowed {
			return GroupSnapshot{}, ErrInvalidGroupAggregate
		}
		if _, duplicate := snapshotByEntity[snapshot.LegalEntityID]; duplicate {
			return GroupSnapshot{}, ErrInvalidGroupAggregate
		}
		snapshotByEntity[snapshot.LegalEntityID] = snapshot
	}

	excluded, unknown := 0, 0
	excludedKnown, unknownKnown := true, true
	value := GroupSnapshot{
		ScopeID: scopeID, ScopeName: scopeName, ScopeKind: "ORGANIZATION",
		Freshness: FreshnessCurrent, ProjectionVersion: "group-oversight-v1",
		Coverage: GroupCoverage{AuthorizedChildren: len(ordered)},
		Children: make([]GroupChild, 0, len(ordered)),
		Contributors: make([]GroupContributor, 0, len(ordered)),
	}
	revisionParts := make([]string, 0, len(ordered))
	for _, entity := range ordered {
		snapshot, found := snapshotByEntity[entity.ID]
		if !found {
			value.Coverage.MissingChildren++
			value.Freshness = FreshnessStale
			unknownKnown = false
			revisionParts = append(revisionParts, entity.ID+":missing")
			value.Children = append(value.Children, GroupChild{
				LegalEntityID: entity.ID, Code: entity.Code, Name: entity.Name,
				Jurisdiction: entity.Jurisdiction, State: GroupChildMissing,
			})
			continue
		}
		state := GroupChildCurrent
		if snapshot.Freshness != FreshnessCurrent {
			state = GroupChildStale
			value.Coverage.StaleChildren++
			value.Freshness = FreshnessStale
		}
		value.Coverage.ContributingChildren++
		value.Coverage.Population += snapshot.Coverage.Population
		addCounts(&value.Counts, snapshot.Counts)
		if snapshot.Coverage.Excluded == nil {
			excludedKnown = false
		} else {
			excluded += *snapshot.Coverage.Excluded
		}
		if snapshot.Coverage.Unknown == nil {
			unknownKnown = false
		} else {
			unknown += *snapshot.Coverage.Unknown
		}
		generatedAt := snapshot.GeneratedAt.UTC()
		postureAsOf := snapshot.PostureAsOf.UTC()
		if postureAsOf.IsZero() {
			postureAsOf = generatedAt
		}
		if value.GeneratedAt.IsZero() || generatedAt.After(value.GeneratedAt) {
			value.GeneratedAt = generatedAt
		}
		if value.PostureAsOf.IsZero() || postureAsOf.Before(value.PostureAsOf) {
			value.PostureAsOf = postureAsOf
		}
		revisionParts = append(revisionParts, entity.ID+":"+snapshot.SnapshotID)
		coverageCopy := snapshot.Coverage
		countsCopy := snapshot.Counts
		value.Children = append(value.Children, GroupChild{
			LegalEntityID: entity.ID, Code: entity.Code, Name: entity.Name, Jurisdiction: entity.Jurisdiction,
			State: state, SnapshotID: snapshot.SnapshotID, GeneratedAt: &generatedAt, PostureAsOf: &postureAsOf,
			Coverage: &coverageCopy, Counts: &countsCopy,
		})
		value.Contributors = append(value.Contributors, GroupContributor{
			LegalEntityID: entity.ID, SnapshotID: snapshot.SnapshotID, GeneratedAt: generatedAt,
			PostureAsOf: postureAsOf, ProjectionVersion: snapshot.ProjectionVersion,
		})
	}
	if value.Coverage.ContributingChildren == 0 {
		return GroupSnapshot{}, ErrInvalidGroupAggregate
	}
	if excludedKnown {
		value.Coverage.Excluded = &excluded
	}
	if unknownKnown && value.Coverage.MissingChildren == 0 {
		value.Coverage.Unknown = &unknown
	}
	sort.Strings(revisionParts)
	hash := sha256.Sum256([]byte(strings.Join(revisionParts, "\n")))
	value.ContributorRevision = hex.EncodeToString(hash[:])
	return value, nil
}

func addCounts(target *Counts, source Counts) {
	target.CriticalHigh += source.CriticalHigh
	target.Overdue += source.Overdue
	target.DueSoon += source.DueSoon
	target.RoutingFailures += source.RoutingFailures
	target.Unassigned += source.Unassigned
	target.OutcomeFailures += source.OutcomeFailures
}
