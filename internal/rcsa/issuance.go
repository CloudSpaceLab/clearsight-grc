package rcsa

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/evidence"
)

type IssueResult struct {
	Link         DistributionLink          `json:"link"`
	Distribution evidence.FormDistribution `json:"distribution"`
}

func (s *Service) ConfigureDistributions(distributions *evidence.DistributionService) {
	if s != nil {
		s.distributions = distributions
	}
}

func (s *Service) IssueItem(ctx context.Context, scope Scope, cycleID, itemID, actorID string) (IssueResult, error) {
	if s == nil || s.repository == nil || s.distributions == nil {
		return IssueResult{}, ErrInvalid
	}
	scope, err := normalizeScope(scope)
	cycleID = strings.TrimSpace(cycleID)
	itemID = strings.TrimSpace(itemID)
	actorID = strings.TrimSpace(actorID)
	if err != nil || cycleID == "" || itemID == "" || actorID == "" {
		return IssueResult{}, ErrInvalid
	}
	aggregate, err := s.repository.GetCycle(ctx, scope, cycleID)
	if err != nil {
		return IssueResult{}, err
	}
	if aggregate.Cycle.CreatedBy != actorID {
		return IssueResult{}, ErrInvalid
	}
	var item *Item
	for index := range aggregate.Items {
		if aggregate.Items[index].ID == itemID {
			item = &aggregate.Items[index]
			break
		}
	}
	if item == nil {
		return IssueResult{}, ErrNotFound
	}
	if existing, ok := aggregate.Distributions[itemID]; ok {
		bundle, getErr := s.distributions.Get(ctx, aggregate.Cycle.TenantID, aggregate.Cycle.LegalEntityID, existing.DistributionID)
		if getErr != nil {
			return IssueResult{}, getErr
		}
		return IssueResult{Link: existing, Distribution: bundle.Distribution}, nil
	}
	now := s.now()
	if !aggregate.Cycle.DueAt.After(now) {
		return IssueResult{}, ErrConflict
	}
	title := fmt.Sprintf("%s · %s", aggregate.Cycle.Name, item.RiskCode)
	purpose := "Complete the first-line RCSA assessment for this risk."
	bundle, err := s.distributions.Create(ctx, evidence.CreateDistributionInput{
		IdempotencyKey:      "rcsa:" + aggregate.Cycle.ID + ":" + item.ID,
		TenantID:            aggregate.Cycle.TenantID,
		LegalEntityID:       aggregate.Cycle.LegalEntityID,
		FormTemplateID:      aggregate.Cycle.FormTemplateID,
		FormTemplateVersion: aggregate.Cycle.FormTemplateVersion,
		SubjectType:         "RISK",
		SubjectID:           item.RiskID,
		Title:               title,
		Purpose:             purpose,
		AccessPolicy:        evidence.AccessDirectMagicLink,
		EstimatedMinutes:    15,
		Deadline:            aggregate.Cycle.DueAt,
		RouteExpiresAt:      aggregate.Cycle.DueAt,
		CreatedBy:           actorID,
		Recipients: []evidence.DistributionRecipientInput{{
			Role: evidence.RecipientTo, Type: evidence.RecipientInternalPrincipal, PrincipalID: item.RespondentPrincipalID,
		}},
	})
	if err != nil {
		if errors.Is(err, evidence.ErrDistributionConflict) {
			return IssueResult{}, ErrConflict
		}
		return IssueResult{}, err
	}
	if bundle.Distribution.SubjectType != "RISK" || bundle.Distribution.SubjectID != item.RiskID ||
		bundle.Distribution.TenantID != aggregate.Cycle.TenantID || bundle.Distribution.LegalEntityID != aggregate.Cycle.LegalEntityID {
		return IssueResult{}, ErrInvalid
	}
	link, err := s.repository.AttachDistribution(ctx, scope, DistributionLink{
		ItemID: item.ID, DistributionID: bundle.Distribution.ID, IssuedAt: bundle.Distribution.UpdatedAt,
	})
	if err != nil {
		return IssueResult{}, err
	}
	return IssueResult{Link: link, Distribution: bundle.Distribution}, nil
}

var _ = time.Time{}
