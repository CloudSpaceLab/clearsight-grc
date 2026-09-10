import { useEffect, useRef, useState } from "react";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { loadVendorRiskWork, openFinding, openRiskAction, overdueRiskAction, summarizeVendorRiskWork, type VendorRiskWork } from "../vendorRiskWork";
import { Button, Notice, StatusBadge } from "./ui";
import "./vendor-portfolio.css";

type WorkFilter = "ALL" | "OPEN" | "ACTIONS" | "OVERDUE";
export function VendorPortfolio({ records, hasMore, onOpenMatter, detail = false, refreshKey = 0 }: {
  records: VendorRelationshipAggregate[]; hasMore: boolean; onOpenMatter?: (id: string) => void; detail?: boolean; refreshKey?: number;
}) {
  const [work, setWork] = useState<VendorRiskWork>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  const [filter, setFilter] = useState<WorkFilter>("OPEN");
  const [limit, setLimit] = useState(15);
  const list = useRef<HTMLDivElement>(null);
  const ids = records.map(record => record.relationship.id).join(",");
  useEffect(() => {
    let current = true;
    setLoading(true); setError(false); setWork(undefined);
    void loadVendorRiskWork(ids ? ids.split(",") : []).then(value => { if (current) setWork(value); }).catch(() => { if (current) setError(true); }).finally(() => { if (current) setLoading(false); });
    return () => { current = false; };
  }, [ids, refreshKey, retry]);
  const now = Date.now();
  const totals = work && !loading ? summarizeVendorRiskWork(work, now) : undefined;
  const items = (work?.items ?? []).filter(item => filter === "ALL" || openFinding(item) && (filter === "OPEN" || item.record.actions.some(action => filter === "OVERDUE" ? overdueRiskAction(action, now) : openRiskAction(action))));
  const sorted = [...items].sort((a, b) => Number(b.record.actions.some(action => overdueRiskAction(action, now))) - Number(a.record.actions.some(action => overdueRiskAction(action, now))) || a.record.matter.title.localeCompare(b.record.matter.title));
  function select(value: WorkFilter) { setFilter(value); setLimit(15); list.current?.scrollIntoView?.({ block: "start" }); }
  const metrics = [
    { label: "Open findings", value: totals?.findings, filter: "OPEN", action: "Review findings", tone: "accent" },
    { label: "Open actions", value: totals?.openActions, filter: "ACTIONS", action: "Review actions", tone: "neutral" },
    { label: "Overdue actions", value: totals?.overdueActions, filter: "OVERDUE", action: "Review overdue actions", tone: "warning" },
  ] as const;
  return <section className={`vendor-portfolio${detail ? " vendor-portfolio--detail" : ""}`} aria-label={detail ? "Vendor findings and actions" : "Vendor portfolio metrics"}>
    <div className="vendor-portfolio-scope"><span>{records.length} loaded {records.length === 1 ? "service" : "services"}{hasMore ? " · More services available" : ""}</span>{work && <span>Checked <time dateTime={work.checkedAt}>{new Date(work.checkedAt).toLocaleString()}</time></span>}</div>
    <div className="vendor-metrics">
      <div className="vendor-metric vendor-metric--portfolio" role="group" aria-label="Vendor services"><span className="vendor-metric-label">Vendor services</span><strong className="vendor-metric-value">{records.length}</strong><span>{new Set(records.map(record => record.vendor.id)).size} vendors · Current search</span><Button variant="quiet" onPress={() => select("ALL")}>Review linked findings</Button></div>
      {metrics.map(metric => <div key={metric.label} className={`vendor-metric vendor-metric--${metric.tone}`} role="group" aria-label={metric.label}><span className="vendor-metric-label">{metric.label}</span><strong className="vendor-metric-value">{metric.value ?? "Unknown"}</strong><Button variant="quiet" onPress={() => select(metric.filter)}>{metric.action}</Button></div>)}
    </div>
    <div className="vendor-findings" ref={list}>
      <header><h2>Findings and actions</h2><div className="vendor-findings-filters">{([ ["OPEN", "Open findings"], ["OVERDUE", "Overdue actions"], ["ALL", "All findings"] ] as const).map(([value, label]) => <Button key={value} variant={filter === value ? "secondary" : "quiet"} aria-pressed={filter === value} onPress={() => select(value)}>{label}</Button>)}</div></header>
      {loading && <p role="status">Checking linked findings…</p>}
      {error && <Notice tone="error">Linked findings could not be checked. <Button onPress={() => setRetry(value => value + 1)}>Retry findings</Button></Notice>}
      {work && !work.complete && <Notice tone="warning">Some linked findings could not be checked. Totals unknown. <Button onPress={() => setRetry(value => value + 1)}>Retry findings</Button></Notice>}
      {!loading && !error && work?.complete && items.length === 0 && <p>No linked findings match this filter for the loaded services.</p>}
      <ul>{sorted.slice(0, limit).map(({ record, relationshipIDs }) => {
        const facts = record.matter.known_facts ?? {};
        const accountableFunction = sourceFieldValue(facts, "BUSINESS OWNER");
        const actions = record.actions.filter(action => filter === "OVERDUE" ? overdueRiskAction(action, now) : filter === "ACTIONS" ? openRiskAction(action) : true);
        const services = records.filter(item => relationshipIDs.includes(item.relationship.id));
        return <li key={record.matter.id} className="vendor-finding">
          <div className="vendor-finding-heading"><div><h3>{record.matter.title}</h3><p>{services.map(item => `${item.vendor.legal_name} · ${item.relationship.service_name}`).join("; ")}</p></div><StatusBadge tone={record.matter.status === "CLOSED" ? "success" : "neutral"}>{record.status_label || stateLabel(record.matter.status)}</StatusBadge></div>
          <div className="vendor-finding-source">{facts.sample === true && <span>Sample data</span>}{typeof facts.source_rating === "string" && facts.source_rating && <span>Source rating: {facts.source_rating}</span>}{typeof facts.source_assessor === "string" && facts.source_assessor && <span>Internal assessor: {facts.source_assessor}</span>}{accountableFunction && <span>Accountable function: {accountableFunction}</span>}{typeof facts.source_owner === "string" && facts.source_owner && <span>Action performer: {facts.source_owner}</span>}{typeof facts.source_period === "string" && facts.source_period && <span>Source period: {facts.source_period}</span>}</div>
          <ul className="vendor-finding-actions">{actions.slice(0, 5).map(action => <li key={action.id}><span>{action.title}</span><div><StatusBadge tone={action.status === "BLOCKED" ? "warning" : "neutral"}>{stateLabel(action.status)}</StatusBadge>{action.due_at ? <span className={overdueRiskAction(action, now) ? "vendor-action-overdue" : ""}>{facts.source_file ? "Source target" : "Due"} <time dateTime={action.due_at}>{new Date(action.due_at).toLocaleDateString()}</time>{overdueRiskAction(action, now) ? " · Overdue" : ""}</span> : <span>No action deadline</span>}</div></li>)}</ul>
          <footer>{typeof facts.source_file === "string" && <small>{facts.source_file}{typeof facts.source_range === "string" ? ` · ${facts.source_range}` : ""}</small>}{actions.length > 5 && <small>{actions.length - 5} further actions</small>}{onOpenMatter && <Button aria-label={`Review ${record.matter.title}`} onPress={() => onOpenMatter(record.matter.id)}>Review finding</Button>}</footer>
        </li>;
      })}</ul>
      {sorted.length > limit && <Button onPress={() => setLimit(value => value + 15)}>Load more findings</Button>}
      {!!totals?.implementedActions && <p>{totals.implementedActions} implemented {totals.implementedActions === 1 ? "action still requires" : "actions still require"} outcome verification.</p>}
    </div>
  </section>;
}
function stateLabel(value: string) { const labels: Record<string, string> = { TRIAGE: "Initial review", ASSESSMENT: "Assessment", DECISION_REQUIRED: "Decision needed", ACTION_IN_PROGRESS: "Actions in progress", VERIFICATION: "Outcome verification", CLOSED: "Closed", CANCELLED: "Cancelled", PLANNED: "Planned", IN_PROGRESS: "In progress", BLOCKED: "Blocked", IMPLEMENTED: "Implemented" }; return labels[value] ?? "Status unknown"; }
function sourceFieldValue(facts: Record<string, unknown>, label: string) {
  if (!Array.isArray(facts.source_fields)) return "";
  const field = facts.source_fields.find((value): value is { label: string; value: string } => Boolean(value) && typeof value === "object" && "label" in value && "value" in value && typeof value.label === "string" && typeof value.value === "string" && value.label.trim().toUpperCase() === label);
  return field?.value.trim() ?? "";
}
