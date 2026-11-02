import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadActorWorkDueCounts, loadActorWorkPage, type ActorWorkDueCounts, type ActorWorkPage } from "../api";
import { ApiError } from "../http";
import type { AttentionItem } from "../types";
import { AssignedWorkQueue, type AssignedWorkQueueProps } from "./AssignedWorkQueue";

vi.mock("../api", () => ({ loadActorWorkDueCounts: vi.fn(), loadActorWorkPage: vi.fn() }));

const scope = { tenantID: "bank", legalEntityID: "entity", principalID: "reviewer" };
const receiptScope = { tenant_id: scope.tenantID, legal_entity_id: scope.legalEntityID, principal_id: scope.principalID };
const counts: ActorWorkDueCounts = {
  total: 110, overdue: 43, due_soon: 25, undated: 9,
  as_of: "2026-10-31T09:00:00Z", coverage: "WORKFLOW_ASSIGNMENTS_ONLY",
  count_basis: "SOURCE_SNAPSHOT", scope: receiptScope,
};
const item: AttentionItem = {
  id: "workflow-visible", type: "MATTER", title: "Review retained evidence",
  why_now: "Evidence received", scope: "Compliance", state: "Ready", evidence: "Case 1",
  owner: "Reviewer", due_at: "", primary_action: "Open issue",
  action_target_type: "MATTER", action_target_id: "matter-visible",
};
type ScopedPage = ActorWorkPage & { scope: ActorWorkDueCounts["scope"] };
const page: ScopedPage = {
  items: [item], has_more: true, next_cursor: "page-two", as_of: counts.as_of,
  coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE", scope: receiptScope,
};

function queue(overrides: Partial<AssignedWorkQueueProps> = {}) {
  return <AssignedWorkQueue presentation="workspace" state="live" items={[]}
    sourceSearchScopeKey="bank:entity:reviewer:0" sourceCountScope={scope}
    onOpenItem={vi.fn()} {...overrides}/>;
}

function searchAssignments() {
  fireEvent.change(screen.getByRole("searchbox"), { target: { value: "retained" } });
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((accept) => { resolve = accept; });
  return { promise, resolve };
}

beforeEach(() => {
  vi.resetAllMocks();
  vi.mocked(loadActorWorkDueCounts).mockResolvedValue(counts);
});

