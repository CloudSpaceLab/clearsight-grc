import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { presentVendorExceptions, type VendorExceptionBand, type VendorExceptionFilter } from "../vendorExceptionPresentation";
import { loadVendorRiskWork, summarizeVendorRiskWork, type VendorRiskWork } from "../vendorRiskWork";
import { Button, Notice, RankedBarList, SelectableRecord, StatusBadge } from "./ui";
import "./vendor-portfolio.css";

type StoredView = { filter: VendorExceptionFilter; vendor: string; owner: string; rating: string; exceptionsOpen: boolean };
const viewStorageKey = "clearsight.vendor-exception-view";
const scrollStorageKey = "clearsight.vendor-exception-scroll";

function storedScroll(): { top: number } | null {
  try {
    const value = window.sessionStorage.getItem(scrollStorageKey);
    const top = Number(value);
    if (value !== null && Number.isFinite(top) && top >= 0) return { top };
  } catch { /* session storage is optional */ }
  return null;
}

function storedView(): StoredView {
  try {
    const value = JSON.parse(window.sessionStorage.getItem(viewStorageKey) ?? "null") as Partial<StoredView> | null;
    if (value && ["ATTENTION", "OVERDUE", "OPEN", "ALL"].includes(value.filter ?? "")) {
      return { filter: value.filter!, vendor: value.vendor ?? "", owner: value.owner ?? "", rating: value.rating ?? "", exceptionsOpen: value.exceptionsOpen === true };
    }
  } catch { /* session storage is optional */ }
  return { filter: "ATTENTION", vendor: "", owner: "", rating: "", exceptionsOpen: false };
}

