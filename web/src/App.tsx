import { useEffect, useMemo, useState } from "react";
import { loadCaptureRequest, loadToday, resolveAuthority, submitCaptureRequest } from "./api";
import type { AttentionItem, AuthorityResolution, CaptureRequest } from "./types";

const fallbackItems: AttentionItem[] = [
  {
    id: "fallback-1",
    type: "REGULATORY_CHANGE",
    title: "Review proposed CBN digital-channel obligations",
    why_now: "Seven source-linked provisions may affect mobile banking and two payment vendors.",
    scope: "Digital Channels · Bank NG",
    state: "Applicability review",
    evidence: "Official source verified",
    owner: "Regulatory Compliance",
    due_at: new Date(Date.now() + 3 * 86400000).toISOString(),
    primary_action: "Review seven proposed obligations"
  },
  {
    id: "fallback-2",
    type: "MATTER",
    title: "Resolve four privileged-access exceptions",
    why_now: "IAM and HR evidence resolved 1,246 accounts; four still lack current business-need evidence.",
    scope: "Treasury Operations · July 2026",
    state: "Waiting for focused response",
    evidence: "99.7% population resolved",
    owner: "Treasury Technology",
    due_at: new Date(Date.now() + 36 * 3600000).toISOString(),
    primary_action: "Confirm four account owners"
  }
];

function App() {
  const [items, setItems] = useState<AttentionItem[]>(fallbackItems);
  const [connection, setConnection] = useState<"loading" | "live" | "fallback">("loading");
  const [resolution, setResolution] = useState<AuthorityResolution | null>(null);
  const [capture, setCapture] = useState<CaptureRequest | null>(null);
  const [activePanel, setActivePanel] = useState<"none" | "routing" | "capture">("none");

  useEffect(() => {
    loadToday()
      .then((value) => {
        setItems(value);
        setConnection("live");
      })
      .catch(() => setConnection("fallback"));
  }, []);

  const dueSoon = useMemo(() => items.filter((item) => Date.parse(item.due_at) - Date.now() < 4 * 86400000).length, [items]);

  async function inspectRouting() {
    setActivePanel("routing");
    if (!resolution) setResolution(await resolveAuthority().catch(() => null));
  }

  async function openCapture() {
    setActivePanel("capture");
    if (!capture) setCapture(await loadCaptureRequest().catch(() => null));
  }

  return (
    <div className="app-shell">
      <aside className="sidebar" aria-label="Primary navigation">
        <div className="brand-mark" aria-label="ClearSight">C</div>
        <nav>
          {['Today', 'Programs', 'Work', 'Explore', 'Configure'].map((label, index) => (
            <button className={index === 0 ? 'nav-item active' : 'nav-item'} key={label} aria-current={index === 0 ? 'page' : undefined}>
              <span>{label.slice(0, 1)}</span><b>{label}</b>
            </button>
          ))}
        </nav>
        <button className="avatar" aria-label="Account menu">AO</button>
      </aside>

      <main>
        <header className="topbar">
          <div>
            <span className="eyebrow">Demonstration Bank Nigeria · Control Assurance</span>
            <h1>Today</h1>
            <p>Only the work that needs your judgment or action.</p>
          </div>
          <div className="topbar-actions">
            <span className={`connection ${connection}`}>{connection === 'live' ? 'API live' : connection === 'fallback' ? 'Preview data' : 'Connecting'}</span>
            <button className="secondary-button" onClick={inspectRouting}>Inspect authority</button>
            <button className="primary-button" onClick={openCapture}>Open capture wizard</button>
          </div>
        </header>

        <section className="brief-grid" aria-label="Current brief">
          <div className="brief-stat"><span>Needs attention</span><strong>{items.length}</strong><small>Material items only</small></div>
          <div className="brief-stat"><span>Due soon</span><strong>{dueSoon}</strong><small>Within four days</small></div>
          <div className="brief-stat verified"><span>Automatically maintained</span><strong>18</strong><small>No intervention required</small></div>
        </section>

        <section className="section-header">
          <div><h2>Your attention</h2><p>Grouped by the outcome required from you.</p></div>
          <button className="text-button">Analyst view</button>
        </section>

        <section className="attention-list">
          {items.map((item) => <AttentionCard item={item} key={item.id} />)}
        </section>

        <section className="quiet-section">
          <div><span className="verified-dot" /> No material change in 6 Programs</div>
          <p>Evidence and source checks completed 8 minutes ago.</p>
        </section>
      </main>

      {activePanel !== "none" && (
        <div className="panel-backdrop" onMouseDown={() => setActivePanel("none")}>
          <aside className="side-panel" onMouseDown={(event) => event.stopPropagation()} aria-label={activePanel === 'routing' ? 'Authority explanation' : 'Capture wizard'}>
            <button className="panel-close" onClick={() => setActivePanel("none")} aria-label="Close">×</button>
            {activePanel === "routing" ? <RoutingPanel resolution={resolution} /> : <CapturePanel request={capture} />}
          </aside>
        </div>
      )}
    </div>
  );
}

