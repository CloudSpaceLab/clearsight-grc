import { useEffect, useRef, useState } from "react";
import { loadCompletedResponses, type CompletedResponseSummary } from "../formsDistributionApi";
import { ApiError } from "../http";
import "./matter-responses.css";

export function MatterResponses({ matterID }: { matterID: string }) {
  const [items, setItems] = useState<CompletedResponseSummary[]>([]); const [cursor, setCursor] = useState<string>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading"); const [moreError, setMoreError] = useState(false); const [busy, setBusy] = useState(false); const [attempt, setAttempt] = useState(0);
  const generation = useRef(0); const lock = useRef<number | null>(null); const cursors = useRef(new Set<string>());
  const query = { subject_type: "MATTER" as const, subject_id: matterID, current_only: true, sort: "COMPLETED_DESC" as const, limit: 20 };
  useEffect(() => {
    const token = ++generation.current; cursors.current = new Set(); lock.current = null; setState("loading"); setItems([]); setCursor(undefined); setMoreError(false); setBusy(false);
    loadCompletedResponses(query).then((page) => {
      if (token !== generation.current) return;
      if (!valid(page.items, matterID)) throw new Error("Invalid response scope");
      if (page.next_cursor) cursors.current.add(page.next_cursor);
      setItems(deduplicate(page.items)); setCursor(page.next_cursor); setState("ready");
    }).catch(() => { if (token === generation.current) { setItems([]); setState("error"); } });
    return () => { generation.current++; };
  }, [matterID, attempt]);
  async function more() {
    if (!cursor || lock.current !== null) return;
    const token = generation.current; lock.current = token; setBusy(true); setMoreError(false);
    try {
      const page = await loadCompletedResponses({ ...query, cursor });
      if (token !== generation.current) return;
      if (!valid(page.items, matterID)) { setItems([]); setCursor(undefined); setState("error"); return; }
      if (page.next_cursor && cursors.current.has(page.next_cursor)) { setCursor(undefined); setMoreError(true); return; }
      if (page.next_cursor) cursors.current.add(page.next_cursor);
      setItems((previous) => deduplicate([...previous, ...page.items])); setCursor(page.next_cursor);
    } catch (error) {
      if (token !== generation.current) return;
      if (error instanceof ApiError && (error.status === 401 || error.status === 403)) { setItems([]); setCursor(undefined); setState("error"); }
      else setMoreError(true);
    } finally { if (lock.current === token) lock.current = null; if (token === generation.current) setBusy(false); }
  }
  return <section className="v-panel"><h2>Form responses</h2>{state === "loading" && <p role="status">Loading responses…</p>}{state === "error" && <><p role="alert">Responses unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></>}{state === "ready" && <>{items.length === 0 ? <p>No current responses available to you for this issue.</p> : <ul className="v-matter-responses">{items.map((item) => <li key={item.id}><strong>{item.title}</strong><p>{item.submitted_by_display_name || "Respondent name unavailable"} · {date(item.completed_at)}</p><a href={`#/forms/responses/${encodeURIComponent(item.id)}`}>View {item.title}</a></li>)}</ul>}{moreError && <p role="alert">More responses unavailable.</p>}{moreError && !cursor && <button type="button" onClick={() => setAttempt((value) => value + 1)}>Reload responses</button>}{cursor && <button type="button" disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : moreError ? "Retry responses" : "Load more responses"}</button>}</>}</section>;
}

function valid(items: CompletedResponseSummary[], matterID: string) { return Array.isArray(items) && items.every((item) => item && typeof item.id === "string" && Boolean(item.id) && item.subject_type === "MATTER" && item.subject_id === matterID && item.current === true && typeof item.title === "string" && Boolean(item.title)); }
function deduplicate(items: CompletedResponseSummary[]) { const ids = new Set<string>(); return items.filter((item) => { if (ids.has(item.id)) return false; ids.add(item.id); return true; }); }
function date(value: string) { const parsed = new Date(value); return Number.isNaN(parsed.getTime()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(parsed); }
