import type { StatusTone } from "./components/ui";
import type { RCSACycle, RCSAHandoff, RCSAPhaseStage, RCSAStatus, RCSATriggerKind } from "./rcsaTypes";

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

export type RCSAPhasePathStep = {
  id: "COLLECTION" | "INDEPENDENT_CHALLENGE" | "RISK_ACCEPTANCE" | "REMEDIATION_VERIFICATION";
  label: string;
  state: "complete" | "current" | "pending" | "not_required";
};

export function rcsaPhaseTone(stage: RCSAPhaseStage): StatusTone {
  switch (stage) {
    case "RISK_ACCEPTANCE": return "success";
    case "REMEDIATION_VERIFICATION": return "warning";
    case "INDEPENDENT_CHALLENGE": return "warning";
    case "COLLECTION": return "info";
    case "COMPLETE": return "success";
    case "CANCELLED": return "neutral";
    default: return "unknown";
  }
}

export function rcsaPhasePath(stage: RCSAPhaseStage): RCSAPhasePathStep[] {
  const steps: RCSAPhasePathStep[] = [
    { id: "COLLECTION", label: "First-line collection", state: "pending" },
    { id: "INDEPENDENT_CHALLENGE", label: "Independent challenge", state: "pending" },
    { id: "RISK_ACCEPTANCE", label: "Risk acceptance", state: "pending" },
    { id: "REMEDIATION_VERIFICATION", label: "Remediation verification", state: "pending" },
  ];
  if (stage === "COLLECTION") {
    steps[0].state = "current";
    return steps;
  }
  if (stage === "INDEPENDENT_CHALLENGE") {
    steps[0].state = "complete";
    steps[1].state = "current";
    return steps;
  }
  if (stage === "RISK_ACCEPTANCE") {
    steps[0].state = "complete";
    steps[1].state = "complete";
    steps[2].state = "current";
    steps[3].state = "not_required";
    return steps;
  }
  if (stage === "REMEDIATION_VERIFICATION") {
    steps[0].state = "complete";
    steps[1].state = "complete";
    steps[2].state = "not_required";
    steps[3].state = "current";
    return steps;
  }
  if (stage === "COMPLETE") {
    steps[0].state = "complete";
    steps[1].state = "complete";
  }
  return steps;
}

export function formatRCSADate(value?: string) {
  if (!value) return "Not recorded";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Not recorded";
  return new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(date).replace("Sept", "Sep");
}

export function formatRCSAPeriod(start?: string, end?: string) {
  if (!start && !end) return "Not recorded";
  if (start && end) return `${formatRCSADate(start)} – ${formatRCSADate(end)}`;
  return start ? `From ${formatRCSADate(start)}` : `To ${formatRCSADate(end)}`;
}

export function rcsaOwnerLabel(cycle: RCSACycle, displayName?: string) {
  return displayName?.trim() || cycle.first_line_owner_principal_id || "Not assigned";
}
