package governance

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/identity"
)

const (
	maxEscalationSequences    = 16
	maxEscalationSteps        = 8
	maxDepartmentLevelsUp     = 8
	maxEscalationRoleTargets  = 8
	maxEscalationGroupTargets = 8
)

type EscalationSequence struct {
	ID      string
	Trigger string
	Steps   []EscalationStep
}

type EscalationStep struct {
	After              time.Duration
	Responsibility     string
	DepartmentLevelsUp *int
	SourceRoles         []string
	TargetRoles         []string
	TargetGroupIDs      []string
}

type escalationEnvelope struct {
	Escalations []json.RawMessage `json:"escalations"`
}

type escalationSequenceDefinition struct {
	ID      string                     `json:"id"`
	Trigger string                     `json:"trigger"`
	Steps   []escalationStepDefinition `json:"steps"`
}

type escalationStepDefinition struct {
	After              string                      `json:"after"`
	Responsibility     string                      `json:"responsibility"`
	DepartmentLevelsUp *int                        `json:"department_levels_up,omitempty"`
	SourceRoles        []string                    `json:"source_roles,omitempty"`
	Targets            *escalationTargetDefinition `json:"targets,omitempty"`
}

type escalationTargetDefinition struct {
	Roles  []string `json:"roles,omitempty"`
	Groups []string `json:"groups,omitempty"`
}

func ParseEscalationSequences(definition json.RawMessage) ([]EscalationSequence, error) {
	var envelope escalationEnvelope
	if err := json.Unmarshal(definition, &envelope); err != nil {
		return nil, fmt.Errorf("decode escalation policy: %w", err)
	}
	if len(envelope.Escalations) == 0 {
		return nil, nil
	}
	if len(envelope.Escalations) > maxEscalationSequences {
		return nil, fmt.Errorf("policy supports at most %d escalation sequences", maxEscalationSequences)
	}

	seenIDs := make(map[string]struct{}, len(envelope.Escalations))
	seenTriggers := make(map[string]string, len(envelope.Escalations))
	result := make([]EscalationSequence, 0, len(envelope.Escalations))
	for _, raw := range envelope.Escalations {
		var value escalationSequenceDefinition
		if err := decodeStrictJSON(raw, &value); err != nil {
			return nil, fmt.Errorf("decode escalation sequence: %w", err)
		}
		id := strings.TrimSpace(value.ID)
		trigger := strings.ToUpper(strings.TrimSpace(value.Trigger))
		if id == "" || !supportedEscalationTrigger(trigger) {
			return nil, fmt.Errorf("each escalation sequence requires a unique id and supported trigger")
		}
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("duplicate escalation sequence id %s", id)
		}
		seenIDs[id] = struct{}{}
		if prior, exists := seenTriggers[trigger]; exists {
			return nil, fmt.Errorf("escalation trigger %s is already defined by sequence %s", trigger, prior)
		}
		seenTriggers[trigger] = id
		if len(value.Steps) == 0 || len(value.Steps) > maxEscalationSteps {
			return nil, fmt.Errorf("escalation sequence %s must contain 1-%d steps", id, maxEscalationSteps)
		}

		sequence := EscalationSequence{ID: id, Trigger: trigger, Steps: make([]EscalationStep, 0, len(value.Steps))}
		var previous time.Duration
		for index, step := range value.Steps {
			after, err := time.ParseDuration(strings.TrimSpace(step.After))
			if err != nil || after < 0 {
				return nil, fmt.Errorf("escalation sequence %s step %d has invalid after duration", id, index+1)
			}
			if index > 0 && after <= previous {
				return nil, fmt.Errorf("escalation sequence %s step thresholds must increase", id)
			}
			responsibility := strings.ToUpper(strings.TrimSpace(step.Responsibility))
			if !supportedEscalationResponsibility(responsibility) {
				return nil, fmt.Errorf("escalation sequence %s step %d has unsupported responsibility", id, index+1)
			}
			if step.DepartmentLevelsUp != nil && (*step.DepartmentLevelsUp < 0 || *step.DepartmentLevelsUp > maxDepartmentLevelsUp) {
				return nil, fmt.Errorf("escalation sequence %s step %d department_levels_up must be between 0 and %d", id, index+1, maxDepartmentLevelsUp)
			}

			sourceRoles, err := normalizeEscalationRoles(step.SourceRoles, maxEscalationRoleTargets)
			if err != nil {
				return nil, fmt.Errorf("escalation sequence %s step %d source_roles: %w", id, index+1, err)
			}
			if step.SourceRoles != nil && len(sourceRoles) == 0 {
				return nil, fmt.Errorf("escalation sequence %s step %d source_roles cannot be empty", id, index+1)
			}

			var targetRoles, targetGroups []string
			if step.Targets != nil {
				targetRoles, err = normalizeEscalationRoles(step.Targets.Roles, maxEscalationRoleTargets)
				if err != nil {
					return nil, fmt.Errorf("escalation sequence %s step %d target roles: %w", id, index+1, err)
				}
				targetGroups, err = normalizeEscalationGroupIDs(step.Targets.Groups)
				if err != nil {
					return nil, fmt.Errorf("escalation sequence %s step %d target groups: %w", id, index+1, err)
				}
				if len(targetRoles) == 0 && len(targetGroups) == 0 {
					return nil, fmt.Errorf("escalation sequence %s step %d targets must contain at least one role or group", id, index+1)
				}
			}

			sequence.Steps = append(sequence.Steps, EscalationStep{
				After:              after,
				Responsibility:     responsibility,
				DepartmentLevelsUp: step.DepartmentLevelsUp,
				SourceRoles:         sourceRoles,
				TargetRoles:         targetRoles,
				TargetGroupIDs:      targetGroups,
			})
			previous = after
		}
		result = append(result, sequence)
	}
	return result, nil
}

