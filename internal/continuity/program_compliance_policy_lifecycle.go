package continuity

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

// The Program owns calculation configuration. Only the current proposal and
// effective revision are projected; immutable prior revisions remain in the
// existing Program event stream.
type ProgramCompliancePolicyRevision struct {
	Number     int64                              `json:"number"`
	Definition ProgramComplianceCalculationPolicy `json:"definition"`
	Checksum   string                             `json:"checksum"`
	MakerID    string                             `json:"maker_id"`
	ProposedAt time.Time                          `json:"proposed_at"`
	ApproverID string                             `json:"approver_id,omitempty"`
	ApprovedAt *time.Time                         `json:"approved_at,omitempty"`
}

type ProgramCompliancePolicyPosition struct {
	LastRevision int64                            `json:"last_revision"`
	Active       *ProgramCompliancePolicyRevision `json:"active,omitempty"`
	Pending      *ProgramCompliancePolicyRevision `json:"pending,omitempty"`
}

type ProposeProgramCompliancePolicyInput struct {
	TenantID        string                             `json:"tenant_id,omitempty"`
	ProgramID       string                             `json:"program_id"`
	ExpectedVersion int64                              `json:"expected_version"`
	Definition      ProgramComplianceCalculationPolicy `json:"definition"`
	ActorID         string                             `json:"actor_id,omitempty"`
	Rationale       string                             `json:"rationale"`
}

type DecideProgramCompliancePolicyInput struct {
	TenantID        string `json:"tenant_id,omitempty"`
	ProgramID       string `json:"program_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ActorID         string `json:"actor_id,omitempty"`
	Rationale       string `json:"rationale"`
}

type programCompliancePolicyChangedEvent struct {
	Position  ProgramCompliancePolicyPosition `json:"position"`
	Rationale string                          `json:"rationale"`
}

func (s *Service) ProposeProgramCompliancePolicy(ctx context.Context, input ProposeProgramCompliancePolicyInput) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	maker := strings.TrimSpace(input.ActorID)
	rationale := strings.TrimSpace(input.Rationale)
	if !programComplianceActorMatches(ctx, input.TenantID, maker) ||
		rationale == "" || len(rationale) > 2000 {
		return ProgramAggregate{}, ErrProgramComplianceCalculationInvalid
	}
	position := cloneProgramCompliancePolicyPosition(aggregate.Program.CalculationPolicy)
	if position.Pending != nil {
		return ProgramAggregate{}, ErrInvalidState
	}
	if position.LastRevision < 0 || position.LastRevision >= 1000000 {
		return ProgramAggregate{}, ErrProgramComplianceCalculationInvalid
	}
	number := position.LastRevision + 1
	definition := input.Definition
	definition.Revision = fmt.Sprintf("program-v%d", number)
	if err := ValidateProgramComplianceCalculationPolicy(definition); err != nil {
		return ProgramAggregate{}, err
	}
	checksum, err := programCompliancePolicyChecksum(definition)
	if err != nil {
		return ProgramAggregate{}, err
	}
	position.LastRevision = number
	position.Pending = &ProgramCompliancePolicyRevision{
		Number: number, Definition: definition, Checksum: checksum,
		MakerID: maker, ProposedAt: s.now().UTC(),
	}
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID,
		input.ExpectedVersion, EventProgramCalculationPolicyProposed,
		programCompliancePolicyChangedEvent{Position: position, Rationale: rationale}, maker, input.ProgramID)
}

func (s *Service) ApproveProgramCompliancePolicy(ctx context.Context, input DecideProgramCompliancePolicyInput) (ProgramAggregate, error) {
	return s.decideProgramCompliancePolicy(ctx, input, true)
}

func (s *Service) RejectProgramCompliancePolicy(ctx context.Context, input DecideProgramCompliancePolicyInput) (ProgramAggregate, error) {
	return s.decideProgramCompliancePolicy(ctx, input, false)
}

func (s *Service) decideProgramCompliancePolicy(ctx context.Context, input DecideProgramCompliancePolicyInput, approve bool) (ProgramAggregate, error) {
	aggregate, err := s.programForMutation(ctx, input.TenantID, input.ProgramID, input.ExpectedVersion)
	if err != nil {
		return ProgramAggregate{}, err
	}
	checker := strings.TrimSpace(input.ActorID)
	rationale := strings.TrimSpace(input.Rationale)
	position := cloneProgramCompliancePolicyPosition(aggregate.Program.CalculationPolicy)
	if !programComplianceActorMatches(ctx, input.TenantID, checker) ||
		rationale == "" || len(rationale) > 2000 ||
		position.Pending == nil || checker == position.Pending.MakerID {
		return ProgramAggregate{}, ErrInvalidState
	}
	eventType := EventProgramCalculationPolicyRejected
	if approve {
		if err := ValidateProgramComplianceCalculationPolicy(position.Pending.Definition); err != nil {
			return ProgramAggregate{}, err
		}
		checksum, err := programCompliancePolicyChecksum(position.Pending.Definition)
		if err != nil {
			return ProgramAggregate{}, err
		}
		if position.Pending.Checksum != checksum {
			return ProgramAggregate{}, ErrProgramComplianceCalculationInvalid
		}
		now := s.now().UTC()
		position.Pending.ApproverID = checker
		position.Pending.ApprovedAt = &now
		position.Active = position.Pending
		eventType = EventProgramCalculationPolicyApproved
	}
	position.Pending = nil
	return s.applyProgramValueAndResult(ctx, aggregate, input.TenantID, input.ProgramID,
		input.ExpectedVersion, eventType,
		programCompliancePolicyChangedEvent{Position: position, Rationale: rationale}, checker, input.ProgramID)
}

// Copy before approval: no material command may mutate the old Program value
// in an in-memory repository when an event fails to commit.
func cloneProgramCompliancePolicyPosition(source *ProgramCompliancePolicyPosition) ProgramCompliancePolicyPosition {
	if source == nil {
		return ProgramCompliancePolicyPosition{}
	}
	value := *source
	copyRevision := func(revision *ProgramCompliancePolicyRevision) *ProgramCompliancePolicyRevision {
		if revision == nil {
			return nil
		}
		result := *revision
		result.Definition.Categories = append([]ProgramComplianceCategoryWeight(nil), revision.Definition.Categories...)
		result.Definition.ExpectedVendorAssessments = append([]ProgramComplianceExpectedVendorAssessment(nil), revision.Definition.ExpectedVendorAssessments...)
		if revision.ApprovedAt != nil {
			at := *revision.ApprovedAt
			result.ApprovedAt = &at
		}
		return &result
	}
	value.Active = copyRevision(source.Active)
	value.Pending = copyRevision(source.Pending)
	return value
}

// Do not trust actor_id supplied by a request body. HTTP commands additionally
// require the current Program owner/authorizer route before invoking the service.
func programComplianceActorMatches(ctx context.Context, tenant, principalID string) bool {
	if principalID == "" {
		return false
	}
	if hasTrustedSystemScope(ctx) {
		return true
	}
	actor, ok := identity.FromContext(ctx)
	return ok && actor.TenantID == strings.TrimSpace(tenant) && actor.PrincipalID == principalID
}

// Retain the same canonical policy fingerprint for approval and projection.
// Calculation never trusts a policy revision from an unchecked source.
func programCompliancePolicyChecksum(policy ProgramComplianceCalculationPolicy) (string, error) {
	encoded, err := json.Marshal(policy)
	if err != nil {
		return "", err
	}
	checksum := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", checksum), nil
}
