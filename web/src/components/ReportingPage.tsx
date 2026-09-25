import { useEffect, useMemo, useState } from "react";
import {
  createReportDefinition as createReportDefinitionRequest,
  createReportRun as createReportRunRequest,
  downloadReportRun as downloadReportRunRequest,
  getReportDefinitionHistory as getReportDefinitionHistoryRequest,
  listReportDefinitions as listReportDefinitionsRequest,
  listReportFilterFields as listReportFilterFieldsRequest,
  listReportRuns as listReportRunsRequest,
  transitionReportDefinition as transitionReportDefinitionRequest,
} from "../reportingApi";
import type {
  ReportDataset,
  ReportDefinition,
  ReportDefinitionAction,
  ReportDefinitionInput,
  ReportDefinitionRevision,
  ReportDefinitionStatus,
  ReportDefinitionTransitionInput,
  ReportFilterExpression,
  ReportFilterFieldDefinition,
  ReportFormat,
  ReportRun,
  ReportScopeKind,
} from "../reportingTypes";
import { ReportFilterEditor, validateReportFilter } from "./ReportFilterEditor";
import { Button, DataTable, EmptyState, Notice, SelectField, StatusBadge, TextArea, TextField, type DataColumn, type StatusTone } from "./ui";
import "./ropa.css";

type LoadState = "loading" | "live" | "error";
type RunState = LoadState;

export type ReportingPageProps = {
  organizationName?: string;
  legalEntityName?: string;
  embedded?: boolean;
  onOpenRegister?: () => void;
  onBack?: () => void;
  loadFilterFields?: typeof listReportFilterFieldsRequest;
  loadDefinitions?: typeof listReportDefinitionsRequest;
  loadRuns?: typeof listReportRunsRequest;
  loadDefinitionHistory?: typeof getReportDefinitionHistoryRequest;
  createDefinition?: typeof createReportDefinitionRequest;
  transitionDefinition?: typeof transitionReportDefinitionRequest;
  createRun?: typeof createReportRunRequest;
  downloadRun?: typeof downloadReportRunRequest;
};

const defaultDraft: DefinitionDraft = {
  code: "",
  name: "",
  description: "",
  dataset: "PROCESSING_ACTIVITIES",
  scope_kind: "LEGAL_ENTITY",
  scope_ref: "",
  format: "XLSX",
  effective_from: "",
  filter: { kind: "group", operator: "and", children: [] },
};

type DefinitionDraft = {
  code: string;
  name: string;
  description: string;
  dataset: ReportDataset;
  scope_kind: ReportScopeKind;
  scope_ref: string;
  format: ReportFormat;
  effective_from: string;
  filter: ReportFilterExpression;
};

