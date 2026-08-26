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
	actor, ok := identity.FromContext(ctx)
	if !ok {
		return continuity.ProgramAggregate{}, errors.New("verified identity is required")
	}
	if a.deps.Continuity == nil {
		return continuity.ProgramAggregate{}, errors.New("continuity service is unavailable")
	}
	belongs, err := a.programBelongsToActorTenant(ctx, actor.TenantID, value.Program.ID)
	if err != nil {
		return continuity.ProgramAggregate{}, err
	}
	if !belongs {
		return continuity.ProgramAggregate{}, continuity.ErrNotFound
	}
	return a.deps.Continuity.ProgramForPrincipal(ctx, value, actor.PrincipalID, at)
}

func (a *API) programsForActor(ctx context.Context, values []continuity.ProgramAggregate, at *time.Time) ([]continuity.ProgramAggregate, error) {
	actor, ok := identity.FromContext(ctx)
	if !ok {
		return nil, errors.New("verified identity is required")
	}
	if len(values) == 0 {
		return []continuity.ProgramAggregate{}, nil
	}
	if a.deps.Continuity == nil {
		return nil, errors.New("continuity service is unavailable")
	}
	canonicalTenant := values[0].Program.TenantID
	for _, value := range values[1:] {
		if value.Program.TenantID != canonicalTenant {
			return []continuity.ProgramAggregate{}, nil
		}
	}
	belongs, err := a.programBelongsToActorTenant(ctx, actor.TenantID, values[0].Program.ID)
	if err != nil {
		return nil, err
	}
	if !belongs {
		return []continuity.ProgramAggregate{}, nil
	}
	return a.deps.Continuity.ProgramsForPrincipal(ctx, values, actor.PrincipalID, at)
}

func (a *API) programBelongsToActorTenant(ctx context.Context, actorTenantID, programID string) (bool, error) {
	_, err := a.deps.Continuity.GetProgram(ctx, actorTenantID, programID)
	if errors.Is(err, continuity.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *API) writeProgramResult(w http.ResponseWriter, r *http.Request, value continuity.ProgramAggregate, err error, success int, at *time.Time) {
	if err != nil {
		writeContinuityError(w, err)
		return
	}
	value, err = a.programForActor(r.Context(), value, at)
	if err != nil {
		if _, ok := identity.FromContext(r.Context()); !ok {
			httpx.WriteError(w, http.StatusUnauthorized, "identity_required", "A verified sign-in is required.")
			return
		}
		writeContinuityError(w, err)
		return
	}
	httpx.WriteJSON(w, success, value)
}
