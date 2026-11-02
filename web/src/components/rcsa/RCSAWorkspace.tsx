import { useEffect, useMemo, useState } from "react";
import { getRCSACycle, listRCSACycles } from "../../rcsaApi";
import { rcsaCycleStageDistribution } from "../../rcsaCycleDistribution";
import type { RCSAControlSnapshot, RCSACycleDetail, RCSACycleSummary, RCSARiskSnapshot, RCSAStatus } from "../../rcsaTypes";
import { formatRCSADate, formatRCSAPeriod, rcsaHandoffTone, rcsaOwnerLabel, rcsaPhasePath, rcsaPhaseTone, rcsaStatusLabel, rcsaStatusTone, rcsaTriggerLabel } from "../../rcsaPresentation";
import { Button, Card, DataTable, EmptyState, FocusedSheet, Notice, SelectField, StackedDistribution, StatusBadge, Surface, type DataColumn } from "../ui";
import "./rcsa.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  embedded?: boolean;
  targetID?: string;
  onTarget: (id?: string) => void;
  onOpenEvidence?: (requestID: string, cycleID: string) => void;
  onOpenMatter?: (matterID: string, cycleID: string) => void;
};

type LoadState = "loading" | "live" | "error";
type StatusFilter = RCSAStatus | "ALL";

const statusOptions: ReadonlyArray<{ id: StatusFilter; label: string }> = [
  { id: "ALL", label: "All statuses" },
  { id: "DRAFT", label: "Draft" },
  { id: "ASSESSMENT_OPEN", label: "First line" },
  { id: "AWAITING_CHALLENGE", label: "Challenge" },
  { id: "COMPLETED", label: "Completed" },
  { id: "CANCELLED", label: "Cancelled" },
];

