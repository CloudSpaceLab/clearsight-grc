package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/authority"
	"github.com/CloudSpaceLab/clearsight-grc/internal/capture"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) live(w http.ResponseWriter, _ *http.Request) { httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "live"}) }
func (a *API) ready(w http.ResponseWriter, _ *http.Request) { httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready", "mode": "scaffold-memory"}) }
func (a *API) context(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tenant": map[string]string{"id": "bank-demo", "name": "ClearSight Demonstration Bank"}, "legal_entity": map[string]string{"id": "bank-ng", "name": "Demonstration Bank Nigeria"}, "actor": map[string]string{"id": "user-demo", "name": "Amaka Okafor", "role": "Control Assurance Lead"}, "policy_version": "demo-2026-08-05"})
}
func (a *API) today(w http.ResponseWriter, _ *http.Request) { httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": a.deps.Today.List(), "generated_at": time.Now().UTC()}) }

func (a *API) resolveAuthority(w http.ResponseWriter, r *http.Request) {
	var input authority.ResolveInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil { httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error()); return }
	resolution, err := a.deps.Authority.Resolve(input)
	if errors.Is(err, authority.ErrNoRoute) { httpx.WriteError(w, http.StatusUnprocessableEntity, "routing_failed", "No eligible route exists for the supplied scope and responsibility."); return }
	if err != nil { httpx.WriteError(w, http.StatusInternalServerError, "resolution_failed", "Authority could not be resolved."); return }
	httpx.WriteJSON(w, http.StatusOK, resolution)
}

func (a *API) getCaptureRequest(w http.ResponseWriter, r *http.Request) {
	request, err := a.deps.Capture.Get(r.PathValue("id"))
	if errors.Is(err, capture.ErrRequestNotFound) { httpx.WriteError(w, http.StatusNotFound, "not_found", "The request was not found."); return }
	if err != nil { httpx.WriteError(w, http.StatusInternalServerError, "request_failed", "The request could not be loaded."); return }
	httpx.WriteJSON(w, http.StatusOK, request)
}

func (a *API) submitCaptureRequest(w http.ResponseWriter, r *http.Request) {
	var submission capture.Submission
	if err := httpx.DecodeJSON(w, r, &submission); err != nil { httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error()); return }
	receipt, err := a.deps.Capture.Submit(r.PathValue("id"), submission)
	var validation capture.ValidationError
	switch {
	case errors.As(err, &validation): httpx.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", validation.Error())
	case errors.Is(err, capture.ErrRequestNotFound): httpx.WriteError(w, http.StatusNotFound, "not_found", "The request was not found.")
	case errors.Is(err, capture.ErrVersionConflict): httpx.WriteError(w, http.StatusConflict, "version_conflict", "The request changed. Reload before submitting.")
	case errors.Is(err, capture.ErrRequestClosed): httpx.WriteError(w, http.StatusConflict, "request_closed", "The request is no longer open.")
	case err != nil: httpx.WriteError(w, http.StatusInternalServerError, "submission_failed", "The response could not be submitted.")
	default: httpx.WriteJSON(w, http.StatusOK, receipt)
	}
}

func (a *API) redeemInvitation(w http.ResponseWriter, r *http.Request) {
	var body struct{ Token string `json:"token"` }
	if err := httpx.DecodeJSON(w, r, &body); err != nil { httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error()); return }
	session, err := a.deps.Invitations.Redeem(body.Token)
	if err != nil { httpx.WriteError(w, http.StatusUnauthorized, "invitation_unavailable", "This invitation is unavailable. Request a new invitation from the sender."); return }
	httpx.WriteJSON(w, http.StatusOK, session)
}
