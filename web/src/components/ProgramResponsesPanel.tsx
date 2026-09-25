import { useCallback, useEffect, useRef, useState } from "react";
import {
  loadCompletedResponse,
  loadCompletedResponses,
  type CompletedResponseDetail,
  type CompletedResponseSummary,
  type ResponseScore,
} from "../formsDistributionApi";
import { DocumentBrowser } from "./documents/DocumentBrowser";
import { EmptyState as RecordEmptyState } from "./EmptyState";
import { concernText, concernTone, coverageText, scorePresentation } from "./forms/responseScorePresentation";
import { ResponseAssessment } from "./forms/ResponseAssessment";
import { Button, DataTable, EmptyState, FocusedSheet, Notice, StatusBadge, Tabs, type DataColumn } from "./ui";

type ListState = "loading" | "live" | "unavailable";
type DetailState = "idle" | "loading" | "live" | "error";
const responseSections = [{ id: "ANSWERS", label: "Answers" }, { id: "DOCUMENTS", label: "Documents" }, { id: "REVIEW", label: "Review" }] as const;

export function ProgramResponsesPanel({ programID }: { programID: string }) {
  const [items, setItems] = useState<CompletedResponseSummary[]>([]);
  const [listState, setListState] = useState<ListState>("loading");
  const [nextCursor, setNextCursor] = useState<string>();
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string>();
  const [selectedID, setSelectedID] = useState<string>();
  const [detailState, setDetailState] = useState<DetailState>("idle");
  const [detail, setDetail] = useState<CompletedResponseDetail>();
  const [detailError, setDetailError] = useState<string>();
  const listSequence = useRef(0);
  const detailSequence = useRef(0);

  const loadList = useCallback(async (cursor?: string) => {
    const sequence = listSequence.current;
    if (!cursor) setListState("loading");
    setError(undefined);
    try {
      const page = await loadCompletedResponses({ subject_type: "PROGRAM", subject_id: programID, current_only: true, sort: "COMPLETED_DESC", limit: 20, cursor });
      if (sequence !== listSequence.current) return;
      setItems((current) => cursor ? [...current, ...page.items] : page.items);
      setNextCursor(page.next_cursor);
      setListState("live");
    } catch (cause) {
      if (sequence !== listSequence.current) return;
      if (cursor) {
        setError(cause instanceof Error ? cause.message : "More responses could not be loaded. The responses already shown remain available.");
      } else {
        setItems([]);
        setNextCursor(undefined);
        setListState("unavailable");
      }
    } finally {
      if (sequence === listSequence.current) setLoadingMore(false);
    }
  }, [programID]);

  useEffect(() => {
    const sequence = ++listSequence.current;
    void loadList();
    return () => { listSequence.current++; };
  }, [loadList]);

  async function loadMore() {
    if (!nextCursor || loadingMore) return;
    setLoadingMore(true);
    await loadList(nextCursor);
  }

  async function openResponse(id: string) {
    const sequence = ++detailSequence.current;
    setSelectedID(id);
    setDetail(undefined);
    setDetailError(undefined);
    setDetailState("loading");
    try {
      const value = await loadCompletedResponse(id);
      if (sequence !== detailSequence.current) return;
      setDetail(value);
      setDetailState("live");
    } catch (cause) {
      if (sequence !== detailSequence.current) return;
      setDetailError(cause instanceof Error ? cause.message : "This submitted response could not be loaded.");
      setDetailState("error");
    }
  }

  const columns: readonly DataColumn<CompletedResponseSummary>[] = [
    { id: "form", header: "Form", render: (value) => <div className="program-responses__form"><strong>{value.title}</strong><span>Revision {value.revision}{value.current ? " · Current" : " · Historical"} · Form revision {value.form_template_version}</span></div>, accessibleText: (value) => `${value.title}, response revision ${value.revision}, form revision ${value.form_template_version}` },
    { id: "completed", header: "Submitted", render: (value) => <time dateTime={value.completed_at}>{formatDateTime(value.completed_at)}</time>, accessibleText: (value) => formatDateTime(value.completed_at) },
    { id: "score", header: "Assessment result", render: (value) => <div className="program-response-result"><ConcernBadge score={value.score}/><ScoreCell score={value.score}/></div>, accessibleText: (value) => `${concernText(value.score)} · ${scoreAccessibleText(value.score)}` },
    { id: "action", header: "Review", kind: "action", render: (value) => <Button variant="quiet" aria-label={`Review ${value.title} response`} onPress={() => void openResponse(value.id)}>Review response</Button>, accessibleText: (value) => `Review ${value.title} response` },
  ];

  return <article className="program-record-panel program-wide-panel program-responses-panel" aria-labelledby="program-responses-heading">
    <div className="program-panel-heading"><div><span className="eyebrow">Responses</span><h2 id="program-responses-heading">Submitted data</h2><p>Submitted answers and documents · Assessment scores do not establish document validity.</p></div>{items.length > 0 && <span className="program-response-count">{items.length} responses loaded{nextCursor ? " · More available" : ""}</span>}</div>

    {listState === "loading" && items.length === 0 && <p role="status">Loading submitted responses for this Program…</p>}
    {listState === "unavailable" && items.length === 0 && <RecordEmptyState kind="unavailable" label="Completed responses for this Program" title="Submitted data could not be loaded" description="Completed responses cannot be reviewed while the response list is unavailable. Retry the list before reviewing evidence." action="Retry submitted data" onAction={() => void loadList()}/>}
    {listState === "live" && items.length === 0 && <RecordEmptyState label="Completed responses for this Program" title="No data collected yet" description="No completed responses are recorded for this Program. Start a form collection or add a collection check in Data collection to collect responses." action="Open Data collection" onAction={() => { window.location.hash = `#programs/${encodeURIComponent(programID)}/monitoring`; }}/>}
    {error && items.length > 0 && <Notice tone="error">{error} The responses already shown remain available.</Notice>}
    {(listState === "live" || items.length > 0) && items.length > 0 && <DataTable
      ariaLabel="Submitted responses for this Program"
      rows={items}
      rowKey={(value) => value.id}
      rowName={(value) => `${value.title}, submitted ${formatDateTime(value.completed_at)}, ${scoreAccessibleText(value.score)}`}
      columns={columns}
      isLoading={listState === "loading"}
      pagination={nextCursor ? { label: "Submitted response pages", nextLabel: "Load more responses", onNext: () => void loadMore(), isLoading: loadingMore } : undefined}
    />}

    {selectedID && <FocusedSheet label={`Review ${detail?.response.title ?? items.find((value) => value.id === selectedID)?.title ?? "submitted"} response`} size="wide" panelClassName="program-responses-review" onClose={() => { detailSequence.current++; setSelectedID(undefined); setDetail(undefined); setDetailError(undefined); setDetailState("idle"); }}>
      <ResponseReviewSheet key={selectedID} state={detailState} detail={detail} error={detailError} onRetry={() => void openResponse(selectedID)}/>
    </FocusedSheet>}
  </article>;
}