export function VendorPortfolio({ records, hasMore, onOpenMatter, detail = false, refreshKey = 0, register, registerLoading = false, onOpenVendor }: {
  records: VendorRelationshipAggregate[]; hasMore: boolean; onOpenMatter?: (id: string) => void; detail?: boolean; refreshKey?: number; register?: ReactNode; registerLoading?: boolean; onOpenVendor?: (record: VendorRelationshipAggregate) => void;
}) {
  const initialView = useMemo(() => detail ? { filter: "OPEN" as const, vendor: "", owner: "", rating: "", exceptionsOpen: false } : storedView(), [detail]);
  const [work, setWork] = useState<VendorRiskWork>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [retry, setRetry] = useState(0);
  const [filter, setFilter] = useState<VendorExceptionFilter>(initialView.filter);
  const [vendor, setVendor] = useState(initialView.vendor);
  const [owner, setOwner] = useState(initialView.owner);
  const [rating, setRating] = useState(initialView.rating);
  const [page, setPage] = useState(1);
  const [exceptionsOpen, setExceptionsOpen] = useState(initialView.exceptionsOpen);
  const pageSize = 20;
  const list = useRef<HTMLElement>(null);
  const pendingScroll = useRef<{ top: number } | null | undefined>(undefined);
  const ids = records.map(record => record.relationship.id).join(",");

  useEffect(() => {
    let current = true;
    setLoading(true); setError(false); setWork(undefined);
    void loadVendorRiskWork(ids ? ids.split(",") : []).then(value => { if (current) setWork(value); }).catch(() => { if (current) setError(true); }).finally(() => { if (current) setLoading(false); });
    return () => { current = false; };
  }, [ids, refreshKey, retry]);

  useEffect(() => {
    if (!detail) try { window.sessionStorage.setItem(viewStorageKey, JSON.stringify({ filter, vendor, owner, rating, exceptionsOpen })); } catch { /* session storage is optional */ }
  }, [detail, filter, vendor, owner, rating, exceptionsOpen]);

  useEffect(() => {
    // Both the register and queue affect the page height. Restoring on mount
    // clamps a deep position before their rows exist and loses the return row.
    if (detail || loading || registerLoading) return;
    if (pendingScroll.current === undefined) pendingScroll.current = storedScroll();
    const position = pendingScroll.current;
    if (!position) return;
    const frame = window.requestAnimationFrame(() => {
      window.scrollTo({ top: position.top });
      pendingScroll.current = null;
      try { window.sessionStorage.removeItem(scrollStorageKey); } catch { /* session storage is optional */ }
    });
    // Keep the intent if a frame is cancelled, including StrictMode cleanup.
    return () => window.cancelAnimationFrame(frame);
  }, [detail, loading, registerLoading, work]);

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
    if (register && !detail) setExceptionsOpen(true);
    requestAnimationFrame(() => list.current?.scrollIntoView?.({ block: "start" }));
  }

  function openMatter(id: string) {
    if (!detail) try { window.sessionStorage.setItem(scrollStorageKey, String(window.scrollY)); } catch { /* session storage is optional */ }
    onOpenMatter?.(id);
  }

  // Only the loaded relationship page is represented here. A partial linked
  // Matter query cannot establish a bank-wide exception distribution.
  const criticalities = (["CRITICAL", "IMPORTANT", "STANDARD", "UNSPECIFIED"] as const).map(value => ({
    label: value === "UNSPECIFIED" ? "Not recorded" : value === "CRITICAL" ? "Critical" : value === "IMPORTANT" ? "Important" : "Standard",
    value,
    count: records.filter(item => (item.relationship.criticality ?? "UNSPECIFIED") === value).length,
  }));
  const stageLabels = [
    ["DRAFT", "Draft"],
    ["TRIAGE", "Initial review"],
    ["ASSESSMENT", "Assessment"],
    ["DECISION_REQUIRED", "Decision needed"],
    ["RESPONSE_PREPARATION", "Response preparation"],
    ["ACTION_IN_PROGRESS", "Actions underway"],
    ["VERIFICATION", "Outcome checks"],
  ] as const;
  const openFindings = work?.complete ? work.items.filter(item => !["CLOSED", "CANCELLED"].includes(item.record.matter.status)) : undefined;
  const unmappedStages = openFindings?.filter((item) => !stageLabels.some(([status]) => item.record.matter.status === status)).length ?? 0;
  const stages = [...stageLabels.map(([status, label]) => ({
    status, label, count: openFindings?.filter(item => item.record.matter.status === status).length,
  })), ...(unmappedStages ? [{ status: "OTHER", label: "Other recorded stage", count: unmappedStages }] : [])];
  const attention = work?.complete ? presentVendorExceptions(work.items, "ATTENTION", now).slice(0, 3) : undefined;
  const renewals = records.filter((record) => record.relationship.renewal_at && validDate(record.relationship.renewal_at))
    .sort((left, right) => Date.parse(left.relationship.renewal_at!) - Date.parse(right.relationship.renewal_at!)).slice(0, 3);
  const isPortfolio = Boolean(register) && !detail;

  const summary = [
    { label: `${records.length} vendor ${records.length === 1 ? "service" : "services"}`, value: records.length, filter: "ALL" as const },
    { label: `${totals?.findings ?? "Unknown"} open exceptions`, value: totals?.findings, filter: "OPEN" as const },
    { label: `${totals?.openActions ?? "Unknown"} open actions`, value: totals?.openActions, filter: "OPEN" as const },
    { label: `${totals?.overdueActions ?? "Unknown"} overdue actions`, value: totals?.overdueActions, filter: "OVERDUE" as const },
  ];

  const exceptionList = <section className="vendor-exceptions" aria-labelledby="vendor-exceptions-title" ref={list}>
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
      {!isPortfolio && error && <Notice tone="error">Vendor exceptions could not be checked. <Button onPress={() => setRetry(value => value + 1)}>Retry exceptions</Button></Notice>}
      {!isPortfolio && work && !work.complete && <Notice tone="warning">Some vendor exceptions could not be checked. Totals unknown. <Button onPress={() => setRetry(value => value + 1)}>Retry exceptions</Button></Notice>}
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
    </section>;

  return <section className={`vendor-portfolio${detail ? " vendor-portfolio--detail" : ""}${isPortfolio ? " vendor-portfolio-overview" : ""}`} aria-label={detail ? "Vendor exceptions" : "Vendor exception overview"}>
    <div className="vendor-overview-summary" role="region" aria-label="Vendor overview summary">
      <div className="vendor-summary-counts">
        {summary.map((item, index) => <Button key={item.label} variant="quiet" aria-label={item.label} aria-pressed={filter === item.filter && (index !== 0 || filter === "ALL")} onPress={() => isPortfolio && index === 0 ? document.getElementById("vendor-portfolio-register")?.scrollIntoView?.({ block: "start" }) : select(item.filter)}>
          <strong>{index === 0 ? item.value : item.value ?? "Unknown"}</strong><span>{item.label.replace(/^\S+\s/, "")}</span>
        </Button>)}
      </div>
      <div className="vendor-summary-context">
        {isPortfolio && sampleData && <span>Includes sample exceptions</span>}
        {hasMore && <span>Loaded services are incomplete</span>}
        {work && <span>Checked <time dateTime={work.checkedAt}>{formatChecked(work.checkedAt)}</time></span>}
      </div>
    </div>

    {isPortfolio && <div className="vendor-portfolio-body"><div className="vendor-portfolio-main">
      <div className="vendor-overview-insights">
        <section className="vendor-portfolio-card" aria-labelledby="vendor-criticality-heading"><header><div><h2 id="vendor-criticality-heading">Service criticality</h2><p>{records.length} loaded relationships</p></div></header>
          <RankedBarList ariaLabel="Service criticality distribution" layout="inline" items={criticalities.map((item) => ({ id: item.value, label: item.label, value: item.count }))}/>
        </section>
        <section className="vendor-portfolio-card" aria-labelledby="vendor-stages-heading"><header><div><h2 id="vendor-stages-heading">Exception stages</h2><p>{openFindings ? `${openFindings.length} open exceptions` : "Linked issues"}</p></div></header>
          {openFindings ? openFindings.length ? <RankedBarList ariaLabel="Open exception stages" layout="inline" items={stages.filter((item) => item.count).map((item) => ({ id: item.status, label: item.label, value: item.count ?? 0 }))}/> : <p>No open exceptions in the loaded linked issues.</p> : <p>Stage totals unavailable until linked issues are checked.</p>}
        </section>
      </div>
      {register}
    </div><aside className="vendor-portfolio-rail" aria-label="Vendor attention and renewals">
      <section className="vendor-portfolio-card" aria-labelledby="vendor-attention-heading"><header><div><h2 id="vendor-attention-heading">Needs attention</h2><p>Loaded linked exceptions</p></div></header>
        {loading ? <p role="status">Checking vendor exceptions…</p> : error ? <Notice tone="error">Vendor exceptions could not be checked. <Button onPress={() => setRetry(value => value + 1)}>Retry exceptions</Button></Notice>
          : work && !work.complete ? <Notice tone="warning">Some vendor exceptions could not be checked. Totals unknown. <Button onPress={() => setRetry(value => value + 1)}>Retry exceptions</Button></Notice>
            : attention?.length ? <ul className="vendor-attention-preview">{attention.map((row) => <li key={row.item.record.matter.id}>{onOpenMatter ? <SelectableRecord variant="list" title={row.item.record.matter.title} metadata={bandLabel(row.band)} aria-label={`Open attention: ${row.item.record.matter.title}`} onPress={() => openMatter(row.item.record.matter.id)}/> : <div><strong>{row.item.record.matter.title}</strong><span>{bandLabel(row.band)}</span></div>}</li>)}</ul> : <p>No matching attention items in loaded linked issues.</p>}
        <Button variant="quiet" onPress={() => select("ATTENTION")}>View exceptions</Button>
      </section>
      <section className="vendor-portfolio-card" aria-labelledby="vendor-renewals-heading"><header><div><h2 id="vendor-renewals-heading">Service renewals</h2><p>Recorded dates in loaded relationships</p></div></header>
        {renewals.length ? <ul className="vendor-renewal-list">{renewals.map((record) => <li key={record.relationship.id}>{onOpenVendor ? <SelectableRecord variant="list" title={record.vendor.legal_name} metadata={`${record.relationship.service_name} · ${formatDate(record.relationship.renewal_at!)}`} aria-label={`Open renewal for ${record.vendor.legal_name}, ${record.relationship.service_name}`} onPress={() => onOpenVendor(record)}/> : <><strong>{record.vendor.legal_name}</strong><time dateTime={record.relationship.renewal_at}>{formatDate(record.relationship.renewal_at!)}</time></>}</li>)}</ul> : <p>No renewal dates are recorded in the loaded relationships.</p>}
      </section>
    </aside></div>}

    {!detail && !isPortfolio && <div className="vendor-overview-panels">
      <section aria-labelledby="vendor-criticality-heading">
        <header><h2 id="vendor-criticality-heading">Service criticality</h2><span>Loaded relationships</span></header>
        <strong>{records.length} services</strong>
        <div className="vendor-overview-bars">{criticalities.map(item => <div key={item.value}>
          <div><span>{item.label}</span><span>{item.count}</span></div>
          <div className="vendor-overview-track"><span data-criticality={item.value}
            style={{ width: records.length ? `${item.count / records.length * 100}%` : "0%" }}/></div>
        </div>)}</div>
        {hasMore && <p>More relationships are available. Counts cover loaded records.</p>}
      </section>
      <section aria-labelledby="vendor-stages-heading">
        <header><h2 id="vendor-stages-heading">Exception stages</h2><span>Linked issues</span></header>
        {openFindings ? <>
          <strong>{openFindings.length} open exceptions</strong>
          <div className="vendor-overview-bars">{stages.map(item => <div key={item.status}>
            <div><span>{item.label}</span><span>{item.count}</span></div>
            <div className="vendor-overview-track"><span
              style={{ width: openFindings.length ? `${(item.count ?? 0) / openFindings.length * 100}%` : "0%" }}/></div>
          </div>)}</div>
        </> : <p>Stage totals unavailable until linked issues are checked.</p>}
      </section>
      <section aria-labelledby="vendor-attention-heading">
        <header><h2 id="vendor-attention-heading">Needs attention</h2><span>Linked exceptions</span></header>
        {attention === undefined ? <p>Attention items unavailable until linked issues are checked.</p>
          : attention.length === 0 ? <p>No matching attention items in loaded linked issues.</p>
          : <ul className="vendor-attention-preview">{attention.map(row => <li key={row.item.record.matter.id}>
            <div><strong>{row.item.record.matter.title}</strong><span>{bandLabel(row.band)}</span></div>
            {onOpenMatter && <Button variant="quiet" size="compact"
              aria-label={`Open attention: ${row.item.record.matter.title}`}
              onPress={() => openMatter(row.item.record.matter.id)}>Review</Button>}
          </li>)}</ul>}
        <Button variant="quiet" onPress={() => select("ATTENTION")}>View exceptions</Button>
      </section>
    </div>}

    {isPortfolio ? <details className="vendor-portfolio-exceptions" open={exceptionsOpen} onToggle={(event) => setExceptionsOpen(event.currentTarget.open)}><summary>Vendor exceptions</summary>{exceptionList}</details> : exceptionList}
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
