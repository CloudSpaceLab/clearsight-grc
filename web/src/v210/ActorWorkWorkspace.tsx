import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadActorWorkDueCounts, loadActorWorkPage, type ActorWorkDueCounts, type ActorWorkPage, type RuntimeContext } from "../api";
import { validWorkCounts, validWorkPage, validWorkTimestamp } from "../workSourceReceipt";
import { apiErrorKind } from "../http";
import type { AttentionItem } from "../types";
import { FormApprovals } from "./FormApprovals";
import { ProgramResponseReviews } from "./ProgramResponseReviews";
import "./actor-work.css";

type Result = { key: string; state: "loading" | "ready" | "unavailable"; items: AttentionItem[]; asOf?: string; cursor?: string; hasMore: boolean; moreBusy: boolean; moreError: boolean };
const initial = (key: string, state: Result["state"] = "loading"): Result => ({ key, state, items: [], hasMore: false, moreBusy: false, moreError: false });

export function ActorWorkWorkspace({ context }: { context: RuntimeContext }) {
  const { id: tenantID } = context.tenant, { id: legalEntityID } = context.legal_entity, { id: principalID } = context.actor;
  const scope = { tenantID, legalEntityID, principalID };
  const [draft, setDraft] = useState(""); const [search, setSearch] = useState(""); const [retry, setRetry] = useState(0);
  const [view, setView] = useState<"work" | "forms">("work");
  const key = JSON.stringify([tenantID, legalEntityID, principalID, search, retry]);
  const [result, setResult] = useState<Result>(() => initial(key));
  const scopeKey = JSON.stringify([tenantID, legalEntityID, principalID]);
  const [counts, setCounts] = useState<{ key: string; value?: ActorWorkDueCounts; state: "loading" | "ready" | "unavailable" }>({ key: scopeKey, state: "loading" });
  const visibleCounts = counts.key === scopeKey ? counts : { key: scopeKey, state: "loading" as const };
  const generation = useRef(0); const moreLock = useRef(false);
  const visible = result.key === key ? result : initial(key);
  useEffect(() => {
    let active = true; setCounts({ key: scopeKey, state: "loading" });
    if (!tenantID || !legalEntityID || legalEntityID === "*" || !principalID) { setCounts({ key: scopeKey, state: "unavailable" }); return () => { active = false; }; }
    void loadActorWorkDueCounts().then((value) => { if (active) setCounts(validWorkCounts(value, { tenantID, legalEntityID, principalID }) ? { key: scopeKey, value, state: "ready" } : { key: scopeKey, state: "unavailable" }); }).catch(() => { if (active) setCounts({ key: scopeKey, state: "unavailable" }); });
    return () => { active = false; };
  }, [scopeKey, tenantID, legalEntityID, principalID, retry]);
  useEffect(() => {
    const current = ++generation.current; moreLock.current = false; setResult(initial(key));
    if (!tenantID || !legalEntityID || legalEntityID === "*" || !principalID) { setResult(initial(key, "unavailable")); return () => { generation.current++; }; }
    void loadActorWorkPage(search).then((page) => {
      if (generation.current !== current) return;
      if (!validWorkPage(page, { tenantID, legalEntityID, principalID })) { setResult(initial(key, "unavailable")); return; }
      setResult({ key, state: "ready", items: page.items, asOf: page.as_of, cursor: page.next_cursor, hasMore: page.has_more, moreBusy: false, moreError: false });
    }).catch(() => { if (generation.current === current) setResult(initial(key, "unavailable")); });
    return () => { generation.current++; };
  }, [key, tenantID, legalEntityID, principalID, search]);

  async function loadMore() {
    if (moreLock.current || visible.state !== "ready" || !visible.hasMore || !visible.cursor) return;
    moreLock.current = true; const current = generation.current; const cursor = visible.cursor; setResult((value) => ({ ...value, moreBusy: true, moreError: false }));
    try {
      const page: ActorWorkPage = await loadActorWorkPage(search, cursor);
      if (generation.current !== current) return;
      if (!validWorkPage(page, scope) || page.as_of !== visible.asOf || page.next_cursor === cursor) { setResult(initial(key, "unavailable")); return; }
      setResult((value) => { const seen = new Set(value.items.map((item) => item.id)); return { ...value, items: [...value.items, ...page.items.filter((item) => !seen.has(item.id))], cursor: page.next_cursor, hasMore: page.has_more, moreBusy: false }; });
    } catch (error) {
      if (generation.current !== current) return;
      if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) setResult(initial(key, "unavailable"));
      else setResult((value) => ({ ...value, moreBusy: false, moreError: true }));
    } finally { if (generation.current === current) moreLock.current = false; }
  }
  function submit(event: FormEvent) { event.preventDefault(); setSearch(draft.trim()); }
  return <section className="v-actor-work"><header className="v-page-heading"><div><p className="v-eyebrow">My work</p><h1>Work &amp; approvals</h1></div></header>
    <div className="v-work-tabs" role="group" aria-label="Work views"><button type="button" aria-pressed={view === "work"} onClick={() => setView("work")}>Open work</button><button type="button" aria-pressed={view === "forms"} onClick={() => setView("forms")}>Approvals &amp; reviews</button></div>
    {view === "forms" ? <div className="v-approval-sections" key={JSON.stringify([tenantID, legalEntityID, principalID])}><ProgramResponseReviews scope={scope}/><FormApprovals/></div> : <>
    {visibleCounts.state === "ready" && visibleCounts.value && <section className="v-work-counts" aria-label="Workflow assignment counts"><div className="v-work-count-items"><div><span>Open assignments</span><strong>{visibleCounts.value.total}</strong></div><div><span>Overdue</span><strong className={visibleCounts.value.overdue > 0 ? "is-overdue" : undefined}>{visibleCounts.value.overdue}</strong></div><div><span>Due in 7 days</span><strong className={visibleCounts.value.due_soon > 0 ? "is-due" : undefined}>{visibleCounts.value.due_soon}</strong></div><div><span>No due date</span><strong>{visibleCounts.value.undated}</strong></div></div><p>All workflow assignments · As of {formatDateTime(visibleCounts.value.as_of)}</p></section>}
    {visibleCounts.state === "unavailable" && <p className="v-work-counts-unavailable">Assignment counts unavailable.</p>}
    <form role="search" className="v-work-search" onSubmit={submit}><label htmlFor="work-search">Search work</label><div><input id="work-search" value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Work item or reference"/><button type="submit">Search</button></div></form>
    {visible.state === "loading" && <p role="status">Loading work…</p>}
    {visible.state === "unavailable" && <div className="v-panel"><p role="alert">Work unavailable.</p><button onClick={() => setRetry((value) => value + 1)}>Try again</button></div>}
    {visible.state === "ready" && <div className="v-panel v-work-list v-actor-assignment-list"><div className="v-work-list-heading"><h2>Open work</h2><p>Workflow assignments · As of {formatDateTime(visible.asOf)}</p></div>{visible.items.length === 0 ? <p className="v-work-empty">{search ? "No work matches the search." : "No open workflow assignments."}</p> : <div className="v-program-table v-work-table"><table><thead><tr><th>Work item</th><th>Type</th><th>Owner</th><th>Due</th><th>State</th><th><span className="v-work-sr-only">Action</span></th></tr></thead><tbody>{visible.items.map((item) => { const href = matterWorkHref(item); const urgency = dueUrgency(item.due_at, visible.asOf); return <tr key={item.id}><td>{href ? <a href={href}>{item.title}</a> : <strong>{item.title}</strong>}</td><td><span className="v-mobile-label">Type</span>{human(item.type)}</td><td><span className="v-mobile-label">Owner</span>{item.owner || "Owner unknown"}</td><td><span className="v-mobile-label">Due</span>{urgency && <span className={`v-work-due is-${urgency === "Overdue" ? "overdue" : "soon"}`}>{urgency}</span>}<small>{item.due_at ? formatDate(item.due_at) : "No due date"}</small></td><td><span className="v-mobile-label">State</span><span className="v-work-state">{human(item.state)}</span></td><td>{href && <a className="v-work-open" href={href} aria-label={`Open ${item.title}`}>Open <span aria-hidden="true">→</span></a>}</td></tr>; })}</tbody></table></div>}{visible.moreError && <p role="alert">More work unavailable. Try again.</p>}{visible.hasMore && <button className="v-work-more" type="button" disabled={visible.moreBusy} onClick={() => void loadMore()}>{visible.moreBusy ? "Loading…" : "Load more work"}</button>}</div>}
    </>}
  </section>;
}
function human(value?: string) { if (!value) return "Unknown"; return value.toLowerCase().replaceAll("_", " ").replace(/^./, (letter) => letter.toUpperCase()); }
function formatDate(value?: string) { if (!value || !Number.isFinite(Date.parse(value))) return "unknown"; return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(new Date(value)); }
function formatDateTime(value?: string) { if (!value || !validWorkTimestamp(value)) return "unknown"; return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(new Date(value)) + " UTC"; }
function dueUrgency(due?: string, asOf?: string) { if (!due || !asOf || !validWorkTimestamp(due) || !validWorkTimestamp(asOf)) return undefined; const remaining = Date.parse(due) - Date.parse(asOf); return remaining < 0 ? "Overdue" : remaining <= 7 * 86_400_000 ? "Due soon" : undefined; }
function matterWorkHref(item: AttentionItem) {
  const id = item.action_target_id;
  if (item.action_target_type !== "MATTER" || typeof id !== "string" || !id.trim() || id.length > 200 || /[\u0000-\u001f\u007f]/.test(id)) return undefined;
  try { return `#/work/issues/${encodeURIComponent(id)}`; } catch { return undefined; }
}
