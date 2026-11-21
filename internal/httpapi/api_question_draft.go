package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/apiquestion"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type questionDraftStore interface {
	ReadDraft(context.Context, string, int64, string) (apiquestion.Draft, error)
	WriteDraft(context.Context, string, int64, string, int64, apiquestion.DraftInput) (apiquestion.Receipt, error)
}
type questionDraftHandler struct{ store questionDraftStore }

// These handlers are intentionally not registered until their authority and
// activation paths pass the release checks.
func (a *API) getQuestionDraft(w http.ResponseWriter, r *http.Request) {
	questionDraftHandler{store: a.deps.QuestionDrafts}.read(w, r)
}
func (a *API) writeQuestionDraft(w http.ResponseWriter, r *http.Request) {
	questionDraftHandler{store: a.deps.QuestionDrafts}.write(w, r)
}

type questionDraftCommand struct {
	ExpectedVersion *int64 `json:"expected_version"`
	Mode            string `json:"mode"`
	Endpoint        string `json:"endpoint,omitempty"`
	ResultType      string `json:"result_type,omitempty"`
}

func (h questionDraftHandler) write(w http.ResponseWriter, r *http.Request) {
	secureNoStore(w)
	form, version, field, ok := h.authorizedRequest(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	content, err := io.ReadAll(r.Body)
	var command questionDraftCommand
	if err != nil || decodeQuestionDraft(content, &command) != nil || command.ExpectedVersion == nil || *command.ExpectedVersion < 0 {
		httpx.WriteError(w, http.StatusBadRequest, "question_draft_invalid", "Check the question settings.")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 16 || len(key) > 128 {
		httpx.WriteError(w, http.StatusBadRequest, "question_draft_invalid", "Check the question settings.")
		return
	}
	receipt, err := h.store.WriteDraft(r.Context(), form, version, field, *command.ExpectedVersion, apiquestion.DraftInput{Mode: command.Mode, Endpoint: command.Endpoint, ResultType: command.ResultType, Key: key})
	if err != nil {
		writeQuestionDraftError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, receipt)
}

func (h questionDraftHandler) read(w http.ResponseWriter, r *http.Request) {
	secureNoStore(w)
	form, version, field, ok := h.authorizedRequest(w, r)
	if !ok {
		return
	}
	draft, err := h.store.ReadDraft(r.Context(), form, version, field)
	if err != nil {
		writeQuestionDraftError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, draft)
}

func (h questionDraftHandler) authorizedRequest(w http.ResponseWriter, r *http.Request) (string, int64, string, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" || !identity.HasPermission(actor, identity.PermissionConfigWrite) {
		httpx.WriteError(w, http.StatusForbidden, "question_draft_denied", "Question settings unavailable. Check your access.")
		return "", 0, "", false
	}
	if h.store == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "question_draft_unavailable", "Question settings unavailable. Try again later.")
		return "", 0, "", false
	}
	form, field := strings.TrimSpace(r.PathValue("id")), strings.TrimSpace(r.PathValue("field_id"))
	version, err := strconv.ParseInt(r.PathValue("version"), 10, 64)
	if err != nil || version < 1 || form == "" || field == "" || len(field) > 80 {
		httpx.WriteError(w, http.StatusBadRequest, "question_draft_invalid", "Check the form and question.")
		return "", 0, "", false
	}
	return form, version, field, true
}

func writeQuestionDraftError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apiquestion.ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "question_draft_invalid", "Check the question settings.")
	case errors.Is(err, apiquestion.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "question_draft_conflict", "Question settings changed. Reload before saving.")
	default:
		httpx.WriteError(w, http.StatusNotFound, "question_draft_unavailable", "Question settings unavailable. Check your access.")
	}
}

func decodeQuestionDraft(payload []byte, command *questionDraftCommand) error {
	return decodeStrictDraftObject(payload, command)
}

func decodeStrictDraftObject(payload []byte, command any) error {
	d := json.NewDecoder(bytes.NewReader(payload))
	start, err := d.Token()
	if err != nil || start != json.Delim('{') {
		return errors.New("invalid object")
	}
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return errors.New("duplicate field")
		}
		seen[key] = true
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return err
		}
	}
	if _, err := d.Token(); err != nil {
		return err
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("trailing data")
	}
	strict := json.NewDecoder(bytes.NewReader(payload))
	strict.DisallowUnknownFields()
	return strict.Decode(command)
}
