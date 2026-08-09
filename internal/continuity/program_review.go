package continuity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/platform/id"
)

const programReviewItemLimit = 8

type ProgramReviewCheckpoint struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	ProgramID         string    `json:"program_id"`
	PrincipalID       string    `json:"principal_id"`
	ProgramVersion    int64     `json:"program_version"`
	ProjectionVersion int64     `json:"projection_version"`
	AcceptedAt        time.Time `json:"accepted_at"`
}

type ProgramReviewChange struct {
	Kind       string `json:"kind"`
	Summary    string `json:"summary"`
	ObjectType string `json:"object_type,omitempty"`
	ObjectID   string `json:"object_id,omitempty"`
}

type ProgramReviewDigest struct {
	ProgramID                string                   `json:"program_id"`
	State                    string                   `json:"state"`
	ReviewRequired           bool                     `json:"review_required"`
	Checkpoint               *ProgramReviewCheckpoint `json:"checkpoint,omitempty"`
	CurrentProgramVersion    int64                    `json:"current_program_version"`
	CurrentProjectionVersion int64                    `json:"current_projection_version"`
	CurrentOverall           ProgramState             `json:"current_overall"`
	BaselineOverall          ProgramState             `json:"baseline_overall,omitempty"`
	OpenMatterCount          int                      `json:"open_matter_count"`
	OpenMatterDelta          int                      `json:"open_matter_delta,omitempty"`
	Changes                  []ProgramReviewChange    `json:"changes"`
	ChangesTotal             int                      `json:"changes_total"`
	ChangesOmitted           int                      `json:"changes_omitted"`
	CurrentExceptions        []StateReason            `json:"current_exceptions"`
	CurrentExceptionsTotal   int                      `json:"current_exceptions_total"`
	NewExceptions            []StateReason            `json:"new_exceptions"`
	NewExceptionsTotal       int                      `json:"new_exceptions_total"`
	ResolvedExceptions       []StateReason            `json:"resolved_exceptions"`
	ResolvedExceptionsTotal  int                      `json:"resolved_exceptions_total"`
}

type AcceptProgramReviewInput struct {
	TenantID                  string `json:"-"`
	ProgramID                 string `json:"-"`
	PrincipalID               string `json:"-"`
	ExpectedProgramVersion    int64  `json:"expected_program_version"`
	ExpectedProjectionVersion int64  `json:"expected_projection_version"`
}

type ProgramReviewRepository interface {
	LatestProgramReview(ctx context.Context, tenant, programID, principalID string) (*ProgramReviewCheckpoint, error)
	RecordProgramReview(ctx context.Context, checkpoint ProgramReviewCheckpoint) (ProgramReviewCheckpoint, error)
	ProgramStateVersion(ctx context.Context, tenant, programID string, projectionVersion int64) (*ProgramStateSnapshot, error)
}

