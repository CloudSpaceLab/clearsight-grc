import { useId, useMemo, useState } from "react";
import type { ReportDefinition, ReportRun } from "../../reportingTypes";
import { NavigationIcon } from "../NavigationIcon";
import { Button, EmptyState, Notice, SearchField, SelectableRecord, SelectField, StatusBadge } from "../ui";
import {
  formatDateTime, humanizeCode, isExpired, isReportAvailable, reportBelongsToArea,
  reportDatasetLabel, runStatusLabel, runStatusTone, type ReportArea,
} from "./reportRunPresentation";
import "./reports-library.css";

type ReportStatusFilter = "ALL" | "READY" | "IN_PROGRESS" | "FAILED" | "EXPIRED";

export type ReportsLibraryProps = {
  runs: ReportRun[];
  definitionsByID: ReadonlyMap<string, ReportDefinition>;
  state: "loading" | "live" | "error";
  error?: string;
  query: string;
  onQueryChange: (query: string) => void;
  area: ReportArea;
  onAreaChange: (area: ReportArea) => void;
  hasMore?: boolean;
  loadingMore?: boolean;
  downloading?: boolean;
  scopeLabel?: string;
  onLoadMore: () => void;
  onRetry: () => void;
  onInspect: (run: ReportRun) => void;
  onDownload: (run: ReportRun) => void;
};

const areas = [
  { id: "ALL", label: "All areas" }, { id: "VENDORS", label: "Vendors" },
  { id: "PROGRAMS", label: "Programs" }, { id: "WORK", label: "Work" },
] as const;
const statuses = [
  { id: "ALL", label: "All statuses" }, { id: "READY", label: "Ready" },
  { id: "IN_PROGRESS", label: "In progress" }, { id: "FAILED", label: "Failed" },
  { id: "EXPIRED", label: "Expired" },
] as const;

