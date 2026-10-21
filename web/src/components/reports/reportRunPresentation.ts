import type { ReportDataset, ReportRun } from "../../reportingTypes";
import type { StatusTone } from "../ui";

export type ReportArea = "ALL" | "VENDORS" | "PROGRAMS" | "WORK" | "PROCESSING";

export function reportDatasetLabel(dataset: ReportDataset | ReportArea) {
  switch (dataset) {
    case "VENDORS": return "Vendors";
    case "PROGRAMS": return "Programs";
    case "MATTER_BOARD_BRIEF": return "Board brief";
    case "WORK":
    case "MATTERS":
    case "MATTER_EXCEPTIONS": return "Work";
    case "PROCESSING":
    case "PROCESSING_ACTIVITIES":
    case "PROCESSING_ACTIVITY_EXCEPTIONS": return "Processing activities";
    default: return "All reports";
  }
}

export function reportBelongsToArea(dataset: ReportDataset, area: Exclude<ReportArea, "ALL">) {
  if (area === "PROCESSING") return dataset === "PROCESSING_ACTIVITIES" || dataset === "PROCESSING_ACTIVITY_EXCEPTIONS";
  if (area === "WORK") return dataset === "MATTERS" || dataset === "MATTER_EXCEPTIONS" || dataset === "MATTER_BOARD_BRIEF";
  return dataset === area;
}

export function isExpired(run: ReportRun) {
  return Date.parse(run.expires_at) <= Date.now();
}

export function isReportAvailable(run: ReportRun) {
  return run.status === "READY" && Number.isFinite(Date.parse(run.expires_at)) && !isExpired(run);
}

export function runStatusLabel(run: ReportRun): string {
  if (run.status === "READY") {
    if (isExpired(run)) return "Expired";
    return isReportAvailable(run) ? "Ready" : "Availability unknown";
  }
  switch (run.status) {
    case "QUEUED": return "Queued";
    case "RUNNING": return "Generating";
    case "FAILED": return "Failed";
    default: return "Unknown";
  }
}

export function runStatusTone(run: ReportRun): StatusTone {
  if (run.status === "READY") {
    if (isExpired(run)) return "warning";
    return isReportAvailable(run) ? "success" : "unknown";
  }
  switch (run.status) {
    case "QUEUED":
    case "RUNNING": return "info";
    case "FAILED": return "error";
    default: return "unknown";
  }
}

export function formatDateTime(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

export function humanizeCode(value: string) {
  return value.replaceAll("_", " ").replaceAll("-", " ").toLowerCase().replace(/\b\w/g, (character) => character.toUpperCase());
}
