import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadActorWorkDueCounts, loadActorWorkPage, type ActorWorkDueCounts } from "../api";
import type { AttentionItem } from "../types";
import { AssignedWorkQueue } from "./AssignedWorkQueue";

vi.mock("../api", () => ({ loadActorWorkDueCounts: vi.fn(), loadActorWorkPage: vi.fn() }));

const scope = { tenantID: "bank", legalEntityID: "entity", principalID: "actor" };
const assignmentCounts: ActorWorkDueCounts = {
  total: 110, overdue: 43, due_soon: 25, undated: 9,
  as_of: "2026-10-27T09:00:00Z",
  coverage: "WORKFLOW_ASSIGNMENTS_ONLY",
  count_basis: "SOURCE_SNAPSHOT",
  scope: { tenant_id: scope.tenantID, legal_entity_id: scope.legalEntityID, principal_id: scope.principalID },
};
const localItem: AttentionItem = {
  id: "local-evidence", type: "EVIDENCE_REQUEST", title: "Review received evidence",
  why_now: "Submission received", scope: "Ops", state: "Ready",
  evidence: "Submitted", owner: "Reviewer", due_at: "", primary_action: "Inspect",
  intervention_class: "EVIDENCE_EXCEPTION",
};

beforeEach(() => vi.resetAllMocks());

describe("source-qualified Work deadline measures", () => {
  it("uses complete workflow deadline counts without inventing approval/evidence totals", async () => {
    vi.mocked(loadActorWorkDueCounts).mockResolvedValue(assignmentCounts);
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    const metrics = screen.getByLabelText("Work measures");
    expect(await within(metrics).findByRole("button", { name: /Overdue: 43.*Workflow assignments/ })).toBeTruthy();
    expect(within(metrics).getByRole("button", { name: /Due next 7 days: 25.*Workflow assignments/ })).toBeTruthy();
    expect(within(metrics).getByRole("button", { name: /Your approvals: 0.*Loaded work/ })).toBeTruthy();
    expect(within(metrics).getByRole("button", { name: /Evidence needed: 1.*Loaded work/ })).toBeTruthy();
    expect(screen.getByText(/Workflow assignment totals/)).toBeTruthy();
  });

  it("fails closed when a count response belongs to another tenant", async () => {
    vi.mocked(loadActorWorkDueCounts).mockResolvedValue({
      ...assignmentCounts, scope: { ...assignmentCounts.scope, tenant_id: "other" },
    });
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    expect(await screen.findByText("Assignment totals unavailable. Showing loaded work measures.")).toBeTruthy();
    const metrics = screen.getByLabelText("Work measures");
    expect(within(metrics).getByRole("button", { name: /Overdue: 0.*Loaded work/ })).toBeTruthy();
    expect(within(metrics).queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
  });

  it("drops stale source counts immediately when the actor scope changes", async () => {
    vi.mocked(loadActorWorkDueCounts).mockResolvedValueOnce(assignmentCounts)
      .mockImplementationOnce(() => new Promise<ActorWorkDueCounts>(() => {}));
    const { rerender } = render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    await waitFor(() => expect(screen.getByRole("button", { name: /Overdue: 43/ })).toBeTruthy());
    rerender(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:another:0"
      sourceCountScope={{ ...scope, principalID: "another" }} onOpenItem={vi.fn()}/>);
    expect(screen.getByRole("button", { name: /Overdue: 0.*Loaded work/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
  });

  it("does not request global counts when the selected organization area is unsupported", () => {
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      onOpenItem={vi.fn()}/>);
    expect(loadActorWorkDueCounts).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: /Overdue: 0.*Loaded work/ })).toBeTruthy();
  });

  it("opens and pages the entire overdue source after clearing a competing title search", async () => {
    const counts = { ...assignmentCounts, total: 140, overdue: 61, as_of: "2026-10-31T10:00:00.123456Z" };
    vi.mocked(loadActorWorkDueCounts).mockResolvedValue(counts);
    const overdue = Array.from({ length: 61 }, (_, index) => ({
      ...localItem, id: `overdue-${index}`, title: `Overdue assignment ${index + 1}`, due_at: "2026-10-30T00:00:00Z",
    }));
    const receipt = { as_of: "2026-10-31T10:10:00Z", coverage: "WORKFLOW_ASSIGNMENTS_ONLY" as const,
      count_basis: "NOT_AVAILABLE" as const, scope: counts.scope };
    vi.mocked(loadActorWorkPage).mockImplementation(async (query, cursor) => query
      ? { ...receipt, items: [], has_more: false }
      : { ...receipt, items: cursor ? overdue.slice(50) : overdue.slice(0, 50), has_more: !cursor,
          next_cursor: cursor ? undefined : "remaining-overdue" });
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[localItem]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    const card = await screen.findByRole("button", { name: /Overdue: 61/ });
    const search = screen.getByRole("searchbox");
    fireEvent.change(search, { target: { value: "competing query" } });
    fireEvent.click(card);
    expect((search as HTMLInputElement).value).toBe("");
    expect(await screen.findByRole("row", { name: "Overdue assignment 50" })).toBeTruthy();
    expect(loadActorWorkPage).toHaveBeenCalledWith("", undefined, { due_before: counts.as_of });
    fireEvent.click(screen.getByRole("button", { name: "Load more assignments" }));
    expect(await screen.findByRole("row", { name: "Overdue assignment 61" })).toBeTruthy();
    expect(loadActorWorkPage).toHaveBeenLastCalledWith("", "remaining-overdue", { due_before: counts.as_of });
    expect(screen.queryByRole("button", { name: "Load more assignments" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getByRole("row", { name: localItem.title })).toBeTruthy();
    expect(screen.queryByRole("row", { name: "Overdue assignment 61" })).toBeNull();
  });

  it("keeps source-selected upcoming boundary rows despite a later page timestamp", async () => {
    vi.mocked(loadActorWorkDueCounts).mockResolvedValue(assignmentCounts);
    vi.mocked(loadActorWorkPage).mockResolvedValue({
      items: [{ ...localItem, title: "Due at counted boundary", due_at: assignmentCounts.as_of }],
      has_more: false, as_of: "2026-10-27T10:00:00Z", coverage: "WORKFLOW_ASSIGNMENTS_ONLY",
      count_basis: "NOT_AVAILABLE", scope: assignmentCounts.scope,
    });
    render(<AssignedWorkQueue presentation="workspace" state="live" items={[]}
      sourceSearchScopeKey="bank:entity:actor:0" sourceCountScope={scope} onOpenItem={vi.fn()}/>);
    fireEvent.click(await screen.findByRole("button", { name: /Due next 7 days: 25/ }));
    expect(await screen.findByRole("row", { name: "Due at counted boundary" })).toBeTruthy();
    expect(loadActorWorkPage).toHaveBeenCalledWith("", undefined, {
      due_from: assignmentCounts.as_of, due_before: "2026-11-03T09:00:00.000001Z",
    });
  });
});