export function ReportingPage({
  organizationName,
  legalEntityName,
  embedded = false,
  onOpenRegister,
  onBack,
  loadFilterFields = listReportFilterFieldsRequest,
  loadDefinitions = listReportDefinitionsRequest,
  loadRuns = listReportRunsRequest,
  loadDefinitionHistory = getReportDefinitionHistoryRequest,
  createDefinition = createReportDefinitionRequest,
  transitionDefinition = transitionReportDefinitionRequest,
  createRun = createReportRunRequest,
  downloadRun = downloadReportRunRequest,
}: ReportingPageProps) {
  const scope = legalEntityName || "this legal entity";
  const [fieldResponse, setFieldResponse] = useState<{ fields: ReportFilterFieldDefinition[] }>();
  const [definitions, setDefinitions] = useState<ReportDefinition[]>([]);
  const [runs, setRuns] = useState<ReportRun[]>([]);
  const [selectedDefinitionID, setSelectedDefinitionID] = useState<string>();
  const [history, setHistory] = useState<ReportDefinitionRevision[]>([]);
  const [state, setState] = useState<LoadState>("loading");
  const [runsState, setRunsState] = useState<RunState>("loading");
  const [historyState, setHistoryState] = useState<LoadState>("loading");
  const [loadError, setLoadError] = useState<string>();
  const [runsError, setRunsError] = useState<string>();
  const [historyError, setHistoryError] = useState<string>();
  const [retry, setRetry] = useState(0);
  const [historyRetry, setHistoryRetry] = useState(0);
  const [showCreate, setShowCreate] = useState(false);
  const [draft, setDraft] = useState<DefinitionDraft>(defaultDraft);
  const [draftError, setDraftError] = useState<string>();
  const [commandState, setCommandState] = useState<"idle" | "saving" | "running" | "downloading">("idle");
  const [commandMessage, setCommandMessage] = useState<string>();
  const [commandError, setCommandError] = useState<string>();

  const selectedDefinition = definitions.find((definition) => definition.id === selectedDefinitionID) ?? definitions[0];
  const visibleRuns = useMemo(() => selectedDefinition ? runs.filter((run) => run.definition_id === selectedDefinition.id) : runs, [runs, selectedDefinition]);

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setRunsState("loading");
    setLoadError(undefined);
    setRunsError(undefined);
    void Promise.allSettled([
      loadFilterFields(controller.signal),
      loadDefinitions(false, controller.signal),
      loadRuns({}, controller.signal),
    ]).then(([fieldResult, definitionResult, runResult]) => {
      if (controller.signal.aborted) return;
      if (fieldResult.status === "fulfilled") setFieldResponse(fieldResult.value);
      if (definitionResult.status === "fulfilled") {
        const nextDefinitions = definitionResult.value;
        setDefinitions(nextDefinitions);
        setSelectedDefinitionID((current) => nextDefinitions.some((definition) => definition.id === current) ? current : nextDefinitions[0]?.id);
        setState("live");
      } else {
        setState("error");
        setLoadError(readError(definitionResult.reason, "Report definitions could not be loaded. Check the connection and try again."));
      }
      if (runResult.status === "fulfilled") {
        setRuns(runResult.value);
        setRunsState("live");
      } else {
        setRunsState("error");
        setRunsError(readError(runResult.reason, "Report runs could not be loaded. Check the connection and try again."));
      }
    });
    return () => controller.abort();
  }, [loadDefinitions, loadFilterFields, loadRuns, retry]);

  useEffect(() => {
    if (!selectedDefinition) {
      setHistory([]);
      setHistoryState("live");
      return;
    }
    const controller = new AbortController();
    setHistoryState("loading");
    setHistoryError(undefined);
    void loadDefinitionHistory(selectedDefinition.id, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setHistory(value);
      setHistoryState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setHistoryState("error");
      setHistoryError(readError(error, "The decision history for this report could not be loaded. Check the connection and try again."));
    });
    return () => controller.abort();
  }, [loadDefinitionHistory, selectedDefinition?.id, historyRetry]);

  function openRegister() {
    if (onOpenRegister) {
      onOpenRegister();
      return;
    }
    if (typeof window !== "undefined") window.location.hash = "#ropa";
  }

  function goBack() {
    if (onBack) {
      onBack();
      return;
    }
    openRegister();
  }

  function refresh() {
    setRetry((value) => value + 1);
    setHistoryRetry((value) => value + 1);
  }

  function beginCreate() {
    setDraft(defaultDraft);
    setDraftError(undefined);
    setCommandError(undefined);
    setShowCreate(true);
  }

  function updateDraft(patch: Partial<DefinitionDraft>) {
    setDraft((current) => ({ ...current, ...patch }));
    setDraftError(undefined);
  }

  async function saveDefinition() {
    const filterMessage = validateReportFilter(draft.filter, fieldResponse?.fields ?? [], draft.dataset);
    if (!draft.code.trim() || !draft.name.trim() || filterMessage) {
      setDraftError(filterMessage ?? "Enter a report code and name, and complete the filter before saving the definition.");
      return;
    }
    setCommandState("saving");
    setCommandError(undefined);
    try {
      const input: ReportDefinitionInput = {
        code: draft.code.trim().toUpperCase(),
        name: draft.name.trim(),
        description: draft.description.trim(),
        dataset: draft.dataset,
        scope_kind: draft.scope_kind,
        scope_ref: draft.scope_kind === "LEGAL_ENTITY" ? undefined : draft.scope_ref.trim() || undefined,
        format: draft.format,
        filter: draft.filter,
        effective_from: draft.effective_from ? new Date(`${draft.effective_from}T00:00:00Z`).toISOString() : undefined,
      };
      const created = await createDefinition(input);
      setDefinitions((current) => [created, ...current]);
      setSelectedDefinitionID(created.id);
      setShowCreate(false);
      setCommandMessage("Report definition saved as a draft. Send it for review when the filter and scope are ready.");
    } catch (error: unknown) {
      setCommandError(readError(error, "The report definition could not be saved. Check the fields and try again."));
    } finally {
      setCommandState("idle");
    }
  }

  async function transitionSelected(action: ReportDefinitionAction) {
    if (!selectedDefinition) return;
    setCommandState("saving");
    setCommandError(undefined);
    const input: ReportDefinitionTransitionInput = {
      expected_version: selectedDefinition.version,
      checksum_seen: selectedDefinition.checksum,
    };
    try {
      const updated = await transitionDefinition(selectedDefinition.id, action, input);
      setDefinitions((current) => current.map((definition) => definition.id === updated.id ? updated : definition));
      setCommandMessage(actionMessage(action));
    } catch (error: unknown) {
      setCommandError(readError(error, "The report decision could not be recorded. Check the current report state and try again."));
    } finally {
      setCommandState("idle");
    }
  }

  async function runSelected() {
    if (!selectedDefinition || !runAvailability(selectedDefinition).allowed) return;
    setCommandState("running");
    setCommandError(undefined);
    try {
      const run = await createRun(selectedDefinition.id, selectedDefinition.current_version);
      setRuns((current) => [run, ...current]);
      setCommandMessage("Report run queued. Check the run state before relying on a downloaded file.");
    } catch (error: unknown) {
      setCommandError(readError(error, "The report run could not be queued. Check the definition and try again."));
    } finally {
      setCommandState("idle");
    }
  }

  async function downloadSelected(run: ReportRun) {
    if (run.status !== "READY") return;
    setCommandState("downloading");
    setCommandError(undefined);
    try {
      const result = await downloadRun(run.id);
      saveBlob(result.blob, result.filename ?? `${run.definition_code}.${run.format === "NDJSON" ? "ndjson" : run.format === "XLSX" ? "xlsx" : "csv"}`);
      setCommandMessage("Report file downloaded after the protected access check.");
    } catch (error: unknown) {
      setCommandError(readError(error, "The report file could not be downloaded. Check the run state and try again."));
    } finally {
      setCommandState("idle");
    }
  }

  const statusCounts = countDefinitionStates(definitions);
  const scopeLabel = `${scope} · report definitions`;

  return <section className="reporting-page" aria-labelledby={embedded ? undefined : "reporting-heading"} aria-label={embedded ? "Report templates" : undefined}>
    {embedded ? <div className="section-header report-section-header">
      <div><span className="eyebrow">Report governance</span><h2>Report templates</h2><p>Configure reusable report populations and keep review, authorisation and version history separate from generated files.</p></div>
      <div className="topbar-actions">
        <Button variant="secondary" onPress={refresh} isLoading={state === "loading" || runsState === "loading"}>Refresh</Button>
        <Button variant="primary" onPress={showCreate ? () => setShowCreate(false) : beginCreate}>{showCreate ? "Close form" : "New template"}</Button>
      </div>
    </div> : <header className="topbar ropa-page-header">
      <div>
        <span className="eyebrow">{organizationName || "ClearSight"} · {scope}</span>
        <h1 id="reporting-heading">Reports</h1>
        <p>Current report definitions, source scope and completed report runs for {scope}.</p>
      </div>
      <div className="topbar-actions">
        {onOpenRegister && <Button variant="secondary" onPress={goBack}>Open processing activities</Button>}
        <Button variant="secondary" onPress={refresh} isLoading={state === "loading" || runsState === "loading"}>Refresh reports</Button>
        <Button variant="primary" onPress={showCreate ? () => setShowCreate(false) : beginCreate}>{showCreate ? "Close report form" : "Define a report"}</Button>
      </div>
    </header>}

    <section className="report-status-strip" aria-label="Report definition status">
      <div><span className="eyebrow">Report governance</span><h2>Definitions checked for {scope}</h2><p>These counts come from the report definitions returned for the current legal entity.</p></div>
      <div className="report-status-strip__metrics">
        <ReportMetric label="Definitions" value={definitions.length} />
        <ReportMetric label="Waiting for review" value={statusCounts.pendingReview} tone="warning" />
        <ReportMetric label="Reviewed, not authorised" value={statusCounts.reviewed} tone="info" />
        <ReportMetric label="Active" value={statusCounts.active} tone="success" />
      </div>
    </section>

    {commandMessage && <Notice tone="success"><span>{commandMessage}</span></Notice>}
    {commandError && <Notice tone="error"><span>{commandError}</span></Notice>}

    {showCreate && <DefinitionCreateForm draft={draft} fields={fieldResponse?.fields ?? []} error={draftError} busy={commandState === "saving"} onChange={updateDraft} onSave={saveDefinition} />}

    <section className="report-definitions" aria-labelledby="report-definitions-heading">
      <div className="section-header report-section-header">
        <div><h2 id="report-definitions-heading">Report definitions</h2><p>Choose a definition to inspect its governance history and latest bounded run.</p></div>
      </div>
      {state === "loading" && definitions.length === 0 && <p className="ropa-load-state" role="status">Loading report definitions…</p>}
      {state === "error" && <Notice tone="error"><span>{loadError || "Report definitions could not be loaded. Check the connection and try again."}</span> <Button variant="secondary" size="compact" onPress={refresh}>Retry definitions</Button></Notice>}
      {state === "live" && definitions.length === 0 && <EmptyState population={scopeLabel} title="No report definitions are recorded in this legal entity yet" description="The current legal entity has no governed report definitions. The next valid action is to use Define a report to record the dataset, scope and published filter fields." />}
      {definitions.length > 0 && <DefinitionTable definitions={definitions} selectedID={selectedDefinition?.id} onSelect={setSelectedDefinitionID} />}
    </section>

    {selectedDefinition && <SelectedDefinitionPanel
      definition={selectedDefinition}
      history={history}
      historyState={historyState}
      historyError={historyError}
      commandState={commandState}
      onRetryHistory={() => setHistoryRetry((value) => value + 1)}
      onTransition={transitionSelected}
      onRun={runSelected}
    />}

    {!embedded && <section className="report-runs" aria-labelledby="report-runs-heading">
      <div className="section-header report-section-header">
        <div><h2 id="report-runs-heading">Report runs</h2><p>Each row records the material source boundary used to generate the file and whether the bounded population completed.</p></div>
      </div>
      {runsState === "loading" && runs.length === 0 && <p className="ropa-load-state" role="status">Loading report runs…</p>}
      {runsState === "error" && <Notice tone="error"><span>{runsError || "Report runs could not be loaded. Check the connection and try again."}</span> <Button variant="secondary" size="compact" onPress={refresh}>Retry runs</Button></Notice>}
      {runsState === "live" && visibleRuns.length === 0 && <EmptyState population={`${scope} · report runs for ${selectedDefinition?.name || "the selected definition"}`} title="No report runs are recorded for this definition" description="The selected definition has no queued, completed or failed run in the current legal entity. The next valid action is to activate the definition or run it when its governance state allows." />}
      {visibleRuns.length > 0 && <RunTable runs={visibleRuns} onDownload={downloadSelected} downloading={commandState === "downloading"} />}
    </section>}
  </section>;
}

