package httpapi

import (
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
	"strconv"
)

func (a *API) getAdoptedAssessmentSetup(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, _, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	items, err := service.AdoptedAssessmentOptions(r.Context(), actor, r.PathValue("id"))
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (a *API) createAdoptedAssessment(w http.ResponseWriter, r *http.Request) {
	service, ok := a.monitoringService(w)
	if !ok {
		return
	}
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	var body startFormCollectionRequest
	if err = httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, 400, "invalid_request", "Check the assessment period and respondent.")
		return
	}
	input := body.StartCollectionInput
	if input.AdoptionID == "" {
		httpx.WriteError(w, 422, "adoption_required", "Select an approved Program adoption.")
		return
	}
	input.ProgramID = aggregate.Program.ID
	input.LegalEntityID = aggregate.Program.LegalEntityID
	value, err := service.StartCollection(r.Context(), actor, input)
	if err != nil {
		writeMonitoringError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}
func (a *API) getInternalResponseDraft(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	query := r.URL.Query()
	value, err := service.GetInternalAssessmentDraft(r.Context(), r.PathValue("id"), evidence.CorrectionReference(query.Get("submission_id"), query.Get("review_id")))
	if writeAssessmentResponseError(w, err) {
		return
	}
	httpx.WriteJSON(w, 200, value)
}
func (a *API) saveInternalResponseDraft(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	var input evidence.SaveDraftInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, 400, "invalid_request", "Check the draft answers.")
		return
	}
	value, err := service.SaveInternalResponseDraft(r.Context(), r.PathValue("id"), input)
	if writeAssessmentResponseError(w, err) {
		return
	}
	httpx.WriteJSON(w, 200, value)
}
func (a *API) listProgramAssessmentReviews(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	// Review receipts contain review rationales and document references. Apply
	// the current Program read policy before querying this scoped population.
	if _, err = a.programForActor(r.Context(), aggregate, nil); err != nil {
		httpx.WriteError(w, 404, "not_found", "Program not found.")
		return
	}
	items, err := service.SubmissionReviews(r.Context(), actor.TenantID, actor.LegalEntityID, aggregate.Program.ID)
	if writeAssessmentResponseError(w, err) {
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"items": items, "limited": len(items) == 500})
}
func (a *API) reviewProgramAssessment(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	if err = a.requireMonitoringResponsibility(r.Context(), actor.TenantID, aggregate.Program, "PROGRAM", aggregate.Program.ID, authority.ResponsibilityAuthorizer, aggregate.Program.AuthorityPrincipalID, "program.assessment.review", 3); err != nil {
		httpx.WriteError(w, 403, "review_forbidden", "The current Program authority must independently review this assessment.")
		return
	}
	var input struct {
		TenantID      string `json:"tenant_id,omitempty"`
		LegalEntityID string `json:"legal_entity_id,omitempty"`
		evidence.RecordSubmissionReviewInput
	}
	if err = httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, 400, "invalid_request", "Review every question group and give a decision reason.")
		return
	}
	// Identity is obtained by the evidence service from verified request context.
	if _, err = identity.Require(r.Context()); err != nil {
		httpx.WriteError(w, 401, "identity_required", "Sign in to review this assessment.")
		return
	}
	value, err := service.ReviewInternalSubmission(r.Context(), aggregate.Program.ID, r.PathValue("request_id"), input.RecordSubmissionReviewInput)
	if writeAssessmentResponseError(w, err) {
		return
	}
	httpx.WriteJSON(w, 200, value)
}
func writeAssessmentResponseError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, evidence.ErrNotFound), errors.Is(err, evidence.ErrRecipientMismatch):
		httpx.WriteError(w, 404, "not_found", "The assessment is not available to this account.")
	case errors.Is(err, evidence.ErrVersionConflict):
		httpx.WriteError(w, 409, "version_conflict", "This assessment changed. Reload it before saving again.")
	case errors.Is(err, evidence.ErrRequestClosed):
		httpx.WriteError(w, 409, "assessment_closed", "The assessment is closed or its deadline has passed.")
	case errors.Is(err, evidence.ErrDraftInvalid):
		httpx.WriteError(w, 422, "invalid_answers", "Check the answers, group decisions and required reasons.")
	default:
		httpx.WriteError(w, 500, "assessment_failed", "The assessment could not be saved. Try again or contact your administrator.")
	}
	return true
}

func (a *API) getProgramAssessmentWorkspace(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	_, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	if _, err = a.programForActor(r.Context(), aggregate, nil); err != nil {
		httpx.WriteError(w, 404, "not_found", "Program not found.")
		return
	}
	value, err := service.ProgramAssessmentWorkspace(r.Context(), aggregate.Program.ID)
	if writeAssessmentResponseError(w, err) {
		return
	}
	httpx.WriteJSON(w, 200, value)
}
func (a *API) getOwnAssessmentSubmission(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, 401, "identity_required", "Sign in to view your submitted response.")
		return
	}
	req, err := service.GetRequestForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, r.PathValue("id"))
	if err != nil || !evidence.RequestAssignedTo(req, actor.PrincipalID) {
		httpx.WriteError(w, 404, "not_found", "Submitted response not found.")
		return
	}
	value, err := service.GetSubmissionForRequest(r.Context(), actor.TenantID, req.ID)
	if writeAssessmentResponseError(w, err) {
		return
	}
	if value.SubmittedBy != actor.PrincipalID {
		httpx.WriteError(w, 404, "not_found", "Submitted response not found.")
		return
	}
	httpx.WriteJSON(w, 200, value)
}

func (a *API) getProgramAssessmentReview(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	_, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	if _, err = a.programForActor(r.Context(), aggregate, nil); err != nil {
		httpx.WriteError(w, 404, "not_found", "Program not found.")
		return
	}
	value, err := service.ProgramSubmissionReview(r.Context(), aggregate.Program.ID, r.PathValue("review_id"))
	if writeAssessmentResponseError(w, err) {
		return
	}
	httpx.WriteJSON(w, 200, value)
}

func (a *API) getProgramAssessmentHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.evidenceService(w)
	if !ok {
		return
	}
	actor, aggregate, err := a.bindMonitoringProgram(r, r.PathValue("id"))
	if err != nil {
		writeMonitoringScopeError(w, err)
		return
	}
	if _, err := a.programForActor(r.Context(), aggregate, nil); err != nil {
		httpx.WriteError(w, 404, "not_found", "Program not found.")
		return
	}
	params := r.URL.Query()
	query := evidence.AssessmentHistoryQuery{ProgramID: aggregate.Program.ID, RequestID: r.PathValue("request_id"), SubmissionID: params.Get("submission_id")}
	if params.Has("before_revision") {
		query.BeforeRevision, err = strconv.ParseInt(params.Get("before_revision"), 10, 64)
	}
	if err == nil && params.Has("before_review") {
		query.BeforeReview, err = strconv.ParseInt(params.Get("before_review"), 10, 64)
	}
	if err != nil || query.BeforeRevision < 0 || query.BeforeReview < 0 {
		httpx.WriteError(w, 400, "invalid_cursor", "Choose a valid response or review history page.")
		return
	}
	value, err := service.ProgramAssessmentHistory(r.Context(), query)
	if writeAssessmentResponseError(w, err) {
		return
	}
	if value.CanReview {
		value.CanReview = a.requireMonitoringResponsibility(r.Context(), actor.TenantID, aggregate.Program,
			"PROGRAM", aggregate.Program.ID, authority.ResponsibilityAuthorizer, aggregate.Program.AuthorityPrincipalID,
			"program.assessment.review", 3) == nil
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(w, 200, value)
}
