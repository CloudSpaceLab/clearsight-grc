import type { ProgramSummary } from "./summaryTypes";
import type { DistributionSegment } from "./components/ui/StackedDistribution";

export type ProgramListSummary = Omit<ProgramSummary, "open_matter_count"> & { open_matter_count?: number };

type Bucket = "attention" | "current" | "setup" | "notApplicable";

const attentionStates = new Set([
  "AT_RISK", "GAP_IDENTIFIED", "EVIDENCE_INSUFFICIENT", "IMPLEMENTATION_PENDING", "OVERDUE",
]);

export function hasProgramAssessment(item: ProgramListSummary) {
  return Number.isInteger(item.assessed_program_version) && item.assessed_program_version > 0;
}

export function needsProgramAssessment(item: ProgramListSummary) {
  return !hasProgramAssessment(item) || item.projection_stale
    || item.assessed_program_version !== item.program_version;
}

export function programPortfolioBucket(item: ProgramListSummary): Bucket {
  if (needsProgramAssessment(item) || item.program.status === "DRAFT") return "setup";
  if (attentionStates.has(item.overall_state)) return "attention";
  if (item.overall_state === "CURRENT") return "current";
  if (item.overall_state === "NOT_APPLICABLE") return "notApplicable";
  return "setup";
}

export function summarizeProgramPortfolio(items: readonly ProgramListSummary[]) {
  const counts = { attention: 0, current: 0, setup: 0, notApplicable: 0 };
  const knownIssuePrograms: Array<{ id: string; label: string; count: number; owner: string }> = [];
  let excludedFromIssueComparison = 0;

  for (const item of items) {
    counts[programPortfolioBucket(item)] += 1;
    const open = item.open_matter_count;
    if (needsProgramAssessment(item) || item.program.status === "DRAFT" || typeof open !== "number" || !Number.isInteger(open) || open < 0) {
      excludedFromIssueComparison += 1;
      continue;
    }
    if (open > 0) knownIssuePrograms.push({
      id: item.program.id,
      label: item.program.name,
      count: open,
      owner: item.program.owning_function || "Function not recorded",
    });
  }

  knownIssuePrograms.sort((a, b) => b.count - a.count || a.label.localeCompare(b.label));

  const segments: DistributionSegment[] = [
    { id: "attention", label: "Follow-up", tone: "warning", count: counts.attention },
    { id: "current", label: "Current", tone: "success", count: counts.current },
    { id: "setup", label: "Needs assessment", tone: "unknown", count: counts.setup },
    { id: "not-applicable", label: "Not applicable", tone: "neutral", count: counts.notApplicable },
  ];

  return { ...counts, segments, knownIssuePrograms: knownIssuePrograms.slice(0, 4), excludedFromIssueComparison };
}
