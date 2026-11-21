package continuity

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

type FrameworkProgramUsage struct {
	TemplateCode       string `json:"template_code,omitempty"`
	EditionID          string `json:"edition_id,omitempty"`
	ActiveProgramCount int    `json:"active_program_count"`
}

type frameworkProgramUsageReader interface {
	FrameworkProgramUsage(context.Context, string, string, time.Time) ([]FrameworkProgramUsage, error)
}

func (s *Service) FrameworkProgramUsage(ctx context.Context) ([]FrameworkProgramUsage, error) {
	actor, err := identity.Require(ctx)
	if err != nil || strings.TrimSpace(actor.LegalEntityID) == "" || actor.LegalEntityID == "*" {
		return nil, ErrNotFound
	}
	entity, err := s.resolveLegalEntity(ctx, actor.TenantID, actor.LegalEntityID)
	if err != nil {
		return nil, err
	}
	reader, ok := s.repo.(frameworkProgramUsageReader)
	if !ok {
		return nil, fmt.Errorf("framework Program counts are unavailable")
	}
	return reader.FrameworkProgramUsage(ctx, actor.TenantID, entity, s.now().UTC())
}

func (r *MemoryRepository) FrameworkProgramUsage(_ context.Context, tenant, entity string, at time.Time) ([]FrameworkProgramUsage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	groups := map[string]map[string]bool{}
	identities := map[string]FrameworkProgramUsage{}
	for _, aggregate := range r.programs[tenant] {
		program := aggregate.Program
		if program.LegalEntityID != entity || program.Status != ProgramActive || program.EffectiveFrom.After(at) || (program.EffectiveUntil != nil && !at.Before(*program.EffectiveUntil)) {
			continue
		}
		for _, adoption := range aggregate.FrameworkAdoptions {
			if string(adoption.Status) != "ACTIVE" || adoption.EffectiveFrom == nil || adoption.EffectiveFrom.After(at) || (adoption.EffectiveUntil != nil && !at.Before(*adoption.EffectiveUntil)) {
				continue
			}
			key := "edition:" + adoption.EditionID
			value := FrameworkProgramUsage{EditionID: adoption.EditionID}
			if adoption.TemplateCode != "" {
				key = "template:" + adoption.TemplateCode
				value = FrameworkProgramUsage{TemplateCode: adoption.TemplateCode}
			}
			if groups[key] == nil {
				groups[key] = map[string]bool{}
				identities[key] = value
			}
			groups[key][program.ID] = true
		}
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]FrameworkProgramUsage, 0, len(keys))
	for _, key := range keys {
		value := identities[key]
		value.ActiveProgramCount = len(groups[key])
		result = append(result, value)
	}
	return result, nil
}
