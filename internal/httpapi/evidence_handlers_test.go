package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

const demoEvidenceRequestID = "019fd333-3333-7333-8333-333333333333"

func TestEvidenceSourcesEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/sources?tenant_id=bank-demo&legal_entity_id=bank-ng", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []evidence.Source `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 {
		t.Fatalf("expected two sources, got %d", len(body.Items))
	}
}

func TestEvidenceSourcesEndpointFiltersExactVerifiedEntityBeforeLimit(t *testing.T) {
	now := time.Now().UTC()
	service := evidence.NewService(evidence.NewMemoryRepository([]evidence.Source{
		{ID: "foreign", TenantID: "bank", LegalEntityID: "entity-b", Name: "A foreign", Status: evidence.SourceActive, CreatedAt: now},
		{ID: "exact", TenantID: "bank", LegalEntityID: "entity-a", Name: "B exact", Status: evidence.SourceActive, CreatedAt: now},
	}, nil), evidence.NewMemoryObjectStore())
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "person-a", "entity-a"), Evidence: service})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/sources?tenant_id=bank&limit=1", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []evidence.Source `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].ID != "exact" {
		t.Fatalf("unexpected exact-entity sources: %#v", body.Items)
	}
}

func TestEvidenceSourcesWildcardIdentityRequiresExplicitExactEntity(t *testing.T) {
	service := evidence.NewService(evidence.NewMemoryRepository([]evidence.Source{{ID: "exact", TenantID: "bank", LegalEntityID: "entity-a", Name: "Exact", Status: evidence.SourceActive}}, nil), evidence.NewMemoryObjectStore())
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "admin", "*"), Evidence: service})

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/evidence/sources?tenant_id=bank", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("wildcard without entity returned %d: %s", missing.Code, missing.Body.String())
	}

	exact := httptest.NewRecorder()
	handler.ServeHTTP(exact, httptest.NewRequest(http.MethodGet, "/api/v1/evidence/sources?tenant_id=bank&legal_entity_id=entity-a", nil))
	if exact.Code != http.StatusOK {
		t.Fatalf("wildcard with exact entity returned %d: %s", exact.Code, exact.Body.String())
	}
}

func TestEvidenceRequestQueueIncludesRequestsCreatedByActor(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests?limit=50", nil)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		Items []evidence.Request `json:"items"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	foundExternalCreatedRequest := false
	for _, item := range body.Items {
		if item.ID == demoExternalEvidenceRequestID && item.CreatedBy == "role-cro" {
			foundExternalCreatedRequest = true
		}
	}
	if !foundExternalCreatedRequest {
		t.Fatalf("requester queue did not include the actor's external request: %#v", body.Items)
	}
}

func TestEvidenceReviewerCanListAndReadSubmittedResponseOnly(t *testing.T) {
	now := time.Now().UTC()
	requestValue := evidence.Request{
		ID: "request-review", TenantID: "bank", LegalEntityID: "entity-a", SubjectType: "PROGRAM", SubjectID: "program-a",
		Title: "Monthly encryption review", Status: evidence.RequestReady, CreatedBy: "owner", Deadline: now.Add(time.Hour), Version: 1,
		KnownFacts: map[string]string{"reviewer": "auditor"}, Fields: []evidence.Field{{ID: "reference", Label: "Evidence reference", Type: "short_text"}},
		AudienceType: "INTERNAL", Recipient: evidence.Recipient{Type: evidence.RecipientInternalPrincipal, PrincipalID: "respondent", State: evidence.RecipientStateAssigned},
	}
	repo := evidence.NewMemoryRepository(nil, []evidence.Request{requestValue})
	if _, err := repo.Submit(t.Context(), evidence.Submission{ID: "submission-a", TenantID: "bank", RequestID: requestValue.ID, SubmittedBy: "respondent", Channel: "INTERNAL", Answers: formcontract.TextAnswers(map[string]string{"reference": "ENC-2026-09"}), ExpectedVersion: 1, SubmittedAt: now}); err != nil {
		t.Fatal(err)
	}
	service := evidence.NewService(repo, evidence.NewMemoryObjectStore())
	handler := New(Dependencies{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Identity: identity.NewDevelopmentAuthenticator("bank", "auditor", "entity-a"), Evidence: service})

	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests?limit=50", nil))
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), requestValue.ID) {
		t.Fatalf("reviewer list = %d %s", listResponse.Code, listResponse.Body.String())
	}
	reviewResponse := httptest.NewRecorder()
	handler.ServeHTTP(reviewResponse, httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+requestValue.ID+"/review-submission", nil))
	if reviewResponse.Code != http.StatusOK || !strings.Contains(reviewResponse.Body.String(), "ENC-2026-09") {
		t.Fatalf("review response = %d %s", reviewResponse.Code, reviewResponse.Body.String())
	}

	deniedAPI := &API{deps: Dependencies{Evidence: service}}
	deniedRequest := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+requestValue.ID+"/review-submission", nil).WithContext(identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "auditor-other"}))
	deniedRequest.SetPathValue("id", requestValue.ID)
	deniedResponse := httptest.NewRecorder()
	deniedAPI.getEvidenceReviewSubmission(deniedResponse, deniedRequest)
	if deniedResponse.Code != http.StatusNotFound {
		t.Fatalf("unassigned reviewer status = %d body=%s", deniedResponse.Code, deniedResponse.Body.String())
	}
}

