package rcsa

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/monitoring"
	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
	"github.com/CloudSpaceLab/clearsight-grc/internal/risk"
)

const maxCycleRisks = 200

type ReusableFormReader interface {
	ReusableFormRevision(context.Context, string, string, string, int64) (monitoring.FormTemplate, error)
}

type Service struct {
	repository Repository
	risks      *risk.Service
	forms      ReusableFormReader
	Now        func() time.Time
}

func NewService(repository Repository, risks *risk.Service, forms ReusableFormReader) *Service {
	return &Service{repository: repository, risks: risks, forms: forms}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Aggregate, error) {
	if s == nil || s.repository == nil || s.risks == nil || s.forms == nil {
		return Aggregate{}, ErrInvalid
	}
	scope, err := normalizeScope(Scope{TenantID: input.TenantID, LegalEntityID: input.LegalEntityID})
	if err != nil {
		return Aggregate{}, err
	}
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	input.Name = strings.TrimSpace(input.Name)
	input.FormTemplateID = strings.TrimSpace(input.FormTemplateID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.Code == "" || len(input.Code) > 128 || input.Name == "" || len(input.Name) > 240 ||
		input.FormTemplateID == "" || input.FormTemplateVersion < 1 || input.ActorID == "" ||
		input.PeriodStart.IsZero() || input.PeriodEnd.IsZero() || input.DueAt.IsZero() || input.ChallengeDueAt.IsZero() ||
		!input.PeriodEnd.After(input.PeriodStart) || input.DueAt.Before(input.PeriodEnd) ||
		input.ChallengeDueAt.Before(input.DueAt) || len(input.RiskIDs) < 1 || len(input.RiskIDs) > maxCycleRisks {
		return Aggregate{}, ErrInvalid
	}
	form, err := s.forms.ReusableFormRevision(ctx, scope.TenantID, scope.LegalEntityID, input.FormTemplateID, input.FormTemplateVersion)
	if err != nil || form.Status != monitoring.LifecycleActive || !form.IsCurrent ||
		form.ID != input.FormTemplateID || form.Version != input.FormTemplateVersion ||
		form.LegalEntityID != scope.LegalEntityID || form.ProgramID != "" {
		return Aggregate{}, ErrInvalid
	}

	now := s.now()
	cycleID, err := id.NewUUIDv7()
	if err != nil {
		return Aggregate{}, err
	}
	cycle := Cycle{
		ID: cycleID, TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID,
		Code: input.Code, Name: input.Name, FormTemplateID: form.ID, FormTemplateVersion: form.Version,
		PeriodStart: input.PeriodStart.UTC(), PeriodEnd: input.PeriodEnd.UTC(),
		DueAt: input.DueAt.UTC(), ChallengeDueAt: input.ChallengeDueAt.UTC(),
		CreatedBy: input.ActorID, CreatedAt: now,
	}

	seen := make(map[string]struct{}, len(input.RiskIDs))
	items := make([]Item, 0, len(input.RiskIDs))
	controls := make([]ItemControl, 0)
	for _, rawRiskID := range input.RiskIDs {
		riskID := strings.TrimSpace(rawRiskID)
		if riskID == "" {
			return Aggregate{}, ErrInvalid
		}
		if _, exists := seen[riskID]; exists {
			return Aggregate{}, ErrDuplicate
		}
		seen[riskID] = struct{}{}
		current, getErr := s.risks.Get(ctx, risk.Scope{TenantID: scope.TenantID, LegalEntityID: scope.LegalEntityID}, riskID)
		if getErr != nil || current.Risk.Status != risk.StatusActive || strings.TrimSpace(current.Risk.OwnerPrincipalID) == "" {
			return Aggregate{}, ErrInvalid
		}
		itemID, idErr := id.NewUUIDv7()
		if idErr != nil {
			return Aggregate{}, idErr
		}
		item := Item{
			ID: itemID, CycleID: cycle.ID, TenantID: current.Risk.TenantID, LegalEntityID: current.Risk.LegalEntityID,
			RiskID: current.Risk.ID, RiskVersion: current.Risk.Version, RiskCode: current.Risk.Code, RiskName: current.Risk.Name,
			RespondentPrincipalID: current.Risk.OwnerPrincipalID, CreatedAt: now,
		}
		items = append(items, item)
		for _, control := range current.Controls {
			if control.RiskVersion > current.Risk.Version {
				return Aggregate{}, ErrInvalid
			}
			controls = append(controls, ItemControl{ItemID: item.ID, RiskID: item.RiskID, ControlLinkID: control.ID})
		}
	}
	return s.repository.CreateCycle(ctx, cycle, items, controls)
}

func (s *Service) Get(ctx context.Context, scope Scope, cycleID string) (Aggregate, error) {
	if s == nil || s.repository == nil {
		return Aggregate{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	if err != nil || strings.TrimSpace(cycleID) == "" {
		return Aggregate{}, ErrInvalid
	}
	return s.repository.GetCycle(ctx, scope, strings.TrimSpace(cycleID))
}

func (s *Service) AttachDistribution(ctx context.Context, scope Scope, itemID, distributionID string, issuedAt time.Time) (DistributionLink, error) {
	if s == nil || s.repository == nil {
		return DistributionLink{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	if err != nil || strings.TrimSpace(itemID) == "" || strings.TrimSpace(distributionID) == "" || issuedAt.IsZero() {
		return DistributionLink{}, ErrInvalid
	}
	return s.repository.AttachDistribution(ctx, scope, DistributionLink{
		ItemID: strings.TrimSpace(itemID), DistributionID: strings.TrimSpace(distributionID), IssuedAt: issuedAt.UTC(),
	})
}

func (s *Service) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func normalizeRepositoryError(err error) error {
	if errors.Is(err, risk.ErrNotFound) || errors.Is(err, monitoring.ErrNotFound) {
		return ErrInvalid
	}
	return err
}
