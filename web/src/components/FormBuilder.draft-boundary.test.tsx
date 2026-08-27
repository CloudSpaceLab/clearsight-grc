import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createFormTemplate } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { FormBuilder } from "./FormBuilder";

vi.mock("../monitoringApi", () => ({ createFormTemplate: vi.fn() }));

const incompleteComplianceForm: FormTemplate = {
  id: "form-1",
  tenant_id: "bank-1",
  code: "VENDOR",
  name: "Vendor review",
  purpose: "Collect current vendor evidence.",
  scoring_mode: "COMPLIANCE",
  presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
  sections: [{ id: "identity", title: "Vendor identity", weight: 100 }],
  fields: [{
    id: "registration",
    section_id: "identity",
    label: "Registration verified",
    type: "yes_no",
    required: true,
    options: ["Yes", "No"],
    scoring: { weight: 80, answer_scores: { Yes: 100, No: 0 } },
  }],
  status: "DRAFT",
  is_current: false,
  version: 1,
  created_at: "2026-08-27T00:00:00Z",
  updated_at: "2026-08-27T00:00:00Z",
};

beforeEach(() => vi.clearAllMocks());

describe("FormBuilder draft validation boundary", () => {
  it("keeps Program authoring strict because its create API uses canonical Normalize", async () => {
    render(<FormBuilder programID="program-1" initialValue={incompleteComplianceForm} onSaved={vi.fn()} onCancel={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    const alert = await screen.findByRole("alert");
    expect(alert.textContent).toContain("20% remains to allocate in Vendor identity");
    expect(createFormTemplate).not.toHaveBeenCalled();
  });

  it("lets direct Forms persist the same structurally valid draft while approval remains blocked", async () => {
    const saveDraft = vi.fn().mockResolvedValue(incompleteComplianceForm);
    render(<FormBuilder
      initialValue={incompleteComplianceForm}
      saveDraft={saveDraft}
      onSendForApproval={vi.fn()}
      onSaved={vi.fn()}
      onCancel={vi.fn()}
      allowIncompleteComplianceDraft
    />);

    expect((screen.getByRole("button", { name: "Send for approval" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    await waitFor(() => expect(saveDraft).toHaveBeenCalledTimes(1));
  });
});
