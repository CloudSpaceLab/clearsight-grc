//go:build postgres && postgresintegration

package sourceaccess

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresSingletonPageBindingAndOrdinaryKeyRequirement(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ids := make([]string, 9)
	for i := range ids {
		ids[i], err = id.NewUUIDv7()
		if err != nil {
			t.Fatal(err)
		}
	}
	tenant, actor, source, connectionID, viewID, bindingID := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
	cleanup := func() {
		for _, query := range []string{`DELETE FROM source_bindings WHERE tenant_id=$1::uuid`, `DELETE FROM source_views WHERE tenant_id=$1::uuid`, `DELETE FROM source_connections WHERE tenant_id=$1::uuid`, `DELETE FROM evidence_sources WHERE tenant_id=$1::uuid`, `DELETE FROM principals WHERE tenant_id=$1::uuid`, `DELETE FROM tenants WHERE id=$1::uuid`} {
			if _, err := pool.Exec(context.Background(), query, tenant); err != nil {
				t.Errorf("cleanup: %v", err)
			}
		}
	}
	defer cleanup()
	if _, err = pool.Exec(ctx, `INSERT INTO tenants(id,slug,name) VALUES($1::uuid,$2,'Singleton test')`, tenant, "singleton-"+tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO principals(id,tenant_id,kind,display_name) VALUES($1::uuid,$2::uuid,'PERSON','Owner')`, actor, tenant); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO evidence_sources(id,tenant_id,code,name,source_type,authority_class,owner_principal_id,expected_freshness_minutes,health,status,version) VALUES($1::uuid,$2::uuid,'SOURCE','Source','SYSTEM','SYSTEM_OF_RECORD',$3::uuid,15,'UNKNOWN','ACTIVE',1)`, source, tenant, actor); err != nil {
		t.Fatal(err)
	}
	repo := NewPostgresCatalogRepository(pool)
	now := time.Now().UTC().Add(-time.Minute)
	conn := catalogConnectionRevision(now)
	conn.RevisionID, conn.ConnectionID, conn.TenantID, conn.SourceID, conn.CreatedBy, conn.OwnerPrincipalID = ids[6], connectionID, tenant, source, actor, actor
	conn.AdapterKind = AdapterRESTJSON
	conn.AdapterVersion = RESTJSONAdapterVersion
	conn.SecretRef = ""
	conn.Definition = json.RawMessage(`{"base_url":"https://bank.example","authentication":{"kind":"NONE"}}`)
	conn.DeclaredCapabilities = []Capability{CapabilityPage}
	conn.VerifiedCapabilities = []Capability{CapabilityPage}
	if _, err = repo.CreateConnectionRevision(ctx, conn); err != nil {
		t.Fatal(err)
	}
	view := catalogViewRevision(now)
	view.RevisionID, view.ViewID, view.TenantID, view.SourceID, view.ConnectionID, view.CreatedBy = ids[7], viewID, tenant, source, connectionID, actor
	view.Definition = json.RawMessage(`{"path":"/check","singleton":true,"result_type":"BOOLEAN","pagination":{"mode":"NONE"}}`)
	view.StableKeys = []string{}
	view.NativeSchema = []NativeField{{Name: "result", NativeType: "json:boolean"}}
	if _, err = repo.CreateViewRevision(ctx, view); err != nil {
		t.Fatal(err)
	}
	binding := catalogBindingRevision(now)
	binding.RevisionID, binding.BindingID, binding.TenantID, binding.SourceID, binding.ViewID, binding.CreatedBy = ids[8], bindingID, tenant, source, viewID, actor
	binding.Operations = []Operation{OperationPage}
	binding.SelectedFields = []string{"result"}
	binding.KeyFields = []string{}
	binding.Mapping = json.RawMessage(`{}`)
	binding.ParameterSchema = json.RawMessage(`{}`)
	if _, err = repo.CreateBindingRevision(ctx, binding); err != nil {
		t.Fatalf("singleton PAGE: %v", err)
	}
	// An ordinary REST view still requires a stable PAGE key at the database boundary.
	ordinary := view
	ordinary.RevisionID = "38888888-8888-7888-8888-888888888888"
	ordinary.ViewID = "38888888-8888-7888-8888-888888888889"
	ordinary.Code = "ORDINARY"
	ordinary.Definition = json.RawMessage(`{"path":"/records","pagination":{"mode":"NONE"}}`)
	if _, err = repo.CreateViewRevision(ctx, ordinary); err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO source_bindings SELECT '37777777-7777-7777-8777-777777777777'::uuid,'37777777-7777-7777-8777-777777777778'::uuid,tenant_id,source_id,$2::uuid,view_version,'INVALID_PAGE',name,purpose,operations,selected_fields,key_fields,page_rows,response_bytes,lookup_values,timeout_ms,mapping,parameter_schema,output_schema,required_freshness_minutes,completeness,sensitivity_handling,status,is_current,effective_from,effective_until,version,created_by,created_at,updated_at FROM source_bindings WHERE binding_id=$1::uuid`, bindingID, ordinary.ViewID)
	if err == nil {
		t.Fatal("ordinary keyless PAGE accepted")
	}
	// Exercise the database guard directly: bypassing service validation must not allow LOOKUP without a key.
	_, err = pool.Exec(ctx, `INSERT INTO source_bindings SELECT '39999999-9999-7999-8999-999999999999'::uuid,'39999999-9999-7999-8999-999999999998'::uuid,tenant_id,source_id,view_id,view_version,'INVALID_LOOKUP',name,purpose,'["LOOKUP"]'::jsonb,selected_fields,key_fields,page_rows,response_bytes,lookup_values,timeout_ms,mapping,parameter_schema,output_schema,required_freshness_minutes,completeness,sensitivity_handling,status,is_current,effective_from,effective_until,version,created_by,created_at,updated_at FROM source_bindings WHERE binding_id=$1::uuid`, bindingID)
	if err == nil {
		t.Fatal("keyless LOOKUP accepted")
	}
}
