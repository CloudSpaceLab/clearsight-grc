import { useEffect, useRef, useState } from "react";
import { loadMatterSummaries as loadMatterSummariesRequest } from "../api";
import type { MatterSummary } from "../summaryTypes";
import {
  createReportDefinition as createReportDefinitionRequest,
  listReportDefinitions as listReportDefinitionsRequest,
  transitionReportDefinition as transitionReportDefinitionRequest,
} from "../reportingApi";
import type { ReportDefinition, ReportDefinitionAction, ReportDefinitionStatus, ReportFilterExpression, ReportFormat } from "../reportingTypes";
import {
  buildReportSetupInput, reportSetupArea, reportSetupAreaLabel, reportSetupAreaOptions,
  reportSetupFocus, reportSetupFocusLabel, reportSetupFocusOptions,
  type ReportSetupArea, type ReportSetupFocus,
} from "./reports/reportTemplatePresets";
import { formatDateTime, humanizeCode } from "./reports/reportRunPresentation";
import { Button, Notice, SearchField, SelectField, StatusBadge, TextField, type StatusTone } from "./ui";
import "./reports/report-builder.css";

type LoadState = "loading" | "live" | "error";
const newSetupID = "__new__";

export type ReportingPageProps = {
  organizationName?: string;
  legalEntityName?: string;
  embedded?: boolean;
  onBack?: () => void;
  onDefinitionChange?: (definition: ReportDefinition) => void;
  definitionSource?: {
    items: ReportDefinition[];
    state: LoadState;
    refresh: () => void;
    onChange: (definition: ReportDefinition) => void;
  };
  onGenerate?: (definition: ReportDefinition) => void;
  loadDefinitions?: typeof listReportDefinitionsRequest;
  createDefinition?: typeof createReportDefinitionRequest;
  transitionDefinition?: typeof transitionReportDefinitionRequest;
  loadMatters?: typeof loadMatterSummariesRequest;
};

