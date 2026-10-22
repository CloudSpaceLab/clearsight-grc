import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { loadHomeMetrics as loadHomeMetricsRequest } from "../../metricApi";
import { ReportsOverview } from "./ReportsOverview";
import { ReportsLibrary } from "./ReportsLibrary";
import {
  formatDateTime, humanizeCode, isExpired, isReportAvailable, reportBelongsToArea,
  reportDatasetLabel, runStatusLabel, runStatusTone, type ReportArea,
} from "./reportRunPresentation";
import type { ReportingPeriodQuery } from "../../reportingPeriod";
import {
  createReportRun as createReportRunRequest,
  downloadReportRun as downloadReportRunRequest,
  listReportDefinitions as listReportDefinitionsRequest,
  listReportOwnerOptions as listReportOwnerOptionsRequest,
  listReportRunPage as listReportRunPageRequest,
} from "../../reportingApi";
import type { ReportDefinition, ReportOwnerOption, ReportRun } from "../../reportingTypes";
import { ReportingPage, type ReportingPageProps } from "../ReportingPage";
import { reportSetupArea, reportSetupFocus, reportSetupFocusLabel } from "./reportTemplatePresets";
import {
  Button,
  EmptyState,
  FocusedSheet,
  Notice,
  SelectField,
  StatusBadge,
  Tabs,
  TextField,
  type DataColumn,
} from "../ui";
import "./reports.css";

const ReportsExplore = lazy(() => import("./ReportsExplore").then(({ ReportsExplore }) => ({ default: ReportsExplore })));

type WorkspaceTab = "overview" | "explore" | "library" | "schedules" | "builder";
type LoadState = "loading" | "live" | "error";

export type ReportsWorkspaceProps = {
  organizationName?: string;
  legalEntityName?: string;
  legalEntityID?: string;
  organizationScopeID?: string;
  scopeAuthorized?: boolean;
  loadMetricBasis?: typeof loadHomeMetricsRequest;
  onOpenRisk?: (riskID: string) => void;
  onOpenLoss?: (lossID: string) => void;
  onOpenWork?: () => void;
  onOpenPrograms?: () => void;
  loadDefinitions?: typeof listReportDefinitionsRequest;
  createDefinition?: ReportingPageProps["createDefinition"];
  transitionDefinition?: ReportingPageProps["transitionDefinition"];
  loadMatters?: ReportingPageProps["loadMatters"];
  loadRunPage?: typeof listReportRunPageRequest;
  loadOwners?: typeof listReportOwnerOptionsRequest;
  createRun?: typeof createReportRunRequest;
  downloadRun?: typeof downloadReportRunRequest;
  onOpenInsights?: (period?: ReportingPeriodQuery) => void;
};

const generationAreaOptions: readonly { id: Exclude<ReportArea, "ALL">; label: string }[] = [
  { id: "VENDORS", label: "Vendors" },
  { id: "PROGRAMS", label: "Programs" },
  { id: "WORK", label: "Work" },
  { id: "PROCESSING", label: "Processing activities" },
];

