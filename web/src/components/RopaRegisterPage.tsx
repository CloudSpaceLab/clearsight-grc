import { useEffect, useState } from "react";
import { fetchDashboard, listProcessingActivities, type RopaProcessingActivityListParams } from "../ropaApi";
import type { ProcessingActivity, ProcessingActivityPage, ProcessingActivityStatus, RegisterSummary } from "../ropaTypes";
import { RopaDashboardStrip } from "./RopaDashboardStrip";
import { Button, DataTable, EmptyState, FilterBar, Notice, SearchField, SelectField, StatusBadge, type DataColumn, type StatusTone } from "./ui";
import "./ropa.css";

type RegisterPageProps = {
  organizationName?: string;
  legalEntityName?: string;
  onOpenActivity?: (id: string) => void;
  onOpenReports?: () => void;
  loadSummary?: (signal?: AbortSignal) => Promise<RegisterSummary>;
  loadActivities?: (params?: RopaProcessingActivityListParams, signal?: AbortSignal) => Promise<ProcessingActivityPage>;
};

type ListState = "loading" | "live" | "error";
type SummaryState = "loading" | "live" | "error";

const unavailableSummary = async (_signal?: AbortSignal): Promise<RegisterSummary> => {
  throw new Error("Couldn’t load processing activities.");
};

const unavailableActivities = async (_params?: RopaProcessingActivityListParams, _signal?: AbortSignal): Promise<ProcessingActivityPage> => {
  throw new Error("The processing activity register could not be loaded. Try again.");
};

const statusOptions: ReadonlyArray<{ id: ProcessingActivityStatus; label: string }> = [
  { id: "NEW", label: "Not started" },
  { id: "OPEN", label: "In progress" },
  { id: "CLOSED", label: "Complete" },
];

export function RopaRegisterPage({ organizationName, legalEntityName, onOpenActivity, onOpenReports, loadSummary = fetchDashboard, loadActivities = listProcessingActivities }: RegisterPageProps) {
  const scope = legalEntityName || "this legal entity";
  const summaryReader = loadSummary ?? unavailableSummary;
  const activityReader = loadActivities ?? unavailableActivities;
  const [summary, setSummary] = useState<RegisterSummary>();
  const [summaryState, setSummaryState] = useState<SummaryState>("loading");
  const [summaryRetry, setSummaryRetry] = useState(0);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState<ProcessingActivityStatus>();
  const [cursors, setCursors] = useState<string[]>([]);
  const [page, setPage] = useState<ProcessingActivityPage>({ rows: [], has_more: false });
  const [listState, setListState] = useState<ListState>("loading");
  const [listRetry, setListRetry] = useState(0);
  const currentCursor = cursors[cursors.length - 1];

  useEffect(() => {
    const controller = new AbortController();
    setSummaryState("loading");
    void summaryReader(controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setSummary(value);
      setSummaryState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setSummary(undefined);
      setSummaryState("error");
    });
    return () => controller.abort();
  }, [summaryReader, summaryRetry]);

  useEffect(() => {
    const controller = new AbortController();
    setListState("loading");
    setPage({ rows: [], has_more: false });
    const params: RopaProcessingActivityListParams = {
      status,
      search: search.trim() || undefined,
      cursor: currentCursor,
    };
    void activityReader(params, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPage(value);
      setListState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setListState("error");
    });
    return () => controller.abort();
  }, [activityReader, status, search, currentCursor, listRetry]);

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

  const hasFilters = Boolean(search.trim() || status);
  const pagination = (cursors.length > 0 || (page.has_more && Boolean(page.next_cursor))) ? {
    label: "Processing activity pages",
    previousLabel: "Load previous page",
    nextLabel: "Load next page",
    onPrevious: cursors.length > 0 ? () => setCursors((value) => value.slice(0, -1)) : undefined,
    onNext: page.has_more && page.next_cursor ? () => setCursors((value) => [...value, page.next_cursor!]) : undefined,
    isLoading: listState === "loading",
  } : undefined;

  const columns: readonly DataColumn<ProcessingActivity>[] = [
    {
      id: "activity",
      header: "Processing activity",
      mobileLayout: "full-width",
      render: (item) => <span className="ropa-register-identity"><strong>{item.name || "Processing activity name not recorded"}</strong><small>{item.code || "Code not recorded"}</small></span>,
      accessibleText: (item) => `${item.name || "Processing activity name not recorded"}, ${item.code || "code not recorded"}`,
    },
    { id: "purpose", header: "Purpose", render: (item) => valueOrNotRecorded(item.purpose), accessibleText: (item) => valueOrNotRecorded(item.purpose) },
    { id: "lawful_basis", header: "Lawful basis", render: (item) => valueOrNotRecorded(item.lawful_basis), accessibleText: (item) => valueOrNotRecorded(item.lawful_basis) },
    { id: "owner", header: "Owner", render: (item) => ownerLabel(item), accessibleText: (item) => ownerLabel(item) },
    { id: "review", header: "Next review", render: (item) => <ReviewValue activity={item}/>, accessibleText: (item) => reviewAccessibleText(item) },
    { id: "status", header: "Status", kind: "status", render: (item) => <StatusBadge tone={statusTone(item.status)}>{statusLabel(item.status)}</StatusBadge>, accessibleText: (item) => statusLabel(item.status) },
  ];

  return <section className="ropa-register-page" aria-labelledby="ropa-register-heading">
    <header className="topbar ropa-page-header">
      <div>
        <span className="eyebrow">{organizationName || "Processing activity register"}</span>
        <h1 id="ropa-register-heading">Processing activity register</h1>
      </div>
      <div className="topbar-actions">
        <Button variant="secondary" onPress={openReports}>Reports</Button>
        <Button variant="secondary" onPress={() => { retrySummary(); retryList(); }} isLoading={summaryState === "loading" || listState === "loading"}>Refresh</Button>
      </div>
    </header>

    {summaryState === "loading" && !summary && <p className="ropa-load-state" role="status">Loading summary…</p>}
    {summaryState === "error" && <Notice tone="error">
      <span>Summary unavailable.</span> <Button variant="secondary" size="compact" onPress={retrySummary}>Retry</Button>
    </Notice>}
    {summary && <RopaDashboardStrip summary={summary} legalEntityName={scope} onRetry={retrySummary} onOpenStatus={(nextStatus) => { setSearch(""); setStatus(nextStatus); setCursors([]); setListRetry((value) => value + 1); }}/>}

    <section className="ropa-register-list" aria-labelledby="ropa-register-list-heading">
      <div className="section-header ropa-register-list__header">
        <div><h2 id="ropa-register-list-heading">Processing activities</h2></div>
      </div>
      <FilterBar
        label="Processing activity filters"
        fields={<>
          <SearchField label="Search processing activities" value={search} onChange={changeSearch} placeholder="Name, code or purpose" isLoading={listState === "loading"}/>
          <SelectField label="Activity status" value={status} placeholder="All activity statuses" options={statusOptions} onChange={changeStatus}/>
        </>}
        resultCount={listState === "live" ? page.rows.length : undefined}
        resultLabel={(count) => `${count} shown`}
        clearLabel="Clear filters"
        onClear={hasFilters ? clearFilters : undefined}
      />

      {listState === "loading" && page.rows.length === 0 && <p className="ropa-load-state" role="status">Loading activities…</p>}
      {listState === "error" && <Notice tone="error">
        <span>Couldn’t load activities.</span> <Button variant="secondary" size="compact" onPress={retryList}>Retry</Button>
      </Notice>}
      {listState === "live" && page.rows.length === 0 && hasFilters
        ? <EmptyState population={`${scope} · current register filters`} title="No matches" description="Change filters or search."/>
        : listState === "live" && page.rows.length === 0
          ? <EmptyState population={`${scope} · current register`} title="No activities" description="Add the first processing activity."/>
          : null}
      {page.rows.length > 0 && <DataTable
        ariaLabel="Processing activity register"
        rows={page.rows}
        rowKey={(item) => item.id}
        rowName={(item) => `${item.name || "Processing activity"}, ${item.code || "code not recorded"}, ${statusLabel(item.status)}`}
        columns={columns}
        onRowAction={(item) => openActivity(item.id)}
        isLoading={listState === "loading"}
        pagination={pagination}
      />}
      {listState === "live" && page.has_more && !page.next_cursor && <Notice tone="warning">More activities exist. Retry to continue.</Notice>}
    </section>
  </section>;
}

