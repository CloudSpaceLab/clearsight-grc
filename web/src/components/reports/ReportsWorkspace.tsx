import { useEffect, useMemo, useState } from "react";
import {
  createReportRun as createReportRunRequest,
  downloadReportRun as downloadReportRunRequest,
  listReportDefinitions as listReportDefinitionsRequest,
  listReportRunPage as listReportRunPageRequest,
} from "../../reportingApi";
import type { ReportDataset, ReportDefinition, ReportRun } from "../../reportingTypes";
import { ReportingPage } from "../ReportingPage";
import { reportSetupFocus, reportSetupFocusLabel } from "./reportTemplatePresets";
import {
  Button,
  DataTable,
  EmptyState,
  FocusedSheet,
  Notice,
  SearchField,
  SelectField,
  StatusBadge,
  Tabs,
  type DataColumn,
  type StatusTone,
} from "../ui";
import "./reports.css";

type WorkspaceTab = "library" | "templates";
type ReportArea = "ALL" | "VENDORS" | "PROGRAMS" | "WORK";
type LoadState = "loading" | "live" | "error";

export type ReportsWorkspaceProps = {
  organizationName?: string;
  legalEntityName?: string;
  loadDefinitions?: typeof listReportDefinitionsRequest;
  loadRunPage?: typeof listReportRunPageRequest;
  createRun?: typeof createReportRunRequest;
  downloadRun?: typeof downloadReportRunRequest;
};

const areaOptions = [
  { id: "ALL", label: "All reports" },
  { id: "VENDORS", label: "Vendors" },
  { id: "PROGRAMS", label: "Programs" },
  { id: "WORK", label: "Work" },
] as const;

const generationAreaOptions: readonly { id: Exclude<ReportArea, "ALL">; label: string }[] = [
  { id: "VENDORS", label: "Vendors" },
  { id: "PROGRAMS", label: "Programs" },
  { id: "WORK", label: "Work" },
];

