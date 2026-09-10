import { useEffect, useMemo, useRef, useState } from "react";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { presentVendorExceptions, type VendorExceptionBand, type VendorExceptionFilter } from "../vendorExceptionPresentation";
import { loadVendorRiskWork, summarizeVendorRiskWork, type VendorRiskWork } from "../vendorRiskWork";
import { Button, Notice, StatusBadge } from "./ui";
import "./vendor-portfolio.css";

type StoredView = { filter: VendorExceptionFilter; vendor: string; owner: string; rating: string };
const viewStorageKey = "clearsight.vendor-exception-view";
const scrollStorageKey = "clearsight.vendor-exception-scroll";

function storedView(): StoredView {
  try {
    const value = JSON.parse(window.sessionStorage.getItem(viewStorageKey) ?? "null") as Partial<StoredView> | null;
    if (value && ["ATTENTION", "OVERDUE", "OPEN", "ALL"].includes(value.filter ?? "")) {
      return { filter: value.filter!, vendor: value.vendor ?? "", owner: value.owner ?? "", rating: value.rating ?? "" };
    }
  } catch { /* session storage is optional */ }
  return { filter: "ATTENTION", vendor: "", owner: "", rating: "" };
}

export function VendorPortfolio({ records, hasMore, onOpenMatter, detail = false, refreshKey = 0 }: {
  records: VendorRelationshipAggregate[]; hasMore: boolean; onOpenMatter?: (id: string) => void; detail?: boolean; refreshKey?: number;
}) {
  const initialView = useMemo(() => detail ? { filter: "OPEN" as const, vendor: "", owner: "", rating: "" } : storedView(), [detail]);
  const [work, setWork] = useState<VendorRiskWork>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  const [filter, setFilter] = useState<VendorExceptionFilter>(initialView.filter);
  const [vendor, setVendor] = useState(initialView.vendor);
  const [owner, setOwner] = useState(initialView.owner);
  const [rating, setRating] = useState(initialView.rating);
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const list = useRef<HTMLElement>(null);
  const ids = records.map(record => record.relationship.id).join(",");

  useEffect(() => {
    let current = true;
    setLoading(true); setError(false); setWork(undefined);
    void loadVendorRiskWork(ids ? ids.split(",") : []).then(value => { if (current) setWork(value); }).catch(() => { if (current) setError(true); }).finally(() => { if (current) setLoading(false); });
    return () => { current = false; };
  }, [ids, refreshKey, retry]);

  useEffect(() => {
    if (!detail) try { window.sessionStorage.setItem(viewStorageKey, JSON.stringify({ filter, vendor, owner, rating })); } catch { /* session storage is optional */ }
  }, [detail, filter, vendor, owner, rating]);

  useEffect(() => {
    if (detail) return;
    try {
      const stored = Number(window.sessionStorage.getItem(scrollStorageKey));
      if (stored > 0) requestAnimationFrame(() => window.scrollTo({ top: stored }));
    } catch { /* session storage is optional */ }
  }, [detail]);

  const now = Date.now();
  const totals = work && !loading ? summarizeVendorRiskWork(work, now) : undefined;
  const allRows = presentVendorExceptions(work?.items ?? [], "ALL", now);
  const vendorOptions = [...new Map(records.map(record => [record.vendor.id, record.vendor.legal_name])).entries()].sort((left, right) => left[1].localeCompare(right[1]));
  const ownerOptions = [...new Set(allRows.map(row => row.owner).filter((value): value is string => Boolean(value)))].sort();
  const ratingOptions = [...new Set(allRows.map(row => row.sourceRating).filter((value): value is string => Boolean(value)))].sort();
  const rows = presentVendorExceptions(work?.items ?? [], filter, now).filter(row => {
    const linked = records.filter(record => row.item.relationshipIDs.includes(record.relationship.id));
    return (!vendor || linked.some(record => record.vendor.id === vendor)) && (!owner || row.owner === owner) && (!rating || row.sourceRating === rating);
  });
  const sampleData = allRows.some(row => row.item.record.matter.known_facts?.sample === true);
  const pageCount = Math.max(1, Math.ceil(rows.length / pageSize));
  const visibleRows = rows.slice((page - 1) * pageSize, page * pageSize);

  useEffect(() => { if (page > pageCount) setPage(pageCount); }, [page, pageCount]);

  function select(value: VendorExceptionFilter) {
    setFilter(value); setPage(1);
    list.current?.scrollIntoView?.({ block: "start" });
  }

  function openMatter(id: string) {
    try { window.sessionStorage.setItem(scrollStorageKey, String(window.scrollY)); } catch { /* session storage is optional */ }
    onOpenMatter?.(id);
  }

  const summary = [
    { label: `${records.length} vendor ${records.length === 1 ? "service" : "services"}`, value: records.length, filter: "ALL" as const },
    { label: `${totals?.findings ?? "Unknown"} open exceptions`, value: totals?.findings, filter: "OPEN" as const },
    { label: `${totals?.openActions ?? "Unknown"} open actions`, value: totals?.openActions, filter: "OPEN" as const },
    { label: `${totals?.overdueActions ?? "Unknown"} overdue actions`, value: totals?.overdueActions, filter: "OVERDUE" as const },
  ];

  return <section className={`vendor-portfolio${detail ? " vendor-portfolio--detail" : ""}`} aria-label={detail ? "Vendor exceptions" : "Vendor exception overview"}>
    <div className="vendor-overview-summary" role="region" aria-label="Vendor overview summary">
      <div className="vendor-summary-counts">
        {summary.map((item, index) => <Button key={item.label} variant="quiet" aria-label={item.label} aria-pressed={filter === item.filter && (index !== 0 || filter === "ALL")} onPress={() => select(item.filter)}>
          <strong>{index === 0 ? item.value : item.value ?? "Unknown"}</strong><span>{item.label.replace(/^\S+\s/, "")}</span>
        </Button>)}
      </div>
      <div className="vendor-summary-context">
        {hasMore && <span>Loaded services are incomplete</span>}
        {work && <span>Checked <time dateTime={work.checkedAt}>{formatChecked(work.checkedAt)}</time></span>}
      </div>
    </div>

    <section className="vendor-exceptions" aria-labelledby="vendor-exceptions-title" ref={list}>
      <header className="vendor-exceptions-header">
        <div><div className="vendor-exceptions-heading"><h2 id="vendor-exceptions-title">Vendor exceptions</h2>{sampleData && <span className="vendor-sample-label">Sample data</span>}</div><p>{rows.length} {rows.length === 1 ? "exception matches" : "exceptions match"} the active filters</p></div>
        <div className="vendor-attention-filters" aria-label="Exception status">
          {([ ["ATTENTION", "Needs attention"], ["OVERDUE", "Overdue"], ["OPEN", "All open"], ["ALL", "All exceptions"] ] as const).map(([value, label]) => <Button key={value} variant={filter === value ? "secondary" : "quiet"} aria-pressed={filter === value} onPress={() => select(value)}>{label}</Button>)}
        </div>
        <label className="vendor-mobile-status-filter">Status<select aria-label="Exception status" value={filter} onChange={event => select(event.target.value as VendorExceptionFilter)}><option value="ATTENTION">Needs attention</option><option value="OVERDUE">Overdue</option><option value="OPEN">All open</option><option value="ALL">All exceptions</option></select></label>
        <div className="vendor-facet-filters">
          <label>Vendor<select aria-label="Vendor" value={vendor} onChange={event => { setVendor(event.target.value); setPage(1); }}><option value="">All vendors</option>{vendorOptions.map(([id, name]) => <option key={id} value={id}>{name}</option>)}</select></label>
          <label>Owner<select aria-label="Owner" value={owner} onChange={event => { setOwner(event.target.value); setPage(1); }}><option value="">All owners</option>{ownerOptions.map(value => <option key={value}>{value}</option>)}</select></label>
          <label>Source rating<select aria-label="Source rating" value={rating} onChange={event => { setRating(event.target.value); setPage(1); }}><option value="">All ratings</option>{ratingOptions.map(value => <option key={value}>{value}</option>)}</select></label>
        </div>
      </header>

      {loading && <p className="vendor-queue-state" role="status">Checking vendor exceptions…</p>}
      {error && <Notice tone="error">Vendor exceptions could not be checked. <Button onPress={() => setRetry(value => value + 1)}>Retry exceptions</Button></Notice>}
      {work && !work.complete && <Notice tone="warning">Some vendor exceptions could not be checked. Totals unknown. <Button onPress={() => setRetry(value => value + 1)}>Retry exceptions</Button></Notice>}
      {!loading && !error && rows.length === 0 && <p className="vendor-queue-state">No vendor exceptions match the current filters.</p>}

      <ol className="vendor-exception-list">
        {visibleRows.map(row => {
          const { record, relationshipIDs } = row.item;
          const services = records.filter(item => relationshipIDs.includes(item.relationship.id));
          return <li key={record.matter.id} className={`vendor-exception-row vendor-exception-row--${row.band.toLowerCase()}`}>
            <div className="vendor-exception-identity"><h3>{record.matter.title}</h3><p>{services.map(item => `${item.vendor.legal_name} · ${item.relationship.service_name}`).join("; ") || "Vendor service not recorded"}</p></div>
            <div className="vendor-exception-facts">
              <span><small>Rating</small><strong>{row.sourceRating ?? "Not recorded"}</strong></span>
              <span><small>Owner</small><strong>{row.owner ?? "Not assigned"}</strong></span>
            </div>
            <div className="vendor-exception-next"><small>Next action</small><strong>{row.nextAction?.title ?? "No open action recorded"}</strong>{row.openActionCount > 1 && <span>{row.openActionCount} open · {row.overdueActionCount} overdue</span>}</div>
            <div className="vendor-exception-state"><StatusBadge tone={bandTone(row.band)}>{bandLabel(row.band)}</StatusBadge>{row.nextAction?.due_at && validDate(row.nextAction.due_at) ? <span className={row.band === "OVERDUE" ? "vendor-action-overdue" : ""}>{deadlineLabel(row.nextAction.due_at, now)}<small>{formatDate(row.nextAction.due_at)}</small></span> : <span>No deadline recorded</span>}</div>
            <div className="vendor-exception-review"><StatusBadge tone={record.matter.status === "CLOSED" ? "success" : "neutral"}>{record.status_label || stateLabel(record.matter.status)}</StatusBadge>{onOpenMatter && <Button aria-label={`Review exception: ${record.matter.title}`} onPress={() => openMatter(record.matter.id)}>Review exception</Button>}</div>
          </li>;
        })}
      </ol>
      {rows.length > pageSize && <nav className="vendor-exception-pagination" aria-label="Exception pages"><Button variant="secondary" isDisabled={page === 1} onPress={() => { setPage(value => value - 1); list.current?.scrollIntoView?.({ block: "start" }); }}>Previous</Button><span>Page {page} of {pageCount}</span><Button variant="secondary" isDisabled={page === pageCount} onPress={() => { setPage(value => value + 1); list.current?.scrollIntoView?.({ block: "start" }); }}>Next</Button></nav>}
      {!!totals?.implementedActions && <p className="vendor-queue-state">{totals.implementedActions} implemented {totals.implementedActions === 1 ? "action requires" : "actions require"} outcome verification.</p>}
    </section>
  </section>;
}

