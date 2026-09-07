package aigovernance

import (
	"context"
	"errors"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func (p *RuntimeProvider) GatewayEmergencyControl(ctx context.Context, tenantID, environment string) (aigateway.EmergencyControlState, error) {
	if p == nil || p.repo == nil {
		return aigateway.EmergencyControlState{}, ErrNotFound
	}
	repo, ok := p.repo.(gatewayEmergencyRepository)
	if !ok {
		return aigateway.EmergencyControlState{}, ErrNotFound
	}
	tenantID = strings.TrimSpace(tenantID)
	environment = normalizeGatewayEnvironment(environment)
	if tenantID == "" || environment == "" {
		return aigateway.EmergencyControlState{}, ErrInvalid
	}
	value, err := repo.GatewayEmergencyControl(ctx, tenantID, environment)
	if errors.Is(err, ErrNotFound) {
		return aigateway.EmergencyControlState{TenantID: tenantID, Environment: environment}, nil
	}
	if err != nil {
		return aigateway.EmergencyControlState{}, err
	}
	return aigateway.EmergencyControlState{
		TenantID: value.TenantID, Environment: value.Environment, Frozen: value.Frozen,
		RecordVersion: value.RecordVersion, UpdatedAt: value.UpdatedAt,
	}, nil
}
