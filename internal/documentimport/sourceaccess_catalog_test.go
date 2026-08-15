package documentimport

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func TestGovernedTabularImportIsReusableThroughSourceCatalog(t *testing.T) {
	ctx := context.Background()
	imports := NewService(NewMemoryRepository(), evidence.NewMemoryObjectStore())
	document, err := imports.Import(ctx, ImportInput{
		TenantID: "tenant-a", FileName: "accounts.csv", MediaType: "text/csv", CreatedBy: "actor-a",
	}, strings.NewReader("id,name,status\n1,Ada,ACTIVE\n2,Grace,REVIEW\n"))
	if err != nil {
		t.Fatal(err)
	}

	adapters := sourceaccess.DefaultCatalogAdapters()
	adapters[sourceaccess.AdapterTabularArtifact] = imports.SourceAccessAdapter()
	catalog := sourceaccess.NewCatalogService(
		sourceaccess.NewMemoryCatalogRepository([]sourceaccess.SourceScope{{TenantID: "tenant-a", SourceID: "source-a"}}),
		sourceaccess.EnvironmentSecretResolver{},
		adapters,
	)
	actor := sourceaccess.CatalogActor{TenantID: "tenant-a", PrincipalID: "actor-a"}
	connection, err := catalog.CreateConnectionDraft(ctx, actor, "source-a", sourceaccess.CreateConnectionDraftInput{
		Code: "ACCOUNT_FILE", Name: "Account file", AdapterKind: sourceaccess.AdapterTabularArtifact,
		AdapterVersion: sourceaccess.TabularArtifactAdapterVersion,
		DeclaredCapabilities: []sourceaccess.Capability{sourceaccess.CapabilityInspect, sourceaccess.CapabilityPage, sourceaccess.CapabilityLookup},
	})
	if err != nil {
		t.Fatal(err)
	}
	view, err := catalog.CreateViewDraft(ctx, actor, connection.ConnectionID, sourceaccess.CreateViewDraftInput{
		ConnectionVersion: connection.Version, Code: "ACCOUNTS", Name: "Accounts",
		Definition: []byte(`{"document_id":"` + document.ID + `","resource":"records"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	inspected, err := catalog.InspectViewDraft(ctx, actor, view.ViewID, view.Version, sourceaccess.InspectViewDraftInput{StableKeys: []string{"id"}})
	if err != nil {
		t.Fatal(err)
	}
	if inspected.View.SchemaFingerprint == "" || len(inspected.View.NativeSchema) != 3 || inspected.Schema.Receipt.ArtifactID != document.ID {
		t.Fatalf("inspection did not retain artifact-backed schema truth: %#v", inspected)
	}
	binding, err := catalog.CreateBindingDraft(ctx, actor, inspected.View.ViewID, sourceaccess.CreateBindingDraftInput{
		ViewVersion: inspected.View.Version, Code: "ACCOUNT_ACCESS", Name: "Account access", Purpose: "account-review",
		Operations: []sourceaccess.Operation{sourceaccess.OperationPage, sourceaccess.OperationLookup},
		SelectedFields: []string{"id", "name", "status"}, KeyFields: []string{"id"},
		Limits: sourceaccess.ResourceLimits{PageRows: 10, ResponseBytes: 64 << 10, LookupValues: 10, Timeout: 2 * time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := catalog.PreviewBinding(ctx, "tenant-a", binding.BindingID, binding.Version, sourceaccess.PageRequest{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Records) != 1 || page.Records[0]["name"].Text != "Ada" || page.NextCursor == nil || page.Receipt.ArtifactID != document.ID {
		t.Fatalf("catalog preview did not reuse the imported artifact: %#v", page)
	}
	lookup, err := catalog.LookupBinding(ctx, "tenant-a", binding.BindingID, binding.Version, sourceaccess.LookupRequest{Values: []sourceaccess.Scalar{sourceaccess.StringValue("2")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Records) != 1 || lookup.Records[0]["name"].Text != "Grace" || lookup.Receipt.ArtifactSHA256 != document.SHA256 {
		t.Fatalf("catalog lookup did not reuse the imported artifact: %#v", lookup)
	}
}
