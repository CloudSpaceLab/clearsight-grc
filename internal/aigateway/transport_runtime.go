package aigateway

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type transportManager struct {
	config           RuntimeConfig
	source           TransportSnapshotSource
	emergency        EmergencyControlSource
	resolver         SecretResolver
	refresh          time.Duration
	emergencyRefresh time.Duration
	now              func() time.Time
	slots            sync.Map
}

type transportSlot struct {
	mu                 sync.Mutex
	router             *router
	appliedChecksum    string
	appliedRevision    int64
	desiredChecksum    string
	desiredRevision    int64
	expiresAt          time.Time
	lastError          string
	emergencyExpiresAt time.Time
	emergencyFrozen    bool
	emergencyRevision  int64
	emergencyError     string
}

type TransportApplyStatus struct {
	TenantID           string `json:"tenant_id"`
	Environment        string `json:"environment"`
	DesiredRevision    int64  `json:"desired_revision"`
	DesiredChecksum    string `json:"desired_checksum,omitempty"`
	AppliedRevision    int64  `json:"applied_revision"`
	AppliedChecksum    string `json:"applied_checksum,omitempty"`
	EmergencySupported bool   `json:"emergency_supported"`
	EmergencyRevision  int64  `json:"emergency_revision"`
	OutboundFrozen     bool   `json:"outbound_frozen"`
	Degraded           bool   `json:"degraded"`
	ErrorCode          string `json:"error_code,omitempty"`
}

func newTransportManager(config RuntimeConfig, source TransportSnapshotSource, resolver SecretResolver) (*transportManager, error) {
	if source == nil || resolver == nil {
		return nil, fmt.Errorf("gateway transport control plane is incomplete")
	}
	emergency, ok := source.(EmergencyControlSource)
	if !ok || emergency == nil {
		return nil, fmt.Errorf("gateway emergency control source is incomplete")
	}
	refresh := config.GovernanceRefresh
	if refresh <= 0 {
		refresh = defaultGovernanceRefresh
	}
	emergencyRefresh := defaultEmergencyRefresh
	if refresh < emergencyRefresh {
		emergencyRefresh = refresh
	}
	return &transportManager{
		config: config, source: source, emergency: emergency, resolver: resolver,
		refresh: refresh, emergencyRefresh: emergencyRefresh, now: time.Now,
	}, nil
}

func (m *transportManager) ready() bool {
	return m != nil && m.source != nil && m.source.Ready()
}

func (m *transportManager) ensureOutboundAllowed(ctx context.Context, workload Workload) error {
	if m == nil || m.emergency == nil {
		return ErrEmergencyControlUnavailable
	}
	tenantID := strings.TrimSpace(workload.TenantID)
	environment := m.environmentFor(workload)
	key := tenantID + "|" + environment
	raw, _ := m.slots.LoadOrStore(key, &transportSlot{})
	slot := raw.(*transportSlot)
	slot.mu.Lock()
	defer slot.mu.Unlock()
	return m.checkEmergencyLocked(ctx, slot, tenantID, environment, m.now().UTC())
}

func (m *transportManager) routerFor(ctx context.Context, workload Workload) (*router, error) {
	if m == nil || m.source == nil {
		return nil, ErrUnavailable
	}
	tenantID := strings.TrimSpace(workload.TenantID)
	environment := m.environmentFor(workload)
	key := tenantID + "|" + environment
	raw, _ := m.slots.LoadOrStore(key, &transportSlot{})
	slot := raw.(*transportSlot)
	slot.mu.Lock()
	defer slot.mu.Unlock()
	now := m.now().UTC()
	if err := m.checkEmergencyLocked(ctx, slot, tenantID, environment, now); err != nil {
		return nil, err
	}
	if slot.router != nil && now.Before(slot.expiresAt) {
		return slot.router, nil
	}
	snapshot, err := m.source.ActiveTransportSnapshot(ctx, tenantID, environment)
	if err != nil {
		slot.expiresAt = now.Add(m.refresh)
		slot.lastError = "TRANSPORT_REFRESH_FAILED"
		if slot.router != nil {
			return slot.router, nil
		}
		return nil, withCause(ErrUnavailable, err)
	}
	if snapshot.Version < 1 || snapshot.Checksum == "" || snapshot.TenantID != tenantID || !strings.EqualFold(snapshot.Environment, environment) {
		slot.expiresAt = now.Add(m.refresh)
		slot.lastError = "TRANSPORT_SNAPSHOT_INVALID"
		if slot.router != nil {
			return slot.router, nil
		}
		return nil, ErrUnavailable
	}
	slot.desiredRevision = snapshot.Version
	slot.desiredChecksum = snapshot.Checksum
	if slot.router != nil && slot.appliedChecksum == snapshot.Checksum && slot.appliedRevision == snapshot.Version {
		slot.expiresAt = now.Add(m.refresh)
		slot.lastError = ""
		return slot.router, nil
	}
	candidate, err := m.buildRouter(ctx, snapshot)
	if err != nil {
		slot.expiresAt = now.Add(m.refresh)
		slot.lastError = "TRANSPORT_APPLY_FAILED"
		if slot.router != nil {
			return slot.router, nil
		}
		return nil, withCause(ErrUnavailable, err)
	}
	slot.router = candidate
	slot.appliedChecksum = snapshot.Checksum
	slot.appliedRevision = snapshot.Version
	slot.expiresAt = now.Add(m.refresh)
	slot.lastError = ""
	return candidate, nil
}

