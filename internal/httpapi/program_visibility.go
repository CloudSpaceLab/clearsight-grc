package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/httpx"
)

func (a *API) programForActor(ctx context.Context, value continuity.ProgramAggregate, at *time.Time) (continuity.ProgramAggregate, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return continuity.ProgramAggregate{}, err
	}
	if value.Program.TenantID != actor.TenantID {
		return continuity.ProgramAggregate{}, continuity.ErrNotFound
	}
	if a.deps.Continuity == nil {
		return continuity.ProgramAggregate{}, errors.New("continuity service is unavailable")
	}
	return a.deps.Continuity.ProgramForPrincipal(ctx, value, actor.PrincipalID, at)
}

func (a *API) programsForActor(ctx context.Context, values []continuity.ProgramAggregate, at *time.Time) ([]continuity.ProgramAggregate, error) {
	actor, err := identity.Require(ctx)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return []continuity.ProgramAggregate{}, nil
	}
	for _, value := range values {
		if value.Program.TenantID != actor.TenantID {
			return nil, continuity.ErrNotFound
		}
	}
	if a.deps.Continuity == nil {
		return nil, errors.New("continuity service is unavailable")
	}
	return a.deps.Continuity.ProgramsForPrincipal(ctx, values, actor.PrincipalID, at)
}

func (a *API) writeProgramResult(w http.ResponseWriter, r *http.Request, value continuity.ProgramAggregate, err error, success int, at *time.Time) {
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	value, err = a.programForActor(r.Context(), value, at)
	if err != nil {
		if errors.Is(err, identity.ErrIdentityRequired) {
			httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
			return
		}
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, success, value)
}
