import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { AIGovernancePolicy, AIGovernanceWorkload } from "../../types";
import { AIGatewayExceptionControl } from "./AIGatewayExceptionControl";

const mocks = vi.hoisted(() => ({
  loadContext: vi.fn(),
  loadGatewayBaselines: vi.fn(),
  createShadow: vi.fn(),
  createEnforcement: vi.fn(),
  transition: vi.fn(),
}));

vi.mock("../../api", async () => {
  const actual = await vi.importActual<typeof import("../../api")>("../../api");
  return { ...actual, loadContext: mocks.loadContext };
});
vi.mock("../../aiGovernanceControlApi", async () => {
  const actual = await vi.importActual<typeof import("../../aiGovernanceControlApi")>("../../aiGovernanceControlApi");
  return { ...actual, loadGatewayBaselines: mocks.loadGatewayBaselines };
});
vi.mock("../../aiGatewayExceptionApi", async () => {
  const actual = await vi.importActual<typeof import("../../aiGatewayExceptionApi")>("../../aiGatewayExceptionApi");
  return {
    ...actual,
    createGatewayExceptionShadowDraft: mocks.createShadow,
    createGatewayExceptionEnforcementRevision: mocks.createEnforcement,
    transitionGatewayException: mocks.transition,
  };
});

const baseline = {
  id: "baseline-3",
  tenant_id: "bank",
  code: "ORG_AI_BASELINE",
  name: "Organization baseline",
  action_class: "AI_GATEWAY_BASELINE",
  eligibility: {},
  blast_radius_limit: {},
  verification_contract: {},
  definition: {
    default_action: "ALLOW",
    rules: [
      { id: "org-instruction", action: "ALLOW", reason_code: "ORG_BASELINE_APPLIED", obligations: [{ code: "ORG_INSTRUCTION", detail: "Never reveal secrets." }] },
      { id: "prompt-injection-high", action: "DENY", reason_code: "PROMPT_INJECTION_HIGH" },
      { id: "approval-rule", action: "REQUIRE_APPROVAL", reason_code: "SENSITIVE_ACTION_APPROVAL" },
    ],
  },
  status: "ACTIVE",
  rollout_mode: "ENFORCE",
  maker_id: "baseline-maker",
  version: 3,
  record_version: 4,
};

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

function configureContext(actor = "maker-a") {
  mocks.loadContext.mockResolvedValue({ actor: { id: actor }, tenant: { id: "bank" }, capabilities: { config_write: true } });
  mocks.loadGatewayBaselines.mockResolvedValue([baseline]);
}

it("creates a bounded Shadow exception from exact baseline rules and workloads", async () => {
  configureContext();
  mocks.createShadow.mockResolvedValue({ id: "exception-1" });
  const onChanged = vi.fn();
  render(<AIGatewayExceptionControl policies={[]} policyState="live" workloads={[workload]} workloadState="live" onChanged={onChanged}/>);

  expect(await screen.findByText("Organization baseline · v3")).toBeTruthy();
  expect(screen.queryByText("org-instruction")).toBeNull();
  expect(screen.getByText("prompt-injection-high")).toBeTruthy();
  expect(screen.getByText("approval-rule")).toBeTruthy();

  const workloadChoice = screen.getByText("Customer assistant").closest("label");
  const denyChoice = screen.getByText("prompt-injection-high").closest("label");
  if (!workloadChoice || !denyChoice) throw new Error("Expected bounded exception choices");
  fireEvent.click(within(workloadChoice).getByRole("checkbox"));
  fireEvent.click(within(denyChoice).getByRole("checkbox"));
  fireEvent.change(screen.getByRole("textbox", { name: "Justification" }), { target: { value: "Temporary controlled validation for ticket INC-42" } });

  const submit = screen.getByRole("button", { name: "Create Shadow exception" }) as HTMLButtonElement;
  await waitFor(() => expect(submit.disabled).toBe(false));
  fireEvent.click(submit);

  await waitFor(() => expect(mocks.createShadow).toHaveBeenCalledTimes(1));
  const input = mocks.createShadow.mock.calls[0][0];
  expect(input.name).toBe("Temporary AI baseline exception");
  expect(input.baseline.id).toBe("baseline-3");
  expect(input.baseline.version).toBe(3);
  expect(input.workloadRecordIds).toEqual(["workload-record"]);
  expect(input.environments).toEqual(["PRODUCTION"]);
  expect(input.waivedRuleIds).toEqual(["prompt-injection-high"]);
  expect(input.justification).toBe("Temporary controlled validation for ticket INC-42");
  expect(new Date(input.effectiveUntil).getTime()).toBeGreaterThan(Date.now());
  expect(onChanged).toHaveBeenCalledOnce();
});

it("enforces maker-checker lifecycle for an exception revision", async () => {
  configureContext("checker-b");
  const exception = {
    id: "exception-pending",
    tenant_id: "bank",
    code: "ORG_AI_BASELINE_EXCEPTION:ticket-42",
    name: "Ticket 42",
    action_class: "AI_GATEWAY_BASELINE_EXCEPTION",
    eligibility: {
      target_baseline_id: "baseline-3",
      target_baseline_version: 3,
      workload_record_ids: ["workload-record"],
      environments: ["PRODUCTION"],
      waived_rule_ids: ["prompt-injection-high"],
      justification: "Controlled test",
    },
    blast_radius_limit: {},
    verification_contract: {},
    rollout_mode: "SHADOW",
    checksum: "exception-checksum",
    status: "PENDING_APPROVAL",
    maker_id: "maker-a",
    effective_until: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
    version: 1,
    record_version: 2,
  } satisfies AIGovernancePolicy;
  mocks.transition.mockResolvedValue({ ...exception, status: "APPROVED", checker_id: "checker-b", record_version: 3 });

  render(<AIGatewayExceptionControl policies={[exception]} policyState="live" workloads={[workload]} workloadState="live"/>);
  const approve = await screen.findByRole("button", { name: "Approve" });
  fireEvent.click(approve);
  await waitFor(() => expect(mocks.transition).toHaveBeenCalledWith("exception-pending", "approve", 2));
  expect(screen.queryByText("Awaiting independent checker")).toBeNull();
});