func TestEvidenceExactReadAndSubmissionDoNotRevealCrossEntityRequest(t *testing.T) {
	crossEntity := evidence.Request{
		ID: "request-other-entity", TenantID: "bank", LegalEntityID: "entity-b", SubjectType: "MATTER", SubjectID: "matter-b",
		Title: "Restricted evidence", Status: evidence.RequestReady, Recipient: evidence.Recipient{Type: evidence.RecipientInternalPrincipal, PrincipalID: "person-a"},
		CreatedBy: "person-a", Deadline: time.Now().UTC().Add(time.Hour), Version: 1,
	}
	service := evidence.NewService(evidence.NewMemoryRepository(nil, []evidence.Request{crossEntity}), evidence.NewMemoryObjectStore())
	api := &API{deps: Dependencies{Evidence: service}}
	actorContext := identity.WithActor(t.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity-a", PrincipalID: "person-a"})

	read := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/request-other-entity", nil).WithContext(actorContext)
	read.SetPathValue("id", crossEntity.ID)
	readResponse := httptest.NewRecorder()
	api.getEvidenceRequest(readResponse, read)

	submit := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/request-other-entity/submissions", strings.NewReader(`{"tenant_id":"bank","legal_entity_id":"entity-a","submitted_by":"person-a","answers":{},"expected_version":1}`)).WithContext(actorContext)
	submit.SetPathValue("id", crossEntity.ID)
	submitResponse := httptest.NewRecorder()
	api.submitEvidenceRequest(submitResponse, submit)

	if readResponse.Code != http.StatusNotFound || submitResponse.Code != http.StatusNotFound {
		t.Fatalf("cross-entity statuses read=%d submit=%d; bodies read=%s submit=%s", readResponse.Code, submitResponse.Code, readResponse.Body.String(), submitResponse.Body.String())
	}
	if readResponse.Body.String() != submitResponse.Body.String() {
		t.Fatalf("cross-entity responses differ: read=%s submit=%s", readResponse.Body.String(), submitResponse.Body.String())
	}
}

func TestEvidenceMagicLinkSessionAndSubmission(t *testing.T) {
	handler := testHandler()
	const audience = "manager@example.com"
	issuePayload := `{"tenant_id":"bank-demo","audience":"` + audience + `","purpose":"Branch resilience response","ttl_minutes":60}`
	issue := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", strings.NewReader(issuePayload))
	issueResponse := httptest.NewRecorder()
	handler.ServeHTTP(issueResponse, issue)
	if issueResponse.Code != http.StatusCreated {
		t.Fatalf("issue expected 201, got %d: %s", issueResponse.Code, issueResponse.Body.String())
	}
	var invitation evidence.IssuedInvitation
	if err := json.NewDecoder(issueResponse.Body).Decode(&invitation); err != nil {
		t.Fatal(err)
	}

	wrongAudience := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/invitations/redeem", strings.NewReader(`{"token":"`+invitation.Token+`","audience":"other@example.com"}`))
	wrongAudienceResponse := httptest.NewRecorder()
	handler.ServeHTTP(wrongAudienceResponse, wrongAudience)
	if wrongAudienceResponse.Code != http.StatusUnauthorized {
		t.Fatalf("audience mismatch expected 401, got %d: %s", wrongAudienceResponse.Code, wrongAudienceResponse.Body.String())
	}

	redeem := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/invitations/redeem", strings.NewReader(`{"token":"`+invitation.Token+`","audience":"`+audience+`"}`))
	redeemResponse := httptest.NewRecorder()
	handler.ServeHTTP(redeemResponse, redeem)
	if redeemResponse.Code != http.StatusOK {
		t.Fatalf("redeem expected 200, got %d: %s", redeemResponse.Code, redeemResponse.Body.String())
	}
	var session evidence.RedeemedSession
	if err := json.NewDecoder(redeemResponse.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	replay := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/invitations/redeem", strings.NewReader(`{"token":"`+invitation.Token+`","audience":"`+audience+`"}`))
	replayResponse := httptest.NewRecorder()
	handler.ServeHTTP(replayResponse, replay)
	if replayResponse.Code != http.StatusUnauthorized {
		t.Fatalf("replay expected 401, got %d", replayResponse.Code)
	}
	get := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/session", nil)
	get.Header.Set("Authorization", "Bearer "+session.SessionToken)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("session expected 200, got %d: %s", getResponse.Code, getResponse.Body.String())
	}

	var upload bytes.Buffer
	uploadWriter := multipart.NewWriter(&upload)
	uploadHeader := make(textproto.MIMEHeader)
	uploadHeader.Set("Content-Disposition", `form-data; name="file"; filename="external-note.txt"`)
	uploadHeader.Set("Content-Type", "text/plain")
	uploadFile, err := uploadWriter.CreatePart(uploadHeader)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = uploadFile.Write([]byte("external supporting note"))
	_ = uploadWriter.Close()
	uploadRequest := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/artifacts", &upload)
	uploadRequest.Header.Set("Content-Type", uploadWriter.FormDataContentType())
	uploadRequest.Header.Set("Authorization", "Bearer "+session.SessionToken)
	uploadResponse := httptest.NewRecorder()
	handler.ServeHTTP(uploadResponse, uploadRequest)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("external session upload expected 201, got %d: %s", uploadResponse.Code, uploadResponse.Body.String())
	}
	var externalArtifact evidence.Artifact
	if err := json.NewDecoder(uploadResponse.Body).Decode(&externalArtifact); err != nil {
		t.Fatal(err)
	}
	if externalArtifact.RequestID != demoExternalEvidenceRequestID || externalArtifact.CreatedBy != "" {
		t.Fatalf("external artifact carried false principal identity or wrong request: %#v", externalArtifact)
	}

	submit := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/session/submissions", strings.NewReader(`{"answers":{"condition":"Operational"},"expected_version":1}`))
	submit.Header.Set("Authorization", "Bearer "+session.SessionToken)
	submitResponse := httptest.NewRecorder()
	handler.ServeHTTP(submitResponse, submit)
	if submitResponse.Code != http.StatusOK {
		t.Fatalf("submit expected 200, got %d: %s", submitResponse.Code, submitResponse.Body.String())
	}
}

