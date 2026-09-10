package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/people"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) employeeProfile(w http.ResponseWriter, r *http.Request) {
	if a.deps.People == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "people_unavailable", "Employee records are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	profile, err := a.deps.People.Profile(r.Context(), people.Scope{Viewer: actor, PersonID: r.PathValue("person_id")})
	if peopleError(w, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, profile)
}

func (a *API) employeeWork(w http.ResponseWriter, r *http.Request) {
	a.employeePage(w, r, func(query people.PageQuery) (any, error) { return a.deps.People.Work(r.Context(), query) })
}
func (a *API) employeeAssignments(w http.ResponseWriter, r *http.Request) {
	a.employeePage(w, r, func(query people.PageQuery) (any, error) { return a.deps.People.Assignments(r.Context(), query) })
}
func (a *API) employeeActivity(w http.ResponseWriter, r *http.Request) {
	a.employeePage(w, r, func(query people.PageQuery) (any, error) { return a.deps.People.Activity(r.Context(), query) })
}

func (a *API) employeePage(w http.ResponseWriter, r *http.Request, get func(people.PageQuery) (any, error)) {
	if a.deps.People == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "people_unavailable", "Employee records are unavailable.")
		return
	}
	actor, err := identity.Require(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
		return
	}
	query, err := employeePageQuery(r, actor)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The employee record query is invalid.")
		return
	}
	value, err := get(query)
	if peopleError(w, err) {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, value)
}

func employeePageQuery(r *http.Request, actor identity.Actor) (people.PageQuery, error) {
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return people.PageQuery{}, err
		}
		limit = parsed
	}
	var from, to *time.Time
	for _, target := range []struct {
		raw  string
		into **time.Time
	}{{r.URL.Query().Get("from"), &from}, {r.URL.Query().Get("to"), &to}} {
		if strings.TrimSpace(target.raw) == "" {
			continue
		}
		value, err := time.Parse(time.RFC3339, target.raw)
		if err != nil {
			return people.PageQuery{}, err
		}
		parsed := value.UTC()
		*target.into = &parsed
	}
	return people.PageQuery{Scope: people.Scope{Viewer: actor, PersonID: r.PathValue("person_id")}, Limit: limit, Cursor: r.URL.Query().Get("cursor"), From: from, To: to, Category: r.URL.Query().Get("category"), State: r.URL.Query().Get("state")}, nil
}

func peopleError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, people.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "employee_not_found", "The employee record was not found.")
	case errors.Is(err, people.ErrInvalid):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The employee record query is invalid.")
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "employee_read_failed", "Employee records could not be loaded.")
	}
	return true
}
