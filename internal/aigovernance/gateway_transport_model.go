package aigovernance

import (
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

const (
	GatewayTransportDraft           = "DRAFT"
	GatewayTransportPendingApproval = "PENDING_APPROVAL"
	GatewayTransportApproved        = "APPROVED"
	GatewayTransportActive          = "ACTIVE"
	GatewayTransportSuspended       = "SUSPENDED"
	GatewayTransportRetired         = "RETIRED"
)

type GatewayTransportRevision struct {
	ID            string                        `json:"id"`
	TenantID      string                        `json:"tenant_id"`
	Environment   string                        `json:"environment"`
	Definition    aigateway.TransportDefinition `json:"definition"`
	Status        string                        `json:"status"`
	MakerID       string                        `json:"maker_id"`
	CheckerID     string                        `json:"checker_id,omitempty"`
	ChangeReason  string                        `json:"change_reason"`
	Checksum      string                        `json:"checksum"`
	SubmittedAt   *time.Time                    `json:"submitted_at,omitempty"`
	ApprovedAt    *time.Time                    `json:"approved_at,omitempty"`
	ActivatedAt   *time.Time                    `json:"activated_at,omitempty"`
	SuspendedAt   *time.Time                    `json:"suspended_at,omitempty"`
	RetiredAt     *time.Time                    `json:"retired_at,omitempty"`
	CreatedAt     time.Time                     `json:"created_at"`
	UpdatedAt     time.Time                     `json:"updated_at"`
	Version       int64                         `json:"version"`
	RecordVersion int64                         `json:"record_version"`
}

type CreateGatewayTransportInput struct {
	TenantID     string                        `json:"tenant_id"`
	Environment  string                        `json:"environment"`
	Definition   aigateway.TransportDefinition `json:"definition"`
	MakerID      string                        `json:"maker_id"`
	ChangeReason string                        `json:"change_reason"`
}

type GatewayTransportTransitionInput struct {
	TenantID        string `json:"tenant_id"`
	ID              string `json:"id"`
	ActorID         string `json:"actor_id"`
	ExpectedVersion int64  `json:"expected_version"`
}
