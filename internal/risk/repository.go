package risk

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalid         = errors.New("risk input is invalid")
	ErrNotFound        = errors.New("risk not found")
	ErrDuplicate       = errors.New("risk code already exists")
	ErrVersionConflict = errors.New("risk version conflict")
	ErrScopeMismatch   = errors.New("risk is outside the requested scope")
)

type Repository interface {
	Create(context.Context, Risk, Event) (Risk, error)
	Get(context.Context, Scope, string) (Risk, error)
	Update(context.Context, Scope, Risk, int64, Event) (Risk, error)
	AddAssessment(context.Context, Scope, string, int64, Assessment, Event) (Risk, Assessment, error)
	AddAppetite(context.Context, Scope, string, int64, AppetiteStatement, Event) (Risk, AppetiteStatement, error)
	AddControl(context.Context, Scope, string, int64, ControlLink, Event) (Risk, ControlLink, error)
	Assessments(context.Context, Scope, string, int) ([]Assessment, error)
	AppetiteStatements(context.Context, Scope, string, int) ([]AppetiteStatement, error)
	CurrentAppetite(context.Context, Scope, string, time.Time) (*AppetiteStatement, error)
	Controls(context.Context, Scope, string, int) ([]ControlLink, error)
	List(context.Context, Scope, ListFilter) (Page, error)
}

func normalizeScope(scope Scope) (Scope, error) {
	scope.TenantID = strings.TrimSpace(scope.TenantID)
	scope.LegalEntityID = strings.TrimSpace(scope.LegalEntityID)
	if scope.TenantID == "" || scope.LegalEntityID == "" || scope.LegalEntityID == "*" {
		return Scope{}, ErrInvalid
	}
	return scope, nil
}
