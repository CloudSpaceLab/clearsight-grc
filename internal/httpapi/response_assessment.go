package httpapi

import (
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
)

func (a *API) getResponseAssessment(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	requested := ""
	if actor.LegalEntityID == "*" {
		requested = r.URL.Query().Get("legal_entity_id")
	}
	entity, ok := distributionLegalEntity(w, r, actor, requested)
	if !ok {
		return
	}
	result, err := service.GetResponseAssessment(r.Context(), actor.TenantID, entity, actor.PrincipalID, r.PathValue("revision_id"))
	if err != nil {
		writeResponseAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.responseAssessmentWithLabels(r.Context(), actor, entity, result))
}
func (a *API) recordResponseAssessment(w http.ResponseWriter, r *http.Request) {
	documentProtection(w)
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	var body struct {
		ExpectedVersion *int64                          `json:"expected_version"`
		Decisions       []evidence.FieldAssessmentInput `json:"decisions"`
		LegalEntityID   string                          `json:"legal_entity_id,omitempty"`
		ReviewerID      string                          `json:"reviewer_id,omitempty"`
		ActorID         string                          `json:"actor_id,omitempty"`
	}
	if err := httpx.DecodeJSON(w, r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "assessment_request_invalid", "Check the assessment entries and try again.")
		return
	}
	if body.ExpectedVersion == nil {
		writeResponseAssessmentError(w, evidence.ErrAssessmentInvalid)
		return
	}
	entity, ok := distributionLegalEntity(w, r, actor, body.LegalEntityID)
	if !ok {
		return
	}
	result, err := service.RecordResponseAssessment(r.Context(), actor.TenantID, entity, r.PathValue("revision_id"), evidence.RecordResponseAssessmentInput{ExpectedVersion: *body.ExpectedVersion, Decisions: body.Decisions})
	if err != nil {
		writeResponseAssessmentError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.responseAssessmentWithLabels(r.Context(), actor, entity, result))
}
func writeResponseAssessmentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evidence.ErrAssessmentConflict):
		httpx.WriteError(w, http.StatusConflict, "assessment_changed", "This response or assessment has changed. Reload before saving.")
	case errors.Is(err, evidence.ErrAssessmentForbidden):
		httpx.WriteError(w, http.StatusForbidden, "assessment_not_permitted", "You cannot assess these fields under the current review route. Ask the form owner to check reviewer responsibilities.")
	case errors.Is(err, evidence.ErrAssessmentInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "assessment_invalid", "Choose an approved outcome and enter a reason for each field you assess.")
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "response_not_found", "The submitted response was not found in your permitted work.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "assessment_unavailable", "Assessment unavailable. Try again.")
	}
}
