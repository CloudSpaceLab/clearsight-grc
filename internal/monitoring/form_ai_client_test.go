package monitoring

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFormAIClientRejectsTenantBindingBeforeGatewayCall(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	client, err := NewHTTPFormAIClient(FormAIGatewayConfig{
		GatewayURL: server.URL, TenantID: "bank-a", WorkloadID: "forms-authoring", Credential: "credential-123", ModelAlias: "authoring",
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Propose(t.Context(), FormAIClientRequest{
		TenantID: "bank-b", LegalEntityID: "entity-a", PrincipalID: "maker-a", Objective: "Draft a form.",
		SnapshotSHA256: strings.Repeat("a", 64),
	})
	if !errors.Is(err, ErrFormAIUnavailable) || called {
		t.Fatalf("tenant mismatch reached gateway: err=%v called=%v", err, called)
	}
}

func TestFormAIClientRejectsUnknownGovernedFieldSemantics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp-1","status":"completed","model":"authoring","output":[{"type":"function_call","name":"submit_form_proposal","arguments":"{\"sections\":[{\"id\":\"general\",\"title\":\"General\"}],\"changes\":[{\"kind\":\"ADD_FIELD\",\"confidence\":0.9,\"field\":{\"key\":\"owner\",\"section_id\":\"general\",\"label\":\"Control owner\",\"type\":\"short_text\",\"scoring\":{\"weight\":100}}}]}"}]}`))
	}))
	defer server.Close()
	client, err := NewHTTPFormAIClient(FormAIGatewayConfig{
		GatewayURL: server.URL, TenantID: "bank-a", WorkloadID: "forms-authoring", Credential: "credential-123", ModelAlias: "authoring",
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Propose(t.Context(), FormAIClientRequest{
		TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker-a", Objective: "Draft a form.",
		SnapshotSHA256: strings.Repeat("a", 64),
	})
	if !errors.Is(err, ErrFormAIUnavailable) {
		t.Fatalf("unknown scoring semantics were accepted: %v", err)
	}
}

func TestFormAIClientCapturesGatewayAndValidationProvenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer credential-123" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", "req_0011223344556677")
		w.Header().Set("X-ClearSight-Policy", "form-authoring-policy@7")
		w.Header().Set("X-ClearSight-Route", "openai-primary")
		_, _ = w.Write([]byte(`{"id":"resp-2","status":"completed","model":"authoring","output":[{"type":"function_call","name":"submit_form_proposal","arguments":"{\"sections\":[{\"id\":\"general\",\"title\":\"General\"}],\"changes\":[{\"kind\":\"ADD_FIELD\",\"source_ref\":\"source-1\",\"confidence\":0.9,\"field\":{\"key\":\"owner\",\"section_id\":\"general\",\"label\":\"Control owner\",\"type\":\"short_text\"}}]}"}]}`))
	}))
	defer server.Close()
	client, err := NewHTTPFormAIClient(FormAIGatewayConfig{
		GatewayURL: server.URL, TenantID: "bank-a", WorkloadID: "forms-authoring", Credential: "credential-123", ModelAlias: "authoring",
		PromptVersion: "FORM_AUTHORING_V7", Timeout: 5 * time.Second,
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Propose(t.Context(), FormAIClientRequest{
		TenantID: "bank-a", LegalEntityID: "entity-a", PrincipalID: "maker-a", Objective: "Draft a control-owner field.",
		SnapshotSHA256: strings.Repeat("b", 64),
		Source: &FormAISourceSnapshot{DocumentID: "doc-1", Version: 3, SHA256: strings.Repeat("c", 64), Elements: []documentimport.ExtractedElement{{Ref: "source-1", Kind: documentimport.ElementParagraph, Text: "Who owns this control?", Anchor: documentimport.SourceAnchor{Page: 2}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FieldChanges) != 1 || result.FieldChanges[0].Anchor.Page != 2 {
		t.Fatalf("source anchored result = %#v", result)
	}
	if result.Provenance.WorkloadID != "forms-authoring" || result.Provenance.PolicyRef != "form-authoring-policy@7" || result.Provenance.GatewayRequestID != "req_0011223344556677" || result.Provenance.GatewayResponseID != "resp-2" || result.Provenance.RouteID != "openai-primary" || result.Provenance.PromptVersion != "FORM_AUTHORING_V7" || len(result.Provenance.ValidationResults) == 0 {
		t.Fatalf("gateway provenance = %#v", result.Provenance)
	}
}
