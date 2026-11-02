import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadActorWorkPage, type ActorWorkPage } from "../api";
import type { WorkDueWindow } from "./workQueuePresentation";
import { useActorWorkSearch } from "./useActorWorkSearch";

vi.mock("../api", () => ({ loadActorWorkPage: vi.fn() }));

const scope = { tenantID: "bank", legalEntityID: "entity", principalID: "actor" };
const page: ActorWorkPage = {
  items: [{ id: "assignment", type: "MATTER", title: "Review audit", why_now: "Review due", scope: "Ops", state: "Ready", evidence: "Received", owner: "Reviewer", due_at: "", primary_action: "Inspect" }], next_cursor: "second-page", has_more: true,
  as_of: "2026-10-26T12:00:00Z",
  coverage: "WORKFLOW_ASSIGNMENTS_ONLY", count_basis: "NOT_AVAILABLE",
  scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "actor" },
};

beforeEach(() => vi.resetAllMocks());

describe("source Work query paging", () => {
  it("sends the same active status and due bounds through every cursor", async () => {
    vi.mocked(loadActorWorkPage)
      .mockResolvedValueOnce(page)
      .mockResolvedValueOnce({ ...page, next_cursor: undefined, has_more: false });

    const { result } = renderHook(() =>
      useActorWorkSearch({ search: "audit", scopeKey: "actor:entity", enabled: true, scope, status: "In progress", dueWindow: "7" }));

    await waitFor(() => expect(result.current.state).toBe("live"));
    const first = vi.mocked(loadActorWorkPage).mock.calls[0]!;
    expect(first[0]).toBe("audit");
    expect(first[2]?.status).toBe("IN_PROGRESS");
    expect(first[2]?.due_before).toMatch(/Z$/);

    act(() => result.current.loadMore());
    await waitFor(() => expect(result.current.hasMore).toBe(false));
    expect(vi.mocked(loadActorWorkPage).mock.calls[1]).toEqual([
      "audit", "second-page", first[2],
    ]);
  });

  it("invalidates old pages immediately when the selected source filter changes", async () => {
    vi.mocked(loadActorWorkPage).mockResolvedValue(page);
    const { result, rerender } = renderHook(
      ({ dueWindow }: { dueWindow: WorkDueWindow }) =>
        useActorWorkSearch({ search: "audit", scopeKey: "actor:entity", enabled: true, scope, status: "Ready", dueWindow }),
      { initialProps: { dueWindow: "ALL" as WorkDueWindow } },
    );

    await waitFor(() => expect(result.current.state).toBe("live"));
    rerender({ dueWindow: "OVERDUE" });
    expect(result.current.state).toBe("loading");
    expect(result.current.items).toHaveLength(0);
    await waitFor(() => expect(vi.mocked(loadActorWorkPage).mock.calls.length).toBe(2));
    const second = vi.mocked(loadActorWorkPage).mock.calls[1]!;
    expect(second[2]?.status).toBe("READY");
    expect(second[2]?.due_before).toMatch(/Z$/);
  });
});