// DepartmentScope applies an escalation step's department ancestry rule to a
// normalized department path. A nil level means legal-entity scope. A false
// result means the requested ancestry exceeds the supplied path.
func DepartmentScope(base []string, levels *int) ([]string, bool) {
	if levels == nil {
		return nil, true
	}
	if *levels < 0 || len(base) == 0 {
		return nil, false
	}
	keep := len(base) - *levels
	if keep < 1 {
		return nil, false
	}
	return append([]string(nil), base[:keep]...), true
}

func validateEscalationSequences(definition json.RawMessage) error {
	_, err := ParseEscalationSequences(definition)
	return err
}

func normalizeEscalationRoles(values []string, limit int) ([]string, error) {
	if len(values) > limit {
		return nil, fmt.Errorf("supports at most %d role selectors", limit)
	}
	normalized := identity.NormalizeRoleCodes(values)
	if len(normalized) != len(values) {
		return nil, fmt.Errorf("role selectors must be non-empty and unique")
	}
	for _, role := range normalized {
		if len(role) > 80 {
			return nil, fmt.Errorf("role selector is too long")
		}
	}
	return normalized, nil
}

func normalizeEscalationGroupIDs(values []string) ([]string, error) {
	if len(values) > maxEscalationGroupTargets {
		return nil, fmt.Errorf("supports at most %d group selectors", maxEscalationGroupTargets)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if !validUUID(value) {
			return nil, fmt.Errorf("group selectors must be ClearSight directory group UUIDs")
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("group selectors must be unique")
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	raw := strings.ReplaceAll(value, "-", "")
	if len(raw) != 32 {
		return false
	}
	_, err := hex.DecodeString(raw)
	return err == nil
}

func supportedEscalationTrigger(value string) bool {
	switch value {
	case "OVERDUE", "NO_ROUTE", "AUTHORITY_INSUFFICIENT", "MATERIALITY_INCREASE", "RECIPIENT_UNAVAILABLE", "CONFLICT":
		return true
	default:
		return false
	}
}

func supportedEscalationResponsibility(value string) bool {
	switch value {
	case "PERFORMER", "ACCOUNTABLE_OWNER", "PROPOSER", "REVIEWER", "INDEPENDENT_CHALLENGER", "AUTHORIZER", "SIGNATORY", "TRANSMITTER", "ACKNOWLEDGEMENT_RECORDER", "ESCALATION_OWNER":
		return true
	default:
		return false
	}
}

func decodeStrictJSON(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected trailing JSON")
		}
		return err
	}
	return nil
}
