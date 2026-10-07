package presentationprefs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("presentation preference not found")

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, tenantID, principalID string, roleCodes []string) (Preferences, error) {
	tenantID, principalID = strings.TrimSpace(tenantID), strings.TrimSpace(principalID)
	if s == nil || s.repo == nil || tenantID == "" || principalID == "" {
		return Preferences{}, ErrInvalid
	}
	stored, err := s.repo.Get(ctx, tenantID, principalID)
	if errors.Is(err, ErrNotFound) {
		stored = Stored{TenantID: tenantID, PrincipalID: principalID, HomeFocus: HomeFocusAuto, PortfolioLens: PortfolioLensAuto}
	} else if err != nil {
		return Preferences{}, err
	}
	return decorate(stored, roleCodes), nil
}

func (s *Service) Update(ctx context.Context, tenantID, principalID string, roleCodes []string, input UpdateInput) (Preferences, error) {
	tenantID, principalID = strings.TrimSpace(tenantID), strings.TrimSpace(principalID)
	if s == nil || s.repo == nil || tenantID == "" || principalID == "" || input.ExpectedVersion < 0 ||
		!validHomeFocus(input.HomeFocus) || !validPortfolioLens(input.PortfolioLens) {
		return Preferences{}, ErrInvalid
	}
	stored, err := s.repo.Upsert(ctx, Stored{
		TenantID:      tenantID,
		PrincipalID:   principalID,
		HomeFocus:     input.HomeFocus,
		PortfolioLens: input.PortfolioLens,
	}, input.ExpectedVersion)
	if err != nil {
		return Preferences{}, err
	}
	return decorate(stored, roleCodes), nil
}

func decorate(stored Stored, roleCodes []string) Preferences {
	effectiveHome, effectivePortfolio := roleDefaults(roleCodes)
	if stored.HomeFocus != HomeFocusAuto {
		effectiveHome = stored.HomeFocus
	}
	if stored.PortfolioLens != PortfolioLensAuto {
		effectivePortfolio = stored.PortfolioLens
	}
	return Preferences{
		TenantID:               stored.TenantID,
		PrincipalID:            stored.PrincipalID,
		HomeFocus:              stored.HomeFocus,
		PortfolioLens:          stored.PortfolioLens,
		EffectiveHomeFocus:     effectiveHome,
		EffectivePortfolioLens: effectivePortfolio,
		UpdatedAt:              stored.UpdatedAt,
		Version:                stored.Version,
	}
}

func roleDefaults(roleCodes []string) (HomeFocus, PortfolioLens) {
	roles := normalizeRoles(roleCodes)
	for _, role := range []string{"CRO", "CCO", "CISO", "DPO", "GENERAL_COUNSEL", "EXECUTIVE", "GRC_ADMIN", "GRC_ADMINISTRATOR", "RISK_MANAGER"} {
		if _, ok := roles[role]; ok {
			if role == "DPO" {
				return HomeFocusPosture, PortfolioLensProcessingActivities
			}
			return HomeFocusPosture, PortfolioLensRisks
		}
	}
	if _, ok := roles["RISK_OWNER"]; ok {
		return HomeFocusMyWork, PortfolioLensRisks
	}
	if _, ok := roles["BUSINESS_OWNER"]; ok {
		return HomeFocusMyWork, PortfolioLensVendors
	}
	return HomeFocusMyWork, PortfolioLensPrograms
}

type MemoryRepository struct {
	mu     sync.Mutex
	values map[string]Stored
	now    func() time.Time
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{values: map[string]Stored{}, now: time.Now}
}

func (r *MemoryRepository) Get(_ context.Context, tenantID, principalID string) (Stored, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.values[tenantID+":"+principalID]
	if !ok {
		return Stored{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) Upsert(_ context.Context, value Stored, expected int64) (Stored, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := value.TenantID + ":" + value.PrincipalID
	current, exists := r.values[key]
	if (!exists && expected != 0) || (exists && current.Version != expected) {
		return Stored{}, ErrVersionConflict
	}
	value.Version = expected + 1
	value.UpdatedAt = r.now().UTC()
	r.values[key] = value
	return value, nil
}
