import type { RiskIndicatorState } from "../../riskTypes";
import type { StatusTone } from "../ui";

export function indicatorStateLabel(state: RiskIndicatorState) {
  if (state === "NORMAL") return "Normal";
  if (state === "WATCH") return "Watch";
  if (state === "BREACH") return "Breach";
  return "Unknown";
}

export function indicatorTone(state: RiskIndicatorState): StatusTone {
  if (state === "NORMAL") return "success";
  if (state === "WATCH") return "warning";
  if (state === "BREACH") return "error";
  return "unknown";
}

export function formatIndicatorDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.valueOf())) return value;
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

export function formatIndicatorCoverage(value: number) {
  return `${Math.round(value * 100)}%`;
}

export function monitoringBandLabel(value: string) {
  if (value === "LOW") return "Low concern";
  if (value === "MODERATE") return "Moderate concern";
  if (value === "HIGH") return "High concern";
  if (value === "CRITICAL") return "Critical concern";
  return "Concern not assessed";
}
