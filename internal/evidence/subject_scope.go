package evidence

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrSubjectUnsupported   = errors.New("evidence request subject type is not supported for exact scope resolution")
	ErrSubjectScopeMismatch = errors.New("evidence request subject is outside the legal entity scope")
	ErrSubjectAccessDenied  = errors.New("evidence request creator cannot access the subject")
)

// SubjectScope is the canonical tenant/entity identity resolved from an
// authoritative subject row. Callers must not manufacture it from request
// fields or browser state.
type SubjectScope struct {
	TenantID      string
	LegalEntityID string
	SubjectType   string
	SubjectID     string
}

type SubjectScopeResolver interface {
	ResolveSubjectScope(context.Context, string, string, string) (SubjectScope, error)
}

func resolveCreateSubjectScope(ctx context.Context, repo Repository, input CreateRequestInput) (SubjectScope, error) {
	resolver, ok := repo.(SubjectScopeResolver)
	if !ok {
		// Non-production repositories created before entity-scoped requests do
		// not have authoritative subject tables. They may continue to serve old
		// fixtures, but cannot accept an entity-scoped creation command.
		if strings.TrimSpace(input.LegalEntityID) != "" {
			return SubjectScope{}, ErrSubjectUnsupported
		}
		return SubjectScope{TenantID: input.TenantID, SubjectType: input.SubjectType, SubjectID: input.SubjectID}, nil
	}
	scope, err := resolver.ResolveSubjectScope(ctx, input.TenantID, input.SubjectType, input.SubjectID)
	if err != nil {
		return SubjectScope{}, err
	}
	if strings.TrimSpace(scope.TenantID) == "" || strings.TrimSpace(scope.LegalEntityID) == "" ||
		scope.TenantID != input.TenantID || !strings.EqualFold(scope.SubjectType, input.SubjectType) || scope.SubjectID != input.SubjectID {
		return SubjectScope{}, ErrSubjectScopeMismatch
	}
	expected := strings.TrimSpace(input.LegalEntityID)
	if expected == "" || expected != scope.LegalEntityID {
		return SubjectScope{}, ErrSubjectScopeMismatch
	}
	return scope, nil
}

func validateCurrentRequestScope(ctx context.Context, repo Repository, request Request, expectedLegalEntityID string) error {
	expectedLegalEntityID = strings.TrimSpace(expectedLegalEntityID)
	resolver, exact := repo.(SubjectScopeResolver)
	if !exact {
		if expectedLegalEntityID != "" && request.LegalEntityID != expectedLegalEntityID {
			return ErrSubjectScopeMismatch
		}
		return nil
	}
	if expectedLegalEntityID == "" || request.LegalEntityID == "" || request.LegalEntityID != expectedLegalEntityID {
		return ErrSubjectScopeMismatch
	}
	scope, err := resolver.ResolveSubjectScope(ctx, request.TenantID, request.SubjectType, request.SubjectID)
	if err != nil {
		return err
	}
	if scope.LegalEntityID != request.LegalEntityID {
		return ErrSubjectScopeMismatch
	}
	return nil
}
