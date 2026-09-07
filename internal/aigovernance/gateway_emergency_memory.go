package aigovernance

import "context"

func (r *MemoryRepository) GatewayEmergencyControl(_ context.Context, tenantID, environment string) (GatewayEmergencyControl, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.gatewayEmergency[memKey(tenantID, environment)]
	if !ok {
		return GatewayEmergencyControl{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) SetGatewayEmergencyControl(_ context.Context, value GatewayEmergencyControl, expected int64) (GatewayEmergencyControl, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := memKey(value.TenantID, value.Environment)
	current, exists := r.gatewayEmergency[key]
	if !exists {
		if expected != 0 || value.RecordVersion != 1 {
			return GatewayEmergencyControl{}, ErrConflict
		}
	} else if current.RecordVersion != expected || value.RecordVersion != expected+1 {
		return GatewayEmergencyControl{}, ErrConflict
	}
	r.gatewayEmergency[key] = value
	return value, nil
}
