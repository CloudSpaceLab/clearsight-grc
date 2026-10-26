import type { ActorWorkFilters } from "../api";
import type { WorkDueWindow } from "./workQueuePresentation";

const dayMs = 24 * 60 * 60 * 1000;

export function isSourceWorkStatus(value: string): boolean {
  return ["READY", "IN_PROGRESS", "BLOCKED", "ESCALATED"].includes(
    value.trim().toUpperCase().replace(/\\s+/g, "_"),
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
