package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
	"net/http"
)

// Form owners receive only the active lookup selector, never connection settings.
func (a *API) getFormValidationBinding(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, 401, "identity_required", "Sign in to choose a validation source.")
		return
	}
	catalog, ok := a.sourceCatalog(w)
	if !ok {
		return
	}
	capture, ok := a.evidenceService(w)
	if !ok {
		return
	}
	version, ok := catalogVersion(w, r)
	if !ok {
		return
	}
	binding, err := catalog.Binding(r.Context(), actor.TenantID, r.PathValue("binding_id"), version)
	if err != nil {
		httpx.WriteError(w, 404, "source_not_found", "The validation source is unavailable in this organization.")
		return
	}
	if err = capture.ValidateActiveSourcesForEntity(r.Context(), actor.TenantID, actor.LegalEntityID, []string{binding.SourceID}); err != nil {
		httpx.WriteError(w, 404, "source_not_found", "The validation source is unavailable in this organization.")
		return
	}
	lookup := false
	for _, operation := range binding.Operations {
		if operation == sourceaccess.OperationLookup {
			lookup = true
		}
	}
	if binding.Status != sourceaccess.RevisionActive || !lookup || len(binding.KeyFields) != 1 {
		httpx.WriteError(w, 422, "source_not_ready", "Choose an active source with a single lookup key.")
		return
	}
	httpx.WriteJSON(w, 200, map[string]any{"binding_id": binding.BindingID, "version": binding.Version, "name": binding.Name, "status": binding.Status, "operations": binding.Operations, "key_fields": binding.KeyFields, "selected_fields": binding.SelectedFields})
}
