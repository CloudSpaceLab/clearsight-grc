import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FormTemplate } from "../monitoringTypes";
import { FormBuilder } from "./FormBuilder";

const draft: FormTemplate = {
  id: "form-1",
  tenant_id: "bank-1",
  code: "VENDOR",
  name: "Vendor review",
  purpose: "Collect current vendor evidence.",
  scoring_mode: "NONE",
  presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
  sections: [{ id: "identity", title: "Identity" }],
  fields: [{ id: "name", section_id: "identity", label: "Registered name", type: "short_text", required: true }],
  status: "DRAFT",
  is_current: false,
  version: 1,
  created_at: "2026-08-27T00:00:00Z",
  updated_at: "2026-08-27T00:00:00Z",
};

describe("FormBuilder approval recovery", () => {
  it("returns the newly persisted immutable draft when its approval transition fails", async () => {
    const persisted = { ...draft, name: "Updated vendor review", version: 2 };
    const saveDraft = vi.fn().mockResolvedValue(persisted);
    const sendForApproval = vi.fn().mockRejectedValue(new Error("approval service unavailable"));
    const onSaved = vi.fn();
    render(<FormBuilder
      initialValue={draft}
      saveDraft={saveDraft}
      onSendForApproval={sendForApproval}
      onSaved={onSaved}
      onCancel={vi.fn()}
      allowIncompleteComplianceDraft
    />);

    fireEvent.change(screen.getByLabelText("Form name"), { target: { value: "Updated vendor review" } });
    fireEvent.click(screen.getByRole("button", { name: "Send for approval" }));

    await waitFor(() => expect(saveDraft).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(sendForApproval).toHaveBeenCalledWith(persisted));
    await waitFor(() => expect(onSaved).toHaveBeenCalledWith(persisted));
  });
});
