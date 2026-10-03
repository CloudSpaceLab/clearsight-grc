package oversight

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type groupRepositoryStub struct {
	value GroupProjection
	err   error
}

func (s groupRepositoryStub) LatestGroup(context.Context, string) (GroupProjection, error) {
	return s.value, s.err
}

type groupAccessStub struct {
	values []access.LegalEntityAccess
	err    error
}

func (s groupAccessStub) ResolveLegalEntityAccess(context.Context, string, string, []string) ([]access.LegalEntityAccess, error) {
	return s.values, s.err
}

func TestGroupServiceAggregatesOnlyAuthorizedChildrenAndKeepsIncompleteCoverage(t *testing.T) {
	now := time.Date(2026, 10, 3, 20, 0, 0, 0, time.UTC)
	excluded, unknown := 1, 2
	repo := groupRepositoryStub{value: GroupProjection{
		ID: "group-run-1", TenantID: "bank", GeneratedAt: now, ProjectionVersion: GroupProjectionVersion,
		ActiveChildCount: 4, CapturedChildCount: 3, MissingChildCount: 1, StaleChildCount: 1,
		Children: []GroupChildFact{
			{LegalEntityID: "entity-a", LegalEntityCode: "A", LegalEntityName: "A", State: GroupChildAvailable, ChildSnapshotID: "snapshot-a", ChildGeneratedAt: timePtr(now.Add(-2 * time.Minute)), ChildProjectionVersion: ProjectionVersion, Coverage: Coverage{Population: 10, Excluded: &excluded, Unknown: &unknown}, Counts: Counts{CriticalHigh: 2, Overdue: 1}},
			{LegalEntityID: "entity-b", LegalEntityCode: "B", LegalEntityName: "B", State: GroupChildStale, ChildSnapshotID: "snapshot-b", ChildGeneratedAt: timePtr(now.Add(-30 * time.Minute)), ChildProjectionVersion: ProjectionVersion, Coverage: Coverage{Population: 20, Excluded: intPtr(0), Unknown: intPtr(1)}, Counts: Counts{CriticalHigh: 3, DueSoon: 4}},
			{LegalEntityID: "entity-c", LegalEntityCode: "C", LegalEntityName: "Restricted sibling", State: GroupChildAvailable, ChildSnapshotID: "snapshot-c", ChildGeneratedAt: timePtr(now), ChildProjectionVersion: ProjectionVersion, Coverage: Coverage{Population: 999}, Counts: Counts{CriticalHigh: 999}},
			{LegalEntityID: "entity-d", LegalEntityCode: "D", LegalEntityName: "D", State: GroupChildMissing},
		},
	}}
	resolver := groupAccessStub{values: []access.LegalEntityAccess{
		{LegalEntityID: "entity-a", PermissionCodes: []string{identity.PermissionOversightRead}},
		{LegalEntityID: "entity-b", PermissionCodes: []string{identity.PermissionOversightRead}},
		{LegalEntityID: "entity-c", PermissionCodes: []string{identity.PermissionConfigRead}},
		{LegalEntityID: "entity-d", PermissionCodes: []string{identity.PermissionOversightRead}},
	}}
	service := NewGroupService(repo, resolver)
	service.Now = func() time.Time { return now }

	value, err := service.Get(context.Background(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "group-reader",
		PermissionCodes: []string{identity.PermissionOversightRead},
	})
	if err != nil {
		t.Fatal(err)
	}
	if value.RevisionID != "group-run-1" || value.Freshness != FreshnessStale {
		t.Fatalf("group metadata = %#v", value)
	}
	if value.Coverage.AuthorizedChildren != 3 || value.Coverage.IncludedChildren != 2 ||
		value.Coverage.MissingChildren != 1 || value.Coverage.StaleChildren != 1 || value.Coverage.Complete {
		t.Fatalf("group coverage = %#v", value.Coverage)
	}
	if value.Counts.CriticalHigh != 5 || value.Counts.Overdue != 1 || value.Counts.DueSoon != 4 {
		t.Fatalf("group counts = %#v", value.Counts)
	}
	if value.RecordCoverage.Population != 30 || value.RecordCoverage.Excluded == nil || *value.RecordCoverage.Excluded != 1 ||
		value.RecordCoverage.Unknown == nil || *value.RecordCoverage.Unknown != 3 {
		t.Fatalf("record coverage = %#v", value.RecordCoverage)
	}
	if len(value.Children) != 3 {
		t.Fatalf("authorized children = %#v", value.Children)
	}
	for _, child := range value.Children {
		if child.LegalEntityID == "entity-c" || child.Counts.CriticalHigh == 999 {
			t.Fatalf("restricted sibling leaked into group snapshot: %#v", child)
		}
	}
}

