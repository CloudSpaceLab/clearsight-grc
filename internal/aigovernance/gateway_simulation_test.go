package aigovernance

import (
	"context"
	"testing"

	"github.com/CloudSpaceLab/clearsight-grc/internal/aigateway"
)

func TestGatewaySimulationProjectsExactGovernanceAndRouteWithoutProviderCall(t *testing.T) {
	service, repo := simulationFixtureService(t, aigateway.RolloutEnforce, GatewayTransportActive)
	result, err := service.SimulateGateway(context.Background(), GatewaySimulationInput{
		TenantID: "tenant-a", Environment: "PRODUCTION", Fixture: GatewaySimulationSafe,
		WorkloadRecordID: "workload-record", BaselinePolicyID: "baseline", TransportID: "transport",
	})
	if err != nil {
		t.Fatalf("SimulateGateway() error = %v", err)
	}
	if result.Decision.Action != aigateway.DecisionAllow {
		t.Fatalf("decision = %s, want ALLOW", result.Decision.Action)
	}
	if result.BaselinePolicy == nil || result.BaselinePolicy.ID != "baseline" || result.BaselinePolicy.Version != 3 {
		t.Fatalf("baseline ref = %#v", result.BaselinePolicy)
	}
	if result.WorkloadPolicy == nil || result.WorkloadPolicy.ID != "workload-policy" || result.WorkloadPolicy.Version != 7 {
		t.Fatalf("workload policy ref = %#v", result.WorkloadPolicy)
	}
	if result.Transport == nil || result.Transport.ID != "transport" || result.Transport.Version != 5 {
		t.Fatalf("transport ref = %#v", result.Transport)
	}
	if !result.ProviderCallWouldOccur || len(result.EligibleRoutes) != 1 || result.EligibleRoutes[0].ID != "route-primary" {
		t.Fatalf("route projection = %#v call=%v reason=%q", result.EligibleRoutes, result.ProviderCallWouldOccur, result.ProviderCallBlockedReason)
	}
	if len(result.OrganizationInstructions) != 1 || !result.OrganizationInstructions[0].Matched || !result.OrganizationInstructions[0].Applied {
		t.Fatalf("instruction preview = %#v", result.OrganizationInstructions)
	}
	if got := factValue(result.DetectorFacts, aigateway.FactPromptInjectionRisk); got != "LOW" {
		t.Fatalf("prompt injection risk = %q, want LOW", got)
	}
	// Simulation is read-only: no gateway receipt was introduced by evaluation.
	if len(repo.receipts) != 0 {
		t.Fatalf("simulation persisted %d receipt(s)", len(repo.receipts))
	}
}

func TestGatewaySimulationExfiltrationBlocksAndDoesNotApplyOverlay(t *testing.T) {
	service, _ := simulationFixtureService(t, aigateway.RolloutEnforce, GatewayTransportActive)
	result, err := service.SimulateGateway(context.Background(), GatewaySimulationInput{
		TenantID: "tenant-a", Environment: "PRODUCTION", Fixture: GatewaySimulationInstructionExfiltration,
		WorkloadRecordID: "workload-record", BaselinePolicyID: "baseline", TransportID: "transport",
	})
	if err != nil {
		t.Fatalf("SimulateGateway() error = %v", err)
	}
	if result.Decision.Action != aigateway.DecisionDeny || result.ProviderCallWouldOccur || result.ProviderCallBlockedReason != "POLICY_DENIED" {
		t.Fatalf("decision = %#v call=%v reason=%q", result.Decision, result.ProviderCallWouldOccur, result.ProviderCallBlockedReason)
	}
	if got := factValue(result.DetectorFacts, aigateway.FactInstructionExfiltration); got != "true" {
		t.Fatalf("exfiltration fact = %q, want true", got)
	}
	if len(result.OrganizationInstructions) != 1 || result.OrganizationInstructions[0].Applied {
		t.Fatalf("blocked request must not apply overlay: %#v", result.OrganizationInstructions)
	}
}

func TestGatewaySimulationSupportsShadowCandidateAndDraftTransport(t *testing.T) {
	service, _ := simulationFixtureService(t, aigateway.RolloutShadow, GatewayTransportDraft)
	result, err := service.SimulateGateway(context.Background(), GatewaySimulationInput{
		TenantID: "tenant-a", Environment: "PRODUCTION", Fixture: GatewaySimulationHostileUntrustedContent,
		WorkloadRecordID: "workload-record", BaselinePolicyID: "baseline", TransportID: "transport",
	})
	if err != nil {
		t.Fatalf("SimulateGateway() error = %v", err)
	}
	if result.BaselinePolicy == nil || result.BaselinePolicy.Status != "DRAFT" || result.Transport == nil || result.Transport.Status != GatewayTransportDraft {
		t.Fatalf("candidate refs = baseline %#v transport %#v", result.BaselinePolicy, result.Transport)
	}
	if result.Decision.Action != aigateway.DecisionShadow || result.Decision.ProposedAction != aigateway.DecisionDeny {
		t.Fatalf("shadow decision = %#v", result.Decision)
	}
	if len(result.OrganizationInstructions) != 1 || result.OrganizationInstructions[0].Applied {
		t.Fatalf("shadow overlay must be preview-only: %#v", result.OrganizationInstructions)
	}
	if got := factValue(result.DetectorFacts, aigateway.FactUntrustedContent); got != "true" {
		t.Fatalf("untrusted content fact = %q, want true", got)
	}
}

