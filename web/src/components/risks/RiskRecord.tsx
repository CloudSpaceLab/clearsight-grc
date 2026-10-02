import { useEffect, useState } from "react";
import { getRisk } from "../../riskApi";
import type { RiskAggregate, RiskAppetiteStatement, RiskAssessment, RiskControlDetail } from "../../riskTypes";
import { apiErrorKind } from "../../http";
import { Button, DataTable, EmptyState, Notice, StatusBadge, Surface, type DataColumn } from "../ui";
import { appetiteLabel, appetiteTone, assessmentKindLabel, controlEvidenceSummary, controlImplementationLabel, controlImplementationTone, currentAppetiteLabel, currentAppetiteTone, dimensionSummary, formatRiskDate, riskStatusLabel, riskStatusTone, scopeEntries } from "./riskPresentation";

type Props = {
  riskID: string;
  onBack: () => void;
  onOpenProgramControl?: (programID: string, objectiveID: string) => void;
  loadRisk?: (id: string, signal?: AbortSignal) => Promise<RiskAggregate>;
};

type LoadState = "loading" | "live" | "not-found" | "error";

export function RiskRecord({ riskID, onBack, onOpenProgramControl, loadRisk = getRisk }: Props) {
  const [value, setValue] = useState<RiskAggregate>();
  const [state, setState] = useState<LoadState>("loading");
  const [retry, setRetry] = useState(0);

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

  const risk = value.risk;
  const latestAssessment = value.assessments[0];
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
      render: (item) => <StatusBadge tone={controlImplementationTone(item.implementation_status)}>{controlImplementationLabel(item.implementation_status)}</StatusBadge>,
      accessibleText: (item) => controlImplementationLabel(item.implementation_status),
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
        <div><dt>Appetite</dt><dd><StatusBadge tone={currentAppetiteTone(risk.version, latestAssessment, value.active_appetite)}>{currentAppetiteLabel(risk.version, latestAssessment, value.active_appetite)}</StatusBadge></dd></div>
        <div><dt>Status</dt><dd><StatusBadge tone={riskStatusTone(risk.status)}>{riskStatusLabel(risk.status)}</StatusBadge></dd></div>
        <div><dt>Owner</dt><dd>{risk.owner_principal_id ? "Assigned" : "Not assigned"}</dd></div>
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
      <div className="section-header"><div><h2 id="risk-controls-heading">Controls</h2><p>Existing Program controls linked to this risk. Evidence remains governed in the Program.</p></div></div>
      {value.control_details_complete === false && <Notice tone="warning">Some linked control details are unavailable in the current scope.</Notice>}
      {value.control_details?.length ? <DataTable
        ariaLabel="Risk controls"
        rows={value.control_details}
        rowKey={(item) => item.link.id}
        rowName={(item) => `${item.definition.name}, ${item.implementation_name}, ${controlImplementationLabel(item.implementation_status)}`}
        columns={controlColumns}
        onRowAction={onOpenProgramControl ? (item) => onOpenProgramControl(item.program_id, item.objective_id) : undefined}
        rowActionLabel="Open control"
      /> : <EmptyState population={risk.name} title="No linked controls" description="No existing Program control is linked to this risk."/>}
    </section>

    <section className="risk-record__history" aria-labelledby="risk-assessments-heading">
      <div className="section-header"><div><h2 id="risk-assessments-heading">Assessments</h2><p>Approved assessment records and their appetite position.</p></div></div>
      {value.assessments.length ? <DataTable
        ariaLabel="Risk assessments"
        rows={value.assessments}
        rowKey={(item) => item.id}
        rowName={(item) => `${assessmentKindLabel(item.kind)}, ${item.method_code}, ${appetiteLabel(item.appetite_position)}`}
        columns={assessmentColumns}
      /> : <EmptyState population={risk.name} title="No assessments" description="No governed assessment has been recorded for this risk."/>}
    </section>

    <section className="risk-record__history" aria-labelledby="risk-appetite-heading">
      <div className="section-header"><div><h2 id="risk-appetite-heading">Appetite history</h2><p>Versioned appetite statements retained for this risk.</p></div></div>
      {value.appetite.length ? <DataTable
        ariaLabel="Risk appetite history"
        rows={value.appetite}
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