describe("Work source receipt validation", () => {
  const invalidReceipts: Array<[string, ScopedPage]> = [
    ["tenant", { ...page, scope: { ...receiptScope, tenant_id: "other-bank" } }],
    ["legal entity", { ...page, scope: { ...receiptScope, legal_entity_id: "other-entity" } }],
    ["principal", { ...page, scope: { ...receiptScope, principal_id: "other-reviewer" } }],
    ["missing scope", { ...page, scope: undefined } as unknown as ScopedPage],
    ["coverage", { ...page, coverage: "ALL_WORK" } as unknown as ScopedPage],
    ["count basis", { ...page, count_basis: "LOADED" } as unknown as ScopedPage],
  ];

  it.each(invalidReceipts)("rejects an initial page with an invalid %s receipt", async (_label, invalidPage) => {
    vi.mocked(loadActorWorkPage).mockResolvedValue(invalidPage);
    render(queue());
    searchAssignments();

    expect(await screen.findByRole("button", { name: "Try again" })).toBeTruthy();
    expect(screen.queryByRole("row", { name: item.title })).toBeNull();
    expect(screen.queryByRole("button", { name: "Load more assignments" })).toBeNull();
  });

  it("clears retained rows when a later page belongs to another actor", async () => {
    const deniedTitle = "Review another actor's retained evidence";
    vi.mocked(loadActorWorkPage).mockResolvedValueOnce(page).mockResolvedValueOnce({
      ...page, items: [{ ...item, id: "workflow-denied", title: deniedTitle }],
      scope: { ...receiptScope, principal_id: "other-reviewer" }, has_more: false,
    });
    render(queue());
    searchAssignments();
    expect(await screen.findByRole("row", { name: item.title })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load more assignments" }));
    expect(await screen.findByRole("button", { name: "Try again" })).toBeTruthy();
    expect(screen.queryByRole("row", { name: item.title })).toBeNull();
    expect(screen.queryByRole("row", { name: deniedTitle })).toBeNull();
  });

  it.each([401, 403])("clears retained source rows after pagination returns HTTP %s", async (status) => {
    vi.mocked(loadActorWorkPage).mockResolvedValueOnce(page)
      .mockRejectedValueOnce(new ApiError(status, "Assigned work access changed.", "work_scope_denied"));
    render(queue());
    searchAssignments();
    expect(await screen.findByRole("row", { name: item.title })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load more assignments" }));
    expect(await screen.findByRole("button", { name: "Try again" })).toBeTruthy();
    expect(screen.queryByRole("row", { name: item.title })).toBeNull();
    expect(screen.queryByRole("button", { name: "Load more assignments" })).toBeNull();
  });

  it("retains valid rows after a recoverable pagination failure and retries the same cursor", async () => {
    const nextItem = { ...item, id: "workflow-next", title: "Review more retained evidence" };
    vi.mocked(loadActorWorkPage).mockResolvedValueOnce(page)
      .mockRejectedValueOnce(new ApiError(0, "Connection lost.", "connection_lost"))
      .mockResolvedValueOnce({ ...page, items: [nextItem], has_more: false, next_cursor: undefined });
    render(queue());
    searchAssignments();
    expect(await screen.findByRole("row", { name: item.title })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load more assignments" }));
    expect((await screen.findByRole("alert")).textContent).toContain("More assignments could not be loaded");
    expect(screen.getByRole("row", { name: item.title })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Load more assignments" }));
    expect(await screen.findByRole("row", { name: nextItem.title })).toBeTruthy();
    expect(screen.getByRole("row", { name: item.title })).toBeTruthy();
    const requests = vi.mocked(loadActorWorkPage).mock.calls;
    expect(requests[2]).toEqual(requests[1]);
  });
});

describe("Work count scope re-entry", () => {
  it("clears a source deadline selection after refresh so the next count click queries the source", async () => {
    const refreshedCounts = { ...counts, overdue: 44, as_of: "2026-10-31T09:01:00Z" };
    vi.mocked(loadActorWorkDueCounts).mockResolvedValueOnce(counts).mockResolvedValueOnce(refreshedCounts);
    vi.mocked(loadActorWorkPage)
      .mockResolvedValueOnce({ ...page, has_more: false, next_cursor: undefined, as_of: "2026-10-31T09:00:01Z" })
      .mockResolvedValueOnce({ ...page, has_more: false, next_cursor: undefined, as_of: "2026-10-31T09:01:01Z" });
    const { rerender } = render(queue());
    fireEvent.click(await screen.findByRole("button", { name: /Overdue: 43/ }));
    expect(await screen.findByRole("row", { name: item.title })).toBeTruthy();
    expect(loadActorWorkPage).toHaveBeenCalledTimes(1);

    rerender(queue({ sourceSearchScopeKey: "bank:entity:reviewer:1" }));
    const refreshedCard = await screen.findByRole("button", { name: /Overdue: 44/ });
    expect(refreshedCard.getAttribute("aria-pressed")).toBe("false");
    expect(screen.queryByRole("row", { name: item.title })).toBeNull();
    fireEvent.click(refreshedCard);
    expect(await screen.findByRole("row", { name: item.title })).toBeTruthy();
    expect(loadActorWorkPage).toHaveBeenCalledTimes(2);
    expect(loadActorWorkPage).toHaveBeenLastCalledWith("", undefined, { due_before: refreshedCounts.as_of });
  });

  it("does not preserve a loaded-only deadline selection when its source count arrives", async () => {
    const pendingCounts = deferred<ActorWorkDueCounts>();
    vi.mocked(loadActorWorkDueCounts).mockReturnValueOnce(pendingCounts.promise);
    vi.mocked(loadActorWorkPage).mockResolvedValue({ ...page, has_more: false, next_cursor: undefined });
    render(queue());
    const loadedCard = screen.getByRole("button", { name: /Overdue: 0.*Loaded work/ });
    fireEvent.click(loadedCard);
    expect(loadedCard.getAttribute("aria-pressed")).toBe("true");

    await act(async () => pendingCounts.resolve(counts));
    const sourceCard = await screen.findByRole("button", { name: /Overdue: 43.*Workflow assignments/ });
    expect(sourceCard.getAttribute("aria-pressed")).toBe("false");
    fireEvent.click(sourceCard);
    expect(await screen.findByRole("row", { name: item.title })).toBeTruthy();
    expect(loadActorWorkPage).toHaveBeenCalledWith("", undefined, { due_before: counts.as_of });
  });

  it("waits for a fresh count receipt after returning from an unsupported scope", async () => {
    const refreshed = deferred<ActorWorkDueCounts>();
    vi.mocked(loadActorWorkDueCounts).mockResolvedValueOnce(counts).mockReturnValueOnce(refreshed.promise);
    const { rerender } = render(queue());
    expect(await screen.findByRole("button", { name: /Overdue: 43/ })).toBeTruthy();

    rerender(queue({ sourceSearchScopeKey: undefined, sourceCountScope: undefined }));
    expect(screen.queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
    rerender(queue());
    expect(screen.queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
    await waitFor(() => expect(loadActorWorkDueCounts).toHaveBeenCalledTimes(2));

    await act(async () => refreshed.resolve({ ...counts, overdue: 37 }));
    expect(await screen.findByRole("button", { name: /Overdue: 37/ })).toBeTruthy();
  });

  it("does not reuse an earlier receipt when returning before the intervening actor read finishes", async () => {
    const otherActor = deferred<ActorWorkDueCounts>();
    const refreshed = deferred<ActorWorkDueCounts>();
    vi.mocked(loadActorWorkDueCounts).mockResolvedValueOnce(counts)
      .mockReturnValueOnce(otherActor.promise).mockReturnValueOnce(refreshed.promise);
    const { rerender } = render(queue());
    expect(await screen.findByRole("button", { name: /Overdue: 43/ })).toBeTruthy();

    rerender(queue({
      sourceSearchScopeKey: "bank:entity:other-reviewer:0",
      sourceCountScope: { ...scope, principalID: "other-reviewer" },
    }));
    expect(screen.queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
    rerender(queue());
    expect(screen.queryByRole("button", { name: /Overdue: 43/ })).toBeNull();
    await waitFor(() => expect(loadActorWorkDueCounts).toHaveBeenCalledTimes(3));

    await act(async () => otherActor.resolve({
      ...counts, overdue: 61, scope: { ...receiptScope, principal_id: "other-reviewer" },
    }));
    expect(screen.queryByRole("button", { name: /Overdue: (43|61)/ })).toBeNull();
    await act(async () => refreshed.resolve({ ...counts, overdue: 37 }));
    expect(await screen.findByRole("button", { name: /Overdue: 37/ })).toBeTruthy();
  });
});
