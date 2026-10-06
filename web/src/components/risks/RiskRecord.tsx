import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { listControlCatalogCandidates, type ControlCatalogCandidatePage } from "../../controlCatalogApi";
import { getRisk, linkRiskControl, type LinkRiskControlResponse } from "../../riskApi";
import type { RiskAggregate, RiskAppetiteStatement, RiskAssessment, RiskControlDetail } from "../../riskTypes";
import { apiErrorKind } from "../../http";
import { Button, DataTable, EmptyState, Notice, SelectField, StatusBadge, Surface, type DataColumn } from "../ui";
import { appetiteLabel, appetiteTone, assessmentKindLabel, controlEvidenceSummary, controlImplementationStatusLabel, controlImplementationStatusTone, currentAppetiteLabel, currentAppetiteTone, dimensionSummary, formatRiskDate, riskCurrentPositionAssessment, riskStatusLabel, riskStatusTone, scopeEntries } from "./riskPresentation";
import { RiskIndicatorsSection } from "./RiskIndicatorsSection";
import { NotificationDeliveryHistory } from "../NotificationDeliveryHistory";

type Props = {
  riskID: string;
  actorID?: string;
  onBack: () => void;
  onOpenProgram?: (programID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  onOpenProgramControl?: (programID: string, objectiveID: string) => void;
  loadRisk?: (id: string, signal?: AbortSignal) => Promise<RiskAggregate>;
  loadControlCandidates?: (programID?: string) => Promise<ControlCatalogCandidatePage>;
  linkControl?: (riskID: string, expectedRiskVersion: number, catalogLinkID: string) => Promise<LinkRiskControlResponse>;
};

type LoadState = "loading" | "live" | "not-found" | "error";

export function RiskRecord({ riskID, actorID, onBack, onOpenProgram, onOpenMatter, onOpenProgramControl, loadRisk = getRisk, loadControlCandidates = listControlCatalogCandidates, linkControl = linkRiskControl }: Props) {
  const [value, setValue] = useState<RiskAggregate>();
  const [state, setState] = useState<LoadState>("loading");
  const [retry, setRetry] = useState(0);
  const [linkMode, setLinkMode] = useState(false);
  const [candidateState, setCandidateState] = useState<"idle" | "loading" | "live" | "error">("idle");
  const [candidatePage, setCandidatePage] = useState<ControlCatalogCandidatePage>({ items: [], complete: true });
  const [selectedCatalogLinkID, setSelectedCatalogLinkID] = useState<string>();
  const [linkBusy, setLinkBusy] = useState(false);
  const [linkError, setLinkError] = useState("");

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setValue(undefined);
    void loadRisk(riskID, controller.signal).then((loaded) => {
      if (controller.signal.aborted) return;
      setValue(loaded);
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setState(apiErrorKind(error) === "not_found" ? "not-found" : "error");
    });
    return () => controller.abort();
  }, [loadRisk, retry, riskID]);

  if (state === "loading" && !value) return <section className="risk-record-page"><p className="risk-load-state" role="status">Loading risk…</p></section>;
  if (state === "not-found") return <section className="risk-record-page"><EmptyState population="Current legal-entity risk register" title="Risk not found" description="Return to the risk register." action={<Button variant="secondary" onPress={onBack}>Back to risks</Button>}/></section>;
  if (state === "error" || !value) return <section className="risk-record-page"><EmptyState population="Selected risk record" title="Risk unavailable" description="The risk could not be loaded." action={<Button variant="secondary" onPress={() => setRetry((current) => current + 1)}>Try again</Button>} role="alert"/></section>;

  const aggregate = value;
  const risk = aggregate.risk;
  const currentAssessment = riskCurrentPositionAssessment(aggregate.assessments);
  const scope = scopeEntries(risk.scope ?? {});

  const assessmentColumns: readonly DataColumn<RiskAssessment>[] = [
    { id: "kind", header: "Assessment", render: (item) => assessmentKindLabel(item.kind), accessibleText: (item) => assessmentKindLabel(item.kind) },
    { id: "method", header: "Method", render: (item) => <span className="risk-record__stack"><strong>{item.method_code}</strong><small>{item.method_version}</small></span>, accessibleText: (item) => `${item.method_code}, ${item.method_version}` },
    { id: "dimensions", header: "Dimensions", render: (item) => dimensionSummary(item.dimensions), accessibleText: (item) => dimensionSummary(item.dimensions) },
    { id: "appetite", header: "Appetite", kind: "status", render: (item) => <StatusBadge tone={appetiteTone(item.appetite_position)}>{appetiteLabel(item.appetite_position)}</StatusBadge>, accessibleText: (item) => appetiteLabel(item.appetite_position) },
    { id: "assessed", header: "Assessed", render: (item) => formatRiskDate(item.assessed_at), accessibleText: (item) => formatRiskDate(item.assessed_at) },
  ];

  const appetiteColumns: readonly DataColumn<RiskAppetiteStatement>[] = [
    { id: "version", header: "Version", kind: "number", render: (item) => item.version, accessibleText: (item) => String(item.version) },
    { id: "statement", header: "Statement", mobileLayout: "full-width", render: (item) => item.statement, accessibleText: (item) => item.statement },
    { id: "effective", header: "Effective", render: (item) => formatRiskDate(item.effective_from), accessibleText: (item) => formatRiskDate(item.effective_from) },
    { id: "until", header: "Until", render: (item) => item.effective_until ? formatRiskDate(item.effective_until) : "No end date", accessibleText: (item) => item.effective_until ? formatRiskDate(item.effective_until) : "No end date" },
    { id: "status", header: "Status", kind: "status", render: (item) => <StatusBadge tone={item.status === "ACTIVE" ? "info" : "unknown"}>{item.status === "ACTIVE" ? "Active" : "Retired"}</StatusBadge>, accessibleText: (item) => item.status === "ACTIVE" ? "Active" : "Retired" },
  ];

  const controlColumns: readonly DataColumn<RiskControlDetail>[] = [
    {
      id: "control",
      header: "Control",
      mobileLayout: "full-width",
      render: (item) => <span className="risk-record__stack"><strong>{item.definition.name}</strong><small>{[item.definition.code, item.definition.category].filter(Boolean).join(" · ")}</small></span>,
      accessibleText: (item) => [item.definition.name, item.definition.code, item.definition.category].filter(Boolean).join(", "),
    },
    {
      id: "implementation",
      header: "Implementation",
      render: (item) => <span className="risk-record__stack"><strong>{item.implementation_name}</strong><small>{item.program_name}</small></span>,
      accessibleText: (item) => `${item.implementation_name}, ${item.program_name}`,
    },
    {
      id: "state",
      header: "State",
      kind: "status",
      render: (item) => <StatusBadge tone={controlImplementationStatusTone(item.implementation_status)}>{controlImplementationStatusLabel(item.implementation_status)}</StatusBadge>,
      accessibleText: (item) => controlImplementationStatusLabel(item.implementation_status),
    },
    {
      id: "owner",
      header: "Owner",
      render: (item) => item.owner_display_name || (item.owner_assigned ? "Assigned" : "Not assigned"),
      accessibleText: (item) => item.owner_display_name || (item.owner_assigned ? "Assigned" : "Not assigned"),
    },
    {
      id: "evidence",
      header: "Evidence",
      mobileLayout: "full-width",
      render: (item) => controlEvidenceSummary(item),
      accessibleText: (item) => controlEvidenceSummary(item),
    },
  ];

  const linkedCatalogIDs = new Set((aggregate.controls ?? []).map((item) => item.catalog_link_id));
  const availableCandidates = candidatePage.items.filter((item) => !linkedCatalogIDs.has(item.catalog_link_id));
  const candidateOptions = availableCandidates.map((item) => ({
    id: item.catalog_link_id,
    label: `${item.definition.name} · ${item.implementation_name} · ${item.program_name}`,
  }));
  const canLinkControl = Boolean(actorID && risk.owner_principal_id && actorID === risk.owner_principal_id);

  async function beginLinkControl() {
    setLinkMode(true);
    setCandidateState("loading");
    setLinkError("");
    setSelectedCatalogLinkID(undefined);
    try {
      const page = await loadControlCandidates();
      setCandidatePage(page);
      const existing = new Set((aggregate.controls ?? []).map((item) => item.catalog_link_id));
      const first = page.items.find((item) => !existing.has(item.catalog_link_id));
      setSelectedCatalogLinkID(first?.catalog_link_id);
      setCandidateState("live");
    } catch (error) {
      setCandidateState("error");
      setLinkError(error instanceof Error ? error.message : "Reusable controls could not be loaded.");
    }
  }

  async function saveControlLink(event: FormEvent) {
    event.preventDefault();
    if (!selectedCatalogLinkID || linkBusy) return;
    setLinkBusy(true);
    setLinkError("");
    try {
      await linkControl(risk.id, risk.version, selectedCatalogLinkID);
      const loaded = await loadRisk(riskID);
      setValue(loaded);
      setLinkMode(false);
      setCandidateState("idle");
      setSelectedCatalogLinkID(undefined);
    } catch (error) {
      setLinkError(error instanceof Error ? error.message : "The control could not be linked.");
    } finally {
      setLinkBusy(false);
    }
  }

  return <section className="risk-record-page" aria-labelledby="risk-record-heading">
    <header className="topbar risk-page-header">
      <div>
        <span className="eyebrow">{risk.code}{risk.category ? ` · ${risk.category}` : ""}</span>
        <h1 id="risk-record-heading">{risk.name}</h1>
      </div>
      <Button variant="secondary" onPress={onBack}>Back to risks</Button>
    </header>

    <Surface>
      <dl className="risk-record__state" role="group" aria-label="Current risk state">
        <div><dt>{currentAssessment ? `${assessmentKindLabel(currentAssessment.kind)} appetite` : "Appetite"}</dt><dd><StatusBadge tone={currentAppetiteTone(risk.version, currentAssessment, aggregate.active_appetite)}>{currentAppetiteLabel(risk.version, currentAssessment, aggregate.active_appetite)}</StatusBadge></dd></div>
        <div><dt>Status</dt><dd><StatusBadge tone={riskStatusTone(risk.status)}>{riskStatusLabel(risk.status)}</StatusBadge></dd></div>
        <div><dt>Owner</dt><dd>{aggregate.owner_display_name || (risk.owner_principal_id ? "Assigned · name unavailable" : "Not assigned")}</dd></div>
        <div><dt>Updated</dt><dd>{formatRiskDate(risk.updated_at)}</dd></div>
      </dl>
    </Surface>

    <div className="risk-record__overview">
      <Surface>
        <section className="risk-record__section">
          <h2>Risk statement</h2>
          <p>{risk.statement}</p>
          <h3>Potential impact</h3>
          <p>{risk.impact}</p>
          {(risk.cause || risk.event) && <dl className="risk-record__facts">
            {risk.cause && <div><dt>Cause</dt><dd>{risk.cause}</dd></div>}
            {risk.event && <div><dt>Event</dt><dd>{risk.event}</dd></div>}
          </dl>}
        </section>
      </Surface>
      <Surface>
        <section className="risk-record__section">
          <h2>Scope</h2>
          {scope.length ? <dl className="risk-record__facts">{scope.map((entry) => <div key={entry.label}><dt>{entry.label}</dt><dd>{entry.value}</dd></div>)}</dl> : <p>No additional service or objective scope recorded.</p>}
        </section>
      </Surface>
    </div>

    <section className="risk-record__history" aria-labelledby="risk-controls-heading">
      <div className="section-header">
        <div><h2 id="risk-controls-heading">Controls</h2><p>Existing Program controls linked to this risk. Evidence remains governed in the Program.</p></div>
        {canLinkControl && !linkMode && <Button variant="secondary" size="compact" onPress={() => void beginLinkControl()}>Link control</Button>}
      </div>
      {linkMode && <div className="risk-record__link-form">
        <div><h3>Link existing control</h3><p>Choose a reusable control already catalogued in this legal entity.</p></div>
        {candidateState === "loading" && <p role="status">Loading reusable controls…</p>}
        {candidateState === "error" && <Notice tone="error">{linkError || "Reusable controls could not be loaded."}</Notice>}
        {candidateState === "live" && !candidatePage.complete && <Notice tone="warning">Some reusable controls are unavailable in the current scope.</Notice>}
        {candidateState === "live" && candidateOptions.length === 0 && <p>No unlinked reusable controls are available.</p>}
        {candidateState === "live" && candidateOptions.length > 0 && <form onSubmit={(event) => void saveControlLink(event)}>
          <SelectField label="Reusable control" value={selectedCatalogLinkID} placeholder="Choose a control" options={candidateOptions} onChange={setSelectedCatalogLinkID}/>
          {linkError && <Notice tone="error">{linkError}</Notice>}
          <div className="risk-record__link-actions">
            <Button type="submit" isDisabled={linkBusy || !selectedCatalogLinkID}>{linkBusy ? "Linking…" : "Link control"}</Button>
            <Button variant="secondary" type="button" onPress={() => { setLinkMode(false); setLinkError(""); }}>Cancel</Button>
          </div>
        </form>}
        {candidateState !== "loading" && (candidateState !== "live" || candidateOptions.length === 0) && <Button variant="secondary" size="compact" onPress={() => { setLinkMode(false); setLinkError(""); }}>Close</Button>}
      </div>}
      {aggregate.control_details_complete === false && <Notice tone="warning">Some linked control details are unavailable in the current scope.</Notice>}
      {aggregate.control_details?.length ? <DataTable
        ariaLabel="Risk controls"
        rows={aggregate.control_details}
        rowKey={(item) => item.link.id}
        rowName={(item) => `${item.definition.name}, ${item.implementation_name}, ${controlImplementationStatusLabel(item.implementation_status)}`}
        columns={controlColumns}
        onRowAction={onOpenProgramControl ? (item) => onOpenProgramControl(item.program_id, item.objective_id) : undefined}
        rowActionLabel="Open control"
      /> : <EmptyState population={risk.name} title="No linked controls" description="No existing Program control is linked to this risk."/>}
    </section>

    <RiskIndicatorsSection
      risk={risk}
      actorID={actorID}
      indicators={aggregate.indicators ?? []}
      details={aggregate.indicator_details ?? []}
      detailsComplete={aggregate.indicator_details_complete !== false}
      onReload={async () => {
        const loaded = await loadRisk(riskID);
        setValue(loaded);
      }}
      onOpenProgram={onOpenProgram}
      onOpenMatter={onOpenMatter}
    />

    <NotificationDeliveryHistory
      items={aggregate.notification_history}
      complete={aggregate.notification_history_complete !== false}
      className="risk-record__history"
    />

    <section className="risk-record__history" aria-labelledby="risk-assessments-heading">
      <div className="section-header"><div><h2 id="risk-assessments-heading">Assessments</h2><p>Approved assessment records and their appetite position.</p></div></div>
      {aggregate.assessments.length ? <DataTable
        ariaLabel="Risk assessments"
        rows={aggregate.assessments}
        rowKey={(item) => item.id}
        rowName={(item) => `${assessmentKindLabel(item.kind)}, ${item.method_code}, ${appetiteLabel(item.appetite_position)}`}
        columns={assessmentColumns}
      /> : <EmptyState population={risk.name} title="No assessments" description="No governed assessment has been recorded for this risk."/>}
    </section>

    <section className="risk-record__history" aria-labelledby="risk-appetite-heading">
      <div className="section-header"><div><h2 id="risk-appetite-heading">Appetite history</h2><p>Versioned appetite statements retained for this risk.</p></div></div>
      {aggregate.appetite.length ? <DataTable
        ariaLabel="Risk appetite history"
        rows={aggregate.appetite}
        rowKey={(item) => item.id}
        rowName={(item) => `Appetite version ${item.version}, ${item.statement}`}
        columns={appetiteColumns}
      /> : <EmptyState population={risk.name} title="No appetite statement" description="No appetite statement has been recorded for this risk."/>}
    </section>
  </section>;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
