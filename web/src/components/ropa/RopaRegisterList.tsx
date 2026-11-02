import type { ProcessingActivity, ProcessingActivityPage, ProcessingActivityStatus } from "../../ropaTypes";
import { Button, DataTable, EmptyState, FilterChip, Notice, SearchField, SelectField, StatusBadge, type DataColumn, type DataTablePagination, type StatusTone } from "../ui";

export type RegisterListState = "loading" | "live" | "error" | "denied";

type Props = {
  scope: string;
  search: string;
  status?: ProcessingActivityStatus;
  includeRetired: boolean;
  onRemoveRetired: () => void;
  page: ProcessingActivityPage;
  state: RegisterListState;
  activityID?: string;
  pagination?: DataTablePagination;
  onSearch: (value: string) => void;
  onStatus: (value: ProcessingActivityStatus | undefined) => void;
  onClear: () => void;
  onRetry: () => void;
  onOpen: (id: string) => void;
};

const statusOptions: ReadonlyArray<{ id: ProcessingActivityStatus; label: string }> = [
  { id: "NEW", label: "Not started" },
  { id: "OPEN", label: "In progress" },
  { id: "CLOSED", label: "Complete" },
];

export function RopaRegisterList({ scope, search, status, includeRetired, onRemoveRetired, page, state, activityID, pagination, onSearch, onStatus, onClear, onRetry, onOpen }: Props) {
  const hasQuery = Boolean(search.trim() || status);
  const hasFilters = hasQuery || includeRetired;
  const isLaterPage = Boolean(pagination?.onPrevious);
  const columns: readonly DataColumn<ProcessingActivity>[] = [
    {
      id: "activity",
      header: "Processing activity",
      mobileLayout: "full-width",
      render: (item) => <span className="ropa-register-identity"><strong>{item.name || "Processing activity name not recorded"}</strong><small>{item.code || "Code not recorded"}</small></span>,
      accessibleText: (item) => `${item.name || "Processing activity name not recorded"}, ${item.code || "code not recorded"}`,
    },
    { id: "subjects", header: "Data subjects", render: (item) => valueOrNotRecorded(item.data_subject_categories), accessibleText: (item) => valueOrNotRecorded(item.data_subject_categories) },
    { id: "lawful_basis", header: "Lawful basis", render: (item) => valueOrNotRecorded(item.lawful_basis), accessibleText: (item) => valueOrNotRecorded(item.lawful_basis) },
    { id: "review", header: "Next review", render: (item) => <ReviewValue activity={item}/>, accessibleText: (item) => reviewAccessibleText(item) },
    { id: "owner", header: "Owner", render: (item) => ownerLabel(item), accessibleText: (item) => ownerLabel(item) },
    { id: "status", header: "Status", kind: "status", render: (item) => <StatusBadge tone={statusTone(item.status)}>{statusLabel(item.status)}</StatusBadge>, accessibleText: (item) => statusLabel(item.status) },
  ];

  return <section className="ropa-register-list" aria-labelledby="ropa-register-list-heading">
    <header className="ropa-register-list__header">
      <h2 id="ropa-register-list-heading">Processing activities</h2>
      <p>{scope} · Lawful basis, data subjects and review dates</p>
    </header>
    <div className="ropa-register-list__filters" role="group" aria-label="Processing activity filters">
      <SearchField label="Search processing activities" value={search} onChange={onSearch} placeholder="Name, code or purpose" isLoading={state === "loading"}/>
      <SelectField label="Activity status" value={status} placeholder="All activity statuses" options={statusOptions} onChange={onStatus}/>
      <div className="ropa-register-list__results" aria-live="polite">
        {state === "live" && <span>{page.rows.length} shown{page.has_more ? " · More available" : ""}</span>}
        {hasFilters && <Button variant="quiet" size="compact" onPress={onClear}>Clear filters</Button>}
      </div>
    </div>
    {includeRetired && <div className="ropa-register-list__scope"><FilterChip label="Activity scope" value="Includes ended activities" onRemove={onRemoveRetired}/></div>}
    {state === "loading" && <p className="ropa-load-state" role="status">Loading activities…</p>}
    {state === "denied" && <Notice tone="warning">Processing activities are unavailable with your current access. Contact your administrator.</Notice>}
    {state === "error" && <Notice tone="error"><span>Couldn’t load activities.</span> <Button variant="secondary" size="compact" onPress={onRetry}>Retry</Button></Notice>}
    {state === "live" && page.rows.length === 0 && <EmptyState
      population={`${scope} · ${isLaterPage ? "current page" : hasQuery ? "current register filters" : "current register"}`}
      title={isLaterPage ? "No more activities" : hasQuery ? "No matches" : "No activities"}
      description={isLaterPage ? "Return to the previous page." : hasQuery ? "Change filters or search." : includeRetired ? "No processing activities were found for this legal entity." : "No current processing activities were found for this legal entity."}
    />}
    {page.rows.length > 0 && <DataTable
      ariaLabel="Processing activity register"
      rows={page.rows}
      rowKey={(item) => item.id}
      rowName={(item) => `${item.name || "Processing activity"}, ${item.code || "code not recorded"}, ${statusLabel(item.status)}`}
      columns={columns}
      selectedKey={activityID}
      onRowAction={(item) => onOpen(item.id)}
      isLoading={state === "loading"}
      pagination={pagination}
      responsiveTo="container"
    />}
    {page.rows.length === 0 && pagination?.onPrevious && state !== "denied" && <nav className="ropa-register-list__recovery" aria-label="Processing activity pages"><Button variant="secondary" isDisabled={state === "loading"} onPress={pagination.onPrevious}>Load previous page</Button></nav>}
    {state === "live" && page.has_more && !page.next_cursor && <Notice tone="warning">More activities exist. Retry to continue.</Notice>}
  </section>;
}

function ReviewValue({ activity }: { activity: ProcessingActivity }) {
  const review = parseReviewDate(activity.next_review_date);
  if (!activity.next_review_date) return <span>Not recorded</span>;
  if (!review) return <span>Review date unavailable</span>;
  return <span>{formatDate(activity.next_review_date)}</span>;
}

function reviewAccessibleText(activity: ProcessingActivity): string {
  if (!activity.next_review_date) return "Not recorded";
  const review = parseReviewDate(activity.next_review_date);
  if (!review) return "Review date unavailable";
  return formatDate(activity.next_review_date);
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

function formatDate(value: string): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "date unavailable";
  return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", timeZone: "UTC" }).format(date);
}
