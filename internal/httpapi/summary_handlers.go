package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type matterSummaryRead struct {
	continuity.MatterSummary
	OwnerDisplayName       string `json:"owner_display_name,omitempty"`
	OrganizationScopeLabel string `json:"organization_scope_label,omitempty"`
}

type matterSummaryPageRead struct {
	Items       []matterSummaryRead `json:"items"`
	NextCursor  string              `json:"next_cursor,omitempty"`
	GeneratedAt time.Time           `json:"generated_at"`
}

type matterAggregateRead struct {
	continuity.MatterAggregate
	OwnerDisplayName       string `json:"owner_display_name,omitempty"`
	OrganizationScopeLabel string `json:"organization_scope_label,omitempty"`
}

func (a *API) listProgramSummaries(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	assignedToMe, parseOK := summaryBoolQuery(w, r, "assigned_to_me")
	if !parseOK {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	selection, err := a.resolveOrganizationScopeSelection(r.Context(), actor, r.URL.Query().Get("organization_scope_id"), true)
	if err != nil {
		writeOrganizationScopeRequestError(w, err, "This organization scope is not available for Programs.")
		return
	}
	page, err := service.ListProgramSummaries(r.Context(), tenant, continuity.SummaryQuery{
		Search: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"),
		OverallState: r.URL.Query().Get("overall_state"), Jurisdiction: r.URL.Query().Get("jurisdiction"),
		OrganizationScopeID: selection.ID, OrganizationScopeIDs: selection.IDs,
		AssignedToMe: assignedToMe, Cursor: r.URL.Query().Get("cursor"), Limit: limit,
	})
	if err != nil {
		writeSummaryError(w, err, "Programs could not be loaded.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) listMatterSummaries(w http.ResponseWriter, r *http.Request) {
	service, ok := a.continuityService(w)
	if !ok {
		return
	}
	tenant, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	priority := 0
	if value := r.URL.Query().Get("priority"); value != "" {
		var parseErr error
		priority, parseErr = strconv.Atoi(value)
		if parseErr != nil {
			priority = -1
		}
	}
	assignedToMe, parseOK := summaryBoolQuery(w, r, "assigned_to_me")
	if !parseOK {
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	page, err := service.ListMatterSummaries(r.Context(), tenant, continuity.SummaryQuery{
		Search:       r.URL.Query().Get("q"),
		Status:       r.URL.Query().Get("status"),
		ProgramID:    r.URL.Query().Get("program_id"),
		MatterType:   r.URL.Query().Get("matter_type"),
		DueCondition: r.URL.Query().Get("due"),
		Priority:     priority,
		AssignedToMe: assignedToMe,
		Cursor:       r.URL.Query().Get("cursor"),
		Limit:        limit,
	})
	if err != nil {
		writeSummaryError(w, err, "Issues and changes could not be loaded.")
		return
	}
	page.Items = filterMatterSummaries(r.Context(), page.Items)
	httpx.WriteJSON(w, http.StatusOK, a.matterSummaryPageRead(r.Context(), actor, page))
}

func (a *API) matterSummaryPageRead(ctx context.Context, actor identity.Actor, page continuity.MatterSummaryPage) matterSummaryPageRead {
	ownerIDsByEntity := map[string][]string{}
	for _, item := range page.Items {
		entity := strings.TrimSpace(item.Matter.LegalEntityID)
		ownerID := strings.TrimSpace(item.Matter.OwnerPrincipalID)
		if entity != "" && ownerID != "" {
			ownerIDsByEntity[entity] = append(ownerIDsByEntity[entity], ownerID)
		}
	}
	ownerLabelsByEntity := make(map[string]map[string]string, len(ownerIDsByEntity))
	for entity, ownerIDs := range ownerIDsByEntity {
		ownerLabelsByEntity[entity] = a.exactAssessmentLabels(ctx, actor, entity, ownerIDs)
	}
	scopeLabels := a.matterSummaryOrganizationScopeLabels(ctx, actor)
	items := make([]matterSummaryRead, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, matterSummaryRead{
			MatterSummary:          item,
			OwnerDisplayName:       ownerLabelsByEntity[item.Matter.LegalEntityID][item.Matter.OwnerPrincipalID],
			OrganizationScopeLabel: scopeLabels[item.Matter.OrganizationScopeID],
		})
	}
	return matterSummaryPageRead{Items: items, NextCursor: page.NextCursor, GeneratedAt: page.GeneratedAt}
}

func (a *API) matterAggregateRead(ctx context.Context, actor identity.Actor, aggregate continuity.MatterAggregate) matterAggregateRead {
	labels := a.exactAssessmentLabels(ctx, actor, aggregate.Matter.LegalEntityID, []string{aggregate.Matter.OwnerPrincipalID})
	return matterAggregateRead{
		MatterAggregate:        aggregate,
		OwnerDisplayName:       labels[aggregate.Matter.OwnerPrincipalID],
		OrganizationScopeLabel: a.matterSummaryOrganizationScopeLabels(ctx, actor)[aggregate.Matter.OrganizationScopeID],
	}
}

func (a *API) matterSummaryOrganizationScopeLabels(ctx context.Context, actor identity.Actor) map[string]string {
	labels := map[string]string{}
	resolver, ok := a.deps.RuntimeContext.(runtimecontext.HierarchyResolver)
	if !ok {
		return labels
	}
	hierarchy, err := resolver.ResolveHierarchy(ctx, runtimecontext.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	})
	if err != nil {
		return labels
	}
	for _, scope := range hierarchy.OrganizationScopes {
		label := strings.Join(scope.DepartmentPath, " / ")
		if label == "" {
			label = strings.TrimSpace(scope.Name)
		}
		if strings.TrimSpace(scope.ID) != "" && label != "" {
			labels[scope.ID] = label
		}
	}
	return labels
}

func writeSummaryError(w http.ResponseWriter, err error, fallback string) {
	if strings.Contains(strings.ToLower(err.Error()), "invalid cursor") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cursor", "The page cursor is invalid or no longer usable. Reload the first page.")
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), "invalid filter") {
		message := strings.TrimSpace(strings.TrimPrefix(err.Error(), "invalid filter:"))
		if message == "" {
			message = "Check the selected filters and try again."
		}
		httpx.WriteError(w, http.StatusBadRequest, "invalid_filter", message)
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "summary_failed", fallback)
}

func summaryBoolQuery(w http.ResponseWriter, r *http.Request, name string) (bool, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return false, true
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_filter", "Assigned to me must be true or false.")
		return false, false
	}
	return parsed, true
}
