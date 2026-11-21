package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/apiquestion"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type draftFixture struct {
	writes   int
	expected int64
	input    apiquestion.DraftInput
}

func (d *draftFixture) ReadDraft(context.Context, string, int64, string) (apiquestion.Draft, error) {
	return apiquestion.Draft{Version: 1, DraftInput: apiquestion.DraftInput{Mode: "API", Endpoint: "https://bank.example/status", ResultType: "BOOLEAN"}}, nil
}
func (d *draftFixture) WriteDraft(_ context.Context, _ string, _ int64, _ string, expected int64, input apiquestion.DraftInput) (apiquestion.Receipt, error) {
	d.writes++
	d.expected = expected
	d.input = input
	return apiquestion.Receipt{ID: "receipt", Version: expected + 1}, nil
}
func draftRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Idempotency-Key", "11111111-1111-4111-8111-111111111111")
	r.SetPathValue("id", "form")
	r.SetPathValue("version", "1")
	r.SetPathValue("field_id", "q")
	return r.WithContext(identity.WithActor(r.Context(), identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker", Kind: "PERSON", PermissionCodes: []string{identity.PermissionConfigWrite}, ExpiresAt: time.Now().Add(time.Hour)}))
}
func TestQuestionDraftRejectsBodyIdentityAndDuplicateFields(t *testing.T) {
	for _, body := range []string{`{"expected_version":0,"mode":"MANUAL","actor_id":"forged"}`, `{"expected_version":0,"mode":"MANUAL","mode":"API"}`, `{"mode":"MANUAL"}`} {
		fixture := &draftFixture{}
		recorder := httptest.NewRecorder()
		questionDraftHandler{store: fixture}.write(recorder, draftRequest(body))
		if recorder.Code != http.StatusBadRequest || fixture.writes != 0 {
			t.Fatalf("body %s: status %d, writes %d", body, recorder.Code, fixture.writes)
		}
	}
}
func TestQuestionDraftWritesExactVersionAndNeverCaches(t *testing.T) {
	fixture := &draftFixture{}
	recorder := httptest.NewRecorder()
	questionDraftHandler{store: fixture}.write(recorder, draftRequest(`{"expected_version":2,"mode":"API","endpoint":"https://bank.example/status","result_type":"BOOLEAN"}`))
	if recorder.Code != http.StatusOK || fixture.writes != 1 || fixture.expected != 2 || fixture.input.Mode != "API" || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status %d, fixture %+v, headers %v", recorder.Code, fixture, recorder.Header())
	}
}

func TestQuestionDraftReadRequiresPersonAndDoesNotCache(t *testing.T) {
	fixture := &draftFixture{}
	recorder := httptest.NewRecorder()
	r := draftRequest("")
	questionDraftHandler{store: fixture}.read(recorder, r)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" || !strings.Contains(recorder.Body.String(), `"mode":"API"`) {
		t.Fatalf("status %d, headers %v, body %s", recorder.Code, recorder.Header(), recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	questionDraftHandler{store: fixture}.read(recorder, r.WithContext(context.Background()))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("anonymous status %d", recorder.Code)
	}
}