func (s *Service) ProgramReviewDigest(ctx context.Context, tenant, programID, principalID string) (ProgramReviewDigest, error) {
	tenant = strings.TrimSpace(tenant)
	programID = strings.TrimSpace(programID)
	principalID = strings.TrimSpace(principalID)
	if tenant == "" || programID == "" || principalID == "" {
		return ProgramReviewDigest{}, fmt.Errorf("tenant_id, program_id and principal_id are required")
	}
	repo, ok := s.repo.(ProgramReviewRepository)
	if !ok {
		return ProgramReviewDigest{}, fmt.Errorf("Program review checkpoints are unavailable")
	}
	aggregate, err := s.GetProgram(ctx, tenant, programID)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	current, err := currentReviewState(aggregate)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	checkpoint, err := repo.LatestProgramReview(ctx, tenant, programID, principalID)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	digest := ProgramReviewDigest{
		ProgramID:                programID,
		State:                    "NO_BASELINE",
		ReviewRequired:           true,
		Checkpoint:               checkpoint,
		CurrentProgramVersion:    aggregate.Program.Version,
		CurrentProjectionVersion: current.ProjectionVersion,
		CurrentOverall:           current.Overall,
		OpenMatterCount:          current.OpenMatterCount,
		CurrentExceptions:        limitReasons(current.Reasons, programReviewItemLimit),
		CurrentExceptionsTotal:   len(current.Reasons),
		Changes:                  []ProgramReviewChange{},
		NewExceptions:            []StateReason{},
		ResolvedExceptions:       []StateReason{},
	}
	if checkpoint == nil {
		return digest, nil
	}
	baseline, err := repo.ProgramStateVersion(ctx, tenant, programID, checkpoint.ProjectionVersion)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	if baseline == nil || baseline.ProgramVersion != checkpoint.ProgramVersion {
		return ProgramReviewDigest{}, fmt.Errorf("%w: accepted Program review baseline is unavailable", ErrInvalidState)
	}
	digest.BaselineOverall = baseline.Overall
	digest.OpenMatterDelta = current.OpenMatterCount - baseline.OpenMatterCount
	newReasons, resolvedReasons := diffStateReasons(baseline.Reasons, current.Reasons)
	digest.NewExceptionsTotal = len(newReasons)
	digest.NewExceptions = limitReasons(newReasons, programReviewItemLimit)
	digest.ResolvedExceptionsTotal = len(resolvedReasons)
	digest.ResolvedExceptions = limitReasons(resolvedReasons, programReviewItemLimit)

	events, err := s.repo.ProgramEvents(ctx, tenant, programID, nil)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	changes := deriveProgramReviewChanges(aggregate, *baseline, *current, events, checkpoint.ProgramVersion, newReasons)
	digest.ChangesTotal = len(changes)
	if len(changes) > programReviewItemLimit {
		digest.ChangesOmitted = len(changes) - programReviewItemLimit
		changes = changes[:programReviewItemLimit]
	}
	digest.Changes = changes
	digest.ReviewRequired = checkpoint.ProgramVersion != aggregate.Program.Version || checkpoint.ProjectionVersion != current.ProjectionVersion || len(changes) > 0
	if digest.ReviewRequired {
		digest.State = "CHANGED"
	} else {
		digest.State = "CURRENT"
	}
	return digest, nil
}

func (s *Service) AcceptProgramReview(ctx context.Context, input AcceptProgramReviewInput) (ProgramReviewDigest, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.ProgramID = strings.TrimSpace(input.ProgramID)
	input.PrincipalID = strings.TrimSpace(input.PrincipalID)
	if input.TenantID == "" || input.ProgramID == "" || input.PrincipalID == "" || input.ExpectedProgramVersion < 1 || input.ExpectedProjectionVersion < 1 {
		return ProgramReviewDigest{}, fmt.Errorf("tenant_id, program_id, principal_id and positive expected versions are required")
	}
	repo, ok := s.repo.(ProgramReviewRepository)
	if !ok {
		return ProgramReviewDigest{}, fmt.Errorf("Program review checkpoints are unavailable")
	}
	aggregate, err := s.GetProgram(ctx, input.TenantID, input.ProgramID)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	current, err := currentReviewState(aggregate)
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	if aggregate.Program.Version != input.ExpectedProgramVersion || current.ProjectionVersion != input.ExpectedProjectionVersion {
		return ProgramReviewDigest{}, ErrVersionConflict
	}
	checkpointID, err := id.NewUUIDv7()
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	_, err = repo.RecordProgramReview(ctx, ProgramReviewCheckpoint{
		ID:                checkpointID,
		TenantID:          input.TenantID,
		ProgramID:         input.ProgramID,
		PrincipalID:       input.PrincipalID,
		ProgramVersion:    input.ExpectedProgramVersion,
		ProjectionVersion: input.ExpectedProjectionVersion,
		AcceptedAt:        s.now().UTC(),
	})
	if err != nil {
		return ProgramReviewDigest{}, err
	}
	return s.ProgramReviewDigest(ctx, input.TenantID, input.ProgramID, input.PrincipalID)
}

func currentReviewState(aggregate ProgramAggregate) (*ProgramStateSnapshot, error) {
	if aggregate.CurrentState == nil || aggregate.CurrentState.ProjectionVersion < 1 {
		return nil, fmt.Errorf("%w: current Program status has not been projected yet", ErrInvalidState)
	}
	if aggregate.CurrentState.ProgramVersion != aggregate.Program.Version {
		return nil, ErrVersionConflict
	}
	return aggregate.CurrentState, nil
}

