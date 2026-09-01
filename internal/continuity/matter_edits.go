package continuity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type MatterContextChangeKind string

const (
	MatterContextAddFact              MatterContextChangeKind = "ADD_FACT"
	MatterContextCorrectFact          MatterContextChangeKind = "CORRECT_FACT"
	MatterContextRetireFact           MatterContextChangeKind = "RETIRE_FACT"
	MatterContextAddMissing           MatterContextChangeKind = "ADD_MISSING"
	MatterContextResolveMissing       MatterContextChangeKind = "RESOLVE_MISSING"
	MatterContextAddContradiction     MatterContextChangeKind = "ADD_CONTRADICTION"
	MatterContextResolveContradiction MatterContextChangeKind = "RESOLVE_CONTRADICTION"
)

type UpdateMatterDetailsInput struct {
	TenantID        string          `json:"tenant_id"`
	MatterID        string          `json:"matter_id"`
	ExpectedVersion int64           `json:"expected_version"`
	Title           string          `json:"title"`
	Summary         string          `json:"summary"`
	Priority        int             `json:"priority"`
	DueAt           *time.Time      `json:"due_at,omitempty"`
	Scope           json.RawMessage `json:"scope"`
	ActorID         string          `json:"actor_id,omitempty"`
	Rationale       string          `json:"rationale"`
}

type ChangeMatterContextInput struct {
	TenantID           string                  `json:"tenant_id"`
	MatterID           string                  `json:"matter_id"`
	ExpectedVersion    int64                   `json:"expected_version"`
	Kind               MatterContextChangeKind `json:"kind"`
	Key                string                  `json:"key,omitempty"`
	Label              string                  `json:"label,omitempty"`
	Value              json.RawMessage         `json:"value,omitempty"`
	Rationale          string                  `json:"rationale"`
	EvidenceReferences json.RawMessage         `json:"evidence_references"`
	ActorID            string                  `json:"actor_id,omitempty"`
}

type AssignMatterInput struct {
	TenantID                    string `json:"tenant_id"`
	MatterID                    string `json:"matter_id"`
	ExpectedVersion             int64  `json:"expected_version"`
	OwnerPrincipalID            string `json:"owner_principal_id"`
	ActorID                     string `json:"actor_id,omitempty"`
	Rationale                   string `json:"rationale"`
	ReassignmentBasis           string `json:"reassignment_basis,omitempty"`
	OrganizationPositionVersion int64  `json:"organization_position_version,omitempty"`
}

