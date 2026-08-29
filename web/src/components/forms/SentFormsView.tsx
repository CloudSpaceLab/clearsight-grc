import { useEffect, useMemo, useState } from "react";
import { loadDistribution, loadDistributionPage, transitionDistribution, type Distribution, type DistributionDetail, type DistributionDueState, type DistributionQuery, type DistributionStatus } from "../../formsDistributionApi";
import { DistributionComposer } from "./DistributionComposer";
import { DistributionChangePanel } from "./DistributionChangePanel";

const statusOptions: DistributionStatus[] = ["OPEN", "LOCKED", "COMPLETED", "EXPIRED", "REVOKED", "SUPERSEDED"];
const dueOptions: DistributionDueState[] = ["OPEN", "OVERDUE", "CLOSED"];

export function SentFormsView() {
  const [query, setQuery] = useState<DistributionQuery>(() => readQuery());
  const [items, setItems] = useState<Distribution[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [selectedID, setSelectedID] = useState<string>();
  const [detail, setDetail] = useState<DistributionDetail>();
  const [composerOpen, setComposerOpen] = useState(false);
  const [changeMode, setChangeMode] = useState<"amend" | "supersede">();
  const [busy, setBusy] = useState<string>();
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();

  useEffect(() => { void refresh(); }, [query.status, query.due_state, query.subject_type, query.subject_id, query.owner, query.limit]);
  useEffect(() => { if (selectedID) void loadDetail(selectedID); else setDetail(undefined); }, [selectedID]);

  async function refresh() {
    setError(undefined);
    try {
      const page = await loadDistributionPage({ ...query, cursor: undefined, limit: query.limit ?? 25 });
      setItems(page.items); setNextCursor(page.next_cursor);
      setSelectedID((current) => current && page.items.some((value) => value.id === current) ? current : page.items[0]?.id);
    } catch (cause) {
      setItems([]); setNextCursor(undefined); setError(message(cause, "Sent forms could not be loaded."));
    }
  }
  async function loadDetail(id: string) {
    try { setDetail(await loadDistribution(id)); } catch (cause) { setDetail(undefined); setError(message(cause, "Distribution detail could not be loaded.")); }
  }
  async function loadMore() {
    if (!nextCursor || busy) return;
    setBusy("more");
    try {
      const page = await loadDistributionPage({ ...query, cursor: nextCursor, limit: query.limit ?? 25 });
      setItems((current) => [...current, ...page.items]); setNextCursor(page.next_cursor);
    } catch (cause) { setError(message(cause, "More distributions could not be loaded.")); } finally { setBusy(undefined); }
  }
  function updateQuery(patch: Partial<DistributionQuery>) {
    const next = { ...query, ...patch, cursor: undefined };
    setQuery(next); writeQuery(next);
  }
  async function lifecycle(action: "lock" | "reopen" | "revoke") {
    if (!detail || busy) return;
    setBusy(action); setError(undefined);
    try {
      const updated = await transitionDistribution(detail.distribution.id, detail.distribution.version, action);
      setDetail(updated);
      setItems((current) => current.map((value) => value.id === updated.distribution.id ? updated.distribution : value));
    } catch (cause) { setError(message(cause, "Distribution state could not be changed.")); } finally { setBusy(undefined); }
  }

  const counts = useMemo(() => detail ? {
    to: detail.recipients.filter((value) => value.role === "TO" && value.state !== "REVOKED").length,
    cc: detail.recipients.filter((value) => value.role === "CC" && value.state !== "REVOKED").length,
    completed: detail.recipients.filter((value) => value.role === "TO" && value.state === "COMPLETED").length,
  } : undefined, [detail]);

  if (composerOpen) return <DistributionComposer onCancel={() => setComposerOpen(false)} onCreated={(value) => { setComposerOpen(false); setItems((current) => [value.distribution, ...current]); setSelectedID(value.distribution.id); setDetail(value); }}/>
  if (changeMode && detail) return <DistributionChangePanel mode={changeMode} detail={detail} onCancel={() => setChangeMode(undefined)} onSaved={(value, resultNotice) => { setChangeMode(undefined); setItems((current) => [value.distribution, ...current.filter((item) => item.id !== detail.distribution.id && item.id !== value.distribution.id)]); setSelectedID(value.distribution.id); setDetail(value); setNotice(resultNotice); }}/>

  return <section className="forms-task-shell" aria-labelledby="sent-forms-title">
    <div className="forms-task-heading"><div><span>Sender workspace</span><h2 id="sent-forms-title">Sent forms</h2><p>Track each sent form, its recipients, response status, access method and deadline.</p></div><button className="forms-primary" type="button" onClick={() => setComposerOpen(true)}>Send form</button></div>
    {error && <div className="forms-message error" role="alert">{error}</div>}
    {notice && <div className="forms-message" role="status">{notice}</div>}
    <div className="forms-task-toolbar">
      <label><span>Status</span><select value={query.status ?? ""} onChange={(event) => updateQuery({ status: event.target.value ? event.target.value as DistributionStatus : undefined })}><option value="">All states</option>{statusOptions.map((value) => <option key={value} value={value}>{label(value)}</option>)}</select></label>
      <label><span>Due state</span><select value={query.due_state ?? ""} onChange={(event) => updateQuery({ due_state: event.target.value ? event.target.value as DistributionDueState : undefined })}><option value="">Any deadline</option>{dueOptions.map((value) => <option key={value} value={value}>{label(value)}</option>)}</select></label>
      <label><span>Subject type</span><input value={query.subject_type ?? ""} onChange={(event) => updateQuery({ subject_type: event.target.value || undefined })}/></label>
      <label><span>Subject ID</span><input value={query.subject_id ?? ""} onChange={(event) => updateQuery({ subject_id: event.target.value || undefined })}/></label>
      <label><span>Owner</span><input value={query.owner ?? ""} onChange={(event) => updateQuery({ owner: event.target.value || undefined })}/></label>
    </div>
    <div className="forms-task-layout">
      <div className="forms-table-wrap">
        <table className="forms-table"><thead><tr><th>Distribution</th><th>Status</th><th>Form revision</th><th>Deadline</th><th>Subject</th><th><span className="sr-only">Open</span></th></tr></thead><tbody>{items.map((value) => <tr key={value.id} className={selectedID === value.id ? "selected" : ""}><td><strong>{value.title}</strong><span>{value.purpose}</span></td><td><span className={`forms-status status-${value.status.toLowerCase()}`}>{label(value.status)}</span></td><td>v{value.form_template_version}</td><td>{formatDateTime(value.deadline)}</td><td>{value.subject_type} · {value.subject_id}</td><td><button type="button" aria-label={`Open ${value.title}`} onClick={() => setSelectedID(value.id)}>Open</button></td></tr>)}</tbody></table>
        {items.length === 0 && <div className="forms-task-empty"><strong>No sent forms match this view</strong><span>Change the filters or send a form.</span></div>}
        {nextCursor && <button className="forms-load-more" type="button" disabled={busy === "more"} onClick={() => void loadMore()}>{busy === "more" ? "Loading…" : "Load more"}</button>}
      </div>
      <aside className="forms-task-detail" aria-label="Selected distribution">{detail ? <><span className="forms-detail-kicker">{detail.distribution.subject_type}</span><h3>{detail.distribution.title}</h3><p>{detail.distribution.purpose}</p><dl><div><dt>Status</dt><dd>{label(detail.distribution.status)}</dd></div><div><dt>Recipients</dt><dd>{counts?.to ?? 0} To · {counts?.cc ?? 0} CC</dd></div><div><dt>Completed</dt><dd>{counts?.completed ?? 0}/{counts?.to ?? 0}</dd></div><div><dt>Deadline</dt><dd>{formatDateTime(detail.distribution.deadline)}</dd></div><div><dt>Access</dt><dd>{label(detail.distribution.access_policy)}</dd></div><div><dt>Workspace</dt><dd>{label(detail.workspace.status)} · v{detail.workspace.version}</dd></div><div><dt>Form</dt><dd>{detail.distribution.form_template_id} · v{detail.distribution.form_template_version}</dd></div></dl><div className="forms-detail-actions">{detail.distribution.status === "OPEN" && <button type="button" disabled={Boolean(busy)} onClick={() => void lifecycle("lock")}>Lock responses</button>}{detail.distribution.status === "LOCKED" && <button type="button" disabled={Boolean(busy)} onClick={() => void lifecycle("reopen")}>Reopen</button>}{!["REVOKED", "COMPLETED", "SUPERSEDED"].includes(detail.distribution.status) && <button type="button" disabled={Boolean(busy)} onClick={() => void lifecycle("revoke")}>Revoke</button>}{detail.distribution.status === "OPEN" && <button type="button" disabled={Boolean(busy)} onClick={() => setChangeMode("amend")}>Amend distribution</button>}{detail.distribution.status === "OPEN" && <button type="button" disabled={Boolean(busy)} onClick={() => setChangeMode("supersede")}>Replace form version</button>}</div></> : <><span className="forms-detail-kicker">Distribution detail</span><h3>Select a sent form</h3><p>Choose a sent form to review recipient progress, deadline and access method.</p></>}</aside>
    </div>
  </section>;
}

function readQuery(): DistributionQuery {
  const params = new URLSearchParams(window.location.search);
  const status = params.get("dist_status") as DistributionStatus | null;
  const due = params.get("dist_due") as DistributionDueState | null;
  return { status: status || undefined, due_state: due || undefined, subject_type: params.get("dist_subject_type") || undefined, subject_id: params.get("dist_subject_id") || undefined, owner: params.get("dist_owner") || undefined, limit: 25 };
}
function writeQuery(query: DistributionQuery) {
  const url = new URL(window.location.href);
  const set = (key: string, value?: string) => value ? url.searchParams.set(key, value) : url.searchParams.delete(key);
  set("dist_status", query.status); set("dist_due", query.due_state); set("dist_subject_type", query.subject_type?.trim()); set("dist_subject_id", query.subject_id?.trim()); set("dist_owner", query.owner?.trim());
  window.history.replaceState(null, "", `${url.pathname}${url.search}${url.hash}`);
}
function label(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
