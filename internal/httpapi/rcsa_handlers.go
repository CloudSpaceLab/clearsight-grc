package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/rcsa"
)

func (a *API) rcsaService(w http.ResponseWriter) (*rcsa.Service, bool) {
	if a == nil || a.deps.RCSA == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "rcsa_unavailable", "RCSA information is unavailable. Try again.")
		return nil, false
	}
	return a.deps.RCSA, true
}

func (a *API) rcsaActorScope(w http.ResponseWriter, r *http.Request) (identity.Actor, rcsa.Scope, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil || strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "rcsa_scope_unavailable", "Choose an eligible legal entity before working with RCSA.")
		return identity.Actor{}, rcsa.Scope{}, false
	}
	return actor, rcsa.Scope{TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID}, true
}

func (a *API) createRCSACycle(w http.ResponseWriter, r *http.Request) {
	service, ok := a.rcsaService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.rcsaActorScope(w, r)
	if !ok {
		return
	}
	var input rcsa.CreateInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeRCSAError(w, rcsa.ErrInvalid)
		return
	}
	input.TenantID = scope.TenantID
	input.LegalEntityID = scope.LegalEntityID
	input.FirstLineOwnerID = actor.PrincipalID
	input.ActorID = actor.PrincipalID
	value, err := service.Create(r.Context(), input)
	if err != nil {
		writeRCSAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) getRCSACycle(w http.ResponseWriter, r *http.Request) {
	service, ok := a.rcsaService(w)
	if !ok {
		return
	}
	actor, scope, ok := a.rcsaActorScope(w, r)
	if !ok {
		return
	}
	value, err := service.Get(r.Context(), scope, r.PathValue("id"))
	if err != nil {
		writeRCSAError(w, err)
		return
	}
	if value.Cycle.FirstLineOwnerID != actor.PrincipalID {
		writeRCSAError(w, rcsa.ErrNotFound)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func writeRCSAError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, rcsa.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "rcsa_cycle_not_found", "This RCSA cycle is not available in the current scope.")
	case errors.Is(err, rcsa.ErrDuplicate):
		httpx.WriteError(w, http.StatusConflict, "rcsa_cycle_duplicate", "An RCSA cycle with this code already exists in this legal entity.")
	case errors.Is(err, rcsa.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "rcsa_cycle_conflict", "The RCSA cycle changed. Reload it and try again.")
	case errors.Is(err, rcsa.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "rcsa_cycle_invalid", "The RCSA cycle request is not valid. Review its Risk population and try again.")
	default:
		httpx.WriteError(w, http.StatusServiceUnavailable, "rcsa_unavailable", "RCSA information is unavailable. No change was made; try again.")
	}
}
