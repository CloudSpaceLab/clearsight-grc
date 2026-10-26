import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AttentionItem } from "../types";
import { loadActorWorkPage } from "../api";
import { AssignedWorkQueue } from "./AssignedWorkQueue";

vi.mock("../api", () => ({ loadActorWorkPage: vi.fn() }));

const item: AttentionItem = {
  id: "workflow_1", type: "MATTER", title: "Review retained evidence",
  why_now: "Evidence received", scope: "Compliance", state: "Ready", evidence: "Case 1",
  owner: "Reviewer", due_at: "", primary_action: "Open issue",
  action_target_type: "MATTER", action_target_id: "matter-1",
};
const asOf = "2026-10-21T00:00:00Z";

beforeEach(() => { vi.clearAllMocks(); });

describe("source-backed Work title search", () => {
  it("queries beyond the Today feed and pages without changing the default items", async () => {
    vi.mocked(loadActorWorkPage)
      .mockResolvedValueOnce({
        items: [item], next_cursor: "page-two", has_more: true, as_of: asOf,
        coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE",
      })
      .mockResolvedValueOnce({
        items: [{ ...item, id: "workflow_2", title: "Review retained evidence 2" }],
        has_more: false, as_of: asOf,
        coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE",
      });

    const local = { ...item, id: "admin-recovery", title: "Assign issue owner" };
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[local]}
      sourceSearchScopeKey="actor:entity" onOpenItem={vi.fn()}/>);

    const searchField = screen.getByRole("searchbox", { name: "Search assignment titles" });
    searchField.focus();
    fireEvent.change(searchField, { target: { value: "retained evidence" } });
    expect(screen.getByRole("searchbox", { name: "Search assignment titles" })).toBe(searchField);
    expect(document.activeElement).toBe(searchField);
    await waitFor(() => expect(loadActorWorkPage).toHaveBeenCalledWith("retained evidence"));
    const table = await screen.findByRole("table", { name: "Assigned work" });
    expect(within(table).getByRole("row", { name: item.title })).toBeTruthy();
    expect(within(table).queryByRole("row", { name: local.title })).toBeNull();
    expect(screen.getByText(/Workflow assignments only/)).toBeTruthy();
    expect(screen.getByText(/1 of 1 loaded assignment matches/)).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load more assignments" }));
    await waitFor(() => expect(loadActorWorkPage).toHaveBeenCalledWith("retained evidence", "page-two"));
    expect(await within(table).findByRole("row", { name: "Review retained evidence 2" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Load more assignments" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(within(screen.getByRole("table", { name: "Assigned work" })).getByRole("row", { name: local.title })).toBeTruthy();
    expect(loadActorWorkPage).toHaveBeenCalledTimes(2);
  });

  it("never renders results from a previous actor scope", async () => {
    vi.mocked(loadActorWorkPage).mockResolvedValue({
      items: [item], has_more: false, as_of: asOf,
      coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE",
    });
    const { rerender } = render(<AssignedWorkQueue presentation="workspace" state="live" items={[]}
      sourceSearchScopeKey="actor:entity-a" onOpenItem={vi.fn()}/>);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search assignment titles" }), {
      target: { value: "retained" },
    });
    expect(await screen.findByRole("row", { name: item.title })).toBeTruthy();

    vi.mocked(loadActorWorkPage).mockImplementationOnce(() => new Promise(() => {}));
    rerender(<AssignedWorkQueue presentation="workspace" state="live" items={[]}
      sourceSearchScopeKey="actor:entity-b" onOpenItem={vi.fn()}/>);
    expect(screen.queryByRole("row", { name: item.title })).toBeNull();
    expect(screen.getByText("Searching assignment titles…")).toBeTruthy();
    expect(screen.getByRole("searchbox", { name: "Search assignment titles" })).toBeTruthy();
  });

  it("does not claim the approval queue is empty until later pages are checked", async () => {
    const review = { ...item, intervention_class: "REVIEW" as const };
    const approval = { ...item, id: "workflow_approval", title: "Authorize retained evidence", intervention_class: "AUTHORIZATION" as const };
    vi.mocked(loadActorWorkPage)
      .mockResolvedValueOnce({
        items: [review], next_cursor: "remaining", has_more: true, as_of: asOf,
        coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE",
      })
      .mockResolvedValueOnce({
        items: [approval], has_more: false, as_of: asOf,
        coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE",
      });

    render(<AssignedWorkQueue presentation="workspace" state="live" items={[]}
      sourceSearchScopeKey="actor:entity" onOpenItem={vi.fn()}/>);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search assignment titles" }), {
      target: { value: "retained" },
    });
    expect(await screen.findByRole("row", { name: review.title })).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: "Approvals" }));
    expect(screen.getByText("No matching work loaded yet")).toBeTruthy();
    expect(screen.getByText("Load more assignments to check remaining records.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Load more assignments" }));
    expect(await screen.findByRole("row", { name: approval.title })).toBeTruthy();
    expect(screen.queryByText("No matching work loaded yet")).toBeNull();
  });

  it("does not use server search when a narrower scope cannot be queried", () => {
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[item]}
      onOpenItem={vi.fn()}/>);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search loaded work" }), {
      target: { value: "retained" },
    });
    expect(screen.getByRole("row", { name: item.title })).toBeTruthy();
    expect(loadActorWorkPage).not.toHaveBeenCalled();
  });
});
