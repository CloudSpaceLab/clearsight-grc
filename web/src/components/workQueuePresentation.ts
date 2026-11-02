import type { AttentionItem } from "../types";

// The Today projection is an actor-authorized, bounded work feed, not an
// organization-wide count or a historical/completed-work query.
export type WorkQueueFilter = "ALL" | "OVERDUE" | "DUE_SOON" | "APPROVALS" | "EVIDENCE";
export type WorkQueueCounts = Record<WorkQueueFilter, number>;
export type WorkDueWindow = "ALL" | "7" | "30" | "OVERDUE";

const sevenDaysMs = 7 * 24 * 60 * 60 * 1000;
const earliestRealDate = Date.UTC(2000, 0, 1);

export function matchesWorkFilter(item: AttentionItem, filter: WorkQueueFilter, now: number): boolean {
  if (filter === "ALL") return true;
  if (filter === "APPROVALS") return item.intervention_class === "AUTHORIZATION";
  if (filter === "EVIDENCE") return item.intervention_class === "EVIDENCE_EXCEPTION";
  const due = Date.parse(item.due_at);
  if (!Number.isFinite(due) || due < earliestRealDate) return false;
  return filter === "OVERDUE" ? due < now : due >= now && due <= now + sevenDaysMs;
}

export function matchesDueWindow(item: AttentionItem, window: WorkDueWindow, asOf: number): boolean {
  if (window === "ALL") return true;
  const due = Date.parse(item.due_at);
  if (!Number.isFinite(due) || due < earliestRealDate) return false;
  if (window === "OVERDUE") return due < asOf;
  return due <= asOf + Number(window) * 24 * 60 * 60 * 1000;
}

/** Restrict this search to the already authorized Today response. It is not a
 * server search or a count of all work in the tenant. */
export function filterLoadedWork(
  items: readonly AttentionItem[],
  filter: WorkQueueFilter,
  query: string,
  status: string | undefined,
  asOf: number,
  dueWindow: WorkDueWindow = "ALL",
): AttentionItem[] {
  const terms = query.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return items.filter((item) => {
    if (!matchesWorkFilter(item, filter, asOf) || !matchesDueWindow(item, dueWindow, asOf) || (status && item.state !== status)) return false;
    if (terms.length === 0) return true;
    const searchable = [item.title, item.scope, item.owner, item.state, item.evidence]
      .join(" ").toLocaleLowerCase();
    return terms.every((term) => searchable.includes(term));
  });
}

export function countLoadedWork(items: readonly AttentionItem[], now: number): WorkQueueCounts {
  const counts: WorkQueueCounts = { ALL: items.length, OVERDUE: 0, DUE_SOON: 0, APPROVALS: 0, EVIDENCE: 0 };
  for (const item of items) {
    if (matchesWorkFilter(item, "OVERDUE", now)) counts.OVERDUE++;
    if (matchesWorkFilter(item, "DUE_SOON", now)) counts.DUE_SOON++;
    if (matchesWorkFilter(item, "APPROVALS", now)) counts.APPROVALS++;
    if (matchesWorkFilter(item, "EVIDENCE", now)) counts.EVIDENCE++;
  }
  return counts;
}
