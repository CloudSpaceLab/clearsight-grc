import type { StatusTone } from "./components/ui";
import type { RCSACycle, RCSAHandoff, RCSAStatus, RCSATriggerKind } from "./rcsaTypes";

export function rcsaStatusLabel(status: RCSAStatus) {
  switch (status) {
    case "DRAFT": return "Draft";
    case "ASSESSMENT_OPEN": return "First line";
    case "AWAITING_CHALLENGE": return "Challenge";
    case "COMPLETED": return "Completed";
    case "CANCELLED": return "Cancelled";
  }
}

export function rcsaStatusTone(status: RCSAStatus): StatusTone {
  switch (status) {
    case "COMPLETED": return "success";
    case "ASSESSMENT_OPEN": return "info";
    case "AWAITING_CHALLENGE": return "warning";
    case "CANCELLED": return "neutral";
    case "DRAFT": return "unknown";
  }
}

export function rcsaTriggerLabel(kind: RCSATriggerKind) {
  switch (kind) {
    case "SCHEDULED": return "Scheduled";
    case "CHANGE": return "Change";
    case "MANUAL": return "Manual";
  }
}

export function rcsaHandoffTone(handoff: RCSAHandoff): StatusTone {
  switch (handoff.stage) {
    case "COMPLETE": return "success";
    case "CHALLENGE": return "warning";
    case "FIRST_LINE": return "info";
    case "CANCELLED": return "neutral";
    default: return "unknown";
  }
}

export function formatRCSADate(value?: string) {
  if (!value) return "Not recorded";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Not recorded";
  return new Intl.DateTimeFormat("en", { day: "numeric", month: "short", year: "numeric" }).format(date);
}

export function formatRCSAPeriod(start?: string, end?: string) {
  if (!start && !end) return "Not recorded";
  if (start && end) return `${formatRCSADate(start)} – ${formatRCSADate(end)}`;
  return start ? `From ${formatRCSADate(start)}` : `To ${formatRCSADate(end)}`;
}

export function rcsaOwnerLabel(cycle: RCSACycle, displayName?: string) {
  return displayName?.trim() || cycle.first_line_owner_principal_id || "Not assigned";
}
