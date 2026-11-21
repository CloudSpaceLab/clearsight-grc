import { useEffect, useRef, useState } from "react";
import { resolveAuthority, type RuntimeContext } from "../api";
import { createIdempotentReportRun, downloadReportRun, listReportDefinitions, listReportRunPage } from "../reportingApi";
import type { ReportDefinition, ReportRun } from "../reportingTypes";
import "./reports.css";

type LoadState = "loading" | "ready" | "error";
type PendingReport = { definitionId: string; version: number; key: string; completed?: boolean };
export function Reports({ context, enableGeneration = false }: { context: RuntimeContext; enableGeneration?: boolean }) {
  const storageKey = `clearsight-report-request:${encodeURIComponent(JSON.stringify([context.tenant.id, context.legal_entity.id, context.actor.id]))}`;
  const [initial] = useState(() => readPending(storageKey));
  const [definitions, setDefinitions] = useState<ReportDefinition[]>([]);
  const [runs, setRuns] = useState<ReportRun[]>([]);
  const [definitionsState, setDefinitionsState] = useState<LoadState>("loading");
  const [runsState, setRunsState] = useState<LoadState>("loading");
  const [selected, setSelected] = useState<string>(); const [cursor, setCursor] = useState<string>();
  const [retry, setRetry] = useState(0); const [busy, setBusy] = useState(false); const [moreError, setMoreError] = useState(false);
  const [authority, setAuthority] = useState<"checking" | "allowed" | "denied">("checking");
  const [pending, setPending] = useState<PendingReport | null>(initial.pending);
  const [command, setCommand] = useState<"idle" | "pending" | "unknown" | "done">(initial.pending?.completed ? "done" : initial.unavailable || initial.pending ? "unknown" : "idle"); const commandLock = useRef(false);
  const [message, setMessage] = useState(initial.pending?.completed ? "Report requested." : initial.pending ? "Report request status unknown. Check recent reports before requesting another." : initial.unavailable ? "Report requests unavailable. Check browser storage." : ""); const [downloadBusy, setDownloadBusy] = useState(false); const downloadLock = useRef(false);
  const [loadedScope, setLoadedScope] = useState(""); const generation = useRef(0); const controller = useRef<AbortController | undefined>(undefined);
  const scope = `${context.tenant.id}\u0000${context.legal_entity.id}\u0000${context.actor.id}`;
  const entity = context.legal_entity.id;
  const validScope = Boolean(context.tenant.id && context.actor.id && entity && entity !== "*");
  useEffect(() => {
    const token = ++generation.current; const abort = new AbortController(); controller.current = abort;
    setDefinitions([]); setRuns([]); setSelected(undefined); setCursor(undefined); setMoreError(false); setBusy(false); setLoadedScope(scope);
    if (!validScope) { setDefinitionsState("error"); setRunsState("error"); return () => abort.abort(); }
    setDefinitionsState("loading"); setRunsState("loading");
    listReportDefinitions(false, abort.signal).then((items) => {
      if (generation.current !== token) return;
      if (!Array.isArray(items) || items.some((item) => item.legal_entity_id !== entity)) { setDefinitionsState("error"); return; }
      setDefinitions(items); setDefinitionsState("ready");
    }).catch(() => { if (generation.current === token && !abort.signal.aborted) setDefinitionsState("error"); });
    listReportRunPage({ limit: 25 }, abort.signal).then((page) => {
      if (generation.current !== token) return;
      if (!Array.isArray(page.items) || page.items.some((item) => item.legal_entity_id !== entity)) { setRunsState("error"); return; }
      setRuns(page.items); setCursor(page.next_cursor); setRunsState("ready");
    }).catch(() => { if (generation.current === token && !abort.signal.aborted) setRunsState("error"); });
    return () => { abort.abort(); generation.current++; };
  }, [scope, entity, validScope, retry]);
  useEffect(() => {
    let active = true; setAuthority("checking");
    if (!validScope || !enableGeneration) { setAuthority("denied"); return () => { active = false; }; }
    resolveAuthority({ object_type: "REPORT_RUN", object_id: entity, responsibility: "PERFORMER", decision_type: "report.run.create", materiality: 3 }).then((route) => {
      if (!active) return;
      const permitted = Boolean(route.rule_id && route.policy_version && (route.principal?.id === context.actor.id || route.candidate_principals?.some((candidate) => candidate.id === context.actor.id)));
      setAuthority(permitted ? "allowed" : "denied");
    }).catch(() => { if (active) setAuthority("denied"); });
    return () => { active = false; };
  }, [scope, entity, validScope, context.actor.id, retry, enableGeneration]);
  async function more() {
    if (!cursor || busy || !controller.current) return;
    const token = generation.current; const signal = controller.current.signal; setBusy(true); setMoreError(false);
    try {
      const page = await listReportRunPage({ limit: 25, cursor }, signal);
      if (token !== generation.current) return;
      if (!Array.isArray(page.items) || page.items.some((item) => item.legal_entity_id !== entity)) throw new Error("scope mismatch");
      setRuns((items) => [...items, ...page.items]); setCursor(page.next_cursor);
    } catch { if (token === generation.current && !signal.aborted) setMoreError(true); }
    finally { if (token === generation.current) setBusy(false); }
  }
  async function generate(definition: ReportDefinition) {
    if (!enableGeneration || commandLock.current || command !== "idle" || authority !== "allowed" || definition.status !== "ACTIVE" || !definition.effective || definition.legal_entity_id !== entity) return;
    await withLock(async () => {
      const existing = readPending(storageKey);
      if (existing.unavailable || existing.pending) { setPending(existing.pending); setCommand(existing.pending?.completed ? "done" : "unknown"); setMessage(existing.pending?.completed ? "Report requested." : existing.pending ? "Report request status unknown. Check recent reports before requesting another." : "Report requests unavailable. Check browser storage."); return; }
      let next: PendingReport;
      try { next = { definitionId: definition.id, version: definition.current_version, key: crypto.randomUUID() }; localStorage.setItem(storageKey, JSON.stringify(next)); }
      catch { setCommand("unknown"); setMessage("Report requests unavailable. Check browser storage."); return; }
      setPending(next); await executeLocked(next);
    });
  }
  async function execute(request: PendingReport) {
    if (!enableGeneration || commandLock.current || authority !== "allowed" || request.completed) return;
    await withLock(async () => {
      const stored = readPending(storageKey);
      if (stored.unavailable || stored.pending?.key !== request.key) { setCommand("unknown"); setMessage("Report requests unavailable. Check browser storage."); return; }
      if (stored.pending.completed) { setPending(stored.pending); setCommand("done"); setMessage("Report requested."); return; }
      await executeLocked(request);
    });
  }
  async function withLock(action: () => Promise<void>) {
    if (commandLock.current) return;
    if (!navigator.locks?.request) { setCommand("unknown"); setMessage("Report requests unavailable. Check browser support."); return; }
    commandLock.current = true; setCommand("pending");
    try {
      let entered = false;
      await navigator.locks.request(storageKey, { mode: "exclusive", ifAvailable: true }, async (lock) => { if (!lock) return; entered = true; await action(); });
      if (!entered) { setCommand("idle"); setMessage("Another report request is in progress."); }
    } catch { setCommand("unknown"); setMessage("Report request status unknown. Check recent reports before requesting another."); }
    finally { commandLock.current = false; }
  }
  async function executeLocked(request: PendingReport) {
    const token = generation.current; setCommand("pending"); setMessage("");
    try {
      const run = await createIdempotentReportRun(request.definitionId, request.version, request.key);
      if (token !== generation.current) return;
      if (!run.id || run.legal_entity_id !== entity || run.definition_id !== request.definitionId || run.definition_version !== request.version || run.requested_by_ref !== context.actor.id || !["QUEUED", "RUNNING", "READY", "FAILED"].includes(run.status)) throw new Error("unconfirmed");
      const stored = readPending(storageKey);
      if (stored.unavailable || stored.pending?.key !== request.key) throw new Error("receipt unavailable");
      const completed = { ...request, completed: true }; localStorage.setItem(storageKey, JSON.stringify(completed)); setPending(completed);
      if (runsState === "ready") setRuns((items) => [run, ...items.filter((item) => item.id !== run.id)]); setCommand("done"); setMessage("Report requested.");
    } catch { if (token === generation.current) { setCommand("unknown"); setMessage("Report request status unknown. Check recent reports before requesting another."); } }
  }
  async function requestAnother() {
    if (command !== "done" || !pending?.completed) return;
    await withLock(async () => {
      const stored = readPending(storageKey);
      if (stored.unavailable || stored.pending?.key !== pending.key || !stored.pending.completed) { setCommand("unknown"); setMessage("Report request status unknown. Check recent reports before requesting another."); return; }
      try { localStorage.removeItem(storageKey); setPending(null); setCommand("idle"); setMessage(""); }
      catch { setCommand("unknown"); setMessage("Report requests unavailable. Check browser storage."); }
    });
  }
  async function download(run: ReportRun) {
    if (downloadLock.current || !available(run) || run.legal_entity_id !== entity) return;
    downloadLock.current = true; setDownloadBusy(true); if (command !== "unknown") setMessage("");
    try {
      const result = await downloadReportRun(run.id);
      const url = URL.createObjectURL(result.blob); const link = document.createElement("a");
      link.href = url; link.download = safeFilename(result.filename || `report.${run.format.toLowerCase()}`); link.click(); URL.revokeObjectURL(url);
      if (command !== "unknown") setMessage("Report downloaded.");
    } catch { if (command !== "unknown") setMessage("Report download unavailable. Try again."); }
    finally { downloadLock.current = false; setDownloadBusy(false); }
  }
  const current = definitions.find((item) => item.id === selected);
  if (loadedScope !== scope) return <p role="status">Loading reports…</p>;
  return <section className="v-reports"><header className="v-page-heading"><div><p className="v-eyebrow">Reporting</p><h1>Reports & insights</h1></div></header>
    <div className="v-reports-layout"><div className="v-reports-stack">
      <section className="v-panel"><header><h2>Report definitions</h2></header>
        {definitionsState === "loading" && <p role="status">Loading report definitions…</p>}
        {definitionsState === "error" && <><p role="alert">Report definitions unavailable.</p><button disabled={command === "pending"} onClick={() => setRetry((value) => value + 1)}>Try again</button></>}
        {definitionsState === "ready" && (definitions.length ? <ul className="v-reports-list">{definitions.map((item) => <li key={item.id}><div><strong>{item.name}</strong><small>{label(item.dataset)} · {label(item.status)} · Version {item.current_version}</small></div><button type="button" aria-label={`View ${item.name}`} onClick={() => setSelected(item.id)}>View</button></li>)}</ul> : <p>No report definitions in this legal entity.</p>)}
      </section>
      <section className="v-panel"><header><h2>Recent reports</h2></header>
        {runsState === "loading" && <p role="status">Loading recent reports…</p>}
        {runsState === "error" && <><p role="alert">Recent reports unavailable.</p><button disabled={command === "pending"} onClick={() => setRetry((value) => value + 1)}>Try again</button></>}
        {runsState === "ready" && (runs.length ? <ul className="v-reports-list">{runs.map((item) => <li key={item.id}><div><strong>{item.definition_code}</strong><small>{item.format} · {formatDate(item.as_of)} · Version {item.definition_version}</small></div><span className="v-status">{runLabel(item)}</span>{available(item) && <button type="button" aria-label={`Download ${item.definition_code}`} disabled={downloadBusy} onClick={() => void download(item)}>Download</button>}</li>)}</ul> : <p>No report runs in this legal entity.</p>)}
        {moreError && <p role="alert">More reports unavailable.</p>}{cursor && <button type="button" disabled={busy} onClick={() => void more()}>{busy ? "Loading…" : "Load more reports"}</button>}
      </section>
    </div><aside className="v-panel v-reports-detail" aria-label="Report definition details"><h2>{current?.name ?? "Report details"}</h2>{current ? <><p>{current.description}</p><dl><div><dt>Dataset</dt><dd>{label(current.dataset)}</dd></div><div><dt>Scope</dt><dd>{label(current.scope_kind)}</dd></div><div><dt>Format</dt><dd>{current.format}</dd></div><div><dt>Status</dt><dd>{label(current.status)}</dd></div><div><dt>Version</dt><dd>{current.current_version}</dd></div></dl>{enableGeneration && current.status === "ACTIVE" && current.effective && (authority === "checking" ? <p role="status">Checking report access…</p> : authority === "denied" ? <p>Report generation unavailable for your role.</p> : (command === "idle" || command === "pending") && <button type="button" className="v-primary" disabled={command === "pending"} onClick={() => void generate(current)}>{command === "pending" ? "Requesting…" : "Generate report"}</button>)}</> : <p>Select a report definition.</p>}</aside></div>{message && <p role={command === "unknown" ? "alert" : "status"}>{message}</p>}{enableGeneration && pending && command === "unknown" && authority === "allowed" && <button type="button" onClick={() => void execute(pending)}>Retry report request</button>}{enableGeneration && command === "done" && <button type="button" onClick={() => void requestAnother()}>Request another report</button>}
  </section>;
}
function label(value: string) { const text = value.toLowerCase().replaceAll("_", " "); return text.charAt(0).toUpperCase() + text.slice(1); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(date); }
function available(run: ReportRun) { const expiry = Date.parse(run.expires_at); return run.status === "READY" && Number.isFinite(expiry) && expiry > Date.now(); }
function runLabel(run: ReportRun) { if (run.status === "READY" && !available(run)) return Number.isFinite(Date.parse(run.expires_at)) ? "Expired" : "Availability unknown"; return label(run.status); }
function safeFilename(value: string) { return value.replace(/[\\/\x00-\x1f\x7f]/g, "_").slice(0, 180) || "report"; }
function readPending(key: string): { pending: PendingReport | null; unavailable: boolean } {
  try {
    const value = localStorage.getItem(key); if (!value) return { pending: null, unavailable: false };
    const parsed: unknown = JSON.parse(value);
    if (!parsed || typeof parsed !== "object") return { pending: null, unavailable: true };
    const p = parsed as Partial<PendingReport>;
    if (typeof p.definitionId !== "string" || !p.definitionId || p.definitionId.length > 128 || !Number.isSafeInteger(p.version) || (p.version ?? 0) < 1 || typeof p.key !== "string" || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(p.key) || (p.completed !== undefined && typeof p.completed !== "boolean")) return { pending: null, unavailable: true };
    return { pending: p as PendingReport, unavailable: false };
  } catch { return { pending: null, unavailable: true }; }
}
