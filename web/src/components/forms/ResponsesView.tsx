import { useEffect, useState } from "react";
import { loadDistributionPage, loadResponseRevisions, type Distribution, type ResponseRevision } from "../../formsDistributionApi";
import { Button, EmptyState, Notice, SelectableRecord } from "../ui";

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
    <div className="forms-task-heading"><div><span>Submitted history</span><h2 id="responses-title">Responses</h2><p>Review each submitted version, sign-off assurance and scoring result. Earlier versions remain available for audit.</p></div></div>
    {error && <Notice tone="error">{error}</Notice>}
    <div className="forms-task-layout forms-response-layout">
      <section className="forms-response-distributions" aria-label="Response distributions">
        <div className="forms-response-section-heading"><div><span className="forms-detail-kicker">Sent forms</span><h3>Response distributions</h3></div><span>{distributions.length} loaded</span></div>
        <ul>{distributions.map((value) => <li key={value.id}><SelectableRecord title={value.title} metadata={`${label(value.status)} · Due ${formatDate(value.deadline)}`} isSelected={selectedID === value.id} onPress={() => setSelectedID(value.id)}/></li>)}</ul>
        {nextCursor && <Button size="compact" isDisabled={busy} onPress={() => void loadMoreDistributions()}>Load more distributions</Button>}
      </section>
      <div className="forms-response-review">
        <div className="forms-response-review-heading"><div><span className="forms-detail-kicker">Selected distribution</span><h3>{distribution?.title || "Response history"}</h3></div><span>{busy ? "Loading…" : `${revisions.length} submitted version${revisions.length === 1 ? "" : "s"}`}</span></div>
        <section className="forms-response-revisions" aria-label="Version history">
          <h4>Version history</h4>
          {!busy && revisions.length === 0 ? <EmptyState population={distribution ? `Submitted responses for ${distribution.title}` : "Submitted responses"} title="No submitted responses" description="A response version will appear after a recipient submits this form."/> : <ol>{revisions.map((value) => {
            const title = `Revision ${value.revision}${value.current ? " · Current" : ""}`;
            const metadata = `${label(value.state)} · ${label(value.achieved_assurance)} · ${formatDateTime(value.created_at)}`;
            return <li key={value.id}><SelectableRecord title={title} metadata={metadata} aria-label={`Review ${title}, ${metadata}`} isSelected={selectedRevision?.id === value.id} onPress={() => setSelectedRevision(value)}/></li>;
          })}</ol>}
        </section>
        <section className="forms-response-detail" aria-label="Selected response version">{selectedRevision ? <><div className="forms-response-detail-heading"><div><span className="forms-detail-kicker">Revision {selectedRevision.revision}</span><h3>{selectedRevision.current ? "Current response" : "Historical response"}</h3></div><span>{formatDateTime(selectedRevision.created_at)}</span></div><dl><div><dt>State</dt><dd>{label(selectedRevision.state)}</dd></div><div><dt>Assurance</dt><dd>{label(selectedRevision.achieved_assurance)}</dd></div><div><dt>Compliance score</dt><dd>{selectedRevision.compliance_score === undefined ? "Not scored" : `${selectedRevision.compliance_score}%`}</dd></div><div><dt>Scored coverage</dt><dd>{Math.round(selectedRevision.scored_weight_coverage * 100) / 100}%</dd></div><div><dt>Scoring policy</dt><dd>{selectedRevision.scoring_policy_version || "Not specified"}</dd></div>{selectedRevision.supersedes_revision_id && <div><dt>Supersedes</dt><dd>{selectedRevision.supersedes_revision_id}</dd></div>}</dl>{selectedRevision.signoff_summary && <details><summary>Sign-off summary</summary><pre>{JSON.stringify(selectedRevision.signoff_summary, null, 2)}</pre></details>}{selectedRevision.critical_field_results?.length ? <details><summary>Critical field results</summary><pre>{JSON.stringify(selectedRevision.critical_field_results, null, 2)}</pre></details> : null}<Notice tone="info">Submitted versions cannot be changed. Amend the distribution if the recipient must provide an updated response.</Notice></> : <EmptyState population="Submitted response versions" title="Select a response version" description="Choose a submitted version to review its assurance, score and sign-off."/>}</section>
      </div>
    </div>
  </section>;
}
function label(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (part) => part.toUpperCase()); }
function formatDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(date); }
function formatDateTime(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Unknown" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date); }
function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
