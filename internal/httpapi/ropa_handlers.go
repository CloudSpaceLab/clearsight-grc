package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
	"github.com/CloudSpaceLab/clearsight-grc/internal/ropa"
)

func (a *API) ropaService(w http.ResponseWriter) (*ropa.Service, bool) {
	if a == nil || a.deps.Ropa == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "ropa_unavailable", "The processing activity register is unavailable. Try again in a moment.")
		return nil, false
	}
	return a.deps.Ropa, true
}

type ropaRequestScope struct {
	ropa.ActivityScope
	PrincipalID string
}

// ropaReadScope deliberately requires the tenant query parameter and obtains
// the legal entity and principal only from the verified request identity.
func (a *API) ropaReadScope(w http.ResponseWriter, r *http.Request) (ropaRequestScope, bool) {
	tenantID, ok := requiredQuery(w, r, "tenant_id")
	if !ok {
		return ropaRequestScope{}, false
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to view the processing activity register.")
		return ropaRequestScope{}, false
	}
	if strings.TrimSpace(actor.TenantID) != tenantID {
		// Match the middleware's information-disclosure boundary: do not reveal
		// whether a requested tenant or legal entity exists.
		httpx.WriteError(w, http.StatusNotFound, "ropa_scope_not_found", "The processing activity register is not available for this organization. Review your organization scope and try again.")
		return ropaRequestScope{}, false
	}
	return a.ropaActorScope(w, actor)
}

// ropaCommandScope never reads scope or actor fields from the command body.
func (a *API) ropaCommandScope(w http.ResponseWriter, r *http.Request) (ropaRequestScope, bool) {
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "sign_in_required", "Sign in is required to change a processing activity.")
		return ropaRequestScope{}, false
	}
	return a.ropaActorScope(w, actor)
}

func (a *API) ropaActorScope(w http.ResponseWriter, actor identity.Actor) (ropaRequestScope, bool) {
	tenantID := strings.TrimSpace(actor.TenantID)
	legalEntityID := strings.TrimSpace(actor.LegalEntityID)
	principalID := strings.TrimSpace(actor.PrincipalID)
	if tenantID == "" || legalEntityID == "" || legalEntityID == "*" || principalID == "" {
		httpx.WriteError(w, http.StatusForbidden, "ropa_scope_unavailable", "Your verified identity does not provide one legal entity for this processing activity register. Choose an eligible legal entity and try again.")
		return ropaRequestScope{}, false
	}
	return ropaRequestScope{
		ActivityScope: ropa.ActivityScope{TenantID: tenantID, LegalEntityID: legalEntityID},
		PrincipalID:   principalID,
	}, true
}

type ropaActivityResponse struct {
	StateLabel      string                  `json:"state_label"`
	Activity        ropa.ProcessingActivity `json:"activity"`
	ClosureBlockers []string                `json:"closure_blockers,omitempty"`
}