type UpdateActionInput struct {
	TenantID        string     `json:"tenant_id"`
	MatterID        string     `json:"matter_id"`
	ActionID        string     `json:"action_id"`
	ExpectedVersion int64      `json:"expected_version"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DueAt           *time.Time `json:"due_at,omitempty"`
	ActorID         string     `json:"actor_id,omitempty"`
	Rationale       string     `json:"rationale"`
}

type AssignActionInput struct {
	TenantID                    string `json:"tenant_id"`
	MatterID                    string `json:"matter_id"`
	ActionID                    string `json:"action_id"`
	ExpectedVersion             int64  `json:"expected_version"`
	OwnerPrincipalID            string `json:"owner_principal_id"`
	ActorID                     string `json:"actor_id,omitempty"`
	Rationale                   string `json:"rationale"`
	ReassignmentBasis           string `json:"reassignment_basis,omitempty"`
	OrganizationPositionVersion int64  `json:"organization_position_version,omitempty"`
}

type matterDetailsUpdatedEvent struct {
	Matter    Matter `json:"matter"`
	Previous  Matter `json:"previous"`
	Rationale string `json:"rationale"`
}

type matterContextChangedEvent struct {
	Matter             Matter                  `json:"matter"`
	Kind               MatterContextChangeKind `json:"kind"`
	Key                string                  `json:"key,omitempty"`
	Label              string                  `json:"label,omitempty"`
	PreviousValue      json.RawMessage         `json:"previous_value,omitempty"`
	Value              json.RawMessage         `json:"value,omitempty"`
	Rationale          string                  `json:"rationale"`
	EvidenceReferences []string                `json:"evidence_references"`
}

type matterOwnerChangedEvent struct {
	Matter                      Matter `json:"matter"`
	PreviousOwnerID             string `json:"previous_owner_principal_id,omitempty"`
	OwnerPrincipalID            string `json:"owner_principal_id"`
	Rationale                   string `json:"rationale"`
	ReassignmentBasis           string `json:"reassignment_basis,omitempty"`
	OrganizationPositionVersion int64  `json:"organization_position_version,omitempty"`
}

type actionUpdatedEvent struct {
	Action    Action `json:"action"`
	Previous  Action `json:"previous"`
	Rationale string `json:"rationale"`
}

type actionAssignedEvent struct {
	Action                      Action `json:"action"`
	PreviousOwnerID             string `json:"previous_owner_principal_id,omitempty"`
	OwnerPrincipalID            string `json:"owner_principal_id"`
	Rationale                   string `json:"rationale"`
	ReassignmentBasis           string `json:"reassignment_basis,omitempty"`
	OrganizationPositionVersion int64  `json:"organization_position_version,omitempty"`
}

func (s *Service) UpdateMatterDetails(ctx context.Context, input UpdateMatterDetailsInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	if strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.Summary) == "" {
		return MatterAggregate{}, fmt.Errorf("title and summary are required")
	}
	if input.Priority < 1 || input.Priority > 5 {
		return MatterAggregate{}, fmt.Errorf("priority must be between 1 and 5")
	}
	scope, err := normalizedJSONObject(input.Scope, `{}`)
	if err != nil {
		return MatterAggregate{}, fmt.Errorf("scope: %w", err)
	}

	previous := aggregate.Matter
	matter := previous
	matter.Title = strings.TrimSpace(input.Title)
	matter.Summary = strings.TrimSpace(input.Summary)
	matter.Priority = input.Priority
	matter.DueAt = input.DueAt
	matter.Scope = scope
	if matter.Title == previous.Title && matter.Summary == previous.Summary && matter.Priority == previous.Priority && bytes.Equal(matter.Scope, previous.Scope) && sameOptionalTime(matter.DueAt, previous.DueAt) {
		return MatterAggregate{}, fmt.Errorf("matter details are unchanged")
	}
	matter.UpdatedAt = s.now().UTC()
	value := matterDetailsUpdatedEvent{Matter: matter, Previous: previous, Rationale: strings.TrimSpace(input.Rationale)}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventMatterDetailsUpdated, value, input.ActorID)
}

func (s *Service) ChangeMatterContext(ctx context.Context, input ChangeMatterContextInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	key := strings.TrimSpace(input.Key)
	label := strings.TrimSpace(input.Label)
	facts, err := decodeRawObject(aggregate.Matter.KnownFacts)
	if err != nil {
		return MatterAggregate{}, fmt.Errorf("known_facts: %w", err)
	}
	missing, err := decodeStringList(aggregate.Matter.MissingFacts)
	if err != nil {
		return MatterAggregate{}, fmt.Errorf("missing_facts: %w", err)
	}
	contradictions, err := decodeStringList(aggregate.Matter.Contradictions)
	if err != nil {
		return MatterAggregate{}, fmt.Errorf("contradictions: %w", err)
	}
	references, err := decodeStringListWithFallback(input.EvidenceReferences)
	if err != nil {
		return MatterAggregate{}, fmt.Errorf("evidence_references: %w", err)
	}

	var previousValue json.RawMessage
	var value json.RawMessage
	switch input.Kind {
	case MatterContextAddFact:
		if key == "" || label == "" {
			return MatterAggregate{}, fmt.Errorf("key and label are required to add a fact")
		}
		if _, exists := facts[key]; exists {
			return MatterAggregate{}, fmt.Errorf("fact %q already exists; correct it explicitly", key)
		}
		value, err = normalizedScalarJSON(input.Value)
		if err != nil {
			return MatterAggregate{}, fmt.Errorf("value: %w", err)
		}
		facts[key] = value
	case MatterContextCorrectFact:
		if key == "" || label == "" {
			return MatterAggregate{}, fmt.Errorf("key and label are required to correct a fact")
		}
		var exists bool
		previousValue, exists = facts[key]
		if !exists {
			return MatterAggregate{}, fmt.Errorf("fact %q was not found", key)
		}
		value, err = normalizedScalarJSON(input.Value)
		if err != nil {
			return MatterAggregate{}, fmt.Errorf("value: %w", err)
		}
		if bytes.Equal(previousValue, value) {
			return MatterAggregate{}, fmt.Errorf("fact %q already has that value", key)
		}
		facts[key] = value
	case MatterContextRetireFact:
		if key == "" || label == "" {
			return MatterAggregate{}, fmt.Errorf("key and label are required to retire a fact")
		}
		var exists bool
		previousValue, exists = facts[key]
		if !exists {
			return MatterAggregate{}, fmt.Errorf("fact %q was not found", key)
		}
		delete(facts, key)
	case MatterContextAddMissing:
		if label == "" {
			return MatterAggregate{}, fmt.Errorf("label is required to add missing information")
		}
		if containsLabel(missing, label) {
			return MatterAggregate{}, fmt.Errorf("missing information %q is already recorded", label)
		}
		missing = append(missing, label)
	case MatterContextResolveMissing:
		if key == "" || label == "" {
			return MatterAggregate{}, fmt.Errorf("key and label are required to resolve missing information")
		}
		if _, exists := facts[key]; exists {
			return MatterAggregate{}, fmt.Errorf("fact %q already exists; correct it explicitly", key)
		}
		var found bool
		missing, found = removeLabel(missing, label)
		if !found {
			return MatterAggregate{}, fmt.Errorf("missing information %q was not found", label)
		}
		value, err = normalizedScalarJSON(input.Value)
		if err != nil {
			return MatterAggregate{}, fmt.Errorf("value: %w", err)
		}
		facts[key] = value
	case MatterContextAddContradiction:
		if label == "" {
			return MatterAggregate{}, fmt.Errorf("label is required to add a contradiction")
		}
		if containsLabel(contradictions, label) {
			return MatterAggregate{}, fmt.Errorf("contradiction %q is already recorded", label)
		}
		contradictions = append(contradictions, label)
	case MatterContextResolveContradiction:
		if label == "" {
			return MatterAggregate{}, fmt.Errorf("label is required to resolve a contradiction")
		}
		var found bool
		contradictions, found = removeLabel(contradictions, label)
		if !found {
			return MatterAggregate{}, fmt.Errorf("contradiction %q was not found", label)
		}
	default:
		return MatterAggregate{}, fmt.Errorf("unsupported matter context change kind %q", input.Kind)
	}

	knownFacts, err := json.Marshal(facts)
	if err != nil {
		return MatterAggregate{}, err
	}
	missingFacts, err := json.Marshal(missing)
	if err != nil {
		return MatterAggregate{}, err
	}
	contradictionValues, err := json.Marshal(contradictions)
	if err != nil {
		return MatterAggregate{}, err
	}

	matter := aggregate.Matter
	matter.KnownFacts = knownFacts
	matter.MissingFacts = missingFacts
	matter.Contradictions = contradictionValues
	matter.UpdatedAt = s.now().UTC()
	eventValue := matterContextChangedEvent{
		Matter:             matter,
		Kind:               input.Kind,
		Key:                key,
		Label:              label,
		PreviousValue:      previousValue,
		Value:              value,
		Rationale:          strings.TrimSpace(input.Rationale),
		EvidenceReferences: references,
	}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventMatterContextChanged, eventValue, input.ActorID)
}

func (s *Service) AssignMatter(ctx context.Context, input AssignMatterInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	ownerID := strings.TrimSpace(input.OwnerPrincipalID)
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" || ownerID == "" {
		return MatterAggregate{}, fmt.Errorf("owner_principal_id, actor_id and rationale are required")
	}
	if aggregate.Matter.OwnerPrincipalID == ownerID {
		return MatterAggregate{}, fmt.Errorf("matter is already assigned to that owner")
	}
	matter := aggregate.Matter
	matter.OwnerPrincipalID = ownerID
	matter.UpdatedAt = s.now().UTC()
	value := matterOwnerChangedEvent{
		Matter:                      matter,
		PreviousOwnerID:             aggregate.Matter.OwnerPrincipalID,
		OwnerPrincipalID:            ownerID,
		Rationale:                   strings.TrimSpace(input.Rationale),
		ReassignmentBasis:           strings.TrimSpace(input.ReassignmentBasis),
		OrganizationPositionVersion: input.OrganizationPositionVersion,
	}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventMatterOwnerChanged, value, input.ActorID)
}

func (s *Service) UpdateAction(ctx context.Context, input UpdateActionInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" {
		return MatterAggregate{}, fmt.Errorf("actor_id and rationale are required")
	}
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	if title == "" || description == "" {
		return MatterAggregate{}, fmt.Errorf("title and description are required")
	}
	previous, err := findAction(aggregate.Actions, input.ActionID)
	if err != nil {
		return MatterAggregate{}, err
	}
	if previous.Status == ActionImplemented || previous.Status == ActionCancelled {
		return MatterAggregate{}, ErrInvalidState
	}
	if previous.Title == title && previous.Description == description && sameOptionalTime(previous.DueAt, input.DueAt) {
		return MatterAggregate{}, fmt.Errorf("action details are unchanged")
	}
	action := previous
	action.Title = title
	action.Description = description
	action.DueAt = input.DueAt
	action.UpdatedAt = s.now().UTC()
	action.Version++
	value := actionUpdatedEvent{Action: action, Previous: previous, Rationale: strings.TrimSpace(input.Rationale)}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventActionUpdated, value, input.ActorID)
}

func (s *Service) AssignAction(ctx context.Context, input AssignActionInput) (MatterAggregate, error) {
	aggregate, err := s.matterForMutation(ctx, input.TenantID, input.MatterID, input.ExpectedVersion)
	if err != nil {
		return MatterAggregate{}, err
	}
	ownerID := strings.TrimSpace(input.OwnerPrincipalID)
	if strings.TrimSpace(input.ActorID) == "" || strings.TrimSpace(input.Rationale) == "" || ownerID == "" {
		return MatterAggregate{}, fmt.Errorf("owner_principal_id, actor_id and rationale are required")
	}
	previous, err := findAction(aggregate.Actions, input.ActionID)
	if err != nil {
		return MatterAggregate{}, err
	}
	if previous.Status == ActionImplemented || previous.Status == ActionCancelled {
		return MatterAggregate{}, ErrInvalidState
	}
	if previous.OwnerPrincipalID == ownerID {
		return MatterAggregate{}, fmt.Errorf("action is already assigned to that owner")
	}
	action := previous
	action.OwnerPrincipalID = ownerID
	action.UpdatedAt = s.now().UTC()
	action.Version++
	value := actionAssignedEvent{
		Action:                      action,
		PreviousOwnerID:             previous.OwnerPrincipalID,
		OwnerPrincipalID:            ownerID,
		Rationale:                   strings.TrimSpace(input.Rationale),
		ReassignmentBasis:           strings.TrimSpace(input.ReassignmentBasis),
		OrganizationPositionVersion: input.OrganizationPositionVersion,
	}
	return s.applyMatterValueAndResult(ctx, aggregate, input.TenantID, input.MatterID, input.ExpectedVersion, EventActionAssigned, value, input.ActorID)
}

func normalizedJSONObject(value json.RawMessage, fallback string) (json.RawMessage, error) {
	value, err := normalizedJSON(value, fallback)
	if err != nil {
		return nil, err
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(value, &object); err != nil || object == nil {
		return nil, fmt.Errorf("value must be a JSON object")
	}
	return value, nil
}

func decodeRawObject(value json.RawMessage) (map[string]json.RawMessage, error) {
	value, err := normalizedJSONObject(value, `{}`)
	if err != nil {
		return nil, err
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(value, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func decodeStringList(value json.RawMessage) ([]string, error) {
	var result []string
	if err := json.Unmarshal(value, &result); err != nil {
		return nil, fmt.Errorf("value must be an array of strings")
	}
	if result == nil {
		result = []string{}
	}
	for index := range result {
		result[index] = strings.TrimSpace(result[index])
		if result[index] == "" {
			return nil, fmt.Errorf("list entries must not be empty")
		}
	}
	return result, nil
}

func decodeStringListWithFallback(value json.RawMessage) ([]string, error) {
	if len(bytes.TrimSpace(value)) == 0 {
		return []string{}, nil
	}
	return decodeStringList(value)
}

func normalizedScalarJSON(value json.RawMessage) (json.RawMessage, error) {
	value, err := normalizedJSON(value, "null")
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil, err
	}
	switch decoded.(type) {
	case string, float64, bool:
		return value, nil
	default:
		return nil, fmt.Errorf("value must be text, a number or true/false")
	}
}

func removeLabel(values []string, label string) ([]string, bool) {
	result := make([]string, 0, len(values))
	found := false
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), label) {
			found = true
			continue
		}
		result = append(result, value)
	}
	return result, found
}

func containsLabel(values []string, label string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), label) {
			return true
		}
	}
	return false
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func findAction(actions []Action, actionID string) (Action, error) {
	actionID = strings.TrimSpace(actionID)
	if actionID == "" {
		return Action{}, fmt.Errorf("action_id is required")
	}
	for _, action := range actions {
		if action.ID == actionID {
			return action, nil
		}
	}
	return Action{}, ErrNotFound
}
