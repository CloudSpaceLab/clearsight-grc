package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/reporting"
)

type reportingHTTPRequestScope struct {
	Scope       reporting.ReportScope
	PrincipalID string
}

func (a *API) reportingService(w http.ResponseWriter) (*reporting.Service, bool) {
	if a == nil || a.deps.Reporting == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "reporting_unavailable", "Report definitions and runs are unavailable. Try again in a moment.")
		return nil, false
	}
	return a.deps.Reporting, true
}

func (a *API) reportingCommandScope(w http.ResponseWriter, r *http.Request) (reportingHTTPRequestScope, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to continue with this report.")
		return reportingHTTPRequestScope{}, false
	}
	scope, ok := reportingActorScope(w, actor)
	return reportingHTTPRequestScope{Scope: scope, PrincipalID: strings.TrimSpace(actor.PrincipalID)}, ok
}

func (a *API) reportingReadScope(w http.ResponseWriter, r *http.Request) (reportingHTTPRequestScope, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view this report.")
		return reportingHTTPRequestScope{}, false
	}
	scope, ok := reportingActorScope(w, actor)
	return reportingHTTPRequestScope{Scope: scope, PrincipalID: strings.TrimSpace(actor.PrincipalID)}, ok
}

func reportingActorScope(w http.ResponseWriter, actor identity.Actor) (reporting.ReportScope, bool) {
	tenantID := strings.TrimSpace(actor.TenantID)
	legalEntityID := strings.TrimSpace(actor.LegalEntityID)
	principalID := strings.TrimSpace(actor.PrincipalID)
	if tenantID == "" || principalID == "" || legalEntityID == "" || legalEntityID == "*" {
		httpx.WriteError(w, http.StatusForbidden, "report_scope_unavailable", "Your verified identity does not provide one legal entity for reports. Choose an eligible legal entity and try again.")
		return reporting.ReportScope{}, false
	}
	return reporting.ReportScope{TenantID: tenantID, LegalEntityID: legalEntityID}, true
}

func (a *API) listReportFilterFields(w http.ResponseWriter, _ *http.Request) {
	fields := append([]reporting.ReportFilterFieldDefinition(nil), reporting.ReportFilterFieldVocabulary...)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"fields": fields})
}

func (a *API) listReportDefinitions(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingReadScope(w, r)
	if !ok {
		return
	}
	includeRetired := false
	if value := strings.TrimSpace(r.URL.Query().Get("include_retired")); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "report_retired_filter_invalid", "The report retired-definition filter must be true or false.")
			return
		}
		includeRetired = parsed
	}
	definitions, err := service.ListDefinitions(r.Context(), requestScope.Scope, includeRetired)
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": definitions})
}

func (a *API) getReportDefinition(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingReadScope(w, r)
	if !ok {
		return
	}
	definition, err := service.GetDefinition(r.Context(), requestScope.Scope, r.PathValue("id"))
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, definition)
}

func (a *API) getReportDefinitionHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingReadScope(w, r)
	if !ok {
		return
	}
	revisions, err := service.GetDefinitionHistory(r.Context(), requestScope.Scope, r.PathValue("id"))
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": revisions})
}

func (a *API) proposeReportDefinition(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingCommandScope(w, r)
	if !ok {
		return
	}
	var input reporting.ProposeInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeReportError(w, reporting.ErrInvalid)
		return
	}
	input.TenantID = requestScope.Scope.TenantID
	input.LegalEntityID = requestScope.Scope.LegalEntityID
	input.MakerID = requestScope.PrincipalID
	input.ActorID = requestScope.PrincipalID
	input.ReviewerID = ""
	input.AuthorizerID = ""
	input.CheckerID = ""
	if field, unavailable := unavailableReportFilterField(input.Dataset, input.Filter); unavailable {
		writeUnavailableReportFilter(w, field, input.Dataset)
		return
	}
	definition, err := service.Propose(r.Context(), input)
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, definition)
}

func (a *API) reportDefinitionAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		service, ok := a.reportingService(w)
		if !ok {
			return
		}
		requestScope, ok := a.reportingCommandScope(w, r)
		if !ok {
			return
		}
		var input reporting.DefinitionTransitionInput
		if err := httpx.DecodeJSON(w, r, &input); err != nil {
			writeReportError(w, reporting.ErrInvalid)
			return
		}
		input.Scope = requestScope.Scope
		input.TenantID = requestScope.Scope.TenantID
		input.LegalEntityID = requestScope.Scope.LegalEntityID
		input.DefinitionID = strings.TrimSpace(r.PathValue("id"))
		input.ActorID = requestScope.PrincipalID
		input.ReviewerID = ""
		input.AuthorizerID = ""
		input.CheckerID = ""

		var (
			definition reporting.ReportDefinition
			err        error
		)
		switch action {
		case "submit":
			definition, err = service.Submit(r.Context(), input)
		case "review":
			definition, err = service.Review(r.Context(), input)
		case "activate":
			definition, err = service.Activate(r.Context(), input)
		case "reject":
			definition, err = service.Reject(r.Context(), input)
		case "retire":
			definition, err = service.Retire(r.Context(), input)
		default:
			err = reporting.ErrInvalid
		}
		if err != nil {
			writeReportError(w, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, definition)
	}
}

func (a *API) listReportRuns(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingReadScope(w, r)
	if !ok {
		return
	}
	limit := 50
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "report_run_limit_invalid", "The report run limit must be a whole number from 1 through 100. Review the limit and try again.")
			return
		}
		limit = parsed
	}
	runs, err := service.ListRuns(r.Context(), requestScope.Scope, strings.TrimSpace(r.URL.Query().Get("definition_id")), limit)
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": runs})
}

