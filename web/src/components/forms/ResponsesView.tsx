import { useEffect, useRef, useState } from "react";
import {
  loadCompletedResponse,
  loadCompletedResponses,
  loadResponseRevisions,
  type CompletedResponseDetail,
  type CompletedResponseQuery,
  type CompletedResponseSummary,
  type ResponseConcernBand,
  type ResponseScore,
  type ResponseScoreMode,
  type ResponseScoreState,
  type ResponseSort,
  type ResponseRevision,
} from "../../formsDistributionApi";
import { ApiError } from "../../http";
import { concernText, concernTone, coverageText, scorePresentation } from "./responseScorePresentation";
import { ResponseAssessment } from "./ResponseAssessment";
import {
  ActionLink,
  Button,
  DataTable,
  EmptyState,
  FilterBar,
  FilterChip,
  FocusedSheet,
  Notice,
  SelectField,
  StatusBadge,
  Surface,
  TextField,
  type DataColumn,
} from "../ui";

type ListState = "loading" | "live" | "sign-in-required" | "error";
type DetailState = "idle" | "loading" | "live" | "error";

const sortOptions = [
  { id: "CONCERN_DESC", label: "Needs attention first", description: "Highest adverse score, then most recent" },
  { id: "COMPLETED_DESC", label: "Most recent", description: "Latest submitted response first" },
  { id: "RAW_ASC", label: "Lowest score first" },
  { id: "RAW_DESC", label: "Highest score first" },
] as const;
const concernOptions = [
  { id: "CRITICAL", label: "Critical" }, { id: "HIGH", label: "High" },
  { id: "MODERATE", label: "Moderate" }, { id: "LOW", label: "Low" },
] as const;
const modeOptions = [
  { id: "COMPLIANCE", label: "Compliance" }, { id: "RISK", label: "Risk" },
] as const;
const scoreStateOptions = [
  { id: "FINAL", label: "Final score" }, { id: "PROVISIONAL", label: "Provisional score" },
  { id: "FAILED", label: "Score unavailable" }, { id: "NOT_CONFIGURED", label: "Not scored" },
] as const;