function ReviewValue({ activity }: { activity: ProcessingActivity }) {
  const review = parseReviewDate(activity.next_review_date);
  if (!activity.next_review_date) return <span>Not recorded</span>;
  if (!review) return <span>Review date unavailable</span>;
  const overdue = review.getTime() < startOfToday().getTime();
  return <StatusBadge tone={overdue ? "error" : "info"}>{overdue ? `Overdue · ${formatDate(activity.next_review_date!)}` : `Review due ${formatDate(activity.next_review_date!)}`}</StatusBadge>;
}

function reviewAccessibleText(activity: ProcessingActivity): string {
  if (!activity.next_review_date) return "Not recorded";
  const review = parseReviewDate(activity.next_review_date);
  if (!review) return "Review date unavailable";
  return review.getTime() < startOfToday().getTime() ? `Overdue, ${formatDate(activity.next_review_date)}` : `Review due ${formatDate(activity.next_review_date)}`;
}

function valueOrNotRecorded(value: string | undefined): string {
  return value && value.trim() ? value : "Not recorded";
}

function ownerLabel(activity: ProcessingActivity): string {
  if (activity.owner_display_name?.trim()) return activity.owner_display_name.trim();
  return activity.owner_principal_id ? "Assigned" : "Not assigned";
}

function statusLabel(status: ProcessingActivityStatus): string {
  if (status === "NEW") return "Not started";
  if (status === "OPEN") return "In progress";
  if (status === "CLOSED") return "Complete";
  return "Status unavailable";
}

function statusTone(status: ProcessingActivityStatus): StatusTone {
  if (status === "CLOSED") return "success";
  if (status === "OPEN") return "info";
  return "neutral";
}

function parseReviewDate(value: string | undefined): Date | undefined {
  if (!value) return undefined;
  const date = new Date(`${value.slice(0, 10)}T00:00:00Z`);
  return Number.isFinite(date.getTime()) ? date : undefined;
}

function startOfToday(): Date {
  const now = new Date();
  return new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()));
}

function formatDate(value: string): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "date unavailable";
  return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", timeZone: "UTC" }).format(date);
}

function isAbortError(error: unknown): boolean {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
