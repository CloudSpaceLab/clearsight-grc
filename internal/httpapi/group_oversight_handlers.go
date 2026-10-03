package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/CloudSpaceLab/clearsight-grc/internal/access"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/oversight"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

var errGroupOversightUnavailable = errors.New("group oversight unavailable")

func (a *API) groupOversight(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if a.deps.Oversight == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_unavailable", "Group posture is unavailable. Try again.")
		return
	}
	page, err := a.resolveGroupOversightEntities(r.Context(), actor)
	if err != nil {
		writeGroupOversightError(w, err)
		return
	}
	ids := make([]string, 0, len(page.Items))
	entities := make([]oversight.GroupEntity, 0, len(page.Items))
	for _, item := range page.Items {
		ids = append(ids, item.ID)
		entities = append(entities, oversight.GroupEntity{
			ID: item.ID, Code: item.Code, Name: item.Name, Jurisdiction: item.Jurisdiction,
		})
	}
	snapshots, err := a.deps.Oversight.GetMany(r.Context(), actor.TenantID, ids)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_unavailable", "Group posture is unavailable. Try again.")
		return
	}
	groupRoot, err := a.groupOversightRoot(r.Context(), actor)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_unavailable", "Group posture is unavailable. Try again.")
		return
	}
	group, err := oversight.BuildGroupSnapshot(groupRoot.ID, groupRoot.Name, entities, snapshots)
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_unavailable", "Group posture is unavailable. Try again.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"snapshot": group,
		"metrics":  metricview.FromGroupOversight(group),
	})
}

func (a *API) resolveGroupOversightEntities(ctx context.Context, actor identity.Actor) (access.OversightScopePage, error) {
	resolver, ok := a.deps.Access.(access.OversightScopeResolver)
	if !ok {
		return access.OversightScopePage{}, errGroupOversightUnavailable
	}
	page, err := resolver.ResolveOversightLegalEntities(ctx, actor.TenantID, actor.PrincipalID, 256)
	if err != nil {
		if errors.Is(err, access.ErrPrincipalUnavailable) {
			return access.OversightScopePage{}, errGroupOversightUnavailable
		}
		return access.OversightScopePage{}, err
	}
	if page.HasMore || len(page.Items) < 2 {
		return access.OversightScopePage{}, errGroupOversightUnavailable
	}
	return page, nil
}

func (a *API) groupOversightRoot(ctx context.Context, actor identity.Actor) (runtimecontext.ScopeNode, error) {
	resolver, ok := a.deps.RuntimeContext.(runtimecontext.HierarchyResolver)
	if !ok {
		return runtimecontext.ScopeNode{}, errGroupOversightUnavailable
	}
	hierarchy, err := resolver.ResolveHierarchy(ctx, runtimecontext.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	})
	if err != nil || hierarchy.Root.ID == "" || hierarchy.Root.Name == "" {
		return runtimecontext.ScopeNode{}, errGroupOversightUnavailable
	}
	return hierarchy.Root, nil
}

func (a *API) groupOversightAvailable(ctx context.Context, actor identity.Actor) bool {
	resolver, ok := a.deps.Access.(access.OversightScopeResolver)
	if !ok {
		return false
	}
	page, err := resolver.ResolveOversightLegalEntities(ctx, actor.TenantID, actor.PrincipalID, 2)
	return err == nil && len(page.Items) >= 2
}

func writeGroupOversightError(w http.ResponseWriter, err error) {
	if errors.Is(err, errGroupOversightUnavailable) {
		httpx.WriteError(w, http.StatusForbidden, "group_oversight_not_allowed", "Group posture is not available for this sign-in.")
		return
	}
	httpx.WriteError(w, http.StatusServiceUnavailable, "group_oversight_unavailable", "Group posture is unavailable. Try again.")
}
