package httpapi

import (
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/metricview"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/runtimecontext"
)

type organizationMetricBucket struct {
	Key     string `json:"key"`
	ScopeID string `json:"scope_id,omitempty"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Value   int    `json:"value"`
}

type organizationMetricBreakdown struct {
	SourceID           string                     `json:"source_id"`
	MetricID           string                     `json:"metric_id"`
	DefinitionRevision string                     `json:"definition_revision"`
	Count              int                        `json:"count"`
	ScopeID            string                     `json:"scope_id,omitempty"`
	Items              []organizationMetricBucket `json:"items"`
}

func (a *API) domainMetricOrganizationBreakdown(w http.ResponseWriter, r *http.Request) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "metric_breakdown_scope_unavailable", "Choose an eligible legal entity before viewing organization breakdown.")
		return
	}
	metricID := strings.TrimSpace(r.PathValue("metric_id"))
	definitionRevision := strings.TrimSpace(r.URL.Query().Get("definition_revision"))
	sourceID := strings.TrimSpace(r.URL.Query().Get("source_id"))
	organizationScopeID := strings.TrimSpace(r.URL.Query().Get("organization_scope_id"))
	if _, ok := metricview.DomainDefinition(metricID); !ok || definitionRevision != metricview.DomainDefinitionRevision || sourceID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "metric_breakdown_filter_invalid", "Choose a current domain metric snapshot.")
		return
	}
	reader, ok := a.deps.MetricMembership.(metricview.OrganizationMembershipReader)
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_breakdown_unavailable", "Organization breakdown is unavailable. Try again.")
		return
	}
	resolver, ok := a.deps.RuntimeContext.(runtimecontext.HierarchyResolver)
	if !ok {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_breakdown_unavailable", "Organization breakdown is unavailable. Try again.")
		return
	}
	hierarchy, err := resolver.ResolveHierarchy(r.Context(), runtimecontext.Scope{
		TenantID: actor.TenantID, LegalEntityID: actor.LegalEntityID, PrincipalID: actor.PrincipalID,
	})
	if err != nil || hierarchy.OrganizationScopesTruncated || hierarchy.State == runtimecontext.HierarchyUnavailable {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_breakdown_hierarchy_incomplete", "Organization breakdown is unavailable until the full authorized hierarchy can be verified.")
		return
	}

	if organizationScopeID != "" {
		if _, scopeErr := a.resolveOrganizationScopeSelection(r.Context(), actor, organizationScopeID, true); scopeErr != nil {
			writeOrganizationScopeRequestError(w, scopeErr, "This organization scope is not available for risk breakdown.")
			return
		}
	}

	raw, err := reader.CountSnapshotMembersByOrganization(
		r.Context(), actor.TenantID, actor.LegalEntityID, organizationScopeID,
		sourceID, metricID, definitionRevision,
	)
	switch {
	case errors.Is(err, metricview.ErrMetricMembershipInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "metric_breakdown_filter_invalid", "Choose a current domain metric snapshot.")
		return
	case errors.Is(err, metricview.ErrMetricMembershipNotFound):
		httpx.WriteError(w, http.StatusNotFound, "metric_breakdown_not_found", "The exact metric snapshot is no longer available.")
		return
	case err != nil:
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_breakdown_unavailable", "Organization breakdown is unavailable. Try again.")
		return
	}

	items := organizeMetricMemberCounts(raw.Items, hierarchy.OrganizationScopes, organizationScopeID)
	total := 0
	for _, item := range items {
		total += item.Value
	}
	if total != raw.Count {
		httpx.WriteError(w, http.StatusServiceUnavailable, "metric_breakdown_inconsistent", "Organization breakdown does not match the metric snapshot.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, organizationMetricBreakdown{
		SourceID: raw.SourceID, MetricID: raw.MetricID, DefinitionRevision: raw.DefinitionRevision,
		Count: raw.Count, ScopeID: organizationScopeID, Items: items,
	})
}

func organizeMetricMemberCounts(
	counts []metricview.OrganizationMemberCount,
	nodes []runtimecontext.ScopeNode,
	selectedScopeID string,
) []organizationMetricBucket {
	nodeByID := make(map[string]runtimecontext.ScopeNode, len(nodes))
	for _, node := range nodes {
		if strings.TrimSpace(node.ID) != "" {
			nodeByID[node.ID] = node
		}
	}
	values := make(map[string]*organizationMetricBucket)
	add := func(key, scopeID, label, kind string, value int) {
		if value <= 0 {
			return
		}
		item := values[key]
		if item == nil {
			item = &organizationMetricBucket{Key: key, ScopeID: scopeID, Label: label, Kind: kind}
			values[key] = item
		}
		item.Value += value
	}

	for _, count := range counts {
		if count.Count <= 0 {
			continue
		}
		scopeID := strings.TrimSpace(count.OrganizationScopeID)
		if scopeID == "" {
			add("unattributed", "", "Unattributed", "UNATTRIBUTED", count.Count)
			continue
		}
		node, known := nodeByID[scopeID]
		if !known {
			add("unavailable", "", "Organization unavailable", "UNAVAILABLE", count.Count)
			continue
		}
		if selectedScopeID != "" {
			if scopeID == selectedScopeID {
				add("direct", "", "Direct", "DIRECT", count.Count)
				continue
			}
			child, ok := immediateChildForSelected(node, selectedScopeID, nodeByID)
			if !ok {
				add("unavailable", "", "Organization unavailable", "UNAVAILABLE", count.Count)
				continue
			}
			clickableID := ""
			if child.Filterable {
				clickableID = child.ID
			}
			add("scope:"+child.ID, clickableID, child.Name, "ORGANIZATION_SCOPE", count.Count)
			continue
		}
		root := topOrganizationAncestor(node, nodeByID)
		clickableID := ""
		if root.Filterable {
			clickableID = root.ID
		}
		add("scope:"+root.ID, clickableID, root.Name, "ORGANIZATION_SCOPE", count.Count)
	}

	result := make([]organizationMetricBucket, 0, len(values))
	for _, item := range values {
		result = append(result, *item)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			if result[i].Kind == "ORGANIZATION_SCOPE" {
				return true
			}
			if result[j].Kind == "ORGANIZATION_SCOPE" {
				return false
			}
		}
		if result[i].Value != result[j].Value {
			return result[i].Value > result[j].Value
		}
		return strings.ToLower(result[i].Label) < strings.ToLower(result[j].Label)
	})
	return result
}

func topOrganizationAncestor(node runtimecontext.ScopeNode, nodes map[string]runtimecontext.ScopeNode) runtimecontext.ScopeNode {
	current := node
	seen := map[string]struct{}{current.ID: {}}
	for {
		parent, ok := nodes[current.ParentID]
		if !ok {
			return current
		}
		if _, cycle := seen[parent.ID]; cycle {
			return current
		}
		seen[parent.ID] = struct{}{}
		current = parent
	}
}

func immediateChildForSelected(
	node runtimecontext.ScopeNode,
	selectedScopeID string,
	nodes map[string]runtimecontext.ScopeNode,
) (runtimecontext.ScopeNode, bool) {
	current := node
	seen := map[string]struct{}{current.ID: {}}
	for {
		if current.ParentID == selectedScopeID {
			return current, true
		}
		parent, ok := nodes[current.ParentID]
		if !ok {
			return runtimecontext.ScopeNode{}, false
		}
		if _, cycle := seen[parent.ID]; cycle {
			return runtimecontext.ScopeNode{}, false
		}
		seen[parent.ID] = struct{}{}
		current = parent
	}
}
