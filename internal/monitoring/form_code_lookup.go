package monitoring

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type formCodeRepository interface {
	LatestFormByCode(context.Context, string, string, string, string) (FormTemplate, error)
}

// LatestFormByCode is an exact scoped lookup for governed reference installers.
// Ambiguous codes fail rather than selecting one form or creating a duplicate.
func (s *Service) LatestFormByCode(ctx context.Context, actor Actor, programID, code string) (FormTemplate, error) {
	if err := validateActor(actor); err != nil {
		return FormTemplate{}, err
	}
	if err := validateProgramScope(actor, programID, actor.LegalEntityID); err != nil {
		return FormTemplate{}, err
	}
	if strings.TrimSpace(code) == "" {
		return FormTemplate{}, ErrInvalid
	}
	repo, ok := s.repo.(formCodeRepository)
	if !ok {
		return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("exact form lookup is unavailable"))
	}
	return repo.LatestFormByCode(ctx, actor.TenantID, actor.LegalEntityID, strings.TrimSpace(programID), strings.TrimSpace(code))
}

func (r *MemoryRepository) LatestFormByCode(_ context.Context, tenant, entity, program, code string) (FormTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var current FormTemplate
	for _, form := range r.forms {
		if form.TenantID != tenant || form.LegalEntityID != entity || form.ProgramID != program || form.Code != code {
			continue
		}
		if current.ID != "" && current.ID != form.ID {
			return FormTemplate{}, errors.Join(ErrInvalid, fmt.Errorf("form code identifies more than one form"))
		}
		if current.ID == "" || form.Version > current.Version {
			current = form
		}
	}
	if current.ID == "" {
		return FormTemplate{}, ErrNotFound
	}
	return cloneValue(current), nil
}
