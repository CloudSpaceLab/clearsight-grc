package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) dispatchFormDistribution(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	var request createFormDistributionRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The distribution request must be valid JSON.")
		return
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, request.LegalEntityID)
	if !ok {
		return
	}
	prepared, err := service.Prepare(r.Context(), evidence.CreateDistributionInput{
		TenantID: actor.TenantID, LegalEntityID: legalEntityID,
		FormTemplateID: request.FormTemplateID, FormTemplateVersion: request.FormTemplateVersion,
		SubjectType: request.SubjectType, SubjectID: request.SubjectID, Title: request.Title, Purpose: request.Purpose,
		AccessPolicy: request.AccessPolicy, EstimatedMinutes: request.EstimatedMinutes,
		Deadline: request.Deadline, RouteExpiresAt: request.RouteExpiresAt, ReminderPolicy: request.ReminderPolicy,
		CreatedBy: actor.PrincipalID, Recipients: request.Recipients,
	})
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}

	issued := []evidence.IssuedAccessRoute{}
	if hasExternalTO(prepared.Recipients) {
		access, available := a.formDistributionAccessService(w)
		if !available {
			return
		}
		issued, err = access.EnsureDistributionAccessRoutes(r.Context(), prepared.Distribution.TenantID, prepared.Distribution.LegalEntityID, prepared.Distribution.ID, actor.PrincipalID)
		if err != nil {
			writeFormAccessAdminError(w, err)
			return
		}
	}

	opened, err := service.Open(r.Context(), prepared.Distribution.TenantID, prepared.Distribution.LegalEntityID, prepared.Distribution.ID, prepared.Distribution.Version, actor.PrincipalID)
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}
	response := distributionBundleJSON(opened)
	if len(issued) > 0 {
		secureNoStore(w)
		response["issued_access_routes"] = issued
	}
	httpx.WriteJSON(w, http.StatusCreated, response)
}

func hasExternalTO(recipients []evidence.DistributionRecipient) bool {
	for _, recipient := range recipients {
		if recipient.Role == evidence.RecipientTo && recipient.Type == evidence.RecipientExternalAudience && recipient.State != evidence.DistributionRecipientRevoked {
			return true
		}
	}
	return false
}