export function ResponsesView() {
  const [query, setQuery] = useState<CompletedResponseQuery>(() => readQuery());
  const [listState, setListState] = useState<ListState>("loading");
  const [items, setItems] = useState<CompletedResponseSummary[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string>();
  const [selectedID, setSelectedID] = useState<string>();
  const [detailState, setDetailState] = useState<DetailState>("idle");
  const [detail, setDetail] = useState<CompletedResponseDetail>();
  const [detailError, setDetailError] = useState<string>();
  const [revisions, setRevisions] = useState<ResponseRevision[]>([]);
  const [revisionsError, setRevisionsError] = useState<string>();
  const requestSequence = useRef(0);

  useEffect(() => {
    const sequence = ++requestSequence.current;
    const timer = window.setTimeout(() => { void refresh(sequence); }, 150);
    return () => window.clearTimeout(timer);
  }, [query.sort, query.bands?.join(","), query.modes?.join(","), query.states?.join(","), query.completed_from, query.completed_until, query.subject_type]);

  async function refresh(sequence = ++requestSequence.current) {
    setListState("loading");
    setError(undefined);
    try {
      const page = await loadCompletedResponses({ ...query, cursor: undefined, current_only: true, limit: query.limit ?? 25 });
      if (sequence !== requestSequence.current) return;
      setItems(page.items);
      setNextCursor(page.next_cursor);
      setListState("live");
    } catch (cause) {
      if (sequence !== requestSequence.current) return;
      setItems([]);
      setNextCursor(undefined);
      setError(message(cause, "Responses could not be loaded for the current filters."));
      setListState(cause instanceof ApiError && cause.status === 401 ? "sign-in-required" : "error");
    }
  }

  async function loadMore() {
    if (!nextCursor || loadingMore) return;
    setLoadingMore(true);
    try {
      const page = await loadCompletedResponses({ ...query, cursor: nextCursor, current_only: true, limit: query.limit ?? 25 });
      setItems((current) => [...current, ...page.items]);
      setNextCursor(page.next_cursor);
    } catch (cause) {
      setError(message(cause, "More responses could not be loaded."));
    } finally {
      setLoadingMore(false);
    }
  }

  function updateQuery(patch: Partial<CompletedResponseQuery>) {
    const next = { ...query, ...patch, cursor: undefined };
    setQuery(next);
    writeQuery(next);
  }

  function clearFilters() {
    const next: CompletedResponseQuery = { sort: "CONCERN_DESC", current_only: true, limit: 25 };
    setQuery(next);
    writeQuery(next);
  }

  async function reviewResponse(id: string) {
    const selected = items.find((value) => value.id === id);
    if (!selected) return;
    setSelectedID(id);
    setDetail(undefined);
    setDetailError(undefined);
    setRevisions([]);
    setRevisionsError(undefined);
    setDetailState("loading");
    const [detailResult, revisionsResult] = await Promise.allSettled([
      loadCompletedResponse(id),
      loadResponseRevisions(selected.distribution_id),
    ]);
    if (revisionsResult.status === "fulfilled") setRevisions(revisionsResult.value.items);
    else setRevisionsError(message(revisionsResult.reason, "Response version history could not be loaded."));
    if (detailResult.status === "fulfilled") {
      setDetail(detailResult.value);
      setDetailState("live");
    } else {
      setDetailError(message(detailResult.reason, "This submitted response could not be loaded."));
      setDetailState("error");
    }
  }

  const columns: readonly DataColumn<CompletedResponseSummary>[] = [
    { id: "form", header: "Form", render: (value) => <div className="forms-responses__form"><strong>{value.title}</strong><span>Revision {value.form_template_version}</span></div>, accessibleText: (value) => `${value.title}, form revision ${value.form_template_version}` },
    { id: "subject", header: "Subject", render: (value) => <div className="forms-responses__subject"><strong>{humanize(value.subject_type)}</strong><span>{value.subject_id}</span></div>, accessibleText: (value) => `${humanize(value.subject_type)} ${value.subject_id}` },
    { id: "completed", header: "Submitted", render: (value) => <time dateTime={value.completed_at}>{formatDateTime(value.completed_at)}</time>, accessibleText: (value) => formatDateTime(value.completed_at) },
    { id: "score", header: "Score", render: (value) => <ScoreCell score={value.score}/>, accessibleText: (value) => scoreAccessibleText(value.score) },
    { id: "concern", header: "Concern", kind: "status", render: (value) => <ConcernBadge score={value.score}/>, accessibleText: (value) => concernText(value.score) },
    { id: "coverage", header: "Coverage", kind: "number", render: (value) => coverageText(value.score), accessibleText: (value) => coverageText(value.score) },
    { id: "action", header: "Review", kind: "action", render: (value) => <Button variant="quiet" aria-label={`Review ${value.title} response`} onPress={() => void reviewResponse(value.id)}>Review response</Button>, accessibleText: (value) => `Review ${value.title} response` },
  ];

  return <section className="forms-responses" aria-labelledby="responses-title">
    <header className="forms-responses__heading">
      <div><h2 id="responses-title">Responses</h2></div>
    </header>

    {error && listState === "live" && <Notice tone="error">{error} The responses already shown remain available.</Notice>}
    <div className="forms-responses__sort"><SelectField label="Priority" value={query.sort ?? "CONCERN_DESC"} placeholder="Needs attention first" options={sortOptions} allowsEmpty={false} onChange={(sort) => updateQuery({ sort: sort as ResponseSort | undefined })}/></div>
    <details className="forms-response-filters"><summary>Filters{hasFilters(query) ? " · Applied" : ""}</summary>
    <FilterBar
      label="Response filters"
      fields={<>
        <SelectField label="Concern" value={query.bands?.[0]} placeholder="All concern levels" options={concernOptions} onChange={(band) => updateQuery({ bands: band ? [band as ResponseConcernBand] : undefined })}/>
        <SelectField label="Score meaning" value={query.modes?.[0]} placeholder="All score meanings" options={modeOptions} onChange={(mode) => updateQuery({ modes: mode ? [mode as ResponseScoreMode] : undefined })}/>
        <SelectField label="Score state" value={query.states?.[0]} placeholder="All score states" options={scoreStateOptions} onChange={(state) => updateQuery({ states: state ? [state as ResponseScoreState] : undefined })}/>
        <TextField label="Submitted from" type="date" value={dateInputValue(query.completed_from)} onChange={(value) => updateQuery({ completed_from: startOfDate(value) })}/>
        <TextField label="Submitted until" type="date" value={dateInputValue(query.completed_until)} onChange={(value) => updateQuery({ completed_until: endOfDate(value) })}/>
        <TextField label="Subject type" value={query.subject_type ?? ""} placeholder="For example, Vendor" onChange={(value) => updateQuery({ subject_type: value.trim() || undefined })}/>
      </>}
    />

    </details>
    <div className="forms-responses__filter-summary">{listState === "live" && <output>{items.length} responses on this page</output>}{hasFilters(query) && <Button variant="quiet" onPress={clearFilters}>Clear response filters</Button>}</div>

    {hasFilters(query) && <div className="forms-responses__chips" aria-label="Applied response filters">
      {query.bands?.[0] && <FilterChip label="Concern" value={humanize(query.bands[0])} onRemove={() => updateQuery({ bands: undefined })}/>}
      {query.modes?.[0] && <FilterChip label="Score meaning" value={humanize(query.modes[0])} onRemove={() => updateQuery({ modes: undefined })}/>}
      {query.states?.[0] && <FilterChip label="Score state" value={humanize(query.states[0])} onRemove={() => updateQuery({ states: undefined })}/>}
      {query.completed_from && <FilterChip label="Submitted from" value={formatDate(query.completed_from)} onRemove={() => updateQuery({ completed_from: undefined })}/>}
      {query.completed_until && <FilterChip label="Submitted until" value={formatDate(query.completed_until)} onRemove={() => updateQuery({ completed_until: undefined })}/>}
      {query.subject_type && <FilterChip label="Subject type" value={query.subject_type} onRemove={() => updateQuery({ subject_type: undefined })}/>}
    </div>}

    <div className="forms-responses__results" aria-live="polite">
      {listState === "loading" && items.length === 0 && <Surface><p role="status">Loading responses matching the current filters…</p></Surface>}
      {listState === "sign-in-required" && <EmptyState population="Responses matching the current filters" title="Sign in to review responses" description="Your session ended before this response list could be loaded." action={<ActionLink href="/">Sign in again</ActionLink>}/>}
      {listState === "error" && <EmptyState population="Responses matching the current filters" title="Responses could not be loaded" description={error ?? "The current response query could not be completed."} action={<Button onPress={() => void refresh()}>Try again</Button>}/>}
      {listState === "live" && items.length === 0 && <EmptyState population="Responses matching the current filters" title="No responses match these filters" description="Change or clear the filters to review a different response population."/>}
      {(listState === "live" || items.length > 0) && items.length > 0 && <DataTable
        ariaLabel="Submitted form responses"
        rows={items}
        rowKey={(value) => value.id}
        rowName={(value) => `${value.title}, ${humanize(value.subject_type)} ${value.subject_id}, ${scoreAccessibleText(value.score)}, submitted ${formatDateTime(value.completed_at)}`}
        columns={columns}
        isLoading={listState === "loading"}
        pagination={nextCursor ? { label: "Response pages", nextLabel: "Load more responses", onNext: () => void loadMore(), isLoading: loadingMore } : undefined}
      />}
    </div>

    {selectedID && <FocusedSheet label={`Review ${items.find((value) => value.id === selectedID)?.title ?? "submitted"} response`} size="wide" panelClassName="forms-response-review" onClose={() => { setSelectedID(undefined); setDetail(undefined); setDetailState("idle"); }}>
      <ResponseReview state={detailState} detail={detail} error={detailError} revisions={revisions} revisionsError={revisionsError}/>
    </FocusedSheet>}
  </section>;
}

function ScoreCell({ score }: { score?: ResponseScore }) {
  const presentation = scorePresentation(score);
  return <div className="forms-responses__score"><strong>{presentation.value}</strong><span>{presentation.meaning}</span></div>;
}

function ConcernBadge({ score }: { score?: ResponseScore }) {
  return <StatusBadge tone={concernTone(score)}>{concernText(score)}</StatusBadge>;
}

function ResponseReview({ state, detail, error, revisions, revisionsError }: { state: DetailState; detail?: CompletedResponseDetail; error?: string; revisions: ResponseRevision[]; revisionsError?: string }) {
  if (state === "loading") return <p role="status">Loading the submitted response and score explanation…</p>;
  if (state === "error") return <EmptyState population="The selected submitted response" title="Response details could not be loaded" description={error ?? "Select the response again to retry."}/>;
  if (state !== "live" || !detail) return null;
  return <div className="forms-response-review__content">
    <header className="cs-sheet-heading"><p>Submitted response</p><h2>{detail.response.title}</h2><p>{humanize(detail.response.subject_type)} · {detail.response.subject_id}</p></header>
    <dl className="cs-sheet-facts">
      <div><dt>Submitted</dt><dd>{formatDateTime(detail.response.completed_at)}</dd></div>
      <div><dt>Response revision</dt><dd>{detail.response.revision}{detail.response.current ? " · Current" : " · Historical"}</dd></div>
      <div><dt>Assurance</dt><dd>{assuranceLabel(detail.revision.achieved_assurance)}</dd></div>
    </dl>
    <section className="forms-response-review__history" aria-label="Version history">
      <h3>Version history</h3>
      {revisionsError && <Notice tone="warning">{revisionsError} The selected response remains available.</Notice>}
      {revisions.length > 0 && <ol>{revisions.map((revision) => <li key={revision.id}><strong>Revision {revision.revision}{revision.current ? " · Current" : ""}</strong><span>{assuranceLabel(revision.achieved_assurance)} · {formatDateTime(revision.created_at)}</span></li>)}</ol>}
    </section>
    <ResponseAssessment responseID={detail.response.id} submissionScore={detail.response.score} showResponseContext={false}/>
    <Notice tone="info">This submitted version cannot be changed. Send an amended form when the subject must provide updated information.</Notice>
  </div>;
}

function assuranceLabel(value: string) { return value === "EMAIL_VERIFIED" ? "Email verified" : value === "LINK_POSSESSION" ? "Secure link confirmed" : humanize(value); }

function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown date" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date); }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown time" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
function startOfDate(value: string) { return value ? new Date(`${value}T00:00:00.000Z`).toISOString() : undefined; }
function endOfDate(value: string) { return value ? new Date(`${value}T23:59:59.999Z`).toISOString() : undefined; }
function dateInputValue(value?: string) { return value?.slice(0, 10) ?? ""; }
function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }

