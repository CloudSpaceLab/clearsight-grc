import type { ProgramSummary } from "../summaryTypes";

const positiveVersion = (value: unknown): value is number => typeof value === "number" && Number.isSafeInteger(value) && value > 0;

export function programStatus(item: ProgramSummary): string {
  const version = item.program?.version;
  if (item.projection_stale === true || (positiveVersion(version) && positiveVersion(item.program_version) && item.program_version !== version) || (positiveVersion(version) && positiveVersion(item.assessed_program_version) && item.assessed_program_version !== version)) return "Out of date";
  if (item.projection_stale !== false || !positiveVersion(version) || !positiveVersion(item.program_version) || !positiveVersion(item.assessed_program_version) || !positiveVersion(item.projection_version) || !item.state_generated_at || !Number.isFinite(Date.parse(item.state_generated_at)) || !item.overall_state || !item.state_label?.trim()) return "Unknown";
  return item.state_label;
}

export function programCount(value: unknown): string {
  return typeof value === "number" && Number.isSafeInteger(value) && value >= 0 ? String(value) : "Unknown";
}