export function RCSAWorkspace({ organizationName, legalEntityName, embedded = false, targetID, onTarget, onOpenEvidence, onOpenMatter }: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [items, setItems] = useState<RCSACycleSummary[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [complete, setComplete] = useState(true);
  const [status, setStatus] = useState<StatusFilter>("ALL");
  const [cursorStack, setCursorStack] = useState<string[]>([]);
  const [retry, setRetry] = useState(0);
  const cursor = cursorStack[cursorStack.length - 1];
  const [detail, setDetail] = useState<RCSACycleDetail>();
  const [detailState, setDetailState] = useState<LoadState>("live");

  useEffect(() => {
    setCursorStack([]);
  }, [status]);

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    void listRCSACycles({ status: status === "ALL" ? undefined : status, cursor, limit: 25 }, controller.signal)
      .then((page) => {
        if (controller.signal.aborted) return;
        setItems(page.items);
        setNextCursor(page.next_cursor);
        setComplete(page.complete);
        setState("live");
      })
      .catch(() => {
        if (controller.signal.aborted) return;
        setItems([]);
        setNextCursor(undefined);
        setComplete(false);
        setState("error");
      });
    return () => controller.abort();
  }, [cursor, retry, status]);

  useEffect(() => {
    if (!targetID) {
      setDetail(undefined);
      setDetailState("live");
      return;
    }
    const controller = new AbortController();
    setDetail(undefined);
    setDetailState("loading");
    void getRCSACycle(targetID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setDetail(value.cycle.id === targetID ? value : undefined);
      setDetailState(value.cycle.id === targetID ? "live" : "error");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setDetail(undefined);
      setDetailState("error");
    });
    return () => controller.abort();
  }, [targetID]);

  const cycleStages = useMemo(() => state === "live" ? rcsaCycleStageDistribution(items) : [], [items, state]);
  const challengeCount = cycleStages.find((segment) => segment.id === "AWAITING_CHALLENGE")?.count ?? 0;
  const firstLineCount = cycleStages.find((segment) => segment.id === "ASSESSMENT_OPEN")?.count ?? 0;
  const selectedDetail = detail?.cycle.id === targetID ? detail : undefined;

  return <section className="rcsa-workspace" data-embedded={embedded || undefined}>
    <header className="rcsa-register__heading">
      <div>
        {!embedded && <span className="eyebrow">{organizationName}</span>}
        {embedded ? <h2>Risk &amp; control self-assessments</h2> : <h1>RCSA cycles</h1>}
        <p>Assessment cycles for {legalEntityName || "the current legal entity"}.</p>
      </div>
      <div className="rcsa-register__filter">
        <SelectField
          label="Status"
          value={status}
          placeholder="All statuses"
          options={statusOptions}
          allowsEmpty={false}
          onChange={(value) => { setStatus(value ?? "ALL"); setCursorStack([]); }}
        />
      </div>
    </header>

    {!complete && state === "live" && <Notice tone="warning">Some cycles unavailable. Retry to confirm the full register. <Button variant="quiet" size="compact" onPress={() => setRetry((value) => value + 1)}>Retry</Button></Notice>}
    {state === "loading" && <p className="rcsa-register__count" role="status">Loading RCSA cycles…</p>}
    {state === "live" && items.length > 0 && <>
      <p className="rcsa-register__count">{items.length} {items.length === 1 ? "cycle" : "cycles"} shown{nextCursor ? " · More available" : ""}</p>
      <ul className="rcsa-cycle-grid" aria-label="RCSA cycles">
        {items.map((item) => <li key={item.cycle.id}>
          <Card>
            <div className="rcsa-cycle__body">
              <header className="rcsa-cycle__heading">
                <h3>{item.cycle.name}</h3>
                <StatusBadge tone={rcsaHandoffTone(item.handoff)}>{rcsaStatusLabel(item.cycle.status)}</StatusBadge>
              </header>
              <p className="rcsa-cycle__reference">{item.cycle.code} · {rcsaTriggerLabel(item.cycle.trigger_kind)}</p>
              <dl className="rcsa-cycle__population" aria-label={`${item.cycle.name} frozen population`}>
                <div><dt>Risks</dt><dd>{item.risk_count}</dd></div>
                <div><dt>Controls</dt><dd>{item.control_count}</dd></div>
              </dl>
              <p className="rcsa-cycle__handoff">{item.handoff.label}</p>
              <dl className="rcsa-cycle__facts">
                <div><dt>First-line owner</dt><dd>{rcsaOwnerLabel(item.cycle, item.first_line_owner_display_name)}</dd></div>
                <div><dt>Updated</dt><dd>{formatRCSADate(item.cycle.updated_at)}</dd></div>
              </dl>
              <Button variant="quiet" onPress={() => onTarget(item.cycle.id)} aria-label={`Review cycle for ${item.cycle.name}`}>Review cycle</Button>
            </div>
          </Card>
        </li>)}
      </ul>
    </>}
    {state === "error" && <EmptyState population="RCSA cycles" title="RCSA cycles could not be loaded" description="The cycle register is unavailable." action={<Button variant="secondary" onPress={() => setRetry((value) => value + 1)}>Try again</Button>} role="alert"/>}
    {state === "live" && complete && items.length === 0 && <EmptyState population="RCSA cycles" title="No cycles in this view" description={status === "ALL" ? "No RCSA cycles are available to you in this legal entity." : "No RCSA cycle matches this status."}/>}
    {(cursorStack.length > 0 || nextCursor) && <nav className="rcsa-register__pagination" aria-label="RCSA cycle pages">
      {cursorStack.length > 0 && <Button variant="secondary" isDisabled={state === "loading"} onPress={() => setCursorStack((current) => current.slice(0, -1))}>Previous page</Button>}
      {nextCursor && <Button variant="secondary" isLoading={state === "loading"} onPress={() => setCursorStack((current) => [...current, nextCursor])}>Next page</Button>}
    </nav>}

    {state === "live" && items.length > 0 && <details className="rcsa-register__status-disclosure">
      <summary>Cycle status</summary>
      <section className="rcsa-register__position" aria-label="Status of displayed RCSA cycles">
        <div className="rcsa-register__position-heading">
          <h3>Displayed cycles</h3>
          <small>{items.length} shown{nextCursor ? " · More available" : ""}</small>
        </div>
        <StackedDistribution ariaLabel="RCSA cycle status for displayed records" segments={cycleStages}/>
        {(challengeCount > 0 || firstLineCount > 0) && <div className="rcsa-register__position-actions">
          {challengeCount > 0 && status !== "AWAITING_CHALLENGE" && <Button variant="secondary" size="compact" onPress={() => { setStatus("AWAITING_CHALLENGE"); setCursorStack([]); }}>Review {challengeCount} awaiting challenge</Button>}
          {firstLineCount > 0 && status !== "ASSESSMENT_OPEN" && <Button variant="quiet" size="compact" onPress={() => { setStatus("ASSESSMENT_OPEN"); setCursorStack([]); }}>Review {firstLineCount} first-line</Button>}
        </div>}
        <p>Counts apply to displayed cycles only.</p>
      </section>
    </details>}

    {targetID && detailState === "loading" && <span className="cs-sr-only" role="status">Loading RCSA cycle…</span>}
    {targetID && detailState === "error" && <Notice tone="error">This RCSA cycle could not be loaded. <Button variant="quiet" size="compact" onPress={() => onTarget(undefined)}>Close selection</Button></Notice>}
    {selectedDetail && <FocusedSheet label={`${selectedDetail.cycle.name} RCSA cycle`} panelClassName="rcsa-inspector" onClose={() => onTarget(undefined)}>
      <RCSACycleDetailView detail={selectedDetail} onOpenEvidence={onOpenEvidence} onOpenMatter={onOpenMatter}/>
    </FocusedSheet>}
  </section>;
}

