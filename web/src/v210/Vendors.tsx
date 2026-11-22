import { useEffect, useRef, useState, type FormEvent } from "react";
import { createVendorRelationship, loadVendorRelationship, loadVendorRelationships } from "../vendorApi";
import type { RuntimeContext } from "../api";
import { ApiError, requestJSON } from "../http";
import type { CreateVendorRelationshipInput, VendorRelationshipAggregate } from "../vendorTypes";
import { VendorReview } from "./VendorReview";
import { documentReturnDestination } from "./documentReturn";
import "./vendor-intake.css";
import "./vendor-register.css";

type PendingRelationship = Pick<CreateVendorRelationshipInput, "legal_name" | "service_name" | "criticality" | "privacy_role">;

export function Vendors({ context, relationshipId, assessmentId }: { context: RuntimeContext; relationshipId?: string; assessmentId?: string }) {
  const scopeKey = JSON.stringify([context.tenant.id, context.legal_entity.id, context.actor.id]);
  return <VendorWorkspace key={scopeKey} context={context} relationshipId={relationshipId} assessmentId={assessmentId}/>;
}

function VendorWorkspace({ context, relationshipId, assessmentId }: { context: RuntimeContext; relationshipId?: string; assessmentId?: string }) {
  const scopeKey = JSON.stringify([context.tenant.id, context.legal_entity.id, context.actor.id]);
  const currentScope = useRef(scopeKey); currentScope.current = scopeKey;
  const authorityScope = useRef("");
  const [authority, setAuthority] = useState<"checking" | "allowed" | "unavailable">("checking");
  const [uncertainState, setUncertain] = useState<{ scope: string; value: PendingRelationship }>();
  const uncertain = uncertainState?.scope === scopeKey ? uncertainState.value : undefined;
  const [mode, setMode] = useState<"list" | "create" | "created" | "uncertain">("list");
  const [items, setItems] = useState<VendorRelationshipAggregate[]>([]);
  const [cursor, setCursor] = useState<string>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [listScope, setListScope] = useState("");
  const [draft, setDraft] = useState("");
  const [search, setSearch] = useState("");
  const [view, setView] = useState<"all" | "onboarding" | "active" | "inactive">("all");
  const [attempt, setAttempt] = useState(0);
  const [moreBusy, setMoreBusy] = useState(false);
  const [moreError, setMoreError] = useState(false);
  const [created, setCreated] = useState<VendorRelationshipAggregate>();
  const [selected, setSelected] = useState<VendorRelationshipAggregate>();
  const [selectedState, setSelectedState] = useState<"loading" | "ready" | "error">("loading");
  const [selectedScope, setSelectedScope] = useState("");
  const [selectedAttempt, setSelectedAttempt] = useState(0);
  const generation = useRef(0);
  useEffect(() => {
    let active = true; setAuthority("checking");
    if (!context.tenant.id || !context.legal_entity.id || context.legal_entity.id === "*" || !context.actor.id || context.actor.kind?.toUpperCase() === "SERVICE") { setAuthority("unavailable"); return () => { active = false; }; }
    const input = { tenant_id: context.tenant.id, legal_entity_id: context.legal_entity.id, object_type: "VENDOR_RELATIONSHIP", object_id: context.legal_entity.id, responsibility: "ACCOUNTABLE_OWNER", decision_type: "thirdparty.relationship.create", materiality: 3 };
    requestJSON<{ principal?: { id: string }; candidate_principals?: { id: string }[] }>(import.meta.env.VITE_API_BASE_URL ?? "", "/api/v1/authority/resolve", { method: "POST", body: JSON.stringify(input) }).then((route) => { if (active && currentScope.current === scopeKey) { authorityScope.current = scopeKey; setAuthority(route.principal?.id === context.actor.id || route.candidate_principals?.some((candidate) => candidate.id === context.actor.id) ? "allowed" : "unavailable"); } }).catch(() => { if (active && currentScope.current === scopeKey) setAuthority("unavailable"); });
    return () => { active = false; };
  }, [scopeKey]);
  useEffect(() => {
    if (!relationshipId) return;
    let active = true; setSelected(undefined); setSelectedScope(scopeKey); setSelectedState("loading");
    loadVendorRelationship(relationshipId).then((value) => { if (active && currentScope.current === scopeKey) { if (value?.relationship?.id !== relationshipId || !inScope(value, context)) { setSelectedState("error"); return; } setSelected(value); setSelectedState("ready"); } }).catch(() => { if (active && currentScope.current === scopeKey) setSelectedState("error"); });
    return () => { active = false; };
  }, [relationshipId, selectedAttempt, scopeKey]);
  useEffect(() => {
    if (mode !== "list" || relationshipId) return;
    const token = ++generation.current; setListScope(scopeKey); setState("loading"); setItems([]); setCursor(undefined); setMoreBusy(false); setMoreError(false);
    loadVendorRelationships({ search, limit: 20 }).then((page) => { if (token === generation.current && currentScope.current === scopeKey) { if (!Array.isArray(page.items) || page.items.some((value) => !inScope(value, context))) { setState("error"); return; } setItems(page.items); setCursor(page.next_cursor); setState("ready"); } }).catch(() => { if (token === generation.current && currentScope.current === scopeKey) setState("error"); });
    return () => { generation.current++; };
  }, [mode, search, attempt, relationshipId, scopeKey]);
  async function more() {
    if (!cursor || moreBusy) return;
    const token = generation.current; setMoreBusy(true); setMoreError(false);
    try { const page = await loadVendorRelationships({ search, cursor, limit: 20 }); if (token === generation.current && currentScope.current === scopeKey) { if (!Array.isArray(page.items) || page.items.some((value) => !inScope(value, context))) throw new Error("scope mismatch"); setItems((value) => [...value, ...page.items]); setCursor(page.next_cursor); } }
    catch { if (token === generation.current && currentScope.current === scopeKey) setMoreError(true); }
    finally { if (token === generation.current && currentScope.current === scopeKey) setMoreBusy(false); }
  }
  if (relationshipId) {
    if (selectedScope !== scopeKey || selectedState === "loading") return <p role="status">Loading vendor…</p>;
    if (selectedState === "error" || !selected) return <section><h1>Vendor unavailable</h1><button onClick={() => setSelectedAttempt((value) => value + 1)}>Try again</button><p><a href={documentReturnDestination() ?? "#/vendors"}>{documentReturnDestination() ? "View documents" : "View vendors"}</a></p></section>;
    return <VendorReview key={`${relationshipId}:${assessmentId ?? "current"}`} relationship={selected} assessmentId={assessmentId} backLabel={documentReturnDestination() ? "Documents" : "Vendors"} onActivated={(value) => setSelected((current) => current ? { ...current, relationship: value } : current)} onBack={() => { setMode("list"); window.location.hash = documentReturnDestination() ?? "#/vendors"; }}/>;
  }
  const canCreate = authority === "allowed" && authorityScope.current === scopeKey;
  if (mode === "create" && canCreate) return <VendorIntake context={context} scopeKey={scopeKey} currentScope={currentScope} onCancel={() => setMode("list")} onCreated={(value) => { setCreated(value); setMode("created"); }} onUncertain={(value) => { setUncertain({ scope: scopeKey, value }); setMode("uncertain"); }}/>;
  if (mode === "uncertain" && uncertain) return <section><h1>Relationship status unknown</h1><p role="alert">Check vendors before trying again.</p><RelationshipDetails value={uncertain}/><button className="v-primary" onClick={() => { setDraft(uncertain.legal_name); setSearch(uncertain.legal_name); setMode("list"); }}>View vendors</button></section>;
  if (mode === "created" && created) return <section><h1>Relationship created</h1><div className="v-panel v-result"><h2>{created.vendor.legal_name}</h2><p>{created.relationship.service_name}</p><span className="v-status">{statusLabel(created.relationship.status)}</span></div><p>Due diligence and approval are required before activation.</p>{created.relationship.id && <p><a className="v-primary" href={`#/vendors/${encodeURIComponent(created.relationship.id)}`}>View relationship</a></p>}<button onClick={() => setMode("list")}>View vendors</button></section>;
  const loaded = listScope === scopeKey && state === "ready";
  const visible = items.filter(({ relationship }) => view === "all" || (view === "onboarding" && ["PROPOSED", "UNDER_REVIEW"].includes(relationship.status)) || (view === "active" && relationship.status === "ACTIVE") || (view === "inactive" && ["RESTRICTED", "SUSPENDED", "EXITING", "TERMINATED"].includes(relationship.status)));
  const metrics = [
    { label: "Loaded suppliers", value: new Set(items.map(({ vendor }) => vendor.id)).size, detail: loaded ? `${items.length} relationships` : "" },
    { label: "Onboarding", value: items.filter(({ relationship }) => ["PROPOSED", "UNDER_REVIEW"].includes(relationship.status)).length, detail: "Proposed or under review" },
    { label: "Critical services", value: items.filter(({ relationship }) => relationship.criticality === "CRITICAL").length, detail: "" },
    { label: "Restricted or exited", value: items.filter(({ relationship }) => ["RESTRICTED", "SUSPENDED", "EXITING", "TERMINATED"].includes(relationship.status)).length, detail: "Restricted, suspended, exiting or terminated" },
  ];
  const views = [{ id: "all", label: "Engagements" }, { id: "onboarding", label: "Onboarding" }, { id: "active", label: "Active" }, { id: "inactive", label: "Restricted & exited" }] as const;
  return <section className="v-vendors"><header className="v-page-heading"><div><p className="v-eyebrow">Third-party risk</p><h1>Vendor relationships</h1></div>{canCreate && !uncertain && <button className="v-primary" onClick={() => setMode("create")}>Add vendor</button>}</header>
    {uncertain && <section className="v-panel"><h2>Relationship status unknown</h2><p>Check vendors before trying again.</p><RelationshipDetails value={uncertain}/><button type="button" onClick={() => setUncertain(undefined)}>Dismiss details</button></section>}
    {authority === "checking" && <p role="status">Checking vendor creation access…</p>}{authority === "unavailable" && <p>Vendor creation unavailable.</p>}
    <div className="v-vendor-metrics" aria-label="Loaded vendor relationships">{metrics.map((metric) => <article className="v-panel v-vendor-metric" key={metric.label}><span>{metric.label}</span><strong>{loaded ? metric.value : "—"}</strong>{metric.detail && <small>{metric.detail}</small>}</article>)}</div>
    <p className="v-vendor-coverage">Counts cover loaded relationships.</p>
    <nav className="v-vendor-tabs" aria-label="Vendor relationship filters">{views.map((item) => <button key={item.id} type="button" aria-pressed={view === item.id} onClick={() => setView(item.id)}>{item.label}</button>)}</nav>
    <form className="v-program-search v-vendor-search" onSubmit={(event) => { event.preventDefault(); setSearch(draft.trim()); }}><label htmlFor="v-vendor-search">Find a vendor</label><div><input id="v-vendor-search" value={draft} onChange={(event) => setDraft(event.target.value)} placeholder="Vendor or service"/><button>Search</button></div>{cursor && <p>Filters apply to loaded relationships.</p>}</form>
    {(listScope !== scopeKey || state === "loading") && <p role="status">Loading vendors…</p>}
    {listScope === scopeKey && state === "error" && <div className="v-panel"><p role="alert">Vendors unavailable.</p><button onClick={() => setAttempt(attempt + 1)}>Try again</button></div>}
    {loaded && <section className="v-panel v-vendor-register"><div className="v-vendor-register-head"><h2>Service engagements</h2><span>{visible.length} loaded</span></div><div className="v-program-table">{items.length === 0 ? <p>No vendors match this search.</p> : visible.length === 0 ? <p>No loaded relationships match this filter.</p> : <table><thead><tr><th>Supplier / service</th><th>Criticality</th><th>Lifecycle</th><th>Renewal</th><th><span className="v-sr-only">Action</span></th></tr></thead><tbody>{visible.map(({ vendor, relationship }) => <tr key={relationship.id}><td><a href={`#/vendors/${encodeURIComponent(relationship.id)}`}>{vendor.legal_name}</a><small>{relationship.service_name}</small></td><td><span className="v-mobile-label">Criticality</span>{statusLabel(relationship.criticality)}</td><td><span className="v-mobile-label">Lifecycle</span><span className={`v-vendor-state ${vendorStateClass(relationship.status)}`}>{statusLabel(relationship.status)}</span></td><td><span className="v-mobile-label">Renewal</span>{relationship.renewal_at ? vendorDate(relationship.renewal_at) : "Not recorded"}</td><td><a className="v-vendor-open" href={`#/vendors/${encodeURIComponent(relationship.id)}`} aria-label={`Open ${vendor.legal_name} ${relationship.service_name}`}>Open</a></td></tr>)}</tbody></table>}{moreError && <p role="alert">More vendors unavailable.</p>}{cursor && <button disabled={moreBusy} onClick={() => void more()}>{moreBusy ? "Loading…" : "Load more"}</button>}</div></section>}
  </section>;
}

