package governance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (s *Service) PendingPolicyRevision(ctx context.Context, tenantID, policyID string) (RoutingPolicyRevision, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(policyID) == "" {
		return RoutingPolicyRevision{}, fmt.Errorf("tenant_id and policy_id are required")
	}
	return s.repo.PendingPolicyRevision(ctx, tenantID, policyID)
}

func (s *Service) ProposeEscalationGuardRevision(ctx context.Context, input EscalationGuardRevisionInput) (RoutingPolicyRevision, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.PolicyID = strings.TrimSpace(input.PolicyID)
	input.SequenceID = strings.TrimSpace(input.SequenceID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	if input.TenantID == "" || input.PolicyID == "" || input.SequenceID == "" || input.ActorID == "" || input.StepIndex < 0 || input.ExpectedPolicyVersion < 1 {
		return RoutingPolicyRevision{}, fmt.Errorf("tenant_id, policy_id, sequence_id, actor_id, step_index and expected_policy_version are required")
	}

	policy, err := s.repo.GetPolicy(ctx, input.TenantID, input.PolicyID)
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

	definition, checksum, err := updateEscalationGuardDefinition(
		baseDefinition, input.SequenceID, input.StepIndex,
		input.SourceRoles, input.TargetRoles, input.TargetGroupIDs,
	)
	if err != nil {
		return RoutingPolicyRevision{}, err
	}
	if checksum == baseChecksum {
		return RoutingPolicyRevision{}, fmt.Errorf("%w: escalation guard is unchanged", ErrConflict)
	}
	return s.repo.CreatePolicyRevision(ctx, input.TenantID, input.PolicyID, input.ExpectedPolicyVersion, input.ActorID, definition, checksum, s.now().UTC())
}

func (s *Service) ApprovePolicyRevision(ctx context.Context, input ApprovePolicyRevisionInput) (RoutingPolicy, error) {
	input.TenantID = strings.TrimSpace(input.TenantID)
	input.PolicyID = strings.TrimSpace(input.PolicyID)
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Rationale = strings.TrimSpace(input.Rationale)
	if input.TenantID == "" || input.PolicyID == "" || input.ActorID == "" || input.RevisionVersion < 1 || input.ExpectedPolicyVersion < 1 || input.Rationale == "" {
		return RoutingPolicy{}, fmt.Errorf("tenant_id, policy_id, revision_version, actor_id, expected_policy_version and rationale are required")
	}

	policy, err := s.repo.GetPolicy(ctx, input.TenantID, input.PolicyID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if policy.Status != PolicyActive {
		return RoutingPolicy{}, ErrInvalidTransition
	}
	if policy.Version != input.ExpectedPolicyVersion {
		return RoutingPolicy{}, ErrVersionConflict
	}
	revision, err := s.repo.PendingPolicyRevision(ctx, input.TenantID, input.PolicyID)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if revision.Version != input.RevisionVersion {
		return RoutingPolicy{}, ErrRevisionStale
	}
	if revision.MakerID == "" || revision.MakerID == input.ActorID {
		return RoutingPolicy{}, ErrMakerChecker
	}
	if err := validatePolicyDefinition(revision.Definition); err != nil {
		return RoutingPolicy{}, err
	}
	authorityDefinition, err := authorityOnlyPolicyDefinition(revision.Definition)
	if err != nil {
		return RoutingPolicy{}, err
	}
	authorityPolicy := policy
	authorityPolicy.Definition = authorityDefinition
	findings, err := s.repo.PolicyConflicts(ctx, authorityPolicy)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if len(findings) > 0 {
		return RoutingPolicy{}, fmt.Errorf("%w: %s", ErrConflict, findings[0].Summary)
	}
	findings, err = s.repo.EscalationReferenceConflicts(ctx, input.TenantID, revision.Definition)
	if err != nil {
		return RoutingPolicy{}, err
	}
	if len(findings) > 0 {
		return RoutingPolicy{}, fmt.Errorf("%w: %s", ErrConflict, findings[0].Summary)
	}
	return s.repo.ActivatePolicyRevision(
		ctx, input.TenantID, input.PolicyID, input.ExpectedPolicyVersion,
		input.RevisionVersion, input.ActorID, input.Rationale, s.now().UTC(),
	)
}

func updateEscalationGuardDefinition(value json.RawMessage, sequenceID string, stepIndex int, sourceRoles, targetRoles, targetGroupIDs []string) (json.RawMessage, string, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(value, &document); err != nil {
		return nil, "", fmt.Errorf("decode policy definition: %w", err)
	}
	var sequences []json.RawMessage
	if err := json.Unmarshal(document["escalations"], &sequences); err != nil {
		return nil, "", fmt.Errorf("decode escalation sequences: %w", err)
	}

	found := false
	for sequenceIndex, rawSequence := range sequences {
		var sequence map[string]json.RawMessage
		if err := json.Unmarshal(rawSequence, &sequence); err != nil {
			return nil, "", fmt.Errorf("decode escalation sequence: %w", err)
		}
		var id string
		if err := json.Unmarshal(sequence["id"], &id); err != nil || strings.TrimSpace(id) != sequenceID {
			continue
		}
		var steps []json.RawMessage
		if err := json.Unmarshal(sequence["steps"], &steps); err != nil {
			return nil, "", fmt.Errorf("decode escalation steps: %w", err)
		}
		if stepIndex >= len(steps) {
			return nil, "", fmt.Errorf("escalation step index %d is out of range", stepIndex)
		}
		var step map[string]json.RawMessage
		if err := json.Unmarshal(steps[stepIndex], &step); err != nil {
			return nil, "", fmt.Errorf("decode escalation step: %w", err)
		}

		normalizedSource, err := normalizeEscalationRoles(sourceRoles, maxEscalationRoleTargets)
		if err != nil {
			return nil, "", fmt.Errorf("source roles: %w", err)
		}
		normalizedTargets, err := normalizeEscalationRoles(targetRoles, maxEscalationRoleTargets)
		if err != nil {
			return nil, "", fmt.Errorf("target roles: %w", err)
		}
		normalizedGroups, err := normalizeEscalationGroupIDs(targetGroupIDs)
		if err != nil {
			return nil, "", fmt.Errorf("target groups: %w", err)
		}
		if len(normalizedSource) == 0 {
			delete(step, "source_roles")
		} else {
			step["source_roles"], _ = json.Marshal(normalizedSource)
		}
		if len(normalizedTargets) == 0 && len(normalizedGroups) == 0 {
			delete(step, "targets")
		} else {
			targets := map[string]any{}
			if len(normalizedTargets) > 0 {
				targets["roles"] = normalizedTargets
			}
			if len(normalizedGroups) > 0 {
				targets["groups"] = normalizedGroups
			}
			step["targets"], _ = json.Marshal(targets)
		}
		steps[stepIndex], _ = json.Marshal(step)
		sequence["steps"], _ = json.Marshal(steps)
		sequences[sequenceIndex], _ = json.Marshal(sequence)
		found = true
		break
	}
	if !found {
		return nil, "", fmt.Errorf("escalation sequence %s was not found", sequenceID)
	}
	document["escalations"], _ = json.Marshal(sequences)
	updated, err := json.Marshal(document)
	if err != nil {
		return nil, "", err
	}
	return normalizeDefinition(updated)
}
