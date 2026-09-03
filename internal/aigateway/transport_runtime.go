package aigateway

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type transportManager struct {
	config   RuntimeConfig
	source   TransportSnapshotSource
	resolver SecretResolver
	refresh  time.Duration
	now      func() time.Time
	slots    sync.Map
}

type transportSlot struct {
	mu              sync.Mutex
	router          *router
	appliedChecksum string
	appliedRevision int64
	desiredChecksum string
	desiredRevision int64
	expiresAt       time.Time
	lastError       string
}

type TransportApplyStatus struct {
	TenantID        string `json:"tenant_id"`
	Environment     string `json:"environment"`
	DesiredRevision int64  `json:"desired_revision"`
	DesiredChecksum string `json:"desired_checksum,omitempty"`
	AppliedRevision int64  `json:"applied_revision"`
	AppliedChecksum string `json:"applied_checksum,omitempty"`
	Degraded        bool   `json:"degraded"`
	ErrorCode       string `json:"error_code,omitempty"`
}

func newTransportManager(config RuntimeConfig, source TransportSnapshotSource, resolver SecretResolver) (*transportManager, error) {
	if source == nil || resolver == nil {
		return nil, fmt.Errorf("gateway transport control plane is incomplete")
	}
	refresh := config.GovernanceRefresh
	if refresh <= 0 {
		refresh = defaultGovernanceRefresh
	}
	return &transportManager{config: config, source: source, resolver: resolver, refresh: refresh, now: time.Now}, nil
}

func (m *transportManager) ready() bool {
	return m != nil && m.source != nil && m.source.Ready()
}

func (m *transportManager) routerFor(ctx context.Context, workload Workload) (*router, error) {
	if m == nil || m.source == nil {
		return nil, ErrUnavailable
	}
	environment := strings.ToUpper(strings.TrimSpace(workload.Environment))
	if environment == "" {
		environment = strings.ToUpper(strings.TrimSpace(m.config.Environment))
	}
	key := workload.TenantID + "|" + environment
	raw, _ := m.slots.LoadOrStore(key, &transportSlot{})
	slot := raw.(*transportSlot)
	slot.mu.Lock()
	defer slot.mu.Unlock()
	now := m.now().UTC()
	if slot.router != nil && now.Before(slot.expiresAt) {
		return slot.router, nil
	}
	snapshot, err := m.source.ActiveTransportSnapshot(ctx, workload.TenantID, environment)
	if err != nil {
		slot.expiresAt = now.Add(m.refresh)
		slot.lastError = "TRANSPORT_REFRESH_FAILED"
		if slot.router != nil {
			return slot.router, nil
		}
		return nil, withCause(ErrUnavailable, err)
	}
	if snapshot.Version < 1 || snapshot.Checksum == "" || snapshot.TenantID != workload.TenantID || !strings.EqualFold(snapshot.Environment, environment) {
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

func (m *transportManager) status(tenantID, environment string) TransportApplyStatus {
	status := TransportApplyStatus{TenantID: tenantID, Environment: strings.ToUpper(strings.TrimSpace(environment))}
	if m == nil {
		status.Degraded = true
		status.ErrorCode = "TRANSPORT_CONTROL_UNAVAILABLE"
		return status
	}
	key := tenantID + "|" + status.Environment
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
	status.Degraded = slot.lastError != "" || (slot.desiredRevision > 0 && slot.desiredRevision != slot.appliedRevision)
	status.ErrorCode = slot.lastError
	return status
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
