import { useEffect, useRef, useState, type FormEvent } from "react";
import { loadContext, loadDemoAccounts, loadProgram, loadSessionStatus, loginDemo, type DemoAccount, type RuntimeContext } from "../api";
import { apiErrorKind, SESSION_ENDED_EVENT } from "../http";
import type { ProgramAggregate } from "../types";
import { loadFormTemplateItem } from "../formsApi";
import type { FormLibraryItem } from "../formsTypes";
import { FormsRegister } from "./FormsRegister";
import { Documents } from "./Documents";
import { Vendors } from "./Vendors";
import { Responses } from "./Responses";
import { FormCreator } from "./FormCreator";
import { FormActions } from "./FormActions";
import { FormEditor } from "./FormEditor";
import { APIQuestionWizard } from "./APIQuestionWizard";
import { Frameworks } from "./Frameworks";
import { Home } from "./Home";
import { RiskWorkspace } from "./RiskWorkspace";
import { ProgramRequirements } from "./ProgramRequirements";
import { ProgramCoverage } from "./ProgramCoverage";
import { ProgramIssues } from "./ProgramIssues";
import { ProgramCreator } from "./ProgramCreator";
import { ProgramsRegister } from "./ProgramsRegister";
import { PrivacyWorkspace } from "./PrivacyWorkspace";
import { Reports } from "./Reports";
import { RCSAWorkspace } from "./RCSAWorkspace";
import { KeyIndicators } from "./KeyIndicators";
import { LossWorkspace } from "./LossWorkspace";
import { ActorWorkWorkspace } from "./ActorWorkWorkspace";
import { MatterWorkDetail } from "./MatterWorkDetail";
import { NotificationPreferences } from "./NotificationPreferences";
import { ShellIcon } from "./ShellIcon";
import { isAdministrativeHome } from "./homePresentation";
import { programFilterQuery, readProgramFilters } from "./programFilters";
import "./program-register.css";
const privacyReadAuthorized = false;
const reportReadAuthorized = false;


const navigation = [
  ["home", "Home"], ["frameworks", "Frameworks"], ["programs", "Programs"],
  ["forms", "Forms"], ["vendors", "Vendors"], ["risks", "Risk & Losses"],
  ["privacy", "Privacy"], ["documents", "Documents"], ["work", "Work & Approvals"], ["reports", "Reports"],
] as const;
type Page = typeof navigation[number][0] | "settings" | "preferences";
type Session = { kind: "loading" } | { kind: "ready"; context: RuntimeContext } | { kind: "demo"; accounts: DemoAccount[]; error?: string } | { kind: "signed-out" } | { kind: "unavailable" };

