import { useEffect, useState } from "react";
import { loadProgramSummaries } from "../../api";
import { hasProgramAssessment, needsProgramAssessment } from "../../programPortfolioPresentation";
import { listRisks } from "../../riskApi";
import type { RiskPage } from "../../riskTypes";
import type { ProgramSummary, SummaryPage } from "../../summaryTypes";
import { Button, StatusBadge } from "../ui";

type Props = {
  organizationScopeID?: string;
  refreshToken?: string;
  onOpenProgram?: (id: string) => void;
  onOpenRisk?: (id: string) => void;
  onOpenPrograms?: () => void;
  onOpenRisks?: () => void;
  loadPrograms?: typeof loadProgramSummaries;
  loadRisks?: typeof listRisks;
};
type Result<T> = { scope: string; state: "loading" | "ready" | "error"; data?: T };

export function OverviewRecords({ organizationScopeID, refreshToken, onOpenProgram, onOpenRisk, onOpenPrograms, onOpenRisks, loadPrograms = loadProgramSummaries, loadRisks = listRisks }: Props) {
  const scope = organizationScopeID ?? "";
  const [programs, setPrograms] = useState<Result<SummaryPage<ProgramSummary>>>({ scope, state: "loading" });
  const [risks, setRisks] = useState<Result<RiskPage>>({ scope, state: "loading" });
  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    setPrograms({ scope, state: "loading" });
    setRisks({ scope, state: "loading" });
    void loadPrograms({ organizationScopeID, limit: 4 }).then((data) => {
      if (!active) return;
      if ((data.organization_scope_id ?? "") !== scope) setPrograms({ scope, state: "error" });
      else setPrograms({ scope, state: "ready", data });
    }).catch(() => { if (active) setPrograms({ scope, state: "error" }); });
    void loadRisks({ organizationScopeID, appetitePosition: "BREACHED", status: "ACTIVE", limit: 4 }, controller.signal)
      .then((data) => { if (active) setRisks({ scope, state: "ready", data }); })
      .catch(() => { if (active) setRisks({ scope, state: "error" }); });
    return () => { active = false; controller.abort(); };
  }, [scope, organizationScopeID, refreshToken, loadPrograms, loadRisks]);

  const programResult = programs.scope === scope ? programs : { scope, state: "loading" as const };
  const riskResult = risks.scope === scope ? risks : { scope, state: "loading" as const };
  return <>
    <section className="overview-records overview-programs" aria-labelledby="overview-programs-heading">
      <header><h2 id="overview-programs-heading">Program assurance</h2>{onOpenPrograms && <Button variant="quiet" size="compact" onPress={onOpenPrograms}>All programs</Button>}</header>
      {programResult.state === "loading" ? <p role="status">Loading programs…</p>
        : programResult.state === "error" ? <p>Program information is unavailable.</p>
        : programResult.data?.items.length ? <div className="overview-records__list">{programResult.data.items.map((item) => {
          const missing = !hasProgramAssessment(item), stale = needsProgramAssessment(item);
          const label = missing ? "Unknown" : stale ? "Out of date" : item.state_label || "Unknown";
          const tone = stale ? "unknown" : item.overall_state === "CURRENT" ? "success" : ["OVERDUE", "GAP_IDENTIFIED"].includes(item.overall_state) ? "error" : "warning";
          const requirements = Number.isInteger(item.requirement_count) && item.requirement_count >= 0 ? `${item.requirement_count} ${item.requirement_count === 1 ? "requirement" : "requirements"}` : "Requirement count unavailable";
          return <article className="overview-records__row" key={item.program.id}><div><strong>{item.program.name}</strong><small>{requirements}</small></div><StatusBadge tone={tone}>{label}</StatusBadge>{onOpenProgram && <Button size="compact" aria-label={`Open ${item.program.name}`} onPress={() => onOpenProgram(item.program.id)}>Open</Button>}</article>;
        })}</div> : <p>No programs in this scope.</p>}
      {programResult.data?.next_cursor && <small>More programs available.</small>}
    </section>
    <section className="overview-records overview-risks" aria-labelledby="overview-risks-heading">
      <header><h2 id="overview-risks-heading">Operational risk priorities</h2>{onOpenRisks && <Button variant="quiet" size="compact" onPress={onOpenRisks}>Risk register</Button>}</header>
      {riskResult.state === "loading" ? <p role="status">Loading risks…</p>
        : riskResult.state === "error" ? <p>Risk information is unavailable.</p>
        : riskResult.data?.items.length ? <div className="overview-records__list">{riskResult.data.items.map((item) => <article className="overview-records__row" key={item.risk.id}><StatusBadge tone="error">Above appetite</StatusBadge><div><strong>{item.risk.name}</strong><small>{item.risk.code}</small></div>{onOpenRisk && <Button size="compact" aria-label={`Review ${item.risk.name}`} onPress={() => onOpenRisk(item.risk.id)}>Review</Button>}</article>)}</div> : <p>No active risks recorded above appetite in this scope.</p>}
      {riskResult.data?.next_cursor && <small>More risks available.</small>}
    </section>
  </>;
}