function hasFilters(query: CompletedResponseQuery) {
  return Boolean(query.bands?.length || query.modes?.length || query.states?.length || query.completed_from || query.completed_until || query.subject_type || query.sort && query.sort !== "CONCERN_DESC");
}

function readQuery(): CompletedResponseQuery {
  const params = new URLSearchParams(window.location.search);
  const band = params.get("response_band") as ResponseConcernBand | null;
  const mode = params.get("response_mode") as ResponseScoreMode | null;
  const state = params.get("response_score_state") as ResponseScoreState | null;
  const sort = params.get("response_sort") as ResponseSort | null;
  return {
    sort: sort || "CONCERN_DESC", bands: band ? [band] : undefined, modes: mode ? [mode] : undefined, states: state ? [state] : undefined,
    completed_from: params.get("response_from") || undefined, completed_until: params.get("response_until") || undefined,
    subject_type: params.get("response_subject_type") || undefined, current_only: true, limit: 25,
  };
}

function writeQuery(query: CompletedResponseQuery) {
  const url = new URL(window.location.href);
  const set = (key: string, value?: string) => value ? url.searchParams.set(key, value) : url.searchParams.delete(key);
  set("response_sort", query.sort && query.sort !== "CONCERN_DESC" ? query.sort : undefined);
  set("response_band", query.bands?.[0]); set("response_mode", query.modes?.[0]); set("response_score_state", query.states?.[0]);
  set("response_from", query.completed_from); set("response_until", query.completed_until); set("response_subject_type", query.subject_type?.trim());
  window.history.replaceState(null, "", `${url.pathname}${url.search}${url.hash}`);
}

function scoreAccessibleText(score?: ResponseScore) { const value = scorePresentation(score); return `${value.value}, ${value.meaning}`; }
