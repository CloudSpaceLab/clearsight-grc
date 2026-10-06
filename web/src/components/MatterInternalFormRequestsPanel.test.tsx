import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { createLibraryFormDraft } from "../formsApi";
import { MatterInternalFormRequestsPanel } from "./MatterInternalFormRequestsPanel";

vi.mock("../formsApi", () => ({ createLibraryFormDraft: vi.fn() }));
vi.mock("./forms/DistributionComposer", () => ({ DistributionComposer: () => null }));
vi.mock("./FormBuilder", () => ({
  FormBuilder: ({ saveDraft, onSaved, onCancel }: {
    saveDraft: (input: Record<string, unknown>) => Promise<unknown>;
    onSaved: (form: unknown) => void;
    onCancel: () => void;
  }) => <div>
    <button type="button" onClick={() => void saveDraft({
      code: "ISSUE-CHECK",
      name: "Issue evidence check",
      purpose: "Collect current issue evidence.",
      presentation: { default_mode: "AUTOMATIC" },
      sections: [{ id: "evidence", title: "Evidence" }],
      fields: [{ id: "state", section_id: "evidence", label: "Current state", type: "short_text", required: true }],
    }).then(onSaved)}>Save linked form</button>
    <button type="button" onClick={onCancel}>Cancel builder</button>
  </div>,
}));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(createLibraryFormDraft).mockResolvedValue({
    id: "form-a",
    tenant_id: "bank-a",
    legal_entity_id: "entity-a",
    origin: { type: "MATTER", id: "matter-a" },
    code: "ISSUE-CHECK",
    name: "Issue evidence check",
    purpose: "Collect current issue evidence.",
    scoring_mode: "NONE",
    sensitivity: "INTERNAL",
    presentation: { default_mode: "AUTOMATIC" },
    sections: [{ id: "evidence", title: "Evidence" }],
    fields: [{ id: "state", section_id: "evidence", label: "Current state", type: "short_text", required: true }],
    status: "DRAFT",
    is_current: false,
    version: 1,
    created_at: "2026-10-06T18:00:00Z",
    updated_at: "2026-10-06T18:00:00Z",
  });
});

it("creates an ordinary form draft with the current issue as immutable origin", async () => {
  render(<MatterInternalFormRequestsPanel matterID="matter-a" matterReference="MAT-82BF"/>);

  fireEvent.click(screen.getByRole("button", { name: "Create linked form" }));
  expect(await screen.findByRole("heading", { name: "Create linked form" })).toBeTruthy();
  expect(screen.queryByText("matter-a")).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Save linked form" }));

  await waitFor(() => expect(createLibraryFormDraft).toHaveBeenCalledTimes(1));
  expect(vi.mocked(createLibraryFormDraft).mock.calls[0]?.[0]).toMatchObject({
    code: "ISSUE-CHECK",
    origin: { type: "MATTER", id: "matter-a" },
  });
  expect(await screen.findByText("Form draft created.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save linked form" })).toBeNull();
});
