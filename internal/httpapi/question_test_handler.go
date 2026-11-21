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
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type questionFormReader interface {
	GetLibraryForm(context.Context, string, int64) (monitoring.FormTemplate, error)
}
type questionAuthority interface {
	Authorize(context.Context, commandauth.Request) (commandauth.Decision, error)
}
type questionTester interface {
	Test(context.Context, sourceaccess.QuestionTestInput) (sourceaccess.QuestionTestResult, error)
}
type questionTestHandler struct {
	forms  questionFormReader
	guard  questionAuthority
	audit  sourceaccess.QuestionTestAudit
	tester questionTester
}

type questionTestRequest struct {
	Mode       string          `json:"mode"`
	ResultType string          `json:"result_type"`
	Endpoint   string          `json:"endpoint,omitempty"`
	Sample     json.RawMessage `json:"sample,omitempty"`
}

// TestAPIQuestion validates an example or performs one restricted public HTTPS
// read. It never stores the endpoint, sample, source response, or credentials,
// and does not create a check, observation, or assessment result.
func (a *API) TestAPIQuestion(w http.ResponseWriter, r *http.Request) {
	if a.deps.Monitoring == nil || a.deps.CommandGuard == nil || a.deps.QuestionTestAudit == nil {
		secureNoStore(w)
		httpx.WriteError(w, http.StatusServiceUnavailable, "question_test_unavailable", "Question test unavailable. Try again later.")
		return
	}
	h := questionTestHandler{forms: a.deps.Monitoring, guard: a.deps.CommandGuard, audit: a.deps.QuestionTestAudit, tester: sourceaccess.NewQuestionTester()}
	h.serve(w, r)
}

func (h questionTestHandler) serve(w http.ResponseWriter, r *http.Request) {
	secureNoStore(w)
	actor, err := identity.Require(r.Context())
	if err != nil || actor.Valid(time.Now()) != nil || !strings.EqualFold(actor.Kind, "PERSON") || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusUnauthorized, "question_test_denied", "Question test unavailable. Check your access.")
		return
	}
	if h.forms == nil || h.guard == nil || h.audit == nil || h.tester == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "question_test_unavailable", "Question test unavailable. Try again later.")
		return
	}
	formID, fieldID := strings.TrimSpace(r.PathValue("id")), strings.TrimSpace(r.PathValue("field_id"))
	version, err := strconv.ParseInt(r.PathValue("version"), 10, 64)
	if formID == "" || fieldID == "" || version < 1 || err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "question_test_invalid", "Check the form and question.")
		return
	}
	var input questionTestRequest
	r.Body = http.MaxBytesReader(w, r.Body, 70000)
	payload, readErr := io.ReadAll(r.Body)
	if readErr != nil || decodeQuestionTestRequest(payload, &input) != nil {
		httpx.WriteError(w, http.StatusBadRequest, "question_test_invalid", "Check the test details.")
		return
	}
	if (input.Mode != "SAMPLE" && input.Mode != "LIVE") || (input.ResultType != "BOOLEAN" && input.ResultType != "NUMBER") || (input.Mode == "SAMPLE" && (len(input.Sample) == 0 || input.Endpoint != "")) || (input.Mode == "LIVE" && (input.Endpoint == "" || len(input.Sample) > 0)) {
		httpx.WriteError(w, http.StatusBadRequest, "question_test_invalid", "Check the test details.")
		return
	}
	// The form reader resolves the verified tenant and entity in its scoped query.
	form, err := h.forms.GetLibraryForm(r.Context(), formID, version)
	if err != nil || form.ID != formID || form.Version != version || form.LegalEntityID != actor.LegalEntityID || form.Status != monitoring.LifecycleDraft {
		httpx.WriteError(w, http.StatusNotFound, "question_test_unavailable", "Form or question unavailable.")
		return
	}
	found := false
	for _, field := range form.Fields {
		if field.ID == fieldID && apiquestion.CompatibleQuestionResult(string(field.Type), input.ResultType) {
			found = true
			break
		}
	}
	if !found {
		httpx.WriteError(w, http.StatusNotFound, "question_test_unavailable", "Form or question unavailable.")
		return
	}
	decision, err := h.guard.Authorize(r.Context(), commandauth.Request{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, ObjectType: "FORM_TEMPLATE", ObjectID: formID, Responsibility: authority.ResponsibilityOwner, DecisionType: "forms.template.revise", Materiality: 2})
	if err != nil || !decision.Allowed || !decision.Enforced || decision.Resolution == nil || strings.TrimSpace(decision.Resolution.RuleID) == "" || strings.TrimSpace(decision.Resolution.PolicyVersion) == "" {
		httpx.WriteError(w, http.StatusForbidden, "question_test_denied", "Question test unavailable. Check your access.")
		return
	}
	scope := sourceaccess.QuestionTestActor{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID, AuthorityRuleID: decision.Resolution.RuleID, AuthorityPolicyVersion: decision.Resolution.PolicyVersion}
	attempt, err := h.audit.Reserve(r.Context(), scope, formID, version, fieldID, input.Mode, time.Now().UTC())
	if err != nil {
		if errors.Is(err, sourceaccess.ErrQuestionTestRateLimit) {
			httpx.WriteError(w, http.StatusTooManyRequests, "question_test_limit", "Too many question tests. Try again later.")
		} else {
			httpx.WriteError(w, http.StatusServiceUnavailable, "question_test_unavailable", "Question test unavailable. Try again later.")
		}
		return
	}
	result, testErr := h.tester.Test(r.Context(), sourceaccess.QuestionTestInput{Mode: input.Mode, ResultType: input.ResultType, Endpoint: input.Endpoint, Sample: input.Sample})
	outcome := result.Outcome
	if testErr != nil {
		outcome = sourceaccess.QuestionTestInvalid
	}
	completionCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 3*time.Second)
	defer cancel()
	if err := h.audit.Complete(completionCtx, scope, attempt, outcome, time.Now().UTC()); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "question_test_unavailable", "Question test unavailable. Try again later.")
		return
	}
	if testErr != nil {
		httpx.WriteError(w, http.StatusBadRequest, "question_test_invalid", "Check the test details.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func decodeQuestionTestRequest(payload []byte, input *questionTestRequest) error {
	reader := json.NewDecoder(bytes.NewReader(payload))
	start, err := reader.Token()
	if err != nil || start != json.Delim('{') {
		return errors.New("invalid object")
	}
	seen := map[string]bool{}
	for reader.More() {
		token, err := reader.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] {
			return errors.New("duplicate or invalid field")
		}
		seen[key] = true
		var value json.RawMessage
		if err := reader.Decode(&value); err != nil {
			return err
		}
	}
	if _, err := reader.Token(); err != nil {
		return err
	}
	if reader.Decode(&struct{}{}) != io.EOF {
		return errors.New("trailing data")
	}
	strict := json.NewDecoder(bytes.NewReader(payload))
	strict.DisallowUnknownFields()
	return strict.Decode(input)
}
