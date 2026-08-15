package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

type sourceEventRequest struct {
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
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "Use a verified service identity to send this source event.")
		return
	}
	if actor.Kind != "SERVICE" {
		httpx.WriteError(w, http.StatusForbidden, "service_identity_required", "Use a verified service identity to send this source event.")
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
		httpx.WriteError(w, http.StatusConflict, "source_event_out_of_order", "Send a source event with a position after the last accepted event, or replay the same event ID.")
	case errors.Is(err, sourceaccess.ErrSchemaDrift):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "source_schema_drift", "This source event does not match the configured fields. Correct the event payload or update the source binding before retrying.")
	default:
		writeSourceCatalogError(w, err)
	}
}
