import { useEffect, useMemo, useState } from "react";
import { getRCSACycle, listRCSACycles } from "../../rcsaApi";
import { rcsaCycleStageDistribution } from "../../rcsaCycleDistribution";
import type { RCSAControlSnapshot, RCSACycleDetail, RCSACycleSummary, RCSARiskSnapshot, RCSAStatus } from "../../rcsaTypes";
import { formatRCSADate, formatRCSAPeriod, rcsaHandoffTone, rcsaOwnerLabel, rcsaPhasePath, rcsaPhaseTone, rcsaStatusLabel, rcsaStatusTone, rcsaTriggerLabel } from "../../rcsaPresentation";
import { Button, DataTable, EmptyState, FocusedSheet, Notice, SelectField, StackedDistribution, StatusBadge, Surface, type DataColumn } from "../ui";
import "./rcsa.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
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

export function RCSAWorkspace({ organizationName, legalEntityName, targetID, onTarget, onOpenEvidence, onOpenMatter }: Props) {
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
    setDetailState("loading");
    void getRCSACycle(targetID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setDetail(value);
      setDetailState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setDetail(undefined);
      setDetailState("error");
    });
    return () => controller.abort();
  }, [targetID]);

  const columns = useMemo<readonly DataColumn<RCSACycleSummary>[]>(() => [
    {
      id: "cycle",
      header: "Cycle",
      mobileLayout: "full-width",
      render: (item) => <span className="rcsa-stack"><strong>{item.cycle.name}</strong><small>{item.cycle.code} · {rcsaTriggerLabel(item.cycle.trigger_kind)}</small></span>,
      accessibleText: (item) => `${item.cycle.name}. ${item.cycle.code}. ${rcsaTriggerLabel(item.cycle.trigger_kind)}`,
    },
    {
      id: "stage",
      header: "Stage",
      kind: "status",
      render: (item) => <span className="rcsa-stack"><StatusBadge tone={rcsaHandoffTone(item.handoff)}>{rcsaStatusLabel(item.cycle.status)}</StatusBadge><small>{item.handoff.label}</small></span>,
      accessibleText: (item) => `${rcsaStatusLabel(item.cycle.status)}. ${item.handoff.label}`,
    },
    {
      id: "population",
      header: "Frozen population",
      kind: "number",
      render: (item) => <span className="rcsa-stack"><strong>{item.risk_count} Risks</strong><small>{item.control_count} controls</small></span>,
      accessibleText: (item) => `${item.risk_count} Risks. ${item.control_count} controls`,
    },
    {
      id: "owner",
      header: "First-line owner",
      render: (item) => rcsaOwnerLabel(item.cycle, item.first_line_owner_display_name),
      accessibleText: (item) => rcsaOwnerLabel(item.cycle, item.first_line_owner_display_name),
    },
    {
      id: "updated",
      header: "Updated",
      render: (item) => formatRCSADate(item.cycle.updated_at),
      accessibleText: (item) => formatRCSADate(item.cycle.updated_at),
    },
  ], []);

  return <section className="rcsa-workspace">
    <header className="topbar">
      <div>
        <span className="eyebrow">{organizationName}</span>
        <h1>RCSA cycles</h1>
        <p>First-line assessment and independent challenge for {legalEntityName || "the current legal entity"}.</p>
      </div>
    </header>

    <Surface>
      <div className="rcsa-register__heading">
        <div>
          <h2>Cycles</h2>
          <p>Each cycle keeps the Risk and Control population frozen at creation.</p>
        </div>
        <div className="rcsa-register__filter">
          <SelectField
            label="Status"
            value={status}
            placeholder="All statuses"
            options={statusOptions}
            allowsEmpty={false}
            onChange={(value) => setStatus(value ?? "ALL")}
          />
        </div>
      </div>

      {!complete && state !== "error" && <Notice tone="warning">Some cycles are temporarily unavailable. Available cycles remain unchanged.</Notice>}
      {state === "live" && items.length > 0 && <section className="rcsa-register__position" aria-label="Loaded RCSA cycle stages">
        <div className="rcsa-register__position-heading">
          <div><span className="eyebrow">Assessment cycles</span><h3>Current stages</h3></div>
          <small>{items.length} loaded{nextCursor || cursorStack.length > 0 ? " · Paginated view" : ""}</small>
        </div>
        <StackedDistribution ariaLabel="RCSA stages in loaded cycles" segments={rcsaCycleStageDistribution(items)}/>
        <p>Cycle status, not Risk assessment completion. The frozen Risk and Control populations remain separate.</p>
      </section>}

      {state === "error" && <EmptyState population="RCSA cycles" title="RCSA cycles could not be loaded" description="The cycle register is unavailable." action={<Button variant="secondary" onPress={() => setRetry((value) => value + 1)}>Try again</Button>} role="alert"/>}
      {state === "live" && items.length === 0 && <EmptyState population="RCSA cycles" title="No cycles in this view" description={status === "ALL" ? "No RCSA cycles are available to you in this legal entity." : "No RCSA cycle matches this status."}/>}
      {(state === "loading" || items.length > 0) && <DataTable
        ariaLabel="RCSA cycles"
        rows={items}
        rowKey={(item) => item.cycle.id}
        rowName={(item) => item.cycle.name}
        columns={columns}
        onRowAction={(item) => onTarget(item.cycle.id)}
        rowActionLabel="Review cycle"
        isLoading={state === "loading"}
        pagination={(cursorStack.length > 0 || nextCursor) ? {
          label: "RCSA cycle pages",
          onPrevious: cursorStack.length > 0 ? () => setCursorStack((current) => current.slice(0, -1)) : undefined,
          onNext: nextCursor ? () => setCursorStack((current) => [...current, nextCursor]) : undefined,
          isLoading: state === "loading",
        } : undefined}
      />}
    </Surface>

    {targetID && detailState === "loading" && <span className="cs-sr-only" role="status">Loading RCSA cycle…</span>}
    {targetID && detailState === "error" && <Notice tone="error">This RCSA cycle could not be loaded. <Button variant="quiet" size="compact" onPress={() => onTarget(undefined)}>Close selection</Button></Notice>}
    {detail && <FocusedSheet label={`${detail.cycle.name} RCSA cycle`} size="wide" onClose={() => onTarget(undefined)}>
      <RCSACycleDetailView detail={detail} onOpenEvidence={onOpenEvidence} onOpenMatter={onOpenMatter}/>
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

  return <article className="rcsa-detail">
    <header className="rcsa-detail__header">
      <div>
        <span className="eyebrow">{detail.cycle.code} · {rcsaTriggerLabel(detail.cycle.trigger_kind)}</span>
        <h2>{detail.cycle.name}</h2>
      </div>
      <StatusBadge tone={rcsaStatusTone(detail.cycle.status)}>{rcsaStatusLabel(detail.cycle.status)}</StatusBadge>
    </header>

    {!detail.complete && <Notice tone="warning">Some cycle context is unavailable. The frozen cycle record remains unchanged.</Notice>}

    <section className="rcsa-detail__facts" aria-label="Cycle facts">
      <div><span>Assessment period</span><strong>{formatRCSAPeriod(detail.assessment_period_start, detail.assessment_period_end)}</strong></div>
      <div><span>First-line owner</span><strong>{rcsaOwnerLabel(detail.cycle, detail.first_line_owner_display_name)}</strong></div>
      <div><span>Frozen population</span><strong>{detail.risks.length} Risks · {detail.controls.length} controls</strong></div>
      <div><span>Updated</span><strong>{formatRCSADate(detail.cycle.updated_at)}</strong></div>
    </section>

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

    <Surface>
      <div className="rcsa-detail__handoff">
        <div>
          <span className="eyebrow">Current handoff</span>
          <h3>{detail.handoff.label}</h3>
        </div>
        {detail.handoff.target_id && <Button variant="secondary" onPress={openHandoff}>{detail.handoff.target_type === "MATTER" ? "Open challenge work" : "Open first-line assessment"}</Button>}
      </div>
    </Surface>

    <section className="rcsa-detail__population" aria-labelledby="rcsa-risks">
      <div className="section-header"><div><h3 id="rcsa-risks">Risks</h3><p>Frozen at cycle creation.</p></div></div>
      {detail.risks.length ? <DataTable ariaLabel="RCSA frozen Risks" rows={detail.risks} rowKey={(risk) => risk.risk_id} rowName={(risk) => risk.name} columns={riskColumns}/> : <EmptyState population="Risks" title="No frozen Risks" description="This cycle has no Risk population."/>}
    </section>

    <section className="rcsa-detail__population" aria-labelledby="rcsa-controls">
      <div className="section-header"><div><h3 id="rcsa-controls">Controls</h3><p>Control versions tied to the frozen Risk population.</p></div></div>
      {detail.controls.length ? <DataTable ariaLabel="RCSA frozen Controls" rows={detail.controls} rowKey={(control) => control.risk_control_link_id} rowName={(control) => control.definition_name} columns={controlColumns}/> : <EmptyState population="Controls" title="No frozen Controls" description="No catalogued controls were linked to the frozen Risk population."/>}
    </section>

    <details className="rcsa-detail__lineage">
      <summary>Record details</summary>
      <dl>
        <div><dt>Cycle version</dt><dd>{detail.cycle.version}</dd></div>
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