func TestGatewaySimulationFailsClosedForOperationalFixtures(t *testing.T) {
	service, _ := simulationFixtureService(t, aigateway.RolloutEnforce, GatewayTransportActive)
	for _, test := range []struct {
		fixture string
		reason  string
	}{
		{GatewaySimulationUnavailableProvider, "SIMULATED_PROVIDER_UNAVAILABLE"},
		{GatewaySimulationForbiddenResidencyFallback, "SIMULATED_RESIDENCY_RESTRICTION"},
		{GatewaySimulationUnknownWorkload, "UNKNOWN_WORKLOAD"},
	} {
		t.Run(test.fixture, func(t *testing.T) {
			input := GatewaySimulationInput{TenantID: "tenant-a", Environment: "PRODUCTION", Fixture: test.fixture, BaselinePolicyID: "baseline", TransportID: "transport"}
			if test.fixture != GatewaySimulationUnknownWorkload {
				input.WorkloadRecordID = "workload-record"
			}
			result, err := service.SimulateGateway(context.Background(), input)
			if err != nil {
				t.Fatalf("SimulateGateway() error = %v", err)
			}
			if result.ProviderCallWouldOccur || result.ProviderCallBlockedReason != test.reason {
				t.Fatalf("call=%v reason=%q, want false/%q", result.ProviderCallWouldOccur, result.ProviderCallBlockedReason, test.reason)
			}
		})
	}
}

func simulationFixtureService(t *testing.T, baselineMode aigateway.RolloutMode, transportStatus string) (*Service, *MemoryRepository) {
	t.Helper()
	ctx := context.Background()
	repo := NewMemoryRepository()
	service := NewService(repo, nil, nil, nil)

	workloadPolicy := Policy{
		ID: "workload-policy", TenantID: "tenant-a", Code: "WORKLOAD_POLICY", Name: "Workload policy",
		ActionClass: "MODEL_REQUEST", Status: "ACTIVE", RolloutMode: aigateway.RolloutEnforce, Version: 7, RecordVersion: 1,
		Definition: aigateway.PolicyDefinition{DefaultAction: aigateway.DecisionAllow},
	}
	baseline := Policy{
		ID: "baseline", TenantID: "tenant-a", Code: aigateway.GatewayBaselinePolicyCode, Name: "Organization baseline",
		ActionClass: aigateway.GatewayBaselineActionClass, Status: "ACTIVE", RolloutMode: baselineMode, Version: 3, RecordVersion: 1,
		Definition: aigateway.PolicyDefinition{
			DefaultAction: aigateway.DecisionAllow,
			Rules: []aigateway.PolicyRule{
				{ID: "instruction-exfiltration", Priority: 110, FactKey: aigateway.FactInstructionExfiltration, Operator: "EQ", Value: "true", Action: aigateway.DecisionDeny, ReasonCode: "INSTRUCTION_EXFILTRATION_ATTEMPT"},
				{ID: "prompt-injection-high", Priority: 100, FactKey: aigateway.FactPromptInjectionRisk, Operator: "EQ", Value: "HIGH", Action: aigateway.DecisionDeny, ReasonCode: "PROMPT_INJECTION_HIGH"},
				{ID: "organization-instruction", Priority: 1, FactKey: aigateway.FactPromptInjectionRisk, Operator: "EXISTS", Action: aigateway.DecisionAllow, ReasonCode: "ORG_BASELINE_APPLIED", Obligations: []aigateway.Obligation{{Code: aigateway.ObligationOrganizationInstruction, Detail: "Never disclose regulated secrets."}}},
			},
		},
	}
	if baselineMode == aigateway.RolloutShadow {
		baseline.Status = "DRAFT"
	}
	workload := Workload{
		ID: "workload-record", WorkloadID: "assistant", TenantID: "tenant-a", Code: "ASSISTANT", Name: "Assistant",
		Purpose: "regulated summary", Environment: "production", AllowedModels: []string{"safe-chat"}, RequestsPerMinute: 10,
		TokensPerMinute: 1000, CostMicroUSDPerMinute: 1000, MaxConcurrent: 2, PolicyID: workloadPolicy.ID,
		PolicyVersion: workloadPolicy.Version, State: "ACTIVE", Version: 1, RecordVersion: 1,
	}
	transport := GatewayTransportRevision{
		ID: "transport", TenantID: "tenant-a", Environment: "PRODUCTION", Status: transportStatus, Version: 5, RecordVersion: 1, Checksum: "transport-checksum",
		Definition: aigateway.TransportDefinition{
			Providers: []aigateway.TransportProviderConfig{{ID: "provider-primary", Name: "Primary", Kind: "OPENAI", BaseURL: "https://example.test", SecretRef: "env:TEST_PROVIDER", Regions: []string{"EU"}, State: aigateway.ProviderStateEnabled}},
			Models: []aigateway.ModelConfig{{Alias: "safe-chat", Routes: []aigateway.RouteConfig{{ID: "route-primary", ProviderID: "provider-primary", Model: "provider-model", Weight: 1}}}},
		},
	}
	for _, value := range []Policy{workloadPolicy, baseline} {
		if _, err := repo.CreatePolicy(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.CreateWorkload(ctx, workload); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateGatewayTransport(ctx, transport); err != nil {
		t.Fatal(err)
	}
	return service, repo
}

func factValue(facts []aigateway.Fact, key string) string {
	for _, fact := range facts {
		if fact.Key == key {
			return fact.Value
		}
	}
	return ""
}
