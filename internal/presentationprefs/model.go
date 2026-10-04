package presentationprefs

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalid = errors.New("presentation preference is invalid")
	ErrVersionConflict = errors.New("presentation preference changed")
)

type HomeFocus string

const (
	HomeFocusAuto HomeFocus = "AUTO"
	HomeFocusPosture HomeFocus = "POSTURE"
	HomeFocusMyWork HomeFocus = "MY_WORK"
)

type PortfolioLens string

const (
	PortfolioLensAuto PortfolioLens = "AUTO"
	PortfolioLensPrograms PortfolioLens = "PROGRAMS"
	PortfolioLensRisks PortfolioLens = "RISKS"
	PortfolioLensLosses PortfolioLens = "LOSSES"
	PortfolioLensVendors PortfolioLens = "VENDORS"
	PortfolioLensProcessingActivities PortfolioLens = "PROCESSING_ACTIVITIES"
	PortfolioLensForms PortfolioLens = "FORMS"
)

type Preferences struct {
	TenantID string `json:"tenant_id"`
	PrincipalID string `json:"principal_id"`
	HomeFocus HomeFocus `json:"home_focus"`
	PortfolioLens PortfolioLens `json:"portfolio_lens"`
	EffectiveHomeFocus HomeFocus `json:"effective_home_focus"`
	EffectivePortfolioLens PortfolioLens `json:"effective_portfolio_lens"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	Version int64 `json:"version"`
}

type UpdateInput struct {
	HomeFocus HomeFocus `json:"home_focus"`
	PortfolioLens PortfolioLens `json:"portfolio_lens"`
	ExpectedVersion int64 `json:"expected_version"`
}

type Stored struct {
	TenantID string
	PrincipalID string
	HomeFocus HomeFocus
	PortfolioLens PortfolioLens
	UpdatedAt time.Time
	Version int64
}

type Repository interface {
	Get(context.Context, string, string) (Stored, error)
	Upsert(context.Context, Stored, int64) (Stored, error)
}

func validHomeFocus(value HomeFocus) bool {
	switch value {
	case HomeFocusAuto, HomeFocusPosture, HomeFocusMyWork:
		return true
	default:
		return false
	}
}

func validPortfolioLens(value PortfolioLens) bool {
	switch value {
	case PortfolioLensAuto, PortfolioLensPrograms, PortfolioLensRisks, PortfolioLensLosses, PortfolioLensVendors, PortfolioLensProcessingActivities, PortfolioLensForms:
		return true
	default:
		return false
	}
}

func normalizeRoles(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		value = strings.NewReplacer("-", "_", " ", "_").Replace(value)
		if value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}
