package httpapi

import (
	"net/http"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

// verifiedWorkActor is the shared authenticated actor boundary for read-only
// Work projections. Callers still validate their own supported query filters.
func verifiedWorkActor(w http.ResponseWriter, r *http.Request) (identity.Actor, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return identity.Actor{}, false
	}
	if strings.TrimSpace(actor.TenantID) == "" ||
		strings.TrimSpace(actor.LegalEntityID) == "" ||
		strings.TrimSpace(actor.PrincipalID) == "" {
		httpx.WriteError(w, http.StatusForbidden, "work_scope_required", "Your assigned work scope could not be verified.")
		return identity.Actor{}, false
	}
	return actor, true
}
