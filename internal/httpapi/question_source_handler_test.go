package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestFormValidationBindingIsScopedAndDoesNotReturnConnectionSettings(t *testing.T) {
	const tenant = "11111111-1111-7111-8111-111111111111"
	const source = "12222222-2222-7222-8222-222222222222"
	const connection = "13333333-3333-7333-8333-333333333333"
	const view = "14444444-4444-7444-8444-444444444444"
	const binding = "15555555-5555-7555-8555-555555555555"
	const actorID = "16666666-6666-7666-8666-666666666666"
	ctx := context.Background()
	repo := sourceaccess.NewMemoryCatalogRepository([]sourceaccess.SourceScope{{TenantID: tenant, SourceID: source}})
	now := time.Now().UTC()
	lifecycle := sourceaccess.RevisionLifecycle{Status: sourceaccess.RevisionActive, IsCurrent: true, Version: 1, EffectiveFrom: &now, CreatedBy: actorID, CreatedAt: now, UpdatedAt: now}
	_, err := repo.CreateConnectionRevision(ctx, sourceaccess.ConnectionRevision{RevisionID: connection, ConnectionID: connection, TenantID: tenant, SourceID: source, Code: "SAMPLE", Name: "Sample", AdapterKind: sourceaccess.AdapterPostgres, AdapterVersion: sourceaccess.PostgresAdapterVersion, SecretRef: "env://PRIVATE_SOURCE_SECRET", Definition: json.RawMessage("{}"), DeclaredCapabilities: []sourceaccess.Capability{sourceaccess.CapabilityLookup}, VerifiedCapabilities: []sourceaccess.Capability{sourceaccess.CapabilityLookup}, RevisionLifecycle: lifecycle})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateViewRevision(ctx, sourceaccess.ViewRevision{RevisionID: view, ViewID: view, TenantID: tenant, SourceID: source, ConnectionID: connection, ConnectionVersion: 1, Code: "SAMPLE", Name: "Sample", Definition: json.RawMessage(`{"query":"SELECT employee_id FROM source"}`), OutputKind: sourceaccess.OutputRecords, StableKeys: []string{"employee_id"}, NativeSchema: []sourceaccess.NativeField{{Name: "employee_id", NativeType: "text"}}, SchemaFingerprint: strings.Repeat("a", 64), RevisionLifecycle: lifecycle})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateBindingRevision(ctx, sourceaccess.BindingRevision{RevisionID: binding, BindingID: binding, TenantID: tenant, SourceID: source, ViewID: view, ViewVersion: 1, Code: "SAMPLE", Name: "Approved sample lookup", Purpose: "Staff validation", Operations: []sourceaccess.Operation{sourceaccess.OperationLookup}, SelectedFields: []string{"employee_id"}, KeyFields: []string{"employee_id"}, Limits: sourceaccess.ResourceLimits{PageRows: 10, ResponseBytes: 65536, LookupValues: 1, Timeout: time.Second}, RequiredFreshnessMinutes: 60, RevisionLifecycle: lifecycle})
	if err != nil {
		t.Fatal(err)
	}
	capture := evidence.NewService(evidence.NewMemoryRepository([]evidence.Source{{ID: source, TenantID: tenant, LegalEntityID: "entity-a", Status: evidence.SourceActive}}, nil), evidence.NewMemoryObjectStore())
	api := &API{deps: Dependencies{SourceCatalog: sourceaccess.NewCatalogService(repo, nil, nil), Evidence: capture}}
	for _, tc := range []struct {
		entity, tenant string
		status         int
	}{{"entity-a", tenant, 200}, {"entity-b", tenant, 404}, {"entity-a", "other-tenant", 404}, {"* ", tenant, 404}} {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/forms/validation-bindings/"+binding+"?version=1", nil)
		request.SetPathValue("binding_id", binding)
		request = request.WithContext(identity.WithActor(ctx, identity.Actor{TenantID: tc.tenant, LegalEntityID: tc.entity, PrincipalID: actorID, ExpiresAt: now.Add(time.Hour)}))
		result := httptest.NewRecorder()
		api.getFormValidationBinding(result, request)
		if result.Code != tc.status {
			t.Fatalf("scope %+v status %d body %s", tc, result.Code, result.Body.String())
		}
		if result.Code == 200 && (strings.Contains(result.Body.String(), "PRIVATE_SOURCE_SECRET") || strings.Contains(result.Body.String(), "query") || strings.Contains(result.Body.String(), "connection_id")) {
			t.Fatal("source configuration leaked")
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/?version=1", nil)
	result := httptest.NewRecorder()
	api.getFormValidationBinding(result, request)
	if result.Code != 401 {
		t.Fatalf("unsigned request accepted: %d", result.Code)
	}
}