export function ReportsWorkspace({
  organizationName,
  legalEntityName,
  loadDefinitions = listReportDefinitionsRequest,
  loadRunPage = listReportRunPageRequest,
  createRun = createReportRunRequest,
  downloadRun = downloadReportRunRequest,
}: ReportsWorkspaceProps) {
  const [tab, setTab] = useState<WorkspaceTab>("library");
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
  const [selectedRun, setSelectedRun] = useState<ReportRun>();
  const [command, setCommand] = useState<"idle" | "generating" | "downloading">("idle");
  const [commandMessage, setCommandMessage] = useState<string>();
  const [commandError, setCommandError] = useState<string>();

  useEffect(() => {
    const controller = new AbortController();
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
        setDefinitions(definitionResult.value);
        setDefinitionState("live");
      } else if (!isAbortError(definitionResult.reason)) {
        setDefinitionState("error");
        setDefinitionError("Saved setups unavailable. Existing reports remain available.");
      }
      if (runResult.status === "fulfilled") {
        setRuns(runResult.value.items);
        setNextCursor(runResult.value.next_cursor);
        setRunState("live");
      } else if (!isAbortError(runResult.reason)) {
        setRunState("error");
        setRunError("Reports failed to load.");
      }
    });
    return () => controller.abort();
  }, [loadDefinitions, loadRunPage, refreshKey]);

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
    setGenerationDefinitionID((current) =>
      activeGenerationDefinitions.some((definition) => definition.id === current)
        ? current
        : activeGenerationDefinitions[0]?.id
    );
  }, [activeGenerationDefinitions]);

  const filteredRuns = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    return [...runs]
      .sort((left, right) => Date.parse(right.created_at) - Date.parse(left.created_at))
      .filter((run) => area === "ALL" || reportBelongsToArea(run.dataset, area))
      .filter((run) => {
        if (!normalized) return true;
        const definition = definitionsByID.get(run.definition_id);
        return [
          definition?.name,
          run.definition_code,
          reportDatasetLabel(run.dataset),
          run.status,
          run.format,
        ].some((value) => value?.toLowerCase().includes(normalized));
      });
  }, [area, definitionsByID, query, runs]);

  const readyCount = runs.filter((run) => run.status === "READY" && !isExpired(run)).length;
  const runningCount = runs.filter((run) => run.status === "QUEUED" || run.status === "RUNNING").length;
  const failedCount = runs.filter((run) => run.status === "FAILED").length;

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

  function openTemplates() {
    setGenerateOpen(false);
    setTab("templates");
  }

  async function generateReport() {
    const definition = activeGenerationDefinitions.find((item) => item.id === generationDefinitionID);
    if (!definition) return;
    setCommand("generating");
    setCommandError(undefined);
    setCommandMessage(undefined);
    try {
      const run = await createRun(definition.id, definition.current_version);
      setRuns((current) => [run, ...current.filter((item) => item.id !== run.id)]);
      setGenerateOpen(false);
      setTab("library");
      setCommandMessage(`${definition.name} queued.`);
    } catch (reason: unknown) {
      setCommandError(readError(reason, "Couldn’t generate report. Retry."));
    } finally {
      setCommand("idle");
    }
  }

  async function download(run: ReportRun) {
    if (run.status !== "READY" || isExpired(run)) return;
    setCommand("downloading");
    setCommandError(undefined);
    try {
      const result = await downloadRun(run.id);
      saveBlob(result.blob, result.filename ?? reportFilename(run));
      setCommandMessage("Report downloaded.");
    } catch (reason: unknown) {
      setCommandError(readError(reason, "Download failed. Refresh and retry."));
    } finally {
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
      id: "area",
      header: "Area",
      render: (run) => reportDatasetLabel(run.dataset),
      accessibleText: (run) => reportDatasetLabel(run.dataset),
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
      id: "rows",
      header: "Rows",
      kind: "number",
      render: (run) => run.status === "READY" ? run.row_count.toLocaleString() : "—",
      accessibleText: (run) => run.status === "READY" ? String(run.row_count) : "Not available",
    },
    {
      id: "format",
      header: "Format",
      render: (run) => run.format,
      accessibleText: (run) => run.format,
    },
    {
      id: "file",
      header: "File",
      kind: "action",
      render: (run) => run.status === "READY" && !isExpired(run)
        ? <Button variant="secondary" size="compact" isLoading={command === "downloading"} onPress={() => void download(run)}>Download</Button>
        : <span className="reports-library__unavailable">{isExpired(run) ? "Expired" : "Not ready"}</span>,
      accessibleText: (run) => run.status === "READY" && !isExpired(run) ? "Download available" : isExpired(run) ? "Expired" : "Not ready",
    },
  ];

  return <section className="reports-workspace" aria-labelledby="reports-heading">
    <header className="reports-workspace__header">
      <div>
        <span className="eyebrow">{organizationName || "ClearSight"} · {legalEntityName || "Current legal entity"}</span>
        <h1 id="reports-heading">Reports</h1>
      </div>
      {tab === "library" && <div className="reports-workspace__actions">
        <Button variant="secondary" onPress={refresh} isLoading={runState === "loading" || definitionState === "loading"}>Refresh</Button>
        <Button variant="primary" isDisabled={definitionState === "error"} onPress={() => setGenerateOpen(true)}>Generate report</Button>
      </div>}
    </header>

    <Tabs
      ariaLabel="Reports workspace"
      compactLabel="Reports view"
      items={[{ id: "library", label: "Generated reports" }, { id: "templates", label: "Saved setups" }]}
      selectedKey={tab}
      onSelectionChange={setTab}
    >
      {(activeTab) => activeTab === "library"
        ? <div className="reports-library">
          <section className="reports-summary" aria-label="Report library summary">
            <ReportSummaryMetric label="Available" value={readyCount} />
            <ReportSummaryMetric label="In progress" value={runningCount} tone={runningCount ? "info" : "neutral"} />
            <ReportSummaryMetric label="Failed" value={failedCount} tone={failedCount ? "error" : "neutral"} />
          </section>

          {commandMessage && <Notice tone="success"><span>{commandMessage}</span></Notice>}
          {commandError && <Notice tone="error"><span>{commandError}</span></Notice>}
          {definitionState === "error" && <Notice tone="warning"><span>{definitionError || "Saved report setups are temporarily unavailable. Existing generated reports remain accessible."}</span></Notice>}

          <div className="reports-library__toolbar">
            <SearchField label="Search generated reports" value={query} onChange={setQuery} placeholder="Search reports" isLoading={runState === "loading"} />
            <SelectField label="Area" value={area} placeholder="All reports" options={areaOptions} allowsEmpty={false} onChange={(value) => setArea((value || "ALL") as ReportArea)} />
          </div>

          {runState === "error" && <Notice tone="error"><span>{runError}</span> <Button variant="secondary" size="compact" onPress={refresh}>Retry</Button></Notice>}
          {runState === "loading" && runs.length === 0 && <p className="reports-library__loading" role="status">Loading reports…</p>}
          {runState === "live" && filteredRuns.length === 0 && <EmptyState
            population={area === "ALL" ? "generated reports" : reportDatasetLabel(area)}
            title={runs.length ? "No matches" : "No reports yet"}
            description={runs.length ? "Change filters or search." : "Generate a report to get started."}
          />}
          {filteredRuns.length > 0 && <DataTable
            ariaLabel="Generated reports"
            rows={filteredRuns}
            rowKey={(run) => run.id}
            rowName={(run) => definitionsByID.get(run.definition_id)?.name || humanizeCode(run.definition_code)}
            columns={columns}
            onRowAction={setSelectedRun}
            rowActionLabel="View"
            isLoading={runState === "loading"}
          />}
          {nextCursor && <div className="reports-library__history-footer">
            <Button variant="secondary" isLoading={loadingOlder} onPress={() => void loadOlderReports()}>Load older reports</Button>
          </div>}
        </div>
        : <div className="reports-templates">
          <ReportingPage
            embedded
            organizationName={organizationName}
            legalEntityName={legalEntityName}
            onBack={() => setTab("library")}
          />
        </div>
      }
    </Tabs>

    {generateOpen && <FocusedSheet label="Generate report" onClose={() => setGenerateOpen(false)}>
      <div className="reports-generate">
        <div className="reports-generate__heading">
          <h2>Generate report</h2>
          <p>Select area and setup.</p>
        </div>
        <SelectField
          label="Area"
          value={generationArea}
          placeholder="Choose an area"
          options={generationAreaOptions}
          allowsEmpty={false}
          onChange={(value) => setGenerationArea((value || "VENDORS") as Exclude<ReportArea, "ALL">)}
        />
        {activeGenerationDefinitions.length > 0 ? <>
          <SelectField
            label="Saved setup"
            value={generationDefinitionID}
            placeholder="Choose a saved setup"
            options={activeGenerationDefinitions.map((definition) => ({ id: definition.id, label: definition.name }))}
            allowsEmpty={false}
            onChange={setGenerationDefinitionID}
          />
          {generationDefinitionID && <ReportTemplateSummary definition={activeGenerationDefinitions.find((definition) => definition.id === generationDefinitionID)} />}
          {commandError && <Notice tone="error"><span>{commandError}</span></Notice>}
          <div className="reports-generate__footer">
            <Button variant="secondary" onPress={() => setGenerateOpen(false)}>Cancel</Button>
            <Button variant="primary" isLoading={command === "generating"} onPress={() => void generateReport()}>Generate report</Button>
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

function ReportSummaryMetric({ label, value, tone = "neutral" }: { label: string; value: number; tone?: StatusTone }) {
  return <div className="reports-summary__metric" data-tone={tone}>
    <span>{label}</span>
    <strong>{value.toLocaleString()}</strong>
  </div>;
}

function ReportTemplateSummary({ definition }: { definition?: ReportDefinition }) {
  if (!definition) return null;
  return <dl className="reports-generate__summary">
    <div><dt>Area</dt><dd>{reportDatasetLabel(definition.dataset)}</dd></div>
    <div><dt>Shows</dt><dd>{reportSetupFocusLabel(reportSetupFocus(definition))}</dd></div>
  </dl>;
}

function ReportRunDetails({ run, definition, downloading, onDownload }: { run: ReportRun; definition?: ReportDefinition; downloading: boolean; onDownload: () => void }) {
  const available = run.status === "READY" && !isExpired(run);
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
    {run.failure_code && <Notice tone="error"><span>Generation stopped: {humanizeCode(run.failure_code)}</span></Notice>}
    <div className="report-run-details__footer">
      <Button variant="secondary" isDisabled={!available} isLoading={downloading} onPress={onDownload}>
        {isExpired(run) ? "File expired" : "Download report"}
      </Button>
    </div>
  </div>;
}

function mergeReportRuns(current: ReportRun[], incoming: ReportRun[]) {
  const byID = new Map(current.map((run) => [run.id, run]));
  for (const run of incoming) byID.set(run.id, run);
  return [...byID.values()].sort((left, right) => {
    const dateDifference = Date.parse(right.created_at) - Date.parse(left.created_at);
    return dateDifference || right.id.localeCompare(left.id);
  });
}

function reportDatasetLabel(dataset: ReportDataset | ReportArea) {
  switch (dataset) {
    case "VENDORS": return "Vendors";
    case "PROGRAMS": return "Programs";
    case "WORK":
    case "MATTERS":
    case "MATTER_EXCEPTIONS": return "Work";
    case "PROCESSING_ACTIVITIES":
    case "PROCESSING_ACTIVITY_EXCEPTIONS": return "Processing activities";
    default: return "All reports";
  }
}

function reportBelongsToArea(dataset: ReportDataset, area: Exclude<ReportArea, "ALL">) {
  if (area === "WORK") return dataset === "MATTERS" || dataset === "MATTER_EXCEPTIONS";
  return dataset === area;
}

function runStatusLabel(run: ReportRun): string {
  if (run.status === "READY" && isExpired(run)) return "Expired";
  switch (run.status) {
    case "QUEUED": return "Queued";
    case "RUNNING": return "Generating";
    case "READY": return "Ready";
    case "FAILED": return "Failed";
    default: return "Unknown";
  }
}

function runStatusTone(run: ReportRun): StatusTone {
  if (run.status === "READY" && isExpired(run)) return "warning";
  switch (run.status) {
    case "READY": return "success";
    case "QUEUED":
    case "RUNNING": return "info";
    case "FAILED": return "error";
    default: return "unknown";
  }
}

function isExpired(run: ReportRun) {
  return Date.parse(run.expires_at) <= Date.now();
}

function formatDateTime(value?: string) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

function humanizeCode(value: string) {
  return value.replaceAll("_", " ").replaceAll("-", " ").toLowerCase().replace(/\b\w/g, (character) => character.toUpperCase());
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