func (a *API) createReportRun(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingCommandScope(w, r)
	if !ok {
		return
	}
	var input reporting.CreateRunInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeReportError(w, reporting.ErrInvalid)
		return
	}
	input.Scope = requestScope.Scope
	input.TenantID = requestScope.Scope.TenantID
	input.LegalEntityID = requestScope.Scope.LegalEntityID
	input.ActorID = requestScope.PrincipalID
	input.RequestedByRef = requestScope.PrincipalID
	run, err := service.CreateRun(r.Context(), input)
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, run)
}

func (a *API) getReportRun(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingReadScope(w, r)
	if !ok {
		return
	}
	run, err := service.GetRun(r.Context(), requestScope.Scope, r.PathValue("id"))
	if err != nil {
		writeReportError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, run)
}

func (a *API) downloadReportRun(w http.ResponseWriter, r *http.Request) {
	service, ok := a.reportingService(w)
	if !ok {
		return
	}
	requestScope, ok := a.reportingReadScope(w, r)
	if !ok {
		return
	}
	run, reader, err := service.Open(r.Context(), requestScope.Scope, r.PathValue("id"))
	if err != nil {
		writeReportError(w, err)
		return
	}
	defer reader.Close()

	extension := "csv"
	contentType := "text/csv; charset=utf-8"
	if run.Format == reporting.FormatNDJSON {
		extension = "ndjson"
		contentType = "application/x-ndjson; charset=utf-8"
	} else if run.Format == reporting.FormatXLSX {
		extension = "xlsx"
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	filename := safeReportFilename(run.DefinitionCode) + "." + extension
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("ETag", `"`+run.DataSHA256+`"`)
	if _, err := io.Copy(w, reader); err != nil && a.deps.Logger != nil {
		a.deps.Logger.Warn("report download stream interrupted", "run_id", run.ID, "error", err)
	}
}

func unavailableReportFilterField(dataset reporting.ReportDataset, expression *reporting.ReportFilterExpression) (reporting.ReportFilterField, bool) {
	if expression == nil {
		return "", false
	}
	if strings.EqualFold(strings.TrimSpace(expression.Kind), "condition") {
		if !reportFilterFieldAvailable(dataset, expression.Field) {
			return expression.Field, true
		}
	}
	for index := range expression.Children {
		if field, unavailable := unavailableReportFilterField(dataset, &expression.Children[index]); unavailable {
			return field, true
		}
	}
	return "", false
}

func reportFilterFieldAvailable(dataset reporting.ReportDataset, field reporting.ReportFilterField) bool {
	for _, definition := range reporting.ReportFilterFieldVocabulary {
		if definition.Field == field && (dataset == "" || definition.Dataset == dataset) {
			return true
		}
	}
	return false
}

func availableReportFilterFields(dataset reporting.ReportDataset) []string {
	unique := make(map[string]struct{})
	for _, definition := range reporting.ReportFilterFieldVocabulary {
		if dataset != "" && definition.Dataset != dataset {
			continue
		}
		unique[string(definition.Field)] = struct{}{}
	}
	fields := make([]string, 0, len(unique))
	for field := range unique {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields
}

func writeUnavailableReportFilter(w http.ResponseWriter, field reporting.ReportFilterField, dataset reporting.ReportDataset) {
	available := strings.Join(availableReportFilterFields(dataset), ", ")
	if available == "" {
		available = strings.Join(availableReportFilterFields(""), ", ")
	}
	httpx.WriteError(w, http.StatusBadRequest, "report_filter_field_unavailable", fmt.Sprintf(
		"The report filter field %q is not available for this report. Available fields are: %s. Choose an available field and save the report again.",
		string(field), available,
	))
}

func safeReportFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "clearsight-report"
	}
	var result strings.Builder
	for _, character := range strings.ToUpper(value) {
		if character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' || character == '_' {
			result.WriteRune(character)
		} else {
			result.WriteByte('-')
		}
	}
	return result.String()
}

func writeReportError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reporting.ErrAuthorityUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "report_authority_unavailable", "The current responsibility route for this report could not be checked. No change was made; try again when the authority service is available.")
	case errors.Is(err, reporting.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "report_not_found", "This report definition or run is not available in your legal entity.")
	case errors.Is(err, reporting.ErrReportExpired):
		httpx.WriteError(w, http.StatusGone, "report_run_expired", "This report file expired and is no longer available. Run the report again to create a new file.")
	case errors.Is(err, reporting.ErrReportBoundStopped):
		httpx.WriteError(w, http.StatusConflict, "report_run_bound_stopped", "This report stopped at its 10,000-row limit and produced no file. Narrow the report filter and run it again.")
	case errors.Is(err, reporting.ErrReportNotReady):
		httpx.WriteError(w, http.StatusConflict, "report_run_not_ready", "This report file is not ready to download. Check the run status and try again when it is ready.")
	case errors.Is(err, reporting.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "report_version_conflict", "This report changed since you opened it. Review the current definition or run and try again.")
	case errors.Is(err, reporting.ErrClosureBlocked):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "report_governance_blocked", "This report action cannot proceed because a required maker, reviewer, authorizer, approval or effective-date step is missing. Check the current report responsibility and try again.")
	case errors.Is(err, reporting.ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "report_request_invalid", "The report request is not valid. Check the dataset, scope, format, filter and expected version, then save it again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "report_request_failed", "The report request could not be completed. No change was made; try again.")
	}
}
