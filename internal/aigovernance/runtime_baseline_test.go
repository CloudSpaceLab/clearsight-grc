package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func TestRuntimeProviderAttachesNewestActiveGatewayBaseline(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	workloadPolicy := Policy{
		ID: "workload-policy", TenantID: "tenant-a", Code: "WORKLOAD", Name: "Workload",
		ActionClass: "AI_WORKLOAD", Status: "ACTIVE", RolloutMode: aigateway.RolloutEnforce, Version: 1, RecordVersion: 1,
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
	}
	baselineV1 := Policy{
		ID: "baseline-v1", TenantID: "tenant-a", Code: aigateway.GatewayBaselinePolicyCode, Name: "Baseline v1",
		ActionClass: aigateway.GatewayBaselineActionClass, Status: "ACTIVE", RolloutMode: aigateway.RolloutShadow, Version: 1, RecordVersion: 1,
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
	}
	baselineV2 := baselineV1
	baselineV2.ID = "baseline-v2"
	baselineV2.Name = "Baseline v2"
	baselineV2.Version = 2
	if _, err := repo.CreatePolicy(ctx, workloadPolicy); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreatePolicy(ctx, baselineV1); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreatePolicy(ctx, baselineV2); err != nil {
		t.Fatal(err)
	}
	secret := "runtime-secret"
	digest := sha256.Sum256([]byte(secret))
	workload := Workload{
		ID: "workload-record", WorkloadID: "assistant", TenantID: "tenant-a", Code: "ASSISTANT", Name: "Assistant",
		Purpose: "test", Environment: "production", AllowedModels: []string{"chat"}, RequestsPerMinute: 1,
		TokensPerMinute: 1, CostMicroUSDPerMinute: 1, MaxConcurrent: 1, PolicyID: workloadPolicy.ID, PolicyVersion: 1,
		State: "ACTIVE", Version: 1, RecordVersion: 1, KeySHA256: hex.EncodeToString(digest[:]),
	}
	if _, err := repo.CreateWorkload(ctx, workload); err != nil {
		t.Fatal(err)
	}

	provider := NewRuntimeProvider(repo, nil)
	result, err := provider.Authenticate(ctx, "Bearer "+secret)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if result.Policy.Baseline == nil || result.Policy.Baseline.ID != baselineV2.ID || result.Policy.Baseline.Version != 2 {
		t.Fatalf("baseline = %#v, want newest active revision", result.Policy.Baseline)
	}
}

func TestRuntimeProviderBaselineCacheIsBoundedAndRefreshes(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	provider := NewRuntimeProvider(repo, nil)
	provider.now = func() time.Time { return now }
	provider.baselineTTL = 5 * time.Second

	first, found, err := provider.activeGatewayBaseline(ctx, "tenant-a")
	if err != nil || found || first.ID != "" {
		t.Fatalf("initial baseline = %#v found=%v err=%v", first, found, err)
	}
	baseline := Policy{
		ID: "baseline", TenantID: "tenant-a", Code: aigateway.GatewayBaselinePolicyCode, Name: "Baseline",
		ActionClass: aigateway.GatewayBaselineActionClass, Status: "ACTIVE", RolloutMode: aigateway.RolloutShadow,
		Version: 1, RecordVersion: 1, Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
	}
	if _, err := repo.CreatePolicy(ctx, baseline); err != nil {
		t.Fatal(err)
	}
	if cached, found, err := provider.activeGatewayBaseline(ctx, "tenant-a"); err != nil || found || cached.ID != "" {
		t.Fatalf("negative cache did not hold: %#v found=%v err=%v", cached, found, err)
	}
	now = now.Add(6 * time.Second)
	refreshed, found, err := provider.activeGatewayBaseline(ctx, "tenant-a")
	if err != nil || !found || refreshed.ID != baseline.ID {
		t.Fatalf("refreshed baseline = %#v found=%v err=%v", refreshed, found, err)
	}
}
