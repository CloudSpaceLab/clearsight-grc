package httpapi

import (
	"errors"
	"github.com/CloudSpaceLab/clearsight-grc/internal/documentimport"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/registermigration"
	"net/http"
)

func (a *API) migrationAvailable(w http.ResponseWriter) bool {
	if a.deps.RegisterMigrations == nil {
		httpx.WriteError(w, 503, "migration_unavailable", "Risk register migration is unavailable. Try again later.")
		return false
	}
	return true
}
func (a *API) getRiskRegisterMigration(w http.ResponseWriter, r *http.Request) {
	if !a.migrationAvailable(w) {
		return
	}
	v, err := a.deps.RegisterMigrations.View(r.Context(), r.PathValue("id"))
	writeMigrationResult(w, v, err)
}
func (a *API) saveRiskRegisterMigration(w http.ResponseWriter, r *http.Request) {
	if !a.migrationAvailable(w) {
		return
	}
	var input struct {
		ExpectedVersion int64                       `json:"expected_version"`
		SourceVersion   int64                       `json:"source_version"`
		Selection       registermigration.Selection `json:"selection"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, 400, "invalid_request", "The migration choices could not be read.")
		return
	}
	v, err := a.deps.RegisterMigrations.Save(r.Context(), r.PathValue("id"), input.ExpectedVersion, input.SourceVersion, input.Selection)
	writeMigrationResult(w, v, err)
}
func (a *API) commitRiskRegisterMigration(w http.ResponseWriter, r *http.Request) {
	if !a.migrationAvailable(w) {
		return
	}
	var input struct {
		ExpectedVersion int64 `json:"expected_version"`
	}
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.WriteError(w, 400, "invalid_request", "The migration version could not be read.")
		return
	}
	v, err := a.deps.RegisterMigrations.Import(r.Context(), r.PathValue("id"), input.ExpectedVersion)
	writeMigrationResult(w, v, err)
}
func writeMigrationResult(w http.ResponseWriter, v any, err error) {
	if err == nil {
		httpx.WriteJSON(w, 200, v)
		return
	}
	switch {
	case errors.Is(err, registermigration.ErrAuthority):
		httpx.WriteError(w, 403, "migration_authority", err.Error())
	case errors.Is(err, registermigration.ErrConflict):
		httpx.WriteError(w, 409, "migration_conflict", err.Error())
	case errors.Is(err, documentimport.ErrNotFound), errors.Is(err, registermigration.ErrNotFound):
		httpx.WriteError(w, 404, "migration_not_found", "Risk register migration not found.")
	case errors.Is(err, registermigration.ErrInvalid):
		httpx.WriteError(w, 422, "migration_invalid", err.Error())
	case errors.Is(err, registermigration.ErrSource):
		httpx.WriteError(w, 422, "migration_source", err.Error())
	default:
		httpx.WriteError(w, 422, "migration_unavailable", "The register could not be prepared. Check complete source rows, vendor matches and current assignment authority, then try again.")
	}
}
