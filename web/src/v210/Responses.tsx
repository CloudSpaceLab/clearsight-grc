import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadCompletedResponse, loadCompletedResponses, type CompletedResponseDetail, type CompletedResponseSummary } from "../formsDistributionApi";
import { loadResponseReviews, recordResponseReview, type ResponseReviewDecision, type ResponseReviewPage, type RecordResponseReviewInput } from "../responseReviewApi";
import type { CaptureAnswerValue } from "../types";
import { documentContentURL, loadDocuments } from "../submittedDocumentApi";
import { apiErrorKind } from "../http";
import { documentReturnDestination, scopedDocumentsHref } from "./documentReturn";
import "./responses.css";

type ReviewScope = { tenant: string; entity: string; actor: string };
type PendingReview = { key: string; revision: number; input: RecordResponseReviewInput; needsComment?: boolean };
export function Responses({ id, reviewScope }: { id?: string; reviewScope?: ReviewScope }) {
  const scope = new URLSearchParams(window.location.hash.split("?")[1] ?? "").getAll("distribution_id");
  return id ? <ResponseDetail key={JSON.stringify([id, scope, reviewScope])} id={id} reviewScope={reviewScope}/> : <ResponseList/>;
}

function ResponseList() {
  const [items, setItems] = useState<CompletedResponseSummary[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [search, setSearch] = useState(""); const [draft, setDraft] = useState("");
  const [retry, setRetry] = useState(0); const [moreBusy, setMoreBusy] = useState(false); const [moreError, setMoreError] = useState(false);
  const generation = useRef(0);
  const moreLock = useRef<number | null>(null);
  useEffect(() => {
    const token = ++generation.current; moreLock.current = null; setState("loading"); setItems([]); setCursor(undefined); setMoreBusy(false); setMoreError(false);
    loadCompletedResponses({ search, current_only: false, limit: 25, sort: "COMPLETED_DESC" }).then((page) => { if (token === generation.current) { setItems(page.items); setCursor(page.next_cursor); setState("ready"); } }).catch(() => { if (token === generation.current) setState("error"); });
    return () => { generation.current++; };
  }, [search, retry]);
  async function more() {
    if (!cursor || moreBusy || moreLock.current !== null) return; const token = generation.current; moreLock.current = token; setMoreBusy(true); setMoreError(false);
    try { const page = await loadCompletedResponses({ search, current_only: false, limit: 25, sort: "COMPLETED_DESC", cursor }); if (token === generation.current) { setItems((value) => [...value, ...page.items]); setCursor(page.next_cursor); } }
    catch (error) { if (token === generation.current) { if (accessDenied(error)) { setItems([]); setCursor(undefined); setState("error"); } else setMoreError(true); } }
    finally { if (moreLock.current === token) moreLock.current = null; if (token === generation.current) setMoreBusy(false); }
  }
  return <section className="v-responses"><header className="v-page-heading"><div><p className="v-eyebrow">Forms</p><h1>Responses</h1></div></header>
    <nav className="v-local-nav" aria-label="Form views"><a href="#/forms">Forms</a><a href="#/forms/responses" aria-current="page">Responses</a></nav>
    <form className="v-program-search" onSubmit={(event) => { event.preventDefault(); setSearch(draft.trim()); }}><label htmlFor="v-response-search">Find a response</label><div><input id="v-response-search" value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Form title"/><button>Search</button></div></form>
    {state === "loading" && <p role="status">Loading responses…</p>}
    {state === "error" && <div className="v-panel"><p role="alert">Responses unavailable.</p><button onClick={() => setRetry(retry + 1)}>Try again</button></div>}
    {state === "ready" && <div className="v-panel v-response-list">{items.length === 0 ? <p>No responses match this search.</p> : <ul>{items.map((item) => <li key={item.id}><div><a href={`#/forms/responses/${encodeURIComponent(item.id)}`}>{item.title}</a><small>{submitter(item)} · {timestampLabel(item.completed_at)}</small></div><span className="v-status">{item.current ? "Current response" : "Earlier response"}</span></li>)}</ul>}{moreError && <p role="alert">More responses unavailable.</p>}{cursor && <button onClick={() => void more()} disabled={moreBusy}>{moreBusy ? "Loading…" : "Load more"}</button>}</div>}
  </section>;
}

type Details = { response: CompletedResponseDetail; review?: ResponseReviewPage };
function ResponseDetail({ id, reviewScope }: { id: string; reviewScope?: ReviewScope }) {
 const storageKey = reviewScope && reviewScope.tenant && reviewScope.entity && reviewScope.actor ? "clearsight:review:" + JSON.stringify([reviewScope.tenant,reviewScope.entity,reviewScope.actor,id]) : undefined;
 const [pending, setPending] = useState<PendingReview | null | undefined>(() => readPendingReview(storageKey));
  const [data, setData] = useState<Details>(); const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [attempt, setAttempt] = useState(0); const [decision, setDecision] = useState<"" | ResponseReviewDecision>(""); const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false); const [saveError, setSaveError] = useState<"" | "validation" | "conflict" | "forbidden" | "unknown" | "committed" | "storage">(""); const [saved, setSaved] = useState(false);
  const [historyBusy, setHistoryBusy] = useState(false); const [historyError, setHistoryError] = useState(false);
  const [documentsBusy, setDocumentsBusy] = useState(false); const [documentsError, setDocumentsError] = useState(false);
  const generation = useRef(0);
  const saveLock = useRef(false);
  const historyLock = useRef<number | null>(null);
  const documentsLock = useRef<number | null>(null);
  useEffect(() => {
    const token = ++generation.current; setState("loading"); setData(undefined); setSaveError(""); setSaved(false);
    historyLock.current = null; documentsLock.current = null;
    setHistoryBusy(false); setHistoryError(false); setDocumentsBusy(false); setDocumentsError(false);
    Promise.allSettled([loadCompletedResponse(id), loadResponseReviews(id)]).then(([response, review]) => {
      if (token !== generation.current) return;
      if (response.status === "rejected" || !matchesResponseScope(response.value, id)) { setState("error"); return; }
      setData({ response: response.value, review: review.status === "fulfilled" ? review.value : undefined }); setState("ready");
    });
    return () => { generation.current++; };
  }, [id, attempt]);
  async function save(event: FormEvent) {
    event.preventDefault();
    if (!storageKey || !data?.review || !data.response.answers || !data.response.documents || data.response.documents_next_cursor || !data.response.response.current || !data.response.revision.current || !data.review.may_review || !decision || busy || saveLock.current || historyLock.current !== null || pending !== undefined || (saveError && saveError !== "validation") || new TextEncoder().encode(comment.trim()).length > 4000 || (decision === "CHANGES_REQUESTED" && !comment.trim())) return;
    const input = { expected_version: data.review.version, decision, comment: comment.trim() };
    let current: PendingReview | undefined;
    if (storageKey) {
      try { current = {key: crypto.randomUUID(), revision: data.response.revision.revision, input}; sessionStorage.setItem(storageKey, JSON.stringify({key: current.key, revision: current.revision, input: {expected_version: input.expected_version, decision: input.decision}})); setPending(current); }
      catch { setSaveError("storage"); return; }
    }
    if (!current) return;
    await submitReview(input, current);
  }
  async function submitReview(input: RecordResponseReviewInput, current: PendingReview, isRetry = false) {
    if (!storageKey || !data?.review || !data.response.answers || !data.response.documents || data.response.documents_next_cursor || !data.review.may_review || busy || saveLock.current || historyLock.current !== null) return;
    const revision = current.revision;
    saveLock.current = true;
    const token = generation.current; setBusy(true); setSaved(false); setSaveError(""); let committed = false;
    try {
      const receipt = await recordResponseReview(id, input, current.key);
      if (!receipt || receipt.response_id !== id || !Number.isSafeInteger(revision) || revision < 1 || receipt.response_revision !== revision || !Number.isSafeInteger(input.expected_version) || input.expected_version < 0 || !Number.isSafeInteger(receipt.version) || receipt.version !== input.expected_version + 1 || receipt.decision !== input.decision || typeof receipt.id !== "string" || !receipt.id.trim()) throw new Error("Review receipt unavailable");
      committed = true;
      if (storageKey) { try { sessionStorage.removeItem(storageKey); } catch { /* The receipt remains valid. */ } }
      setPending(undefined);
      const updated = await loadResponseReviews(id);
      if (token === generation.current) { setData((value) => value ? { ...value, review: updated } : value); setDecision(""); setComment(""); setSaved(true); }
    } catch (error) { if (token === generation.current) { const kind = apiErrorKind(error); if (!committed && !isRetry && kind === "validation") { if (storageKey) { try { sessionStorage.removeItem(storageKey); } catch { /* The rejected attempt can remain unavailable. */ } } setPending(undefined); } setSaveError(committed ? "committed" : kind === "validation" ? "validation" : kind === "conflict" ? "conflict" : kind === "forbidden" ? "forbidden" : "unknown"); } }
    finally { saveLock.current = false; if (token === generation.current) setBusy(false); }
  }
  async function older() {
    const before = data?.review?.next_before_version; if (!before || historyBusy || historyLock.current !== null || saveLock.current) return;
    const token = generation.current; historyLock.current = token; setHistoryBusy(true); setHistoryError(false);
    try { const page = await loadResponseReviews(id, before); if (token === generation.current) setData((value) => value?.review ? { ...value, review: { ...value.review, items: [...value.review.items, ...page.items], next_before_version: page.next_before_version } } : value); }
    catch (error) { if (token === generation.current) { if (accessDenied(error)) setData((value) => value ? { ...value, review: undefined } : value); else setHistoryError(true); } }
    finally { if (historyLock.current === token) historyLock.current = null; if (token === generation.current) setHistoryBusy(false); }
  }
  async function moreDocuments() {
    const cursor = data?.response.documents_next_cursor; if (!cursor || documentsBusy || documentsLock.current !== null) return;
    const token = generation.current; documentsLock.current = token; setDocumentsBusy(true); setDocumentsError(false);
    try { const page = await loadDocuments({ response_revision_id: id, current_only: false, cursor, limit: 100 }); if (token === generation.current) setData((value) => value?.response.documents ? { ...value, response: { ...value.response, documents: [...value.response.documents, ...page.items], documents_next_cursor: page.next_cursor } } : value); }
    catch (error) { if (token === generation.current) { if (accessDenied(error)) setData((value) => value ? { ...value, response: { ...value.response, documents: undefined, documents_next_cursor: undefined } } : value); else setDocumentsError(true); } }
    finally { if (documentsLock.current === token) documentsLock.current = null; if (token === generation.current) setDocumentsBusy(false); }
  }
  if (state === "loading") return <p role="status">Loading response…</p>;
  if (state === "error" || !data) return <section className="v-responses"><div className="v-panel"><h1>Response unavailable</h1><button className="v-primary" onClick={() => setAttempt(attempt + 1)}>Try again</button>{documentReturnDestination() && <p><a href={documentReturnDestination()}>View documents</a></p>}</div></section>;
  const { response: { response, revision, answers, documents, documents_next_cursor: documentsNext }, review } = data;
  const isCurrent = response.current && revision.current;
  const canReview = Boolean(storageKey) && pending === undefined && isCurrent && answers && documents && !documentsNext && review?.may_review && (!saveError || saveError === "validation");
  const commentTooLong = new TextEncoder().encode(comment.trim()).length > 4000;
  const returnToDocuments = documentReturnDestination();
  const previousHref = revision.supersedes_revision_id ? previousResponseHref(revision.supersedes_revision_id, response.distribution_id) : undefined;
  const documentHref = scopedDocumentsHref("response", id, undefined, response.distribution_id || undefined);
  return <section className="v-responses"><a className="v-back" href={returnToDocuments ?? "#/forms/responses"}>{returnToDocuments ? "← Documents" : "← Responses"}</a><header className="v-page-heading"><div><p className="v-eyebrow">Response · Version {response.revision}</p><h1>{response.title}</h1></div><span className="v-status">{isCurrent ? "Current response" : "Earlier response"}</span></header>
    <div className="v-response-meta"><span>Submitted by <strong>{submitter(response)}</strong></span>{response.submission_recipient?.label && <span><span>Access recipient</span> <strong>{response.submission_recipient.label}</strong></span>}<span>Submitted <strong>{timestampLabel(response.completed_at)}</strong></span><span>Form version <strong>{response.form_template_version}</strong></span></div>
    {response.submission_recipient?.label && <p>Access recipient does not confirm who submitted the response.</p>}
    {revision.supersedes_revision_id && <p>{previousHref ? <a href={previousHref}>Previous response</a> : "Earlier response unavailable."}</p>}
    {documentHref && <p><a href={documentHref}>View response documents</a></p>}
    <section className="v-panel v-response-answers"><h2>Answers</h2>{!answers ? <p role="alert">Answers unavailable. Reload before reviewing.</p> : answers.length === 0 ? <p>No answers recorded.</p> : <dl>{answers.map(({ field_id, label, type, value }) => <div key={field_id}><dt>{label}</dt><dd>{answerLabel(type, value)}{value?.document && <span className="v-answer-document">{value.document.document_type && <span>Type: {value.document.document_type}</span>}{value.document.reference && <span>Reference: {value.document.reference}</span>}{value.document.issued_by && <span>Issued by: {value.document.issued_by}</span>}{value.document.issued_on && <span>Issued: {dateLabel(value.document.issued_on)}</span>}{value.document.expires_on && <span>Expires: {dateLabel(value.document.expires_on)}</span>}</span>}</dd></div>)}</dl>}</section>
    <section className="v-panel v-response-documents"><h2>Documents</h2>{!documents ? <p role="alert">Documents unavailable. Reload before reviewing.</p> : documents.length === 0 && !documentsNext ? <p>No documents submitted.</p> : <ul>{documents.map((document) => <li key={document.id}><span>{document.file_name}</span>{document.artifact_status === "AVAILABLE" || (document.artifact_status === "STORED_UNSCANNED" && document.demo_unscanned_allowed) ? <a href={documentContentURL(document, true)}>Download</a> : <small>File unavailable</small>}</li>)}</ul>}{documentsNext && <><p>More documents must be loaded before review.</p>{documentsError && <p role="alert">Additional documents unavailable.</p>}<button disabled={documentsBusy} onClick={() => void moreDocuments()}>{documentsBusy ? "Loading…" : "Load more documents"}</button></>}</section>
    <section className="v-panel v-response-review"><h2>Review</h2>{!review ? <><p role="alert">Review unavailable.</p><button onClick={() => setAttempt(attempt + 1)}>Reload response</button></> : <>
      {!isCurrent && <p>Review is available for the current response.</p>}
      {isCurrent && (!review.may_review || !storageKey) && <p>Review unavailable.</p>}
      {saveError && !(pending && saveError === "unknown") && <p role="alert">{{ validation: "Review not saved. Check the decision and comment.", conflict: "The response or review changed. Reload before saving.", forbidden: "You cannot review this response. Reload to check access.", unknown: "Review status unknown. Reload before trying again.", committed: "Review saved. Reload to see review history.", storage: "Review not saved. Browser storage is unavailable." }[saveError]}</p>}
      {saved && <p role="status">Review saved.</p>}
      {pending === null && <p role="alert">Review status unavailable. Check review history before saving another decision.</p>}{pending && <><p role={busy ? "status" : "alert"}>{busy ? "Saving review…" : "Review status unknown. Retrying will submit the same decision."}</p>{!pending.needsComment && pending.input.comment !== "" && <label>{pending.input.decision === "CHANGES_REQUESTED" ? "Reason for changes" : "Comment (optional)"}<textarea readOnly value={pending.input.comment}/></label>}{pending.needsComment && <label>Original comment<textarea value={comment} disabled={busy} onChange={(event) => setComment(event.target.value)}/></label>}<button disabled={busy || saveError === "forbidden" || !review.may_review || (pending.needsComment && pending.input.decision === "CHANGES_REQUESTED" && !comment.trim()) || (pending.needsComment && new TextEncoder().encode(comment.trim()).length > 4000)} onClick={() => void submitReview(pending.needsComment ? {...pending.input, comment: comment.trim()} : pending.input,pending,true)}>{busy ? "Saving…" : "Retry review"}</button></>}
      {canReview && <form onSubmit={(event) => void save(event)}><label>Review decision<select required disabled={busy} value={decision} onChange={(event) => setDecision(event.target.value as typeof decision)}><option value="">Select decision</option><option value="REVIEWED">Reviewed</option><option value="CHANGES_REQUESTED">Request changes</option></select></label><label>{decision === "CHANGES_REQUESTED" ? "Reason for changes" : "Comment (optional)"}<textarea disabled={busy} required={decision === "CHANGES_REQUESTED"} value={comment} onChange={(event) => setComment(event.target.value)}/></label>{commentTooLong && <p role="alert">Comment must be 4,000 bytes or fewer.</p>}{decision === "CHANGES_REQUESTED" && <p>The respondent is not notified automatically.</p>}<button className="v-primary" disabled={busy || historyBusy || commentTooLong}>{busy ? "Saving…" : "Save review"}</button></form>}
      {((saveError && saveError !== "validation") || !answers || !documents) && <button onClick={() => setAttempt(attempt + 1)}>Reload response</button>}
      <h3>Review history</h3>{review.items.length === 0 ? <p>No reviews recorded.</p> : <ul className="v-review-history">{review.items.map((item) => <li key={item.id}><strong>{item.decision === "REVIEWED" ? "Reviewed" : "Changes requested"}</strong><span>{item.reviewer_display_name || "Reviewer name unavailable"} · {timestampLabel(item.reviewed_at)}</span>{item.comment && <p>{item.comment}</p>}</li>)}</ul>}{historyError && <p role="alert">Earlier reviews unavailable.</p>}{review.next_before_version && <button disabled={busy || historyBusy} onClick={() => void older()}>{historyBusy ? "Loading…" : "Load earlier reviews"}</button>}
    </>}</section>
  </section>;
}

