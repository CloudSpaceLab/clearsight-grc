package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
)

type groupOversightRepoStub struct {
	value oversight.GroupProjection
}

func (s groupOversightRepoStub) LatestGroup(context.Context, string) (oversight.GroupProjection, error) {
	return s.value, nil
}

type groupOversightAccessStub struct {
	values []access.LegalEntityAccess
}

func (s groupOversightAccessStub) ResolveLegalEntityAccess(context.Context, string, string, []string) ([]access.LegalEntityAccess, error) {
	return s.values, nil
}

func TestGroupOversightHandlerReturnsOnlyAuthorizedAggregateFacts(t *testing.T) {
	now := time.Now().UTC()
	repository := groupOversightRepoStub{value: oversight.GroupProjection{
		ID: "group-run", TenantID: "bank", GeneratedAt: now, ProjectionVersion: oversight.GroupProjectionVersion,
		Children: []oversight.GroupChildFact{
			{LegalEntityID: "entity-a", LegalEntityName: "A", State: oversight.GroupChildAvailable, ChildSnapshotID: "snapshot-a", ChildGeneratedAt: &now, ChildProjectionVersion: oversight.ProjectionVersion, Counts: oversight.Counts{CriticalHigh: 2}},
			{LegalEntityID: "entity-b", LegalEntityName: "B", State: oversight.GroupChildAvailable, ChildSnapshotID: "snapshot-b", ChildGeneratedAt: &now, ChildProjectionVersion: oversight.ProjectionVersion, Counts: oversight.Counts{CriticalHigh: 3}},
			{LegalEntityID: "entity-c", LegalEntityName: "Restricted", State: oversight.GroupChildAvailable, ChildSnapshotID: "snapshot-c", ChildGeneratedAt: &now, ChildProjectionVersion: oversight.ProjectionVersion, Counts: oversight.Counts{CriticalHigh: 999}},
		},
	}}
	resolver := groupOversightAccessStub{values: []access.LegalEntityAccess{
		{LegalEntityID: "entity-a", PermissionCodes: []string{identity.PermissionOversightRead}},
		{LegalEntityID: "entity-b", PermissionCodes: []string{identity.PermissionOversightRead}},
	}}
	service := oversight.NewGroupService(repository, resolver)
	service.Now = func() time.Time { return now }
	api := &API{deps: Dependencies{GroupOversight: service}}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/oversight/group", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-c", PrincipalID: "group-reader",
	}))
	response := httptest.NewRecorder()

	api.groupOversightSnapshot(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var value oversight.GroupSnapshot
	if err := json.NewDecoder(response.Body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value.Counts.CriticalHigh != 5 || len(value.Children) != 2 {
		t.Fatalf("unsafe group response: %#v", value)
	}
	for _, child := range value.Children {
		if child.LegalEntityID == "entity-c" || child.LegalEntityName == "Restricted" || child.Counts.CriticalHigh == 999 {
			t.Fatalf("restricted sibling leaked into group response: %#v", child)
		}
	}
}

func TestGroupOversightHandlerDoesNotRevealSingleAuthorizedSibling(t *testing.T) {
	now := time.Now().UTC()
	service := oversight.NewGroupService(groupOversightRepoStub{value: oversight.GroupProjection{
		ID: "group-run", TenantID: "bank", GeneratedAt: now, ProjectionVersion: oversight.GroupProjectionVersion,
		Children: []oversight.GroupChildFact{
			{LegalEntityID: "entity-a", State: oversight.GroupChildAvailable, ChildSnapshotID: "a", ChildGeneratedAt: &now, ChildProjectionVersion: oversight.ProjectionVersion},
			{LegalEntityID: "entity-b", State: oversight.GroupChildAvailable, ChildSnapshotID: "b", ChildGeneratedAt: &now, ChildProjectionVersion: oversight.ProjectionVersion},
		},
	}}, groupOversightAccessStub{values: []access.LegalEntityAccess{
		{LegalEntityID: "entity-a", PermissionCodes: []string{identity.PermissionOversightRead}},
	}})
	service.Now = func() time.Time { return now }
	api := &API{deps: Dependencies{GroupOversight: service}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/oversight/group", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "reader",
		PermissionCodes: []string{identity.PermissionOversightRead},
	}))
	response := httptest.NewRecorder()

	api.groupOversightSnapshot(response, request)

	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "group_scope_forbidden") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