export function ReportingPage({
  organizationName, legalEntityName, embedded = false, onBack, onDefinitionChange, definitionSource, onGenerate,
  loadDefinitions = listReportDefinitionsRequest,
  createDefinition = createReportDefinitionRequest,
  transitionDefinition = transitionReportDefinitionRequest,
  loadMatters = loadMatterSummariesRequest,
}: ReportingPageProps) {
  const [localDefinitions, setLocalDefinitions] = useState<ReportDefinition[]>([]);
  const [localState, setLocalState] = useState<LoadState>("loading");
  const definitions = definitionSource?.items ?? localDefinitions;
  const state = definitionSource?.state ?? localState;
  const hasDefinitionSource = Boolean(definitionSource);
  const [refreshKey, setRefreshKey] = useState(0);
  const [selectedID, setSelectedID] = useState<string>();
  const [showCreate, setShowCreate] = useState(true);
  const [name, setName] = useState("");
  const [area, setArea] = useState<ReportSetupArea>("VENDORS");
  const [focus, setFocus] = useState<ReportSetupFocus>("OVERVIEW");
  const [matterSearch, setMatterSearch] = useState("");
  const [matterID, setMatterID] = useState("");
  const [matterOptions, setMatterOptions] = useState<MatterSummary[]>([]);
  const [matterState, setMatterState] = useState<LoadState>("live");
  const [command, setCommand] = useState<"idle" | "saving" | "transitioning">("idle");
  const commandPending = useRef(false);
  const [message, setMessage] = useState<string>();
  const [error, setError] = useState<string>();

  useEffect(() => {
    if (hasDefinitionSource) return;
    const controller = new AbortController();
    setLocalState("loading");
    void loadDefinitions(true, controller.signal).then((items) => {
      if (controller.signal.aborted) return;
      setLocalDefinitions(items);
      setLocalState("live");
    }).catch((reason: unknown) => {
      if (!controller.signal.aborted && !isAbortError(reason)) setLocalState("error");
    });
    return () => controller.abort();
  }, [hasDefinitionSource, loadDefinitions, refreshKey]);

  useEffect(() => {
    if (!showCreate || area !== "BOARD_BRIEF") return;
    let active = true;
    setMatterState("loading");
    void loadMatters({ q: matterSearch.trim(), limit: 20 }).then((page) => {
      if (!active) return;
      setMatterOptions(page.items);
      setMatterState("live");
      setMatterID((current) => page.items.some((item) => item.matter.id === current) ? current : "");
    }).catch(() => {
      if (!active) return;
      setMatterOptions([]);
      setMatterState("error");
    });
    return () => { active = false; };
  }, [area, loadMatters, matterSearch, showCreate]);

  const selected = showCreate ? undefined : definitions.find((item) => item.id === selectedID);
  const selectionUnavailable = !showCreate && !selected;
  const busy = command !== "idle";
  const displayArea = selected ? reportSetupArea(selected) : area;
  const displayFocus = selected ? reportSetupFocus(selected) : focus;
  const format: ReportFormat = selected?.format ?? (area === "BOARD_BRIEF" ? "PDF" : "XLSX");
  const scopeLabel = legalEntityName || organizationName || "Current legal entity";
  const issue = matterOptions.find((item) => item.matter.id === (selected?.scope_kind === "MATTER" ? selected.scope_ref : matterID));
  const nextAction = selected ? nextSetupAction(selected.status) : undefined;
  const canGenerate = selected?.status === "ACTIVE" && selected.effective;
  const scopeKind = selected?.scope_kind ?? (area === "BOARD_BRIEF" ? "MATTER" : "LEGAL_ENTITY");
  const scopeRef = scopeKind === "MATTER" && issue ? `${issue.matter.reference} — ${issue.matter.title}`
    : selected?.scope_ref ?? (area === "BOARD_BRIEF" ? matterID : undefined);

  function updateDefinition(definition: ReportDefinition) {
    if (definitionSource) definitionSource.onChange(definition);
    else setLocalDefinitions((current) => [definition, ...current.filter((item) => item.id !== definition.id)]);
    onDefinitionChange?.(definition);
  }

  function beginCreate() {
    if (commandPending.current) return;
    setName(""); setArea("VENDORS"); setFocus("OVERVIEW");
    setMatterSearch(""); setMatterID(""); setMatterOptions([]); setMatterState("live");
    setError(undefined); setMessage(undefined); setShowCreate(true);
  }

  async function saveSetup() {
    if (commandPending.current || state !== "live") return;
    if (name.trim().length < 3) { setError("Enter a setup name of at least 3 characters."); return; }
    if (area === "BOARD_BRIEF" && !matterID) { setError("Choose an issue for the board brief."); return; }
    commandPending.current = true;
    setCommand("saving"); setError(undefined); setMessage(undefined);
    try {
      const created = await createDefinition(buildReportSetupInput(name.trim(), area, focus, matterID));
      updateDefinition(created);
      setSelectedID(created.id); setShowCreate(false);
      setMessage(`${created.name} saved.`);
    } catch (reason: unknown) {
      setError(readCommandError(reason, "Couldn’t save setup. Retry."));
    } finally {
      commandPending.current = false;
      setCommand("idle");
    }
  }

  async function transition(action: ReportDefinitionAction) {
    if (!selected || commandPending.current || state !== "live") return;
    commandPending.current = true;
    setCommand("transitioning"); setError(undefined); setMessage(undefined);
    try {
      const updated = await transitionDefinition(selected.id, action, {
        expected_version: selected.version, checksum_seen: selected.checksum,
      });
      updateDefinition(updated);
      setMessage(transitionMessage(action));
    } catch (reason: unknown) {
      setError(readCommandError(reason, transitionFailure(action)));
    } finally {
      commandPending.current = false;
      setCommand("idle");
    }
  }

  return <section className="report-builder" aria-labelledby="report-builder-heading" data-embedded={embedded || undefined}>
    <header className="report-builder__heading">
      <div><span className="report-builder__eyebrow">Report designer</span><h2 id="report-builder-heading">Build report</h2><p>{scopeLabel}</p></div>
      <div className="report-builder__actions">
        <Button variant="secondary" isDisabled={busy} onPress={beginCreate}>New setup</Button>
        {showCreate
          ? <Button variant="primary" isLoading={command === "saving"} isDisabled={state !== "live"} onPress={() => void saveSetup()}>Save setup</Button>
          : nextAction
            ? <Button variant="primary" isLoading={command === "transitioning"} isDisabled={state !== "live"} onPress={() => void transition(nextAction.action)}>{nextAction.label}</Button>
            : selected?.status === "ACTIVE" && onGenerate
              ? <Button variant="primary" isDisabled={!canGenerate || busy || state !== "live"} onPress={() => onGenerate(selected)}>Generate report</Button>
              : null}
      </div>
    </header>
    {message && <Notice tone="success">{message}</Notice>}
    {error && <Notice tone="error">{error}</Notice>}
    {state === "error" && <Notice tone="error">Couldn’t load setups. Refresh setups before saving or changing a report.</Notice>}

    <div className="report-builder__layout">
      <aside className="report-builder__setup" aria-label="Report setup">
        <h3>Report setup</h3>
        <SelectField label="Saved setup" value={showCreate ? newSetupID : selected?.id}
          placeholder={state === "loading" ? "Loading setups…" : "Choose a setup"} allowsEmpty={false}
          isDisabled={busy || state === "loading"}
          options={[{ id: newSetupID, label: "New report setup" }, ...definitions.map((item) => ({ id: item.id, label: item.name, description: setupStatusLabel(item) }))]}
          onChange={(value) => {
            if (!value || commandPending.current) return;
            setShowCreate(value === newSetupID);
            if (value !== newSetupID) setSelectedID(value);
            setError(undefined); setMessage(undefined);
          }}/>
        {state === "loading" && <p role="status">Loading setups…</p>}
        {state === "live" && definitions.length === 0 && <p>No saved setups in this legal entity. Enter a name to create one.</p>}
        {showCreate ? <>
          <TextField label="Name" value={name} onChange={(value) => { setName(value); setError(undefined); }}
            placeholder="e.g. Monthly vendor overview" maxLength={120} isRequired isDisabled={busy}/>
          <SelectField label="Area" value={area} placeholder="Choose an area" allowsEmpty={false}
            options={reportSetupAreaOptions} isDisabled={busy}
            onChange={(value) => {
              if (!value) return;
              setArea(value); setFocus("OVERVIEW"); setMatterSearch(""); setMatterID(""); setMatterOptions([]); setError(undefined);
            }}/>
          {area === "BOARD_BRIEF" ? <>
            <SearchField label="Find issue" value={matterSearch} onChange={setMatterSearch}
              placeholder="Search reference or title" isLoading={matterState === "loading"} isDisabled={busy}/>
            <SelectField label="Issue" value={matterID || undefined} isRequired isDisabled={busy || matterState === "loading"}
              placeholder={matterState === "loading" ? "Loading issues" : "Choose an issue"}
              options={matterOptions.map((item) => ({ id: item.matter.id, label: `${item.matter.reference} — ${item.matter.title}` }))}
              onChange={(value) => { setMatterID(value || ""); setError(undefined); }}/>
            {matterState === "error" && <Notice tone="warning">Issues could not be loaded. Retry the search.</Notice>}
          </> : <SelectField label="Report type" value={focus} placeholder="Choose a report type" allowsEmpty={false}
            options={reportSetupFocusOptions} isDisabled={busy} onChange={(value) => { if (value) setFocus(value); }}/>}
        </> : selected && <dl className="report-builder__facts">
          <div><dt>Area</dt><dd>{reportSetupAreaLabel(displayArea)}</dd></div>
          <div><dt>Report type</dt><dd>{displayArea === "BOARD_BRIEF" ? "Issue brief" : reportSetupFocusLabel(displayFocus)}</dd></div>
          <div><dt>Definition revision</dt><dd>{selected.current_version}</dd></div>
          <div><dt>Updated</dt><dd>{formatDate(selected.updated_at)}</dd></div>
        </dl>}
        <Button variant="quiet" isLoading={state === "loading"} isDisabled={busy} onPress={() => definitionSource ? definitionSource.refresh() : setRefreshKey((value) => value + 1)}>Refresh setups</Button>
        <p className="report-builder__help">{selectionUnavailable ? "Choose a saved setup or start a new setup." : showCreate ? "Save this setup, then send it for independent review before activation." : "Saved contents are fixed for this definition revision. Create a new setup for different contents."}</p>
      </aside>

      <article className="report-builder__paper" aria-label="Setup preview">
        {selectionUnavailable ? <Notice tone="warning">The selected setup is no longer available. Choose a saved setup or start a new setup.</Notice> : <>
        <header className="report-builder__paper-heading">
          <span className="report-builder__eyebrow">Setup preview</span>
          <h3>{selected?.name || name.trim() || "New report"}</h3>
          <p>{scopeLabel} · {reportSetupAreaLabel(displayArea)}</p>
          <p>Contents and filters only. Generate an approved setup to view report data.</p>
        </header>
        <section className="report-builder__paper-section" aria-labelledby="report-contents-heading">
          <h4 id="report-contents-heading">Report contents</h4>
          <ol className="report-builder__contents">
            {reportContents(format, displayArea).map((item) => <li key={item.title}><strong>{item.title}</strong><p>{item.description}</p></li>)}
          </ol>
        </section>
        <section className="report-builder__paper-section" aria-labelledby="report-population-heading">
          <h4 id="report-population-heading">Scope and filters</h4>
          <dl className="report-builder__facts">
            <div><dt>Legal entity</dt><dd>{scopeLabel}</dd></div>
            <div><dt>Population</dt><dd>{reportSetupAreaLabel(displayArea)} · {displayArea === "BOARD_BRIEF" ? "Issue brief" : reportSetupFocusLabel(displayFocus)}</dd></div>
            {scopeKind !== "LEGAL_ENTITY" && <div><dt>{scopeKind === "MATTER" ? "Issue" : "Program"}</dt><dd>{scopeKind === "MATTER" && selected && !issue ? `Issue name unavailable. Saved reference: ${scopeRef || "Not recorded"}` : scopeRef || "Choose an issue in Report setup."}</dd></div>}
          </dl>
          {selected ? <FilterSummary filter={selected.filter}/> : <p>{presetPopulation(displayArea, focus)}</p>}
          <p>Records and source timestamps are captured when the report runs. Report totals are unavailable until generation finishes.</p>
        </section>
        {selected && <section className="report-builder__paper-section" aria-labelledby="report-approval-heading">
          <h4 id="report-approval-heading">Review and approval</h4>
          <dl className="report-builder__facts">
            <div><dt>Proposer</dt><dd>{selected.maker_id || "Not recorded"}</dd></div>
            <div><dt>Reviewer</dt><dd>{selected.reviewer_id || "Not recorded"}</dd></div>
            <div><dt>Authorizer</dt><dd>{selected.checker_id || "Not recorded"}</dd></div>
          </dl>
          {selected.reviewer_note && <p>{selected.reviewer_note}</p>}
          <details className="report-builder__audit"><summary>Setup record</summary><dl className="report-builder__facts">
            <div><dt>Reference</dt><dd>{selected.code}</dd></div>
            <div><dt>Version</dt><dd>{selected.version}</dd></div>
            <div><dt>Checksum</dt><dd>{selected.checksum}</dd></div>
            <div><dt>Approved</dt><dd>{selected.approved_at ? formatDateTime(selected.approved_at) : "Not recorded"}</dd></div>
            <div><dt>Effective from</dt><dd>{selected.effective_from ? formatDateTime(selected.effective_from) : "Not recorded"}</dd></div>
            <div><dt>Effective until</dt><dd>{selected.effective_until ? formatDateTime(selected.effective_until) : "No end date recorded"}</dd></div>
          </dl></details>
        </section>}
        <footer className="report-builder__paper-footer">Setup preview · {format} · {selected ? `Revision ${selected.current_version}` : "Not saved"}</footer>
        </>}
      </article>

      <aside className="report-builder__output" aria-label="Output">
        <h3>Output</h3>
        <div className="report-builder__output-fact"><span>File format</span><strong>{selectionUnavailable ? "Unavailable" : formatLabel(format)}</strong></div>
        <div className="report-builder__output-fact"><span>Delivery</span><strong>Manual download</strong></div>
        <div className="report-builder__output-fact"><span>Status</span><StatusBadge tone={selected ? setupStatusTone(selected) : "neutral"}>{selectionUnavailable ? "Unavailable" : selected ? setupStatusLabel(selected) : "Not saved"}</StatusBadge></div>
        <p className="report-builder__output-note">Download generated files from Reports. Recurring delivery is unavailable.</p>
        {!selectionUnavailable && <ol className="report-builder__steps" aria-label="Report preparation">
          <li><strong>Save setup</strong><span>{selected ? "Saved" : "Not saved"}</span></li>
          <li><strong>Independent review</strong><span>{selected?.reviewer_id ? "Recorded" : selected?.status === "PENDING_REVIEW" ? "Awaiting review" : "Not recorded"}</span></li>
          <li><strong>Activation</strong><span>{canGenerate ? "Active" : selected?.status === "RETIRED" ? "Retired" : selected?.checker_id ? "Not effective" : "Not approved"}</span></li>
        </ol>}
        <p className="report-builder__guidance">{selectionUnavailable ? "Select an available setup to review its output and approval status." : selected ? setupGuidance(selected) : "Save the report setup to begin review."}</p>
        {selected?.status === "ACTIVE" && <Button variant="quiet" isLoading={command === "transitioning"} isDisabled={state !== "live"} onPress={() => void transition("retire")}>Retire setup</Button>}
        {onBack && <Button variant="secondary" onPress={onBack}>View reports</Button>}
      </aside>
    </div>
  </section>;
}

