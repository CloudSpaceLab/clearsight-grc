package httpapi

import (
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"net/http"
)

// Until the repository enforces verified principal and subject visibility,
// deny these reads before querying; tenant/entity scope alone is insufficient.
func distributionReadUnavailable(w http.ResponseWriter, r *http.Request) bool {
	documentProtection(w)
	actor, ok := distributionActor(w, r)
	if !ok {
		return true
	}
	if _, ok = distributionLegalEntity(w, r, actor, r.URL.Query().Get("legal_entity_id")); !ok {
		return true
	}
	httpx.WriteError(w, http.StatusServiceUnavailable, "distribution_read_unavailable", "Form requests unavailable. Try again later.")
	return true
}
