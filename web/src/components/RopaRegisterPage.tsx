import { lazy, Suspense, useEffect, useState } from "react";
import { fetchDashboard, listProcessingActivities, type RopaProcessingActivityListParams } from "../ropaApi";
import type { ProcessingActivityPage, ProcessingActivityStatus, RegisterSummary } from "../ropaTypes";
import { RopaDashboardStrip } from "./RopaDashboardStrip";
import { Button, FocusedSheet, Notice, Tabs } from "./ui";
import { RopaRegisterList, type RegisterListState } from "./ropa/RopaRegisterList";
import "./ropa.css";
import "./ropa/privacy-workspace.css";

const RopaActivityPage = lazy(() => import("./RopaActivityPage").then((module) => ({ default: module.RopaActivityPage })));
export type PrivacyView = "overview" | "register";
const views = [{ id: "overview" as const, label: "Overview" }, { id: "register" as const, label: "ROPA" }];

type RegisterPageProps = {
  organizationName?: string;
  legalEntityName?: string;
  view?: PrivacyView;
  activityID?: string;
  onViewChange?: (view: PrivacyView) => void;
  onCloseActivity?: () => void;
  onOpenActivity?: (id: string) => void;
  onOpenReports?: () => void;
  loadSummary?: (signal?: AbortSignal) => Promise<RegisterSummary>;
  loadActivities?: (params?: RopaProcessingActivityListParams, signal?: AbortSignal) => Promise<ProcessingActivityPage>;
};

type SummaryState = "loading" | "live" | "error" | "denied";

const unavailableSummary = async (_signal?: AbortSignal): Promise<RegisterSummary> => {
  throw new Error("Couldn’t load processing activities.");
};

const unavailableActivities = async (_params?: RopaProcessingActivityListParams, _signal?: AbortSignal): Promise<ProcessingActivityPage> => {
  throw new Error("The processing activity register could not be loaded. Try again.");
};