func deriveProgramReviewChanges(aggregate ProgramAggregate, baseline, current ProgramStateSnapshot, events []Event, baselineProgramVersion int64, newReasons []StateReason) []ProgramReviewChange {
	changes := make([]ProgramReviewChange, 0, 16)
	seen := make(map[string]struct{})
	appendChange := func(change ProgramReviewChange) {
		key := change.Kind + "\x00" + change.Summary + "\x00" + change.ObjectType + "\x00" + change.ObjectID
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		changes = append(changes, change)
	}
	if baseline.Overall != current.Overall {
		appendChange(ProgramReviewChange{Kind: "STATE", Summary: fmt.Sprintf("Overall status changed from %s to %s.", humanProgramState(baseline.Overall), humanProgramState(current.Overall))})
	}
	for _, change := range dimensionChanges(baseline.Dimensions, current.Dimensions) {
		appendChange(change)
	}
	if baseline.OpenMatterCount != current.OpenMatterCount {
		delta := current.OpenMatterCount - baseline.OpenMatterCount
		if delta > 0 {
			appendChange(ProgramReviewChange{Kind: "ISSUE", Summary: fmt.Sprintf("%d additional open issue(s) now affect this Program.", delta)})
		} else {
			appendChange(ProgramReviewChange{Kind: "ISSUE", Summary: fmt.Sprintf("%d open issue(s) affecting this Program were resolved or removed.", -delta)})
		}
	}
	for _, reason := range newReasons {
		appendChange(ProgramReviewChange{Kind: "EXCEPTION", Summary: reason.Summary, ObjectType: reason.ObjectType, ObjectID: reason.ObjectID})
	}
	for _, event := range events {
		if event.AggregateVersion <= baselineProgramVersion {
			continue
		}
		if change, ok := programEventChange(aggregate, event); ok {
			appendChange(change)
		}
	}
	return changes
}

func programEventChange(aggregate ProgramAggregate, event Event) (ProgramReviewChange, bool) {
	switch event.Type {
	case EventProgramStatusChanged:
		var value Program
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "PROGRAM", Summary: "Operating status changed to " + humanProgramState(ProgramState(value.Status)) + ".", ObjectType: "PROGRAM", ObjectID: value.ID}, true
		}
	case EventRequirementAdded:
		var value Requirement
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "REQUIREMENT", Summary: "Requirement changed: " + value.Title + ".", ObjectType: "REQUIREMENT", ObjectID: value.ID}, true
		}
	case EventApplicabilityDetermined:
		var value Applicability
		if json.Unmarshal(event.Payload, &value) == nil {
			label := requirementTitle(aggregate.Requirements, value.RequirementID)
			return ProgramReviewChange{Kind: "APPLICABILITY", Summary: fmt.Sprintf("Applicability for %s is now %s.", label, humanToken(string(value.Status))), ObjectType: "REQUIREMENT", ObjectID: value.RequirementID}, true
		}
	case EventControlObjectiveAdded:
		var value ControlObjective
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "SAFEGUARD", Summary: "Control objective changed: " + value.Name + ".", ObjectType: "CONTROL_OBJECTIVE", ObjectID: value.ID}, true
		}
	case EventControlImplementationAdded:
		var value ControlImplementation
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "SAFEGUARD", Summary: "Safeguard changed: " + value.Name + ".", ObjectType: "CONTROL_IMPLEMENTATION", ObjectID: value.ID}, true
		}
	case EventRequirementControlLinked:
		var value RequirementControlLink
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "MAPPING", Summary: "A requirement-to-safeguard mapping changed.", ObjectType: "REQUIREMENT", ObjectID: value.RequirementID}, true
		}
	case EventEvidenceContractAdded:
		var value EvidenceContract
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "EVIDENCE", Summary: "Evidence check changed: " + value.Name + ".", ObjectType: "EVIDENCE_CONTRACT", ObjectID: value.ID}, true
		}
	case EventEvidenceAssessmentRecorded:
		var value EvidenceAssessment
		if json.Unmarshal(event.Payload, &value) == nil {
			label := evidenceContractName(aggregate.EvidenceContracts, value.ContractID)
			return ProgramReviewChange{Kind: "EVIDENCE", Summary: fmt.Sprintf("Evidence for %s was assessed as %s.", label, humanToken(string(value.Conclusion))), ObjectType: "EVIDENCE_CONTRACT", ObjectID: value.ContractID}, true
		}
	case EventProgramTriggerRecorded:
		var value Trigger
		if json.Unmarshal(event.Payload, &value) == nil {
			return ProgramReviewChange{Kind: "CHANGE", Summary: "Observed change: " + humanToken(value.Type) + ".", ObjectType: value.SubjectType, ObjectID: value.SubjectID}, true
		}
	}
	return ProgramReviewChange{}, false
}

