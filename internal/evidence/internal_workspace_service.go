package evidence

import (
	"context"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// internalWorkspaceStore is deliberately separate from capability-based
// sessions. Store methods repeat identity and recipient checks under locks.
type internalWorkspaceStore interface {
	getInternalWorkspace(context.Context, string, time.Time) (Request, ResponseWorkspaceView, error)
	saveInternalWorkspace(context.Context, string, SaveWorkspaceInput, time.Time) (ResponseWorkspaceView, error)
	submitInternalWorkspace(context.Context, string, SubmitWorkspaceInput, time.Time) (WorkspaceSubmissionResult, error)
}

// InternalWorkspaceAuthorizer resolves the current performer route for the
// exact request. It is invoked while material state is locked; PostgreSQL
// implementations receive the material transaction in ctx.
type InternalWorkspaceAuthorizer func(context.Context, identity.Actor, Request, string) (string, error)
type internalWorkspaceAuthorityKey struct{}
type internalWorkspaceAuthorityCheck struct {
	authorize InternalWorkspaceAuthorizer
	command   string
}

func (service *DistributionAccessService) WithInternalWorkspaceAuthorizer(authorize InternalWorkspaceAuthorizer) *DistributionAccessService {
	if service != nil {
		service.internalAuthorizer = authorize
	}
	return service
}

func authorizeInternalWorkspaceLocked(ctx context.Context, actor identity.Actor, request Request) (string, error) {
	check, ok := ctx.Value(internalWorkspaceAuthorityKey{}).(internalWorkspaceAuthorityCheck)
	if !ok {
		return "", nil
	} // Private store operations are not an authorization boundary.
	if check.authorize == nil {
		return "", ErrWorkspaceUnavailable
	}
	route, err := check.authorize(ctx, actor, request, check.command)
	if err != nil || strings.TrimSpace(route) == "" {
		return "", ErrWorkspaceUnavailable
	}
	return route, nil
}

func (service *DistributionAccessService) internalWorkspace(ctx context.Context, requestID string) (internalWorkspaceStore, time.Time, error) {
	if service == nil || service.store == nil || strings.TrimSpace(requestID) == "" {
		return nil, time.Time{}, ErrWorkspaceUnavailable
	}
	now := service.currentTime()
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(now) != nil || !strings.EqualFold(strings.TrimSpace(actor.Kind), "PERSON") || actor.LegalEntityID == "*" {
		return nil, time.Time{}, ErrWorkspaceUnavailable
	}
	store, ok := service.store.(internalWorkspaceStore)
	if !ok {
		return nil, time.Time{}, ErrWorkspaceUnavailable
	}
	return store, now, nil
}

func (service *DistributionAccessService) GetInternalWorkspace(ctx context.Context, requestID string) (Request, ResponseWorkspaceView, error) {
	store, now, err := service.internalWorkspace(ctx, requestID)
	if err != nil {
		return Request{}, ResponseWorkspaceView{}, err
	}
	request, view, err := store.getInternalWorkspace(ctx, strings.TrimSpace(requestID), now)
	if err != nil {
		return Request{}, ResponseWorkspaceView{}, normalizeWorkspaceError(err)
	}
	return request, cloneResponseWorkspaceView(view), nil
}

func (service *DistributionAccessService) SaveInternalWorkspace(ctx context.Context, requestID string, input SaveWorkspaceInput) (ResponseWorkspaceView, error) {
	store, now, err := service.internalWorkspace(ctx, requestID)
	if err != nil {
		return ResponseWorkspaceView{}, err
	}
	if service.internalAuthorizer == nil {
		return ResponseWorkspaceView{}, ErrWorkspaceUnavailable
	}
	ctx = context.WithValue(ctx, internalWorkspaceAuthorityKey{}, internalWorkspaceAuthorityCheck{service.internalAuthorizer, "forms.response.save"})
	view, err := store.saveInternalWorkspace(ctx, strings.TrimSpace(requestID), input, now)
	if err != nil {
		return ResponseWorkspaceView{}, normalizeWorkspaceError(err)
	}
	return cloneResponseWorkspaceView(view), nil
}

func (service *DistributionAccessService) SubmitInternalWorkspace(ctx context.Context, requestID string, input SubmitWorkspaceInput) (WorkspaceSubmissionResult, error) {
	store, now, err := service.internalWorkspace(ctx, requestID)
	if err != nil {
		return WorkspaceSubmissionResult{}, err
	}
	if service.internalAuthorizer == nil {
		return WorkspaceSubmissionResult{}, ErrWorkspaceUnavailable
	}
	ctx = context.WithValue(ctx, internalWorkspaceAuthorityKey{}, internalWorkspaceAuthorityCheck{service.internalAuthorizer, "forms.response.submit"})
	result, err := store.submitInternalWorkspace(ctx, strings.TrimSpace(requestID), input, now)
	if err != nil {
		return WorkspaceSubmissionResult{}, normalizeWorkspaceError(err)
	}
	return result, nil
}

func (store *MemoryDistributionAccessStore) getInternalWorkspace(ctx context.Context, requestID string, now time.Time) (Request, ResponseWorkspaceView, error) {
	return store.getInternalMemoryWorkspace(ctx, requestID, now)
}
func (store *MemoryDistributionAccessStore) saveInternalWorkspace(ctx context.Context, requestID string, input SaveWorkspaceInput, now time.Time) (ResponseWorkspaceView, error) {
	return store.saveInternalMemoryWorkspace(ctx, requestID, input, now)
}
func (store *MemoryDistributionAccessStore) submitInternalWorkspace(ctx context.Context, requestID string, input SubmitWorkspaceInput, now time.Time) (WorkspaceSubmissionResult, error) {
	return store.submitInternalMemoryWorkspace(ctx, requestID, input, now)
}
