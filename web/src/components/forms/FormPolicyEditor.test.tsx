import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FormPolicyEditor } from "./FormPolicyEditor";

const forms = [
  { id: "form-1", name: "Vendor certification", code: "VENDOR-CERTIFICATION", version: 4 },
  { id: "form-2", name: "Control assurance", code: "CONTROL-ASSURANCE", version: 2 },
];

describe("FormPolicyEditor", () => {
  it("captures the selected approved form revision, blast radius and outcome check without actor fields", async () => {
    const save = vi.fn();
    render(<FormPolicyEditor forms={forms} onCancel={() => undefined} onCreate={save}/>);

    fireEvent.change(screen.getByLabelText("Policy name"), { target: { value: "Review poor vendor scores" } });
    fireEvent.change(screen.getByLabelText("Policy code"), { target: { value: "poor-vendor-score" } });
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Create a review issue for an adverse completed vendor response." } });
    fireEvent.click(screen.getByRole("button", { name: /Approved form revision/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Control assurance · CONTROL-ASSURANCE · revision 2" }));
    fireEvent.change(screen.getByLabelText("Effective from"), { target: { value: "2026-09-02T08:00" } });
    fireEvent.change(screen.getByLabelText("Effective until"), { target: { value: "2026-12-01T08:00" } });
    fireEvent.change(screen.getByLabelText("Issue title"), { target: { value: "Review adverse vendor response" } });
    fireEvent.change(screen.getByLabelText("What needs to happen next"), { target: { value: "Review the response and record treatment." } });
    fireEvent.change(screen.getByLabelText("Expected outcome"), { target: { value: "The score is no longer adverse or treatment is accepted." } });
    fireEvent.click(screen.getByRole("button", { name: "Create policy draft" }));

    const input = save.mock.calls[0]?.[0];
    expect(input.eligibility).toMatchObject({ form_template_id: "form-2", form_template_version: 2, current_only: true });
    expect(input.blast_radius).toEqual({ per_run: 10, per_day: 50 });
    expect(input.code).toBe("poor-vendor-score");
    expect(input.create_automation_policy).toBe(true);
    expect(input.eligibility.subject_types).toEqual(["VENDOR_RELATIONSHIP"]);
    expect(screen.queryByLabelText("Automation policy ID")).toBeNull();
    expect(screen.queryByLabelText("Subject types")).toBeNull();
    expect(input.action.type).toBe("VENDOR_DEFICIENCY");
    expect(input.outcome_contract.failure_response).toBe("ESCALATE");
    expect(input.effective_from).toBe(new Date("2026-09-02T08:00").toISOString());
    expect(input.effective_until).toBe(new Date("2026-12-01T08:00").toISOString());
    expect(input).not.toHaveProperty("actor_id");
    expect(input).not.toHaveProperty("maker_id");
  });

  it("blocks an effective window that ends before it starts", () => {
    const save = vi.fn();
    render(<FormPolicyEditor forms={forms} onCancel={() => undefined} onCreate={save}/>);
    for (const [label, entry] of [
      ["Policy name", "Review poor scores"], ["Policy code", "poor-score"], ["Purpose", "Create a governed review issue."],
      ["Issue title", "Review response"],
      ["What needs to happen next", "Review and record treatment."], ["Expected outcome", "The response concern is treated."],
    ] as const) fireEvent.change(screen.getByLabelText(label), { target: { value: entry } });
    fireEvent.change(screen.getByLabelText("Effective from"), { target: { value: "2026-12-01T08:00" } });
    fireEvent.change(screen.getByLabelText("Effective until"), { target: { value: "2026-09-02T08:00" } });

    fireEvent.click(screen.getByRole("button", { name: "Create policy draft" }));

    expect(screen.getByText("Effective until must be later than effective from.")).toBeTruthy();
    expect(save).not.toHaveBeenCalled();
  });

  it("fails closed when there is no active approved scoring form revision", () => {
    render(<FormPolicyEditor forms={[]} onCancel={() => undefined} onCreate={vi.fn()}/>);
    expect(screen.getByText(/No active approved scoring forms are available/)).toBeTruthy();
    expect((screen.getByRole("button", { name: "Create policy draft" }) as HTMLButtonElement).disabled).toBe(true);
  });

  it("keeps a form-source failure recoverable without exposing raw IDs", () => {
    const retry = vi.fn();
    render(<FormPolicyEditor forms={[]} formsError="Approved scoring forms cannot be checked right now." onRetryForms={retry} onCancel={() => undefined} onCreate={vi.fn()}/>);
    expect(screen.queryByLabelText("Form template ID")).toBeNull();
    expect(screen.queryByLabelText("Form revision")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Retry form choices" }));
    expect(retry).toHaveBeenCalledTimes(1);
  });
  it("creates a new guardrail draft when a selected policy's limits change", async () => {
    const save=vi.fn();
    const choice={id:"automation-scoped",name:"Vendor concern handling",purpose:"Review adverse vendor evidence.",status:"ACTIVE",version:3,eligibility:{form_template_id:"form-1",form_template_version:4,subject_types:["VENDOR_RELATIONSHIP"],current_only:true,minimum_coverage:0.8,bands:["HIGH" as const]},blast_radius:{per_run:5,per_day:20},outcome_contract:{expected_outcome:"Concern treated.",check_after_minutes:60,failure_response:"ESCALATE" as const},rollout:"SHADOW" as const};
    render(<FormPolicyEditor forms={forms} automationChoices={[choice]} onCancel={() => undefined} onCreate={save}/>);
    for(const [label,value] of ([["Policy name","Vendor review"],["Policy code","vendor-review"],["Purpose","Review adverse vendor results."],["Issue title","Review vendor evidence"],["What needs to happen next","Review and treat the concern."]] as const)) fireEvent.change(screen.getByLabelText(label),{target:{value}});
    fireEvent.click(screen.getByRole("button",{name:/Automation policy/}));
    fireEvent.click(await screen.findByRole("option",{name:/Vendor concern handling/}));
    expect(screen.getByText(/Review adverse vendor evidence\./)).toBeTruthy();
    fireEvent.change(screen.getByLabelText("Maximum new issues per run"),{target:{value:"6"}});
    fireEvent.click(screen.getByRole("button",{name:"Create policy draft"}));
    expect(save.mock.calls[0]?.[0]).toMatchObject({create_automation_policy:true,blast_radius:{per_run:6,per_day:20}});
  });

  it("defaults forms with bank review to completed bank assessment", () => {
    render(<FormPolicyEditor forms={[{...forms[0]!,requiresBankAssessment:true}]} onCancel={() => undefined} onCreate={vi.fn()}/>);
    expect(screen.getByRole("button",{name:/Completed bank assessment/})).toBeTruthy();
    expect(screen.getByText(/Required bank reviews must be complete/)).toBeTruthy();
  });

});
