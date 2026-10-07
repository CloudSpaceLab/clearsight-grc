import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { createLibraryFormDraft, loadFormTemplatePage } from "../formsApi";
import type { FormLibraryItem } from "../formsTypes";
import type { FormTemplate } from "../monitoringTypes";
import { MatterInternalFormRequestsPanel } from "./MatterInternalFormRequestsPanel";

vi.mock("../formsApi", () => ({
  createLibraryFormDraft: vi.fn(),
  loadFormTemplatePage: vi.fn(),
}));
vi.mock("./forms/DistributionComposer", () => ({ DistributionComposer: () => null }));
vi.mock("./forms/SubjectFormActivity", () => ({ SubjectFormActivity: () => <div>Form activity</div> }));
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
      presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false },
      sections: [{ id: "evidence", title: "Evidence" }],
      fields: [{ id: "state", section_id: "evidence", label: "Current state", type: "short_text", required: true }],
    }).then(onSaved)}>Save linked form</button>
    <button type="button" onClick={onCancel}>Cancel builder</button>
  </div>,
}));

const createdDraft: FormTemplate = {
  id: "form-a",
  tenant_id: "bank-a",
  legal_entity_id: "entity-a",
  origin: { type: "MATTER", id: "matter-a" },
  code: "ISSUE-CHECK",
  name: "Issue evidence check",
  purpose: "Collect current issue evidence.",
  scoring_mode: "NONE",
  sensitivity: "INTERNAL",
  presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false },
  sections: [{ id: "evidence", title: "Evidence" }],
  fields: [{ id: "state", section_id: "evidence", label: "Current state", type: "short_text", required: true }],
  status: "DRAFT",
  is_current: false,
  version: 1,
  created_at: "2026-10-06T18:00:00Z",
  updated_at: "2026-10-06T18:00:00Z",
};

function linkedItem(id: string, name: string, status: FormTemplate["status"], version: number): FormLibraryItem {
  return {
    template: {
      ...createdDraft,
      id,
      code: id.toUpperCase(),
      name,
      status,
      version,
      is_current: status === "ACTIVE",
      updated_at: `2026-10-0${Math.min(version + 1, 9)}T18:00:00Z`,
    },
    ...(status === "ACTIVE" ? { active_version: version, active_status: "ACTIVE" as const } : {}),
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(createLibraryFormDraft).mockResolvedValue(createdDraft);
  vi.mocked(loadFormTemplatePage).mockResolvedValue({ items: [] });
});

it("creates an ordinary form draft with the current issue as immutable origin and refreshes linked forms", async () => {
  render(<MatterInternalFormRequestsPanel matterID="matter-a" matterReference="MAT-82BF"/>);

  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenCalledWith(
    { origin_type: "MATTER", origin_id: "matter-a", limit: 6 },
    expect.any(AbortSignal),
  ));
  expect(await screen.findByText("No linked forms recorded.")).toBeTruthy();

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
  expect(screen.getByRole("link", { name: "Open form draft" }).getAttribute("href")).toBe("#forms/form-a");
  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenCalledTimes(2));
  expect(screen.queryByRole("button", { name: "Save linked form" })).toBeNull();
});

it("shows bounded linked Forms with lifecycle and exact handoffs, then appends the next page", async () => {
  const draft = linkedItem("form-linked-draft", "Issue evidence draft", "DRAFT", 2);
  const active = linkedItem("form-linked-active", "Approved issue evidence", "ACTIVE", 4);
  vi.mocked(loadFormTemplatePage)
    .mockResolvedValueOnce({ items: [draft], next_cursor: "forms-cursor-2" })
    .mockResolvedValueOnce({ items: [active] });

  render(<MatterInternalFormRequestsPanel matterID="matter-a" matterReference="MAT-82BF"/>);

  expect(await screen.findByText("Issue evidence draft")).toBeTruthy();
  expect(screen.getByText("Revision 2")).toBeTruthy();
  expect(screen.getByText("Draft")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Open form" }).getAttribute("href")).toBe("#forms/form-linked-draft");
  expect(screen.getByText("1 shown")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Load more linked forms" }));
  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenLastCalledWith({
    origin_type: "MATTER",
    origin_id: "matter-a",
    cursor: "forms-cursor-2",
    limit: 6,
  }));
  expect(await screen.findByText("Approved issue evidence")).toBeTruthy();
  expect(screen.getByText("Issue evidence draft")).toBeTruthy();
  expect(screen.getByText("Revision 4")).toBeTruthy();
  expect(screen.getByText("Active")).toBeTruthy();
  expect(screen.getAllByRole("link", { name: "Open form" })).toHaveLength(2);
  expect(screen.queryByRole("button", { name: "Load more linked forms" })).toBeNull();
  expect(screen.queryByText(/2 linked forms total/i)).toBeNull();
});

it("keeps issue work available and retries a failed linked-forms read locally", async () => {
  const draft = linkedItem("form-linked-draft", "Recovered linked form", "DRAFT", 1);
  vi.mocked(loadFormTemplatePage)
    .mockRejectedValueOnce(new Error("library unavailable"))
    .mockResolvedValueOnce({ items: [draft] });

  render(<MatterInternalFormRequestsPanel matterID="matter-a" matterReference="MAT-82BF"/>);

  expect(await screen.findByText(/Linked forms are unavailable\. Other issue work remains available\./)).toBeTruthy();
  expect(screen.getByText("Form activity")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Create linked form" })).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Retry linked forms" }));
  expect(await screen.findByText("Recovered linked form")).toBeTruthy();
  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenCalledTimes(2));
});

it("retains linked Forms and offers local retry when a later page fails", async () => {
  const first = linkedItem("form-linked-first", "First linked form", "DRAFT", 1);
  const second = linkedItem("form-linked-second", "Second linked form", "ACTIVE", 2);
  vi.mocked(loadFormTemplatePage)
    .mockResolvedValueOnce({ items: [first], next_cursor: "forms-cursor-2" })
    .mockRejectedValueOnce(new Error("next page unavailable"))
    .mockResolvedValueOnce({ items: [second] });

  render(<MatterInternalFormRequestsPanel matterID="matter-a" matterReference="MAT-82BF"/>);

  expect(await screen.findByText("First linked form")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more linked forms" }));
  expect(await screen.findByText("More linked forms could not be loaded.")).toBeTruthy();
  expect(screen.getByText("First linked form")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Retry linked forms" }));
  expect(await screen.findByText("Second linked form")).toBeTruthy();
  expect(screen.getByText("First linked form")).toBeTruthy();
  await waitFor(() => expect(screen.queryByText("More linked forms could not be loaded.")).toBeNull());
});