func (a *API) getRopaDashboard(w http.ResponseWriter, r *http.Request) {
	service, ok := a.ropaService(w)
	if !ok {
		return
	}
	scope, ok := a.ropaReadScope(w, r)
	if !ok {
		return
	}
	summary, err := service.RegisterSummary(r.Context(), scope.TenantID, scope.LegalEntityID)
	if err != nil {
		writeROPAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (a *API) listRopaProcessingActivities(w http.ResponseWriter, r *http.Request) {
	service, ok := a.ropaService(w)
	if !ok {
		return
	}
	scope, ok := a.ropaReadScope(w, r)
	if !ok {
		return
	}
	limit, ok := ropaQueryInt(w, r, "limit", 0, 0)
	if !ok {
		return
	}
	includeRetired, err := strconv.ParseBool(ropaQueryValue(r, "include_retired", "false"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "ropa_filter_invalid", "The processing activity retired-record filter must be true or false.")
		return
	}
	ownerID := ropaQueryValue(r, "owner_id", "")
	if ownerID == "" {
		ownerID = ropaQueryValue(r, "owner_principal_id", "")
	}
	page, err := service.ListActivities(r.Context(), scope.ActivityScope, ropa.ListActivitiesFilter{
		Status:           ropa.Status(ropaQueryValue(r, "status", "")),
		LawfulBasis:      ropaQueryValue(r, "lawful_basis", ""),
		OwnerPrincipalID: ownerID,
		Search:           ropaQueryValue(r, "search", ""),
		IncludeRetired:   includeRetired,
		Cursor:           ropaQueryValue(r, "cursor", ""),
		Limit:            limit,
	})
	if err != nil {
		writeROPAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, page)
}

func (a *API) getRopaProcessingActivity(w http.ResponseWriter, r *http.Request) {
	service, ok := a.ropaService(w)
	if !ok {
		return
	}
	scope, ok := a.ropaReadScope(w, r)
	if !ok {
		return
	}
	activity, err := service.GetActivity(r.Context(), scope.ActivityScope, r.PathValue("id"))
	if err != nil {
		writeROPAError(w, err)
		return
	}
	blockers, err := service.ClosureBlockers(r.Context(), scope.ActivityScope, activity.ID)
	if err != nil {
		writeROPAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ropaActivityResponse{
		StateLabel:      activity.Status.String(),
		Activity:        activity,
		ClosureBlockers: blockers,
	})
}

func (a *API) getRopaProcessingActivityHistory(w http.ResponseWriter, r *http.Request) {
	service, ok := a.ropaService(w)
	if !ok {
		return
	}
	scope, ok := a.ropaReadScope(w, r)
	if !ok {
		return
	}
	afterVersion, ok := ropaQueryInt64(w, r, "after_version", 0, 0)
	if !ok {
		return
	}
	limit, ok := ropaQueryInt(w, r, "limit", 0, 0)
	if !ok {
		return
	}

	var (
		events  []ropa.Event
		hasMore bool
		err     error
	)
	if reader := a.deps.RopaEventsReader; reader != nil {
		// Verify the exact current record through the service before using a
		// separately injected read repository. This keeps a stale or wrong-scope
		// reader from turning a missing activity into a history disclosure.
		if _, err = service.GetActivity(r.Context(), scope.ActivityScope, r.PathValue("id")); err == nil {
			events, hasMore, err = reader.ActivityEvents(r.Context(), scope.ActivityScope, r.PathValue("id"), afterVersion, limit)
		}
	} else {
		events, hasMore, err = service.ActivityEvents(r.Context(), scope.ActivityScope, r.PathValue("id"), afterVersion, limit)
	}
	if err != nil {
		writeROPAError(w, err)
		return
	}
	if err := validateRopaEventScope(scope.ActivityScope, r.PathValue("id"), events); err != nil {
		writeROPAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"events":   events,
		"has_more": hasMore,
	})
}

func validateRopaEventScope(scope ropa.ActivityScope, activityID string, events []ropa.Event) error {
	for _, event := range events {
		if event.AggregateType != "PROCESSING_ACTIVITY" || event.TenantID != scope.TenantID || event.LegalEntityID != scope.LegalEntityID || event.AggregateID != activityID {
			return ropa.ErrScopeMismatch
		}
	}
	return nil
}

func (a *API) createRopaProcessingActivity(w http.ResponseWriter, r *http.Request) {
	service, ok := a.ropaService(w)
	if !ok {
		return
	}
	scope, ok := a.ropaCommandScope(w, r)
	if !ok {
		return
	}
	var input ropa.CreateActivityInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeROPAError(w, ropa.ErrInvalid)
		return
	}
	input.TenantID = scope.TenantID
	input.LegalEntityID = scope.LegalEntityID
	input.ActorID = scope.PrincipalID
	activity, err := service.CreateActivity(r.Context(), input)
	if err != nil {
		writeROPAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, a.ropaCommandActivityResponse(activity))
}

func (a *API) updateRopaProcessingActivity(w http.ResponseWriter, r *http.Request) {
	service, ok := a.ropaService(w)
	if !ok {
		return
	}
	scope, ok := a.ropaCommandScope(w, r)
	if !ok {
		return
	}
	var input ropa.UpdateActivityInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeROPAError(w, ropa.ErrInvalid)
		return
	}
	input.TenantID = scope.TenantID
	input.LegalEntityID = scope.LegalEntityID
	input.ActivityID = r.PathValue("id")
	input.ID = ""
	input.ActorID = scope.PrincipalID
	activity, err := service.UpdateActivity(r.Context(), input)
	if err != nil {
		writeROPAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.ropaCommandActivityResponse(activity))
}

