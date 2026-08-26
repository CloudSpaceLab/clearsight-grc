package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

func TestRequesterListsSanitizedActiveExternalSessions(t *testing.T) {
	handler := testHandler()
	const audience = "manager@example.com"
	issue := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", strings.NewReader(`{"audience":"`+audience+`","purpose":"Branch resilience response","ttl_minutes":60}`))
	issueResponse := httptest.NewRecorder()
	handler.ServeHTTP(issueResponse, issue)
	var invitation evidence.IssuedInvitation
	if issueResponse.Code != http.StatusCreated || json.NewDecoder(issueResponse.Body).Decode(&invitation) != nil {
		t.Fatalf("issue failed: %d %s", issueResponse.Code, issueResponse.Body.String())
	}
	redeem := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/invitations/redeem", strings.NewReader(`{"token":"`+invitation.Token+`","audience":"`+audience+`"}`))
	redeemResponse := httptest.NewRecorder()
	handler.ServeHTTP(redeemResponse, redeem)
	var redeemed evidence.RedeemedSession
	if redeemResponse.Code != http.StatusOK || json.NewDecoder(redeemResponse.Body).Decode(&redeemed) != nil {
		t.Fatalf("redeem failed: %d %s", redeemResponse.Code, redeemResponse.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/sessions?limit=50", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, list)
	if response.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body evidence.ActiveSessionMetadataPage
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != redeemed.SessionID || body.Items[0].AudienceHint != "m***@example.com" || body.HasMore {
		t.Fatalf("unexpected active sessions: %#v", body)
	}
	serialized := strings.ToLower(response.Body.String())
	for _, protected := range []string{strings.ToLower(redeemed.SessionToken), "token_hash", "invitation_id", "request_id", "tenant_id"} {
		if strings.Contains(serialized, protected) {
			t.Fatalf("session inventory exposed %q: %s", protected, response.Body.String())
		}
	}
}

func TestActiveExternalSessionListValidatesLimitAndHidesScopeFailures(t *testing.T) {
	handler := testHandler()

	invalid := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/sessions?limit=51", nil)
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid limit expected 400, got %d: %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	denied := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/sessions?limit=50", nil)
	denied.Header.Set("X-ClearSight-Demo-Principal", "role-other")
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusNotFound || !strings.Contains(deniedResponse.Body.String(), `"error":"not_found"`) {
		t.Fatalf("scope failure expected generic 404, got %d: %s", deniedResponse.Code, deniedResponse.Body.String())
	}
}
