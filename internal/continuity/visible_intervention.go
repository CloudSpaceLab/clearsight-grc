package continuity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

var ErrInterventionUnavailable = errors.New("intervention lookup unavailable")

// VisibleIntervention is the minimal issue information exposed by an authorized exact-trigger lookup.
type VisibleIntervention struct {
	ID        string
	Reference string
	Status    MatterStatus
	Priority  int
	CreatedAt time.Time
}
type visibleInterventionRepository interface {
	VisibleIntervention(context.Context, string, string, string, string, string) (VisibleIntervention, error)
}

func (s *Service) VisibleIntervention(ctx context.Context, tenant, entity, trigger, program string) (VisibleIntervention, error) {
	actor, err := identity.Require(ctx)
	if err != nil || actor.Valid(time.Now()) != nil || actor.TenantID != tenant || actor.LegalEntityID != entity || strings.TrimSpace(entity) == "" || entity == "*" || strings.TrimSpace(trigger) == "" || strings.TrimSpace(program) == "" {
		return VisibleIntervention{}, ErrNotFound
	}
	repo, ok := s.repo.(visibleInterventionRepository)
	if !ok {
		return VisibleIntervention{}, ErrInterventionUnavailable
	}
	return repo.VisibleIntervention(ctx, tenant, entity, actor.PrincipalID, trigger, program)
}

func (r *MemoryRepository) VisibleIntervention(_ context.Context, tenant, entity, principal, trigger, program string) (VisibleIntervention, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, value := range r.matters[tenant] {
		m := value.Matter
		if m.TenantID != tenant || m.LegalEntityID != entity || m.TriggerKey != trigger || m.Status == MatterClosed || m.Status == MatterCancelled || !MatterAggregateVisibleTo(value, principal) {
			continue
		}
		for _, link := range value.Links {
			if link.ProgramID == program && link.RetiredAt == nil {
				return VisibleIntervention{ID: m.ID, Reference: m.Reference, Status: m.Status, Priority: m.Priority, CreatedAt: m.CreatedAt}, nil
			}
		}
	}
	return VisibleIntervention{}, ErrNotFound
}
