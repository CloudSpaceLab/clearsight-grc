import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TodayInterventions } from "./TodayInterventions";
import type { AttentionItem, Readiness } from "../types";

const item: AttentionItem = {
  id: "intervention-1",
  type: "REGULATORY_CHANGE",
  title: "Review digital-channel obligations",
  why_now: "A governing source changed.",
  scope: "Digital Channels",
  state: "Applicability review",
  evidence: "Official source verified",
  owner: "Regulatory Compliance",
  due_at: "2026-08-09T12:00:00Z",
  primary_action: "Review proposed obligations",
  action_target_type: "MATTER",
  action_target_id: "matter-1",
  intervention_class: "REVIEW",
  material_conclusion: "Seven provisions may change current obligations.",
  recommendation: { proposed_action: "Review proposed obligations", rationale: "Human applicability review is required." },
};

const readiness: Readiness = {
  tenant_id: "bank-demo",
  status: "AT_RISK",
  baseline_known: true,
  generated_at: "2026-08-06T15:30:00Z",
  dimensions: { current: 18, aging: 1, at_risk: 1, unknown: 0, blocked_routing: 0, pending_human: 1 },
  active_drifts: [],
  recommended_actions: ["Review the changed requirement."],
};

describe("TodayInterventions", () => {
  it("foregrounds the human gate and keeps continuous checks collapsed", () => {
    const onOpen = vi.fn();
    render(<TodayInterventions items={[item]} connection="live" readiness={readiness} readinessState="live" onOpenItem={onOpen}/>);

    expect(screen.getByRole("heading", { name: "1 item requires your action" })).toBeTruthy();
    expect(screen.getByText("Seven provisions may change current obligations.")).toBeTruthy();
    expect(screen.getByText("Review proposed obligations")).toBeTruthy();
    const continuousChecks = screen.getByText("Continuous checks").closest("details") as HTMLDetailsElement | null;
    expect(continuousChecks?.open).toBe(false);

    fireEvent.click(screen.getByRole("button", { name: "Review and act" }));
    expect(onOpen).toHaveBeenCalledWith(item);
  });

  it("does not claim an empty queue while assigned work is still loading", () => {
    render(<TodayInterventions items={[]} connection="loading" readiness={null} readinessState="loading" onOpenItem={vi.fn()}/>);

    expect(screen.getByRole("heading", { name: "Loading assigned work" })).toBeTruthy();
    expect(screen.getByText("Loading assigned work…")).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "No assigned items" })).toBeNull();
    expect(screen.queryByText("0 items require your action")).toBeNull();
  });
});
