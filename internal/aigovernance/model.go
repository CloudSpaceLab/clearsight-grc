package aigovernance

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

var (
	ErrNotFound          = errors.New("ai governance object not found")
	ErrConflict          = errors.New("ai governance version conflict")
	ErrInvalid           = errors.New("ai governance object invalid")
	ErrInvalidTransition = errors.New("ai governance transition invalid")
	ErrMakerChecker      = errors.New("ai governance maker-checker violation")
	ErrGrantInvalid      = errors.New("ai execution grant invalid")
)

type Policy struct {
	ID                   string                     `json:"id"`
	TenantID             string                     `json:"tenant_id"`
	Code                 string                     `json:"code"`
	Name                 string                     `json:"name"`
	ActionClass          string                     `json:"action_class"`
	Eligibility          json.RawMessage            `json:"eligibility"`
	BlastRadiusLimit     json.RawMessage            `json:"blast_radius_limit"`
	VerificationContract json.RawMessage            `json:"verification_contract"`
	Definition           aigateway.PolicyDefinition `json:"definition"`
	Status               string                     `json:"status"`
	RolloutMode          aigateway.RolloutMode      `json:"rollout_mode"`
	MakerID              string                     `json:"maker_id"`
	CheckerID            string                     `json:"checker_id,omitempty"`
	Checksum             string                     `json:"checksum"`
	EffectiveFrom        *time.Time                 `json:"effective_from,omitempty"`
	EffectiveUntil       *time.Time                 `json:"effective_until,omitempty"`
	SubmittedAt          *time.Time                 `json:"submitted_at,omitempty"`
	ApprovedAt           *time.Time                 `json:"approved_at,omitempty"`
	ActivatedAt          *time.Time                 `json:"activated_at,omitempty"`
	SuspendedAt          *time.Time                 `json:"suspended_at,omitempty"`
	RetiredAt            *time.Time                 `json:"retired_at,omitempty"`
	Version              int64                      `json:"version"`
	RecordVersion        int64                      `json:"record_version"`
}

type Workload struct {
	ID                    string            `json:"id"`
	WorkloadID            string            `json:"workload_id"`
	TenantID              string            `json:"tenant_id"`
	Code                  string            `json:"code"`
	Name                  string            `json:"name"`
	Purpose               string            `json:"purpose"`
	Environment           string            `json:"environment"`
	OwnerPrincipalID      string            `json:"owner_principal_id"`
	ServicePrincipalID    string            `json:"service_principal_id,omitempty"`
	AllowedModels         []string          `json:"allowed_models"`
	RequestsPerMinute     int64             `json:"requests_per_minute"`
	TokensPerMinute       int64             `json:"tokens_per_minute"`
	CostMicroUSDPerMinute int64             `json:"cost_microusd_per_minute"`
	MaxConcurrent         int64             `json:"max_concurrent"`
	VerifiedMetadata      map[string]string `json:"verified_metadata,omitempty"`
	ApprovedResources     []string          `json:"approved_resources,omitempty"`
	PolicyID              string            `json:"policy_id"`
	PolicyVersion         int64             `json:"policy_version"`
	State                 string            `json:"state"`
	MakerID               string            `json:"maker_id,omitempty"`
	CheckerID             string            `json:"checker_id,omitempty"`
	EffectiveFrom         *time.Time        `json:"effective_from,omitempty"`
	EffectiveUntil        *time.Time        `json:"effective_until,omitempty"`
	SubmittedAt           *time.Time        `json:"submitted_at,omitempty"`
	ApprovedAt            *time.Time        `json:"approved_at,omitempty"`
	ActivatedAt           *time.Time        `json:"activated_at,omitempty"`
	SuspendedAt           *time.Time        `json:"suspended_at,omitempty"`
	RetiredAt             *time.Time        `json:"retired_at,omitempty"`
	Checksum              string            `json:"checksum"`
	CreatedAt             time.Time         `json:"created_at,omitempty"`
	UpdatedAt             time.Time         `json:"updated_at,omitempty"`
	Version               int64             `json:"version"`
	RecordVersion         int64             `json:"record_version"`
	KeySHA256             string            `json:"-"`
}

