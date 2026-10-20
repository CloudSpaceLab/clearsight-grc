package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
)

func TestTodayEndpointEncodesNoAssignedWorkAsEmptyCollection(t *testing.T) {
	version, rules := authority.DemoPolicySet()
	handler := New(Dependencies{
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		AllowedOrigin: "http://localhost:5173",
		Mode:          "test-memory",
		Identity:      identity.NewDevelopmentAuthenticator("bank-demo", "role-cro", "bank-ng"),
		Authority:     authority.NewResolver(version, rules),
		Today: today.NewDynamicService(func(context.Context, identity.Actor) ([]today.AttentionItem, error) {
			return nil, nil
		}),
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/today", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	var body struct {
		Items json.RawMessage `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(body.Items) != "[]" {
		t.Fatalf("expected an empty JSON collection, got %s", body.Items)
	}
	var coverage map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &coverage); err != nil {
		t.Fatal(err)
	}
	if string(coverage["count_basis"]) != `"LOADED"` {
		t.Fatalf("Today counts must be explicitly loaded-only: %s", response.Body.String())
	}
	if _, known := coverage["workflow_has_more"]; known {
		t.Fatal("legacy loaders must not invent workflow completeness")
	}
}

func TestTodayEndpointReturnsExplicitActorWorkflowOverflow(t *testing.T) {
	version, rules := authority.DemoPolicySet()
	more := true
	handler := New(Dependencies{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		AllowedOrigin: "http://localhost:5173",
		Mode: "test-memory",
		Identity: identity.NewDevelopmentAuthenticator("bank-demo", "role-cro", "bank-ng"),
		Authority: authority.NewResolver(version, rules),
		Today: today.NewSnapshotService(func(_ context.Context, actor identity.Actor) (today.ActorWorkSnapshot, error) {
			if actor.PrincipalID != "role-cro" {
				t.Fatalf("unexpected reader: %s", actor.PrincipalID)
			}
			return today.ActorWorkSnapshot{
				Items: []today.AttentionItem{{ID: "work-1", Title: "Review evidence"}},
				WorkflowHasMore: &more,
			}, nil
		}),
	})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/today", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("Today response: %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Items []today.AttentionItem `json:"items"`
		CountBasis string `json:"count_basis"`
		WorkflowHasMore *bool `json:"workflow_has_more"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.CountBasis != "LOADED" || payload.WorkflowHasMore == nil || !*payload.WorkflowHasMore || len(payload.Items) != 1 {
		t.Fatalf("unexpected bounded Today contract: %+v", payload)
	}
}