function DefinitionCreateForm({ draft, fields, error, busy, onChange, onSave }: { draft: DefinitionDraft; fields: readonly ReportFilterFieldDefinition[]; error?: string; busy: boolean; onChange: (patch: Partial<DefinitionDraft>) => void; onSave: () => void }) {
  const datasetFields = fields.filter((field) => field.dataset === draft.dataset);
  return <section className="report-create-panel" aria-labelledby="report-create-heading">
    <div className="section-header report-section-header"><div><h2 id="report-create-heading">Define a report</h2><p>Save a draft first. The draft must then be sent through review and authorisation before a run is available.</p></div></div>
    {error && <Notice tone="error"><span>{error}</span></Notice>}
    <div className="report-create-panel__grid">
      <TextField label="Report code" value={draft.code} onChange={(value) => onChange({ code: value.toUpperCase() })} description="Use a short code that identifies this report in audit history and downloads." isRequired maxLength={48} />
      <TextField label="Report name" value={draft.name} onChange={(value) => onChange({ name: value })} description="Name the business question or population this report answers." isRequired maxLength={120} />
      <TextArea label="Description" value={draft.description} onChange={(value) => onChange({ description: value })} description="State the purpose, audience or evidence question this report supports." maxLength={1000} />
      <SelectField label="Dataset" value={draft.dataset} placeholder="Choose a dataset" options={[
        { id: "VENDORS", label: "Vendors" },
        { id: "PROGRAMS", label: "Programs" },
        { id: "MATTERS", label: "Work — all issues and changes" },
        { id: "MATTER_EXCEPTIONS", label: "Work — exceptions and overdue obligations" },
        { id: "PROCESSING_ACTIVITIES", label: "Processing activities" },
        { id: "PROCESSING_ACTIVITY_EXCEPTIONS", label: "Processing activities with open exceptions" },
      ]} onChange={(value) => value && onChange({ dataset: value, scope_kind: value === "VENDORS" ? "LEGAL_ENTITY" : draft.scope_kind, filter: { kind: "group", operator: "and", children: [] }, scope_ref: "" })} isRequired />
      <SelectField label="Scope" value={draft.scope_kind} placeholder="Choose a scope" options={draft.dataset === "VENDORS" ? [
        { id: "LEGAL_ENTITY", label: "Whole legal entity" },
      ] : [
        { id: "LEGAL_ENTITY", label: "Whole legal entity" },
        { id: "PROGRAM", label: "One Program" },
        { id: "MATTER", label: "One issue or change" },
      ]} onChange={(value) => value && onChange({ scope_kind: value, scope_ref: "" })} isRequired />
      {draft.scope_kind !== "LEGAL_ENTITY" && <TextField label={draft.scope_kind === "PROGRAM" ? "Program identifier" : "Issue or change identifier"} value={draft.scope_ref} onChange={(value) => onChange({ scope_ref: value })} description="Enter the stored identifier returned by the authoritative record." isRequired />}
      <SelectField label="File format" value={draft.format} placeholder="Choose a file format" options={[{ id: "XLSX", label: "Excel workbook" }, { id: "CSV", label: "CSV spreadsheet" }, { id: "NDJSON", label: "NDJSON data file" }]} onChange={(value) => value && onChange({ format: value })} isRequired />
      <TextField label="Effective from" type="date" value={draft.effective_from} onChange={(value) => onChange({ effective_from: value })} description="Leave blank if the authorizer should choose the effective date during activation." />
    </div>
    <ReportFilterEditor fields={fields} dataset={draft.dataset} value={draft.filter} onChange={(filter) => onChange({ filter })} onSave={() => onSave()} />
    <div className="report-create-panel__actions"><Button variant="primary" onPress={onSave} isLoading={busy}>Save report draft</Button></div>
  </section>;
}

