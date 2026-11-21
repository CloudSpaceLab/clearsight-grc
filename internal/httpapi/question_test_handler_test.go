package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type questionFormStub struct {
	value monitoring.FormTemplate
	err   error
}

func (s questionFormStub) GetLibraryForm(context.Context, string, int64) (monitoring.FormTemplate, error) {
	return s.value, s.err
}

type questionGuardStub struct {
	err      error
	decision commandauth.Decision
}

func (s questionGuardStub) Authorize(context.Context, commandauth.Request) (commandauth.Decision, error) {
	return s.decision, s.err
}

type questionAuditStub struct {
	reserveErr, completeErr   error
	reservations, completions int
}

func (s *questionAuditStub) Reserve(context.Context, sourceaccess.QuestionTestActor, string, int64, string, string, time.Time) (string, error) {
	s.reservations++
	return "receipt", s.reserveErr
}
func (s *questionAuditStub) Complete(context.Context, sourceaccess.QuestionTestActor, string, sourceaccess.QuestionTestOutcome, time.Time) error {
	s.completions++
	return s.completeErr
}

type questionTesterStub struct{ calls int }

func (s *questionTesterStub) Test(context.Context, sourceaccess.QuestionTestInput) (sourceaccess.QuestionTestResult, error) {
	s.calls++
	return sourceaccess.QuestionTestResult{Mode: "LIVE", Outcome: sourceaccess.QuestionTestValid, ResultType: "BOOLEAN", CheckedAt: time.Now().UTC()}, nil
}

