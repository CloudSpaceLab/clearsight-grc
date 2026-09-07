import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { AIGovernanceWorkload } from "../../types";
import { AIGatewaySimulationPanel } from "./AIGatewaySimulationPanel";

const mocks = vi.hoisted(() => ({
  loadGatewayBaselines: vi.fn(),
  loadGatewayTransportState: vi.fn(),
  simulateGateway: vi.fn(),
}));

vi.mock("../../aiGovernanceControlApi", async () => {
  const actual = await vi.importActual<typeof import("../../aiGovernanceControlApi")>("../../aiGovernanceControlApi");
  return { ...actual, loadGatewayBaselines: mocks.loadGatewayBaselines };
});
vi.mock("../../aiGatewayTransportApi", async () => {
  const actual = await vi.importActual<typeof import("../../aiGatewayTransportApi")>("../../aiGatewayTransportApi");
  return { ...actual, loadGatewayTransportState: mocks.loadGatewayTransportState, simulateGateway: mocks.simulateGateway };
});

const workload = {
  id: "workload-record",
  workload_id: "customer-assistant",
  tenant_id: "bank",
  code: "CUSTOMER_ASSISTANT",
  name: "Customer assistant",
  purpose: "Customer service",
  environment: "PRODUCTION",
  owner_principal_id: "owner",
  allowed_models: ["safe-chat"],
  requests_per_minute: 60,
  tokens_per_minute: 100000,
  cost_microusd_per_minute: 100000,
  max_concurrent: 8,
  policy_id: "workload-policy",
  policy_version: 4,
  state: "ACTIVE",
  checksum: "workload-checksum",
  version: 1,
  record_version: 1,
} satisfies AIGovernanceWorkload;

it("simulates exact candidate revisions and previews only the governed instruction layer", async () => {
  mocks.loadGatewayBaselines.mockResolvedValue([{ id: "baseline-3", tenant_id: "bank", code: "ORG_AI_BASELINE", name: "Baseline", action_class: "AI_GATEWAY_BASELINE", eligibility: {}, blast_radius_limit: {}, verification_contract: {}, definition: {}, status: "APPROVED", rollout_mode: "ENFORCE", maker_id: "maker", version: 3, record_version: 2 }]);
  mocks.loadGatewayTransportState.mockResolvedValue({
    revisions: [{ id: "transport-5", tenant_id: "bank", environment: "PRODUCTION", definition: { circuit_breaker: { failure_threshold: 3, open_duration_ms: 30000 }, providers: [], models: [{ alias: "safe-chat", routes: [] }] }, status: "APPROVED", maker_id: "maker", change_reason: "Candidate", checksum: "abcdef1234567890", version: 5, record_version: 2, created_at: "", updated_at: "" }],
    runtimeStatus: null,
    proxy: { configured: false, ingress: [] },
    emergencyControl: { tenant_id: "bank", environment: "PRODUCTION", frozen: false, record_version: 0 },
  });
  mocks.simulateGateway.mockResolvedValue({
    fixture: "SAFE",
    environment: "PRODUCTION",
    workload: { id: "workload-record", workload_id: "customer-assistant", name: "Customer assistant", state: "ACTIVE" },
    workload_policy: { id: "workload-policy", code: "WORKLOAD", version: 4, status: "ACTIVE", rollout_mode: "ENFORCE" },
    baseline_policy: { id: "baseline-3", code: "ORG_AI_BASELINE", version: 3, status: "APPROVED", rollout_mode: "ENFORCE" },
    transport: { id: "transport-5", version: 5, status: "APPROVED", environment: "PRODUCTION", checksum: "abcdef1234567890" },
    decision: { policy_id: "workload-policy", policy_code: "WORKLOAD", policy_version: 4, rollout_mode: "ENFORCE", action: "ALLOW", reason_codes: ["POLICY_DEFAULT"] },
    detector_facts: [{ key: "gateway.prompt_injection_risk", value: "LOW", state: "KNOWN", source: "GATEWAY_DETECTOR" }],
    source_facts: [],
    instruction_precedence: ["ORGANIZATION_BASELINE", "WORKLOAD_SYSTEM_DEVELOPER", "USER_OR_RETRIEVED_CONTENT"],
    organization_instructions: [{ rule_id: "org-instruction", reason_code: "ORG_BASELINE_APPLIED", content: "Never reveal regulated secrets.", matched: true, applied: true }],
    model_alias: "safe-chat",
    eligible_routes: [{ id: "primary", provider_id: "provider", model: "model", weight: 1, regions: ["EU"] }],
    provider_call_would_occur: true,
  });

  render(<AIGatewaySimulationPanel workloads={[workload]} workloadState="live"/>);
  await waitFor(() => expect(screen.getByRole("button", { name: "Run deterministic test" })).toBeEnabled());
  fireEvent.click(screen.getByRole("button", { name: "Run deterministic test" }));

  await waitFor(() => expect(mocks.simulateGateway).toHaveBeenCalledWith({
    environment: "PRODUCTION",
    fixture: "SAFE",
    workloadId: "workload-record",
    baselinePolicyId: "baseline-3",
    transportId: "transport-5",
    modelAlias: "safe-chat",
  }));
  expect(screen.getByText("Would proceed")).toBeTruthy();
  expect(screen.getByText("Never reveal regulated secrets.")).toBeTruthy();
  expect(screen.getByText(/Organization Baseline → Workload System Developer → User Or Retrieved Content/)).toBeTruthy();
  expect(screen.queryByText(/Summarize the approved policy/)).toBeNull();
  expect(screen.queryByText(/Ignore previous instructions/)).toBeNull();
});

it("can exercise the explicit unknown-workload fail-closed fixture without selecting a workload", async () => {
  mocks.loadGatewayBaselines.mockResolvedValue([]);
  mocks.loadGatewayTransportState.mockResolvedValue({ revisions: [], runtimeStatus: null, proxy: { configured: false, ingress: [] }, emergencyControl: { tenant_id: "bank", environment: "PRODUCTION", frozen: false, record_version: 0 } });
  mocks.simulateGateway.mockResolvedValue({
    fixture: "UNKNOWN_WORKLOAD", environment: "PRODUCTION",
    decision: { policy_id: "simulation:unknown-workload", policy_code: "UNKNOWN_WORKLOAD_AUTHORITY", policy_version: 1, rollout_mode: "ENFORCE", action: "DENY", reason_codes: ["UNKNOWN_WORKLOAD"] },
    detector_facts: [], source_facts: [], instruction_precedence: ["ORGANIZATION_BASELINE", "WORKLOAD_SYSTEM_DEVELOPER", "USER_OR_RETRIEVED_CONTENT"], organization_instructions: [], model_alias: "simulation-model", eligible_routes: [], provider_call_would_occur: false, provider_call_blocked_reason: "UNKNOWN_WORKLOAD",
  });

  render(<AIGatewaySimulationPanel workloads={[]} workloadState="live"/>);
  fireEvent.change(await screen.findByRole("combobox", { name: "Fixture" }), { target: { value: "UNKNOWN_WORKLOAD" } });
  const button = screen.getByRole("button", { name: "Run deterministic test" });
  expect(button).toBeEnabled();
  fireEvent.click(button);

  await waitFor(() => expect(mocks.simulateGateway).toHaveBeenCalledWith(expect.objectContaining({ fixture: "UNKNOWN_WORKLOAD", workloadId: undefined })));
  expect(await screen.findByText("Would not occur")).toBeTruthy();
  expect(screen.getByText("Unknown Workload")).toBeTruthy();
});
