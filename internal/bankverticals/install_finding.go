package bankverticals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/continuity"
)

func (s *Service) ensureLegacyFinding(ctx context.Context, config SeedConfig, program continuity.ProgramAggregate, sourceID string) error {
	matter, err := s.continuity.MatterByTriggerKey(ctx, config.TenantID, triggerFindingRemediation)
	if errors.Is(err, continuity.ErrNotFound) {
		_, err = s.seedLegacyFinding(ctx, config, program, sourceID)
		return err
	}
	if err != nil {
		return err
	}
	if !referenceMatter(matter.Matter, JourneyFindingRemediation) {
		return fmt.Errorf("finding reference trigger is already used by a non-reference issue")
	}
	if matter.Matter.Status == continuity.MatterCancelled {
		return fmt.Errorf("finding reference issue is cancelled and cannot be repaired")
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterAssessment, "The legacy finding and affected scope were reconciled.")
	if err != nil {
		return err
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterActionsInProgress, "The remediation was assigned to the processing-record owners.")
	if err != nil {
		return err
	}
	actions := currentActions(matter.Actions)
	if len(actions) == 0 {
		matter, err = s.continuity.AddAction(ctx, continuity.AddActionInput{
			TenantID:         config.TenantID,
			MatterID:         matter.Matter.ID,
			ExpectedVersion:  matter.Matter.Version,
			Title:            "Approve retention periods for the 14 affected records",
			Description:      "Record the approved retention period, approving owner and policy reference for each affected processing activity.",
			OwnerPrincipalID: config.OwnerPrincipalID,
			DueAt:            timePointer(config.Now.Add(-10 * 24 * time.Hour)),
			ActorID:          config.ActorID,
		})
		if err != nil {
			return err
		}
		actions = currentActions(matter.Actions)
	}
	action := actions[len(actions)-1]
	for _, target := range []continuity.ActionStatus{continuity.ActionInProgress, continuity.ActionImplemented} {
		if actionRank(action.Status) >= actionRank(target) {
			continue
		}
		matter, err = s.continuity.TransitionAction(ctx, continuity.TransitionActionInput{TenantID: config.TenantID, MatterID: matter.Matter.ID, ActionID: action.ID, ExpectedVersion: matter.Matter.Version, To: target, ActorID: config.OwnerPrincipalID})
		if err != nil {
			return err
		}
		actions = currentActions(matter.Actions)
		action = actions[len(actions)-1]
	}
	contract := activeVerificationContract(matter, action.ID)
	if contract == nil {
		matter, err = s.continuity.AddVerificationContract(ctx, continuity.AddVerificationContractInput{
			TenantID:                 config.TenantID,
			MatterID:                 matter.Matter.ID,
			ExpectedVersion:          matter.Matter.Version,
			ActionID:                 action.ID,
			ExpectedOutcome:          "All 14 affected processing records contain an approved retention period, owner and policy reference.",
			Baseline:                 mustJSON(map[string]any{"complete_records": 0, "affected_records": 14}),
			Scope:                    mustJSON(map[string]any{"finding_reference": "IA-PRIV-2024-07"}),
			MeasurementSourceID:      sourceID,
			Threshold:                mustJSON(map[string]any{"complete_records": 14, "exceptions": 0}),
			ObservationPeriodMinutes: 0,
			AuthorityPrincipalID:     config.ReviewerPrincipalID,
			FailureResponse:          "REOPEN",
			ActorID:                  config.ActorID,
		})
		if err != nil {
			return err
		}
		contract = activeVerificationContract(matter, action.ID)
	}
	matter, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterVerification, "Implementation is complete and independent checking has started.")
	if err != nil {
		return err
	}
	if !contractHasIndependentPass(matter, *contract) {
		observedAt := config.Now
		if action.ImplementedAt != nil && action.ImplementedAt.After(observedAt) {
			observedAt = *action.ImplementedAt
		}
		if contract.CreatedAt.After(observedAt) {
			observedAt = contract.CreatedAt
		}
		observedAt = observedAt.Add(time.Duration(contract.ObservationPeriodMinutes) * time.Minute)
		matter, err = s.continuity.RecordVerificationResult(ctx, continuity.RecordVerificationResultInput{
			TenantID:            config.TenantID,
			MatterID:            matter.Matter.ID,
			ExpectedVersion:     matter.Matter.Version,
			ContractID:          contract.ID,
			Result:              continuity.VerificationPassed,
			Observations:        mustJSON(map[string]any{"complete_records": 14, "exceptions": 0}),
			EvidenceReferences:  mustJSON([]string{"processing inventory export", "retention owner approvals", "sample re-performance"}),
			ReviewerPrincipalID: config.ReviewerPrincipalID,
			Rationale:           "The reviewer re-performed the check and confirmed all 14 records contain the required approved retention information.",
			ObservedAt:          observedAt,
		})
		if err != nil {
			return err
		}
	}
	_, err = advanceMatter(ctx, s.continuity, config, matter, continuity.MatterClosed, "The remediation was implemented and the independent outcome check passed.")
	return err
}
