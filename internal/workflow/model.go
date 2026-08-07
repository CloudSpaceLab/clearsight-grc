package workflow

import (
	"encoding/json"
	"time"
)

type Status string

const (
	StatusReady      Status = "READY"
	StatusInProgress Status = "IN_PROGRESS"
	StatusBlocked    Status = "BLOCKED"
	StatusEscalated  Status = "ESCALATED"
	StatusCompleted  Status = "COMPLETED"
	StatusCancelled  Status = "CANCELLED"
)

type Task struct {
	ID             string            `json:"id"`
	TenantID       string            `json:"tenant_id"`
	WorkflowID     string            `json:"workflow_id"`
	StepKey        string            `json:"step_key"`
	Responsibility string            `json:"responsibility"`
	PrincipalID    string            `json:"principal_id,omitempty"`
	Title          string            `json:"title"`
	Status         Status            `json:"status"`
	DueAt          *time.Time        `json:"due_at,omitempty"`
	ClaimedAt      *time.Time        `json:"claimed_at,omitempty"`
	CompletedAt    *time.Time        `json:"completed_at,omitempty"`
	Context        map[string]string `json:"context,omitempty"`
	Version        int64             `json:"version"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`

	// Projection metadata is intentionally internal-only. It lets actor-facing
	// reads enforce canonical Matter visibility/materiality without leaking raw
	// access policy or duplicating Matter state into the Task API contract.
	WorkflowKind   string          `json:"-"`
	MatterID       string          `json:"-"`
	MatterPriority int             `json:"-"`
	MatterScope    json.RawMessage `json:"-"`
}

type ListFilter struct {
	TenantID                  string
	PrincipalID               string
	Status                    Status
	WorkflowKind              string
	SupportedMatterWorkOnly   bool
	ActiveOnly                bool
	VisibleMatterWorkOnly     bool
	VisibleMatterActionsOnly  bool // narrow compatibility alias; callers should prefer VisibleMatterWorkOnly
	Limit                     int
}
