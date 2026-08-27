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
	assignedToMe, parseOK := summaryBoolQuery(w, r, "assigned_to_me")
	if !parseOK {
		return
	}
	page, err := service.ListProgramSummaries(r.Context(), tenant, continuity.SummaryQuery{
		Search: r.URL.Query().Get("q"), Status: r.URL.Query().Get("status"),
		OverallState: r.URL.Query().Get("overall_state"), Jurisdiction: r.URL.Query().Get("jurisdiction"),
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
	httpx.WriteJSON(w, http.StatusOK, page)
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
