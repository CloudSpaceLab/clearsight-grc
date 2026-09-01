import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FormPolicyEditor } from "./FormPolicyEditor";

describe("FormPolicyEditor", () => {
  it("captures the exact form revision, blast radius and outcome check without actor fields", () => {
    const save = vi.fn();
    render(<FormPolicyEditor onCancel={() => undefined} onCreate={save}/>);

    fireEvent.change(screen.getByLabelText("Policy name"), { target: { value: "Review poor vendor scores" } });
    fireEvent.change(screen.getByLabelText("Policy code"), { target: { value: "poor-vendor-score" } });
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Create a review issue for an adverse completed vendor response." } });
    fireEvent.change(screen.getByLabelText("Form template ID"), { target: { value: "form-1" } });
    fireEvent.change(screen.getByLabelText("Form revision"), { target: { value: "4" } });
    fireEvent.change(screen.getByLabelText("Automation policy ID"), { target: { value: "automation-1" } });
    fireEvent.change(screen.getByLabelText("Automation policy revision"), { target: { value: "2" } });
    fireEvent.change(screen.getByLabelText("Effective from"), { target: { value: "2026-09-02T08:00" } });
    fireEvent.change(screen.getByLabelText("Effective until"), { target: { value: "2026-12-01T08:00" } });
    fireEvent.change(screen.getByLabelText("Issue title"), { target: { value: "Review adverse vendor response" } });
    fireEvent.change(screen.getByLabelText("Required handling"), { target: { value: "Review the response and record treatment." } });
    fireEvent.change(screen.getByLabelText("Expected outcome"), { target: { value: "The score is no longer adverse or treatment is accepted." } });
    fireEvent.click(screen.getByRole("button", { name: "Create policy draft" }));

    const input = save.mock.calls[0]?.[0];
    expect(input.eligibility).toMatchObject({ form_template_id: "form-1", form_template_version: 4, current_only: true });
    expect(input.blast_radius).toEqual({ per_run: 10, per_day: 50 });
    expect(input.code).toBe("poor-vendor-score");
    expect(input.action.type).toBe("VENDOR_DEFICIENCY");
    expect(input.outcome_contract.failure_response).toBe("ESCALATE");
    expect(input.effective_from).toBe(new Date("2026-09-02T08:00").toISOString());
    expect(input.effective_until).toBe(new Date("2026-12-01T08:00").toISOString());
    expect(input).not.toHaveProperty("actor_id");
    expect(input).not.toHaveProperty("maker_id");
  });

  it("blocks an effective window that ends before it starts", () => {
    const save = vi.fn();
    render(<FormPolicyEditor onCancel={() => undefined} onCreate={save}/>);
    for (const [label, entry] of [
      ["Policy name", "Review poor scores"], ["Policy code", "poor-score"], ["Purpose", "Create a governed review issue."],
      ["Form template ID", "form-1"], ["Automation policy ID", "automation-1"], ["Issue title", "Review response"],
      ["Required handling", "Review and record treatment."], ["Expected outcome", "The response concern is treated."],
    ] as const) fireEvent.change(screen.getByLabelText(label), { target: { value: entry } });
    fireEvent.change(screen.getByLabelText("Effective from"), { target: { value: "2026-12-01T08:00" } });
    fireEvent.change(screen.getByLabelText("Effective until"), { target: { value: "2026-09-02T08:00" } });

    fireEvent.click(screen.getByRole("button", { name: "Create policy draft" }));

    expect(screen.getByText("Effective until must be later than effective from.")).toBeTruthy();
    expect(save).not.toHaveBeenCalled();
  });
});
