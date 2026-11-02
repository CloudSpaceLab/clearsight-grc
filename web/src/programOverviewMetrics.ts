import type { ProgramSection } from "./appRouting";
import type { ProgramAggregate, ProgramComplianceSnapshot } from "./types";
import type { MetricCardQuality } from "./components/ui/MetricCard";

export type ProgramOverviewMetric = {
  id: "approved" | "assessed" | "issues" | "checks";
  label: string;
  value: number | string;
  detail: string;
  meta?: string;
  quality: MetricCardQuality;
  qualityLabel?: string;
  target: ProgramSection;
  actionLabel: string;
};

const validCount = (value: unknown): value is number =>
  typeof value === "number" && Number.isSafeInteger(value) && value >= 0;

function currentSnapshot(aggregate: ProgramAggregate) {
  const snapshot = aggregate.current_state;
  const timestamp = snapshot?.generated_at;
  if (!snapshot || !timestamp || !Number.isFinite(Date.parse(timestamp)) ||
      new Date(timestamp).getUTCFullYear() < 2000 ||
      !validCount(snapshot.program_version) || snapshot.program_version === 0) return null;
  return {
    snapshot,
    quality: snapshot.program_version === aggregate.program.version ? "current" as const : "stale" as const,
    meta: "Calculated " + timestamp.slice(0, 10) + " · Program v" + snapshot.program_version,
  };
}

function requirementCategory(aggregate: ProgramAggregate, calculation: ProgramComplianceSnapshot | undefined) {
  const policy = aggregate.program.calculation_policy?.active;
  if (!policy?.approved_at || !Array.isArray(policy.definition?.categories) ||
      !Array.isArray(calculation?.categories) ||
      policy.definition.revision !== calculation.policy_revision ||
      calculation.state === "NOT_ASSESSED") return null;
  const definition = policy.definition.categories.filter((item) => item.code === "REQUIREMENTS");
  const matches = calculation.categories.filter((item) => item.code === "REQUIREMENTS");
  if (definition.length !== 1 || matches.length !== 1 || definition[0]!.weight_bps !== matches[0]!.weight_bps) return null;
  const category = matches[0]!;
  if (![category.eligible_count, category.assessed_count, category.unknown_count, category.excluded_count]
      .every(validCount) || category.assessed_count > category.eligible_count ||
      category.unknown_count > category.eligible_count ||
      category.assessed_count + category.unknown_count > category.eligible_count ||
      !validCount(category.coverage_bps) || category.coverage_bps > 10000) return null;
  return category;
}

/** Only versioned persisted populations; no browser inference of support, gaps or compliance. */
export function programOverviewMetrics(aggregate: ProgramAggregate): readonly ProgramOverviewMetric[] {
  const source = currentSnapshot(aggregate);
  const category = source && requirementCategory(aggregate, source.snapshot.calculation);
  const assessedQuality: MetricCardQuality = !source || !category ? "unknown" :
    source.quality === "stale" ? "stale" :
    source.snapshot.calculation?.state === "PROVISIONAL" ||
    category.unknown_count > 0 || category.coverage_bps < 10000 ? "partial" : "current";
  const issueCount = source?.snapshot.open_matter_count;
  const knownIssues = validCount(issueCount);
  const approved = Array.isArray(aggregate.requirements)
    ? aggregate.requirements.filter((item) => item.status === "APPROVED").length : null;
  const checks = Array.isArray(aggregate.evidence_contracts)
    ? aggregate.evidence_contracts.filter((item) => item.status === "ACTIVE").length : null;

  return [
    {
      id: "approved", label: "Approved requirements", value: approved ?? "—",
      detail: "Approved obligations", meta: "Program v" + aggregate.program.version,
      quality: approved === null ? "unknown" : "current",
      target: "requirements-controls", actionLabel: "Open requirements",
    },
    {
      id: "assessed", label: "Assessed requirements", value: category?.assessed_count ?? "—",
      detail: category ? category.assessed_count + " of " + category.eligible_count + " eligible" : "Calculation unavailable",
      meta: source?.meta, quality: assessedQuality,
      qualityLabel: assessedQuality === "partial" && source?.snapshot.calculation?.state === "PROVISIONAL" ? "Provisional" : undefined,
      target: "requirements-controls", actionLabel: "Review requirements",
    },
    {
      id: "issues", label: "Open issues", value: knownIssues ? issueCount! : "—",
      detail: knownIssues ? "Recorded open issues" : "Issue count unavailable",
      meta: source?.meta,
      quality: source && knownIssues ? source.quality : "unknown",
      target: "issues-actions", actionLabel: "Review issues",
    },
    {
      id: "checks", label: "Active evidence checks", value: checks ?? "—",
      detail: "Configured evidence checks", meta: "Program v" + aggregate.program.version,
      quality: checks === null ? "unknown" : "current",
      target: "evidence-results", actionLabel: "Review evidence",
    },
  ];
}
