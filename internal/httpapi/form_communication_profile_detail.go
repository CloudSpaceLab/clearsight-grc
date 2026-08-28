package httpapi

import (
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) getCommunicationProfile(w http.ResponseWriter, r *http.Request) {
	service, actor, legalEntityID, ok := a.communicationContext(w, r, r.URL.Query().Get("legal_entity_id"))
	if !ok {
		return
	}
	version, ok := communicationVersion(w, r.PathValue("version"))
	if !ok {
		return
	}
	value, err := service.GetProfile(r.Context(), actor.TenantID, legalEntityID, version)
	if err != nil {
		writeCommunicationError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}
