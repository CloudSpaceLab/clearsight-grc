// Deterministic browser evidence. Imported only from evidenceMain; never from the customer entry.
import type { GroupOversightSnapshot } from "./groupOversightApi";
import type { GroupPostureBundle, GroupPostureChild } from "./groupPostureApi";
import type { ReportingPeriodQuery } from "./reportingPeriod";
import { GroupOversightWorkspace } from "./components/oversight/GroupOversightWorkspace";

const at = "2026-10-07T12:00:00Z";
const children = [
  { legal_entity_id: "opco-alpha", legal_entity_code: "NG", legal_entity_name: "Alpha Bank", jurisdiction: "NG", counts: { risks_outside_appetite: 6, indicator_breaches: 2, assurance_failures: 1, loss_events: 4 } },
  { legal_entity_id: "opco-beta", legal_entity_code: "GH", legal_entity_name: "Beta Bank", jurisdiction: "GH", counts: { risks_outside_appetite: 3, indicator_breaches: 1, assurance_failures: 1, loss_events: 1 } },
  { legal_entity_id: "opco-gamma", legal_entity_code: "KE", legal_entity_name: "Gamma Bank", jurisdiction: "KE", counts: { risks_outside_appetite: 0, indicator_breaches: 0, assurance_failures: 0, loss_events: 0 } },
] as const;

function posture(partial: boolean, period: ReportingPeriodQuery): GroupPostureBundle {
  const records: GroupPostureChild[] = children.map((entity, index) => {
    const missing = partial && index === 2;
    return {
      ...entity,
      risk_state: missing ? "MISSING" : "AVAILABLE",
      completeness: missing ? "UNKNOWN" : "COMPLETE",
      source_id: missing ? undefined : `source-${index + 1}`,
      source_generated_at: missing ? undefined : at,
      source_revision: missing ? undefined : "enterprise-domain-v1",
      definition_revision: missing ? undefined : "enterprise-domain-v1",
      counts: { ...entity.counts },
      unknown: 0,
      excluded: 0,
      freshness: missing ? "STALE" : "CURRENT",
    };
  });
  const riskCount = records.filter((item) => item.risk_state !== "MISSING").length;
  const longerPeriod = period.start_date < "2026-09-08";
  return {
    generated_at: at,
    period_start: `${period.start_date}T00:00:00Z`,
    period_end: at,
    definition_revision: "enterprise-domain-v1",
    risk_coverage: { authorized_children: 3, included_children: riskCount, missing_children: 3 - riskCount, stale_children: 0, partial_children: 0, complete: riskCount === 3 },
    loss_coverage: { authorized_children: 3, included_children: 3, missing_children: 0, stale_children: 0, partial_children: 0, complete: true },
    counts: { risks_outside_appetite: 9, indicator_breaches: 3, assurance_failures: 2, loss_events: longerPeriod ? 9 : 5 },
    children: records.map((item) => ({
      ...item,
      counts: { ...item.counts, loss_events: longerPeriod && item.legal_entity_id === "opco-alpha" ? 8 : item.counts.loss_events },
    })),
  };
}

function attention(): GroupOversightSnapshot {
  const counts = { critical_high: 5, overdue: 2, due_soon: 1, routing_failures: 1, unassigned: 0, outcome_failures: 3 };
  return {
    revision_id: "group-oversight-t7",
    generated_at: at,
    projection_version: "group-oversight-v1",
    freshness: "CURRENT",
    coverage: { authorized_children: 3, included_children: 3, missing_children: 0, stale_children: 0, complete: true },
    record_coverage: { population: 11, excluded: 0, unknown: 0 },
    counts,
    children: children.map((child, index) => ({
      legal_entity_id: child.legal_entity_id,
      legal_entity_code: child.legal_entity_code,
      legal_entity_name: child.legal_entity_name,
      jurisdiction: child.jurisdiction,
      state: "AVAILABLE" as const,
      child_snapshot_id: `snapshot-${child.legal_entity_code}`,
      child_generated_at: at,
      child_projection_version: "oversight-v5",
      coverage: { population: [5, 4, 2][index] ?? 0, excluded: 0, unknown: 0 },
      counts: [
        { critical_high: 3, overdue: 1, due_soon: 1, routing_failures: 0, unassigned: 0, outcome_failures: 1 },
        { critical_high: 2, overdue: 1, due_soon: 0, routing_failures: 1, unassigned: 0, outcome_failures: 2 },
        { critical_high: 0, overdue: 0, due_soon: 0, routing_failures: 0, unassigned: 0, outcome_failures: 0 },
      ][index]!,
    })),
  };
}

export function HomeT7GroupEvidence({ partial }: { partial: boolean }) {
  return <main className="oversight-workspace">
    <GroupOversightWorkspace
      organizationName="Example Group"
      initialSnapshot={attention()}
      loadPosture={async (period) => posture(partial, period)}
      onOpenLegalEntity={(id) => { document.querySelector("main")?.setAttribute("data-opened-opco", id); }}
      onOpenWork={() => { document.querySelector("main")?.setAttribute("data-work-handoff", "true"); }}
      now={new Date(at)}
    />
  </main>;
}