function bandTone(band: VendorExceptionBand) { return band === "OVERDUE" || band === "BLOCKED" ? "warning" : band === "CLOSED" ? "success" : "neutral"; }
function bandLabel(band: VendorExceptionBand) { const labels: Record<VendorExceptionBand, string> = { OVERDUE: "Overdue", BLOCKED: "Blocked", INCOMPLETE: "Assignment incomplete", DUE_SOON: "Due soon", OPEN: "Open", CLOSED: "Closed" }; return labels[band]; }
function stateLabel(value: string) { const labels: Record<string, string> = { TRIAGE: "Initial review", ASSESSMENT: "Assessment", DECISION_REQUIRED: "Decision needed", ACTION_IN_PROGRESS: "Actions in progress", VERIFICATION: "Outcome verification", CLOSED: "Closed", CANCELLED: "Cancelled" }; return labels[value] ?? "Status unknown"; }
function formatDate(value: string) { return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date(value)); }
function formatChecked(value: string) { return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" }).format(new Date(value)); }
function validDate(value: string) { return Number.isFinite(Date.parse(value)); }
function deadlineLabel(value: string, now: number) {
  const deadline = Date.parse(value);
  const days = Math.max(0, Math.ceil(Math.abs(deadline - now) / (24 * 60 * 60 * 1000)));
  if (deadline < now) return `${days} ${days === 1 ? "day" : "days"} overdue`;
  if (days === 0) return "Due today";
  return `Due in ${days} ${days === 1 ? "day" : "days"}`;
}
