package apiquestion

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type catalogProofFixture struct {
	binding    sourceaccess.BindingRevision
	view       sourceaccess.ViewRevision
	connection sourceaccess.ConnectionRevision
}

func (f catalogProofFixture) Binding(context.Context, string, string, int64) (sourceaccess.BindingRevision, error) {
	return f.binding, nil
}
func (f catalogProofFixture) View(context.Context, string, string, int64) (sourceaccess.ViewRevision, error) {
	return f.view, nil
}
func (f catalogProofFixture) Connection(context.Context, string, string, int64) (sourceaccess.ConnectionRevision, error) {
	return f.connection, nil
}

type sourceScopeFixture struct{ err error }

func (s sourceScopeFixture) ValidateActiveSourcesForEntity(context.Context, string, string, []string) error {
	return s.err
}

func TestCatalogQuestionBindingProofChecksScopeLineageAndContract(t *testing.T) {
	now := time.Now().UTC().Add(-time.Minute)
	life := sourceaccess.RevisionLifecycle{Status: sourceaccess.RevisionActive, IsCurrent: true, Version: 1, EffectiveFrom: &now}
	fixture := catalogProofFixture{
		binding:    sourceaccess.BindingRevision{BindingID: "binding", TenantID: "tenant", SourceID: "source", ViewID: "view", ViewVersion: 1, Operations: []sourceaccess.Operation{sourceaccess.OperationPage}, SelectedFields: []string{"result"}, RequiredFreshnessMinutes: 60, RevisionLifecycle: life},
		view:       sourceaccess.ViewRevision{ViewID: "view", TenantID: "tenant", SourceID: "source", ConnectionID: "connection", ConnectionVersion: 1, OutputKind: sourceaccess.OutputRecords, Definition: json.RawMessage(`{"path":"/check","singleton":true,"result_type":"BOOLEAN","pagination":{"mode":"NONE"}}`), NativeSchema: []sourceaccess.NativeField{{Name: "result", NativeType: "json:boolean"}}, RevisionLifecycle: life},
		connection: sourceaccess.ConnectionRevision{ConnectionID: "connection", TenantID: "tenant", SourceID: "source", AdapterKind: sourceaccess.AdapterRESTJSON, Definition: json.RawMessage(`{"base_url":"https://bank.example","authentication":{"kind":"NONE"}}`), VerifiedCapabilities: []sourceaccess.Capability{sourceaccess.CapabilityPage}, RevisionLifecycle: life},
	}
	actor := identity.Actor{TenantID: "tenant", LegalEntityID: "entity", PrincipalID: "maker", Kind: "PERSON", ExpiresAt: time.Now().Add(time.Hour)}
	ctx := identity.WithActor(context.Background(), actor)
	check := func(f catalogProofFixture, scope sourceScopeFixture, endpoint string) error {
		return NewCatalogBindingVerifier(f, scope).AuthorizeQuestionBinding(ctx, "tenant", "entity", "form", 1, "question", "binding", 1, "BOOLEAN", endpoint)
	}
	if err := check(fixture, sourceScopeFixture{}, "https://bank.example/check"); err != nil {
		t.Fatalf("valid contract: %v", err)
	}
	if err := check(fixture, sourceScopeFixture{err: errors.New("denied")}, "https://bank.example/check"); err == nil {
		t.Fatal("cross-entity source accepted")
	}
	if err := check(fixture, sourceScopeFixture{}, "https://bank.example/other"); err == nil {
		t.Fatal("different endpoint accepted")
	}
	bad := fixture
	bad.view.Definition = json.RawMessage(`{"path":"/check","singleton":true,"result_type":"NUMBER","pagination":{"mode":"NONE"}}`)
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("wrong result type accepted")
	}
	bad = fixture
	bad.binding.Status = sourceaccess.RevisionPaused
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("paused binding accepted")
	}
	bad = fixture
	bad.connection.SourceID = "other"
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("different source accepted")
	}
	bad = fixture
	bad.view.Definition = json.RawMessage(`{"path":"/check","singleton":true,"result_type":"BOOLEAN","fixed_query":{"token":"x"},"pagination":{"mode":"NONE"}}`)
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("fixed query accepted")
	}
	bad = fixture
	bad.binding.Mapping = json.RawMessage(`{"result":"other"}`)
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("unreviewed mapping accepted")
	}
	bad = fixture
	bad.view.NativeSchema = []sourceaccess.NativeField{{Name: "result", NativeType: "json:boolean", Nullable: true}}
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("nullable result accepted")
	}
	bad = fixture
	bad.connection.VerifiedCapabilities = nil
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("unverified page capability accepted")
	}
	bad = fixture
	bad.connection.Definition = json.RawMessage(`{"base_url":"https://u:secret@bank.example","authentication":{"kind":"NONE"}}`)
	if err := check(bad, sourceScopeFixture{}, "https://bank.example/check"); err == nil {
		t.Fatal("URL credentials accepted")
	}
}
