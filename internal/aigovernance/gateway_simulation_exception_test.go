package aigovernance

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func TestGatewaySimulationAppliesAndAttributesExactBaselineException(t *testing.T) {
	service, repo := simulationFixtureService(t, aigateway.RolloutEnforce, GatewayTransportActive)
	now := time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	expires := now.Add(2 * time.Hour)
	scope, _ := json.Marshal(GatewayBaselineExceptionScope{
		TargetBaselineID: "baseline", TargetBaselineVersion: 3,
		WorkloadRecordIDs: []string{"workload-record"}, Environments: []string{"PRODUCTION"},
		WaivedRuleIDs: []string{"prompt-injection-high"}, Justification: "Controlled hostile-content validation",
	})
	if _, err := repo.CreatePolicy(context.Background(), Policy{
		ID: "exception-simulation", TenantID: "tenant-a", Code: aigateway.GatewayBaselineExceptionCodeRoot + ":simulation",
		Name: "Simulation exception", ActionClass: aigateway.GatewayBaselineExceptionActionClass,
		Eligibility: scope, Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
		Status: "ACTIVE", RolloutMode: aigateway.RolloutEnforce, EffectiveUntil: &expires, Version: 2, RecordVersion: 1,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.SimulateGateway(context.Background(), GatewaySimulationInput{
		TenantID: "tenant-a", Environment: "PRODUCTION", Fixture: GatewaySimulationHostileUntrustedContent,
		WorkloadRecordID: "workload-record", BaselinePolicyID: "baseline", TransportID: "transport",
	})
	if err != nil {
		t.Fatalf("SimulateGateway() error = %v", err)
	}
	if result.Decision.Action != aigateway.DecisionAllow || !result.ProviderCallWouldOccur {
		t.Fatalf("decision = %#v call=%v reason=%q", result.Decision, result.ProviderCallWouldOccur, result.ProviderCallBlockedReason)
	}
	if len(result.BaselineExceptions) != 1 || result.BaselineExceptions[0].ID != "exception-simulation" || result.BaselineExceptions[0].Version != 2 {
		t.Fatalf("baseline exceptions = %#v", result.BaselineExceptions)
	}
	if !containsString(result.Decision.ReasonCodes, "ORG_BASELINE_APPLIED") {
		t.Fatalf("reason codes = %#v, want surviving baseline instruction rule", result.Decision.ReasonCodes)
	}
	if len(repo.receipts) != 0 {
		t.Fatalf("simulation persisted %d receipt(s)", len(repo.receipts))
	}
}