function DefinitionTable({ definitions, selectedID, onSelect }: { definitions: readonly ReportDefinition[]; selectedID?: string; onSelect: (id: string) => void }) {
  const columns: readonly DataColumn<ReportDefinition>[] = [
    { id: "definition", header: "Report definition", mobileLayout: "full-width", render: (definition) => <span className="report-definition-identity"><strong>{definition.name}</strong><small>{definition.code}</small></span>, accessibleText: (definition) => `${definition.name}, ${definition.code}` },
    { id: "status", header: "Governance state", kind: "status", render: (definition) => <StatusBadge tone={definitionStatusTone(definition.status)}>{definitionStatusLabel(definition.status)}</StatusBadge>, accessibleText: (definition) => definitionStatusLabel(definition.status) },
    { id: "dataset", header: "Dataset", render: (definition) => datasetLabel(definition.dataset), accessibleText: (definition) => datasetLabel(definition.dataset) },
    { id: "scope", header: "Scope", render: (definition) => scopeDefinitionLabel(definition), accessibleText: (definition) => scopeDefinitionLabel(definition) },
    { id: "roles", header: "Decision roles", render: (definition) => <span className="report-role-summary"><RoleSummary label="Proposer" value={definition.maker_id} /><RoleSummary label="Reviewer" value={definition.reviewer_id} /><RoleSummary label="Authorizer" value={definition.checker_id} /></span>, accessibleText: (definition) => `Proposer ${definition.maker_id || "not recorded"}, reviewer ${definition.reviewer_id || "not recorded"}, authorizer ${definition.checker_id || "not recorded"}` },
    { id: "effective", header: "Effective date", render: (definition) => effectiveDateLabel(definition), accessibleText: (definition) => effectiveDateLabel(definition) },
    { id: "version", header: "Current version", kind: "number", render: (definition) => String(definition.current_version), accessibleText: (definition) => `Version ${definition.current_version}` },
  ];
  return <DataTable ariaLabel="Report definitions" rows={definitions} rowKey={(definition) => definition.id} rowName={(definition) => `${definition.name}, ${definitionStatusLabel(definition.status)}, ${datasetLabel(definition.dataset)}, ${scopeDefinitionLabel(definition)}`} columns={columns} selectedKey={selectedID} onSelectionChange={(definition) => onSelect(definition.id)} onRowAction={(definition) => onSelect(definition.id)} />;
}

