package continuity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

type verificationContractSupersededEvent struct {
	Prior       VerificationContract `json:"prior"`
	Replacement VerificationContract `json:"replacement"`
	Rationale   string               `json:"rationale"`
}

type verificationContractRetiredEvent struct {
	Contract  VerificationContract `json:"contract"`
	Rationale string               `json:"rationale"`
}

func (s *Service) SupersedeVerificationContract(ctx context.Context, input SupersedeVerificationContractInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	prior, ok := verificationContractByID(aggregate.VerificationContracts, input.ContractID)
	if !ok {
		return MatterAggregate{}, ErrNotFound
	}
	if prior.Status != VerificationActive {
		return MatterAggregate{}, ErrInvalidState
	}
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	replacement, err := s.buildReplacementVerificationContract(ctx, aggregate, prior.ID, input)
	if err != nil {
		return MatterAggregate{}, err
	}
	now := s.now().UTC()
	prior.Status = VerificationRetired
	prior.UpdatedAt = now
	prior.Version++
	replacement.CreatedAt = now
	replacement.UpdatedAt = now
	payload := verificationContractSupersededEvent{Prior: prior, Replacement: replacement, Rationale: strings.TrimSpace(input.Rationale)}
	if err = s.applyMatterValue(ctx, input.TenantID, input.MatterID, input.ExpectedVersion, EventVerificationContractSuperseded, payload, input.ActorID); err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, input.TenantID, input.MatterID)
}

func (s *Service) RetireVerificationContract(ctx context.Context, input RetireVerificationContractInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	contract, ok := verificationContractByID(aggregate.VerificationContracts, input.ContractID)
	if !ok {
		return MatterAggregate{}, ErrNotFound
	}
	if contract.Status != VerificationActive {
		return MatterAggregate{}, ErrInvalidState
	}
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	contract.Status = VerificationRetired
	contract.UpdatedAt = s.now().UTC()
	contract.Version++
	payload := verificationContractRetiredEvent{Contract: contract, Rationale: strings.TrimSpace(input.Rationale)}
	if err = s.applyMatterValue(ctx, input.TenantID, input.MatterID, input.ExpectedVersion, EventVerificationContractRetired, payload, input.ActorID); err != nil {
		return MatterAggregate{}, err
	}
	return s.GetMatter(ctx, input.TenantID, input.MatterID)
}

func (s *Service) buildReplacementVerificationContract(ctx context.Context, aggregate MatterAggregate, supersedesID string, input SupersedeVerificationContractInput) (VerificationContract, error) {
	if strings.TrimSpace(input.ExpectedOutcome) == "" || strings.TrimSpace(input.FailureResponse) == "" {
		return VerificationContract{}, fmt.Errorf("expected_outcome and failure_response are required")
	}
	if input.ActionID != "" && !containsAction(aggregate.Actions, input.ActionID) {
		return VerificationContract{}, fmt.Errorf("action_id does not belong to this matter")
	}
	if !validFailureResponse(input.FailureResponse) {
		return VerificationContract{}, fmt.Errorf("failure_response must be REOPEN, CREATE_MATTER, ESCALATE or BLOCK_CLOSE")
	}
	if input.ObservationPeriodMinutes < 0 || input.ObservationPeriodMinutes > 525600 {
		return VerificationContract{}, fmt.Errorf("observation_period_minutes is outside the supported range")
	}
	if err := s.validateEvidenceSources(ctx, input.TenantID, aggregate.Matter.LegalEntityID, []string{input.MeasurementSourceID}); err != nil {
		return VerificationContract{}, err
	}
	baseline, err := normalizedJSON(input.Baseline, `{}`)
	if err != nil {
		return VerificationContract{}, err
	}
	scope, err := normalizedJSON(input.Scope, `{}`)
	if err != nil {
		return VerificationContract{}, err
	}
	threshold, err := normalizedJSON(input.Threshold, `{}`)
	if err != nil {
		return VerificationContract{}, err
	}
	replacementID, err := id.NewUUIDv7()
	if err != nil {
		return VerificationContract{}, err
	}
	return VerificationContract{
		ID: replacementID, TenantID: input.TenantID, MatterID: input.MatterID, SupersedesContractID: supersedesID,
		ActionID: input.ActionID, ExpectedOutcome: strings.TrimSpace(input.ExpectedOutcome), Baseline: json.RawMessage(baseline), Scope: json.RawMessage(scope),
		MeasurementSourceID: input.MeasurementSourceID, Threshold: json.RawMessage(threshold), ObservationPeriodMinutes: input.ObservationPeriodMinutes,
		AuthorityPrincipalID: input.AuthorityPrincipalID, FailureResponse: strings.ToUpper(strings.TrimSpace(input.FailureResponse)), Status: VerificationActive, Version: 1,
	}, nil
}
