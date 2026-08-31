import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import type { FormTemplate } from "../monitoringTypes";
import { FormBuilder } from "./FormBuilder";

const form: FormTemplate = {
  id: "form-order",
  tenant_id: "bank",
  legal_entity_id: "entity",
  code: "ORDER",
  name: "Ordering review",
  purpose: "Verify section-local question ordering.",
  status: "DRAFT",
  is_current: false,
  version: 1,
  created_at: "2026-08-27T00:00:00Z",
  updated_at: "2026-08-27T00:00:00Z",
  scoring_mode: "NONE",
  presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
  sections: [
    { id: "first", title: "First section" },
    { id: "second", title: "Second section" },
  ],
  fields: [
    { id: "q1", section_id: "first", label: "First A", type: "short_text", required: true },
    { id: "q2", section_id: "first", label: "First B", type: "short_text", required: true },
    { id: "q3", section_id: "second", label: "Second A", type: "short_text", required: true },
    { id: "q4", section_id: "second", label: "Second B", type: "short_text", required: true },
  ],
};

it("moves questions only among siblings in the same section", async () => {
  const saveDraft = vi.fn().mockResolvedValue(form);
  render(<FormBuilder initialValue={form} saveDraft={saveDraft} onSaved={vi.fn()} onCancel={vi.fn()}/>);

  fireEvent.click(screen.getByRole("button", { name: "Second A" }));
  expect((screen.getByRole("button", { name: "Move Second A up" }) as HTMLButtonElement).disabled).toBe(true);

  fireEvent.click(screen.getByRole("button", { name: "Second B" }));
  expect((screen.getByRole("button", { name: "Move Second B down" }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Move Second B up" }));
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

  await waitFor(() => expect(saveDraft).toHaveBeenCalledTimes(1));
  expect(saveDraft.mock.calls[0]?.[0].fields.map((field: { id: string }) => field.id)).toEqual(["q1", "q2", "q4", "q3"]);
});
