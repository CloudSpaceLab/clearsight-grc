import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AIGovernancePanel } from "./AIGovernancePanel";

const policy = { id: "p1", tenant_id: "bank", code: "AI-CONTROL", name: "AI control", action_class: "MODEL_REQUEST", eligibility: {}, blast_radius_limit: {}, verification_contract: {}, rollout_mode: "SHADOW", checksum: "abc", status: "ACTIVE", maker_id: "maker", checker_id: "checker", version: 1, record_version: 4 };
const workload = { id: "w1", workload_id: "agent", tenant_id: "bank", code: "AGENT", name: "Payments agent", purpose: "payments", environment: "PROD", owner_principal_id: "owner", allowed_models: ["general"], requests_per_minute: 60, tokens_per_minute: 10000, cost_microusd_per_minute: 100000, max_concurrent: 4, policy_id: "p1", policy_version: 1, state: "ACTIVE", checksum: "def", version: 1, record_version: 4 };

describe("AIGovernancePanel", () => {
  it("shows compact policy/workload truth without creating a dashboard wall", () => {
    render(<AIGovernancePanel policies={[policy]} policyState="live" workloads={[workload]} workloadState="live"/>);
    expect(screen.getByRole("heading", { name: "Governed model access" })).toBeTruthy();
    expect(screen.getByText("Payments agent")).toBeTruthy();
    expect(screen.getAllByText("Shadow").length).toBeGreaterThan(0);
    expect(screen.getByText(/Routine allowed traffic does not create Today work/)).toBeTruthy();
  });

  it("keeps degraded state explicit", () => {
    render(<AIGovernancePanel policies={[]} policyState="unavailable" workloads={[]} workloadState="unavailable"/>);
    expect(screen.getByText("AI policies are unavailable")).toBeTruthy();
    expect(screen.getByText("AI workloads are unavailable")).toBeTruthy();
  });
});