func questionActor() identity.Actor {
	now := time.Now()
	return identity.Actor{TenantID: "bank", LegalEntityID: "entity", PrincipalID: "person", Kind: "PERSON", IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)}
}
func questionRequest(actor identity.Actor, body string) *http.Request {
	r := httptest.NewRequest("POST", "/api/v1/forms/templates/form/revisions/1/questions/q/test", strings.NewReader(body))
	r.SetPathValue("id", "form")
	r.SetPathValue("version", "1")
	r.SetPathValue("field_id", "q")
	return r.WithContext(identity.WithActor(r.Context(), actor))
}
func validQuestionHandler(audit *questionAuditStub, tester *questionTesterStub) questionTestHandler {
	return questionTestHandler{forms: questionFormStub{value: monitoring.FormTemplate{ID: "form", TenantID: "bank", LegalEntityID: "entity", Lifecycle: monitoring.Lifecycle{Version: 1, Status: monitoring.LifecycleDraft}, Fields: []monitoring.TemplateField{{ID: "q", Type: formcontract.TypeYesNo}}}}, guard: questionGuardStub{decision: commandauth.Decision{Allowed: true, Enforced: true, Resolution: &authority.Resolution{RuleID: "route", PolicyVersion: "v1"}}}, audit: audit, tester: tester}
}
func TestQuestionEndpointAcceptsScopedCanonicalTenantID(t *testing.T) {
	audit := &questionAuditStub{}
	h := validQuestionHandler(audit, &questionTesterStub{})
	h.forms = questionFormStub{value: monitoring.FormTemplate{ID: "form", TenantID: "canonical-uuid", LegalEntityID: "entity", Lifecycle: monitoring.Lifecycle{Version: 1, Status: monitoring.LifecycleDraft}, Fields: []monitoring.TemplateField{{ID: "q", Type: formcontract.TypeYesNo}}}}
	w := httptest.NewRecorder()
	h.serve(w, questionRequest(questionActor(), `{"mode":"SAMPLE","result_type":"BOOLEAN","sample":{"result":false}}`))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestQuestionEndpointRejectsResultTypeIncompatibleWithField(t *testing.T) {
	for _, tc := range []struct{ fieldType, resultType, mode string }{
		{"yes_no", "NUMBER", "LIVE"},
		{"decimal", "BOOLEAN", "SAMPLE"},
		{"percentage", "BOOLEAN", "LIVE"},
		{"short_text", "BOOLEAN", "SAMPLE"},
	} {
		t.Run(tc.fieldType+"_"+tc.resultType+"_"+tc.mode, func(t *testing.T) {
			audit, tester := &questionAuditStub{}, &questionTesterStub{}
			h := validQuestionHandler(audit, tester)
			form := h.forms.(questionFormStub).value
			form.Fields[0].Type = formcontract.Type(tc.fieldType)
			h.forms = questionFormStub{value: form}
			body := `{"mode":"SAMPLE","result_type":"` + tc.resultType + `","sample":{"result":true}}`
			if tc.mode == "LIVE" {
				body = `{"mode":"LIVE","result_type":"` + tc.resultType + `","endpoint":"https://example.test/check"}`
			}
			w := httptest.NewRecorder()
			h.serve(w, questionRequest(questionActor(), body))
			if w.Code < 400 || tester.calls != 0 || audit.reservations != 0 {
				t.Fatalf("status=%d calls=%d reservations=%d", w.Code, tester.calls, audit.reservations)
			}
		})
	}
}

func TestQuestionEndpointAcceptsCompatibleStoredFieldTypes(t *testing.T) {
	for _, fieldType := range []formcontract.Type{formcontract.TypeDecimal, formcontract.TypePercentage} {
		t.Run(string(fieldType), func(t *testing.T) {
			audit := &questionAuditStub{}
			h := validQuestionHandler(audit, &questionTesterStub{})
			form := h.forms.(questionFormStub).value
			form.Fields[0].Type = fieldType
			h.forms = questionFormStub{value: form}
			h.tester = sourceaccess.NewQuestionTester()
			w := httptest.NewRecorder()
			h.serve(w, questionRequest(questionActor(), `{"mode":"SAMPLE","result_type":"NUMBER","sample":{"result":12.5}}`))
			if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"outcome":"VALID"`) || audit.completions != 1 {
				t.Fatalf("status=%d body=%s completions=%d", w.Code, w.Body.String(), audit.completions)
			}
		})
	}
}

func TestQuestionEndpointReservesAndCompletesAuditBeforeReturning(t *testing.T) {
	audit := &questionAuditStub{}
	tester := &questionTesterStub{}
	h := validQuestionHandler(audit, tester)
	w := httptest.NewRecorder()
	h.serve(w, questionRequest(questionActor(), `{"mode":"LIVE","result_type":"BOOLEAN","endpoint":"https://example.test/check"}`))
	if w.Code != 200 || audit.reservations != 1 || audit.completions != 1 || tester.calls != 1 {
		t.Fatalf("status=%d audit=%+v calls=%d body=%s", w.Code, audit, tester.calls, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "example.test") || strings.Contains(w.Body.String(), "receipt") {
		t.Fatal("sensitive context exposed")
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store")
	}
}
func TestQuestionEndpointFailsClosedOnScopeAuthorityAndAuditFailures(t *testing.T) {
	for _, scenario := range []string{"missing actor", "wrong entity", "authority", "audit", "limited", "missing field", "identity injection", "duplicate mode", "unenforced", "missing authority version"} {
		t.Run(scenario, func(t *testing.T) {
			audit := &questionAuditStub{}
			tester := &questionTesterStub{}
			h := validQuestionHandler(audit, tester)
			actor := questionActor()
			body := `{"mode":"SAMPLE","result_type":"BOOLEAN","sample":{"result":false}}`
			switch scenario {
			case "missing actor":
				actor.PrincipalID = ""
			case "wrong entity":
				actor.LegalEntityID = "other"
			case "authority":
				h.guard = questionGuardStub{err: commandauth.ErrNotAuthorized}
			case "audit":
				audit.reserveErr = errors.New("down")
			case "limited":
				audit.reserveErr = sourceaccess.ErrQuestionTestRateLimit
			case "missing field":
				h.forms = questionFormStub{value: monitoring.FormTemplate{ID: "form", TenantID: "bank", LegalEntityID: "entity", Lifecycle: monitoring.Lifecycle{Version: 1, Status: monitoring.LifecycleDraft}}}
			case "identity injection":
				body = `{"mode":"SAMPLE","result_type":"BOOLEAN","sample":{"result":false},"principal_id":"another"}`
			case "duplicate mode":
				body = `{"mode":"SAMPLE","mode":"LIVE","result_type":"BOOLEAN","endpoint":"https://example.test"}`
			case "unenforced":
				h.guard = questionGuardStub{decision: commandauth.Decision{Allowed: true, Enforced: false}}
			case "missing authority version":
				h.guard = questionGuardStub{decision: commandauth.Decision{Allowed: true, Enforced: true, Resolution: &authority.Resolution{RuleID: "route"}}}
			}
			w := httptest.NewRecorder()
			h.serve(w, questionRequest(actor, body))
			if w.Code < 400 || tester.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, tester.calls, w.Body.String())
			}
		})
	}
}
func TestQuestionEndpointDoesNotReturnSuccessIfAuditCompletionFails(t *testing.T) {
	audit := &questionAuditStub{completeErr: errors.New("unavailable")}
	tester := &questionTesterStub{}
	h := validQuestionHandler(audit, tester)
	w := httptest.NewRecorder()
	h.serve(w, questionRequest(questionActor(), `{"mode":"SAMPLE","result_type":"BOOLEAN","sample":{"result":false}}`))
	if w.Code != 503 || tester.calls != 1 {
		t.Fatalf("status=%d calls=%d", w.Code, tester.calls)
	}
	var body map[string]any
	if json.NewDecoder(bytes.NewReader(w.Body.Bytes())).Decode(&body) != nil {
		t.Fatal("not JSON")
	}
}

func TestQuestionEndpointWithoutConfiguredServicesIsUnavailable(t *testing.T) {
	w := httptest.NewRecorder()
	(&API{}).TestAPIQuestion(w, questionRequest(questionActor(), `{"mode":"SAMPLE","result_type":"BOOLEAN","sample":{"result":false}}`))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("missing dependencies: status=%d headers=%v", w.Code, w.Header())
	}
}

func TestQuestionEndpointChecksRealSampleWithoutContactingEndpoint(t *testing.T) {
	audit := &questionAuditStub{}
	h := validQuestionHandler(audit, &questionTesterStub{})
	h.tester = sourceaccess.NewQuestionTester()
	w := httptest.NewRecorder()
	h.serve(w, questionRequest(questionActor(), `{"mode":"SAMPLE","result_type":"BOOLEAN","sample":{"result":false}}`))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"outcome":"VALID"`) || !strings.Contains(w.Body.String(), `"mode":"SAMPLE"`) || audit.completions != 1 {
		t.Fatalf("sample status=%d body=%s completions=%d", w.Code, w.Body.String(), audit.completions)
	}
}

func TestQuestionEndpointRouteRequiresConfigurationPermission(t *testing.T) {
	const path = "/api/v1/forms/templates/{id}/revisions/{version}/questions/{field_id}/test"
	var spec routeSpec
	found := false
	for _, route := range (&API{}).routes() {
		if route.Method == http.MethodPost && route.Path == path {
			spec = route
			found = true
			break
		}
	}
	if !found || spec.Class != routeAuthenticatedOperation || spec.Permission != identity.PermissionConfigWrite {
		t.Fatalf("unexpected route: %#v", spec)
	}
	called := false
	handler := (&API{}).routeAccess(spec, func(http.ResponseWriter, *http.Request) { called = true })
	w := httptest.NewRecorder()
	handler(w, questionRequest(questionActor(), `{}`))
	if w.Code != http.StatusForbidden || called {
		t.Fatalf("unauthorized access status=%d called=%v", w.Code, called)
	}
	actor := questionActor()
	actor.PermissionCodes = []string{identity.PermissionConfigWrite}
	w = httptest.NewRecorder()
	handler(w, questionRequest(actor, `{}`))
	if !called {
		t.Fatalf("permitted actor rejected: %d", w.Code)
	}
}