function readPendingReview(key?: string): PendingReview | null | undefined {
 if (!key) return undefined;
 try { const raw = sessionStorage.getItem(key); if (raw === null) return undefined; const value: unknown = JSON.parse(raw);
  if (!value || typeof value !== "object") return null;
  const item = value as PendingReview; const input = item.input;
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(item.key) || !Number.isSafeInteger(item.revision) || item.revision < 1 || !input || !Number.isSafeInteger(input.expected_version) || input.expected_version < 0 || (input.decision !== "REVIEWED" && input.decision !== "CHANGES_REQUESTED") || (input as Partial<RecordResponseReviewInput>).comment !== undefined) return null;
  return {...item, input: {...input, comment: ""}, needsComment: true};
 } catch { return null; }
}
function submitter(item: CompletedResponseSummary) { return item.submitted_by_display_name || (item.submitted_by ? "Submitter name unavailable" : "Submitter not identified"); }
function matchesResponseScope(value: CompletedResponseDetail, id: string) {
  const params = new URLSearchParams(window.location.hash.split("?")[1] ?? "");
  const expected = params.getAll("distribution_id");
  return value?.response?.id === id && value?.revision?.id === id && (expected.length === 0 || (expected.length === 1 && expected[0] !== "" && value.response.distribution_id === expected[0]));
}
function previousResponseHref(id: string, distribution: string | undefined) {
  if (!validHistoryIdentifier(id) || !validHistoryIdentifier(distribution)) return undefined;
  const params = new URLSearchParams();
  params.set("distribution_id", distribution);
  const destination = documentReturnDestination();
  if (destination) params.set("return_to", destination);
  const query = params.toString();
  try { return `#/forms/responses/${encodeURIComponent(id)}?${query}`; } catch { return undefined; }
}
function validHistoryIdentifier(value: unknown): value is string { return typeof value === "string" && value.length > 0 && value.length <= 200 && value.trim() === value && value !== "*" && !/[\u0000-\u001f\u007f]/.test(value); }
function accessDenied(error: unknown) { return ["unauthorized", "forbidden"].includes(apiErrorKind(error)); }
function timestampLabel(value: string) {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.exec(value);
  if (!match) return "time unavailable";
  const year = Number(match[1]), month = Number(match[2]), day = Number(match[3]);
  const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  if (month < 1 || month > 12 || day < 1 || day > days[month - 1]! || Number(match[4]) > 23 || Number(match[5]) > 59 || Number(match[6]) > 59) return "time unavailable";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "time unavailable" : `${new Intl.DateTimeFormat("en-GB", { day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23", timeZone: "UTC" }).format(date)} UTC`;
}
function dateLabel(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "date unavailable" : new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(date); }
function answerLabel(type?: string, answer?: CaptureAnswerValue) {
  if (!answer) return "No answer";
  const fieldType = type?.trim().toLowerCase();
  const text = answer.text || answer.values?.[0];
  if ((fieldType === "checkbox" || fieldType === "attestation") && text) {
    const confirmed = text.trim().toLowerCase();
    if (confirmed === "yes" || confirmed === "true") return "Confirmed";
    if (confirmed === "no" || confirmed === "false") return "Not confirmed";
  }
  if (answer.text) {
    if (fieldType === "date" && /^\d{4}-\d{2}-\d{2}$/.test(answer.text)) {
      const date = new Date(`${answer.text}T00:00:00Z`);
      if (!Number.isNaN(date.getTime()) && date.toISOString().slice(0, 10) === answer.text) return dateLabel(answer.text);
    }
    if (fieldType === "percentage" && /^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:e[+-]?\d+)?$/i.test(answer.text.trim())) return `${answer.text}%`;
    return answer.text;
  }
  if (answer.values?.length) return answer.values.join(", ");
  const count = answer.artifact_ids?.length || (answer.document?.artifact_id ? 1 : 0);
  if (!count) return "No answer";
  if (fieldType === "signature") return "Signature submitted";
  if (fieldType === "photo") return "Photo submitted";
  return `${count} ${count === 1 ? "document" : "documents"} submitted`;
}
