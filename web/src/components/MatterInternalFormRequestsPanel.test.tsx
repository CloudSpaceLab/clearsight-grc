import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadFormTemplatePage } from "../formsApi";
import type { FormLibraryItem, FormTemplate } from "../formsTypes";
import { MatterInternalFormRequestsPanel } from "./MatterInternalFormRequestsPanel";

vi.mock("../formsApi", () => ({ loadFormTemplatePage: vi.fn() }));
vi.mock("./forms/DistributionComposer", () => ({
  DistributionComposer: ({ subject, internalOnly, onCreated }: {
    subject: { type: string; id: string; label: string };
    internalOnly: boolean;
    onCreated: () => void;
  }) => <div>
    <span>{`${subject.type}:${subject.id}`}</span>
    <span>{internalOnly ? "Employees only" : "External recipients allowed"}</span>
    <button type="button" onClick={onCreated}>Send approved form</button>
  </div>,
}));
vi.mock("./forms/MatterFormLinksPanel", () => ({
  MatterFormLinksPanel: ({ pickerOpen }: { pickerOpen: boolean }) =>
    pickerOpen ? <span>Approved form picker open</span> : null,
}));
vi.mock("./forms/SubjectFormActivity", () => ({ SubjectFormActivity: ({ leading }: { leading?: ReactNode }) => <div>{leading}<span>Form activity</span></div> }));

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
  vi.mocked(loadFormTemplatePage).mockResolvedValue({ items: [] });
});

it("uses an existing form for an employee request without creating a template", async () => {
  render(<MatterInternalFormRequestsPanel matterID="matter-a" matterReference="MAT-82BF"/>);

  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenCalledWith(
    { origin_type: "MATTER", origin_id: "matter-a", limit: 6 },
    expect.any(AbortSignal),
  ));
  expect(screen.queryByText("No linked forms yet")).toBeNull();
  expect(screen.getByRole("group", { name: "Form actions" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Create linked form" })).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Link existing form" }));
  expect(await screen.findByText("Approved form picker open")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Request response" }));
  expect(await screen.findByRole("heading", { name: "Request response" })).toBeTruthy();
  expect(screen.getByText("MATTER:matter-a")).toBeTruthy();
  expect(screen.getByText("Employees only")).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Send approved form" }));
  expect(await screen.findByText("Form request created.")).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Request response" })).toBeNull();
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
  expect(screen.getByRole("button", { name: "Request response" })).toBeTruthy();

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
