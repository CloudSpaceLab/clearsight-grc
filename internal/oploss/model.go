package oploss

import "time"

type EventType string

type Status string

type RecoveryKind string

const EventInternalFraud EventType = "INTERNAL_FRAUD"
const EventExternalFraud EventType = "EXTERNAL_FRAUD"
const EventEmploymentPractices EventType = "EMPLOYMENT_PRACTICES"
const EventClientProductsBusinessPractices EventType = "CLIENT_PRODUCTS_BUSINESS_PRACTICES"
const EventDamageToPhysicalAssets EventType = "DAMAGE_TO_PHYSICAL_ASSETS"
const EventBusinessDisruptionSystems EventType = "BUSINESS_DISRUPTION_SYSTEM_FAILURES"
const EventExecutionDeliveryProcess EventType = "EXECUTION_DELIVERY_PROCESS_MANAGEMENT"
const EventOther EventType = "OTHER"

const StatusActive Status = "ACTIVE"
const StatusVoided Status = "VOIDED"

const RecoveryCash RecoveryKind = "RECOVERY"
const RecoveryReversal RecoveryKind = "REVERSAL"

type Loss struct {
	ID                  string    `json:"id"`
	TenantID            string    `json:"tenant_id"`
	LegalEntityID       string    `json:"legal_entity_id"`
	OrganizationScopeID string    `json:"organization_scope_id,omitempty"`
	Code                string    `json:"code"`
	Title               string    `json:"title"`
	EventType           EventType `json:"event_type"`
	Cause               string    `json:"cause"`
	Description         string    `json:"description"`
	GrossAmountMinor    int64     `json:"gross_amount_minor"`
	Currency            string    `json:"currency"`
	OccurredAt          time.Time `json:"occurred_at"`
	DiscoveredAt        time.Time `json:"discovered_at"`
	RiskID              string    `json:"risk_id,omitempty"`
	MatterID            string    `json:"matter_id,omitempty"`
	OwnerPrincipalID    string    `json:"owner_principal_id,omitempty"`
	Status              Status    `json:"status"`
	Version             int64     `json:"version"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Recovery struct {
	ID          string       `json:"id"`
	LossID      string       `json:"loss_id"`
	LossVersion int64        `json:"loss_version"`
	Kind        RecoveryKind `json:"kind"`
	AmountMinor int64        `json:"amount_minor"`
	Currency    string       `json:"currency"`
	Reference   string       `json:"reference,omitempty"`
	RecoveredAt time.Time    `json:"recovered_at"`
	ActorID     string       `json:"actor_id,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

type Totals struct {
	GrossAmountMinor     int64  `json:"gross_amount_minor"`
	RecoveredAmountMinor int64  `json:"recovered_amount_minor"`
	NetLossMinor         int64  `json:"net_loss_minor"`
	Currency             string `json:"currency"`
	RecoveryStatus       string `json:"recovery_status"`
}

type Aggregate struct {
	Loss       Loss       `json:"loss"`
	Recoveries []Recovery `json:"recoveries"`
	Totals     Totals     `json:"totals"`
}

type Summary struct {
	Loss   Loss   `json:"loss"`
	Totals Totals `json:"totals"`
}

type Page struct {
	Items               []Summary `json:"items"`
	NextCursor          string    `json:"next_cursor,omitempty"`
	OrganizationScopeID string    `json:"organization_scope_id,omitempty"`
}

type ListFilter struct {
	Status               Status
	EventType            EventType
	Currency             string
	OrganizationScopeID  string
	OrganizationScopeIDs []string
	RiskID               string
	RecoveryStatus       string
	Search               string
	Cursor               string
	Limit                int
}

type Scope struct {
	TenantID      string
	LegalEntityID string
}

type CreateInput struct {
	TenantID            string    `json:"tenant_id,omitempty"`
	LegalEntityID       string    `json:"legal_entity_id,omitempty"`
	OrganizationScopeID string    `json:"organization_scope_id,omitempty"`
	Code                string    `json:"code"`
	Title               string    `json:"title"`
	EventType           EventType `json:"event_type"`
	Cause               string    `json:"cause"`
	Description         string    `json:"description"`
	GrossAmountMinor    int64     `json:"gross_amount_minor"`
	Currency            string    `json:"currency"`
	OccurredAt          time.Time `json:"occurred_at"`
	DiscoveredAt        time.Time `json:"discovered_at"`
	RiskID              string    `json:"risk_id,omitempty"`
	MatterID            string    `json:"matter_id,omitempty"`
	OwnerPrincipalID    string    `json:"owner_principal_id,omitempty"`
	ActorID             string    `json:"actor_id,omitempty"`
}

type UpdateInput struct {
	TenantID            string    `json:"tenant_id,omitempty"`
	LegalEntityID       string    `json:"legal_entity_id,omitempty"`
	LossID              string    `json:"loss_id,omitempty"`
	ExpectedVersion     int64     `json:"expected_version"`
	OrganizationScopeID string    `json:"organization_scope_id,omitempty"`
	Title               string    `json:"title"`
	EventType           EventType `json:"event_type"`
	Cause               string    `json:"cause"`
	Description         string    `json:"description"`
	GrossAmountMinor    int64     `json:"gross_amount_minor"`
	Currency            string    `json:"currency"`
	OccurredAt          time.Time `json:"occurred_at"`
	DiscoveredAt        time.Time `json:"discovered_at"`
	RiskID              string    `json:"risk_id,omitempty"`
	MatterID            string    `json:"matter_id,omitempty"`
	OwnerPrincipalID    string    `json:"owner_principal_id,omitempty"`
	Status              Status    `json:"status"`
	ActorID             string    `json:"actor_id,omitempty"`
}

type RecoveryInput struct {
	TenantID        string       `json:"tenant_id,omitempty"`
	LegalEntityID   string       `json:"legal_entity_id,omitempty"`
	LossID          string       `json:"loss_id,omitempty"`
	ExpectedVersion int64        `json:"expected_version"`
	Kind            RecoveryKind `json:"kind"`
	AmountMinor     int64        `json:"amount_minor"`
	Reference       string       `json:"reference,omitempty"`
	RecoveredAt     time.Time    `json:"recovered_at"`
	ActorID         string       `json:"actor_id,omitempty"`
}

type Event struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	LegalEntityID string    `json:"legal_entity_id"`
	LossID        string    `json:"loss_id"`
	LossVersion   int64     `json:"loss_version"`
	Type          string    `json:"type"`
	ActorID       string    `json:"actor_id,omitempty"`
	OccurredAt    time.Time `json:"occurred_at"`
}
