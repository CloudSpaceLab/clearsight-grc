import type { RCSACycleSummary, RCSAStatus } from "./rcsaTypes";
import type { DistributionSegment } from "./components/ui/StackedDistribution";

const stages: ReadonlyArray<{ id: RCSAStatus; label: string; tone: DistributionSegment["tone"] }> = [
  { id: "DRAFT", label: "Draft", tone: "unknown" },
  { id: "ASSESSMENT_OPEN", label: "First-line collection", tone: "neutral" },
  { id: "AWAITING_CHALLENGE", label: "Awaiting challenge or resolution", tone: "warning" },
  { id: "COMPLETED", label: "Completed", tone: "success" },
  { id: "CANCELLED", label: "Cancelled", tone: "unknown" },
];

export function rcsaCycleStageDistribution(items: readonly RCSACycleSummary[]): DistributionSegment[] {
  const count = new Map<RCSAStatus, number>();
  for (const item of items) count.set(item.cycle.status, (count.get(item.cycle.status) ?? 0) + 1);
  return stages.map(({ id, label, tone }) => ({ id, label, tone, count: count.get(id) ?? 0 }));
}