function FilterSummary({ filter }: { filter?: ReportFilterExpression }) {
  if (!filter || (filter.kind === "group" && !filter.children?.length)) return <p>No additional saved filters.</p>;
  if (filter.kind === "condition") {
    const enumField = ["status", "overall_state", "due_condition", "matter_type", "priority"].includes(filter.field ?? "");
    const value = filter.value ?? "Not recorded";
    return <p>{humanizeCode(filter.field ?? "Field")} {filter.operator === "is_not" ? "is not" : filter.operator === "contains" ? "contains" : filter.operator} {enumField ? humanizeCode(value) : value}.</p>;
  }
  return <div className="report-builder__filters"><p>{filter.operator === "or" ? "Any of these conditions:" : "All of these conditions:"}</p><ul>{filter.children?.map((child, index) => <li key={index}><FilterSummary filter={child}/></li>)}</ul></div>;
}

function reportContents(format: ReportFormat, area: ReportSetupArea) {
  if (area === "BOARD_BRIEF") return [
    { title: "Issue position", description: "The selected issue, current state, ownership and linked Programs." },
    { title: "Decisions and follow-up", description: "Recorded decisions, actions, outcome checks, losses and recovery." },
    { title: "Supporting context", description: "Linked forms, responses, vendor work and connected sources." },
  ];
  if (format === "XLSX") return [
    { title: "Summary", description: "Included row count and a chart when the records support a category breakdown." },
    { title: "Data", description: "The included records and their available fields." },
    { title: "Report information", description: "Report title, as-of time and included row count." },
  ];
  return [{ title: "Record data", description: format === "NDJSON" ? "One included record per line in NDJSON format." : "Included records and their available fields in CSV format." }];
}

