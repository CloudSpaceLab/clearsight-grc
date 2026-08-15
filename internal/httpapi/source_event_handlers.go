package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"net/http"
)

type sourceEventRequest struct {
	TenantID string                           `json:"tenant_id"`
	EventID  string                           `json:"event_id"`
	Position *sourceaccess.CheckpointPosition `json:"position,omitempty"`
	Payload  json.RawMessage                  `json:"payload"`
}

func (a *API) ingestSourceBindingEvent(w http.ResponseWriter, r *http.Request) {
	service, ok := a.sourceCatalog(w)
	if !ok {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified service identity is required.")
		return
	}
	if actor.Kind != "SERVICE" {
		httpx.WriteError(w, http.StatusForbidden, "service_identity_required", "Source events must be delivered by a verified service principal.")
		return
	}
	var input sourceEventRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := service.CaptureBindingChange(r.Context(), actor.TenantID, r.PathValue("id"), 0, sourceaccess.ChangeEvent{EventID: input.EventID, Position: input.Position, Payload: input.Payload})
	if err != nil {
		writeSourceEventError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusAccepted, result)
}
func writeSourceEventError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sourceaccess.ErrCheckpointConflict):
		httpx.WriteError(w, http.StatusConflict, "source_event_out_of_order", "The event position is not newer than the current governed checkpoint.")
	case errors.Is(err, sourceaccess.ErrSchemaDrift):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "source_schema_drift", "The event payload does not match the governed source schema.")
	default:
		writeSourceCatalogError(w, err)
	}
}
