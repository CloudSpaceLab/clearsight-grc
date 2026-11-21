package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

func TestInternalWorkspaceHandlersDoNotAcceptMissingIdentity(t *testing.T) {
	api := &API{}
	for _, scenario := range []struct {
		method  string
		handler http.HandlerFunc
		body    string
	}{
		{http.MethodGet, api.getInternalFormWorkspace, ""},
		{http.MethodPatch, api.saveInternalFormWorkspace, `{"expected_version":1,"edits":[]}`},
		{http.MethodPost, api.submitInternalFormWorkspace, `{"expected_version":1}`},
		{http.MethodPost, api.uploadInternalFormWorkspaceDocument, ""},
	} {
		req := httptest.NewRequest(scenario.method, "/api/v1/forms/requests/request/workspace", strings.NewReader(scenario.body))
		req.SetPathValue("request_id", "request")
		res := httptest.NewRecorder()
		scenario.handler(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("%s status: %d", scenario.method, res.Code)
		}
		if res.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s missing no-store", scenario.method)
		}
	}
}

func TestInternalWorkspaceDocumentUnknownOutcomeDoesNotClaimFailure(t *testing.T) {
	response := httptest.NewRecorder()
	writeInternalWorkspaceFailure(response, evidence.ErrArtifactOutcomeUnknown)
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "Document status unknown") {
		t.Fatalf("unknown outcome: %d %s", response.Code, response.Body.String())
	}
}

