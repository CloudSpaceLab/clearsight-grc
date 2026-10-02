import type { StatusTone } from "../ui";
import type { RiskAppetitePosition, RiskAssessment, RiskAssessmentKind, RiskStatus } from "../../riskTypes";

const dateTime = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  timeZone: "UTC",
});

export function riskStatusLabel(status: RiskStatus): string {
  if (status === "ACTIVE") return "Active";
  if (status === "DRAFT") return "Draft";
  return "Retired";
}

export function riskStatusTone(status: RiskStatus): StatusTone {
  if (status === "ACTIVE") return "info";
  if (status === "DRAFT") return "neutral";
  return "unknown";
}

export function appetiteLabel(position: RiskAppetitePosition | undefined, hasCurrentStatement = true): string {
  if (!position) return "Not assessed";
  if (position === "BREACHED") return "Outside appetite";
  if (position === "APPROACHING") return "Near appetite limit";
  if (position === "WITHIN") return "Within appetite";
  return hasCurrentStatement ? "Position unknown" : "No current appetite";
}

export function appetiteTone(position: RiskAppetitePosition | undefined): StatusTone {
  if (position === "BREACHED") return "error";
  if (position === "APPROACHING") return "warning";
  if (position === "WITHIN") return "success";
  return "unknown";
}

export function currentAssessmentPosition(riskVersion: number, assessment: RiskAssessment | undefined): RiskAppetitePosition | undefined {
  return assessment?.risk_version === riskVersion ? assessment.appetite_position : undefined;
}

export function currentAppetiteLabel(riskVersion: number, assessment: RiskAssessment | undefined, hasCurrentStatement = true): string {
  if (assessment && assessment.risk_version !== riskVersion) return "Reassessment needed";
  return appetiteLabel(assessment?.appetite_position, hasCurrentStatement);
}

export function currentAppetiteTone(riskVersion: number, assessment: RiskAssessment | undefined): StatusTone {
  if (assessment && assessment.risk_version !== riskVersion) return "unknown";
  return appetiteTone(assessment?.appetite_position);
}

export function assessmentKindLabel(kind: RiskAssessmentKind): string {
  const labels: Record<RiskAssessmentKind, string> = {
    INHERENT: "Inherent",
    CURRENT: "Current",
    RESIDUAL: "Residual",
    TARGET: "Target",
    STRESSED: "Stressed",
    ACCEPTED: "Accepted",
  };
  return labels[kind];
}

export function formatRiskDate(value: string | undefined): string {
  if (!value) return "Not recorded";
  const date = new Date(value);
  return Number.isFinite(date.getTime()) ? dateTime.format(date) : "Date unavailable";
}

export function scopeEntries(scope: Record<string, unknown>): Array<{ label: string; value: string }> {
  return Object.entries(scope)
    .filter(([, value]) => value !== null && value !== undefined && value !== "")
    .slice(0, 12)
    .map(([key, value]) => ({ label: humanizeKey(key), value: readableValue(value) }));
}

export function dimensionSummary(dimensions: Record<string, unknown>): string {
  const entries = Object.entries(dimensions)
    .filter(([, value]) => ["string", "number", "boolean"].includes(typeof value))
    .slice(0, 4);
  if (!entries.length) return "Dimensions recorded";
  return entries.map(([key, value]) => `${humanizeKey(key)} ${String(value)}`).join(" · ");
}

function humanizeKey(value: string): string {
  const text = value.replaceAll("_", " ").replaceAll("-", " ").trim();
  return text ? text[0]!.toUpperCase() + text.slice(1) : "Scope";
}

function readableValue(value: unknown): string {
  if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") return String(value);
  if (Array.isArray(value)) {
    const readable = value.filter((item) => ["string", "number", "boolean"].includes(typeof item)).slice(0, 5);
    return readable.length ? readable.join(", ") : "Recorded";
  }
  return "Recorded";
}