function RCSACycleDetailView({ detail, onOpenEvidence, onOpenMatter }: { detail: RCSACycleDetail; onOpenEvidence?: (id: string, cycleID: string) => void; onOpenMatter?: (id: string, cycleID: string) => void }) {
  const riskColumns: readonly DataColumn<RCSARiskSnapshot>[] = [
    { id: "risk", header: "Risk", mobileLayout: "full-width", render: (risk) => <span className="rcsa-stack"><strong>{risk.name}</strong><small>{risk.code}</small></span>, accessibleText: (risk) => `${risk.name}. ${risk.code}` },
    { id: "category", header: "Category", render: (risk) => risk.category || "Not classified", accessibleText: (risk) => risk.category || "Not classified" },
    { id: "version", header: "Frozen version", kind: "number", render: (risk) => risk.risk_version, accessibleText: (risk) => String(risk.risk_version) },
  ];
  const controlColumns: readonly DataColumn<RCSAControlSnapshot>[] = [
    { id: "control", header: "Control", mobileLayout: "full-width", render: (control) => <span className="rcsa-stack"><strong>{control.definition_name}</strong><small>{control.definition_code}</small></span>, accessibleText: (control) => `${control.definition_name}. ${control.definition_code}` },
    { id: "implementation", header: "Implementation", mobileLayout: "full-width", render: (control) => <span className="rcsa-stack"><strong>{control.implementation_name}</strong><small>Version {control.implementation_version}</small></span>, accessibleText: (control) => `${control.implementation_name}. Version ${control.implementation_version}` },
    { id: "status", header: "Status", kind: "status", render: (control) => <StatusBadge tone={control.implementation_status === "IMPLEMENTED" ? "success" : "warning"}>{control.implementation_status.replaceAll("_", " ").toLowerCase()}</StatusBadge>, accessibleText: (control) => control.implementation_status },
  ];

  const openHandoff = () => {
    if (!detail.handoff.target_id) return;
    if (detail.handoff.target_type === "EVIDENCE_REQUEST") onOpenEvidence?.(detail.handoff.target_id, detail.cycle.id);
    if (detail.handoff.target_type === "MATTER") onOpenMatter?.(detail.handoff.target_id, detail.cycle.id);
  };
  const canOpenHandoff = detail.handoff.target_id && (detail.handoff.target_type === "EVIDENCE_REQUEST" ? onOpenEvidence : detail.handoff.target_type === "MATTER" ? onOpenMatter : undefined);

  return <article className="rcsa-detail">
    <header className="rcsa-detail__header">
      <div>
        <span className="eyebrow">{detail.cycle.code} · {rcsaTriggerLabel(detail.cycle.trigger_kind)}</span>
        <h2>{detail.cycle.name}</h2>
      </div>
      <StatusBadge tone={rcsaStatusTone(detail.cycle.status)}>{rcsaStatusLabel(detail.cycle.status)}</StatusBadge>
    </header>

    {!detail.complete && <Notice tone="warning">Some cycle details unavailable.</Notice>}

    <section className="rcsa-detail__facts" aria-label="Cycle facts">
      <div><span>Assessment period</span><strong>{formatRCSAPeriod(detail.assessment_period_start, detail.assessment_period_end)}</strong></div>
      <div><span>First-line owner</span><strong>{rcsaOwnerLabel(detail.cycle, detail.first_line_owner_display_name)}</strong></div>
      <div><span>Frozen population</span><strong>{detail.risks.length} Risks · {detail.controls.length} controls</strong></div>
      <div><span>Updated</span><strong>{formatRCSADate(detail.cycle.updated_at)}</strong></div>
    </section>

    <Surface>
      <div className="rcsa-detail__handoff">
        <div>
          <span className="eyebrow">Current handoff</span>
          <h3>{detail.handoff.label}</h3>
        </div>
        {canOpenHandoff && <Button variant="secondary" onPress={openHandoff}>{detail.handoff.target_type === "MATTER" ? "Open challenge work" : "Open first-line assessment"}</Button>}
      </div>
    </Surface>

    <Surface>
      <section className="rcsa-detail__phase" aria-label="RCSA cycle path">
        <div className="rcsa-detail__phase-heading">
          <div>
            <span className="eyebrow">Current phase</span>
            <h3>{detail.phase.label}</h3>
            <p>{detail.phase.detail}</p>
          </div>
          <StatusBadge tone={rcsaPhaseTone(detail.phase.stage)}>{detail.phase.label}</StatusBadge>
        </div>
        <ol className="rcsa-phase-path">
          {rcsaPhasePath(detail.phase.stage).map((step) => <li className={`is-${step.state}`} key={step.id}>
            <span aria-hidden="true"/>
            <div><strong>{step.label}</strong><small>{phaseStepStateLabel(step.state)}</small></div>
          </li>)}
        </ol>
      </section>
    </Surface>

    <section className="rcsa-detail__population" aria-labelledby="rcsa-risks">
      <div className="section-header"><div><h3 id="rcsa-risks">Risks</h3><p>Frozen at cycle creation.</p></div></div>
      {detail.risks.length ? <DataTable ariaLabel="RCSA frozen Risks" rows={detail.risks} rowKey={(risk) => risk.risk_id} rowName={(risk) => risk.name} columns={riskColumns} responsiveTo="container"/> : <EmptyState population="Risks" title="No frozen Risks" description="This cycle has no Risk population."/>}
    </section>

    <section className="rcsa-detail__population" aria-labelledby="rcsa-controls">
      <div className="section-header"><div><h3 id="rcsa-controls">Controls</h3><p>Control versions tied to the frozen Risk population.</p></div></div>
      {detail.controls.length ? <DataTable ariaLabel="RCSA frozen Controls" rows={detail.controls} rowKey={(control) => control.risk_control_link_id} rowName={(control) => control.definition_name} columns={controlColumns} responsiveTo="container"/> : <EmptyState population="Controls" title="No frozen Controls" description="No catalogued controls were linked to the frozen Risk population."/>}
    </section>

    <details className="rcsa-detail__lineage">
      <summary>Record details</summary>
      <dl>
        <div><dt>Cycle version</dt><dd>{detail.cycle.version}</dd></div>
        <div><dt>First-line owner reference</dt><dd><code>{detail.cycle.first_line_owner_principal_id}</code></dd></div>
        <div><dt>Population checksum</dt><dd><code>{detail.cycle.population_checksum}</code></dd></div>
        {detail.cycle.first_line_response_revision_id && <div><dt>First-line response revision</dt><dd><code>{detail.cycle.first_line_response_revision_id}</code></dd></div>}
      </dl>
    </details>
  </article>;
}

function phaseStepStateLabel(state: "complete" | "current" | "pending" | "not_required" | "unknown") {
  if (state === "complete") return "Complete";
  if (state === "current") return "Current";
  if (state === "not_required") return "Not required";
  if (state === "unknown") return "Outcome not classified";
  return "Pending decision";
}
