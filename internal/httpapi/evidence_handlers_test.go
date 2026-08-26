package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

const demoEvidenceRequestID = "019fd333-3333-7333-8333-333333333333"

func TestEvidenceSourcesEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evidence/sources?tenant_id=bank-demo", nil)
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