function SelectedDefinitionPanel({ definition, history, historyState, historyError, commandState, onRetryHistory, onTransition, onRun }: { definition: ReportDefinition; history: readonly ReportDefinitionRevision[]; historyState: LoadState; historyError?: string; commandState: string; onRetryHistory: () => void; onTransition: (action: ReportDefinitionAction) => void; onRun: () => void }) {
  const availability = runAvailability(definition);
  return <>
    <section className="report-selected-definition" role="region" aria-label="Selected report definition">
      <div className="report-selected-definition__header">
        <div><span className="eyebrow">Selected definition</span><h2>{definition.name}</h2><p>{definition.description || "No report purpose is recorded for this definition."}</p></div>
        <StatusBadge tone={definitionStatusTone(definition.status)}>{definitionStatusLabel(definition.status)}</StatusBadge>
      </div>
      <dl className="report-definition-facts">
        <Fact label="Dataset" value={datasetLabel(definition.dataset)} />
        <Fact label="Scope" value={scopeDefinitionLabel(definition)} />
        <Fact label="Proposer" value={definition.maker_id || "Not recorded"} />
        <Fact label="Reviewer" value={definition.reviewer_id || "Not recorded"} />
        <Fact label="Authorizer" value={definition.checker_id || "Not recorded"} />
        <Fact label="Effective date" value={effectiveDateLabel(definition)} />
      </dl>
      <div className="report-selected-definition__actions">
        <Button variant="primary" onPress={onRun} isDisabled={!availability.allowed} isLoading={commandState === "running"} aria-describedby="report-run-reason">Run report</Button>
        <span id="report-run-reason" className="report-control-reason">{availability.reason}</span>
        {definition.status === "DRAFT" && <Button variant="secondary" onPress={() => onTransition("submit")} isLoading={commandState === "saving"}>Send for review</Button>}
        {definition.status === "PENDING_REVIEW" && <Button variant="secondary" onPress={() => onTransition("review")} isLoading={commandState === "saving"}>Record review</Button>}
        {definition.status === "REVIEWED" && <Button variant="secondary" onPress={() => onTransition("activate")} isLoading={commandState === "saving"}>Activate report</Button>}
        {definition.status === "ACTIVE" && <Button variant="quiet" onPress={() => onTransition("retire")} isLoading={commandState === "saving"}>Retire report</Button>}
      </div>
    </section>

    <section className="report-history" role="region" aria-label="Report definition history" aria-busy={historyState === "loading" || undefined}>
      <div className="section-header report-section-header"><div><h2>Decision history</h2><p>Each decision is recorded against the version and checksum the responsible role reviewed.</p></div></div>
      {historyState === "loading" && <p className="ropa-load-state" role="status">Loading report decision history…</p>}
      {historyState === "error" && <Notice tone="error"><span>{historyError || "The report decision history could not be loaded. Try again."}</span> <Button variant="secondary" size="compact" onPress={onRetryHistory}>Retry history</Button></Notice>}
      {historyState === "live" && history.length === 0 && <EmptyState population={`Decision history for ${definition.name}`} title="No decision history is recorded" description="The selected definition has no stored proposal or decision record. Review the current definition before relying on its governance state." />}
      {historyState === "live" && history.length > 0 && <ol className="report-history-list">{history.map((revision) => <HistoryRow key={`${revision.definition_id}-${revision.version}`} revision={revision} />)}</ol>}
    </section>
  </>;
}

