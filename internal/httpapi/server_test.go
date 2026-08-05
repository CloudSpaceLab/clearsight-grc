package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/today"
)

func testHandler() http.Handler {
	version, rules := authority.DemoPolicySet()
	return New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), AllowedOrigin: "http://localhost:5173", Authority: authority.NewResolver(version, rules), Capture: capture.NewService(capture.DemoRequests()), Invitations: capture.NewInvitationService(time.Now), Today: today.NewService(today.DemoItems())})
}

func TestTodayEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/today", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK { t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String()) }
	var body struct{ Items []today.AttentionItem `json:"items"` }
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil { t.Fatalf("decode: %v", err) }
	if len(body.Items) != 3 { t.Fatalf("expected 3 attention items, got %d", len(body.Items)) }
}

func TestAuthorityResolutionEndpoint(t *testing.T) {
	payload := []byte(`{"tenant_id":"bank-demo","legal_entity_id":"bank-ng","object_type":"MATTER","object_id":"matter-1","responsibility":"AUTHORIZER","materiality":5}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/authority/resolve", bytes.NewReader(payload))
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK { t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String()) }
}
