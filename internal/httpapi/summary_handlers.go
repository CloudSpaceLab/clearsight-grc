package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

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
	page, err := service.ListProgramSummaries(r.Context(), tenant, continuity.SummaryQuery{
		Search: r.URL.Query().Get("q"),
		Status: r.URL.Query().Get("status"),
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  limit,
	})
	if err != nil {
		writeSummaryError(w, err, "Programs could not be loaded.")
		return
	}
	// Internal projection timestamps can move solely because of a restricted
	// Matter. Use the Program's actor-visible update time on the HTTP surface so
	// summary timing cannot reveal hidden issue activity.
	for index := range page.Items {
		if page.Items[index].StateGeneratedAt == nil {
			continue
		}
		updatedAt := page.Items[index].Program.UpdatedAt
		page.Items[index].StateGeneratedAt = &updatedAt
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
	page, err := service.ListMatterSummaries(r.Context(), tenant, continuity.SummaryQuery{
		Search: r.URL.Query().Get("q"),
		Status: r.URL.Query().Get("status"),
		Cursor: r.URL.Query().Get("cursor"),
		Limit:  limit,
	})
	if err != nil {
		writeSummaryError(w, err, "Issues and changes could not be loaded.")
		return
	}
	page.Items = filterMatterSummaries(r.Context(), page.Items)
	httpx.WriteJSON(w, http.StatusOK, page)
}

func writeSummaryError(w http.ResponseWriter, err error, fallback string) {
	if strings.Contains(strings.ToLower(err.Error()), "invalid cursor") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_cursor", "The page cursor is invalid or no longer usable. Reload the first page.")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, "summary_failed", fallback)
}
