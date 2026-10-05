package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *Service) ProposeEscalationSequenceRevision(ctx context.Context, input EscalationSequenceRevisionInput) (RoutingPolicyRevision, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.ToLower(strings.TrimSpace(input.LegalEntityID))
	input.PolicyID = strings.TrimSpace(input.PolicyID)
	input.SequenceID = strings.TrimSpace(input.SequenceID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.LegalEntityID == "*" || input.PolicyID == "" || input.SequenceID == "" || input.ActorID == "" || input.ExpectedPolicyVersion < 1 {
		return RoutingPolicyRevision{}, fmt.Errorf("tenant_id, legal_entity_id, policy_id, sequence_id, actor_id and expected_policy_version are required")
	}
	if len(input.Steps) == 0 || len(input.Steps) > maxEscalationSteps {
		return RoutingPolicyRevision{}, fmt.Errorf("escalation sequence must contain 1-%d levels", maxEscalationSteps)
	}

	policy, err := s.repo.GetPolicyForEntity(ctx, input.TenantID, input.LegalEntityID, input.PolicyID)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if policy.Status != PolicyActive {
		return RoutingPolicyRevision{}, ErrInvalidTransition
	}
	if policy.Version != input.ExpectedPolicyVersion {
		return RoutingPolicyRevision{}, ErrVersionConflict
	}

	baseDefinition := append([]byte(nil), policy.Definition...)
	baseChecksum := policy.Checksum
	pending, pendingErr := s.repo.PendingPolicyRevision(ctx, input.TenantID, input.PolicyID)
	if pendingErr == nil {
		if strings.TrimSpace(pending.MakerID) != input.ActorID {
			return RoutingPolicyRevision{}, fmt.Errorf("%w: another maker already has a pending routing policy revision", ErrConflict)
		}
		baseDefinition = append([]byte(nil), pending.Definition...)
		baseChecksum = pending.Checksum
	} else if !errors.Is(pendingErr, ErrNotFound) {
		return RoutingPolicyRevision{}, pendingErr
	}

	sequence, err := canonicalEscalationSequence(input)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	definition, checksum, err := upsertEscalationSequenceDefinition(baseDefinition, sequence)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if checksum == baseChecksum {
		return RoutingPolicyRevision{}, fmt.Errorf("%w: escalation sequence is unchanged", ErrConflict)
	}
	return s.repo.CreatePolicyRevision(ctx, input.TenantID, input.PolicyID, input.ExpectedPolicyVersion, input.ActorID, definition, checksum, s.now().UTC())
}

func (s *Service) ProposeEscalationRollback(ctx context.Context, input EscalationRollbackInput) (RoutingPolicyRevision, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.LegalEntityID = strings.ToLower(strings.TrimSpace(input.LegalEntityID))
	input.PolicyID = strings.TrimSpace(input.PolicyID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.LegalEntityID == "" || input.LegalEntityID == "*" || input.PolicyID == "" || input.ActorID == "" || input.SourceVersion < 1 || input.ExpectedPolicyVersion < 1 {
		return RoutingPolicyRevision{}, fmt.Errorf("tenant_id, legal_entity_id, policy_id, source_version, actor_id and expected_policy_version are required")
	}
	policy, err := s.repo.GetPolicyForEntity(ctx, input.TenantID, input.LegalEntityID, input.PolicyID)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if policy.Status != PolicyActive {
		return RoutingPolicyRevision{}, ErrInvalidTransition
	}
	if policy.Version != input.ExpectedPolicyVersion {
		return RoutingPolicyRevision{}, ErrVersionConflict
	}
	if input.SourceVersion >= policy.CurrentVersion {
		return RoutingPolicyRevision{}, fmt.Errorf("%w: rollback source must be an earlier approved version", ErrConflict)
	}
	if pending, pendingErr := s.repo.PendingPolicyRevision(ctx, input.TenantID, input.PolicyID); pendingErr == nil {
		if strings.TrimSpace(pending.MakerID) != input.ActorID {
			return RoutingPolicyRevision{}, fmt.Errorf("%w: another maker already has a pending routing policy revision", ErrConflict)
		}
		return RoutingPolicyRevision{}, fmt.Errorf("%w: finish or discard the current pending revision before restoring an earlier version", ErrConflict)
	} else if !errors.Is(pendingErr, ErrNotFound) {
		return RoutingPolicyRevision{}, pendingErr
	}

	source, err := s.repo.GetPolicyVersion(ctx, input.TenantID, input.LegalEntityID, input.PolicyID, input.SourceVersion)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if source.ApprovedAt == nil {
		return RoutingPolicyRevision{}, fmt.Errorf("%w: rollback source must be approved", ErrConflict)
	}
	definition, checksum, err := normalizeDefinition(source.Definition)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if checksum == policy.Checksum {
		return RoutingPolicyRevision{}, fmt.Errorf("%w: selected version matches the active policy", ErrConflict)
	}
	return s.repo.CreatePolicyRevision(ctx, input.TenantID, input.PolicyID, input.ExpectedPolicyVersion, input.ActorID, definition, checksum, s.now().UTC())
}

func canonicalEscalationSequence(input EscalationSequenceRevisionInput) (EscalationSequence, error) {
	steps := make([]map[string]any, 0, len(input.Steps))
	for _, value := range input.Steps {
		step := map[string]any{
			"after":          strings.TrimSpace(value.After),
			"responsibility": strings.ToUpper(strings.TrimSpace(value.Responsibility)),
		}
		if value.DepartmentLevelsUp != nil {
			step["department_levels_up"] = *value.DepartmentLevelsUp
		}
		if len(value.SourceRoles) > 0 {
			step["source_roles"] = value.SourceRoles
		}
		targets := map[string]any{}
		if len(value.TargetRoles) > 0 {
			targets["roles"] = value.TargetRoles
		}
		if len(value.TargetGroupIDs) > 0 {
			targets["groups"] = value.TargetGroupIDs
		}
		if len(value.TargetPositionIDs) > 0 {
			targets["positions"] = value.TargetPositionIDs
		}
		if len(targets) > 0 {
			step["targets"] = targets
		}
		steps = append(steps, step)
	}
	raw, err := json.Marshal(map[string]any{
		"escalations": []map[string]any{{
			"id":                input.SequenceID,
			"trigger":           "OVERDUE",
			"terminal_handling": strings.ToUpper(strings.TrimSpace(input.TerminalHandling)),
			"recovery_action":   strings.ToUpper(strings.TrimSpace(input.RecoveryAction)),
			"steps":             steps,
		}},
	})
	if err != nil {
		return EscalationSequence{}, err
	}
	sequences, err := ParseEscalationSequences(raw)
	if err != nil {
		return EscalationSequence{}, err
	}
	if len(sequences) != 1 {
		return EscalationSequence{}, fmt.Errorf("exactly one OVERDUE escalation sequence is required")
	}
	return sequences[0], nil
}

func upsertEscalationSequenceDefinition(value json.RawMessage, sequence EscalationSequence) (json.RawMessage, string, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(value, &document); err != nil {
		return nil, "", fmt.Errorf("decode policy definition: %w", err)
	}
	var existing []json.RawMessage
	if raw := document["escalations"]; len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &existing); err != nil {
			return nil, "", fmt.Errorf("decode escalation sequences: %w", err)
		}
	}

	encoded, err := encodeEscalationSequence(sequence)
	if err != nil {
		return nil, "", err
	}
	replaced := false
	for index, raw := range existing {
		var header struct {
			ID      string `json:"id"`
			Trigger string `json:"trigger"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return nil, "", fmt.Errorf("decode escalation sequence: %w", err)
		}
		if strings.TrimSpace(header.ID) == sequence.ID || strings.EqualFold(strings.TrimSpace(header.Trigger), sequence.Trigger) {
			existing[index] = encoded
			replaced = true
			break
		}
	}
	if !replaced {
		if len(existing) >= maxEscalationSequences {
			return nil, "", fmt.Errorf("policy supports at most %d escalation sequences", maxEscalationSequences)
		}
		existing = append(existing, encoded)
	}
	document["escalations"], _ = json.Marshal(existing)
	updated, err := json.Marshal(document)
	if err != nil {
		return nil, "", err
	}
	return normalizeDefinition(updated)
}

func encodeEscalationSequence(sequence EscalationSequence) (json.RawMessage, error) {
	steps := make([]map[string]any, 0, len(sequence.Steps))
	for _, value := range sequence.Steps {
		step := map[string]any{
			"after":          value.After.String(),
			"responsibility": value.Responsibility,
		}
		if value.DepartmentLevelsUp != nil {
			step["department_levels_up"] = *value.DepartmentLevelsUp
		}
		if len(value.SourceRoles) > 0 {
			step["source_roles"] = append([]string(nil), value.SourceRoles...)
		}
		targets := map[string]any{}
		if len(value.TargetRoles) > 0 {
			targets["roles"] = append([]string(nil), value.TargetRoles...)
		}
		if len(value.TargetGroupIDs) > 0 {
			targets["groups"] = append([]string(nil), value.TargetGroupIDs...)
		}
		if len(value.TargetPositionIDs) > 0 {
			targets["positions"] = append([]string(nil), value.TargetPositionIDs...)
		}
		if len(targets) > 0 {
			step["targets"] = targets
		}
		steps = append(steps, step)
	}
	return json.Marshal(map[string]any{
		"id":                sequence.ID,
		"trigger":           sequence.Trigger,
		"terminal_handling": sequence.TerminalHandling,
		"recovery_action":   sequence.RecoveryAction,
		"steps":             steps,
	})
}

func escalationNextDueAt(baseline time.Time, step EscalationStep) time.Time {
	return baseline.UTC().Add(step.After)
}