func TestInternalWorkspaceHTTPUsesVerifiedRecipientAndCreatesRevision(t *testing.T) {
	now := time.Now().UTC()
	ctx := context.Background()
	person := evidence.RecipientCandidate{PrincipalID: "person", TenantID: "bank", LegalEntityIDs: []string{"entity"}, Kind: "PERSON", Active: true, ReadableSubjects: map[string]bool{"PROGRAM:program": true}}
	repo := evidence.NewMemoryRepositoryWithRecipientCandidates(nil, nil, []evidence.RecipientCandidate{person})
	forms := httpPolicyFormReader{form: evidence.DistributionFormRevision{ID: "form", TenantID: "bank", LegalEntityID: "entity", Version: 1, Active: true, Fields: []formcontract.Field{{ID: "q1", Label: "Access reviewed?", Type: formcontract.TypeYesNo, Required: true}, {ID: "file", Label: "Document", Type: formcontract.TypeFile}}}}
	var key [32]byte
	for i := range key {
		key[i] = 42
	}
	keyring, err := evidence.NewRecipientKeyring("key", map[string][32]byte{"key": key})
	if err != nil {
		t.Fatal(err)
	}
	store := evidence.NewMemoryDistributionStore(repo, forms, keyring)
	distributions := evidence.NewDistributionService(store)
	bundle, err := distributions.Prepare(ctx, evidence.CreateDistributionInput{TenantID: "bank", LegalEntityID: "entity", FormTemplateID: "form", FormTemplateVersion: 1, SubjectType: "PROGRAM", SubjectID: "program", Title: "Access review", Purpose: "Review access", AccessPolicy: evidence.AccessDirectEmailOTP, EstimatedMinutes: 5, Deadline: now.Add(time.Hour), RouteExpiresAt: now.Add(time.Hour), CreatedBy: "owner", Recipients: []evidence.DistributionRecipientInput{{Role: evidence.RecipientTo, Type: evidence.RecipientInternalPrincipal, PrincipalID: "person"}}})
	if err != nil {
		t.Fatal(err)
	}
	bundle, err = distributions.Open(ctx, "bank", "entity", bundle.Distribution.ID, bundle.Distribution.Version, "owner")
	if err != nil {
		t.Fatal(err)
	}
	access, err := evidence.NewDistributionAccessService(evidence.NewMemoryDistributionAccessStore(store), keyring, nil, key, 20*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	access.WithInternalWorkspaceAuthorizer(func(_ context.Context, actor identity.Actor, request evidence.Request, command string) (string, error) {
		if actor.PrincipalID != "person" || request.ID != bundle.Recipients[0].RequestID || (command != "forms.response.save" && command != "forms.response.submit" && command != "forms.response.upload") {
			return "", errors.New("authority denied")
		}
		return "v1:performer", nil
	})
	api := &API{deps: Dependencies{FormDistributionAccess: access, Evidence: evidence.NewService(repo, evidence.NewMemoryObjectStore())}}
	requestID := bundle.Recipients[0].RequestID
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
	call := func(handler http.HandlerFunc, method, body string, who identity.Actor) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, "/api/v1/forms/requests/"+requestID+"/workspace", strings.NewReader(body))
		request.SetPathValue("request_id", requestID)
		request = request.WithContext(identity.WithActor(request.Context(), who))
		result := httptest.NewRecorder()
		handler(result, request)
		return result
	}
	other := actor
	other.PrincipalID = "other"
	if result := call(api.getInternalFormWorkspace, http.MethodGet, "", other); result.Code != http.StatusNotFound {
		t.Fatalf("other principal: %d %s", result.Code, result.Body.String())
	}
	other = actor
	other.TenantID = "other-bank"
	if result := call(api.getInternalFormWorkspace, http.MethodGet, "", other); result.Code != http.StatusNotFound {
		t.Fatalf("other tenant: %d %s", result.Code, result.Body.String())
	}
	other = actor
	other.LegalEntityID = "other-entity"
	if result := call(api.getInternalFormWorkspace, http.MethodGet, "", other); result.Code != http.StatusNotFound {
		t.Fatalf("other entity: %d %s", result.Code, result.Body.String())
	}
	other = actor
	other.Kind = "SERVICE"
	if result := call(api.getInternalFormWorkspace, http.MethodGet, "", other); result.Code != http.StatusNotFound {
		t.Fatalf("service principal: %d %s", result.Code, result.Body.String())
	}
	if result := call(api.getInternalFormWorkspace, http.MethodGet, "", actor); result.Code != http.StatusOK {
		t.Fatalf("read: %d %s", result.Code, result.Body.String())
	}
	var document bytes.Buffer
	writer := multipart.NewWriter(&document)
	if err := writer.WriteField("field_id", "file"); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("file", "report.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("sample document")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	upload := httptest.NewRequest(http.MethodPost, "/workspace/documents", &document)
	upload.Header.Set("Content-Type", writer.FormDataContentType())
	upload.SetPathValue("request_id", requestID)
	upload = upload.WithContext(identity.WithActor(upload.Context(), actor))
	uploadResult := httptest.NewRecorder()
	api.uploadInternalFormWorkspaceDocument(uploadResult, upload)
	if uploadResult.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", uploadResult.Code, uploadResult.Body.String())
	}
	if strings.Contains(uploadResult.Body.String(), "storage_key") || strings.Contains(uploadResult.Body.String(), "tenant_id") {
		t.Fatalf("internal storage details leaked: %s", uploadResult.Body.String())
	}
	var forged bytes.Buffer
	forgedWriter := multipart.NewWriter(&forged)
	if err := forgedWriter.WriteField("field_id", "file"); err != nil {
		t.Fatal(err)
	}
	if err := forgedWriter.WriteField("created_by", "other"); err != nil {
		t.Fatal(err)
	}
	forgedFile, err := forgedWriter.CreateFormFile("file", "report.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := forgedFile.Write([]byte("sample document")); err != nil {
		t.Fatal(err)
	}
	if err := forgedWriter.Close(); err != nil {
		t.Fatal(err)
	}
	forgedRequest := httptest.NewRequest(http.MethodPost, "/workspace/documents", &forged)
	forgedRequest.Header.Set("Content-Type", forgedWriter.FormDataContentType())
	forgedRequest.SetPathValue("request_id", requestID)
	forgedRequest = forgedRequest.WithContext(identity.WithActor(forgedRequest.Context(), actor))
	forgedResult := httptest.NewRecorder()
	api.uploadInternalFormWorkspaceDocument(forgedResult, forgedRequest)
	if forgedResult.Code != http.StatusBadRequest {
		t.Fatalf("forged upload identity: %d %s", forgedResult.Code, forgedResult.Body.String())
	}
	if result := call(api.saveInternalFormWorkspace, http.MethodPatch, `{"expected_version":1,"actor_id":"other","edits":[]}`, actor); result.Code != http.StatusBadRequest {
		t.Fatalf("forged actor: %d %s", result.Code, result.Body.String())
	}
	result := call(api.saveInternalFormWorkspace, http.MethodPatch, `{"expected_version":1,"edits":[{"field_id":"q1","value":{"text":"Yes"}}]}`, actor)
	if result.Code != http.StatusOK {
		t.Fatalf("save: %d %s", result.Code, result.Body.String())
	}
	var saved evidence.ResponseWorkspaceView
	if err := json.Unmarshal(result.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	result = call(api.submitInternalFormWorkspace, http.MethodPost, `{"expected_version":1}`, actor)
	if result.Code != http.StatusConflict {
		t.Fatalf("stale submit: %d %s", result.Code, result.Body.String())
	}
	raw, _ := json.Marshal(evidence.SubmitWorkspaceInput{ExpectedVersion: saved.Workspace.Version})
	result = call(api.submitInternalFormWorkspace, http.MethodPost, string(raw), actor)
	if result.Code != http.StatusCreated {
		t.Fatalf("submit: %d %s", result.Code, result.Body.String())
	}
	var receipt struct {
		Submission struct {
			SubmissionID string `json:"submission_id"`
		} `json:"submission"`
		Revision struct {
			SubmissionID      string `json:"submission_id"`
			AchievedAssurance string `json:"achieved_assurance"`
		} `json:"revision"`
	}
	if err := json.Unmarshal(result.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Revision.SubmissionID == "" || receipt.Revision.SubmissionID != receipt.Submission.SubmissionID || receipt.Revision.AchievedAssurance != string(evidence.AssuranceInternalAuthenticated) {
		t.Fatalf("revision provenance: %+v", receipt.Revision)
	}
}
