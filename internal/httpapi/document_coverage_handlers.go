package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentcoverage"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) documentCoverageService(w http.ResponseWriter) (*documentcoverage.Service, bool) {
	if a.deps.Coverage == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "document_coverage_unavailable", "Document coverage analysis is unavailable.")
		return nil, false
	}
	return a.deps.Coverage, true
}

func (a *API) getDocumentCoverage(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentCoverageService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	value, err := service.Get(r.Context(), documentcoverage.ReadInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, DocumentID: r.PathValue("id"),
		PrincipalID: actor.PrincipalID, Cursor: r.URL.Query().Get("cursor"), Limit: limit,
	})
	if err != nil {
		writeDocumentCoverageError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) reviewDocumentCoverage(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentCoverageService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input struct {
		ExpectedVersion int64                            `json:"expected_version"`
		Decisions       []documentcoverage.DecisionInput `json:"decisions"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	_, err = service.Review(r.Context(), documentcoverage.ReviewInput{
		TenantID: actor.TenantID, DocumentID: r.PathValue("id"), ReviewerID: actor.PrincipalID,
		ExpectedVersion: input.ExpectedVersion, Decisions: input.Decisions,
	})
	if err != nil {
		writeDocumentCoverageError(w, err)
		return
	}
	value, err := service.Get(r.Context(), documentcoverage.ReadInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, DocumentID: r.PathValue("id"), PrincipalID: actor.PrincipalID,
	})
	if err != nil {
		writeDocumentCoverageError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) recompareDocumentCoverage(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentCoverageService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if err := service.Recompare(r.Context(), actor.TenantID, r.PathValue("id")); err != nil {
		writeDocumentCoverageError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "QUEUED"})
}

func (a *API) applyDocumentCoverageSuggestion(w http.ResponseWriter, r *http.Request) {
	service, ok := a.documentCoverageService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	var input struct {
		ExpectedVersion int64 `json:"expected_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	documentID := r.PathValue("id")
	suggestionID := r.PathValue("suggestion_id")
	apply := documentcoverage.ApplySuggestionInput{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, DocumentID: documentID,
		SuggestionID: suggestionID, ExpectedVersion: input.ExpectedVersion, ActorID: actor.PrincipalID,
	}
	prepared, err := service.PrepareSuggestion(r.Context(), apply)
	if err != nil {
		writeDocumentCoverageError(w, err)
		return
	}
	if prepared.Suggestion.Type == documentcoverage.SuggestionLinkRequirement {
		value, err := service.ApplySuggestion(r.Context(), apply)
		if err != nil {
			writeDocumentCoverageError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, value)
		return
	}

	policy := commandPolicy{Responsibility: authority.ResponsibilityOwner, Materiality: 2}
	commandName := ""
	payload := map[string]any{
		"assessment_expected_version": input.ExpectedVersion,
		"document_id":                 documentID,
		"suggestion_id":               suggestionID,
		"candidate_id":                prepared.Candidate.ID,
		"suggestion_type":             prepared.Suggestion.Type,
	}
	switch prepared.Suggestion.Type {
	case documentcoverage.SuggestionAddRequirement:
		commandName = "program.requirement.add"
		policy.ObjectType = "PROGRAM"
		payload["program_id"] = prepared.Suggestion.ProgramID
		payload["expected_version"] = prepared.ProgramVersion
		r.SetPathValue("id", prepared.Suggestion.ProgramID)
	case documentcoverage.SuggestionCreateMatter:
		commandName = "matter.create"
		policy.ObjectType = "MATTER"
		policy.Materiality = 3
		payload["priority"] = 3
		r.SetPathValue("id", "")
	case documentcoverage.SuggestionCreateProgram:
		commandName = "program.create"
		policy.ObjectType = "PROGRAM"
		policy.BindLegalEntity = true
		r.SetPathValue("id", "")
	default:
		writeDocumentCoverageError(w, documentcoverage.ErrInvalidReview)
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "suggestion_apply_failed", "The suggested update could not be prepared.")
		return
	}
	restoreJSONBody(r, raw)
	a.command(commandName, policy, func(w http.ResponseWriter, r *http.Request) {
		payloadValues, _, err := commandPayload(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The suggestion command body is invalid.")
			return
		}
		assessmentVersion, ok := int64Value(payloadValues["assessment_expected_version"])
		if !ok || assessmentVersion < 1 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The current analysis version is required.")
			return
		}
		apply.ExpectedVersion = assessmentVersion
		value, err := service.ApplySuggestion(r.Context(), apply)
		if err != nil {
			writeDocumentCoverageError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, value)
	})(w, r)
}

func writeDocumentCoverageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, documentcoverage.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Document coverage analysis was not found.")
	case errors.Is(err, documentcoverage.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "This analysis changed. Reload it before saving your review.")
	case errors.Is(err, documentcoverage.ErrStaleAssessment):
		httpx.WriteError(w, http.StatusConflict, "assessment_stale", "Programs changed after this analysis. Compare the document again before applying changes.")
	case errors.Is(err, documentcoverage.ErrDocumentNotReady):
		httpx.WriteError(w, http.StatusConflict, "document_not_ready", "Document extraction must finish before coverage can be compared.")
	case errors.Is(err, documentcoverage.ErrInvalidReview), errors.Is(err, documentcoverage.ErrInvalidCursor):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "coverage_review_invalid", "Review choices or pagination details are invalid.")
	case errors.Is(err, continuity.ErrNotFound), errors.Is(err, continuity.ErrVersionConflict),
		errors.Is(err, continuity.ErrDuplicate), errors.Is(err, continuity.ErrTriggerDuplicate),
		errors.Is(err, continuity.ErrInvalidState), errors.Is(err, continuity.ErrClosureBlocked):
		writeContinuityError(w, err)
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "document_coverage_failed", "Document coverage could not be completed.")
	}
}