function AttentionCard({ item }: { item: AttentionItem }) {
  const due = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' }).format(new Date(item.due_at));
  return (
    <article className="attention-card">
      <div className="attention-icon">{item.type === 'REGULATORY_CHANGE' ? '§' : '!'}</div>
      <div className="attention-content">
        <div className="card-kicker"><span>{item.state}</span><time>{due}</time></div>
        <h3>{item.title}</h3>
        <p>{item.why_now}</p>
        <div className="card-meta"><span>{item.scope}</span><span>{item.evidence}</span><span>{item.owner}</span></div>
      </div>
      <button className="card-action">{item.primary_action}<span>→</span></button>
    </article>
  );
}

function RoutingPanel({ resolution }: { resolution: AuthorityResolution | null }) {
  return (
    <div className="panel-content">
      <span className="eyebrow">Policy simulation</span>
      <h2>Who authorizes this material Matter?</h2>
      <p>ClearSight resolves responsibility from scope, policy, materiality and current authority—not a hard-coded assignee.</p>
      {resolution ? (
        <div className="resolution-card">
          <div className="principal-avatar">CR</div>
          <div><strong>{resolution.principal.display_name}</strong><span>{resolution.principal.role} · {resolution.principal.kind}</span></div>
          <mark>Eligible</mark>
        </div>
      ) : <div className="skeleton">API unavailable — start the Go service to resolve the live route.</div>}
      <dl className="explanation-list">
        <div><dt>Responsibility</dt><dd>Authorizer</dd></div>
        <div><dt>Legal entity</dt><dd>Demonstration Bank Nigeria</dd></div>
        <div><dt>Materiality</dt><dd>5 · Executive authority</dd></div>
        <div><dt>Policy</dt><dd>{resolution?.policy_version ?? 'demo-2026-08-05'}</dd></div>
      </dl>
      <div className="sequence"><span>Control owner</span><i>→</i><span>Control Assurance</span><i>→</i><b>CRO</b></div>
      <button className="primary-button full">Open routing policy</button>
    </div>
  );
}

function CapturePanel({ request }: { request: CaptureRequest | null }) {
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [receipt, setReceipt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function submit() {
    if (!request) return;
    setError(null);
    try {
      const result = await submitCaptureRequest(request.id, request.version, answers);
      setReceipt(`Submitted ${new Date(result.submitted_at).toLocaleString()}`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Submission failed');
    }
  }

  if (!request) return <div className="panel-content"><span className="eyebrow">Focused capture</span><h2>Request unavailable</h2><p>Start the API to load the live request-scoped wizard.</p></div>;
  if (receipt) return <div className="panel-content"><span className="eyebrow">Receipt</span><h2>Response recorded</h2><div className="success-state">✓</div><p>{receipt}</p><p>Your response is now an Observation. Evidence sufficiency will be evaluated separately.</p></div>;

  return (
    <div className="panel-content">
      <span className="eyebrow">Focused capture · {request.estimated_minutes} minutes</span>
      <h2>{request.title}</h2>
      <p>{request.purpose}</p>
      <div className="why-you"><strong>Why you</strong><span>{request.why_you}</span></div>
      <h3>Already known</h3>
      <dl className="known-facts">{Object.entries(request.known_facts).map(([key, value]) => <div key={key}><dt>{key}</dt><dd>{value}</dd></div>)}</dl>
      {request.fields.map((field) => (
        <label className="field" key={field.id}>
          <span>{field.label}{field.required ? ' *' : ''}</span>
          {field.type === 'single_select' ? (
            <select value={answers[field.id] ?? ''} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })}>
              <option value="">Select one</option>
              {field.options?.map((option) => <option key={option}>{option}</option>)}
            </select>
          ) : <textarea value={answers[field.id] ?? ''} onChange={(event) => setAnswers({ ...answers, [field.id]: event.target.value })} placeholder={field.description} />}
        </label>
      ))}
      {error && <p className="error-text">{error}</p>}
      <div className="wizard-actions"><button className="secondary-button">Wrong recipient</button><button className="primary-button" onClick={submit}>Review and submit</button></div>
    </div>
  );
}

export default App;