function inScope(value: VendorRelationshipAggregate, context: RuntimeContext) {
  return Boolean(context.tenant.id && context.legal_entity.id && context.legal_entity.id !== "*" && value?.vendor?.id && value.vendor.tenant_id === context.tenant.id && value.relationship?.id && value.relationship.vendor_id === value.vendor.id && value.relationship.tenant_id === context.tenant.id && value.relationship.legal_entity_id === context.legal_entity.id);
}

function VendorIntake({ context, scopeKey, currentScope, onCancel, onCreated, onUncertain }: { context: RuntimeContext; scopeKey: string; currentScope: React.RefObject<string>; onCancel: () => void; onCreated: (value: VendorRelationshipAggregate) => void; onUncertain: (value: PendingRelationship) => void }) {
  const [name, setName] = useState(""); const [service, setService] = useState("");
  const [candidates, setCandidates] = useState<VendorRelationshipAggregate[]>([]);
  const [finding, setFinding] = useState(false); const [findError, setFindError] = useState(""); const [hasMore, setHasMore] = useState(false);
  const searchToken = useRef(0);
  const [step, setStep] = useState<0 | 1 | 2>(0);
  const [criticality, setCriticality] = useState<"" | CreateVendorRelationshipInput["criticality"]>("");
  const [privacy, setPrivacy] = useState<"" | CreateVendorRelationshipInput["privacy_role"]>("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const mounted = useRef(true);
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  const lock = useRef(false);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { heading.current?.focus(); }, [step]);
  async function findVendors() {
    if (finding || name.trim().length < 2) return;
    const token = ++searchToken.current; setFinding(true); setFindError(""); setCandidates([]); setHasMore(false);
    try {
      const page = await loadVendorRelationships({ search: name.trim(), limit: 20 });
      if (!mounted.current || currentScope.current !== scopeKey || token !== searchToken.current) return;
      if (!Array.isArray(page.items) || page.items.length > 20 || page.items.some((item) => !item.vendor?.id || !item.vendor.legal_name || item.vendor.tenant_id !== context.tenant.id || !Number.isSafeInteger(item.vendor.version) || item.vendor.version < 1 || !item.relationship?.id || !item.relationship.service_name || item.relationship.vendor_id !== item.vendor.id || item.relationship.tenant_id !== context.tenant.id || item.relationship.legal_entity_id !== context.legal_entity.id || !Number.isSafeInteger(item.relationship.version) || item.relationship.version < 1)) throw new Error("scope mismatch");
      setCandidates(page.items); setHasMore(Boolean(page.next_cursor));
      if (!page.items.length) setFindError("No matching vendors.");
    } catch { if (mounted.current && currentScope.current === scopeKey && token === searchToken.current) setFindError("Vendor search unavailable."); }
    finally { if (mounted.current && currentScope.current === scopeKey && token === searchToken.current) setFinding(false); }
  }
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (lock.current || busy || !name.trim()) return;
    if (step === 0) { setStep(1); return; }
    if (!service.trim() || !criticality || !privacy) return;
    if (step === 1) { setStep(2); return; }
    lock.current = true; setError("");
    setBusy(true);
    try {
      const result = await createVendorRelationship({ legal_name: name.trim(), service_name: service.trim(), criticality, privacy_role: privacy });
      if (!mounted.current || currentScope.current !== scopeKey) return;
      if (!result?.vendor?.id || !result.relationship?.id || result.vendor.tenant_id !== context.tenant.id || result.relationship.tenant_id !== context.tenant.id || result.relationship.legal_entity_id !== context.legal_entity.id || result.relationship.business_owner_principal_id !== context.actor.id || result.relationship.vendor_id !== result.vendor.id || result.vendor.version !== 1 || result.vendor.legal_name !== name.trim() || result.relationship.service_name !== service.trim() || result.relationship.criticality !== criticality || result.relationship.privacy_role !== privacy || result.relationship.status !== "PROPOSED" || result.relationship.version !== 1) throw new Error("unconfirmed");
      onCreated(result);
    }
    catch (cause) { if (!mounted.current || currentScope.current !== scopeKey) return; if (cause instanceof ApiError && [400, 403, 422].includes(cause.status)) { lock.current = false; setBusy(false); setError(cause.message); } else onUncertain({ legal_name: name.trim(), service_name: service.trim(), criticality, privacy_role: privacy }); }
  }
  return <section><button className="v-back v-back-button" onClick={onCancel} disabled={busy}>← Vendors</button><header className="v-page-heading"><div><p className="v-eyebrow">Vendor intake · Step {step + 1} of 3</p><h1>Add a vendor relationship</h1></div></header><form className="v-panel v-intake" onSubmit={(event) => void submit(event)}>
    {step === 0 && <><h2 ref={heading} tabIndex={-1}>Vendor details</h2><label>Legal name<input required value={name} onChange={(event) => { searchToken.current++; setName(event.target.value); setCandidates([]); setFindError(""); setHasMore(false); setFinding(false); }} /></label><button type="button" disabled={finding || name.trim().length < 2} onClick={() => void findVendors()}>{finding ? "Searching…" : "Check existing vendors"}</button>{findError && <p role="status">{findError}</p>}{hasMore && <p>More vendors match. Refine the name.</p>}{candidates.length > 0 && <><p>Matching vendor relationships</p><ul className="v-intake-candidates">{candidates.map((candidate) => <li key={candidate.relationship.id}><a href={`#/vendors/${encodeURIComponent(candidate.relationship.id)}`} target="_blank" rel="noopener noreferrer">{candidate.vendor.legal_name} · {candidate.relationship.service_name} (opens in new tab)</a></li>)}</ul></>}</>}
    {step === 1 && <><h2 ref={heading} tabIndex={-1}>Service details</h2><label>Service<input required value={service} onChange={(event) => setService(event.target.value)} /></label><label>Criticality<select required value={criticality} onChange={(event) => setCriticality(event.target.value as typeof criticality)}><option value="">Select criticality</option><option value="STANDARD">Standard</option><option value="IMPORTANT">Important</option><option value="CRITICAL">Critical</option></select></label><label>Privacy role<select required value={privacy} onChange={(event) => setPrivacy(event.target.value as typeof privacy)}><option value="">Select privacy role</option><option value="NONE">None</option><option value="PROCESSOR">Processor</option><option value="JOINT_CONTROLLER">Joint controller</option></select></label></>}
    {step === 2 && <><h2 ref={heading} tabIndex={-1}>Review relationship</h2><dl className="v-intake-review"><div><dt>Legal name</dt><dd>{name.trim()}</dd></div><div><dt>Service</dt><dd>{service.trim()}</dd></div><div><dt>Criticality</dt><dd>{statusLabel(criticality)}</dd></div><div><dt>Privacy role</dt><dd>{statusLabel(privacy)}</dd></div></dl><div className="v-intake-edit"><button type="button" disabled={busy} onClick={() => setStep(0)}>Edit vendor details</button><button type="button" disabled={busy} onClick={() => setStep(1)}>Edit service details</button></div><p>Due diligence and approval are required before activation.</p></>}
    {error && <p role="alert">{error}</p>}<div className="v-intake-actions"><button type="button" onClick={step === 0 ? onCancel : () => setStep(step === 2 ? 1 : 0)} disabled={busy}>{step === 0 ? "Cancel" : "Back"}</button><button className="v-primary" disabled={busy}>{busy ? "Creating…" : step === 0 ? "Continue" : step === 1 ? "Review relationship" : "Create relationship"}</button></div></form></section>;
}

function RelationshipDetails({ value }: { value: PendingRelationship }) { return <dl className="v-intake-review"><div><dt>Legal name</dt><dd>{value.legal_name}</dd></div><div><dt>Service</dt><dd>{value.service_name}</dd></div><div><dt>Criticality</dt><dd>{statusLabel(value.criticality)}</dd></div><div><dt>Privacy role</dt><dd>{statusLabel(value.privacy_role)}</dd></div></dl>; }

function statusLabel(value: string) { const text = value.toLowerCase().replaceAll("_", " "); return text.charAt(0).toUpperCase() + text.slice(1); }
function vendorStateClass(value: string) { return value === "ACTIVE" ? "is-active" : ["RESTRICTED", "SUSPENDED", "TERMINATED"].includes(value) ? "is-attention" : "is-pending"; }
function vendorDate(value: string) { const date = new Date(value); return Number.isNaN(date.getTime()) ? "Date unavailable" : new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeZone: "UTC" }).format(date); }