func dimensionChanges(before, after ComplianceDimensions) []ProgramReviewChange {
	values := []struct {
		name string
		old  ProgramState
		new  ProgramState
	}{
		{"Interpretation", before.Interpretation, after.Interpretation},
		{"Applicability", before.Applicability, after.Applicability},
		{"Control design", before.ControlDesign, after.ControlDesign},
		{"Implementation", before.Implementation, after.Implementation},
		{"Evidence sufficiency", before.EvidenceSufficiency, after.EvidenceSufficiency},
		{"Operating effectiveness", before.OperatingEffectiveness, after.OperatingEffectiveness},
		{"Exception", before.Exception, after.Exception},
		{"Assurance", before.Assurance, after.Assurance},
		{"Deadline", before.Deadline, after.Deadline},
		{"Source quality", before.SourceQuality, after.SourceQuality},
	}
	result := make([]ProgramReviewChange, 0, 4)
	for _, value := range values {
		if value.old == value.new {
			continue
		}
		result = append(result, ProgramReviewChange{Kind: "STATE", Summary: fmt.Sprintf("%s changed from %s to %s.", value.name, humanProgramState(value.old), humanProgramState(value.new))})
	}
	return result
}

func diffStateReasons(before, after []StateReason) ([]StateReason, []StateReason) {
	beforeByKey := make(map[string]StateReason, len(before))
	afterByKey := make(map[string]StateReason, len(after))
	for _, reason := range before {
		beforeByKey[stateReasonKey(reason)] = reason
	}
	for _, reason := range after {
		afterByKey[stateReasonKey(reason)] = reason
	}
	added := make([]StateReason, 0)
	resolved := make([]StateReason, 0)
	for key, reason := range afterByKey {
		if _, exists := beforeByKey[key]; !exists {
			added = append(added, reason)
		}
	}
	for key, reason := range beforeByKey {
		if _, exists := afterByKey[key]; !exists {
			resolved = append(resolved, reason)
		}
	}
	sortStateReasons(added)
	sortStateReasons(resolved)
	return added, resolved
}

func sortStateReasons(values []StateReason) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && stateReasonKey(values[j]) < stateReasonKey(values[j-1]); j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

func stateReasonKey(reason StateReason) string {
	return strings.ToUpper(strings.TrimSpace(reason.Code)) + "\x00" + strings.ToUpper(strings.TrimSpace(reason.ObjectType)) + "\x00" + strings.TrimSpace(reason.ObjectID)
}

func limitReasons(values []StateReason, limit int) []StateReason {
	if len(values) <= limit {
		return append([]StateReason(nil), values...)
	}
	return append([]StateReason(nil), values[:limit]...)
}

func requirementTitle(values []Requirement, id string) string {
	for _, value := range values {
		if value.ID == id && strings.TrimSpace(value.Title) != "" {
			return value.Title
		}
	}
	return "the requirement"
}

func evidenceContractName(values []EvidenceContract, id string) string {
	for _, value := range values {
		if value.ID == id && strings.TrimSpace(value.Name) != "" {
			return value.Name
		}
	}
	return "the evidence check"
}

func humanProgramState(value ProgramState) string {
	return strings.ToLower(strings.ReplaceAll(string(value), "_", " "))
}

func humanToken(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", " "))
}
