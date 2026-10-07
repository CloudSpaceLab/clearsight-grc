package httpapi

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) groupPostureMetrics(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a == nil || a.deps.GroupPostureMetrics == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_posture_unavailable", "Group posture is unavailable. Try again.")
		return
	}
	tenantID := strings.TrimSpace(actor.TenantID)
	principalID := strings.TrimSpace(actor.PrincipalID)
	if tenantID == "" || principalID == "" {
		httpx.WriteError(w, http.StatusForbidden, "group_posture_scope_unavailable", "Group posture is not available for this sign-in.")
		return
	}
	accessResolver, ok := a.deps.Access.(access.LegalEntityAccessResolver)
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_posture_unavailable", "Group posture is unavailable. Try again.")
		return
	}

	now := time.Now().UTC()
	start, end, valid := lossMetricPeriod(w, r, now)
	if !valid {
		return
	}
	entities, err := a.deps.GroupPostureMetrics.ActiveGroupEntities(r.Context(), tenantID, now)
	switch {
	case errors.Is(err, metricview.ErrGroupPostureInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "group_posture_filter_invalid", "Review the Group posture request.")
		return
	case errors.Is(err, metricview.ErrGroupPostureMissing):
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_posture_not_ready", "Group posture is not ready.")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_posture_unavailable", "Group posture is unavailable. Try again.")
		return
	}

	entityIDs := make([]string, 0, len(entities))
	for _, entity := range entities {
		entityIDs = append(entityIDs, entity.LegalEntityID)
	}
	allowed := make(map[string]struct{}, len(entityIDs))
	for offset := 0; offset < len(entityIDs); offset += access.MaxLegalEntityAccessBatchSize {
		endIndex := min(offset+access.MaxLegalEntityAccessBatchSize, len(entityIDs))
		resolved, resolveErr := accessResolver.ResolveLegalEntityAccess(
			r.Context(), tenantID, principalID, entityIDs[offset:endIndex],
		)
		if resolveErr != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "group_posture_unavailable", "Group posture is unavailable. Try again.")
			return
		}
		for _, item := range resolved {
			if slices.Contains(identity.NormalizePermissionCodes(item.PermissionCodes), identity.PermissionOversightRead) {
				allowed[item.LegalEntityID] = struct{}{}
			}
		}
	}
	authorizedIDs := make([]string, 0, len(allowed))
	for _, entity := range entities {
		if _, ok := allowed[entity.LegalEntityID]; ok {
			authorizedIDs = append(authorizedIDs, entity.LegalEntityID)
		}
	}
	if len(authorizedIDs) < 2 {
		httpx.WriteError(w, http.StatusForbidden, "group_scope_forbidden", "Group posture is not available for this sign-in.")
		return
	}

	bundle, err := a.deps.GroupPostureMetrics.GroupPosture(
		r.Context(), tenantID, authorizedIDs, start, end, now,
	)
	switch {
	case errors.Is(err, metricview.ErrGroupPostureInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "group_posture_filter_invalid", "Review the Group posture request.")
	case errors.Is(err, metricview.ErrGroupPostureMissing):
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_posture_not_ready", "Group posture is not ready.")
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_posture_unavailable", "Group posture is unavailable. Try again.")
	default:
		httpx.WriteJSON(w, http.StatusOK, bundle)
	}
}
