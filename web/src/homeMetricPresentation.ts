import type { HomeMetric, MetricCompleteness } from "./metricApi";
import type { OversightMetric } from "./appRouting";
import type { MetricCardQuality, StatusTone } from "./components/ui";

export type HomeMetricFilter = OversightMetric | "all";

export const headlineMetricDefinitions = [
  { id: "critical_high_open", label: "Critical and high", detail: "Open priority 4–5 issues" },
  { id: "overdue_open", label: "Overdue", detail: "Open issues past their due date" },
  { id: "routing_gaps", label: "Routing gaps", detail: "Active work without a resolved recipient" },
  { id: "outcome_failures", label: "Outcome failures", detail: "Latest outcome check failed or inconclusive" },
] as const;

export function homeMetricQuality(metric: Pick<HomeMetric, "freshness" | "completeness">): MetricCardQuality {
  if (metric.freshness !== "CURRENT") return "stale";
  if (metric.completeness === "UNKNOWN") return "unknown";
  if (metric.completeness === "PARTIAL") return "partial";
  return "current";
}

export function homeMetricTone(metric: Pick<HomeMetric, "id" | "condition">): StatusTone {
  if (metric.condition === "CLEAR") return "success";
  if (metric.id === "critical_high_open" || metric.id === "outcome_failures") return "error";
  return "warning";
}

export function homeMetricFilter(value: string): HomeMetricFilter {
  if (value === "critical-high" || value === "overdue" || value === "routing-gaps" || value === "outcome-failures") return value;
  return "all";
}

export function homeMetricDetail(id: string) {
  return headlineMetricDefinitions.find((item) => item.id === id)?.detail ?? "Current governed metric";
}

export function homeMetricMeta(metric: Pick<HomeMetric, "population" | "excluded" | "unknown" | "basis">) {
  const basis = metric.basis === "CURRENT_POSTURE" ? "Current posture · " : "";
  return `${basis}${metric.population} checked · ${knownCount(metric.excluded)} excluded · ${knownCount(metric.unknown)} unknown`;
}

export function completenessLabel(value: MetricCompleteness) {
  if (value === "PARTIAL") return "Coverage incomplete";
  if (value === "UNKNOWN") return "Coverage unknown";
  return "Complete";
}

function knownCount(value: number | undefined) {
  return value === undefined ? "unknown" : String(value);
}
