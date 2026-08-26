package identity

import (
	"context"
	"errors"
	"fmt"
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
	PermissionVendorRead              = "VENDOR_READ"
)

type DepartmentGrant struct {
	Path            []string `json:"path"`
	RoleCodes       []string `json:"role_codes,omitempty"`
	PermissionCodes []string `json:"permission_codes,omitempty"`
}

type Actor struct {
	TenantID             string            `json:"tenant_id"`
	PrincipalID          string            `json:"principal_id"`
	LegalEntityID        string            `json:"legal_entity_id"`
	Kind                 string            `json:"kind"`
	RoleCodes            []string          `json:"role_codes,omitempty"`
	PermissionCodes      []string          `json:"permission_codes,omitempty"`
	DepartmentGrants     []DepartmentGrant `json:"department_grants,omitempty"`
	AuthenticationMethod string            `json:"authentication_method"`
	AssuranceLevel       string            `json:"assurance_level"`
	SessionID            string            `json:"session_id"`
	IssuedAt             time.Time         `json:"issued_at"`
	ExpiresAt            time.Time         `json:"expires_at"`
}

func (a Actor) Valid(now time.Time) error {
	if !validIdentifier(a.TenantID, 128) || !validIdentifier(a.PrincipalID, 128) || !validIdentifier(a.LegalEntityID, 128) {
		return ErrInvalidIdentity
	}
	if len(strings.TrimSpace(a.Kind)) > 32 || len(strings.TrimSpace(a.AuthenticationMethod)) > 64 || len(strings.TrimSpace(a.AssuranceLevel)) > 512 || len(strings.TrimSpace(a.SessionID)) > 128 {
		return ErrInvalidIdentity
	}
	if a.ExpiresAt.IsZero() || !now.Before(a.ExpiresAt) {
		return ErrExpiredIdentity
	}
	if !a.IssuedAt.IsZero() && a.IssuedAt.After(now.Add(time.Minute)) {
		return ErrInvalidIdentity
	}
	if len(a.RoleCodes) > 32 || len(a.PermissionCodes) > 64 || len(a.DepartmentGrants) > 32 {
		return ErrInvalidIdentity
	}
	if !validCodes(a.RoleCodes, 80) || !validCodes(a.PermissionCodes, 100) {
		return ErrInvalidIdentity
	}
	for _, grant := range a.DepartmentGrants {
		if _, err := NormalizeDepartmentPath(grant.Path); err != nil || len(grant.RoleCodes) > 32 || len(grant.PermissionCodes) > 64 {
			return ErrInvalidIdentity
		}
		if !validCodes(grant.RoleCodes, 80) || !validCodes(grant.PermissionCodes, 100) {
			return ErrInvalidIdentity
		}
	}
	return nil
}

func validIdentifier(value string, maxLength int) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxLength {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func validCodes(values []string, maxLength int) bool {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > maxLength {
			return false
		}
	}
	return true
}

func NormalizeRoleCodes(values []string) []string {
	return normalizeCodes(values, 32)
}

func NormalizePermissionCodes(values []string) []string {
	return normalizeCodes(values, 64)
}

func NormalizeDepartmentPath(values []string) ([]string, error) {
	if len(values) > 12 {
		return nil, fmt.Errorf("department path supports at most 12 levels")
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if value == "" || len(value) > 80 {
			return nil, fmt.Errorf("department path contains an invalid segment")
		}
		result = append(result, value)
	}
	return result, nil
}

func HasPermission(actor Actor, permission string) bool {
	return hasPermission(actor.PermissionCodes, permission)
}

// HasDepartmentPermission checks only an exact department scope. Parent/child
// inheritance must be represented by an explicit governed rule rather than
// inferred from the path hierarchy.
func HasDepartmentPermission(actor Actor, departmentPath []string, permission string) bool {
	path, err := NormalizeDepartmentPath(departmentPath)
	if err != nil || len(path) == 0 {
		return false
	}
	for _, grant := range actor.DepartmentGrants {
		candidate, err := NormalizeDepartmentPath(grant.Path)
		if err == nil && equalStrings(candidate, path) && hasPermission(grant.PermissionCodes, permission) {
			return true
		}
	}
	return false
}

func hasPermission(values []string, permission string) bool {
	permission = normalizeCode(permission)
	if permission == "" {
		return true
	}
	for _, candidate := range values {
		if normalizeCode(candidate) == permission {
			return true
		}
	}
	return false
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
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
