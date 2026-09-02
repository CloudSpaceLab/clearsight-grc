package runtimecontext

import (
	"context"
	"errors"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var ErrContextUnavailable = errors.New("runtime context is unavailable")

type NamedRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ActorRecord struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Kind             string                     `json:"kind,omitempty"`
	AssuranceLevel   string                     `json:"assurance_level,omitempty"`
	Authentication   string                     `json:"authentication,omitempty"`
	SessionID        string                     `json:"session_id,omitempty"`
	RoleCodes        []string                   `json:"role_codes,omitempty"`
	DepartmentGrants []identity.DepartmentGrant `json:"department_grants,omitempty"`
}

type Context struct {
	Tenant      NamedRecord `json:"tenant"`
	LegalEntity NamedRecord `json:"legal_entity"`
	Actor       ActorRecord `json:"actor"`
}

type Resolver interface {
	Resolve(context.Context, identity.Actor) (Context, error)
}

// IdentityResolver is the truthful fallback for runtimes without a durable
// directory. Optional labels must come from explicit fixtures/configuration;
// absent labels fall back to the verified identifier rather than invented data.
type IdentityResolver struct {
	TenantNames      map[string]string
	LegalEntityNames map[string]string
	PrincipalNames   map[string]string
}

func (r IdentityResolver) Resolve(_ context.Context, actor identity.Actor) (Context, error) {
	if strings.TrimSpace(actor.TenantID) == "" || strings.TrimSpace(actor.LegalEntityID) == "" || strings.TrimSpace(actor.PrincipalID) == "" {
		return Context{}, ErrContextUnavailable
	}
	return contextFromActor(
		actor,
		lookupName(r.TenantNames, actor.TenantID),
		lookupName(r.LegalEntityNames, actor.LegalEntityID),
		lookupName(r.PrincipalNames, actor.PrincipalID),
	), nil
}

func contextFromActor(actor identity.Actor, tenantName, legalEntityName, principalName string) Context {
	return Context{
		Tenant:      NamedRecord{ID: actor.TenantID, Name: fallbackName(tenantName, actor.TenantID)},
		LegalEntity: NamedRecord{ID: actor.LegalEntityID, Name: fallbackName(legalEntityName, actor.LegalEntityID)},
		Actor: ActorRecord{
			ID: actor.PrincipalID, Name: fallbackName(principalName, actor.PrincipalID), Kind: actor.Kind,
			AssuranceLevel: actor.AssuranceLevel, Authentication: actor.AuthenticationMethod, SessionID: actor.SessionID,
			RoleCodes: append([]string(nil), actor.RoleCodes...), DepartmentGrants: cloneDepartmentGrants(actor.DepartmentGrants),
		},
	}
}

func lookupName(values map[string]string, id string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[id])
}

func fallbackName(name, id string) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
	}
	return strings.TrimSpace(id)
}

func cloneDepartmentGrants(values []identity.DepartmentGrant) []identity.DepartmentGrant {
	if len(values) == 0 {
		return nil
	}
	out := make([]identity.DepartmentGrant, len(values))
	for i, value := range values {
		out[i] = value
		out[i].Path = append([]string(nil), value.Path...)
		out[i].RoleCodes = append([]string(nil), value.RoleCodes...)
		out[i].PermissionCodes = append([]string(nil), value.PermissionCodes...)
	}
	return out
}