function HistoryRow({ revision }: { revision: ReportDefinitionRevision }) {
  return <li className="report-history-row">
    <div className="report-history-row__heading"><strong>Revision {revision.version}</strong><StatusBadge tone={revision.decision === "APPROVED" ? "success" : revision.decision === "REJECTED" ? "error" : "info"}>{revisionDecisionLabel(revision.decision)}</StatusBadge></div>
    <span>Proposed by {revision.maker_id || "not recorded"} on {formatDateTime(revision.created_at)}</span>
    {revision.reviewed_by && <span>Reviewed by {revision.reviewed_by} on {revision.reviewed_at ? formatDateTime(revision.reviewed_at) : "the recorded review date is unavailable"}</span>}
    {revision.approved_by && <span>Authorised by {revision.approved_by} on {revision.approved_at ? formatDateTime(revision.approved_at) : "the recorded authorisation date is unavailable"}</span>}
    {revision.decision_note && <small>{revision.decision_note}</small>}
  </li>;
}

function RunTable({ runs, onDownload, downloading }: { runs: readonly ReportRun[]; onDownload: (run: ReportRun) => void; downloading: boolean }) {
  const columns: readonly DataColumn<ReportRun>[] = [
    { id: "status", header: "Run state", kind: "status", render: (run) => <StatusBadge tone={runStatusTone(run)}>{runStatusLabel(run)}</StatusBadge>, accessibleText: (run) => runStatusLabel(run) },
    { id: "definition", header: "Report", mobileLayout: "full-width", render: (run) => <span className="report-run-identity"><strong>{run.definition_code}</strong><small>{run.dataset === "PROCESSING_ACTIVITIES" ? "Processing activities" : datasetLabel(run.dataset)}</small></span>, accessibleText: (run) => `${run.definition_code}, ${datasetLabel(run.dataset)}` },
    { id: "as_of", header: "As of", render: (run) => <LabelledValue label="As of" value={formatDateTime(run.as_of)} time={run.as_of} />, accessibleText: (run) => `As of ${formatDateTime(run.as_of)}` },
    { id: "generated", header: "Generated", render: (run) => <LabelledValue label="Generated" value={formatDateTime(run.completed_at || run.created_at)} time={run.completed_at || run.created_at} />, accessibleText: (run) => `Generated ${formatDateTime(run.completed_at || run.created_at)}` },
    { id: "rows", header: "Rows", kind: "number", render: (run) => <span>{run.status === "FAILED" && run.failure_code === "row_limit_exceeded" ? "0 rows · no file produced" : `${formatNumber(run.row_count)} rows`}</span>, accessibleText: (run) => run.status === "FAILED" && run.failure_code === "row_limit_exceeded" ? "0 rows, no file produced" : `${run.row_count} rows` },
    { id: "source", header: "Source boundary", mobileLayout: "full-width", render: (run) => <SourceBoundaryValue run={run} />, accessibleText: (run) => sourceBoundaryAccessibleText(run) },
    { id: "stop", header: "Population and stop", render: (run) => <RunOutcome run={run} />, accessibleText: (run) => runOutcomeAccessibleText(run) },
    { id: "download", header: "File", kind: "action", render: (run) => <Button variant="secondary" size="compact" onPress={() => onDownload(run)} isDisabled={run.status !== "READY"} isLoading={downloading && run.status === "READY"}>{run.status === "READY" ? "Download report" : "No file"}</Button>, accessibleText: (run) => run.status === "READY" ? "Download report" : "No file available" },
  ];
  return <DataTable ariaLabel="Report runs" rows={runs} rowKey={(run) => run.id} rowName={(run) => `${runStatusLabel(run)}, ${run.definition_code}, ${run.row_count} rows, ${run.source_boundary.population_complete ? "complete population" : "incomplete population"}`} columns={columns} />;
}

