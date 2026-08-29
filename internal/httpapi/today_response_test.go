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
}