export function RopaRegisterPage({ organizationName, legalEntityName, view, activityID, onViewChange, onCloseActivity, onOpenActivity, onOpenReports, loadSummary = fetchDashboard, loadActivities = listProcessingActivities }: RegisterPageProps) {
  const scope = legalEntityName || "This legal entity";
  const [localView, setLocalView] = useState<PrivacyView>("register");
  const selectedView = view ?? localView;
  const summaryReader = loadSummary ?? unavailableSummary;
  const activityReader = loadActivities ?? unavailableActivities;
  const [summary, setSummary] = useState<RegisterSummary>();
  const [summaryState, setSummaryState] = useState<SummaryState>("loading");
  const [summaryRetry, setSummaryRetry] = useState(0);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<ProcessingActivityStatus>();
  const [includeRetired, setIncludeRetired] = useState(false);
  const [cursors, setCursors] = useState<string[]>([]);
  const [page, setPage] = useState<ProcessingActivityPage>({ rows: [], has_more: false });
  const [listState, setListState] = useState<RegisterListState>("loading");
  const [listRetry, setListRetry] = useState(0);
  const currentCursor = cursors[cursors.length - 1];

  useEffect(() => {
    if (selectedView !== "overview") return;
    const controller = new AbortController();
    setSummary(undefined);
    setSummaryState("loading");
    void summaryReader(controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setSummary(value);
      setSummaryState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setSummary(undefined);
      setSummaryState(isDenied(error) ? "denied" : "error");
    });
    return () => controller.abort();
  }, [summaryReader, summaryRetry, selectedView]);

  useEffect(() => {
    if (selectedView !== "register") return;
    const controller = new AbortController();
    setListState("loading");
    setPage({ rows: [], has_more: false });
    const params: RopaProcessingActivityListParams = {
      status,
      search: search.trim() || undefined,
      cursor: currentCursor,
      include_retired: includeRetired || undefined,
    };
    void activityReader(params, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPage(value);
      setListState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setListState(isDenied(error) ? "denied" : "error");
    });
    return () => controller.abort();
  }, [activityReader, status, search, currentCursor, includeRetired, listRetry, selectedView]);

  function changeSearch(value: string) {
    setSearch(value);
    setCursors([]);
  }

  function changeStatus(value: ProcessingActivityStatus | undefined) {
    setStatus(value);
    setCursors([]);
  }

  function clearFilters() {
    setSearch("");
    setStatus(undefined);
    setIncludeRetired(false);
    setCursors([]);
  }

  function retryList() {
    setListRetry((value) => value + 1);
  }

  function retrySummary() {
    setSummaryRetry((value) => value + 1);
  }

  function openActivity(id: string) {
    if (onOpenActivity) {
      onOpenActivity(id);
      return;
    }
    if (typeof window !== "undefined") window.location.hash = `#ropa/activity/${encodeURIComponent(id)}`;
  }

  function openReports() {
    if (onOpenReports) {
      onOpenReports();
      return;
    }
    if (typeof window !== "undefined") window.location.hash = "#ropa/reports";
  }

  const pagination = (cursors.length > 0 || (page.has_more && Boolean(page.next_cursor))) ? {
    label: "Processing activity pages",
    previousLabel: "Load previous page",
    nextLabel: "Load next page",
    onPrevious: cursors.length > 0 ? () => setCursors((value) => value.slice(0, -1)) : undefined,
    onNext: page.has_more && page.next_cursor ? () => setCursors((value) => [...value, page.next_cursor!]) : undefined,
    isLoading: listState === "loading",
  } : undefined;

  function changeView(nextView: PrivacyView) {
    setLocalView(nextView);
    onViewChange?.(nextView);
  }

  function closeActivity() {
    if (onCloseActivity) onCloseActivity();
    else if (typeof window !== "undefined") window.location.hash = "#ropa";
  }

  return <section className="privacy-workspace" aria-labelledby="privacy-heading">
    <header className="privacy-workspace__heading">
      <div>
        <p className="privacy-workspace__scope">{[organizationName, scope].filter(Boolean).join(" · ")}</p>
        <h1 id="privacy-heading">Privacy &amp; data</h1>
      </div>
      <svg className="privacy-workspace__wave" viewBox="0 0 560 170" preserveAspectRatio="none" aria-hidden="true">
        <path d="M0 166Q63 115 105 142T213 95 334 117 452 46 560 68V170H0" fill="currentColor"/>
        <path d="M0 170Q98 130 178 139T333 91 465 120 560 77V170H0" fill="currentColor"/>
      </svg>
      <div className="privacy-workspace__actions">
        <Button variant="secondary" onPress={openReports}>Reports</Button>
        <Button variant="secondary" onPress={selectedView === "overview" ? retrySummary : retryList} isLoading={selectedView === "overview" ? summaryState === "loading" : listState === "loading"}>Refresh</Button>
      </div>
    </header>
    <Tabs ariaLabel="Privacy views" items={views} selectedKey={selectedView} onSelectionChange={changeView} retainVisitedPanels>
      {(section) => section === "overview" ? <>
        {summaryState === "loading" && <p className="ropa-load-state" role="status">Loading summary…</p>}
        {summaryState === "denied" && <Notice tone="warning">Processing activity summary is unavailable with your current access. Contact your administrator.</Notice>}
        {summaryState === "error" && <Notice tone="error"><span>Summary unavailable.</span> <Button variant="secondary" size="compact" onPress={retrySummary}>Retry</Button></Notice>}
        {summary && <RopaDashboardStrip summary={summary} legalEntityName={scope} onRetry={retrySummary} onOpenStatus={(nextStatus) => {
          setSearch(""); setStatus(nextStatus); setIncludeRetired(true); setCursors([]); changeView("register");
        }}/>}
      </> : <RopaRegisterList
        scope={scope} search={search} status={status} includeRetired={includeRetired} onRemoveRetired={() => { setIncludeRetired(false); setCursors([]); }} page={page} state={listState} activityID={activityID} pagination={pagination}
        onSearch={changeSearch} onStatus={changeStatus} onClear={clearFilters} onRetry={retryList} onOpen={openActivity}
      />}
    </Tabs>
    {activityID && <FocusedSheet label="Processing activity" closeLabel="Close activity details" onClose={closeActivity} panelClassName="privacy-activity-sheet">
      <Suspense fallback={<p role="status">Loading activity…</p>}>
        <RopaActivityPage key={activityID} activityID={activityID} embedded organizationName={organizationName} legalEntityName={legalEntityName} onBack={closeActivity}/>
      </Suspense>
    </FocusedSheet>}
  </section>;
}

function isAbortError(error: unknown): boolean {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}

function isDenied(error: unknown): boolean {
  return typeof error === "object" && error !== null && "kind" in error && (error.kind === "forbidden" || error.kind === "unauthorized");
}