func (a *API) transitionRopaProcessingActivity(w http.ResponseWriter, r *http.Request) {
	service, ok := a.ropaService(w)
	if !ok {
		return
	}
	scope, ok := a.ropaCommandScope(w, r)
	if !ok {
		return
	}
	var input ropa.TransitionActivityInput
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		writeROPAError(w, ropa.ErrInvalid)
		return
	}
	input.TenantID = scope.TenantID
	input.LegalEntityID = scope.LegalEntityID
	input.ActivityID = r.PathValue("id")
	input.ID = ""
	input.ActorID = scope.PrincipalID
	activity, err := service.TransitionActivity(r.Context(), input)
	if err != nil {
		writeROPAError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, a.ropaCommandActivityResponse(activity))
}

func (a *API) ropaCommandActivityResponse(activity ropa.ProcessingActivity) ropaActivityResponse {
	// The command has already committed at this point. Do not turn a
	// successful material command into a failure because a derived closure
	// read failed; the returned snapshot is sufficient for the bounded local
	// blocker calculation.
	return ropaActivityResponse{
		StateLabel:      activity.Status.String(),
		Activity:        activity,
		ClosureBlockers: ropaClosureBlockers(activity),
	}
}

func ropaClosureBlockers(activity ropa.ProcessingActivity) []string {
	if activity.Status == ropa.StatusClosed {
		return nil
	}
	blockers := make([]string, 0, 4)
	if strings.TrimSpace(activity.LawfulBasis) == "" {
		blockers = append(blockers, "lawful basis")
	}
	if strings.TrimSpace(activity.OwnerPrincipalID) == "" {
		blockers = append(blockers, "named owner")
	}
	if strings.TrimSpace(activity.DataSubjectCategories) == "" {
		blockers = append(blockers, "data subject category")
	}
	if !ropaHasCompletedReview(activity.Reviews) {
		blockers = append(blockers, "completed review")
	}
	return blockers
}

func ropaHasCompletedReview(reviews []ropa.Review) bool {
	for _, review := range reviews {
		if review.CompletedAt != nil && (review.Outcome == "CONFIRMED" || review.Outcome == "REVISED") {
			return true
		}
	}
	return false
}

func writeROPAError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ropa.ErrScopeMismatch):
		// A scope mismatch is deliberately indistinguishable from a missing
		// record so a caller cannot use this boundary to discover another
		// legal entity's activity.
		httpx.WriteError(w, http.StatusNotFound, "ropa_activity_not_found", "This processing activity is not available in your legal entity.")
	case errors.Is(err, ropa.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "ropa_activity_not_found", "This processing activity was not found in your legal entity.")
	case errors.Is(err, ropa.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "ropa_activity_version_conflict", "This processing activity changed since you opened it. Review the current record and try again.")
	case errors.Is(err, ropa.ErrDuplicate):
		httpx.WriteError(w, http.StatusConflict, "ropa_activity_duplicate", "A processing activity with this code already exists in this legal entity. Choose a different code.")
	case errors.Is(err, ropa.ErrClosureBlocked):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "ropa_activity_closure_blocked", "This processing activity cannot be closed until its lawful basis, named owner, data subject category and completed review are recorded. Add the missing facts and try again.")
	case errors.Is(err, ropa.ErrInvalid):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "ropa_activity_invalid", "This processing activity is not valid. Check the code, name and controller, then save it again.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "ropa_unavailable", "The processing activity register could not complete this request. No change was made; try again.")
	}
}

func ropaQueryValue(r *http.Request, name, fallback string) string {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback
	}
	return value
}

func ropaQueryInt(w http.ResponseWriter, r *http.Request, name string, fallback, minimum int) (int, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum {
		httpx.WriteError(w, http.StatusBadRequest, "ropa_filter_invalid", "The processing activity "+name+" filter must be a valid number. Review it and try again.")
		return 0, false
	}
	return parsed, true
}

func ropaQueryInt64(w http.ResponseWriter, r *http.Request, name string, fallback, minimum int64) (int64, bool) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback, true
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < minimum {
		httpx.WriteError(w, http.StatusBadRequest, "ropa_history_cursor_invalid", "The processing activity history cursor must be a valid version number. Review it and try again.")
		return 0, false
	}
	return parsed, true
}
