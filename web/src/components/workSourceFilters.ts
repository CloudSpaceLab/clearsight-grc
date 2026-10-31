import type { ActorWorkFilters } from "../api";
import type { WorkDueWindow } from "./workQueuePresentation";
import { validWorkTimestamp } from "./workSourceReceipt";

const dayMs = 24 * 60 * 60 * 1000;

export type WorkDeadlineDrill = { kind: "OVERDUE" | "DUE_SOON"; asOf: string };

export function workDeadlineFilters({ kind, asOf }: WorkDeadlineDrill): ActorWorkFilters {
  if (!validWorkTimestamp(asOf)) throw new Error("Assignment count time is unavailable");
  if (kind === "OVERDUE") return { due_before: asOf };
  // Date truncates microseconds. Retain the remainder and advance the inclusive
  // source boundary by exactly one microsecond, including millisecond carry.
  const micros = Number((asOf.split(".")[1]?.slice(0, -1) ?? "").padEnd(6, "0").slice(3));
  const end = new Date(Date.parse(asOf) + 7 * dayMs + (micros === 999 ? 1 : 0));
  const exclusiveEnd = end.toISOString().replace("Z", `${String((micros + 1) % 1000).padStart(3, "0")}Z`);
  return { due_from: asOf, due_before: exclusiveEnd };
}

export function isSourceWorkStatus(value: string): boolean {
  return ["READY", "IN_PROGRESS", "BLOCKED", "ESCALATED"].includes(
    value.trim().toUpperCase().replace(/\s+/g, "_"),
  );
}

export function workSourceFilters(status: string | undefined, window: WorkDueWindow, now: number): ActorWorkFilters {
  const filters: ActorWorkFilters = {};
  if (status) {
    const normalized = status.trim().toUpperCase().replace(/\s+/g, "_");
    if (normalized === "READY" || normalized === "IN_PROGRESS" ||
        normalized === "BLOCKED" || normalized === "ESCALATED") {
      filters.status = normalized;
    }
  }

  if (!Number.isFinite(now)) return filters;
  if (window === "OVERDUE") {
    filters.due_before = new Date(now).toISOString();
  } else if (window === "7" || window === "30") {
    // The source SQL uses an exclusive end; the UI includes the final millisecond.
    filters.due_before = new Date(now + Number(window) * dayMs + 1).toISOString();
  }
  return filters;
}
