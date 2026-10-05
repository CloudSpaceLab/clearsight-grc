package workflow

import (
	"context"
	"errors"
	"time"
)

var ErrEscalationSimulationUnavailable = errors.New("escalation simulation unavailable")

type EscalationSimulationInput struct {
	TenantID        string
	LegalEntityID   string
	PolicyID        string
	SequenceID      string
	RevisionVersion int
	Limit           int
	At              time.Time
}

type EscalationSimulationCandidate struct {
	PrincipalID string `json:"principal_id"`
	DisplayName string `json:"display_name"`
}

type EscalationSimulationStep struct {
	Index          int                             `json:"index"`
	DueAt          time.Time                       `json:"due_at"`
	Responsibility string                          `json:"responsibility"`
	Status         string                          `json:"status"`
	Candidates     []EscalationSimulationCandidate `json:"candidates"`
}

type EscalationSimulationScenario struct {
	TaskID                string                     `json:"task_id"`
	MatterID              string                     `json:"matter_id"`
	Title                 string                     `json:"title"`
	CurrentResponsibility string                     `json:"current_responsibility"`
	CurrentPrincipalID    string                     `json:"current_principal_id,omitempty"`
	CurrentPrincipalName  string                     `json:"current_principal_name,omitempty"`
	DueAt                 time.Time                  `json:"due_at"`
	CurrentLevel          int                        `json:"current_level"`
	NextLevel             int                        `json:"next_level"`
	NextDueAt             *time.Time                 `json:"next_due_at,omitempty"`
	RecoveryAction        string                     `json:"recovery_action"`
	Steps                 []EscalationSimulationStep `json:"steps"`
}

type EscalationSimulation struct {
	PolicyID        string                         `json:"policy_id"`
	PolicyCode      string                         `json:"policy_code"`
	ActiveVersion   int                            `json:"active_version"`
	SequenceVersion int                            `json:"sequence_version"`
	SequenceID      string                         `json:"sequence_id"`
	Trigger         string                         `json:"trigger"`
	Checked         int                            `json:"checked"`
	Truncated       bool                           `json:"truncated"`
	Scenarios       []EscalationSimulationScenario `json:"scenarios"`
}

type EscalationSimulator interface {
	SimulateEscalation(context.Context, EscalationSimulationInput) (EscalationSimulation, error)
}
