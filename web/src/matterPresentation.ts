import type { StatusTone } from "./components/ui";
import type { Matter } from "./types";

export function matterPriorityLabel(value: number): string {
  if (value >= 5) return "Critical";
  if (value === 4) return "High";
  if (value === 3) return "Medium";
  if (value === 2) return "Normal";
  return "Low";
}

export function matterPriorityTone(value: number): StatusTone {
  if (value >= 4) return "error";
  if (value === 3) return "warning";
  return "neutral";
}

export type MatterDeadlinePresentation = {
  state: "NONE" | "DUE" | "OVERDUE";
  label: "No due date" | "Due" | "Overdue";
  tone: StatusTone;
  dateTime?: string;
};

export function matterDeadlinePresentation(
  value?: string,
  now = Date.now(),
): MatterDeadlinePresentation {
  if (!value) return { state: "NONE", label: "No due date", tone: "neutral" };

  const due = new Date(value);
  if (!Number.isFinite(due.valueOf())) {
    return { state: "NONE", label: "No due date", tone: "neutral" };
  }

  if (due.valueOf() < now) {
    return { state: "OVERDUE", label: "Overdue", tone: "warning", dateTime: due.toISOString() };
  }
  return { state: "DUE", label: "Due", tone: "neutral", dateTime: due.toISOString() };
}


export type MatterContextKind = "OPERATIONAL_LOSS" | "INDICATOR" | "GENERIC";

export function matterContextKind(matter: Pick<Matter, "type" | "source_type" | "source_id" | "trigger_type">): MatterContextKind {
  const sourceType = matter.source_type?.trim().toUpperCase();
  if (matter.type === "OPERATIONAL_LOSS" && sourceType === "OPERATIONAL_LOSS" && matter.source_id?.trim()) {
    return "OPERATIONAL_LOSS";
  }
  if (
    sourceType === "MONITORING_RESULT"
    && matter.source_id?.trim()
    && (matter.type === "KRI_BREACH" || matter.trigger_type === "MONITORING_RESULT_ADVERSE")
  ) {
    return "INDICATOR";
  }
  return "GENERIC";
}