function SourceBoundaryValue({ run }: { run: ReportRun }) {
  const highWater = Object.entries(run.source_boundary?.source_high_water ?? {});
  return <div className="report-source-boundary">
    <span className="report-source-boundary__label">{sourceProjectionVersionLabel}</span>
    <strong>{run.source_boundary?.projection_version || "Source version unavailable"}</strong>
    <span className="report-source-boundary__label">Source high-water</span>
    {highWater.length > 0 ? <ul>{highWater.map(([source, value]) => <li key={source}><span>{humanizeSource(source)}</span><time dateTime={value}>{formatDateTime(value)}</time></li>)}</ul> : <span>Source high-water unavailable</span>}
  </div>;
}

function RunOutcome({ run }: { run: ReportRun }) {
  const populationComplete = run.source_boundary?.population_complete === true;
  return <div className="report-run-outcome">
    <span>{populationComplete ? "Complete population" : "Population incomplete"}</span>
    {run.status === "READY" && <small>{formatNumber(run.source_boundary?.population ?? run.row_count)} source rows checked</small>}
    {run.status === "FAILED" && <><small className="report-run-outcome__failure">{failureReason(run)}</small><small>Narrow the filter and run it again.</small></>}
    {run.status === "QUEUED" && <small>Waiting for the report worker.</small>}
    {run.status === "RUNNING" && <small>The report worker is generating the bounded file.</small>}
  </div>;
}

function LabelledValue({ label, value, time }: { label: string; value: string; time: string }) {
  return <span className="report-labelled-value"><small>{label}</small><time dateTime={time}>{value}</time></span>;
}

function Fact({ label, value }: { label: string; value: string }) {
  return <div><dt>{label}</dt><dd>{value}</dd></div>;
}

function RoleSummary({ label, value }: { label: string; value?: string }) {
  return <span><small>{label}</small><strong>{value || "Not recorded"}</strong></span>;
}

function ReportMetric({ label, value, tone = "neutral" }: { label: string; value: number; tone?: StatusTone }) {
  return <div className="report-status-strip__metric"><span>{label}</span><strong>{formatNumber(value)}</strong><StatusBadge tone={tone}>{label}</StatusBadge></div>;
}

function countDefinitionStates(definitions: readonly ReportDefinition[]) {
  return {
    pendingReview: definitions.filter((definition) => definition.status === "PENDING_REVIEW").length,
    reviewed: definitions.filter((definition) => definition.status === "REVIEWED").length,
    active: definitions.filter((definition) => definition.status === "ACTIVE").length,
  };
}

export function runAvailability(definition: ReportDefinition): { allowed: boolean; reason: string } {
  if (definition.status === "ACTIVE" && definition.effective === false && definition.effective_from && new Date(definition.effective_from).getTime() > Date.now()) {
    return { allowed: false, reason: `This report starts on ${formatDate(definition.effective_from)}. Run it after the authorizer's effective date.` };
  }
  if (definition.status === "ACTIVE") return { allowed: true, reason: "This definition is active and effective. Running it will create a new bounded report receipt." };
  if (definition.status === "PENDING_REVIEW") return { allowed: false, reason: "A reviewer must complete the report review before an authorizer can authorise this definition." };
  if (definition.status === "REVIEWED") return { allowed: false, reason: "An authorizer must activate this report after the review before it can run." };
  if (definition.status === "DRAFT") return { allowed: false, reason: "A proposer must send this draft for review before it can run." };
  return { allowed: false, reason: "This report is retired. Create a new definition before running a current report." };
}

function definitionStatusLabel(status: ReportDefinitionStatus) {
  if (status === "DRAFT") return "Draft";
  if (status === "PENDING_REVIEW") return "Waiting for review";
  if (status === "REVIEWED") return "Reviewed; authorisation needed";
  if (status === "ACTIVE") return "Active";
  return "Retired";
}

