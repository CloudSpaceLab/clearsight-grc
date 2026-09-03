package aigovernance

import (
	"context"
	"strings"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func (p *RuntimeProvider) ActiveTransportSnapshot(ctx context.Context, tenantID, environment string) (aigateway.TransportSnapshot, error) {
	if p == nil || p.repo == nil {
		return aigateway.TransportSnapshot{}, ErrNotFound
	}
	environment = strings.ToUpper(strings.TrimSpace(environment))
	value, err := p.repo.ActiveGatewayTransport(ctx, strings.TrimSpace(tenantID), environment)
	if err != nil {
		return aigateway.TransportSnapshot{}, err
	}
	if err := validateGatewayTransportChecksum(value); err != nil {
		return aigateway.TransportSnapshot{}, err
	}
	return aigateway.TransportSnapshot{
		ID: value.ID, TenantID: value.TenantID, Environment: value.Environment,
		Version: value.Version, Checksum: value.Checksum, Definition: value.Definition,
	}, nil
}