function presetPopulation(area: ReportSetupArea, focus: ReportSetupFocus) {
  if (area === "BOARD_BRIEF") return "Only the selected issue and its permitted linked records are included.";
  if (focus === "OVERVIEW") return `The report includes authorized ${reportSetupAreaLabel(area).toLowerCase()} in this legal entity.`;
  if (area === "VENDORS") return "Vendors under review, restricted, suspended or exiting are included.";
  if (area === "PROGRAMS") return "Programs with gaps, incomplete evidence, pending work, overdue items, review needs or unknown status are included.";
  return area === "WORK" ? "Issues and changes with recorded exceptions are included." : "Processing activities with recorded exceptions are included.";
}

function nextSetupAction(status: ReportDefinitionStatus): { action: ReportDefinitionAction; label: string } | undefined {
  if (status === "DRAFT") return { action: "submit", label: "Send for review" };
  if (status === "PENDING_REVIEW") return { action: "review", label: "Record review" };
  if (status === "REVIEWED") return { action: "activate", label: "Activate setup" };
  return undefined;
}
function setupGuidance(definition: ReportDefinition) {
  if (definition.status === "DRAFT") return "Send this saved setup for independent review.";
  if (definition.status === "PENDING_REVIEW") return "An independent reviewer must record the review.";
  if (definition.status === "REVIEWED") return "An authorized approver must activate this setup.";
  if (definition.status === "ACTIVE") return definition.effective ? "This setup is ready to generate a report." : "This setup is not currently effective. Review the effective dates in Setup record.";
  return "This setup is retired. Create a new setup to generate another report.";
}
function setupStatusLabel(definition: ReportDefinition) {
  if (definition.status === "DRAFT") return "Draft";
  if (definition.status === "PENDING_REVIEW") return "Needs review";
  if (definition.status === "REVIEWED") return "Ready to activate";
  if (definition.status === "ACTIVE") return definition.effective ? "Ready" : "Not effective";
  return "Retired";
}
function setupStatusTone(definition: ReportDefinition): StatusTone {
  if (definition.status === "ACTIVE") return definition.effective ? "success" : "warning";
  if (definition.status === "PENDING_REVIEW") return "warning";
  if (definition.status === "REVIEWED") return "info";
  if (definition.status === "RETIRED") return "neutral";
  return "unknown";
}
function formatLabel(format: ReportFormat) {
  return format === "XLSX" ? "Excel (.xlsx)" : format === "PDF" ? "PDF" : format === "CSV" ? "CSV" : "NDJSON";
}
function transitionMessage(action: ReportDefinitionAction) {
  return action === "submit" ? "Sent for review." : action === "review" ? "Review recorded." : action === "activate" ? "Activated." : action === "retire" ? "Setup retired." : "Report setup updated.";
}
function transitionFailure(action: ReportDefinitionAction) {
  return action === "review" ? "Independent reviewer required." : action === "activate" ? "Authorized approver required." : "Action failed. Refresh setups and retry.";
}
function readCommandError(error: unknown, fallback: string) {
  if (typeof error === "object" && error !== null && "code" in error && error.code === "report_governance_blocked") return fallback;
  return error instanceof Error && error.message ? error.message : fallback;
}
function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Not recorded" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date);
}
