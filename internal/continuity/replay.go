package continuity

import (
	"encoding/json"
	"fmt"
)

const (
	EventProgramCreated             = "PROGRAM_CREATED"
	EventProgramStatusChanged       = "PROGRAM_STATUS_CHANGED"
	EventRequirementAdded           = "REQUIREMENT_ADDED"
	EventApplicabilityDetermined    = "APPLICABILITY_DETERMINED"
	EventControlObjectiveAdded      = "CONTROL_OBJECTIVE_ADDED"
	EventControlImplementationAdded = "CONTROL_IMPLEMENTATION_ADDED"
	EventRequirementControlLinked   = "REQUIREMENT_CONTROL_LINKED"
	EventEvidenceContractAdded      = "EVIDENCE_CONTRACT_ADDED"
	EventEvidenceAssessmentRecorded = "EVIDENCE_ASSESSMENT_RECORDED"
	EventProgramStateUpdated        = "PROGRAM_STATE_UPDATED"
	EventProgramTriggerRecorded     = "PROGRAM_TRIGGER_RECORDED"

	EventMatterCreated               = "MATTER_CREATED"
	EventMatterLinked                = "MATTER_LINKED"
	EventMatterStateChanged          = "MATTER_STATE_CHANGED"
	EventDecisionAdded               = "DECISION_ADDED"
	EventActionAdded                 = "ACTION_ADDED"
	EventActionStateChanged          = "ACTION_STATE_CHANGED"
	EventVerificationContractAdded   = "VERIFICATION_CONTRACT_ADDED"
	EventVerificationResultRecorded  = "VERIFICATION_RESULT_RECORDED"
	EventResponsePackageAdded        = "RESPONSE_PACKAGE_ADDED"
	EventResponsePackageStateChanged = "RESPONSE_PACKAGE_STATE_CHANGED"
)

type matterStateChange struct {
	From          MatterStatus `json:"from"`
	To            MatterStatus `json:"to"`
	Rationale     string       `json:"rationale"`
	ClosedAt      *string      `json:"closed_at,omitempty"`
	ClosureReason string       `json:"closure_reason,omitempty"`
	ReopenCount   int          `json:"reopen_count"`
}

func reconstructProgram(events []Event) (ProgramAggregate, error) {
	var aggregate ProgramAggregate
	for _, event := range events {
		switch event.Type {
		case EventProgramCreated:
			if err := json.Unmarshal(event.Payload, &aggregate.Program); err != nil {
				return ProgramAggregate{}, fmt.Errorf("decode program event: %w", err)
			}
		case EventProgramStatusChanged:
			previousStatus := aggregate.Program.Status
			previousUntil := aggregate.Program.EffectiveUntil
			var value Program
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			// Historical resume events written before P1.1 cleared effective_until.
			// Domain semantics treat that null as "keep the configured Program
			// period" when resuming a paused Program, matching the normalized
			// PostgreSQL projection invariant introduced in migration 000015.
			if previousStatus == ProgramPaused && value.Status == ProgramActive && previousUntil != nil && value.EffectiveUntil == nil {
				until := *previousUntil
				value.EffectiveUntil = &until
			}
			aggregate.Program = value
		case EventRequirementAdded:
			var value Requirement
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.Requirements = upsertRequirement(aggregate.Requirements, value)
		case EventApplicabilityDetermined:
			var value Applicability
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.Applicability = append(aggregate.Applicability, value)
		case EventControlObjectiveAdded:
			var value ControlObjective
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.ControlObjectives = upsertObjective(aggregate.ControlObjectives, value)
		case EventControlImplementationAdded:
			var value ControlImplementation
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.ControlImplementations = upsertImplementation(aggregate.ControlImplementations, value)
		case EventRequirementControlLinked:
			var value RequirementControlLink
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.RequirementControlLinks = append(aggregate.RequirementControlLinks, value)
		case EventEvidenceContractAdded:
			var value EvidenceContract
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.EvidenceContracts = upsertEvidenceContract(aggregate.EvidenceContracts, value)
		case EventEvidenceAssessmentRecorded:
			var value EvidenceAssessment
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.EvidenceAssessments = append(aggregate.EvidenceAssessments, value)
		case EventProgramStateUpdated:
			var value ProgramStateSnapshot
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.CurrentState = &value
		case EventProgramTriggerRecorded:
			var value Trigger
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProgramAggregate{}, err
			}
			aggregate.Triggers = append(aggregate.Triggers, value)
		default:
			return ProgramAggregate{}, fmt.Errorf("unsupported program event %s", event.Type)
		}
		if aggregate.Program.ID != "" {
			aggregate.Program.Version = event.AggregateVersion
		}
	}
	if aggregate.Program.ID == "" {
		return ProgramAggregate{}, ErrNotFound
	}
	return decorateProgram(aggregate), nil
}

func reconstructMatter(events []Event) (MatterAggregate, error) {
	var aggregate MatterAggregate
	for _, event := range events {
		switch event.Type {
		case EventMatterCreated:
			if err := json.Unmarshal(event.Payload, &aggregate.Matter); err != nil {
				return MatterAggregate{}, fmt.Errorf("decode matter event: %w", err)
			}
		case EventMatterLinked:
			var value MatterLink
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return MatterAggregate{}, err
			}
			aggregate.Links = append(aggregate.Links, value)
		case EventMatterStateChanged:
			var value Matter
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return MatterAggregate{}, err
			}
			aggregate.Matter = value
		case EventDecisionAdded:
			var value Decision
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return MatterAggregate{}, err
			}
			aggregate.Decisions = upsertDecision(aggregate.Decisions, value)
		case EventActionAdded, EventActionStateChanged:
			var value Action
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return MatterAggregate{}, err
			}
			aggregate.Actions = upsertAction(aggregate.Actions, value)
		case EventVerificationContractAdded:
			var value VerificationContract
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return MatterAggregate{}, err
			}
			aggregate.VerificationContracts = upsertVerificationContract(aggregate.VerificationContracts, value)
		case EventVerificationResultRecorded:
			var value VerificationResult
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return MatterAggregate{}, err
			}
			aggregate.VerificationResults = append(aggregate.VerificationResults, value)
		case EventResponsePackageAdded, EventResponsePackageStateChanged:
			var value ResponsePackage
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return MatterAggregate{}, err
			}
			aggregate.ResponsePackages = upsertResponsePackage(aggregate.ResponsePackages, value)
		default:
			return MatterAggregate{}, fmt.Errorf("unsupported matter event %s", event.Type)
		}
		if aggregate.Matter.ID != "" {
			aggregate.Matter.Version = event.AggregateVersion
		}
	}
	if aggregate.Matter.ID == "" {
		return MatterAggregate{}, ErrNotFound
	}
	aggregate.Closure = assessClosure(aggregate)
	return decorateMatter(aggregate), nil
}

func upsertRequirement(values []Requirement, value Requirement) []Requirement {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertObjective(values []ControlObjective, value ControlObjective) []ControlObjective {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertImplementation(values []ControlImplementation, value ControlImplementation) []ControlImplementation {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertEvidenceContract(values []EvidenceContract, value EvidenceContract) []EvidenceContract {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertDecision(values []Decision, value Decision) []Decision {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertAction(values []Action, value Action) []Action {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertVerificationContract(values []VerificationContract, value VerificationContract) []VerificationContract {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}

func upsertResponsePackage(values []ResponsePackage, value ResponsePackage) []ResponsePackage {
	for index := range values {
		if values[index].ID == value.ID {
			values[index] = value
			return values
		}
	}
	return append(values, value)
}
