package notificationprefs

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, tenantID, principalID string) (Preferences, error) {
	tenantID, principalID = strings.TrimSpace(tenantID), strings.TrimSpace(principalID)
	if s == nil || s.repo == nil || tenantID == "" || principalID == "" {
		return Preferences{}, ErrInvalid
	}
	stored, err := s.repo.Get(ctx, tenantID, principalID)
	if errors.Is(err, ErrNotFound) {
		stored = Default(tenantID, principalID)
	} else if err != nil {
		return Preferences{}, err
	}
	return decorate(stored), nil
}

func (s *Service) Update(ctx context.Context, tenantID, principalID string, input UpdateInput) (Preferences, error) {
	tenantID, principalID = strings.TrimSpace(tenantID), strings.TrimSpace(principalID)
	stored := Stored{
		TenantID: tenantID, PrincipalID: principalID,
		DailyDigestEnabled: input.DailyDigestEnabled, DigestMinute: input.DigestMinute,
		TimeZone: strings.TrimSpace(input.TimeZone), QuietHoursEnabled: input.QuietHoursEnabled,
		QuietStartMinute: input.QuietStartMinute, QuietEndMinute: input.QuietEndMinute,
	}
	if s == nil || s.repo == nil || input.ExpectedVersion < 0 || !validate(stored) {
		return Preferences{}, ErrInvalid
	}
	saved, err := s.repo.Upsert(ctx, stored, input.ExpectedVersion)
	if err != nil {
		return Preferences{}, err
	}
	return decorate(saved), nil
}

func decorate(stored Stored) Preferences {
	return Preferences{
		TenantID: stored.TenantID, PrincipalID: stored.PrincipalID,
		DailyDigestEnabled: stored.DailyDigestEnabled, DigestMinute: stored.DigestMinute, TimeZone: stored.TimeZone,
		QuietHoursEnabled: stored.QuietHoursEnabled, QuietStartMinute: stored.QuietStartMinute, QuietEndMinute: stored.QuietEndMinute,
		CriticalEmailRequired: true, UpdatedAt: stored.UpdatedAt, Version: stored.Version,
	}
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

var _ Repository = (*MemoryRepository)(nil)
