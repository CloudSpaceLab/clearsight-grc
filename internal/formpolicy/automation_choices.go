package formpolicy

import (
	"context"
	"encoding/json"
	"github.com/CloudSpaceLab/clearsight-grc/internal/autonomy"
	"time"
)

type AutomationChoice struct {
	UseCount       int             `json:"use_count"`
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Purpose        string          `json:"purpose"`
	Status         string          `json:"status"`
	Version        int64           `json:"version"`
	Eligibility    Eligibility     `json:"eligibility"`
	BlastRadius    BlastRadius     `json:"blast_radius"`
	Outcome        OutcomeContract `json:"outcome_contract"`
	Rollout        RolloutMode     `json:"rollout"`
	EffectiveFrom  *time.Time      `json:"effective_from,omitempty"`
	EffectiveUntil *time.Time      `json:"effective_until,omitempty"`
}

type automationChoiceReader interface {
	ListAutomationChoices(context.Context, string, string, string, int64, int) ([]AutomationChoice, error)
}

func (service *Service) AutomationChoices(ctx context.Context, actor Actor, formID string, version int64) ([]AutomationChoice, error) {
	if service == nil || !validActor(actor) || formID == "" || version < 1 {
		return nil, ErrInvalid
	}
	actor = normalizeActor(actor)
	if err := service.requireActiveForm(ctx, actor, Eligibility{FormTemplateID: formID, FormTemplateVersion: version, ResultBasis: ResultBankAssessed}); err != nil {
		return nil, err
	}
	reader, ok := service.repo.(automationChoiceReader)
	if !ok {
		return nil, ErrAuthorityUnavailable
	}
	return reader.ListAutomationChoices(ctx, actor.TenantID, actor.LegalEntityID, formID, version, 100)
}

func automationChoice(value autonomy.AutomationPolicy) (AutomationChoice, error) {
	result := AutomationChoice{ID: value.ID, Name: value.Name, Status: string(value.Status), Version: value.Version, Rollout: RolloutMode(value.RolloutMode), EffectiveFrom: value.EffectiveFrom, EffectiveUntil: value.EffectiveUntil}
	if json.Unmarshal(value.Eligibility, &result.Eligibility) != nil || json.Unmarshal(value.BlastRadiusLimit, &result.BlastRadius) != nil || json.Unmarshal(value.VerificationContract, &result.Outcome) != nil {
		return result, ErrInvalid
	}
	result.Purpose = result.Outcome.ExpectedOutcome
	return result, nil
}

func (repo *MemoryRepository) ListAutomationChoices(ctx context.Context, tenant, entity, form string, version int64, limit int) ([]AutomationChoice, error) {
	repo.mu.RLock()
	defer repo.mu.RUnlock()
	if repo.automation == nil {
		return []AutomationChoice{}, nil
	}
	values := repo.automation.ListFormPolicyChoices(tenant, form, version, limit)
	result := []AutomationChoice{}
	for _, value := range values {
		choice, err := automationChoice(value)
		if err != nil {
			return nil, err
		}
		for _, policy := range repo.policies {
			if policy.TenantID == tenant && policy.LegalEntityID == entity && policy.AutomationPolicyID == choice.ID && policy.AutomationPolicyVersion == choice.Version {
				choice.UseCount++
				choice.Purpose = policy.Purpose
			}
		}
		result = append(result, choice)
	}
	return result, nil
}
