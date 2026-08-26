package continuity

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type trustedSystemScopeKey struct{}
type trustedSystemScope struct {
	tenant, legalEntity string
	global              bool
}

// WithTrustedSystemScope marks work performed by a known in-process worker,
// replay, migration, or internal maintenance caller. The private context key
// prevents request data from manufacturing this scope.
func WithTrustedSystemScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, trustedSystemScopeKey{}, trustedSystemScope{global: true})
}

// WithTrustedSystemEntityScope grants an in-process caller access to exactly
// one tenant and legal entity. It is intended for entity-specific installers.
func WithTrustedSystemEntityScope(ctx context.Context, tenant, legalEntity string) context.Context {
	return context.WithValue(ctx, trustedSystemScopeKey{}, trustedSystemScope{tenant: strings.TrimSpace(tenant), legalEntity: strings.TrimSpace(legalEntity)})
}

func hasTrustedSystemScope(ctx context.Context) bool {
	_, ok := ctx.Value(trustedSystemScopeKey{}).(trustedSystemScope)
	return ok
}

func visibleToActorLegalEntity(ctx context.Context, tenant, legalEntityID string) bool {
	if scope, ok := ctx.Value(trustedSystemScopeKey{}).(trustedSystemScope); ok {
		return scope.global || (scope.tenant == strings.TrimSpace(tenant) && scope.legalEntity != "" && scope.legalEntity == strings.TrimSpace(legalEntityID))
	}
	actor, ok := identity.FromContext(ctx)
	if !ok {
		return false
	}
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.TenantID) != strings.TrimSpace(tenant) {
		return false
	}
	actorEntity := strings.TrimSpace(actor.LegalEntityID)
	if actorEntity == "" {
		return false
	}
	recordEntity := strings.TrimSpace(legalEntityID)
	if recordEntity == "" {
		return false
	}
	return actorEntity == "*" || actorEntity == recordEntity
}

func withCanonicalLegalEntity(ctx context.Context, tenant, canonical string) context.Context {
	if scope, ok := ctx.Value(trustedSystemScopeKey{}).(trustedSystemScope); ok {
		if scope.global {
			return ctx
		}
		return WithTrustedSystemEntityScope(ctx, tenant, canonical)
	}
	actor, ok := identity.FromContext(ctx)
	if !ok || strings.TrimSpace(actor.LegalEntityID) == "*" {
		return ctx
	}
	actor.LegalEntityID = canonical
	return identity.WithActor(ctx, actor)
}

func actorLegalEntity(ctx context.Context, tenant, requested string) (string, bool) {
	if scope, ok := ctx.Value(trustedSystemScopeKey{}).(trustedSystemScope); ok {
		if !scope.global {
			if scope.tenant != strings.TrimSpace(tenant) || scope.legalEntity == "" {
				return "", false
			}
			return scope.legalEntity, true
		}
		requested = strings.TrimSpace(requested)
		return requested, true
	}
	actor, ok := identity.FromContext(ctx)
	if !ok || strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" {
		return "", false
	}
	if strings.TrimSpace(actor.TenantID) != strings.TrimSpace(tenant) {
		return "", false
	}
	if actor.LegalEntityID == "*" {
		requested = strings.TrimSpace(requested)
		return requested, requested != "" && requested != "*"
	}
	return strings.TrimSpace(actor.LegalEntityID), true
}

func postgresActorScope(ctx context.Context) (enforce bool, tenant, legalEntity string) {
	if scope, ok := ctx.Value(trustedSystemScopeKey{}).(trustedSystemScope); ok {
		if scope.global {
			return false, "", ""
		}
		return true, scope.tenant, scope.legalEntity
	}
	actor, ok := identity.FromContext(ctx)
	if !ok || strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" {
		return true, "", ""
	}
	return true, strings.TrimSpace(actor.TenantID), strings.TrimSpace(actor.LegalEntityID)
}