func TestGroupServiceRequiresAtLeastTwoAuthorizedOpCos(t *testing.T) {
	now := time.Now().UTC()
	repo := groupRepositoryStub{value: GroupProjection{
		ID: "run", TenantID: "bank", GeneratedAt: now, ProjectionVersion: GroupProjectionVersion,
		Children: []GroupChildFact{
			{LegalEntityID: "entity-a", State: GroupChildAvailable, ChildSnapshotID: "a", ChildGeneratedAt: &now, ChildProjectionVersion: ProjectionVersion},
			{LegalEntityID: "entity-b", State: GroupChildAvailable, ChildSnapshotID: "b", ChildGeneratedAt: &now, ChildProjectionVersion: ProjectionVersion},
		},
	}}
	service := NewGroupService(repo, groupAccessStub{values: []access.LegalEntityAccess{
		{LegalEntityID: "entity-a", PermissionCodes: []string{identity.PermissionOversightRead}},
	}})

	_, err := service.Get(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "reader"})
	if !errors.Is(err, ErrGroupForbidden) {
		t.Fatalf("single OpCo group error = %v", err)
	}
}

type boundedGroupAccessStub struct {
	calls int
}

func (s *boundedGroupAccessStub) ResolveLegalEntityAccess(_ context.Context, _, _ string, entityIDs []string) ([]access.LegalEntityAccess, error) {
	s.calls++
	if len(entityIDs) > access.MaxLegalEntityAccessBatchSize {
		return nil, access.ErrPrincipalBatchTooLarge
	}
	values := make([]access.LegalEntityAccess, 0, len(entityIDs))
	for _, id := range entityIDs {
		values = append(values, access.LegalEntityAccess{LegalEntityID: id, PermissionCodes: []string{identity.PermissionOversightRead}})
	}
	return values, nil
}

func TestGroupServiceBatchesLargeOpCoAuthorization(t *testing.T) {
	now := time.Now().UTC()
	children := make([]GroupChildFact, 0, access.MaxLegalEntityAccessBatchSize+1)
	for index := 0; index < access.MaxLegalEntityAccessBatchSize+1; index++ {
		id := fmt.Sprintf("entity-%03d", index)
		children = append(children, GroupChildFact{
			LegalEntityID: id, LegalEntityName: id, State: GroupChildAvailable,
			ChildSnapshotID: "snapshot-" + id, ChildGeneratedAt: &now, ChildProjectionVersion: ProjectionVersion,
		})
	}
	resolver := &boundedGroupAccessStub{}
	service := NewGroupService(groupRepositoryStub{value: GroupProjection{
		ID: "large-run", TenantID: "bank", GeneratedAt: now, ProjectionVersion: GroupProjectionVersion, Children: children,
	}}, resolver)
	service.Now = func() time.Time { return now }

	value, err := service.Get(context.Background(), identity.Actor{TenantID: "bank", PrincipalID: "group-reader"})
	if err != nil {
		t.Fatal(err)
	}
	if resolver.calls != 2 || value.Coverage.AuthorizedChildren != len(children) {
		t.Fatalf("batched authorization calls=%d coverage=%#v", resolver.calls, value.Coverage)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