function currentPage(route: string): Page {
  const key = route.replace(/^#\/?/, "").split(/[/?]/)[0];
  return navigation.some(([id]) => id === key) || key === "settings" || key === "preferences" ? key as Page : "home";
}

export function FreshApp() {
  const [session, setSession] = useState<Session>({ kind: "loading" });
  const [route, setRoute] = useState(() => window.location.hash);
  const page = currentPage(route);
  const parts = route.replace(/^#\/?/, "").split("?")[0]?.split("/") ?? [];
  const encodedID = parts[1];
  let recordID = "";
  let secondaryID = "";
  let invalidRecord = false;
  try { recordID = encodedID ? decodeURIComponent(encodedID) : ""; secondaryID = parts[page === "vendors" ? 3 : 2] ? decodeURIComponent(parts[page === "vendors" ? 3 : 2]!) : ""; } catch { invalidRecord = true; }
  if (page === "vendors" && parts.length > 2 && (parts[2] !== "assessments" || !secondaryID || parts.length !== 4)) invalidRecord = true;
  if (page === "risks" && (["rcsa", "losses"].includes(recordID) ? parts.length > 3 || (parts.length === 3 && !secondaryID) : parts.length > 2)) invalidRecord = true;
  if (page === "privacy" && parts.length > 2) invalidRecord = true;
  if (page === "reports" && parts.length > 1) invalidRecord = true;
  if (page === "programs" && parts.length > 2 && (parts.length !== 3 || !recordID || recordID === "new" || secondaryID !== "issues")) invalidRecord = true;
  if (page === "work" && parts.length > 1 && (parts.length !== 3 || recordID !== "issues" || !secondaryID)) invalidRecord = true;
  if (page === "forms" && recordID === "responses" && parts.length > 3) invalidRecord = true;
  if (page === "forms" && recordID !== "responses" && parts.length > 2 && (!["assign", "edit"].includes(parts[2] ?? "") || parts.length !== 3 || !recordID)) invalidRecord = true;
  const approvalVersions = new URLSearchParams(route.split("?")[1] ?? "").getAll("approval_version");
  const approvalVersion = approvalVersions.length === 1 && /^[1-9]\d*$/.test(approvalVersions[0] ?? "") ? Number(approvalVersions[0]) : undefined;
  if (page === "forms" && approvalVersions.length && (parts.length !== 2 || !recordID || recordID === "responses" || approvalVersion === undefined || !Number.isSafeInteger(approvalVersion))) invalidRecord = true;
  const [menuOpen, setMenuOpen] = useState(false);
  const menuButton = useRef<HTMLButtonElement>(null);
  const menuPanel = useRef<HTMLElement>(null);
  const closeMenuButton = useRef<HTMLButtonElement>(null);
  const menuOpenRef = useRef(false); menuOpenRef.current = menuOpen;
  const focusDestination = useRef(false);
  const focusMenu = useRef(false);
  function dismissMenu() { focusMenu.current = true; setMenuOpen(false); }

  async function openSession() {
    setSession({ kind: "loading" });
    try {
      const status = await loadSessionStatus();
      if (!status.authenticated && status.demo_login_available) {
        const accounts = await loadDemoAccounts();
        setSession({ kind: "demo", accounts });
        return;
      }
      const context = await loadContext();
      setSession({ kind: "ready", context });
    } catch (error) { setSession({ kind: apiErrorKind(error) === "unauthorized" ? "signed-out" : "unavailable" }); }
  }
  useEffect(() => {
    const routeChanged = () => { if (menuOpenRef.current) focusDestination.current = true; setRoute(window.location.hash); setMenuOpen(false); };
    const endSession = () => { focusDestination.current = false; focusMenu.current = false; setMenuOpen(false); setSession({ kind: "signed-out" }); };
    window.addEventListener("hashchange", routeChanged);
    window.addEventListener(SESSION_ENDED_EVENT, endSession);
    void openSession();
    return () => { window.removeEventListener("hashchange", routeChanged); window.removeEventListener(SESSION_ENDED_EVENT, endSession); };
  }, []);
  useEffect(() => {
    if (!menuOpen) return undefined;
    closeMenuButton.current?.focus();
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const close = (event: KeyboardEvent) => {
      if (event.key === "Escape") { event.preventDefault(); dismissMenu(); return; }
      if (event.key !== "Tab") return;
      const items = Array.from(menuPanel.current?.querySelectorAll<HTMLElement>('button:not([disabled]), a[href]') ?? []);
      const first = items[0], last = items[items.length - 1];
      if (!first || !last) return;
      if (event.shiftKey && (document.activeElement === first || !menuPanel.current?.contains(document.activeElement))) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && (document.activeElement === last || !menuPanel.current?.contains(document.activeElement))) { event.preventDefault(); first.focus(); }
    };
    const resize = () => { if (window.innerWidth > 760) { focusDestination.current = true; setMenuOpen(false); } };
    document.addEventListener("keydown", close);
    window.addEventListener("resize", resize);
    return () => { document.removeEventListener("keydown", close); window.removeEventListener("resize", resize); document.body.style.overflow = previousOverflow; };
  }, [menuOpen]);
  useEffect(() => {
    if (menuOpen) return;
    if (focusDestination.current) document.getElementById("v-main")?.focus();
    else if (focusMenu.current) menuButton.current?.focus();
    focusDestination.current = false; focusMenu.current = false;
  }, [menuOpen, route]);

  if (session.kind === "loading") return <main className="v-login"><p role="status">Loading…</p></main>;
  if (session.kind === "signed-out") return <main className="v-login"><h1>Sign in to ClearSight</h1><a className="v-primary" href="/auth/oidc/login">Sign in</a></main>;
  if (session.kind === "unavailable") return <main className="v-login"><h1>ClearSight unavailable</h1><button className="v-primary" onClick={() => void openSession()}>Try again</button></main>;
  if (session.kind === "demo") return <DemoSignIn accounts={session.accounts} onSignedIn={openSession}/>;

  const { context } = session;
  const name = page === "settings" ? "Settings" : navigation.find(([id]) => id === page)?.[1] ?? "Home";
  return <div className="v-app">
    <a className="v-skip" href="#v-main" inert={menuOpen || undefined} onClick={(event) => { event.preventDefault(); document.getElementById("v-main")?.focus(); }}>Skip to content</a>
    <header className="v-topbar" inert={menuOpen || undefined}>
      <button ref={menuButton} className="v-menu" aria-label="Open navigation" aria-controls="v-navigation" aria-expanded={menuOpen} onClick={() => setMenuOpen(!menuOpen)}><ShellIcon name="menu"/></button>
      <a className="v-brand" href="#/home"><span className="v-logo" aria-hidden="true"><i/><i/><i/></span><span><strong>ClearSight</strong><small>GOVERNANCE · RISK · COMPLIANCE</small></span></a>
      <div className="v-entity"><span>LEGAL ENTITY</span><strong>{context.legal_entity.name}</strong></div>
      <a className="v-account" href="#/preferences" aria-label="Email preferences" aria-describedby="v-account-name" title={`${context.actor.name} · Email preferences`}><span className="v-avatar" aria-hidden="true">{initials(context.actor.name)}</span><span className="v-account-name" id="v-account-name">{context.actor.name}</span></a>
    </header>
    {menuOpen && <div className="v-shade" aria-hidden="true" onClick={dismissMenu}/>}
    <aside ref={menuPanel} id="v-navigation" className={`v-sidebar${menuOpen ? " is-open" : ""}`} role={menuOpen ? "dialog" : undefined} aria-modal={menuOpen ? true : undefined} aria-label={menuOpen ? "Navigation" : undefined}>
      {menuOpen && <button ref={closeMenuButton} className="v-nav-close" aria-label="Close navigation" onClick={dismissMenu}>×</button>}
      <div className="v-bank"><span>{initials(context.tenant.name)}</span><strong>{context.tenant.name}</strong></div>
      <p className="v-nav-label">WORKSPACE</p>
      <nav aria-label="Primary navigation">{navigation.filter(([id]) => (id !== "privacy" || privacyReadAuthorized) && (id !== "reports" || reportReadAuthorized)).map(([id, label]) => { const name = id === "home" && isAdministrativeHome(context) ? "Dashboard" : label; return <a href={`#/${id}`} key={id} aria-label={name} title={name} aria-current={page === id ? "page" : undefined} onClick={() => { if (menuOpen && window.location.hash === `#/${id}`) { focusDestination.current = true; setMenuOpen(false); } }}><span className="v-nav-icon"><ShellIcon name={id}/></span><span className="v-nav-text">{name}</span></a>; })}</nav>
      {context.capabilities?.config_read && <a className="v-settings" href="#/settings" aria-label="Settings" title="Settings" aria-current={page === "settings" ? "page" : undefined} onClick={() => { if (menuOpen && window.location.hash === "#/settings") { focusDestination.current = true; setMenuOpen(false); } }}><span className="v-nav-icon"><ShellIcon name="settings"/></span><span className="v-nav-text">Settings</span></a>}
    </aside>
    <main id="v-main" className="v-main" tabIndex={-1} inert={menuOpen || undefined}>{context.mode === "sample" && <p className="v-sample">Sample data</p>}{invalidRecord ? <section><h1>Record unavailable</h1><p>Check the link and try again.</p></section> : page === "home" ? <Home key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}`} context={context}/> : page === "frameworks" ? <Frameworks/> : page === "risks" ? recordID === "rcsa" ? <RCSAWorkspace id={secondaryID || undefined} canViewLosses={Boolean(context.capabilities?.oversight_read)}/> : recordID === "indicators" ? <KeyIndicators canViewLosses={Boolean(context.capabilities?.oversight_read)}/> : recordID === "losses" ? <LossWorkspace id={secondaryID || undefined}/> : <RiskWorkspace id={recordID || undefined} canViewLosses={Boolean(context.capabilities?.oversight_read)}/> : page === "privacy" ? privacyReadAuthorized ? <PrivacyWorkspace id={recordID || undefined}/> : <section><h1>Privacy unavailable</h1></section> : page === "reports" ? reportReadAuthorized ? <Reports key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}`} context={context}/> : <section><h1>Reports unavailable</h1></section> : page === "work" ? recordID === "issues" ? <MatterWorkDetail key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}:${secondaryID}`} id={secondaryID} context={context}/> : <ActorWorkWorkspace key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}`} context={context}/> : page === "programs" ? recordID === "new" ? <ProgramCreator key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}`} context={context}/> : recordID ? <ProgramDetail key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}:${recordID}:${secondaryID}`} id={recordID} showIssues={secondaryID === "issues"}/> : <ProgramsRegister context={context}/> : page === "forms" ? recordID === "responses" ? <Responses key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}:${secondaryID}`} id={secondaryID || undefined} reviewScope={{tenant: context.tenant.id, entity: context.legal_entity.id, actor: context.actor.id}}/> : recordID === "new" && !secondaryID ? <FormCreator context={context}/> : recordID && secondaryID === "assign" ? <section><h1>Form assignment unavailable</h1></section> : recordID && secondaryID === "edit" ? <FormEditor key={recordID} id={recordID}/> : recordID ? <FormDetail key={`${recordID}:${approvalVersion ?? ""}`} id={recordID} approvalVersion={approvalVersion}/> : <FormsRegister/> : page === "preferences" ? <NotificationPreferences key={`${context.tenant.id}:${context.actor.id}`} scope={{tenant:context.tenant.id,actor:context.actor.id}}/> : page === "documents" ? <Documents/> : page === "vendors" ? <Vendors key={`${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}:${recordID}:${secondaryID}`} context={context} relationshipId={recordID || undefined} assessmentId={secondaryID || undefined}/> : <section><p className="v-eyebrow">{context.legal_entity.name}</p><h1>{name}</h1><p className="v-unavailable">Unavailable.</p></section>}</main>
  </div>;
}

function FormDetail({ id, approvalVersion }: { id: string; approvalVersion?: number }) {
  const [item, setItem] = useState<FormLibraryItem>();
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  const [receipt, setReceipt] = useState<{ id: string; version: number; status: "ACTIVE" | "REJECTED" }>();
  useEffect(() => { let active = true; setItem(undefined); setFailed(false); loadFormTemplateItem(id).then((value) => { if (active) { if (value.template?.id !== id) setFailed(true); else setItem(value); } }).catch(() => { if (active) setFailed(true); }); return () => { active = false; }; }, [id, retry]);
  if (failed) return <section>{receipt && <p role="status">{receipt.status === "ACTIVE" ? "Form approved." : "Form rejected."}</p>}<h1>Form unavailable</h1><button onClick={() => setRetry(retry + 1)}>Try again</button></section>;
  if (!item) return <p role="status">Loading form…</p>;
  const { template } = item;
  const approvalChanged = approvalVersion !== undefined && template.version !== approvalVersion;
  const approvalClosed = approvalVersion !== undefined && !approvalChanged && template.status !== "PENDING_APPROVAL";
  const approvalUnavailable = approvalChanged || approvalClosed;
  const receiptMatches = receipt?.id === template.id && receipt.version === template.version && receipt.status === template.status;
  const canTestQuestions = template.status === "DRAFT" && item.authority_available && item.operations?.some((operation) => operation.command === "forms.template.revise" && operation.can_act);
  return <section><a className="v-back" href="#/forms">← Forms</a><header className="v-page-heading"><div><p className="v-eyebrow">Version {template.version}</p><h1>{template.name}</h1></div><span className="v-status">{template.status.toLowerCase().replaceAll("_", " ")}</span></header>{receiptMatches && <p role="status">{receipt.status === "ACTIVE" ? "Form approved." : "Form rejected."}</p>}{approvalUnavailable && !receiptMatches && <div className="v-panel" role="alert"><p>{approvalChanged ? "Form version changed. Review the latest version before deciding." : "This form is no longer awaiting approval."}</p><a href={`#/forms/${encodeURIComponent(template.id)}`}>Review latest form</a></div>}{!approvalUnavailable && item.authority_available && item.operations?.some((operation) => operation.command === "forms.template.revise" && operation.can_act) && <p><a className="v-primary" href={`#/forms/${encodeURIComponent(template.id)}/edit`}>Edit form</a></p>}{!approvalUnavailable && <FormActions item={item} onSaved={(result) => { if (approvalVersion !== undefined && template.status === "PENDING_APPROVAL" && (result.status === "ACTIVE" || result.status === "REJECTED")) setReceipt({ id: result.id, version: result.version, status: result.status }); }} onReload={() => setRetry((value) => value + 1)}/>}<div className="v-panel"><h2>Questions</h2>{!template.fields ? <p>Questions unavailable.</p> : template.fields.length === 0 ? <p>No questions recorded.</p> : <ul className="v-attention">{template.fields.map((field) => <li key={field.id}><div><strong>{field.label}</strong><small>{field.type.replaceAll("_", " ")}{field.required ? " · Required" : ""}</small>{!approvalUnavailable && canTestQuestions && (field.type === "yes_no" || field.type === "decimal" || field.type === "percentage") && <APIQuestionWizard formID={template.id} version={template.version} fieldID={field.id} label={field.label} resultType={field.type === "yes_no" ? "BOOLEAN" : "NUMBER"}/>}</div></li>)}</ul>}</div></section>;
}

