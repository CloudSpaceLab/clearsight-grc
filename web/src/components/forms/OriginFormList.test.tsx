import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { loadFormTemplatePage } from "../../formsApi";
import type { FormLibraryItem } from "../../formsTypes";
import { OriginFormList } from "./OriginFormList";

vi.mock("../../formsApi", () => ({ loadFormTemplatePage: vi.fn() }));

const draft: FormLibraryItem = {
  template: {
    id: "form-draft",
    tenant_id: "bank-a",
    legal_entity_id: "entity-a",
    origin: { type: "MATTER", id: "matter-a" },
    code: "ISSUE-DRAFT",
    name: "Issue evidence draft",
    purpose: "Collect issue evidence.",
    sensitivity: "INTERNAL",
    scoring_mode: "NONE",
    presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false },
    sections: [{ id: "evidence", title: "Evidence" }],
    fields: [{ id: "state", section_id: "evidence", label: "State", type: "short_text", required: true }],
    status: "DRAFT",
    is_current: false,
    version: 2,
    created_at: "2026-10-06T10:00:00Z",
    updated_at: "2026-10-06T11:00:00Z",
  },
  authority_available: true,
  operations: [],
};

const active: FormLibraryItem = {
  ...draft,
  template: {
    ...draft.template,
    id: "form-active",
    code: "ISSUE-ACTIVE",
    name: "Approved issue evidence",
    status: "ACTIVE",
    is_current: true,
    version: 4,
    updated_at: "2026-10-06T12:00:00Z",
  },
  active_version: 4,
  active_status: "ACTIVE",
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadFormTemplatePage).mockResolvedValue({ items: [] });
});

it("loads only the bounded origin population and pages without treating shown rows as a total", async () => {
  vi.mocked(loadFormTemplatePage)
    .mockResolvedValueOnce({ items: [draft], next_cursor: "forms-page-2" })
    .mockResolvedValueOnce({ items: [active] });

  render(<OriginFormList originType="MATTER" originID="matter-a" subjectLabel="MAT-82BF" limit={6}/>);

  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenCalledWith({
    origin_type: "MATTER",
    origin_id: "matter-a",
    limit: 6,
  }, expect.any(AbortSignal)));

  expect(await screen.findByText("Issue evidence draft")).toBeTruthy();
  expect(screen.getByText("Revision v2")).toBeTruthy();
  expect(screen.getByText("Draft")).toBeTruthy();
  expect(screen.getByText("1 shown")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Open form" }).getAttribute("href")).toBe("#forms/form-draft");
  expect(screen.queryByText(/1 linked form|1 total/i)).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Load more linked forms" }));
  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenLastCalledWith({
    origin_type: "MATTER",
    origin_id: "matter-a",
    cursor: "forms-page-2",
    limit: 6,
  }));

  expect(await screen.findByText("Approved issue evidence")).toBeTruthy();
  expect(screen.getByText("Revision v4")).toBeTruthy();
  expect(screen.getByText("Active")).toBeTruthy();
  expect(screen.getByText("Issue evidence draft")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Load more linked forms" })).toBeNull();
  expect(screen.getAllByRole("link", { name: "Open form" })).toHaveLength(2);
});

it("renders an explicit empty state for an issue with no authored forms", async () => {
  render(<OriginFormList originType="MATTER" originID="matter-a" subjectLabel="MAT-82BF"/>);

  expect(await screen.findByText("No linked forms recorded.")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Open form" })).toBeNull();
});

it("keeps prior rows and retries a failed additional page", async () => {
  vi.mocked(loadFormTemplatePage)
    .mockResolvedValueOnce({ items: [draft], next_cursor: "forms-page-2" })
    .mockRejectedValueOnce(new Error("page unavailable"))
    .mockResolvedValueOnce({ items: [active] });

  render(<OriginFormList originType="MATTER" originID="matter-a" subjectLabel="MAT-82BF"/>);

  expect(await screen.findByText("Issue evidence draft")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more linked forms" }));

  expect(await screen.findByText("More linked forms could not be loaded.")).toBeTruthy();
  expect(screen.getByText("Issue evidence draft")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry linked forms" }));

  expect(await screen.findByText("Approved issue evidence")).toBeTruthy();
  expect(screen.getByText("Issue evidence draft")).toBeTruthy();
});

it("keeps other issue work available and retries an unavailable first page", async () => {
  vi.mocked(loadFormTemplatePage)
    .mockRejectedValueOnce(new Error("unavailable"))
    .mockResolvedValueOnce({ items: [draft] });

  render(<OriginFormList originType="MATTER" originID="matter-a" subjectLabel="MAT-82BF"/>);

  expect(await screen.findByText(/Linked forms are unavailable\. Other issue work remains available\./)).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry linked forms" }));

  expect(await screen.findByText("Issue evidence draft")).toBeTruthy();
  await waitFor(() => expect(loadFormTemplatePage).toHaveBeenCalledTimes(2));
});