func TestEvidenceSessionDraftUsesOnlyTheBearerSessionScope(t *testing.T) {
	handler := testHandler()
	session := openExternalEvidenceSession(t, handler)

	get := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/session/draft?tenant_id=another-bank&request_id=another-request", nil)
	get.Header.Set("Authorization", "Bearer "+session.SessionToken)
	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("empty draft expected 200, got %d: %s", getResponse.Code, getResponse.Body.String())
	}
	var empty map[string]any
	if err := json.NewDecoder(getResponse.Body).Decode(&empty); err != nil {
		t.Fatal(err)
	}
	if empty["version"] != float64(0) || empty["answers"] == nil || empty["presentation_mode"] != "AUTOMATIC" {
		t.Fatalf("unexpected empty draft: %#v", empty)
	}
	for _, protected := range []string{"tenant_id", "request_id", "session_id", "id"} {
		if _, exposed := empty[protected]; exposed {
			t.Fatalf("draft response exposed %s: %#v", protected, empty)
		}
	}

	put := httptest.NewRequest(http.MethodPut, "/api/v1/evidence/session/draft", strings.NewReader(`{"answers":{},"presentation_mode":"WIZARD","expected_version":0}`))
	put.Header.Set("Authorization", "Bearer "+session.SessionToken)
	putResponse := httptest.NewRecorder()
	handler.ServeHTTP(putResponse, put)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("draft save expected 200, got %d: %s", putResponse.Code, putResponse.Body.String())
	}
	var saved map[string]any
	if err := json.NewDecoder(putResponse.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	if saved["version"] != float64(1) || saved["presentation_mode"] != "WIZARD" {
		t.Fatalf("unexpected saved draft: %#v", saved)
	}

	stale := httptest.NewRequest(http.MethodPut, "/api/v1/evidence/session/draft", strings.NewReader(`{"answers":{},"presentation_mode":"CLASSIC","expected_version":0}`))
	stale.Header.Set("Authorization", "Bearer "+session.SessionToken)
	staleResponse := httptest.NewRecorder()
	handler.ServeHTTP(staleResponse, stale)
	if staleResponse.Code != http.StatusConflict || !strings.Contains(staleResponse.Body.String(), "saved response changed") {
		t.Fatalf("stale save expected recovery conflict, got %d: %s", staleResponse.Code, staleResponse.Body.String())
	}

	invalid := httptest.NewRequest(http.MethodPut, "/api/v1/evidence/session/draft", strings.NewReader(`{"answers":{"not_requested":{"text":"value"}},"presentation_mode":"CLASSIC","expected_version":1}`))
	invalid.Header.Set("Authorization", "Bearer "+session.SessionToken)
	invalidResponse := httptest.NewRecorder()
	handler.ServeHTTP(invalidResponse, invalid)
	if invalidResponse.Code != http.StatusUnprocessableEntity || !strings.Contains(invalidResponse.Body.String(), "saved response contains an invalid answer") {
		t.Fatalf("invalid draft expected 422, got %d: %s", invalidResponse.Code, invalidResponse.Body.String())
	}

	bodyScoped := httptest.NewRequest(http.MethodPut, "/api/v1/evidence/session/draft", strings.NewReader(`{"tenant_id":"bank-demo","answers":{},"presentation_mode":"CLASSIC","expected_version":1}`))
	bodyScoped.Header.Set("Authorization", "Bearer "+session.SessionToken)
	bodyScopedResponse := httptest.NewRecorder()
	handler.ServeHTTP(bodyScopedResponse, bodyScoped)
	if bodyScopedResponse.Code != http.StatusBadRequest {
		t.Fatalf("client scope should be rejected, got %d: %s", bodyScopedResponse.Code, bodyScopedResponse.Body.String())
	}
}

