import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TodayInterventions } from "./TodayInterventions";
import type { AttentionItem, Readiness } from "../types";

const item: AttentionItem = {
  id: "intervention-1", type: "REGULATORY_CHANGE", title: "Review digital-channel obligations", why_now: "A governing source changed.", scope: "Digital Channels", state: "Applicability review", evidence: "Official source verified", owner: "Regulatory Compliance", due_at: "2026-08-09T12:00:00Z", primary_action: "Review proposed obligations", action_target_type: "MATTER", action_target_id: "matter-1", intervention_class: "REVIEW", material_conclusion: "Seven provisions may change current obligations.", recommendation: { proposed_action: "Review proposed obligations", rationale: "Human applicability review is required." },
};

const readiness: Readiness = {
  tenant_id: "bank-demo", status: "AT_RISK", baseline_known: true, generated_at: "2026-08-06T15:30:00Z", dimensions: { current: 18, aging: 1, at_risk: 1, unknown: 0, blocked_routing: 0, pending_human: 1 }, active_drifts: [], recommended_actions: ["Review the changed requirement."],
};

describe("TodayInterventions", () => {
  it("uses Today as the practical work surface and keeps status checks collapsed", () => {
    const onOpen = vi.fn();
    render(<TodayInterventions items={[item]} connection="live" readiness={readiness} readinessState="live" onOpenItem={onOpen}/>);
    expect(screen.getByText("Today", { selector: ".eyebrow" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "1 item needs your action" })).toBeTruthy();
    expect(screen.getByText("Reviews, approvals and evidence requests assigned to you.")).toBeTruthy();
    expect(screen.getByText("Seven provisions may change current obligations.")).toBeTruthy();
    expect(screen.getByText("Recommended action")).toBeTruthy();
    const statusChecks = screen.getByText("Status checks").closest("details") as HTMLDetailsElement | null;
    expect(statusChecks?.open).toBe(false);
    fireEvent.click(screen.getByRole("button", { name: "Open issue" }));
    expect(onOpen).toHaveBeenCalledWith(item);
  });

  it("opens assigned document work at the exact proposal without using the generic dispatcher", () => {
    const onOpen = vi.fn();
    const documentItem = {
      ...item,
      id: "document-review",
      type: "DOCUMENT_PROPOSAL",
      title: "Review imported proposal",
      recommendation: undefined,
      action_target_type: "DOCUMENT_IMPORT",
      action_target_id: "document 1",
      action_target_sub_id: "proposal 1",
      authority: { responsibility: "REVIEWER", decision_type: "document.proposal.review", materiality: 3 },
    } as AttentionItem & { action_target_sub_id: string };
    window.history.replaceState(null, "", "#today");

    render(<TodayInterventions items={[documentItem]} connection="live" readiness={readiness} readinessState="live" onOpenItem={onOpen} onInspectAuthority={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: "Open proposal" }));

    expect(window.location.hash).toBe("#imports/document%201/proposal%201");
    expect(onOpen).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Check authority" })).toBeNull();
  });

  it("does not relabel an ordinary workflow action as prepared or recommended work", () => {
    render(<TodayInterventions items={[{ ...item, recommendation: undefined }]} connection="live" readiness={readiness} readinessState="live" onOpenItem={vi.fn()}/>);
    expect(screen.getByText("Next action")).toBeTruthy();
    expect(screen.queryByText("Prepared next step")).toBeNull();
    expect(screen.queryByText("Recommended action")).toBeNull();
  });

  it("shows governed verification context only as collapsed outcome-check detail", () => {
    const verificationItem: AttentionItem = {
      ...item,
      id: "verification-1",
      type: "MATTER_WORK",
      title: "Confirm restored ATM availability",
      recommendation: undefined,
      intervention_class: "VERIFICATION",
      primary_action: "Record outcome check",
      material_conclusion: "The observation period has completed and this outcome check has no recorded result.",
      evidence: "Outcome check ready",
      authority: { responsibility: "REVIEWER", decision_type: "matter.outcome.record", materiality: 5 },
      verification: { state: "Outcome check ready", expected_outcome: "ATM remains available for one hour after restoration.", method: "Independent outcome review", next_check_at: "2026-08-08T09:30:00Z" },
    };
    render(<TodayInterventions items={[verificationItem]} connection="live" readiness={readiness} readinessState="live" onOpenItem={vi.fn()}/>);
    expect(screen.getByText("Outcome check", { selector: ".intervention-kind" })).toBeTruthy();
    expect(screen.getByText("Record outcome check")).toBeTruthy();
    const detail = screen.getByText("Outcome check details").closest("details") as HTMLDetailsElement | null;
    expect(detail?.open).toBe(false);
    expect(screen.getByText("ATM remains available for one hour after restoration.")).toBeTruthy();
    expect(screen.queryByText("Recommended action")).toBeNull();
    expect(screen.queryByText("Prepared next step")).toBeNull();
  });

  it("labels external representation work without implying approval", () => {
    const responseItem: AttentionItem = { ...item, recommendation: undefined, intervention_class: "EXTERNAL_REPRESENTATION", primary_action: "Record acknowledgement" };
    render(<TodayInterventions items={[responseItem]} connection="live" readiness={readiness} readinessState="live" onOpenItem={vi.fn()}/>);
    expect(screen.getByText("External response", { selector: ".intervention-kind" })).toBeTruthy();
    expect(screen.queryByText("External approval")).toBeNull();
  });

  it("exposes item-scoped authority only when authority context exists", () => {
    const inspect = vi.fn();
    const authorized = { ...item, authority: { responsibility: "REVIEWER", materiality: 2 } };
    render(<TodayInterventions items={[authorized]} connection="live" readiness={readiness} readinessState="live" onOpenItem={vi.fn()} onInspectAuthority={inspect}/>);
    fireEvent.click(screen.getByRole("button", { name: "Check authority" }));
    expect(inspect).toHaveBeenCalledWith(authorized);
  });

  it("does not claim an empty Today list while assigned work is still loading", () => {
    render(<TodayInterventions items={[]} connection="loading" readiness={null} readinessState="loading" onOpenItem={vi.fn()}/>);
    expect(screen.getByRole("heading", { name: "Loading Today" })).toBeTruthy();
    expect(screen.getByText("Loading Today…")).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Nothing needs your action right now" })).toBeNull();
    expect(screen.queryByText("0 items need your action")).toBeNull();
  });

  it("does not turn an unknown baseline into a no-exception claim", () => {
    const unknownBaseline: Readiness = { ...readiness, baseline_known: false, dimensions: { current: 0, aging: 0, at_risk: 0, unknown: 0, blocked_routing: 0, pending_human: 0 } };
    render(<TodayInterventions items={[]} connection="live" readiness={unknownBaseline} readinessState="live" onOpenItem={vi.fn()}/>);
    expect(screen.getByText("Coverage is incomplete")).toBeTruthy();
    expect(screen.queryByText("No current exceptions recorded")).toBeNull();
  });
});