function ResponseReviewSheet({ state, detail, error, onRetry }: { state: DetailState; detail?: CompletedResponseDetail; error?: string; onRetry: () => void }) {
  const [section, setSection] = useState<"ANSWERS" | "DOCUMENTS" | "REVIEW">("ANSWERS");
  if (state === "loading") return <p role="status">Loading the submitted response and assessment…</p>;
  if (state === "error") return <EmptyState population="The selected submitted response" title="Response details could not be loaded" description={error ?? "Retry to load this submitted response."} action={<Button onPress={onRetry}>Retry response</Button>}/>;
  if (state !== "live" || !detail) return null;
  const score = detail.response.score ?? detail.revision.score;
  return <div className="program-responses-review__content">
    <header className="cs-sheet-heading"><p>Submitted response</p><h2>{detail.response.title}</h2><p>Program · {detail.response.subject_name ?? "Subject name unavailable"}</p></header>
    <dl className="cs-sheet-facts">
      <div><dt>Submitted</dt><dd>{formatDateTime(detail.response.completed_at)}</dd></div>
      <div><dt>Response revision</dt><dd>{detail.response.revision}{detail.response.current ? " · Current" : " · Historical"}</dd></div>
      <div><dt>Assurance</dt><dd>{assuranceLabel(detail.revision.achieved_assurance)}</dd></div>
      <div><dt>Form revision</dt><dd>{detail.response.form_template_version}</dd></div>
      <div><dt>Scoring coverage</dt><dd>{coverageText(score)}</dd></div>
    </dl>
    <Tabs ariaLabel="Response sections" compactLabel="Response section" retainVisitedPanels items={responseSections} selectedKey={section} onSelectionChange={setSection}>
      {(active) => active === "ANSWERS" ? <ResponseAssessment responseID={detail.response.id} answersOnly showDocumentLauncher={false}/> : active === "DOCUMENTS" ? <DocumentBrowser responseRevisionID={detail.response.id} scopeLabel={detail.response.title + " · Revision " + detail.response.revision}/> : <ResponseAssessment responseID={detail.response.id} submissionScore={score} showResponseContext={false} showDocumentLauncher={false} current={detail.response.current}/>}
    </Tabs>
    <Notice tone="info">This submitted response cannot be changed. Send an amended form to collect updated information.</Notice>
  </div>;
}

function ScoreCell({ score }: { score?: ResponseScore }) {
  const presentation = scorePresentation(score);
  return <div className="program-responses__score"><strong>{presentation.value}</strong><span>{presentation.meaning}</span></div>;
}

function ConcernBadge({ score }: { score?: ResponseScore }) {
  return <StatusBadge tone={concernTone(score)}>{concernText(score)}</StatusBadge>;
}

function assuranceLabel(value: string) { return value === "EMAIL_VERIFIED" ? "Email verified" : value === "LINK_POSSESSION" ? "Secure link confirmed" : humanize(value); }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown time" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
function scoreAccessibleText(score?: ResponseScore) { const value = scorePresentation(score); return `${value.value}, ${value.meaning}`; }
