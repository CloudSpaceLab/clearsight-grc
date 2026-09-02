package workflow

import (
	"encoding/json"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/sourceaccess"
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
	ID             string                          `json:"id"`
	TenantID       string                          `json:"tenant_id"`
	WorkflowID     string                          `json:"workflow_id"`
	StepKey        string                          `json:"step_key"`
	Responsibility string                          `json:"responsibility"`
	PrincipalID    string                          `json:"principal_id,omitempty"`
	Title          string                          `json:"title"`
	Status         Status                          `json:"status"`
	DueAt          *time.Time                      `json:"due_at,omitempty"`
	ClaimedAt      *time.Time                      `json:"claimed_at,omitempty"`
	CompletedAt    *time.Time                      `json:"completed_at,omitempty"`
	Context        map[string]string               `json:"context,omitempty"`
	SourceBindings []sourceaccess.BindingReference `json:"source_bindings,omitempty"`
	Version        int64                           `json:"version"`
	CreatedAt      time.Time                       `json:"created_at"`
	UpdatedAt      time.Time                       `json:"updated_at"`

	// Projection metadata is intentionally internal-only. It lets actor-facing
	// reads enforce canonical source-domain visibility without leaking access
	// policy or duplicating source-domain state into the Task API contract.
	WorkflowKind            string          `json:"-"`
	LegalEntityID           string          `json:"-"`
	MatterID                string          `json:"-"`
	MatterPriority          int             `json:"-"`
	MatterScope             json.RawMessage `json:"-"`
	EvidenceRequestID       string          `json:"-"`
	EvidenceRecipientID     string          `json:"-"`
	EvidenceSubjectVisible  bool            `json:"-"`
	DocumentProposalVisible bool            `json:"-"`
}

type ListFilter struct {
	TenantID              string
	LegalEntityID         string
	PrincipalID           string
	Status                Status
	WorkflowKind          string
	ActiveOnly            bool
	VisibleMatterWorkOnly bool
	VisibleActorWorkOnly  bool
	Limit                 int
}