function ProgramDetail({ id, showIssues = false }: { id: string; showIssues?: boolean }) {
  const [value, setValue] = useState<ProgramAggregate>();
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  useEffect(() => { let active = true; setValue(undefined); setFailed(false); loadProgram(id).then((result) => { if (active) { if (result.program?.id !== id) { setFailed(true); return; } setValue(result); } }).catch(() => { if (active) setFailed(true); }); return () => { active = false; }; }, [id, retry]);
  if (failed) return <section><h1>Program unavailable</h1><button onClick={() => setRetry(retry + 1)}>Try again</button></section>;
  if (!value) return <p role="status">Loading program…</p>;
  const filters = readProgramFilters(window.location.hash);
  if (showIssues) return <section className="v-programs"><a className="v-back" href={`#/programs/${encodeURIComponent(id)}${programFilterQuery(filters.search, filters.overallState)}`}>← {value.program.name}</a><header className="v-page-heading"><div><p className="v-eyebrow">{value.program.name}</p><h1>Open issues</h1></div></header><ProgramIssues program={value.program}/></section>;
  return <section className="v-programs"><a className="v-back" href={`#/programs${programFilterQuery(filters.search, filters.overallState)}`}>← Programs</a><header className="v-page-heading"><div><p className="v-eyebrow">{value.program.code}</p><h1>{value.program.name}</h1></div><span className="v-status">{value.program.status.toLowerCase()}</span></header><ProgramCoverage aggregate={value}/><ProgramRequirements aggregate={value} onUpdated={setValue}/></section>;
}

function DemoSignIn({ accounts, onSignedIn }: { accounts: DemoAccount[]; onSignedIn: () => Promise<void> }) {
  const [selected, setSelected] = useState(accounts[0]?.username ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: FormEvent) {
    event.preventDefault();
    const account = accounts.find((item) => item.username === selected);
    if (!account || busy) return;
    setBusy(true); setError("");
    try { await loginDemo(account.username, account.password); await onSignedIn(); }
    catch { setError("Sign-in failed. Try again."); setBusy(false); }
  }
  return <main className="v-login"><form onSubmit={(event) => void submit(event)}><h1>Sign in to ClearSight</h1><label htmlFor="demo-role">Demo account</label><select id="demo-role" value={selected} onChange={(event) => setSelected(event.target.value)}>{accounts.map((account) => <option key={account.username} value={account.username}>{account.label}</option>)}</select>{error && <p role="alert">{error}</p>}<button className="v-primary" disabled={!selected || busy}>Sign in</button></form></main>;
}

function initials(value: string) { return value.trim().split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase(); }
