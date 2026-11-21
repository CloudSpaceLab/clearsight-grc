import { useEffect, useRef, useState } from "react";
import { ApiError } from "../../http";
import { loadResponseReviews, recordResponseReview, type ResponseReviewDecision as Decision, type ResponseReviewPage } from "../../responseReviewApi";
import { Button, Notice, TextArea } from "../ui";

export function ResponseReviewDecision({ responseID, current, onUpdated }: { responseID: string; current?: boolean | null; onUpdated?: () => void }) {
  const [page, setPage] = useState<ResponseReviewPage>();
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [draft, setDraft] = useState("");
  const [requestingChanges, setRequestingChanges] = useState(false);
  const [busy, setBusy] = useState(false);
  const [conflict, setConflict] = useState(false);
  const sequence = useRef(0);

  async function reload() {
    const request = ++sequence.current;
    setBusy(true);
    try {
      const result = await loadResponseReviews(responseID);
      if (request !== sequence.current) return;
      setPage(result); setError(undefined); setConflict(false);
    } catch {
      if (request !== sequence.current) return;
      setPage(undefined); setError("Review history unavailable.");
    } finally { if (request === sequence.current) setBusy(false); }
  }
  useEffect(() => { void reload(); return () => { sequence.current++; }; }, [responseID]);

  async function save(decision: Decision) {
    if (!page || !page.may_review || current !== true || busy || conflict || (decision === "CHANGES_REQUESTED" && !draft.trim())) return;
    const request = sequence.current;
    setBusy(true); setError(undefined); setNotice(undefined);
    try {
      await recordResponseReview(responseID, { expected_version: page.version, decision, comment: decision === "CHANGES_REQUESTED" ? draft.trim() : "" });
    } catch (cause) {
      if (request !== sequence.current) return;
      if (cause instanceof ApiError && cause.kind === "conflict") { setConflict(true); setError("Review changed. Reload before saving."); }
      else if (cause instanceof ApiError && (cause.kind === "forbidden" || cause.kind === "unauthorized")) { setPage(undefined); setError("Review permission unavailable."); }
      else setError("Review could not be saved. Try again.");
      setBusy(false); return;
    }
    if (request !== sequence.current) return;
    setDraft(""); setRequestingChanges(false);
    try { onUpdated?.(); } catch { /* The saved review remains valid. */ }
    try {
      const result = await loadResponseReviews(responseID);
      if (request !== sequence.current) return;
      setPage(result); setNotice("Review saved.");
    } catch {
      if (request !== sequence.current) return;
      setPage(undefined); setError("Review saved. History could not be loaded.");
    } finally { if (request === sequence.current) setBusy(false); }
  }

  async function loadMore() {
    if (!page?.next_before_version || busy) return;
    const request = sequence.current;
    setBusy(true); setError(undefined);
    try {
      const older = await loadResponseReviews(responseID, page.next_before_version);
      if (request !== sequence.current) return;
      if (older.version !== page.version) { setConflict(true); setError("Review changed. Reload before continuing."); return; }
      setPage({ ...older, items: [...page.items, ...older.items] });
    } catch { if (request === sequence.current) setError("Older reviews unavailable. Try again."); }
    finally { if (request === sequence.current) setBusy(false); }
  }

  const mayReview = page?.may_review === true && current === true;
  return <section className="response-review-decision" aria-label="Response review">
    <h3>Response review</h3>
    {!page && !error && <p role="status">Loading review…</p>}
    {error && <Notice tone="error">{error}</Notice>}
    {notice && <Notice tone="success">{notice}</Notice>}
    {(!page || conflict) && error && <Button isDisabled={busy} onPress={() => void reload()}>Reload review</Button>}
    {page && <>
      {current === false && <p>Historical response</p>}
      {current === null && <p>Response currency unavailable.</p>}
      {current === true && !page.may_review && <p>Review permission unavailable.</p>}
      {page.items.length === 0 ? <p>No review recorded</p> : <ol className="response-review-decision__history">{page.items.map((review) => <li key={review.id}><strong>{review.decision === "REVIEWED" ? "Reviewed" : "Changes needed"}</strong><span>{review.reviewer_display_name || "Reviewer name unavailable"} · <time dateTime={review.reviewed_at}>{formatTime(review.reviewed_at)}</time></span>{review.comment && <p>{review.comment}</p>}</li>)}</ol>}
      {page.next_before_version && <Button isDisabled={busy || conflict} onPress={() => void loadMore()}>Load older reviews</Button>}
      {mayReview && !requestingChanges && <div className="response-review-decision__actions"><Button variant="primary" isDisabled={busy || conflict} onPress={() => void save("REVIEWED")}>Mark reviewed</Button><Button isDisabled={busy || conflict} onPress={() => setRequestingChanges(true)}>Record changes needed</Button></div>}
      {mayReview && requestingChanges && <><Notice tone="info">The respondent is not notified.</Notice><TextArea label="Changes needed" value={draft} onChange={setDraft} maxLength={4000} rows={3} isDisabled={busy}/><div className="response-review-decision__actions"><Button variant="primary" isDisabled={!draft.trim() || busy || conflict} onPress={() => void save("CHANGES_REQUESTED")}>Save decision</Button><Button isDisabled={busy} onPress={() => setRequestingChanges(false)}>Cancel</Button></div></>}
    </>}
  </section>;
}

function formatTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Time unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
