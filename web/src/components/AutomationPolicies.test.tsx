import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { AutomationPolicy } from "../types";
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
  status: "ACTIVE",
  version: 2,
};

describe("AutomationPolicies", () => {
  it("shows policy boundaries without claiming execution", () => {
    render(<AutomationPolicies policies={[policy]} state="live"/>);

    expect(screen.getByText("Low-impact evidence refresh")).toBeTruthy();
    expect(screen.getByText(/Review execution history for completed actions and outcome checks/)).toBeTruthy();
    fireEvent.click(screen.getByText("View limits"));
    expect(screen.getByText("25")).toBeTruthy();
    expect(screen.getByText("Source Recheck")).toBeTruthy();
  });

  it("fails quiet when policies are unavailable", () => {
    render(<AutomationPolicies policies={[]} state="unavailable"/>);
    expect(screen.getByRole("heading", { name: "Automation policies could not be loaded" })).toBeTruthy();
    expect(screen.getByText(/Try again before reviewing or changing automated actions/)).toBeTruthy();
  });
});
