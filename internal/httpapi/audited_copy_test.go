package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/commandauth"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
)

func TestAuditedWorkflowErrorsPreserveConditionAndRecovery(t *testing.T) {
	for _, tt := range []struct {
		name          string
		write         func(http.ResponseWriter, error)
		err           error
		status        int
		code, message string
	}{
		{"changed assessment", writeResponseAssessmentError, evidence.ErrAssessmentConflict, 409, "assessment_changed", "This response or assessment has changed. Reload before saving."},
		{"assessment unavailable", writeResponseAssessmentError, errors.New("database detail"), 503, "assessment_unavailable", "Assessment unavailable. Try again."},
		{"assessment authority", writeResponseAssessmentError, evidence.ErrAssessmentForbidden, 403, "assessment_not_permitted", "You cannot assess these fields under the current review route. Ask the form owner to check reviewer responsibilities."},
		{"organization scope", writeCommandAuthorizationError, commandauth.ErrTenantMismatch, 403, "tenant_not_allowed", "This request is outside your organization."},
		{"conversion conflict", writeDocumentConversionError, continuity.ErrDuplicate, 409, "conversion_identity_conflict", "This proposal conflicts with an existing import. Reload the import and review the proposal."},
		{"conversion failed", writeDocumentConversionError, errors.New("database detail"), 422, "conversion_failed", "The Program requirement or control could not be created. No approval was recorded."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.write(w, tt.err)
			assertAPIError(t, w, tt.status, tt.code, tt.message)
		})
	}
}

func TestFormErrorsExplainReviewConfigurationAndConflict(t *testing.T) {
	_, invalid := formcontract.Normalize(formcontract.Contract{Fields: []formcontract.Field{{ID: "report", Label: "Report", Type: formcontract.TypeFile, Assessment: &formcontract.FieldAssessment{Mode: formcontract.AssessmentNone, Required: true}}}})
	if invalid == nil {
		t.Fatal("invalid review accepted")
	}
	for _, tt := range []struct {
		err           error
		status        int
		code, message string
	}{
		{errors.Join(monitoring.ErrInvalid, invalid), 422, "form_invalid", "Enable review for Report before requiring a review or defining its outcomes."},
		{monitoring.ErrConflict, 409, "form_conflict", "This form or saved view has changed. Reload before saving."},
		{monitoring.ErrMakerChecker, 409, "form_conflict", "A different authorized person must approve this form revision."},
		{monitoring.ErrInactive, 409, "form_conflict", "This form revision is not active. Choose an active revision."},
	} {
		w := httptest.NewRecorder()
		writeFormsError(w, tt.err)
		assertAPIError(t, w, tt.status, tt.code, tt.message)
	}
}

func assertAPIError(t *testing.T, w *httptest.ResponseRecorder, status int, code, message string) {
	t.Helper()
	var body struct{ Error, Message string }
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != status || body.Error != code || body.Message != message {
		t.Fatalf("status=%d body=%s; want %d %s %q", w.Code, w.Body.String(), status, code, message)
	}
}

func TestScorePreviewReportsActualInvalidInput(t *testing.T) {
	handler, _ := scorePreviewHandler(t)
	for _, tt := range []struct {
		name    string
		version int64
		answers map[string]formcontract.AnswerValue
		message string
	}{
		{"revision", 0, nil, "Choose a form revision."},
		{"negative revision", -1, nil, "Choose a form revision."},
		{"text bytes", 2, map[string]formcontract.AnswerValue{"certified": formcontract.TextAnswer(strings.Repeat("x", 16385))}, "Use 16,384 bytes or fewer for each text answer."},
		{"field reference", 2, map[string]formcontract.AnswerValue{" ": {}}, "An answer has an invalid question reference. Reload the form and try again."},
		{"selected values", 2, map[string]formcontract.AnswerValue{"certified": {Values: make([]string, 101)}}, "Use 100 selections or fewer for each answer."},
		{"selection bytes", 2, map[string]formcontract.AnswerValue{"certified": {Values: []string{strings.Repeat("x", 2049)}}}, "Use 2,048 bytes or fewer for each selected answer."},
		{"files", 2, map[string]formcontract.AnswerValue{"certified": {ArtifactIDs: make([]string, 101)}}, "Use 100 files or fewer for each answer."},
		{"file reference", 2, map[string]formcontract.AnswerValue{"certified": {ArtifactIDs: []string{strings.Repeat("x", 513)}}}, "An answer has an invalid file reference. Reopen the evidence request and try again."},
		{"document reference", 2, map[string]formcontract.AnswerValue{"certified": {Document: &formcontract.DocumentAnswer{ArtifactID: strings.Repeat("x", 513)}}}, "An answer has an invalid file reference. Reopen the evidence request and try again."},
		{"document metadata", 2, map[string]formcontract.AnswerValue{"certified": {Document: &formcontract.DocumentAnswer{Reference: strings.Repeat("x", 2049)}}}, "Use 2,048 bytes or fewer for the document reference."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(formScorePreviewRequest{FormTemplateVersion: tt.version, Answers: tt.answers})
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/config/form-templates/form-a/score-preview", bytes.NewReader(body)))
			assertAPIError(t, w, 422, "score_preview_invalid", tt.message)
		})
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/config/form-templates/form-a/score-preview", bytes.NewReader(previewAnswersBody(501))))
	assertAPIError(t, w, 422, "score_preview_invalid", "Use 500 answers or fewer.")
}
