package bankverticals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

const (
	regulatoryChecklistActionTitle          = "Update the annual return evidence checklist"
	regulatoryEvidenceCollectionActionTitle = "Collect the two outstanding annual return evidence sections"
	regulatoryEvidenceCollectionOriginKey   = "reference:gaid-2025:outstanding-evidence"
)

func (s *Service) ensureRegulatoryChange(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) error {
	matter, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerRegulatoryChange)
	if errors.Is(err, continuity.ErrNotFound) {
		matter, err = s.seedRegulatoryChange(ctx, config, program, sourceID)
	}
	if err != nil {
		return err
	}
	if !referenceMatter(matter.Matter, JourneyRegulatoryChange) {
		return fmt.Errorf("trigger key %s is already used by a non-reference issue", triggerRegulatoryChange)
	}
	if matter.Matter.Status == continuity.MatterClosed || matter.Matter.Status == continuity.MatterCancelled {
		return fmt.Errorf("regulatory-change reference issue cannot be repaired from %s", matter.Matter.Status)
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterAssessment, "The official source and affected processes were reconciled.")
	if err != nil {
		return err
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterDecisionRequired, "The reconciled change requires an approved bank position.")
	if err != nil {
		return err
	}
	if !currentDecisionApproved(matter.Decisions) {
		matter, err = s.continuity.AddDecision(ctx, continuity.AddDecisionInput{
			TenantID:             config.TenantID,
			MatterID:             matter.Matter.ID,
			ExpectedVersion:      matter.Matter.Version,
			Type:                 "IMPLEMENTATION_APPROACH",
			Status:               continuity.DecisionApproved,
			Options:              mustJSON([]string{"UPDATE_CURRENT_PROCESS", "CREATE_SEPARATE_GAID_PROCESS", "NO_CHANGE_REQUIRED"}),
			SelectedOption:       "UPDATE_CURRENT_PROCESS",
			Rationale:            "The existing annual return process will use source-linked evidence owners and an earlier internal review date.",
			Conditions:           mustJSON([]string{"DPO confirms the final evidence checklist", "DPCO review starts at least 30 days before filing"}),
			AuthorityPrincipalID: config.SignatoryPrincipalID,
		})
		if err != nil {
			return err
		}
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterActionsInProgress, "The approved process changes are being implemented.")
	if err != nil {
		return err
	}
	checklist, checklistFound := regulatoryChecklistAction(matter)
	if !checklistFound {
		matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{
			TenantID:         config.TenantID,
			MatterID:         matter.Matter.ID,
			ExpectedVersion:  matter.Matter.Version,
			Title:            regulatoryChecklistActionTitle,
			Description:      "Assign each evidence section, record its authoritative source and move the internal approval date to 1 March.",
			OwnerPrincipalID: config.OwnerPrincipalID,
			DueAt:            timePointer(config.Now.Add(10 * 24 * time.Hour)),
			ActorID:          config.ActorID,
		})
		if err != nil {
			return err
		}
		checklist, checklistFound = regulatoryChecklistAction(matter)
	}
	if !checklistFound {
		return fmt.Errorf("reference checklist action was not recorded")
	}
	if checklist.Status == continuity.ActionPlanned {
		matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: checklist.ID, ExpectedVersion: matter.Matter.Version, To: continuity.ActionInProgress, ActorID: config.OwnerPrincipalID})
		if err != nil {
			return err
		}
	}
	if _, found := actionByOriginKey(matter.Actions, regulatoryEvidenceCollectionOriginKey); !found {
		matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{
			TenantID:         config.TenantID,
			MatterID:         matter.Matter.ID,
			ExpectedVersion:  matter.Matter.Version,
			Title:            regulatoryEvidenceCollectionActionTitle,
			OriginKey:        regulatoryEvidenceCollectionOriginKey,
			Description:      "Provide the authoritative source and proposed evidence owner for the two annual return sections that are not yet complete.",
			OwnerPrincipalID: config.ContributorPrincipalID,
			DueAt:            timePointer(config.Now.Add(6 * 24 * time.Hour)),
			ActorID:          config.ActorID,
		})
		if err != nil {
			return err
		}
	}
	if !hasActiveVerificationContract(matter, checklist.ID) {
		_, err = s.continuity.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
			TenantID:                 config.TenantID,
			MatterID:                 matter.Matter.ID,
			ExpectedVersion:          matter.Matter.Version,
			ActionID:                 checklist.ID,
			ExpectedOutcome:          "Every required annual return evidence section has an owner, authoritative source, internal approval date and DPCO review status.",
			Baseline:                 mustJSON(map[string]any{"complete_sections": 8, "required_sections": 10}),
			Scope:                    mustJSON(map[string]any{"journey_code": JourneyRegulatoryChange, "filing_year": 2027}),
			Threshold:                mustJSON(map[string]any{"complete_sections": 10, "approved": true}),
			ObservationPeriodMinutes: 0,
			AuthorityPrincipalID:     config.ReviewerPrincipalID,
			FailureResponse:          "BLOCK_CLOSE",
			ActorID:                  config.ActorID,
		})
	}
	return err
}

func currentActionByTitle(actions []continuity.Action, title string) (continuity.Action, bool) {
	for _, action := range actions {
		if action.Title == title && action.Status != continuity.ActionCancelled {
			return action, true
		}
	}
	return continuity.Action{}, false
}

func regulatoryChecklistAction(matter continuity.MatterAggregate) (continuity.Action, bool) {
	for _, contract := range matter.VerificationContracts {
		if contract.Status != continuity.VerificationActive || contract.ActionID == "" || scopeString(contract.Scope, "journey_code") != string(JourneyRegulatoryChange) {
			continue
		}
		for _, action := range matter.Actions {
			if action.ID == contract.ActionID && action.Status != continuity.ActionCancelled {
				return action, true
			}
		}
	}
	return currentActionByTitle(matter.Actions, regulatoryChecklistActionTitle)
}

func actionByOriginKey(actions []continuity.Action, originKey string) (continuity.Action, bool) {
	for _, action := range actions {
		if action.OriginKey == originKey {
			return action, true
		}
	}
	return continuity.Action{}, false
}
