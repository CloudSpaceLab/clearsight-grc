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

type adoptionQuestionFixture struct {
	writes, reads int
	input         apiquestion.AdoptionManualDraft
}

func (f *adoptionQuestionFixture) WriteAdoptionManualDraft(_ context.Context, input apiquestion.AdoptionManualDraft) (apiquestion.Receipt, error) {
	f.writes++
	f.input = input
	return apiquestion.Receipt{ID: "receipt", Version: 1}, nil
}
func (f *adoptionQuestionFixture) ReadAdoptionManualDraft(_ context.Context, input apiquestion.AdoptionManualDraft) (apiquestion.AdoptionQuestionDraft, error) {
	f.reads++
	f.input = input
	return apiquestion.AdoptionQuestionDraft{Version: 1, Mode: "MANUAL"}, nil
}
func adoptionQuestionRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.SetPathValue("id", "program")
	r.SetPathValue("adoption_id", "adoption")
	r.SetPathValue("form_id", "form")
	r.SetPathValue("version", "2")
	r.SetPathValue("field_id", "question")
	r.Header.Set("Idempotency-Key", "11111111-1111-4111-8111-111111111111")
	actor := identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "maker", Kind: "PERSON", PermissionCodes: []string{identity.PermissionConfigWrite}, ExpiresAt: time.Now().Add(time.Hour)}
	return r.WithContext(identity.WithActor(r.Context(), actor))
}
func TestAdoptionQuestionRejectsForgedIdentityDuplicateAndMissingVersions(t *testing.T) {
	for _, body := range []string{`{"program_version":1,"adoption_version":1,"expected_version":0,"actor_id":"forged"}`, `{"program_version":1,"program_version":2,"adoption_version":1,"expected_version":0}`, `{"adoption_version":1,"expected_version":0}`} {
		f := &adoptionQuestionFixture{}
		w := httptest.NewRecorder()
		adoptionQuestionDraftHandler{store: f}.write(w, adoptionQuestionRequest(http.MethodPost, "/", body))
		if w.Code != http.StatusBadRequest || f.writes != 0 {
			t.Fatalf("body %s: %d writes=%d", body, w.Code, f.writes)
		}
	}
}
func TestAdoptionQuestionWritesExactScopeAndDoesNotCache(t *testing.T) {
	f := &adoptionQuestionFixture{}
	w := httptest.NewRecorder()
	adoptionQuestionDraftHandler{store: f}.write(w, adoptionQuestionRequest(http.MethodPost, "/", `{"program_version":3,"adoption_version":4,"expected_version":0,"rationale":"Manual review"}`))
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || f.writes != 1 || f.input.ProgramID != "program" || f.input.AdoptionID != "adoption" || f.input.FormID != "form" || f.input.FormVersion != 2 || f.input.FieldID != "question" || f.input.ProgramVersion != 3 || f.input.AdoptionVersion != 4 {
		t.Fatalf("status=%d input=%+v headers=%v", w.Code, f.input, w.Header())
	}
}
func TestAdoptionQuestionReadRejectsExtraOrAmbiguousScope(t *testing.T) {
	f := &adoptionQuestionFixture{}
	w := httptest.NewRecorder()
	adoptionQuestionDraftHandler{store: f}.read(w, adoptionQuestionRequest(http.MethodGet, "/?program_version=3&adoption_version=4", ""))
	if w.Code != http.StatusOK || f.reads != 1 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), `"mode":"MANUAL"`) {
		t.Fatalf("status=%d reads=%d body=%s", w.Code, f.reads, w.Body.String())
	}
	for _, path := range []string{"/?program_version=3&program_version=4&adoption_version=4", "/?program_version=3&adoption_version=4&actor_id=other"} {
		w = httptest.NewRecorder()
		adoptionQuestionDraftHandler{store: f}.read(w, adoptionQuestionRequest(http.MethodGet, path, ""))
		if w.Code != http.StatusBadRequest || f.reads != 1 {
			t.Fatalf("path %s: %d reads=%d", path, w.Code, f.reads)
		}
	}
}

func TestAdoptionQuestionHandlersFailClosedForAnonymousAndAreNotRegistered(t *testing.T) {
	f := &adoptionQuestionFixture{}
	r := adoptionQuestionRequest(http.MethodGet, "/?program_version=3&adoption_version=4", "")
	w := httptest.NewRecorder()
	adoptionQuestionDraftHandler{store: f}.read(w, r.WithContext(context.Background()))
	if w.Code != http.StatusForbidden || f.reads != 0 {
		t.Fatalf("anonymous status=%d reads=%d", w.Code, f.reads)
	}
	for _, route := range (&API{}).routes() {
		if strings.Contains(route.Path, "question-draft") || strings.Contains(route.Path, "manual-draft") {
			t.Fatalf("unapproved draft route registered: %s", route.Path)
		}
	}
}
