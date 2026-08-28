package httpapi

import (
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

type governedDistributionAmendRequest struct {
	ExpectedVersion    int64                                 `json:"expected_version"`
	Deadline           *time.Time                            `json:"deadline,omitempty"`
	RouteExpiresAt     *time.Time                            `json:"route_expires_at,omitempty"`
	ReminderPolicy     *map[string]any                       `json:"reminder_policy,omitempty"`
	AddRecipients      []evidence.DistributionRecipientInput `json:"add_recipients,omitempty"`
	RevokeRecipientIDs []string                              `json:"revoke_recipient_ids,omitempty"`
}

func (a *API) amendGovernedFormDistribution(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.distributionMutationContext(w, r)
	if !ok {
		return
	}
	var request governedDistributionAmendRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The amendment request must be valid JSON.")
		return
	}
	value, err := service.Amend(r.Context(), actor.TenantID, legalEntityID, r.PathValue("id"), evidence.AmendDistributionInput{
		ExpectedVersion: request.ExpectedVersion,
		Deadline:        request.Deadline,
		RouteExpiresAt:  request.RouteExpiresAt,
		ReminderPolicy:  request.ReminderPolicy,
		AddRecipients:   request.AddRecipients,
		RevokeRecipientIDs: request.RevokeRecipientIDs,
		ActorID:            actor.PrincipalID,
	})
	if err != nil {
		writeFormDistributionError(w, err)
		return
	}

	response := map[string]any{"distribution": distributionBundleJSON(value.Bundle), "impact": value.Impact}
	if value.Impact.RecipientsAdded > 0 && hasExternalTO(value.Bundle.Recipients) && value.Bundle.Distribution.Status != evidence.DistributionLocked {
		if access, available := a.formDistributionAccessService(w); available {
			issued, ensureErr := access.EnsureDistributionAccessRoutes(r.Context(), value.Bundle.Distribution.TenantID, value.Bundle.Distribution.LegalEntityID, value.Bundle.Distribution.ID, actor.PrincipalID)
			if ensureErr != nil {
				response["access_routes_pending"] = true
			} else if len(issued) > 0 {
				secureNoStore(w)
				response["issued_access_routes"] = issued
			}
		} else {
			response["access_routes_pending"] = true
		}
	}
	httpx.WriteJSON(w, http.StatusOK, response)
}
