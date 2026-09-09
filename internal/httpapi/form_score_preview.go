package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

const maxScorePreviewAnswers = 500

type formScorePreviewRequest struct {
	FormTemplateVersion int64                               `json:"form_template_version"`
	Answers             map[string]formcontract.AnswerValue `json:"answers"`
}

func (a *API) previewFormScore(w http.ResponseWriter, r *http.Request) {
	if a.deps.Monitoring == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "form_preview_unavailable", "This form score cannot be calculated right now.")
		return
	}
	var request formScorePreviewRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The score preview must be valid JSON.")
		return
	}
	if request.FormTemplateVersion < 1 {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "score_preview_invalid", "Choose a form revision.")
		return
	}
	if len(request.Answers) > maxScorePreviewAnswers {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "score_preview_invalid", "Use 500 answers or fewer.")
		return
	}
	if message := previewAnswerValidationMessage(request.Answers); message != "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "score_preview_invalid", message)
		return
	}
	form, err := a.deps.Monitoring.GetLibraryForm(r.Context(), r.PathValue("id"), request.FormTemplateVersion)
	if err != nil {
		writeFormScorePreviewError(w, err)
		return
	}
	if form.ScoreProfile == nil || form.ScoringMode == "" || form.ScoringMode == formcontract.ScoringNone || !previewAnswersBelongToForm(request.Answers, form.Fields) {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "score_preview_invalid", "This form revision does not have a valid advanced score profile for these answers.")
		return
	}
	contract := formcontract.Contract{Presentation: form.Presentation, ScoringMode: form.ScoringMode, ScoreProfile: form.ScoreProfile, Sections: form.Sections, Fields: form.Fields}
	result, err := formcontract.EvaluateScoreProfile(*form.ScoreProfile, contract, request.Answers)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "score_preview_invalid", "The stored score rules could not evaluate these answers.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func previewAnswerValidationMessage(answers map[string]formcontract.AnswerValue) string {
	for fieldID, answer := range answers {
		if len(strings.TrimSpace(fieldID)) == 0 || len(fieldID) > 128 {
			return "An answer has an invalid question reference. Reload the form and try again."
		}
		if answer.Text != nil && len(*answer.Text) > 16384 {
			return "Use 16,384 bytes or fewer for each text answer."
		}
		if len(answer.Values) > 100 {
			return "Use 100 selections or fewer for each answer."
		}
		if len(answer.ArtifactIDs) > 100 {
			return "Use 100 files or fewer for each answer."
		}
		for _, value := range answer.Values {
			if len(value) > 2048 {
				return "Use 2,048 bytes or fewer for each selected answer."
			}
		}
		for _, value := range answer.ArtifactIDs {
			if len(value) > 512 {
				return "An answer has an invalid file reference. Reopen the evidence request and try again."
			}
		}
		if document := answer.Document; document != nil {
			if len(document.ArtifactID) > 512 {
				return "An answer has an invalid file reference. Reopen the evidence request and try again."
			}
			if len(document.DocumentType) > 128 {
				return "Use 128 bytes or fewer for the document type."
			}
			if len(document.Reference) > 2048 {
				return "Use 2,048 bytes or fewer for the document reference."
			}
			if len(document.IssuedBy) > 512 {
				return "Use 512 bytes or fewer for the document issuer."
			}
			if len(document.IssuedOn) > 64 || len(document.ExpiresOn) > 64 {
				return "Use 64 bytes or fewer for each document date."
			}
		}
	}
	return ""
}

func previewAnswersBelongToForm(answers map[string]formcontract.AnswerValue, fields []formcontract.Field) bool {
	known := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		known[field.ID] = struct{}{}
	}
	for fieldID := range answers {
		if _, ok := known[fieldID]; !ok {
			return false
		}
	}
	return true
}

func writeFormScorePreviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, monitoring.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "form_revision_not_found", "The selected form revision was not found in this legal entity.")
	case errors.Is(err, monitoring.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "score_preview_invalid", "Choose a valid form revision and try again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "score_preview_failed", "The form score could not be calculated.")
	}
}
