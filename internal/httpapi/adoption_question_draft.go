package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/apiquestion"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type adoptionQuestionStore interface {
	ReadAdoptionManualDraft(context.Context, apiquestion.AdoptionManualDraft) (apiquestion.AdoptionQuestionDraft, error)
	WriteAdoptionManualDraft(context.Context, apiquestion.AdoptionManualDraft) (apiquestion.Receipt, error)
}
type adoptionQuestionDraftHandler struct{ store adoptionQuestionStore }

// These handlers remain unregistered until authority revocation and approval
// semantics are verified end to end.
func (a *API) getAdoptionQuestionDraft(w http.ResponseWriter, r *http.Request) {
	adoptionQuestionDraftHandler{store: a.deps.QuestionDrafts}.read(w, r)
}
func (a *API) writeAdoptionQuestionDraft(w http.ResponseWriter, r *http.Request) {
	adoptionQuestionDraftHandler{store: a.deps.QuestionDrafts}.write(w, r)
}

type adoptionQuestionCommand struct {
	ProgramVersion  *int64 `json:"program_version"`
	AdoptionVersion *int64 `json:"adoption_version"`
	ExpectedVersion *int64 `json:"expected_version"`
	Rationale       string `json:"rationale,omitempty"`
}

func (h adoptionQuestionDraftHandler) write(w http.ResponseWriter, r *http.Request) {
	secureNoStore(w)
	input, ok := h.requestScope(w, r)
	if !ok {
		return
	}
	if r.URL.RawQuery != "" {
		httpx.WriteError(w, http.StatusBadRequest, "adoption_question_invalid", "Check the question settings.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	body, err := io.ReadAll(r.Body)
	var command adoptionQuestionCommand
	if err != nil || decodeStrictDraftObject(body, &command) != nil || command.ProgramVersion == nil || command.AdoptionVersion == nil || command.ExpectedVersion == nil || *command.ProgramVersion < 1 || *command.AdoptionVersion < 1 || *command.ExpectedVersion < 0 {
		httpx.WriteError(w, http.StatusBadRequest, "adoption_question_invalid", "Check the question settings.")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 16 || len(key) > 128 {
		httpx.WriteError(w, http.StatusBadRequest, "adoption_question_invalid", "Check the question settings.")
		return
	}
	input.ProgramVersion = *command.ProgramVersion
	input.AdoptionVersion = *command.AdoptionVersion
	input.ExpectedVersion = *command.ExpectedVersion
	input.Rationale = command.Rationale
	input.Key = key
	receipt, err := h.store.WriteAdoptionManualDraft(r.Context(), input)
	if err != nil {
		writeQuestionDraftError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, receipt)
}

func (h adoptionQuestionDraftHandler) read(w http.ResponseWriter, r *http.Request) {
	secureNoStore(w)
	input, ok := h.requestScope(w, r)
	if !ok {
		return
	}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(query) != 2 || len(query["program_version"]) != 1 || len(query["adoption_version"]) != 1 {
		httpx.WriteError(w, http.StatusBadRequest, "adoption_question_invalid", "Check the question settings.")
		return
	}
	input.ProgramVersion, err = strconv.ParseInt(query.Get("program_version"), 10, 64)
	if err != nil || input.ProgramVersion < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "adoption_question_invalid", "Check the question settings.")
		return
	}
	input.AdoptionVersion, err = strconv.ParseInt(query.Get("adoption_version"), 10, 64)
	if err != nil || input.AdoptionVersion < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "adoption_question_invalid", "Check the question settings.")
		return
	}
	draft, err := h.store.ReadAdoptionManualDraft(r.Context(), input)
	if err != nil {
		writeQuestionDraftError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, draft)
}

func (h adoptionQuestionDraftHandler) requestScope(w http.ResponseWriter, r *http.Request) (apiquestion.AdoptionManualDraft, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || !identity.HasPermission(actor, identity.PermissionConfigWrite) {
		httpx.WriteError(w, http.StatusForbidden, "adoption_question_denied", "Question settings unavailable. Check your access.")
		return apiquestion.AdoptionManualDraft{}, false
	}
	if h.store == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "adoption_question_unavailable", "Question settings unavailable. Try again later.")
		return apiquestion.AdoptionManualDraft{}, false
	}
	input := apiquestion.AdoptionManualDraft{ProgramID: strings.TrimSpace(r.PathValue("id")), AdoptionID: strings.TrimSpace(r.PathValue("adoption_id")), FormID: strings.TrimSpace(r.PathValue("form_id")), FieldID: strings.TrimSpace(r.PathValue("field_id"))}
	input.FormVersion, err = strconv.ParseInt(r.PathValue("version"), 10, 64)
	if err != nil || input.FormVersion < 1 || input.ProgramID == "" || input.AdoptionID == "" || input.FormID == "" || input.FieldID == "" || len(input.FieldID) > 80 {
		httpx.WriteError(w, http.StatusBadRequest, "adoption_question_invalid", "Check the form and question.")
		return apiquestion.AdoptionManualDraft{}, false
	}
	return input, true
}
