package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
)

type groupPostureReaderStub struct {
	entities []metricview.GroupPostureEntity
	bundle   metricview.GroupPostureBundle
	err      error
	gotIDs   []string
}

func (s *groupPostureReaderStub) ActiveGroupEntities(
	context.Context,
	string,
	time.Time,
) ([]metricview.GroupPostureEntity, error) {
	return append([]metricview.GroupPostureEntity(nil), s.entities...), s.err
}

func (s *groupPostureReaderStub) GroupPosture(
	_ context.Context,
	_ string,
	legalEntityIDs []string,
	_ time.Time,
	_ time.Time,
	_ time.Time,
) (metricview.GroupPostureBundle, error) {
	s.gotIDs = append([]string(nil), legalEntityIDs...)
	return s.bundle, s.err
}

type groupPostureAccessStub struct {
	permissions map[string][]string
	batches     []int
}

func (s *groupPostureAccessStub) ResolveOIDC(
	context.Context, string, string, string, string,
) (access.Resolution, error) {
	return access.Resolution{}, nil
}

func (s *groupPostureAccessStub) ResolvePrincipal(
	context.Context, string, string, string,
) (access.Resolution, error) {
	return access.Resolution{}, nil
}

func (s *groupPostureAccessStub) ResolveLegalEntityAccess(
	_ context.Context,
	_, _ string,
	legalEntityIDs []string,
) ([]access.LegalEntityAccess, error) {
	s.batches = append(s.batches, len(legalEntityIDs))
	values := make([]access.LegalEntityAccess, 0, len(legalEntityIDs))
	for _, id := range legalEntityIDs {
		values = append(values, access.LegalEntityAccess{
			LegalEntityID: id,
			PermissionCodes: append([]string(nil), s.permissions[id]...),
		})
	}
	return values, nil
}

func TestGroupPostureMetricsBatchesAuthorizationAndExcludesRestrictedOpCos(t *testing.T) {
	const childCount = 1001
	entities := make([]metricview.GroupPostureEntity, 0, childCount)
	permissions := make(map[string][]string, childCount)
	for index := 0; index < childCount; index++ {
		id := "00000000-0000-4000-8001-" + fmtGroupEntitySuffix(index)
		entities = append(entities, metricview.GroupPostureEntity{
			LegalEntityID: id,
			LegalEntityCode: "OPCO-" + strconv.Itoa(index),
			LegalEntityName: "OpCo " + strconv.Itoa(index),
		})
		if index != childCount-1 {
			permissions[id] = []string{identity.PermissionOversightRead}
		}
	}
	now := time.Now().UTC()
	reader := &groupPostureReaderStub{
		entities: entities,
		bundle: metricview.GroupPostureBundle{
			GeneratedAt: now,
			PeriodStart: now.Add(-30 * 24 * time.Hour),
			PeriodEnd: now,
			DefinitionRevision: metricview.DomainDefinitionRevision,
			RiskCoverage: metricview.GroupPostureCoverage{AuthorizedChildren: childCount - 1, IncludedChildren: childCount - 1, Complete: true},
			LossCoverage: metricview.GroupPostureCoverage{AuthorizedChildren: childCount - 1, IncludedChildren: childCount - 1, Complete: true},
		},
	}
	resolver := &groupPostureAccessStub{permissions: permissions}
	api := &API{deps: Dependencies{
		Access: resolver,
		GroupPostureMetrics: reader,
	}}
	start := now.Add(-30 * 24 * time.Hour).Format("2006-01-02")
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/group/posture?start_date="+start, nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: "group-reader",
	}))
	response := httptest.NewRecorder()

	api.groupPostureMetrics(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if len(resolver.batches) != 3 || resolver.batches[0] != 500 || resolver.batches[1] != 500 || resolver.batches[2] != 1 {
		t.Fatalf("authorization batches=%v", resolver.batches)
	}
	if len(reader.gotIDs) != childCount-1 {
		t.Fatalf("authorized ids=%d", len(reader.gotIDs))
	}
	restricted := entities[childCount-1].LegalEntityID
	for _, id := range reader.gotIDs {
		if id == restricted {
			t.Fatalf("restricted OpCo reached Group posture reader: %s", id)
		}
	}
	var bundle metricview.GroupPostureBundle
	if err := json.NewDecoder(response.Body).Decode(&bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.RiskCoverage.AuthorizedChildren != childCount-1 {
		t.Fatalf("bundle=%#v", bundle)
	}
}

func TestGroupPostureMetricsDoesNotRevealSingleAuthorizedOpCo(t *testing.T) {
	entityA := "00000000-0000-4000-8001-000000000001"
	entityB := "00000000-0000-4000-8001-000000000002"
	reader := &groupPostureReaderStub{entities: []metricview.GroupPostureEntity{
		{LegalEntityID: entityA, LegalEntityName: "A"},
		{LegalEntityID: entityB, LegalEntityName: "B"},
	}}
	resolver := &groupPostureAccessStub{permissions: map[string][]string{
		entityA: {identity.PermissionOversightRead},
	}}
	api := &API{deps: Dependencies{Access: resolver, GroupPostureMetrics: reader}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/group/posture?start_date=2026-09-01", nil)
	request = request.WithContext(identity.WithActor(request.Context(), identity.Actor{
		TenantID: "bank", PrincipalID: "reader",
	}))
	response := httptest.NewRecorder()

	api.groupPostureMetrics(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if len(reader.gotIDs) != 0 {
		t.Fatalf("single authorized OpCo reached Group posture reader: %v", reader.gotIDs)
	}
}

func fmtGroupEntitySuffix(index int) string {
	return fmt.Sprintf("%012d", index+1)
}
