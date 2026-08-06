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

type Actor struct {
	TenantID             string    `json:"tenant_id"`
	PrincipalID          string    `json:"principal_id"`
	LegalEntityID        string    `json:"legal_entity_id"`
	Kind                 string    `json:"kind"`
	RoleCodes            []string  `json:"role_codes,omitempty"`
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
	if len(a.RoleCodes) > 32 {
		return ErrInvalidIdentity
	}
	for _, role := range a.RoleCodes {
		if len(strings.TrimSpace(role)) > 80 {
			return ErrInvalidIdentity
		}
	}
	return nil
}

func NormalizeRoleCodes(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		value = strings.NewReplacer("-", "_", " ", "_").Replace(value)
		for strings.Contains(value, "__") {
			value = strings.ReplaceAll(value, "__", "_")
		}
		value = strings.Trim(value, "_")
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
		if len(result) == 32 {
			break
		}
	}
	return result
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