function definitionStatusTone(status: ReportDefinitionStatus): StatusTone {
  if (status === "ACTIVE") return "success";
  if (status === "PENDING_REVIEW") return "warning";
  if (status === "REVIEWED") return "info";
  if (status === "RETIRED") return "neutral";
  return "unknown";
}

function runStatusLabel(run: ReportRun) {
  if (run.status === "READY") return "Ready";
  if (run.status === "FAILED" && run.failure_code === "row_limit_exceeded") return "Failed at row limit";
  if (run.status === "FAILED") return "Failed";
  if (run.status === "QUEUED") return "Queued";
  return "Running";
}

function runStatusTone(run: ReportRun): StatusTone {
  if (run.status === "READY") return "success";
  if (run.status === "FAILED") return "error";
  if (run.status === "RUNNING") return "info";
  return "neutral";
}

function failureReason(run: ReportRun) {
  if (run.failure_code === "row_limit_exceeded") return "Stopped at the 10,000-row ceiling; no file was produced.";
  if (run.failure_code === "byte_limit_exceeded") return "Stopped at the file-size ceiling; no file was produced.";
  if (run.failure_code === "retry_budget_exhausted") return "Stopped after the report worker's retry budget was exhausted; no file was produced.";
  if (run.failure_code === "source_boundary_mismatch") return "Stopped because the source boundary changed while the report was being generated; no file was produced.";
  return "The report worker stopped before a complete file was produced.";
}

function runOutcomeAccessibleText(run: ReportRun) {
  return `${run.source_boundary?.population_complete ? "Complete population" : "Population incomplete"}. ${failureReason(run)}`;
}

function sourceBoundaryAccessibleText(run: ReportRun) {
  const highWater = Object.entries(run.source_boundary?.source_high_water ?? {}).map(([source, value]) => `${humanizeSource(source)} ${formatDateTime(value)}`).join(", ") || "Source high-water unavailable";
  return `${sourceProjectionVersionLabel} ${run.source_boundary?.projection_version || "unavailable"}; ${highWater}`;
}

function datasetLabel(dataset: ReportDataset) {
  if (dataset === "VENDORS") return "Vendors";
  if (dataset === "PROCESSING_ACTIVITIES") return "Processing activities";
  if (dataset === "PROCESSING_ACTIVITY_EXCEPTIONS") return "Processing activities with open exceptions";
  if (dataset === "PROGRAMS") return "Programs";
  if (dataset === "MATTERS") return "Work — all issues and changes";
  return "Work — exceptions and overdue obligations";
}

function scopeDefinitionLabel(definition: ReportDefinition) {
  if (definition.scope_kind === "LEGAL_ENTITY") return "Whole legal entity";
  if (definition.scope_kind === "PROGRAM") return "One Program";
  return "One issue or change";
}

function effectiveDateLabel(definition: ReportDefinition) {
  if (!definition.effective_from) return "Not recorded";
  const date = new Date(definition.effective_from);
  if (!Number.isFinite(date.getTime())) return "Effective date unavailable";
  return `${definition.effective === false ? "Scheduled" : "Effective"} ${formatDate(definition.effective_from)}`;
}

function revisionDecisionLabel(decision: string) {
  if (decision === "APPROVED") return "Approved";
  if (decision === "REVIEWED") return "Reviewed";
  if (decision === "REJECTED") return "Rejected";
  if (decision === "RETIRED") return "Retired";
  return "Proposed";
}

const sourceProjectionVersionLabel = "Source projection " + "version";

function humanizeSource(source: string) {
  return source.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function formatNumber(value: number | undefined) {
  return Number.isFinite(value) ? (value as number).toLocaleString("en-GB") : "unknown";
}

function formatDate(value: string) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "date unavailable";
  return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", timeZone: "UTC" }).format(date);
}

function formatDateTime(value: string) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "time unavailable";
  return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", timeZone: "UTC" }).format(date);
}

function actionMessage(action: ReportDefinitionAction) {
  if (action === "submit") return "Report definition sent for review.";
  if (action === "review") return "Report review recorded. An authorizer must now activate the definition.";
  if (action === "activate") return "Report definition activated. It can now be run against the selected source boundary.";
  if (action === "reject") return "Report definition rejected. Review the decision note before creating a new proposal.";
  return "Report definition retired. Its history remains available for reconstruction.";
}

function readError(error: unknown, fallback: string) {
  if (typeof error === "object" && error !== null && "message" in error && typeof error.message === "string" && error.message.trim()) return error.message;
  return fallback;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}

function saveBlob(blob: Blob, filename: string) {
  if (typeof URL === "undefined" || typeof URL.createObjectURL !== "function" || typeof document === "undefined") return;
  const href = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = href;
  anchor.download = filename;
  anchor.rel = "noopener";
  anchor.click();
  URL.revokeObjectURL(href);
}
