import type { RiskSummary } from "./riskTypes";
import { currentAppetiteTone } from "./components/risks/riskPresentation";
import type { DistributionSegment } from "./components/ui/StackedDistribution";

export function riskAppetiteDistribution(rows: readonly RiskSummary[]): DistributionSegment[] {
  const counts = { breached: 0, approaching: 0, within: 0, unknown: 0 };

  for (const item of rows) {
    const tone = currentAppetiteTone(item.risk.version, item.latest_assessment, item.active_appetite);
    if (tone === "error") counts.breached += 1;
    else if (tone === "warning") counts.approaching += 1;
    else if (tone === "success") counts.within += 1;
    else counts.unknown += 1;
  }

  return [
    { id: "breached", label: "Outside appetite", tone: "error", count: counts.breached },
    { id: "approaching", label: "Near appetite limit", tone: "warning", count: counts.approaching },
    { id: "within", label: "Within appetite", tone: "success", count: counts.within },
    { id: "unknown", label: "Unknown or outdated", tone: "unknown", count: counts.unknown },
  ];
}
