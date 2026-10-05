import type { StatusTone } from "../ui";
import type { RiskAppetitePosition, RiskAppetiteStatement, RiskAssessment, RiskAssessmentKind, RiskControlDetail, RiskControlImplementationStatus, RiskStatus } from "../../riskTypes";

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

export function currentAppetiteLabel(
  riskVersion: number,
  assessment: RiskAssessment | undefined,
  activeAppetite: RiskAppetiteStatement | undefined,
): string {
  if (!assessment) return "Not assessed";
  if (assessment.risk_version !== riskVersion) return "Reassessment needed";
  if (!assessment.appetite_statement_id) return "No current appetite";
  if (!activeAppetite) return "No current appetite";
  if (assessment.appetite_statement_id !== activeAppetite.id) return "Reassessment needed";
  return appetiteLabel(assessment.appetite_position);
}

export function currentAppetiteTone(
  riskVersion: number,
  assessment: RiskAssessment | undefined,
  activeAppetite: RiskAppetiteStatement | undefined,
): StatusTone {
  if (!assessment || assessment.risk_version !== riskVersion) return "unknown";
  if (!assessment.appetite_statement_id || !activeAppetite || assessment.appetite_statement_id !== activeAppetite.id) return "unknown";
  return appetiteTone(assessment.appetite_position);
}

export function controlImplementationStatusLabel(status: RiskControlImplementationStatus): string {
  if (status === "IN_PROGRESS") return "In progress";
  if (status === "IMPLEMENTED") return "Implemented";
  if (status === "INACTIVE") return "Inactive";
  if (status === "RETIRED") return "Retired";
  return "Planned";
}

export function controlImplementationStatusTone(status: RiskControlImplementationStatus): StatusTone {
  if (status === "IMPLEMENTED") return "success";
  if (status === "IN_PROGRESS") return "info";
  if (status === "INACTIVE") return "warning";
  if (status === "RETIRED") return "unknown";
  return "neutral";
}

export function controlEvidenceSummary(control: RiskControlDetail): string {
  if (!control.evidence.length) return "No active checks";
  const counts = new Map<string, number>();
  let missing = 0;
  for (const check of control.evidence) {
    if (!check.conclusion) {
      missing += 1;
      continue;
    }
    counts.set(check.conclusion, (counts.get(check.conclusion) ?? 0) + 1);
  }
  const parts = [...counts.entries()].map(([conclusion, count]) => `${count} ${evidenceConclusionLabel(conclusion)}`);
  if (missing) parts.push(`${missing} without result`);
  return `${control.evidence.length} active ${control.evidence.length === 1 ? "check" : "checks"} · ${parts.join(" · ")}`;
}

function evidenceConclusionLabel(value: string): string {
  if (value === "PARTIALLY_SUPPORTED") return "partially supported";
  return value.toLowerCase().replaceAll("_", " ");
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

export function assessmentRatingLabel(assessment: RiskAssessment | undefined, riskVersion?: number): string {
  if (!assessment) return "Not assessed";
  if (riskVersion !== undefined && assessment.risk_version !== riskVersion) return "Reassessment needed";
  for (const key of ["risk_level", "rating", "band"] as const) {
    const value = assessment.dimensions?.[key];
    if (typeof value === "string" && value.trim()) return readableRating(value);
  }
  return "See assessment";
}

export function assessmentRatingTone(assessment: RiskAssessment | undefined, riskVersion?: number): StatusTone {
  const value = assessmentRatingLabel(assessment, riskVersion).toUpperCase();
  if (value === "CRITICAL" || value === "SEVERE") return "error";
  if (value === "HIGH") return "warning";
  if (value === "MODERATE" || value === "MEDIUM") return "info";
  if (value === "LOW") return "success";
  return "unknown";
}

function readableRating(value: string): string {
  const normalized = value.trim().replaceAll("_", " ").toLowerCase();
  return normalized ? normalized[0]!.toUpperCase() + normalized.slice(1) : "See assessment";
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
