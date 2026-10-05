package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/invalidation"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

const invalidationHeartbeat = 20 * time.Second

func (a *API) invalidationStream(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.Invalidations == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "updates_unavailable", "Live updates are unavailable. Reload to refresh current data.")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "updates_unavailable", "Live updates are unavailable. Reload to refresh current data.")
		return
	}
	events, cancel := a.deps.Invalidations.Subscribe(invalidation.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	})
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	refreshInvalidationWriteDeadline(w)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(invalidationHeartbeat)
	defer heartbeat.Stop()
	sessionLifetime := time.Until(actor.ExpiresAt)
	if sessionLifetime <= 0 {
		return
	}
	sessionExpiry := time.NewTimer(sessionLifetime)
	defer sessionExpiry.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-sessionExpiry.C:
			return
		case <-heartbeat.C:
			refreshInvalidationWriteDeadline(w)
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, open := <-events:
			if !open {
				return
			}
			payload, err := json.Marshal(struct {
				Revision string `json:"revision"`
			}{Revision: event.Revision})
			if err != nil {
				continue
			}
			refreshInvalidationWriteDeadline(w)
			if _, err := fmt.Fprintf(w, "event: invalidate\ndata: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func refreshInvalidationWriteDeadline(w http.ResponseWriter) {
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(invalidationHeartbeat + 10*time.Second))
}
