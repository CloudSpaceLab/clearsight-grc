package aigateway

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"time"
)

func NewGatewayWithControlPlane(config RuntimeConfig, state WorkloadProvider, facts FactResolver, source TransportSnapshotSource, resolver SecretResolver, logger *slog.Logger) (*Gateway, error) {
	if state == nil {
		return nil, ErrPolicyUnavailable
	}
	if config.TransportMode != TransportDatabase {
		return nil, invalid("transport", "Database transport control is not enabled.")
	}
	if facts == nil {
		facts = defaultFactResolver{}
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	manager, err := newTransportManager(config, source, resolver)
	if err != nil {
		return nil, err
	}
	now := time.Now
	var receipts ReceiptRecorder
	if recorder, ok := state.(ReceiptRecorder); ok {
		receipts = recorder
	}
	return &Gateway{
		config: config, state: state, facts: facts, receipts: receipts, transport: manager,
		budgets: newBudgetManager(), telemetry: newTelemetry(now()), logger: logger, now: now,
	}, nil
}

func (g *Gateway) TransportStatus(tenantID, environment string) TransportApplyStatus {
	if g == nil || g.transport == nil {
		return TransportApplyStatus{TenantID: tenantID, Environment: strings.ToUpper(strings.TrimSpace(environment)), Degraded: true, ErrorCode: "TRANSPORT_CONTROL_UNAVAILABLE"}
	}
	return g.transport.status(tenantID, environment)
}

// RefreshTransportStatus performs the same bounded candidate refresh used by a
// workload request and then returns compact desired-versus-applied state. It is
// intended for authenticated operational health checks and never returns
// provider credentials or request/response content.
func (g *Gateway) RefreshTransportStatus(ctx context.Context, tenantID, environment string) TransportApplyStatus {
	environment = strings.ToUpper(strings.TrimSpace(environment))
	if g == nil || g.transport == nil {
		return TransportApplyStatus{TenantID: tenantID, Environment: environment, Degraded: true, ErrorCode: "TRANSPORT_CONTROL_UNAVAILABLE"}
	}
	_, _ = g.transport.routerFor(ctx, Workload{TenantID: strings.TrimSpace(tenantID), Environment: environment})
	return g.transport.status(strings.TrimSpace(tenantID), environment)
}