func (m *transportManager) checkEmergencyLocked(ctx context.Context, slot *transportSlot, tenantID, environment string, now time.Time) error {
	if now.Before(slot.emergencyExpiresAt) {
		if slot.emergencyError != "" {
			return ErrEmergencyControlUnavailable
		}
		if slot.emergencyFrozen {
			return ErrOutboundFrozen
		}
		return nil
	}
	state, err := m.emergency.GatewayEmergencyControl(ctx, tenantID, environment)
	slot.emergencyExpiresAt = now.Add(m.emergencyRefresh)
	if err != nil {
		slot.emergencyError = "EMERGENCY_CONTROL_UNAVAILABLE"
		return withCause(ErrEmergencyControlUnavailable, err)
	}
	if state.TenantID != tenantID || !strings.EqualFold(state.Environment, environment) || state.RecordVersion < 0 || (state.Frozen && state.RecordVersion < 1) {
		slot.emergencyError = "EMERGENCY_CONTROL_INVALID"
		return ErrEmergencyControlUnavailable
	}
	slot.emergencyFrozen = state.Frozen
	slot.emergencyRevision = state.RecordVersion
	slot.emergencyError = ""
	if state.Frozen {
		return ErrOutboundFrozen
	}
	return nil
}

func (m *transportManager) status(tenantID, environment string) TransportApplyStatus {
	status := TransportApplyStatus{
		TenantID: strings.TrimSpace(tenantID), Environment: strings.ToUpper(strings.TrimSpace(environment)),
		EmergencySupported: m != nil && m.emergency != nil,
	}
	if m == nil {
		status.Degraded = true
		status.ErrorCode = "TRANSPORT_CONTROL_UNAVAILABLE"
		return status
	}
	key := status.TenantID + "|" + status.Environment
	raw, ok := m.slots.Load(key)
	if !ok {
		return status
	}
	slot := raw.(*transportSlot)
	slot.mu.Lock()
	defer slot.mu.Unlock()
	status.DesiredRevision = slot.desiredRevision
	status.DesiredChecksum = slot.desiredChecksum
	status.AppliedRevision = slot.appliedRevision
	status.AppliedChecksum = slot.appliedChecksum
	status.EmergencyRevision = slot.emergencyRevision
	status.OutboundFrozen = slot.emergencyFrozen
	status.Degraded = slot.lastError != "" || slot.emergencyError != "" || (slot.desiredRevision > 0 && slot.desiredRevision != slot.appliedRevision)
	if slot.emergencyError != "" {
		status.ErrorCode = slot.emergencyError
	} else {
		status.ErrorCode = slot.lastError
	}
	return status
}

func (m *transportManager) environmentFor(workload Workload) string {
	environment := strings.ToUpper(strings.TrimSpace(workload.Environment))
	if environment == "" {
		environment = strings.ToUpper(strings.TrimSpace(m.config.Environment))
	}
	return environment
}

func (m *transportManager) buildRouter(ctx context.Context, snapshot TransportSnapshot) (*router, error) {
	resolved, err := ResolveTransportDefinition(ctx, strings.ToLower(snapshot.Environment), m.config.RequestTimeout, snapshot.Definition, m.resolver)
	if err != nil {
		return nil, err
	}
	enabled := make(map[string]struct{}, len(resolved.Providers))
	providers := make(map[string]*providerRuntime, len(resolved.Providers))
	for _, providerConfig := range resolved.Providers {
		var provider Provider
		switch providerConfig.Kind {
		case ProviderKindOpenAI:
			provider = newOpenAIProvider(providerConfig, m.config.MaxProviderBodyBytes, m.config.MaxSSEEventBytes)
		case ProviderKindAnthropic:
			provider = newAnthropicProvider(providerConfig, m.config.MaxProviderBodyBytes, m.config.MaxSSEEventBytes)
		default:
			return nil, fmt.Errorf("unsupported provider adapter kind")
		}
		enabled[providerConfig.ID] = struct{}{}
		providers[providerConfig.ID] = &providerRuntime{provider: provider, config: providerConfig}
	}
	models := cloneModelConfigs(resolved.Models)
	for index := range models {
		routes := models[index].Routes[:0]
		for _, route := range models[index].Routes {
			if _, ok := enabled[route.ProviderID]; ok {
				routes = append(routes, route)
			}
		}
		models[index].Routes = routes
		if len(routes) == 0 {
			return nil, fmt.Errorf("model alias %s has no enabled provider route", models[index].Alias)
		}
	}
	config := m.config
	config.CircuitBreaker = resolved.CircuitBreaker
	config.Models = models
	config.Providers = resolved.Providers
	return newRouter(config, providers)
}
