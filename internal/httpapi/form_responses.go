package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
	"github.com/CloudSpaceLab/clearsight-grc/internal/formcontract"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) listCompletedFormResponses(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	requestedEntity := ""
	if actor.LegalEntityID == "*" {
		requestedEntity = r.URL.Query().Get("legal_entity_id")
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, requestedEntity)
	if !ok {
		return
	}
	query, err := completedResponseQueryFromRequest(r)
	if err != nil {
		writeCompletedResponseError(w, evidence.ErrDistributionInvalid)
		return
	}
	query.TenantID = actor.TenantID
	query.LegalEntityID = legalEntityID
	query.PrincipalID = actor.PrincipalID
	page, err := service.ListCompletedResponses(r.Context(), query)
	if err != nil {
		writeCompletedResponseError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(page.Items))
	for _, value := range page.Items {
		items = append(items, completedResponseSummaryJSON(value))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": page.NextCursor})
}

func (a *API) getCompletedFormResponse(w http.ResponseWriter, r *http.Request) {
	service, ok := a.formDistributionService(w)
	if !ok {
		return
	}
	actor, ok := distributionActor(w, r)
	if !ok {
		return
	}
	requestedEntity := ""
	if actor.LegalEntityID == "*" {
		requestedEntity = r.URL.Query().Get("legal_entity_id")
	}
	legalEntityID, ok := distributionLegalEntity(w, r, actor, requestedEntity)
	if !ok {
		return
	}
	summary, revision, err := service.GetCompletedResponse(r.Context(), actor.TenantID, legalEntityID, actor.PrincipalID, r.PathValue("revision_id"))
	if err != nil {
		writeCompletedResponseError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"response": completedResponseSummaryJSON(summary), "revision": responseRevisionJSON(revision)})
}

func completedResponseQueryFromRequest(r *http.Request) (evidence.CompletedResponseQuery, error) {
	values := r.URL.Query()
	query := evidence.CompletedResponseQuery{
		FormTemplateID: strings.TrimSpace(values.Get("form_template_id")), SubjectType: strings.TrimSpace(values.Get("subject_type")), SubjectID: strings.TrimSpace(values.Get("subject_id")),
		Sort: evidence.ResponseSort(strings.ToUpper(strings.TrimSpace(values.Get("sort")))), Cursor: strings.TrimSpace(values.Get("cursor")), Limit: 25, CurrentOnly: true,
	}
	var err error
	if raw := strings.TrimSpace(values.Get("limit")); raw != "" {
		query.Limit, err = strconv.Atoi(raw)
		if err != nil {
			return evidence.CompletedResponseQuery{}, err
		}
	}
	if raw := strings.TrimSpace(values.Get("form_template_version")); raw != "" {
		query.FormTemplateVersion, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return evidence.CompletedResponseQuery{}, err
		}
	}
	if raw := strings.TrimSpace(values.Get("current_only")); raw != "" {
		query.CurrentOnly, err = strconv.ParseBool(raw)
		if err != nil {
			return evidence.CompletedResponseQuery{}, err
		}
	}
	if query.RawMinimum, err = optionalQueryFloat(values.Get("raw_min")); err != nil {
		return evidence.CompletedResponseQuery{}, err
	}
	if query.RawMaximum, err = optionalQueryFloat(values.Get("raw_max")); err != nil {
		return evidence.CompletedResponseQuery{}, err
	}
	if query.AdverseMinimum, err = optionalQueryFloat(values.Get("adverse_min")); err != nil {
		return evidence.CompletedResponseQuery{}, err
	}
	if query.AdverseMaximum, err = optionalQueryFloat(values.Get("adverse_max")); err != nil {
		return evidence.CompletedResponseQuery{}, err
	}
	if query.CompletedFrom, err = optionalQueryTime(values.Get("completed_from")); err != nil {
		return evidence.CompletedResponseQuery{}, err
	}
	if query.CompletedUntil, err = optionalQueryTime(values.Get("completed_until")); err != nil {
		return evidence.CompletedResponseQuery{}, err
	}
	for _, value := range splitQueryValues(values["mode"]) {
		query.Modes = append(query.Modes, formcontract.ScoringMode(strings.ToUpper(value)))
	}
	for _, value := range splitQueryValues(values["band"]) {
		query.Bands = append(query.Bands, formcontract.ConcernBand(strings.ToUpper(value)))
	}
	for _, value := range splitQueryValues(values["score_state"]) {
		query.States = append(query.States, evidence.ResponseScoreState(strings.ToUpper(value)))
	}
	return query, nil
}

func optionalQueryFloat(value string) (*float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	return &parsed, err
}

func optionalQueryTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func splitQueryValues(values []string) []string {
	result := make([]string, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

func completedResponseSummaryJSON(value evidence.CompletedResponseSummary) map[string]any {
	return map[string]any{
		"id": value.ID, "distribution_id": value.DistributionID,
		"form_template_id": value.FormTemplateID, "form_template_version": value.FormTemplateVersion,
		"title": value.Title, "subject_type": value.SubjectType, "subject_id": value.SubjectID,
		"revision": value.Revision, "current": value.Current, "state": value.State,
		"score": responseScoreJSON(value.Score, false), "completed_at": value.CompletedAt,
	}
}

func responseScoreJSON(value *evidence.ResponseScoreResult, detail bool) any {
	if value == nil {
		return nil
	}
	result := map[string]any{
		"mode": value.Mode, "direction": value.Direction, "raw_score": value.RawScore,
		"adverse_score": value.AdverseScore, "band": value.Band, "coverage": value.Coverage,
		"final": value.Final, "state": value.State, "profile_version": value.ProfileVersion,
		"profile_checksum": value.ProfileChecksum, "evaluator_version": value.EvaluatorVersion,
		"failure_code": value.FailureCode, "calculated_at": value.CalculatedAt,
	}
	if detail {
		result["contribution_results"] = value.ContributionResults
		result["rule_results"] = value.RuleResults
	}
	return result
}

func writeCompletedResponseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, evidence.ErrDistributionInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "response_filters_invalid", "Check the completed-response filters and try again.")
	case errors.Is(err, evidence.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "response_not_found", "The completed response was not found in this legal entity.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "responses_unavailable", "Completed responses could not be loaded. Try again.")
	}
}
