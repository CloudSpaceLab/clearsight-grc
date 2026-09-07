package aigovernance

import "time"

type GatewayEmergencyControl struct {
	ID            string    `json:"id,omitempty"`
	TenantID      string    `json:"tenant_id"`
	Environment   string    `json:"environment"`
	Frozen        bool      `json:"frozen"`
	Reason        string    `json:"reason,omitempty"`
	ActorID       string    `json:"actor_id,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	RecordVersion int64     `json:"record_version"`
}

type SetGatewayEmergencyControlInput struct {
	TenantID        string `json:"tenant_id"`
	Environment     string `json:"environment"`
	Frozen          bool   `json:"frozen"`
	Reason          string `json:"reason"`
	ActorID         string `json:"actor_id"`
	ExpectedVersion int64  `json:"expected_version"`
}

type gatewayEmergencyRepository interface {
	GatewayEmergencyControl(context.Context, string, string) (GatewayEmergencyControl, error)
	SetGatewayEmergencyControl(context.Context, GatewayEmergencyControl, int64) (GatewayEmergencyControl, error)
}
