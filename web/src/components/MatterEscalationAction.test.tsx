import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { MatterAggregate } from "../types";
import { MatterActionsPanel } from "./MatterActionsPanel";

describe("failed outcome escalation Action", () => {
  it("shows the resolved escalation owner and an executable status operation", () => {
    const aggregate: MatterAggregate = {
      type_label: "Audit finding", status_label: "Decision needed", next_action: "Direct corrective work",
      matter: { id: "matter-1", tenant_id: "bank", reference: "MAT-1", type: "AUDIT_FINDING", status: "DECISION_REQUIRED", priority: 4, title: "Resolve access exceptions", summary: "One unsupported entry remains.", scope: {}, known_facts: {}, missing_facts: [], contradictions: [], created_at: "2026-08-26T10:00:00Z", updated_at: "2026-08-26T10:00:00Z", version: 7 },
      links: [], decisions: [], verification_contracts: [], verification_results: [], response_packages: [], closure: { ready: false, reasons: ["One outcome check did not pass."] },
      actions: [{ id: "escalation-action", title: "Direct corrective work for failed outcome", description: "Review the failed outcome check, direct the required corrective work and assign the next action.", owner_principal_id: "hidden-owner-id", required_responsibility: "ESCALATION_OWNER", status: "PLANNED", version: 1 }],
    };
    render(<MatterActionsPanel
      aggregate={aggregate}
      operations={[{
        command: "matter.action.transition", subresource_id: "escalation-action", label: "Update action status", responsibility: "ESCALATION_OWNER", can_act: true,
        assigned_to: { id: "hidden-owner-id", display_name: "Operational Risk Director", kind: "PERSON", role: "Operational Risk" }, allowed_targets: ["IN_PROGRESS", "BLOCKED", "CANCELLED"], reason: "You hold the current escalation responsibility.",
      }]}
      responsibleParties={[{ scope: "ACTION", subresource_id: "escalation-action", responsibility: "ESCALATION_OWNER", display_name: "Operational Risk Director", kind: "PERSON" }]}
      onUpdated={vi.fn()}
      onReload={vi.fn()}
    />);

    expect(screen.getByText("Escalation owner:")).toBeTruthy();
    expect(screen.getByText("Operational Risk Director")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Update status for Direct corrective work for failed outcome" })).toBeTruthy();
    expect(screen.queryByText("hidden-owner-id")).toBeNull();
  });
});
