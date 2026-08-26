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

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
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
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("non-manager invitation expected 422, got %d: %s", response.Code, response.Body.String())
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
