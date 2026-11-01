import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AutomationPolicy, AutomationPolicyPage } from "../types";
import { AutomationPolicies } from "./AutomationPolicies";

const policy: AutomationPolicy = {
  id: "policy-1",
  tenant_id: "bank-demo",
  code: "EVIDENCE-REFRESH",
  name: "Low-impact evidence refresh",
  action_class: "REVERSIBLE_WRITE",
  eligibility: { materiality_max: 2, sensitivity: "INTERNAL" },
  blast_radius_limit: { max_records: 25, external_side_effects: false },
  verification_contract: { method: "source_recheck", required: true },
  rollout_mode: "SHADOW",
  status: "ACTIVE",
  version: 2,
};
const page: AutomationPolicyPage = { items: [policy], tenant_id: "bank-demo", scope: "TENANT", revision_basis: "LATEST_PER_CODE", limit: 200, returned_count: 1, truncated: false };
const route = { tab: "automations" as const, q: "" };

describe("AutomationPolicies", () => {
  it("shows the selected revision and recorded guardrails without claiming an executed outcome", () => {
    render(<AutomationPolicies page={page} state="live" route={route} onRouteChange={vi.fn()} onRefresh={vi.fn()}/>);
    const detail = screen.getByRole("complementary", { name: "Selected automation policy" });
    expect(within(detail).getByRole("heading", { name: "Low-impact evidence refresh" })).toBeTruthy();
    expect(within(detail).getByText("25")).toBeTruthy();
    expect(within(detail).getByText("source_recheck")).toBeTruthy();
    expect(within(detail).getByText(/shadow/i)).toBeTruthy();
    expect(screen.queryByText(/actions executed|successful runs|outcome verified/i)).toBeNull();
    expect(screen.queryByRole("button", { name: /^(activate|pause|edit|simulate|create)/i })).toBeNull();
  });

  it("uses the route callback to inspect a returned policy instead of changing its lifecycle", () => {
    const onRouteChange = vi.fn();
    render(<AutomationPolicies page={page} state="live" route={route} onRouteChange={onRouteChange} onRefresh={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Inspect Low-impact evidence refresh" }));
    expect(onRouteChange).toHaveBeenCalledWith(expect.objectContaining({ tab: "automations", policy: "policy-1", q: "" }));
  });

  it("hides any retained policy data while unavailable and offers a real retry", () => {
    const onRefresh = vi.fn();
    render(<AutomationPolicies page={page} state="unavailable" route={route} onRouteChange={vi.fn()} onRefresh={onRefresh}/>);
    expect(screen.getByRole("heading", { name: "Automation policies could not be loaded" })).toBeTruthy();
    expect(screen.queryByText("Low-impact evidence refresh")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /retry|refresh/i }));
    expect(onRefresh).toHaveBeenCalledOnce();
  });

  it("does not turn a missing guardrail field into a claim that an action has no limits", () => {
    const incomplete = { ...policy, eligibility: {}, blast_radius_limit: {}, verification_contract: {} };
    render(<AutomationPolicies page={{ ...page, items: [incomplete] }} state="live" route={route} onRouteChange={vi.fn()} onRefresh={vi.fn()}/>);
    const detail = screen.getByRole("complementary", { name: "Selected automation policy" });
    expect(within(detail).queryByText(/no additional condition|no limits|unlimited/i)).toBeNull();
    for (const name of ["Eligible when", "Maximum scope", "Outcome check"]) {
      const group = within(detail).getByRole("heading", { name }).closest("section")!;
      expect(within(group).getByText(/not recorded|not provided|not set/i)).toBeTruthy();
    }
  });
});