export function ReportsLibrary({
  runs, definitionsByID, state, error, query, onQueryChange, area, onAreaChange,
  hasMore = false, loadingMore = false, downloading = false, scopeLabel,
  onLoadMore, onRetry, onInspect, onDownload,
}: ReportsLibraryProps) {
  const inspectorID = useId();
  const [selectedID, setSelectedID] = useState<string>();
  const [status, setStatus] = useState<ReportStatusFilter>("ALL");
  const visibleRuns = useMemo(() => {
    const search = query.trim().toLowerCase();
    return runs.filter((run) => (area === "ALL" || reportBelongsToArea(run.dataset, area))
      && matchesStatus(run, status)
      && (!search || [definitionsByID.get(run.definition_id)?.name, run.definition_code,
        reportDatasetLabel(run.dataset), runStatusLabel(run), run.format]
        .some((value) => value?.toLowerCase().includes(search))))
      .sort((left, right) => Date.parse(right.created_at) - Date.parse(left.created_at)
        || right.id.localeCompare(left.id));
  }, [runs, definitionsByID, query, area, status]);
  const selected = state === "error" ? undefined
    : visibleRuns.find((run) => run.id === selectedID) ?? visibleRuns[0];
  const title = (run: ReportRun) => definitionsByID.get(run.definition_id)?.name || humanizeCode(run.definition_code);

  return <section className="reports-v2-library" aria-label="Generated reports" aria-busy={state === "loading" || undefined}>
    <div className="reports-v2-library__register">
      <header className="reports-v2-library__heading">
        <h2>Reports</h2>
        {state !== "error" && <span role="status">{state === "loading" ? "Loading reports…"
          : `${visibleRuns.length} matching · ${runs.length} loaded${hasMore ? " · More available" : ""}`}</span>}
      </header>
      <div className="reports-v2-library__filters">
        <SearchField label="Search generated reports" value={query} onChange={onQueryChange}
          placeholder="Search reports" isLoading={state === "loading"}/>
        <SelectField label="Report status" value={status} placeholder="All statuses" options={statuses}
          allowsEmpty={false} isLabelHidden onChange={(value) => setStatus(value ?? "ALL")}/>
        <SelectField label="Area" value={area} placeholder="All areas" options={areas}
          allowsEmpty={false} isLabelHidden onChange={(value) => onAreaChange(value ?? "ALL")}/>
      </div>
      {state === "error" && <EmptyState compact role="alert" population="Generated reports"
        title="Reports unavailable" description={error || "Reports failed to load."}
        action={<Button variant="secondary" onPress={onRetry}>Retry</Button>}/>}
      {state === "live" && visibleRuns.length === 0 && <EmptyState compact
        population="Generated reports" title={runs.length ? "No matching loaded reports" : "No reports yet"}
        description={runs.length ? hasMore ? "Clear filters or load older reports." : "Change filters or search."
          : "Generate a report to get started."}/>}
      {state !== "error" && visibleRuns.length > 0 && <ul className="reports-v2-library__records" aria-label="Generated report records">
        {visibleRuns.map((run) => <li key={run.id}>
          <SelectableRecord title={title(run)}
            metadata={`${reportDatasetLabel(run.dataset)} · ${formatDateTime(run.completed_at || run.created_at)} · ${run.format}`}
            startContent={<span className="reports-v2-library__file-icon"><NavigationIcon view="reports"/></span>}
            endContent={<StatusBadge tone={runStatusTone(run)}>{runStatusLabel(run)}</StatusBadge>}
            isSelected={selected?.id === run.id} isDisabled={state === "loading"}
            aria-label={`Select ${title(run)}, ${runStatusLabel(run)}, ${formatDateTime(run.created_at)}`}
            aria-controls={inspectorID} onPress={() => setSelectedID(run.id)}/>
        </li>)}
      </ul>}
      {state === "live" && hasMore && <div className="reports-v2-library__pagination">
        <Button variant="secondary" isLoading={loadingMore} onPress={onLoadMore}>Load older reports</Button>
      </div>}
    </div>
    <aside id={inspectorID} className="reports-v2-library__inspector" aria-label="Selected report">
      <span className="eyebrow">Selected report</span>
      {selected ? <>
        <div className="reports-v2-library__identity">
          <span className="reports-v2-library__file-icon" aria-hidden="true"><NavigationIcon view="reports"/></span>
          <div><h2>{title(selected)}</h2><p>{reportDatasetLabel(selected.dataset)}</p></div>
        </div>
        <StatusBadge tone={runStatusTone(selected)}>{runStatusLabel(selected)}</StatusBadge>
        <dl className="reports-v2-library__facts">
          <div><dt>Format</dt><dd>{selected.format}</dd></div>
          <div><dt>Scope</dt><dd>{selected.scope_kind === "LEGAL_ENTITY" ? scopeLabel || "Legal entity"
            : selected.scope_kind === "PROGRAM" ? "Program" : "Issue"}</dd></div>
          <div><dt>Data as of</dt><dd>{formatDateTime(selected.as_of)}</dd></div>
          <div><dt>Rows</dt><dd>{selected.status === "READY" ? selected.row_count.toLocaleString() : "Not ready"}</dd></div>
          <div><dt>Source coverage</dt><dd>{selected.status !== "READY" ? "Not ready"
            : selected.source_boundary?.population_complete === true ? "Complete"
            : selected.source_boundary?.population_complete === false ? "Incomplete" : "Unknown"}</dd></div>
        </dl>
        {selected.status === "FAILED" && <Notice tone="error">{selected.failure_code
          ? `Generation stopped: ${humanizeCode(selected.failure_code)}` : "Report generation failed."}</Notice>}
        {selected.status === "READY" && !isReportAvailable(selected) && <p className="reports-v2-library__file-state">
          {isExpired(selected) ? "File expired. Generate a new report to download."
            : "File availability is unknown. Refresh reports."}
        </p>}
        <div className="reports-v2-library__actions">
          <Button variant="primary" isDisabled={state !== "live" || !isReportAvailable(selected)}
            isLoading={downloading} onPress={() => onDownload(selected)}>
            {isExpired(selected) && selected.status === "READY" ? "File expired" : "Download report"}
          </Button>
          <Button variant="secondary" isDisabled={state !== "live"} onPress={() => onInspect(selected)}>View report details</Button>
        </div>
      </> : <p className="reports-v2-library__file-state">{state === "loading" ? "Loading report details…"
        : state === "error" ? "Report details unavailable." : "Select a report to view its details."}</p>}
    </aside>
  </section>;
}

function matchesStatus(run: ReportRun, status: ReportStatusFilter) {
  switch (status) {
    case "READY": return isReportAvailable(run);
    case "IN_PROGRESS": return run.status === "QUEUED" || run.status === "RUNNING";
    case "EXPIRED": return run.status === "READY" && isExpired(run);
    case "FAILED": return run.status === "FAILED";
    default: return true;
  }
}
