import { useEffect, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import type { ActorWorkScope } from "../api";
import type { CompletedResponseSummary } from "../formsDistributionApi";
import { loadProgramResponseReviews, type ProgramResponseReviewPage } from "./programResponseReviewsApi";

type State = { kind: "loading" | "ready" | "error" | "unavailable"; items: CompletedResponseSummary[]; cursor?: string; busy: boolean; moreError: boolean };
const initial = (kind: State["kind"]): State => ({ kind, items: [], busy: false, moreError: false });

function valid(page: ProgramResponseReviewPage, scope: ActorWorkScope) {
  return page?.coverage === "PROGRAM_RESPONSE_REVIEWS" && page.scope?.tenant_id === scope.tenantID && page.scope?.legal_entity_id === scope.legalEntityID && page.scope?.principal_id === scope.principalID && (page.next_cursor === undefined || typeof page.next_cursor === "string") && Array.isArray(page.items) && page.items.every((item) => typeof item.id === "string" && item.id.length > 0 && item.id.length <= 200 && typeof item.title === "string" && item.subject_type === "PROGRAM" && item.current === true && (item.state === "FINAL" || item.state === "PROVISIONAL") && Number.isSafeInteger(item.revision) && item.revision > 0);
}

export function ProgramResponseReviews({ scope }: { scope: ActorWorkScope }) {
  const { tenantID, legalEntityID, principalID } = scope;
  const [state, setState] = useState<State>(() => initial("loading"));
  const [attempt, setAttempt] = useState(0);
  const generation = useRef(0); const pending = useRef(false);
  useEffect(() => {
    const controller = new AbortController(); const token = ++generation.current;
    pending.current = false; setState(initial("loading"));
    if (!tenantID || !legalEntityID || legalEntityID === "*" || !principalID) { setState(initial("error")); return () => { controller.abort(); generation.current++; }; }
    loadProgramResponseReviews(undefined, controller.signal).then((page) => {
      if (token !== generation.current) return;
      if (!valid(page, { tenantID, legalEntityID, principalID })) { setState(initial("error")); return; }
      setState({ ...initial("ready"), items: page.items, cursor: page.next_cursor || undefined });
    }).catch((error) => { if (token === generation.current && !controller.signal.aborted) setState(initial(apiErrorKind(error) === "not_found" ? "unavailable" : "error")); });
    return () => { controller.abort(); generation.current++; };
  }, [tenantID, legalEntityID, principalID, attempt]);
  async function more() {
    if (!state.cursor || state.busy || pending.current) return;
    const token = generation.current; const cursor = state.cursor; pending.current = true;
    setState((value) => ({ ...value, busy: true, moreError: false }));
    try {
      const page = await loadProgramResponseReviews(cursor);
      if (token !== generation.current) return;
      if (!valid(page, { tenantID, legalEntityID, principalID }) || page.next_cursor === cursor) { setState(initial("error")); return; }
      setState((value) => { const seen = new Set(value.items.map((item) => item.id)); return { ...value, items: [...value.items, ...page.items.filter((item) => !seen.has(item.id))], cursor: page.next_cursor || undefined, busy: false }; });
    } catch (error) {
      if (token !== generation.current) return;
      if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) setState(initial("error"));
      else setState((value) => ({ ...value, busy: false, moreError: true }));
    } finally { if (token === generation.current) pending.current = false; }
  }
  return <section className="v-panel v-work-list v-program-reviews" aria-labelledby="program-review-title"><div className="v-work-list-heading"><h2 id="program-review-title">Program response reviews</h2></div>
    {state.kind === "loading" && <p role="status">Loading reviews…</p>}
    {state.kind === "unavailable" && <p>Program reviews unavailable.</p>}
    {state.kind === "error" && <><p role="alert">Program reviews unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></>}
    {state.kind === "ready" && <>
      {state.items.length ? <ul>{state.items.map((item) => <li key={item.id}><div><strong>{item.title}</strong><span>Submitted by {item.submitted_by_display_name || "name unavailable"}</span><span>{submissionDate(item.completed_at)} · Response version {item.revision}</span></div><a href={`#/forms/responses/${encodeURIComponent(item.id)}`} aria-label={`Review ${item.title}`}>Review response</a></li>)}</ul> : <p className="v-work-empty">{state.cursor ? "No reviews on this page." : "No Program responses await your review."}</p>}
      {state.moreError && <p role="alert">More reviews unavailable. Try again.</p>}
      {state.cursor && <button type="button" className="v-work-more" disabled={state.busy} onClick={() => void more()}>{state.busy ? "Loading…" : "Load more reviews"}</button>}
    </>}
  </section>;
}

function submissionDate(value: string) {
  if (typeof value !== "string" || !value.trim()) return "Submission date unavailable";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Submission date unavailable" : `Submitted ${new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date)}`;
}
