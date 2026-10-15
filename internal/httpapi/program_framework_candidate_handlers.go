package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/frameworkcatalog"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// listProgramFrameworkCandidates is read-only. Matching a clause is never an
// applicability decision, approval, evidence assessment or scoring input.
func (a *API) listProgramFrameworkCandidates(w http.ResponseWriter, r *http.Request) {
	if a.deps.FrameworkCatalog == nil || a.deps.Continuity == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_catalog_unavailable", "Standards are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in to view standard requirements.")
		return
	}
	if actor.LegalEntityID == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "legal_entity_required", "Select a legal entity.")
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "Choose a page size from 1 to 100.")
			return
		}
	}
	program, err := a.deps.Continuity.GetProgram(r.Context(), actor.TenantID, r.PathValue("id"))
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	if program.Program.LegalEntityID != actor.LegalEntityID {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	visible, err := a.programForActor(r.Context(), program, nil)
	if err != nil {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	adoptionID := r.PathValue("adoption_id")
	var editionID string
	for _, adoption := range visible.FrameworkAdoptions {
		if adoption.ID == adoptionID && adoption.Status == continuity.FrameworkAdoptionActive {
			editionID = adoption.EditionID
			break
		}
	}
	if editionID == "" {
		writeContinuityError(w, continuity.ErrNotFound)
		return
	}
	edition, err := a.deps.FrameworkCatalog.GetFrameworkEdition(r.Context(), editionID)
	if err != nil || edition.ID != editionID {
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_edition_unavailable", "This standard edition could not be verified.")
		return
	}
	after := strings.TrimSpace(r.URL.Query().Get("after_code"))
	if len(after) > 128 || after != r.URL.Query().Get("after_code") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cursor", "Check the clause cursor.")
		return
	}
	clauses, err := a.deps.FrameworkCatalog.ListClauses(r.Context(), edition.ID, after, limit)
	if err != nil {
		if errors.Is(err, frameworkcatalog.ErrInvalidPublication) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_cursor", "Check the clause cursor.")
			return
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, "framework_clauses_unavailable", "Standard clauses could not be loaded.")
		return
	}
	candidates, err := frameworkcatalog.PreviewClauseReuse(visible, frameworkcatalog.Edition{
		ID: edition.ID, ReleaseSHA256: edition.ReleaseSHA256,
	}, clauses, time.Now().UTC())
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "framework_mapping_unavailable", "Requirement matches could not be evaluated.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"program_id": visible.Program.ID,
		"program_version": visible.Program.Version,
		"adoption_id": adoptionID,
		"edition_id": editionID,
		"candidates": candidates,
	})
}
