package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
)

func (a *API) sourceCatalog(w http.ResponseWriter) (*sourceaccess.CatalogService, bool) {
	if a.deps.SourceCatalog == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "source_catalog_unavailable", "Connected-source configuration is unavailable.")
		return nil, false
	}
	return a.deps.SourceCatalog, true
}

func sourceCatalogActor(r *http.Request) (sourceaccess.CatalogActor, error) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		return sourceaccess.CatalogActor{}, err
	}
	return sourceaccess.CatalogActor{TenantID: actor.TenantID, PrincipalID: actor.PrincipalID}, nil
}

func (a *API) listSourceConnections(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	values, err := service.Connections(r.Context(), actor.TenantID, r.PathValue("source_id"), catalogLimit(r))
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createSourceConnectionDraft(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	var input sourceaccess.CreateConnectionDraftInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateConnectionDraft(r.Context(), actor, r.PathValue("source_id"), input)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) getSourceConnection(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	version, ok := catalogVersion(w, r)
	if !ok {
		return
	}
	value, err := service.Connection(r.Context(), actor.TenantID, r.PathValue("connection_id"), version)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) listSourceViews(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	values, err := service.Views(r.Context(), actor.TenantID, r.PathValue("connection_id"), catalogLimit(r))
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createSourceViewDraft(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	var input sourceaccess.CreateViewDraftInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateViewDraft(r.Context(), actor, r.PathValue("connection_id"), input)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) getSourceView(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	version, ok := catalogVersion(w, r)
	if !ok {
		return
	}
	value, err := service.View(r.Context(), actor.TenantID, r.PathValue("view_id"), version)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) inspectSourceView(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	version, ok := catalogVersion(w, r)
	if !ok {
		return
	}
	value, err := service.InspectView(r.Context(), actor.TenantID, r.PathValue("view_id"), version)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) listSourceBindings(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	values, err := service.Bindings(r.Context(), actor.TenantID, r.PathValue("view_id"), catalogLimit(r))
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": values})
}

func (a *API) createSourceBindingDraft(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	var input sourceaccess.CreateBindingDraftInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.CreateBindingDraft(r.Context(), actor, r.PathValue("view_id"), input)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, value)
}

func (a *API) getSourceBinding(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	version, ok := catalogVersion(w, r)
	if !ok {
		return
	}
	value, err := service.Binding(r.Context(), actor.TenantID, r.PathValue("binding_id"), version)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) previewSourceBinding(w http.ResponseWriter, r *http.Request) {
	service, actor, ok := a.sourceCatalogRequest(w, r)
	if !ok {
		return
	}
	version, ok := catalogVersion(w, r)
	if !ok {
		return
	}
	var input sourceaccess.PageRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	value, err := service.PreviewBinding(r.Context(), actor.TenantID, r.PathValue("binding_id"), version, input)
	if err != nil {
		writeSourceCatalogError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func (a *API) sourceCatalogWhereUsed(kind sourceaccess.CatalogUsageKind, pathName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service, actor, ok := a.sourceCatalogRequest(w, r)
		if !ok {
			return
		}
		value, err := service.WhereUsed(r.Context(), actor.TenantID, kind, r.PathValue(pathName), catalogLimit(r))
		if err != nil {
			writeSourceCatalogError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, value)
	}
}

func (a *API) sourceCatalogRequest(w http.ResponseWriter, r *http.Request) (*sourceaccess.CatalogService, sourceaccess.CatalogActor, bool) {
	service, ok := a.sourceCatalog(w)
	if !ok {
		return nil, sourceaccess.CatalogActor{}, false
	}
	actor, err := sourceCatalogActor(r)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return nil, sourceaccess.CatalogActor{}, false
	}
	return service, actor, true
}

func catalogVersion(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("version"))
	if raw == "" {
		return 0, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_version", "version must be a positive integer")
		return 0, false
	}
	return value, true
}

func catalogLimit(r *http.Request) int {
	value, _ := strconv.Atoi(strings.TrimSpace(r.URL.Query().Get("limit")))
	if value < 1 || value > sourceaccess.HardMaxCatalogListRows {
		return 100
	}
	return value
}

func writeSourceCatalogError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, sourceaccess.ErrCatalogNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Connected-source configuration was not found.")
	case errors.Is(err, sourceaccess.ErrCatalogConflict):
		httpx.WriteError(w, http.StatusConflict, "catalog_conflict", "Connected-source configuration changed. Reload before continuing.")
	case errors.Is(err, sourceaccess.ErrCatalogInvalid), errors.Is(err, sourceaccess.ErrDefinitionInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "catalog_invalid", "Connected-source configuration is invalid.")
	case errors.Is(err, sourceaccess.ErrLimitExceeded):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "source_limit_exceeded", "The requested source operation exceeds its activated limits.")
	case errors.Is(err, sourceaccess.ErrCapabilityUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "source_capability_unavailable", "The requested source capability is unavailable.")
	case errors.Is(err, sourceaccess.ErrCredentials), errors.Is(err, sourceaccess.ErrConnection):
		httpx.WriteError(w, http.StatusServiceUnavailable, "source_unavailable", "The connected source is unavailable.")
	case errors.Is(err, context.DeadlineExceeded):
		httpx.WriteError(w, http.StatusGatewayTimeout, "source_timeout", "The connected source did not respond within the permitted time.")
	case errors.Is(err, context.Canceled):
		httpx.WriteError(w, http.StatusRequestTimeout, "request_cancelled", "The source operation was cancelled.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "source_catalog_failed", "Connected-source configuration could not be processed.")
	}
}
