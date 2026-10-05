import type { StatusTone } from "./components/ui";

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
