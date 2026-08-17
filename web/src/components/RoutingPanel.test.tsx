import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { RoutingPanel } from "../AppViews";
import type { AttentionItem } from "../types";

const item: AttentionItem = {
  id: "work-1", type: "REGULATORY_CHANGE", title: "Authorize temporary exception", why_now: "Authority is required.", scope: "Retail Payments", state: "Decision needed", evidence: "Independent review complete", owner: "Operational Risk", due_at: "2026-08-09T12:00:00Z", primary_action: "Authorize exception", action_target_type: "MATTER", action_target_id: "matter-1", authority: { responsibility: "AUTHORIZER", decision_type: "RISK_ACCEPTANCE", materiality: 5 },
};

const resolution = {
  principal: { id: "principal-1", display_name: "Ada Okafor", kind: "HUMAN", role: "Entity risk approver" },
  candidate_principals: [
    { id: "principal-1", display_name: "Ada Okafor", kind: "HUMAN", role: "Entity risk approver" },
    { id: "principal-2", display_name: "Tunde Bello", kind: "HUMAN", role: "Deputy risk approver" },
  ],
  strategy: "ANY_OF",
  rule_id: "rule-1",
  policy_version: "routing-v7",
  explanation: "Selected from the current entity-scoped authority policy after delegation and conflict checks.",
};

describe("RoutingPanel", () => {
  it("renders exact server-returned authority facts and candidate semantics", () => {
    render(<RoutingPanel resolution={resolution} item={item} legalEntityName="Example Bank Nigeria" state="live"/>);

    expect(screen.getByRole("heading", { name: "Authority for this item" })).toBeTruthy();
    expect(screen.getByText("Authorize temporary exception")).toBeTruthy();
    expect(screen.getAllByText("Ada Okafor")).toHaveLength(2);
    expect(screen.getByText("Tunde Bello")).toBeTruthy();
    expect(screen.getByText(/Any Of/)).toBeTruthy();
    expect(screen.getByText("routing-v7")).toBeTruthy();
    expect(screen.getByText(resolution.explanation)).toBeTruthy();
    expect(screen.getByText("Approval details")).toBeTruthy();
    expect(screen.queryByText("Critical · Executive approval")).toBeNull();
    expect(screen.queryByText("Control Assurance")).toBeNull();
  });

  it("does not invent an approver or sequence when resolution is unavailable", () => {
    render(<RoutingPanel resolution={null} item={item} legalEntityName="Example Bank Nigeria" state="unavailable"/>);

    expect(screen.getByRole("heading", { name: "Authority resolution is unavailable" })).toBeTruthy();
    expect(screen.queryByText("CRO")).toBeNull();
  });

  it("does not leak authority details when the current role is forbidden", () => {
    render(<RoutingPanel resolution={null} item={item} legalEntityName="Example Bank Nigeria" state="forbidden"/>);
    expect(screen.getByRole("heading", { name: "Authority details are restricted" })).toBeTruthy();
    expect(screen.queryByText("routing-v7")).toBeNull();
  });
});
