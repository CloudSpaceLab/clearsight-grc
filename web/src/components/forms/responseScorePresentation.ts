import type { ResponseScore } from "../../formsDistributionApi";
import type { StatusTone } from "../ui";

export function scorePresentation(score?: ResponseScore): { value: string; meaning: string } {
  if (!score) return { value: "Score unavailable", meaning: "Score not recorded" };
  if (score.state === "NOT_CONFIGURED") return { value: "Not scored", meaning: "No scoring profile applied" };
  if (score.state === "FAILED") return { value: "Score unavailable", meaning: "The response is available for review." };
  if (score.raw_score == null) return { value: "Score unavailable", meaning: "Score not recorded" };
  if (score.mode === "COMPLIANCE") return { value: `${formatNumber(score.raw_score)}% compliance`, meaning: score.band === "LOW" ? "Meets expected level" : score.band === "MODERATE" ? "Review advised" : "Below required level" };
  if (score.mode === "RISK") return { value: `${formatNumber(score.raw_score)}% risk`, meaning: score.band ? `${humanize(score.band)} concern` : "Risk score available" };
  return { value: `${formatNumber(score.raw_score)}%`, meaning: "Calculated score" };
}

export function concernText(score?: ResponseScore) { return score?.band ? `${humanize(score.band)} concern` : score?.state === "FAILED" ? "Score unavailable" : "Not classified"; }
export function concernTone(score?: ResponseScore): StatusTone {
  if (score?.state === "FAILED") return "warning";
  switch (score?.band) { case "CRITICAL": return "error"; case "HIGH": return "warning"; case "MODERATE": return "info"; case "LOW": return "success"; default: return "unknown"; }
}
export function coverageText(score?: ResponseScore) { return typeof score?.coverage === "number" ? `${formatNumber(score.coverage <= 1 ? score.coverage * 100 : score.coverage)}%` : "Not available"; }

function formatNumber(value: number) { return new Intl.NumberFormat(undefined, { maximumFractionDigits: 1 }).format(value); }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
