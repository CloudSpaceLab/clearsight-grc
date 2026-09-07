package aigateway

import (
	"context"
	"time"
)

const defaultEmergencyRefresh = time.Second

type EmergencyControlState struct {
	TenantID      string    `json:"tenant_id"`
	Environment   string    `json:"environment"`
	Frozen        bool      `json:"frozen"`
	RecordVersion int64     `json:"record_version"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
}

type EmergencyControlSource interface {
	GatewayEmergencyControl(context.Context, string, string) (EmergencyControlState, error)
}