export function ReportsWorkspace({
  organizationName,
  legalEntityName,
  legalEntityID,
  organizationScopeID,
  scopeAuthorized = true,
  loadMetricBasis = loadHomeMetricsRequest,
  onOpenRisk,
  onOpenLoss,
  onOpenWork,
  onOpenPrograms,
  loadDefinitions = listReportDefinitionsRequest,
  createDefinition,
  transitionDefinition,
  loadMatters,
  loadRunPage = listReportRunPageRequest,
  loadOwners = listReportOwnerOptionsRequest,
  createRun = createReportRunRequest,
  downloadRun = downloadReportRunRequest,
  onOpenInsights,
}: ReportsWorkspaceProps) {
  const [tab, setTab] = useState<WorkspaceTab>("overview");
  const [metricPeriod, setMetricPeriod] = useState<ReportingPeriodQuery>();
  const [metricPeriodState, setMetricPeriodState] = useState<"idle" | "loading" | "live" | "unavailable">("idle");
  const [definitions, setDefinitions] = useState<ReportDefinition[]>([]);
  const [runs, setRuns] = useState<ReportRun[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [loadingOlder, setLoadingOlder] = useState(false);
  const [runState, setRunState] = useState<LoadState>("loading");
  const [runError, setRunError] = useState<string>();
  const [definitionState, setDefinitionState] = useState<LoadState>("loading");
  const [definitionError, setDefinitionError] = useState<string>();
  const [refreshKey, setRefreshKey] = useState(0);
  const [query, setQuery] = useState("");
  const [area, setArea] = useState<ReportArea>("ALL");
  const [generateOpen, setGenerateOpen] = useState(false);
  const [generationArea, setGenerationArea] = useState<Exclude<ReportArea, "ALL">>("VENDORS");
  const [generationDefinitionID, setGenerationDefinitionID] = useState<string>();
  const [generationStartDate, setGenerationStartDate] = useState("");
  const [generationEndDate, setGenerationEndDate] = useState("");
  const [generationOwnerID, setGenerationOwnerID] = useState<string>();
  const [ownerOptions, setOwnerOptions] = useState<ReportOwnerOption[]>([]);
  const [ownersLoading, setOwnersLoading] = useState(false);
  const [ownersError, setOwnersError] = useState<string>();
  const [selectedRun, setSelectedRun] = useState<ReportRun>();
  const [command, setCommand] = useState<"idle" | "generating" | "downloading">("idle");
  const commandPending = useRef(false);
  const definitionMutation = useRef(0);
  const runMutation = useRef(0);
  const [commandMessage, setCommandMessage] = useState<string>();
  const [commandError, setCommandError] = useState<string>();

  useEffect(() => {
    const controller = new AbortController();
    const mutationAtRead = definitionMutation.current;
    const runMutationAtRead = runMutation.current;
    setRunState("loading");
    setDefinitionState("loading");
    setRunError(undefined);
    setDefinitionError(undefined);
    void Promise.allSettled([
      loadDefinitions(true, controller.signal),
      loadRunPage({ limit: 50 }, controller.signal),
    ]).then(([definitionResult, runResult]) => {
      if (controller.signal.aborted) return;
      if (definitionResult.status === "fulfilled") {
        setDefinitions((current) => mutationAtRead === definitionMutation.current
          ? definitionResult.value
          : mergeDefinitions(definitionResult.value, current));
        setDefinitionState("live");
      } else if (!isAbortError(definitionResult.reason)) {
        setDefinitionState("error");
        setDefinitionError("Saved setups unavailable. Reports remain available.");
      }
      if (runResult.status === "fulfilled") {
        setRuns((current) => runMutationAtRead === runMutation.current
          ? runResult.value.items
          : mergeReportRuns(current, runResult.value.items));
        setNextCursor(runResult.value.next_cursor);
        setRunState("live");
      } else if (!isAbortError(runResult.reason)) {
        setRunState("error");
        setRunError("Reports failed to load.");
      }
    });
    return () => controller.abort();
  }, [loadDefinitions, loadRunPage, refreshKey]);

  // Reuse the governed Home reporting period for loss flows, and never
  // combine metric data from a different verified entity or group.
  useEffect(() => {
    let current = true;
    setMetricPeriod(undefined);
    if (!scopeAuthorized) {
      setMetricPeriodState("idle");
      return;
    }
    setMetricPeriodState("loading");
    void loadMetricBasis(undefined, organizationScopeID).then((bundle) => {
      if (!current) return;
      const scopeMatches = organizationScopeID
        ? bundle.scope_kind === "ORGANIZATION_SCOPE" && bundle.scope_id === organizationScopeID
        : bundle.scope_kind === "LEGAL_ENTITY" && (!legalEntityID || bundle.scope_id === legalEntityID);
      const { start_date, end_date } = bundle.reporting_period;
      const validPeriod = /^\d{4}-\d{2}-\d{2}$/.test(start_date)
        && /^\d{4}-\d{2}-\d{2}$/.test(end_date) && start_date <= end_date;
      if (!scopeMatches || !validPeriod) {
        setMetricPeriodState("unavailable");
        return;
      }
      setMetricPeriod({ start_date, end_date });
      setMetricPeriodState("live");
    }).catch(() => {
      if (current) setMetricPeriodState("unavailable");
    });
    return () => { current = false; };
  }, [scopeAuthorized, legalEntityID, organizationScopeID, loadMetricBasis, refreshKey]);

  const definitionsByID = useMemo(
    () => new Map(definitions.map((definition) => [definition.id, definition])),
    [definitions],
  );

  const activeGenerationDefinitions = useMemo(
    () => definitions.filter((definition) =>
      reportBelongsToArea(definition.dataset, generationArea) &&
      definition.status === "ACTIVE" &&
      definition.effective
    ),
    [definitions, generationArea],
  );

  useEffect(() => {
    if (generationDefinitionID && !activeGenerationDefinitions.some((definition) => definition.id === generationDefinitionID)) {
      setGenerationDefinitionID(undefined);
      if (generateOpen) setCommandError("The selected setup is no longer ready. Choose an approved setup before generating.");
    }
  }, [activeGenerationDefinitions, generationDefinitionID, generateOpen]);

  const generationDefinition = useMemo(
    () => activeGenerationDefinitions.find((definition) => definition.id === generationDefinitionID),
    [activeGenerationDefinitions, generationDefinitionID],
  );

  useEffect(() => {
    setGenerationOwnerID(undefined);
    setOwnerOptions([]);
    setOwnersError(undefined);
    if (!generateOpen || !generationDefinition || generationDefinition.dataset === "MATTER_BOARD_BRIEF") {
      setOwnersLoading(false);
      return;
    }
    const controller = new AbortController();
    setOwnersLoading(true);
    void loadOwners(generationDefinition.dataset, controller.signal).then((values) => {
      if (!controller.signal.aborted) setOwnerOptions(values);
    }).catch((reason: unknown) => {
      if (!controller.signal.aborted && !isAbortError(reason)) {
        setOwnersError(readError(reason, "Assigned owners unavailable."));
      }
    }).finally(() => {
      if (!controller.signal.aborted) setOwnersLoading(false);
    });
    return () => controller.abort();
  }, [generateOpen, generationDefinition, loadOwners]);

  const isBoardBrief = generationDefinition?.dataset === "MATTER_BOARD_BRIEF";
  const dateRangeInvalid = !isBoardBrief && Boolean(generationStartDate && generationEndDate && generationStartDate > generationEndDate);

  const runningCount = runs.filter((run) => run.status === "QUEUED" || run.status === "RUNNING").length;

  useEffect(() => {
    if (tab !== "library" || runState !== "live" || runningCount === 0) return;
    const controller = new AbortController();
    const refreshTimer = window.setTimeout(() => {
      void loadRunPage({ limit: 50 }, controller.signal).then((page) => {
        if (controller.signal.aborted) return;
        setRuns((current) => mergeReportRuns(current, page.items));
      }).catch((reason: unknown) => {
        if (controller.signal.aborted || isAbortError(reason)) return;
        // Background status refresh is best-effort. Manual Refresh remains the
        // explicit recovery path and should own any operator-facing error.
      });
    }, 5000);
    return () => {
      window.clearTimeout(refreshTimer);
      controller.abort();
    };
  }, [loadRunPage, runState, runningCount, tab]);

  function refresh() {
    setRefreshKey((value) => value + 1);
  }

  async function loadOlderReports() {
    if (!nextCursor || loadingOlder) return;
    setLoadingOlder(true);
    setCommandError(undefined);
    try {
      const page = await loadRunPage({ limit: 50, cursor: nextCursor });
      setRuns((current) => mergeReportRuns(current, page.items));
      setNextCursor(page.next_cursor);
    } catch (reason: unknown) {
      setCommandError(readError(reason, "Couldn’t load older reports."));
    } finally {
      setLoadingOlder(false);
    }
  }

  function updateDefinition(definition: ReportDefinition) {
    definitionMutation.current += 1;
    setDefinitions((current) => mergeDefinitions(current, [definition]));
    setDefinitionState("live");
    setDefinitionError(undefined);
  }

  function openGenerate(definition?: ReportDefinition) {
    if (commandPending.current) return;
    if (definitionState !== "live") {
      setCommandError("Refresh setups before generating a report.");
      return;
    }
    if (definition) {
      const current = definitionsByID.get(definition.id);
      if (!current || current.status !== "ACTIVE" || !current.effective
        || current.version !== definition.version || current.current_version !== definition.current_version
        || current.checksum !== definition.checksum) {
        setCommandError("This setup has changed or is no longer ready. Refresh setups and review its current status.");
        return;
      }
      const setupArea = reportSetupArea(definition);
      setGenerationArea(setupArea === "BOARD_BRIEF" ? "WORK" : setupArea);
      setGenerationDefinitionID(definition.id);
    } else {
      setGenerationDefinitionID(generationDefinition?.id ?? activeGenerationDefinitions[0]?.id);
    }
    setGenerationStartDate("");
    setGenerationEndDate("");
    setGenerationOwnerID(undefined);
    setCommandError(undefined);
    setGenerateOpen(true);
  }

  function openTemplates() {
    setGenerateOpen(false);
    setTab("builder");
  }

  async function generateReport() {
    const definition = generationDefinition;
    if (!definition || definitionState !== "live" || dateRangeInvalid || commandPending.current) return;
    commandPending.current = true;
    setCommand("generating");
    setCommandError(undefined);
    setCommandMessage(undefined);
    try {
      const run = await createRun(definition.id, definition.current_version, definition.dataset === "MATTER_BOARD_BRIEF" ? {} : {
        ...(generationStartDate ? { start_date: generationStartDate } : {}),
        ...(generationEndDate ? { end_date: generationEndDate } : {}),
        ...(generationOwnerID ? { owner_principal_id: generationOwnerID } : {}),
      });
      runMutation.current += 1;
      setRuns((current) => [run, ...current.filter((item) => item.id !== run.id)]);
      setGenerateOpen(false);
      setGenerationStartDate("");
      setGenerationEndDate("");
      setGenerationOwnerID(undefined);
      setTab("library");
      setCommandMessage(`${definition.name} queued.`);
    } catch (reason: unknown) {
      setCommandError(readError(reason, "Couldn’t generate report. Retry."));
    } finally {
      commandPending.current = false;
      setCommand("idle");
    }
  }

  async function download(run: ReportRun) {
    if (!isReportAvailable(run) || commandPending.current) return;
    commandPending.current = true;
    setCommand("downloading");
    setCommandError(undefined);
    try {
      const result = await downloadRun(run.id);
      saveBlob(result.blob, result.filename ?? reportFilename(run));
      setCommandMessage("Report downloaded.");
    } catch (reason: unknown) {
      setCommandError(readError(reason, "Download failed. Refresh and retry."));
    } finally {
      commandPending.current = false;
      setCommand("idle");
    }
  }

  const columns: DataColumn<ReportRun>[] = [
    {
      id: "report",
      header: "Report",
      render: (run) => {
        const definition = definitionsByID.get(run.definition_id);
        return <span className="reports-library__name"><strong>{definition?.name || humanizeCode(run.definition_code)}</strong></span>;
      },
      accessibleText: (run) => definitionsByID.get(run.definition_id)?.name || humanizeCode(run.definition_code),
    },
    {
      id: "status",
      header: "Status",
      kind: "status",
      render: (run) => <StatusBadge tone={runStatusTone(run)}>{runStatusLabel(run)}</StatusBadge>,
      accessibleText: runStatusLabel,
    },
    {
      id: "generated",
      header: "Generated",
      render: (run) => <span className="reports-library__date">{formatDateTime(run.completed_at || run.created_at)}{run.status === "READY" && <small>As of {formatDateTime(run.as_of)}</small>}</span>,
      accessibleText: (run) => formatDateTime(run.completed_at || run.created_at),
    },
    {
      id: "file",
      header: "File",
      kind: "action",
      render: (run) => isReportAvailable(run)
        ? <Button variant="secondary" size="compact" isLoading={command === "downloading"} onPress={() => void download(run)}>Download</Button>
        : <span className="reports-library__unavailable">{run.status === "READY" ? runStatusLabel(run) : "Not ready"}</span>,
      accessibleText: (run) => isReportAvailable(run) ? "Download available" : run.status === "READY" ? runStatusLabel(run) : "Not ready",
    },
  ];

  return <section className="reports-workspace" aria-labelledby="reports-heading">
    <header className="reports-workspace__header">
      <div>
        <span className="eyebrow">Reporting · {legalEntityName || organizationName || "Current legal entity"}</span>
        <h1 id="reports-heading">Reports &amp; insights</h1>
      </div>
      <div className="reports-workspace__actions">
        <Button variant="secondary" onPress={refresh} isLoading={runState === "loading" || definitionState === "loading"}>Refresh</Button>
        {tab !== "builder" && <Button variant="secondary" isDisabled={definitionState !== "live"} onPress={() => openGenerate()}>Generate report</Button>}
        <Button variant="primary" onPress={openTemplates}>New report</Button>
      </div>
    </header>
    {tab === "builder" && !generateOpen && commandError && <Notice tone="error">{commandError}</Notice>}

    <Tabs
      ariaLabel="Reports workspace"
      retainVisitedPanels
      items={[{ id: "overview", label: "Overview" }, { id: "explore", label: "Explore" }, { id: "library", label: "Reports" }, { id: "schedules", label: "Schedules" }, { id: "builder", label: "Build report", isContextual: true }]}
      selectedKey={tab}
      onSelectionChange={setTab}
    >
      {(activeTab) => activeTab === "overview"
        ? <ReportsOverview
            legalEntityName={legalEntityName}
            metricPeriod={metricPeriod}
            metricState={metricPeriodState}
            refreshKey={refreshKey}
            scopeAuthorized={scopeAuthorized}
            organizationScopeID={organizationScopeID}
            legalEntityID={legalEntityID}
            runState={runState}
            runs={runs}
            nextCursor={nextCursor}
            definitionsByID={definitionsByID}
            definitionState={definitionState}
            definitionCount={definitions.length}
            columns={columns}
            onSelectRun={setSelectedRun}
            onViewReports={() => setTab("library")}
            onOpenExplore={() => setTab("explore")}
            onOpenBuilder={() => setTab("builder")}
            onOpenWork={onOpenWork}
            onOpenPrograms={onOpenPrograms}
          />
        : activeTab === "explore"
        ? <Suspense fallback={<p role="status">Loading risk and loss analysis…</p>}>
            <ReportsExplore period={metricPeriod} periodState={metricPeriodState} organizationScopeID={organizationScopeID}
              legalEntityID={legalEntityID} scopeAuthorized={scopeAuthorized} refreshKey={refreshKey}
              onOpenRisk={onOpenRisk} onOpenLoss={onOpenLoss} onOpenInsights={onOpenInsights ? () => onOpenInsights(metricPeriod) : undefined}/>
          </Suspense>
        : activeTab === "schedules"
        ? <section className="reports-v2-unavailable" aria-label="Report schedules">
            <h2>Schedules</h2>
            <Notice tone="warning">Recurring report delivery is unavailable. Generated reports remain accessible in Reports.</Notice>
            <Button variant="secondary" onPress={() => setTab("library")}>View reports</Button>
          </section>
        : activeTab === "library"
        ? <div className="reports-library">
          <p className="reports-library__scope">Reports for {legalEntityName || "the current legal entity"}</p>
          {commandMessage && <Notice tone="success">{commandMessage}</Notice>}
          {commandError && <Notice tone="error">{commandError}</Notice>}
          {definitionState === "error" && <Notice tone="warning">{definitionError}</Notice>}
          <ReportsLibrary runs={runs} definitionsByID={definitionsByID} state={runState} error={runError}
            query={query} onQueryChange={setQuery} area={area} onAreaChange={setArea}
            hasMore={Boolean(nextCursor)} loadingMore={loadingOlder} downloading={command === "downloading"}
            scopeLabel={legalEntityName} onLoadMore={() => void loadOlderReports()} onRetry={refresh}
            onInspect={setSelectedRun} onDownload={(run) => void download(run)}/>
        </div>
        : <ReportingPage
            key={legalEntityID || legalEntityName || "current-entity"}
            embedded
            organizationName={organizationName}
            legalEntityName={legalEntityName}
            definitionSource={{ items: definitions, state: definitionState, refresh, onChange: updateDefinition }}
            createDefinition={createDefinition}
            transitionDefinition={transitionDefinition}
            loadMatters={loadMatters}
            onGenerate={openGenerate}
            onBack={() => setTab("library")}
          />
      }
    </Tabs>

    {generateOpen && <FocusedSheet label="Generate report" onClose={() => setGenerateOpen(false)}
      isDismissable={command !== "generating"} closeLabel={command === "generating" ? "Generating report…" : "Close"}>
      <div className="reports-generate">
        <div className="reports-generate__heading">
          <h2>Generate report</h2>
          <p>Choose an approved setup{isBoardBrief ? " for one issue." : ". Date filters apply to when records were created."}</p>
        </div>
        <SelectField
          label="Area"
          value={generationArea}
          placeholder="Choose an area"
          options={generationAreaOptions}
          allowsEmpty={false}
          isDisabled={command === "generating"}
          onChange={(value) => {
            const nextArea = (value || "VENDORS") as Exclude<ReportArea, "ALL">;
            setGenerationArea(nextArea);
            setGenerationDefinitionID(definitions.find((definition) => reportBelongsToArea(definition.dataset, nextArea) && definition.status === "ACTIVE" && definition.effective)?.id);
            setCommandError(undefined);
          }}
        />
        {activeGenerationDefinitions.length > 0 ? <>
          <SelectField
            label="Saved setup"
            value={generationDefinitionID}
            placeholder="Choose a saved setup"
            options={activeGenerationDefinitions.map((definition) => ({ id: definition.id, label: definition.name }))}
            allowsEmpty={false}
            isDisabled={command === "generating"}
            onChange={setGenerationDefinitionID}
          />
          {generationDefinitionID && <ReportTemplateSummary definition={generationDefinition} />}
          {!isBoardBrief && <div className="reports-generate__filters">
            <TextField
              label="Start date"
              type="date"
              value={generationStartDate}
              max={generationEndDate || undefined}
              onChange={setGenerationStartDate}
              isInvalid={dateRangeInvalid}
              isDisabled={command === "generating"}
            />
            <TextField
              label="End date"
              type="date"
              value={generationEndDate}
              min={generationStartDate || undefined}
              onChange={setGenerationEndDate}
              isInvalid={dateRangeInvalid}
              isDisabled={command === "generating"}
              errorMessage={dateRangeInvalid ? "End date must be on or after start date." : undefined}
            />
            <SelectField
              label="Assigned owner"
              value={generationOwnerID}
              placeholder={ownersLoading ? "Loading assigned owners…" : "All assigned owners"}
              options={ownerOptions.map((owner) => ({ id: owner.principal_id, label: owner.display_name }))}
              isDisabled={ownersLoading || Boolean(ownersError) || command === "generating"}
              onChange={setGenerationOwnerID}
            />
          </div>}
          {isBoardBrief && <p>The board brief includes only its saved issue. Date and owner filters do not apply.</p>}
          {!isBoardBrief && ownersError && <Notice tone="warning"><span>{ownersError} Generate without an owner filter or retry.</span></Notice>}
          {commandError && <Notice tone="error"><span>{commandError}</span></Notice>}
          <div className="reports-generate__footer">
            <Button variant="secondary" isDisabled={command === "generating"} onPress={() => setGenerateOpen(false)}>Cancel</Button>
            <Button variant="primary" isDisabled={!generationDefinition || definitionState !== "live" || dateRangeInvalid} isLoading={command === "generating"} onPress={() => void generateReport()}>Generate report</Button>
          </div>
        </> : <div className="reports-generate__empty">
          <EmptyState
            population={reportDatasetLabel(generationArea)}
            title={`No ready ${reportDatasetLabel(generationArea).toLowerCase()} setup`}
            description="Create and approve a setup first."
          />
          <Button variant="primary" onPress={openTemplates}>Create saved setup</Button>
        </div>}
      </div>
    </FocusedSheet>}

    {selectedRun && <FocusedSheet label="Report details" onClose={() => setSelectedRun(undefined)}>
      <ReportRunDetails
        run={selectedRun}
        definition={definitionsByID.get(selectedRun.definition_id)}
        downloading={command === "downloading"}
        onDownload={() => void download(selectedRun)}
      />
    </FocusedSheet>}
  </section>;
}

function ReportTemplateSummary({ definition }: { definition?: ReportDefinition }) {
  if (!definition) return null;
  return <dl className="reports-generate__summary">
    <div><dt>Area</dt><dd>{reportDatasetLabel(definition.dataset)}</dd></div>
    <div><dt>Shows</dt><dd>{definition.dataset === "MATTER_BOARD_BRIEF" ? "Issue brief" : reportSetupFocusLabel(reportSetupFocus(definition))}</dd></div>
    {definition.scope_kind === "MATTER" && <div><dt>Issue</dt><dd>Issue name unavailable. Saved reference: {definition.scope_ref || "Not recorded"}</dd></div>}
  </dl>;
}

function ReportRunDetails({ run, definition, downloading, onDownload }: { run: ReportRun; definition?: ReportDefinition; downloading: boolean; onDownload: () => void }) {
  const available = isReportAvailable(run);
  return <div className="report-run-details">
    <div>
      <span className="eyebrow">{reportDatasetLabel(run.dataset)}</span>
      <h2>{definition?.name || humanizeCode(run.definition_code)}</h2>
    </div>
    <StatusBadge tone={runStatusTone(run)}>{runStatusLabel(run)}</StatusBadge>
    <dl>
      <div><dt>Requested</dt><dd>{formatDateTime(run.created_at)}</dd></div>
      <div><dt>Generated</dt><dd>{run.completed_at ? formatDateTime(run.completed_at) : "Not complete"}</dd></div>
      <div><dt>Data as of</dt><dd>{formatDateTime(run.as_of)}</dd></div>
      <div><dt>Rows</dt><dd>{run.status === "READY" ? run.row_count.toLocaleString() : "—"}</dd></div>
      <div><dt>Format</dt><dd>{run.format}</dd></div>
      <div><dt>Expires</dt><dd>{formatDateTime(run.expires_at)}</dd></div>
    </dl>
    {run.status === "READY" && <section className="report-run-details__source" aria-label="Source snapshot">
      <div className="report-run-details__source-heading">
        <h3>Source snapshot</h3>
        <StatusBadge tone={run.source_boundary?.population_complete === true ? "neutral" : "warning"}>
          {run.source_boundary?.population_complete === true ? "Population complete"
            : run.source_boundary?.population_complete === false ? "Population incomplete" : "Coverage unknown"}
        </StatusBadge>
      </div>
      <dl>
        <div><dt>Source captured</dt><dd>{run.source_boundary?.captured_at ? formatDateTime(run.source_boundary.captured_at) : "Unavailable"}</dd></div>
        <div><dt>Source records</dt><dd>{run.source_boundary?.population?.toLocaleString() ?? "Unknown"}</dd></div>
        <div><dt>Definition revision</dt><dd>{run.definition_version}</dd></div>
      </dl>
      <details>
        <summary>Run identifiers and digests</summary>
        <dl>
          <div><dt>Run ID</dt><dd><code>{run.id}</code></dd></div>
          <div><dt>Recorded file SHA-256</dt><dd><code>{run.data_sha256 || "Unavailable"}</code></dd></div>
          <div><dt>Recorded manifest SHA-256</dt><dd><code>{run.manifest_sha256 || "Unavailable"}</code></dd></div>
        </dl>
      </details>
    </section>}
    {run.failure_code && <Notice tone="error"><span>Generation stopped: {humanizeCode(run.failure_code)}</span></Notice>}
    <div className="report-run-details__footer">
      <Button variant="secondary" isDisabled={!available} isLoading={downloading} onPress={onDownload}>
        {run.status === "READY" && isExpired(run) ? "File expired" : "Download report"}
      </Button>
    </div>
  </div>;
}

function mergeDefinitions(current: ReportDefinition[], incoming: ReportDefinition[]) {
  const byID = new Map(current.map((definition) => [definition.id, definition]));
  for (const definition of incoming) {
    if ((byID.get(definition.id)?.version ?? -1) <= definition.version) byID.set(definition.id, definition);
  }
  return [...byID.values()];
}

function mergeReportRuns(current: ReportRun[], incoming: ReportRun[]) {
  const byID = new Map(current.map((run) => [run.id, run]));
  for (const run of incoming) byID.set(run.id, run);
  return [...byID.values()].sort((left, right) => {
    const dateDifference = Date.parse(right.created_at) - Date.parse(left.created_at);
    return dateDifference || right.id.localeCompare(left.id);
  });
}

function reportFilename(run: ReportRun) {
  const extension = run.format === "NDJSON" ? "ndjson" : run.format.toLowerCase();
  return `${run.definition_code.toLowerCase()}.${extension}`;
}

function saveBlob(blob: Blob, filename: string) {
  const href = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = href;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(href);
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}

function readError(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback;
}
