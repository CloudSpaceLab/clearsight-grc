package identity

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrMissingIdentity = errors.New("verified identity is required")
	ErrInvalidIdentity = errors.New("identity could not be verified")
	ErrExpiredIdentity = errors.New("identity has expired")
)

const (
	PermissionConfigRead              = "CONFIG_READ"
	PermissionConfigWrite             = "CONFIG_WRITE"
	PermissionPlatformOperationsRead  = "PLATFORM_OPERATIONS_READ"
	PermissionPlatformOperationsWrite = "PLATFORM_OPERATIONS_WRITE"
	PermissionPlatformJobsRead        = "PLATFORM_JOBS_READ"
)

type Actor struct {
	TenantID             string    `json:"tenant_id"`
	PrincipalID          string    `json:"principal_id"`
	LegalEntityID        string    `json:"legal_entity_id"`
	Kind                 string    `json:"kind"`
	RoleCodes            []string  `json:"role_codes,omitempty"`
	PermissionCodes      []string  `json:"permission_codes,omitempty"`
	AuthenticationMethod string    `json:"authentication_method"`
	AssuranceLevel       string    `json:"assurance_level"`
	SessionID            string    `json:"session_id"`
	IssuedAt             time.Time `json:"issued_at"`
	ExpiresAt            time.Time `json:"expires_at"`
}

func (a Actor) Valid(now time.Time) error {
	if strings.TrimSpace(a.TenantID) == "" || strings.TrimSpace(a.PrincipalID) == "" || strings.TrimSpace(a.LegalEntityID) == "" {
		return ErrInvalidIdentity
	}
	if a.ExpiresAt.IsZero() || !now.Before(a.ExpiresAt) {
		return ErrExpiredIdentity
	}
	if !a.IssuedAt.IsZero() && a.IssuedAt.After(now.Add(time.Minute)) {
		return ErrInvalidIdentity
	}
	if len(a.RoleCodes) > 32 || len(a.PermissionCodes) > 64 {
		return ErrInvalidIdentity
	}
	for _, role := range a.RoleCodes {
		if len(strings.TrimSpace(role)) > 80 {
			return ErrInvalidIdentity
		}
	}
	for _, permission := range a.PermissionCodes {
		if len(strings.TrimSpace(permission)) > 100 {
			return ErrInvalidIdentity
		}
	}
	return nil
}

func NormalizeRoleCodes(values []string) []string {
	return normalizeCodes(values, 32)
}

func NormalizePermissionCodes(values []string) []string {
	return normalizeCodes(values, 64)
}

func HasPermission(actor Actor, permission string) bool {
	permission = normalizeCode(permission)
	if permission == "" {
		return true
	}
	for _, candidate := range actor.PermissionCodes {
		if normalizeCode(candidate) == permission {
			return true
		}
	}
	return false
}

func normalizeCodes(values []string, limit int) []string {
	result := make([]string, 0, min(len(values), limit))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = normalizeCode(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
		if len(result) == limit {
			break
		}
	}
	return result
}

func normalizeCode(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "_", " ", "_").Replace(value)
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return strings.Trim(value, "_")
}

type contextKey struct{}

func WithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, contextKey{}, actor)
}

func FromContext(ctx context.Context) (Actor, bool) {
	actor, ok := ctx.Value(contextKey{}).(Actor)
	return actor, ok
}

func Require(ctx context.Context) (Actor, error) {
	actor, ok := FromContext(ctx)
	if !ok {
		return Actor{}, ErrMissingIdentity
	}
	return actor, nil
}