func TestEvidenceSessionDraftDoesNotEnumerateEndedAccess(t *testing.T) {
	handler := testHandler()
	missing := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/session/draft", nil)
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missing)
	if missingResponse.Code != http.StatusUnauthorized {
		t.Fatalf("missing session expected 401, got %d", missingResponse.Code)
	}

	session := openExternalEvidenceSession(t, handler)
	revoke := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/sessions/"+session.SessionID+"/revoke", strings.NewReader(`{"tenant_id":"outside-scope"}`))
	revokeResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeResponse, revoke)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("session revoke expected 204, got %d: %s", revokeResponse.Code, revokeResponse.Body.String())
	}

	ended := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/session/draft", nil)
	ended.Header.Set("Authorization", "Bearer "+session.SessionToken)
	endedResponse := httptest.NewRecorder()
	handler.ServeHTTP(endedResponse, ended)
	if endedResponse.Code != http.StatusUnauthorized || !strings.Contains(endedResponse.Body.String(), "access has ended") {
		t.Fatalf("revoked session expected non-enumerating 401, got %d: %s", endedResponse.Code, endedResponse.Body.String())
	}
}

func openExternalEvidenceSession(t *testing.T, handler http.Handler) evidence.RedeemedSession {
	t.Helper()
	const audience = "manager@example.com"
	issue := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", strings.NewReader(`{"tenant_id":"bank-demo","audience":"`+audience+`","purpose":"Complete the response","ttl_minutes":60}`))
	issueResponse := httptest.NewRecorder()
	handler.ServeHTTP(issueResponse, issue)
	if issueResponse.Code != http.StatusCreated {
		t.Fatalf("issue expected 201, got %d: %s", issueResponse.Code, issueResponse.Body.String())
	}
	var invitation evidence.IssuedInvitation
	if err := json.NewDecoder(issueResponse.Body).Decode(&invitation); err != nil {
		t.Fatal(err)
	}
	redeem := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/invitations/redeem", strings.NewReader(`{"token":"`+invitation.Token+`","audience":"`+audience+`"}`))
	redeemResponse := httptest.NewRecorder()
	handler.ServeHTTP(redeemResponse, redeem)
	if redeemResponse.Code != http.StatusOK {
		t.Fatalf("redeem expected 200, got %d: %s", redeemResponse.Code, redeemResponse.Body.String())
	}
	var session evidence.RedeemedSession
	if err := json.NewDecoder(redeemResponse.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	return session
}

func TestEvidenceInvitationRequiresCurrentRequestManager(t *testing.T) {
	const audience = "manager@example.com"
	payload := `{"tenant_id":"bank-demo","audience":"` + audience + `","purpose":"Branch resilience response","ttl_minutes":60}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", strings.NewReader(payload))
	request.Header.Set("X-ClearSight-Demo-Principal", "role-other")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("non-manager invitation expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEvidenceRequesterAdministersInvitationWithoutSecretMetadata(t *testing.T) {
	handler := testHandler()
	const audience = "manager@example.com"
	issue := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", strings.NewReader(`{"audience":"`+audience+`","purpose":"Branch resilience response","ttl_minutes":60}`))
	issuedResponse := httptest.NewRecorder()
	handler.ServeHTTP(issuedResponse, issue)
	if issuedResponse.Code != http.StatusCreated {
		t.Fatalf("issue expected 201, got %d: %s", issuedResponse.Code, issuedResponse.Body.String())
	}
	var issued evidence.IssuedInvitation
	if err := json.NewDecoder(issuedResponse.Body).Decode(&issued); err != nil {
		t.Fatal(err)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", nil)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}
	if strings.Contains(listResponse.Body.String(), issued.Token) || strings.Contains(strings.ToLower(listResponse.Body.String()), "token_hash") {
		t.Fatalf("invitation metadata exposed secret material: %s", listResponse.Body.String())
	}

	denied := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", nil)
	denied.Header.Set("X-ClearSight-Demo-Principal", "role-other")
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != http.StatusNotFound {
		t.Fatalf("non-requester list expected 404, got %d: %s", deniedResponse.Code, deniedResponse.Body.String())
	}

	revoke := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations/"+issued.InvitationID+"/revoke", nil)
	revokeResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeResponse, revoke)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("revoke expected 204, got %d: %s", revokeResponse.Code, revokeResponse.Body.String())
	}
}

func TestEvidenceRequesterReplacesInvitationAndRevokesPriorCapability(t *testing.T) {
	handler := testHandler()
	const audience = "manager@example.com"
	issue := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", strings.NewReader(`{"audience":"`+audience+`","purpose":"Branch resilience response","ttl_minutes":60}`))
	issuedResponse := httptest.NewRecorder()
	handler.ServeHTTP(issuedResponse, issue)
	var prior evidence.IssuedInvitation
	if issuedResponse.Code != http.StatusCreated || json.NewDecoder(issuedResponse.Body).Decode(&prior) != nil {
		t.Fatalf("issue failed: %d %s", issuedResponse.Code, issuedResponse.Body.String())
	}

	replace := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations/"+prior.InvitationID+"/replace", strings.NewReader(`{"audience":"`+audience+`","purpose":"Replace the expired delivery link","ttl_minutes":60}`))
	replacedResponse := httptest.NewRecorder()
	handler.ServeHTTP(replacedResponse, replace)
	if replacedResponse.Code != http.StatusCreated {
		t.Fatalf("replace expected 201, got %d: %s", replacedResponse.Code, replacedResponse.Body.String())
	}

	redeemPrior := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/invitations/redeem", strings.NewReader(`{"token":"`+prior.Token+`","audience":"`+audience+`"}`))
	redeemResponse := httptest.NewRecorder()
	handler.ServeHTTP(redeemResponse, redeemPrior)
	if redeemResponse.Code != http.StatusUnauthorized {
		t.Fatalf("replaced invitation remained usable: %d %s", redeemResponse.Code, redeemResponse.Body.String())
	}
}

func TestEvidenceRequesterRevokesRedeemedSession(t *testing.T) {
	handler := testHandler()
	const audience = "manager@example.com"
	issue := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/invitations", strings.NewReader(`{"audience":"`+audience+`","purpose":"Branch resilience response","ttl_minutes":60}`))
	issuedResponse := httptest.NewRecorder()
	handler.ServeHTTP(issuedResponse, issue)
	var issued evidence.IssuedInvitation
	if issuedResponse.Code != http.StatusCreated || json.NewDecoder(issuedResponse.Body).Decode(&issued) != nil {
		t.Fatalf("issue failed: %d %s", issuedResponse.Code, issuedResponse.Body.String())
	}
	redeem := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/invitations/redeem", strings.NewReader(`{"token":"`+issued.Token+`","audience":"`+audience+`"}`))
	redeemResponse := httptest.NewRecorder()
	handler.ServeHTTP(redeemResponse, redeem)
	var session evidence.RedeemedSession
	if redeemResponse.Code != http.StatusOK || json.NewDecoder(redeemResponse.Body).Decode(&session) != nil {
		t.Fatalf("redeem failed: %d %s", redeemResponse.Code, redeemResponse.Body.String())
	}

	revoke := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/requests/"+demoExternalEvidenceRequestID+"/sessions/"+session.SessionID+"/revoke", nil)
	revokeResponse := httptest.NewRecorder()
	handler.ServeHTTP(revokeResponse, revoke)
	if revokeResponse.Code != http.StatusNoContent {
		t.Fatalf("session revoke expected 204, got %d: %s", revokeResponse.Code, revokeResponse.Body.String())
	}

	load := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/session", nil)
	load.Header.Set("Authorization", "Bearer "+session.SessionToken)
	loadResponse := httptest.NewRecorder()
	handler.ServeHTTP(loadResponse, load)
	if loadResponse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session remained usable: %d %s", loadResponse.Code, loadResponse.Body.String())
	}
}

func TestEvidenceArtifactUpload(t *testing.T) {
	body, contentType := artifactUploadBody(t, "condition.txt", "text/plain", "generator operational")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/artifacts", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	var artifact evidence.Artifact
	if err := json.NewDecoder(response.Body).Decode(&artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Status != evidence.ArtifactStoredUnscanned || artifact.SHA256 == "" || artifact.CreatedBy != "role-cro" {
		t.Fatalf("unexpected artifact: %#v", artifact)
	}
}

func TestEvidenceArtifactUploadRejectsNonRecipientActor(t *testing.T) {
	body, contentType := artifactUploadBody(t, "condition.txt", "text/plain", "not my request")
	request := httptest.NewRequest(http.MethodPost, "/api/v1/evidence/artifacts", body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-ClearSight-Demo-Principal", "role-other")
	response := httptest.NewRecorder()
	testHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("non-recipient upload expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func artifactUploadBody(t *testing.T, fileName, mediaType, contents string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("tenant_id", "bank-demo")
	_ = writer.WriteField("request_id", demoEvidenceRequestID)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+fileName+`"`)
	header.Set("Content-Type", mediaType)
	file, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte(contents))
	_ = writer.Close()
	return &body, writer.FormDataContentType()
}