type DecisionReceipt struct {
	ReceiptID      string                   `json:"receipt_id"`
	TenantID       string                   `json:"tenant_id"`
	RequestID      string                   `json:"request_id"`
	WorkloadID     string                   `json:"workload_id"`
	PolicyID       string                   `json:"policy_id"`
	PolicyCode     string                   `json:"policy_code"`
	PolicyVersion  int64                    `json:"policy_version"`
	Decision       aigateway.DecisionAction `json:"decision"`
	ProposedAction aigateway.DecisionAction `json:"proposed_action,omitempty"`
	ReasonCodes    []string                 `json:"reason_codes,omitempty"`
	Obligations    []string                 `json:"obligations,omitempty"`
	ModelAlias     string                   `json:"model_alias,omitempty"`
	RouteID        string                   `json:"route_id,omitempty"`
	Outcome        string                   `json:"outcome"`
	ErrorCode      string                   `json:"error_code,omitempty"`
	ObservedAt     time.Time                `json:"observed_at"`
	ExpiresAt      time.Time                `json:"expires_at"`
	Fingerprint    string                   `json:"fingerprint"`
}

type ExecutionGrant struct {
	ID            string     `json:"id"`
	TenantID      string     `json:"tenant_id"`
	WorkloadID    string     `json:"workload_id"`
	MatterID      string     `json:"matter_id"`
	DecisionID    string     `json:"decision_id"`
	ActionHash    string     `json:"action_hash"`
	ApprovedBy    string     `json:"approved_by"`
	State         string     `json:"state"`
	ExpiresAt     time.Time  `json:"expires_at"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	RecordVersion int64      `json:"record_version"`
	Token         string     `json:"token,omitempty"`
	TokenSHA256   string     `json:"-"`
}

type CreatePolicyInput struct {
	TenantID             string                     `json:"tenant_id"`
	Code                 string                     `json:"code"`
	Name                 string                     `json:"name"`
	ActionClass          string                     `json:"action_class"`
	Eligibility          json.RawMessage            `json:"eligibility"`
	BlastRadiusLimit     json.RawMessage            `json:"blast_radius_limit"`
	VerificationContract json.RawMessage            `json:"verification_contract"`
	Definition           aigateway.PolicyDefinition `json:"definition"`
	RolloutMode          aigateway.RolloutMode      `json:"rollout_mode"`
	MakerID              string                     `json:"maker_id"`
	EffectiveFrom        *time.Time                 `json:"effective_from,omitempty"`
	EffectiveUntil       *time.Time                 `json:"effective_until,omitempty"`
}

type CreateWorkloadInput struct {
	TenantID              string            `json:"tenant_id"`
	WorkloadID            string            `json:"workload_id"`
	Code                  string            `json:"code"`
	Name                  string            `json:"name"`
	Purpose               string            `json:"purpose"`
	Environment           string            `json:"environment"`
	OwnerPrincipalID      string            `json:"owner_principal_id"`
	ServicePrincipalID    string            `json:"service_principal_id,omitempty"`
	AllowedModels         []string          `json:"allowed_models"`
	RequestsPerMinute     int64             `json:"requests_per_minute"`
	TokensPerMinute       int64             `json:"tokens_per_minute"`
	CostMicroUSDPerMinute int64             `json:"cost_microusd_per_minute"`
	MaxConcurrent         int64             `json:"max_concurrent"`
	VerifiedMetadata      map[string]string `json:"verified_metadata,omitempty"`
	ApprovedResources     []string          `json:"approved_resources,omitempty"`
	PolicyID              string            `json:"policy_id"`
	PolicyVersion         int64             `json:"policy_version"`
	KeySHA256             string            `json:"key_sha256"`
	MakerID               string            `json:"maker_id"`
}

type TransitionInput struct {
	TenantID        string `json:"tenant_id"`
	ID              string `json:"id"`
	ActorID         string `json:"actor_id"`
	ExpectedVersion int64  `json:"expected_version"`
}

type CreateGrantInput struct {
	TenantID   string          `json:"tenant_id"`
	WorkloadID string          `json:"workload_id"`
	MatterID   string          `json:"matter_id"`
	DecisionID string          `json:"decision_id"`
	Action     json.RawMessage `json:"action"`
	TTLMinutes int             `json:"ttl_minutes"`
	ActorID    string          `json:"actor_id"`
}
