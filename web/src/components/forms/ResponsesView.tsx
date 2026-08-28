import { useEffect, useState } from "react";
import { loadDistributionPage, loadResponseRevisions, type Distribution, type ResponseRevision } from "../../formsDistributionApi";

export function ResponsesView() {
  const [distributions, setDistributions] = useState<Distribution[]>([]);
  const [selectedID, setSelectedID] = useState<string>();
  const [revisions, setRevisions] = useState<ResponseRevision[]>([]);
  const [selectedRevision, setSelectedRevision] = useState<ResponseRevision>();
  const [nextCursor, setNextCursor] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();

  useEffect(() => {
    void loadDistributionPage({ limit: 25 }).then((page) => {
      setDistributions(page.items); setNextCursor(page.next_cursor); setSelectedID(page.items[0]?.id);
    }).catch((cause) => setError(message(cause, "Response distributions could not be loaded.")));
  }, []);
  useEffect(() => {
    if (!selectedID) { setRevisions([]); setSelectedRevision(undefined); return; }
    setBusy(true); setError(undefined);
    void loadResponseRevisions(selectedID).then((page) => {
      setRevisions(page.items ?? []);
      setSelectedRevision((page.items ?? []).find((value) => value.current) ?? page.items?.[0]);
    }).catch((cause) => { setRevisions([]); setSelectedRevision(undefined); setError(message(cause, "Response revisions could not be loaded.")); }).finally(() => setBusy(false));
  }, [selectedID]);

  async function loadMoreDistributions() {
    if (!nextCursor || busy) return;
    setBusy(true);
    try {
      const page = await loadDistributionPage({ limit: 25, cursor: nextCursor });
      setDistributions((current) => [...current, ...page.items]); setNextCursor(page.next_cursor);
    } catch (cause) { setError(message(cause, "More response distributions could not be loaded.")); } finally { setBusy(false); }
  }

  const distribution = distributions.find((value) => value.id === selectedID);
  return <section className="forms-task-shell" aria-labelledby="responses-title">
    <div className="forms-task-heading"><div><span>Immutable history</span><h2 id="responses-title">Responses</h2><p>Submitted response revisions are append-only server records. This workspace is read-only by design.</p></div></div>
    {error && <div className="forms-message error" role="alert">{error}</div>}
    <div className="forms-task-layout forms-response-layout">
      <div className="forms-response-distributions"><h3>Distributions</h3><ul>{distributions.map((value) => <li key={value.id}><button type="button" className={selectedID === value.id ? "selected" : ""} onClick={() => setSelectedID(value.id)}><strong>{value.title}</strong><span>{label(value.status)} · {formatDate(value.deadline)}</span></button></li>)}</ul>{nextCursor && <button type="button" disabled={busy} onClick={() => void loadMoreDistributions()}>Load more</button>}</div>
      <div className="forms-response-revisions"><div><h3>{distribution?.title || "Response revisions"}</h3><span>{busy ? "Loading…" : `${revisions.length} immutable revision${revisions.length === 1 ? "" : "s"}`}</span></div>{!busy && revisions.length === 0 ? <div className="forms-task-empty"><strong>No submitted response revisions</strong><span>Draft workspace edits do not appear here until a governed submission creates an immutable revision.</span></div> : <ol>{revisions.map((value) => <li key={value.id}><button type="button" className={selectedRevision?.id === value.id ? "selected" : ""} onClick={() => setSelectedRevision(value)}><strong>Revision {value.revision}{value.current ? " · Current" : ""}</strong><span>{label(value.state)} · {label(value.achieved_assurance)} · {formatDateTime(value.created_at)}</span></button></li>)}</ol>}</div>
      <aside className="forms-task-detail" aria-label="Selected immutable response revision">{selectedRevision ? <><span className="forms-detail-kicker">Revision {selectedRevision.revision}</span><h3>{selectedRevision.current ? "Current response" : "Historical response"}</h3><dl><div><dt>State</dt><dd>{label(selectedRevision.state)}</dd></div><div><dt>Assurance</dt><dd>{label(selectedRevision.achieved_assurance)}</dd></div><div><dt>Compliance score</dt><dd>{selectedRevision.compliance_score === undefined ? "Not scored" : `${selectedRevision.compliance_score}%`}</dd></div><div><dt>Scored coverage</dt><dd>{Math.round(selectedRevision.scored_weight_coverage * 100) / 100}%</dd></div><div><dt>Scoring policy</dt><dd>{selectedRevision.scoring_policy_version || "Not specified"}</dd></div><div><dt>Created</dt><dd>{formatDateTime(selectedRevision.created_at)}</dd></div>{selectedRevision.supersedes_revision_id && <div><dt>Supersedes</dt><dd>{selectedRevision.supersedes_revision_id}</dd></div>}</dl>{selectedRevision.signoff_summary && <details><summary>Sign-off summary</summary><pre>{JSON.stringify(selectedRevision.signoff_summary, null, 2)}</pre></details>}{selectedRevision.critical_field_results?.length ? <details><summary>Critical field results</summary><pre>{JSON.stringify(selectedRevision.critical_field_results, null, 2)}</pre></details> : null}<div className="forms-detail-actions"><button type="button" disabled title="Immutable response revisions cannot be edited from the sender workspace.">Edit response</button></div></> : <><span className="forms-detail-kicker">Revision detail</span><h3>Select a response revision</h3><p>No mutation controls are exposed because response history is immutable.</p></>}</aside>
    </div>
  </section>;
}
function label(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date); }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
