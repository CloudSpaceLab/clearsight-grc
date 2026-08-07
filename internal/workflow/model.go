package workflow

import "time"

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
}

type CreateInput struct {
	TenantID       string            `json:"tenant_id"`
	WorkflowID     string            `json:"workflow_id"`
	StepKey        string            `json:"step_key"`
	Responsibility string            `json:"responsibility"`
	PrincipalID    string            `json:"principal_id,omitempty"`
	Title          string            `json:"title"`
	DueAt          *time.Time        `json:"due_at,omitempty"`
	Context        map[string]string `json:"context,omitempty"`
}

type TransitionInput struct {
	TenantID        string `json:"tenant_id"`
	ActorID         string `json:"actor_id,omitempty"`
	Status          Status `json:"status"`
	ExpectedVersion int64  `json:"expected_version"`
	Reason          string `json:"reason,omitempty"`
}

type ListFilter struct {
	TenantID    string
	PrincipalID string
	Status      Status
	Limit       int
}
