package aigateway

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type mutableTransportSource struct {
	snapshot TransportSnapshot
	err      error
}

func (source *mutableTransportSource) ActiveTransportSnapshot(context.Context, string, string) (TransportSnapshot, error) {
	return source.snapshot, source.err
}
func (source *mutableTransportSource) Ready() bool { return true }

type mapSecretResolver map[string]string

func (resolver mapSecretResolver) ResolveSecret(_ context.Context, ref string) (string, error) {
	value, ok := resolver[ref]
	if !ok {
		return "", errors.New("missing secret")
	}
	return value, nil
}

func transportSnapshot(version int64, routeID, secretRef string) TransportSnapshot {
	return TransportSnapshot{
		ID: "config-revision", TenantID: "tenant-a", Environment: "PRODUCTION", Version: version,
		Checksum: fmt.Sprintf("checksum-v%d", version),
		Definition: TransportDefinition{
			Providers: []TransportProviderConfig{{
				ID: "provider-a", Name: "Provider A", Kind: ProviderKindOpenAI,
				BaseURL: "https://api.openai.com", SecretRef: secretRef, State: ProviderStateEnabled,
			}},
			Models: []ModelConfig{{Alias: "safe-chat", Routes: []RouteConfig{{
				ID: routeID, ProviderID: "provider-a", Model: "gpt-5", Weight: 100,
			}}}},
		},
	}
}

func TestTransportManagerKeepsKnownGoodSnapshotWhenRefreshFails(t *testing.T) {
	now := time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC)
	source := &mutableTransportSource{snapshot: transportSnapshot(1, "route-v1", "env:PROVIDER_V1")}
	manager, err := newTransportManager(RuntimeConfig{
		Environment: "production", RequestTimeout: 2 * time.Minute, GovernanceRefresh: time.Second,
		MaxProviderBodyBytes: defaultMaxProviderBodyBytes, MaxSSEEventBytes: defaultMaxSSEEventBytes,
	}, source, mapSecretResolver{"env:PROVIDER_V1": "12345678"})
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return now }
	workload := Workload{TenantID: "tenant-a", Environment: "production", AllowedModels: map[string]struct{}{"safe-chat": {}}}
	first, err := manager.routerFor(context.Background(), workload)
	if err != nil {
		t.Fatalf("first apply error = %v", err)
	}
	candidates, _, err := first.candidatesFor(workload, "safe-chat", "")
	if err != nil || len(candidates) != 1 || candidates[0].ID != "route-v1" {
		t.Fatalf("first candidates = %#v err=%v", candidates, err)
	}

	now = now.Add(2 * time.Second)
	source.snapshot = transportSnapshot(2, "route-v2", "env:PROVIDER_MISSING")
	retained, err := manager.routerFor(context.Background(), workload)
	if err != nil {
		t.Fatalf("known-good fallback error = %v", err)
	}
	candidates, _, err = retained.candidatesFor(workload, "safe-chat", "")
	if err != nil || candidates[0].ID != "route-v1" {
		t.Fatalf("broken candidate displaced known good: %#v err=%v", candidates, err)
	}
	status := manager.status("tenant-a", "PRODUCTION")
	if !status.Degraded || status.DesiredRevision != 2 || status.AppliedRevision != 1 || status.ErrorCode != "TRANSPORT_APPLY_FAILED" {
		t.Fatalf("degraded status = %#v", status)
	}
	if status.DesiredChecksum != source.snapshot.Checksum || status.AppliedChecksum != "checksum-v1" {
		t.Fatalf("desired/applied checksums = %#v", status)
	}
}

func TestTransportManagerAtomicallyAppliesNewValidRevision(t *testing.T) {
	now := time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC)
	source := &mutableTransportSource{snapshot: transportSnapshot(1, "route-v1", "env:PROVIDER_V1")}
	resolver := mapSecretResolver{"env:PROVIDER_V1": "12345678", "env:PROVIDER_V2": "abcdefgh"}
	manager, err := newTransportManager(RuntimeConfig{
		Environment: "production", RequestTimeout: 2 * time.Minute, GovernanceRefresh: time.Second,
		MaxProviderBodyBytes: defaultMaxProviderBodyBytes, MaxSSEEventBytes: defaultMaxSSEEventBytes,
	}, source, resolver)
	if err != nil {
		t.Fatal(err)
	}
	manager.now = func() time.Time { return now }
	workload := Workload{TenantID: "tenant-a", Environment: "production", AllowedModels: map[string]struct{}{"safe-chat": {}}}
	if _, err := manager.routerFor(context.Background(), workload); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	source.snapshot = transportSnapshot(2, "route-v2", "env:PROVIDER_V2")
	updated, err := manager.routerFor(context.Background(), workload)
	if err != nil {
		t.Fatalf("second apply error = %v", err)
	}
	candidates, _, err := updated.candidatesFor(workload, "safe-chat", "")
	if err != nil || len(candidates) != 1 || candidates[0].ID != "route-v2" {
		t.Fatalf("updated candidates = %#v err=%v", candidates, err)
	}
	status := manager.status("tenant-a", "PRODUCTION")
	if status.Degraded || status.DesiredRevision != 2 || status.AppliedRevision != 2 || status.DesiredChecksum != source.snapshot.Checksum || status.AppliedChecksum != source.snapshot.Checksum {
		t.Fatalf("status = %#v", status)
	}
}

func TestTransportManagerRejectsAliasWithoutEnabledRoute(t *testing.T) {
	snapshot := transportSnapshot(1, "route-v1", "env:PROVIDER_V1")
	snapshot.Definition.Providers[0].State = ProviderStateSuspended
	source := &mutableTransportSource{snapshot: snapshot}
	manager, err := newTransportManager(RuntimeConfig{
		Environment: "production", RequestTimeout: 2 * time.Minute,
		MaxProviderBodyBytes: defaultMaxProviderBodyBytes, MaxSSEEventBytes: defaultMaxSSEEventBytes,
	}, source, mapSecretResolver{"env:PROVIDER_V1": "12345678"})
	if err != nil {
		t.Fatal(err)
	}
	workload := Workload{TenantID: "tenant-a", Environment: "production", AllowedModels: map[string]struct{}{"safe-chat": {}}}
	if _, err := manager.routerFor(context.Background(), workload); err == nil {
		t.Fatal("transport with no enabled route unexpectedly applied")
	}
	status := manager.status("tenant-a", "PRODUCTION")
	if !status.Degraded || status.DesiredRevision != 1 || status.AppliedRevision != 0 || status.ErrorCode != "TRANSPORT_APPLY_FAILED" {
		t.Fatalf("failed first apply status = %#v", status)
	}
}
