package monitoring

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeFormAIGatewayToolResponse(t, w, "resp-1", `{"sections":[]}`)
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
		t.Fatalf("malformed governed response was accepted: %v", err)
	}
}

func TestFormAIClientCapturesGatewayAndValidationProvenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer credential-123" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("X-Request-ID", "req_0011223344556677")
		w.Header().Set("X-ClearSight-Policy", "form-authoring-policy@7")
		w.Header().Set("X-ClearSight-Route", "openai-primary")
		writeFormAIGatewayToolResponse(t, w, "resp-2", `{"sections":[]}`)
	}))
	defer server.Close()
	client, err := NewHTTPFormAIClient(FormAIGatewayConfig{
		GatewayURL: server.URL, TenantID: "bank-a", WorkloadID: "forms-authoring", Credential: "credential-123", ModelAlias: "authoring",
		PromptVersion: "FORM_AUTHORING_V7", Timeout: 5 * time.Second,
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, _ = client, documentimport.ElementParagraph
}

func writeFormAIGatewayToolResponse(t *testing.T, w http.ResponseWriter, id, arguments string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"id": id, "status": "completed", "model": "authoring",
		"output": []map[string]string{{"type": "function_call", "name": "submit_form_proposal", "arguments": arguments}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}
