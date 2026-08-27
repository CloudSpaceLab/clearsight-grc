import { useEffect, useMemo, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  createLibraryFormDraft,
  createLibraryFormRevision,
  deleteSavedFormView,
  instantiateStarterTemplate,
  loadFormTemplatePage,
  loadFormTemplateRevision,
  loadReusableFormTemplateRefs,
  loadSavedFormViews,
  loadStarterTemplates,
  saveFormView,
  transitionFormTemplateRevision,
} from "../formsApi";
import type {
  FormLibraryItem,
  FormTemplate,
  FormTemplateQuery,
  ReusableFormTemplateRef,
  SavedFormView,
  StarterTemplate,
} from "../formsTypes";
import type { CreateFormTemplateInput, FormTemplate as MonitoringFormTemplate, LifecycleStatus } from "../monitoringTypes";
import { FormBuilder } from "./FormBuilder";
import { isTemplateApprovalReady } from "./forms/formQuality";

const tabs = ["Templates", "Sent forms", "Responses", "Imports", "Communications"] as const;
type FormsTab = typeof tabs[number];
type LoadState = "loading" | "live" | "unavailable";
type Layout = "table" | "grid";
type EditorState = { mode: "create" } | { mode: "edit"; template: FormTemplate };

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  targetID?: string;
  initialSearch?: string;
  onTarget?: (id?: string) => void;
};

export function FormsWorkspace({ organizationName = "Organization", legalEntityName = "Legal entity", targetID, initialSearch, onTarget }: Props) {
  const [activeTab, setActiveTab] = useState<FormsTab>("Templates");
  const [query, setQuery] = useState<FormTemplateQuery>(() => readFormsQuery(window.location.hash, initialSearch));
  const [page, setPage] = useState<{ items: FormLibraryItem[]; next_cursor?: string }>({ items: [] });
  const [state, setState] = useState<LoadState>("loading");
  const [layout, setLayout] = useState<Layout>("table");
  const [starters, setStarters] = useState<StarterTemplate[]>([]);
  const [reusableTemplates, setReusableTemplates] = useState<ReusableFormTemplateRef[]>([]);
  const [savedViews, setSavedViews] = useState<SavedFormView[]>([]);
  const [selectedIDs, setSelectedIDs] = useState<Set<string>>(new Set());
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [starterOpen, setStarterOpen] = useState(false);
  const [saveViewOpen, setSaveViewOpen] = useState(false);
  const [savedViewName, setSavedViewName] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const loadEpoch = useRef(0);

  const selected = useMemo(() => page.items.find((item) => item.template.id === targetID), [page.items, targetID]);
  const selectedItems = useMemo(() => page.items.filter((item) => selectedIDs.has(item.template.id)), [page.items, selectedIDs]);
  const bulkTransition = selectedItems.length > 0 && selectedItems.every((item) => item.template.status === "DRAFT" && isTemplateApprovalReady(item.template))
    ? "PENDING_APPROVAL" as const
    : undefined;
  const recent = page.items.slice(0, 3);

  useEffect(() => {
    const sync = () => setQuery(readFormsQuery(window.location.hash, initialSearch));
    window.addEventListener("hashchange", sync);
    window.addEventListener("popstate", sync);
    return () => {
      window.removeEventListener("hashchange", sync);
      window.removeEventListener("popstate", sync);
    };
  }, [initialSearch]);

  useEffect(() => {
    const timer = window.setTimeout(() => { void refresh(query); }, query.search ? 250 : 0);
    return () => window.clearTimeout(timer);
  }, [query.search, query.status, query.owner, query.program, query.use, query.tag, query.limit]);

  useEffect(() => {
    let active = true;
    void Promise.allSettled([loadStarterTemplates(), loadSavedFormViews(), loadReusableFormTemplateRefs()]).then(([starterResult, viewResult, reusableResult]) => {
      if (!active) return;
      if (starterResult.status === "fulfilled") setStarters(starterResult.value);
      if (viewResult.status === "fulfilled") setSavedViews(viewResult.value);
      if (reusableResult.status === "fulfilled") setReusableTemplates(reusableResult.value);
    });
    return () => { active = false; };
  }, []);

  useEffect(() => {
    const visible = new Set(page.items.map((item) => item.template.id));
    setSelectedIDs((current) => new Set([...current].filter((id) => visible.has(id))));
  }, [page.items]);

  async function refresh(nextQuery = query) {
    const epoch = ++loadEpoch.current;
    setState("loading");
    setError(null);
    try {
      const next = await loadFormTemplatePage({ ...nextQuery, cursor: undefined, limit: nextQuery.limit ?? 25 });
      if (epoch !== loadEpoch.current) return;
      setPage(next);
      setState("live");
    } catch (cause) {
      if (epoch !== loadEpoch.current) return;
      setPage({ items: [] });
      setState("unavailable");
      setError(cause instanceof Error ? cause.message : "Form templates could not be loaded.");
    }
  }

  async function refreshReusableTemplates() {
    try {
      setReusableTemplates(await loadReusableFormTemplateRefs());
    } catch {
      // The editor remains usable without reusable section suggestions.
    }
  }

  async function loadMore() {
    if (!page.next_cursor || busy) return;
    setBusy("load-more");
    setError(null);
    try {
      const next = await loadFormTemplatePage({ ...query, cursor: page.next_cursor, limit: query.limit ?? 25 });
      setPage({ items: [...page.items, ...next.items], next_cursor: next.next_cursor });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "More form templates could not be loaded.");
    } finally {
      setBusy(null);
    }
  }

  function changeQuery(patch: Partial<FormTemplateQuery>) {
    const next = { ...query, ...patch, cursor: undefined };
    setQuery(next);
    writeFormsLocation(next, targetID, true);
  }

  function choose(id?: string) {
    onTarget?.(id);
    writeFormsLocation(query, id, Boolean(onTarget));
  }

  function openCreate() {
    setEditor({ mode: "create" });
    setStarterOpen(false);
    setError(null);
  }

  function openEdit(template: FormTemplate) {
    setEditor({ mode: "edit", template });
    setError(null);
  }

  function applySavedView(view: SavedFormView) {
    const next: FormTemplateQuery = {
      search: view.filter.search,
      status: view.filter.status,
      owner: view.filter.owner_principal_id,
      program: view.filter.program_id,
      use: view.filter.use,
      tag: view.filter.tag,
      limit: view.filter.limit ?? query.limit ?? 25,
    };
    setQuery(next);
    writeFormsLocation(next, undefined, true);
    onTarget?.(undefined);
  }

  async function submitSavedView(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = savedViewName.trim();
    if (!name || busy) return;
    setBusy("save-view");
    setError(null);
    try {
      const saved = await saveFormView(name, query);
      setSavedViews((current) => [...current.filter((item) => item.id !== saved.id), saved].sort((a, b) => a.name.localeCompare(b.name)));
      setSavedViewName("");
      setSaveViewOpen(false);
      setNotice(`Saved “${saved.name}” for this Forms library.`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The saved view could not be stored.");
    } finally {
      setBusy(null);
    }
  }

  async function removeSavedView(id: string) {
    if (busy) return;
    setBusy(`delete-view:${id}`);
    setError(null);
    try {
      await deleteSavedFormView(id);
      setSavedViews((current) => current.filter((item) => item.id !== id));
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The saved view could not be removed.");
    } finally {
      setBusy(null);
    }
  }

  async function useStarter(starter: StarterTemplate) {
    if (busy) return;
    setBusy(`starter:${starter.code}`);
    setError(null);
    try {
      const created = await instantiateStarterTemplate(starter.code);
      setStarterOpen(false);
      setEditor({ mode: "edit", template: created });
      choose(created.id);
      setNotice("Starter copied into an ordinary draft. Review its exact fields and quality checks before approval.");
      await refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The starter template could not be copied.");
    } finally {
      setBusy(null);
    }
  }

  async function saveEditorDraft(input: CreateFormTemplateInput) {
    if (!editor || editor.mode === "create") return createLibraryFormDraft(input);
    return createLibraryFormRevision(editor.template.id, editor.template.version, input);
  }

  async function sendEditorForApproval(form: MonitoringFormTemplate) {
    return transitionFormTemplateRevision(form.id, form.version, "PENDING_APPROVAL");
  }

  async function editorSaved(form: MonitoringFormTemplate) {
    setEditor(null);
    choose(form.id);
    setNotice(form.status === "PENDING_APPROVAL" ? "Draft sent for independent approval." : "Form draft saved as a new governed revision.");
    await Promise.all([refresh(), refreshReusableTemplates()]);
  }

  async function transition(item: FormLibraryItem, to: LifecycleStatus) {
    if (busy) return;
    if (to === "PENDING_APPROVAL" && !isTemplateApprovalReady(item.template)) {
      setError("Resolve this draft’s approval-quality checks in the editor before sending it for approval.");
      return;
    }
    setBusy(`transition:${item.template.id}`);
    setError(null);
    try {
      await transitionFormTemplateRevision(item.template.id, item.template.version, to);
      setNotice(to === "PENDING_APPROVAL" ? "Draft sent for independent approval." : to === "ACTIVE" ? "The approved revision is now active." : "The revision state was updated.");
      await Promise.all([refresh(), refreshReusableTemplates()]);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The form state could not be changed.");
    } finally {
      setBusy(null);
    }
  }

  async function runBulkTransition() {
    if (!bulkTransition || busy) return;
    setBusy("bulk-transition");
    setError(null);
    try {
      await Promise.all(selectedItems.map((item) => transitionFormTemplateRevision(item.template.id, item.template.version, bulkTransition)));
      setSelectedIDs(new Set());
      setNotice(`${selectedItems.length} approval-ready drafts were sent for independent approval.`);
      await refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The selected forms could not all be updated. The library has been refreshed.");
      await refresh();
    } finally {
      setBusy(null);
    }
  }

  return <section className="forms-workspace" aria-labelledby="forms-title">
    <header className="forms-header">
      <div><span className="eyebrow">{organizationName} · {legalEntityName}</span><h1 id="forms-title">Forms</h1><p>Build governed reusable templates, then distribute exact approved revisions without duplicating evidence or vendor records.</p></div>
      {activeTab === "Templates" && !editor && <button className="forms-primary" type="button" onClick={openCreate}>Create form template</button>}
    </header>

    <nav className="forms-tabs" aria-label="Forms sections">
      {tabs.map((tab) => <button key={tab} type="button" className={activeTab === tab ? "active" : ""} aria-current={activeTab === tab ? "page" : undefined} onClick={() => { setActiveTab(tab); setEditor(null); }}>{tab}</button>)}
    </nav>

    {error && <div className="forms-message error" role="alert">{error}</div>}
    {notice && <div className="forms-message" role="status">{notice}<button type="button" aria-label="Dismiss Forms notice" onClick={() => setNotice(null)}>×</button></div>}

    {activeTab === "Templates" && editor ? <div className="forms-editor-shell">
      <FormBuilder
        key={editor.mode === "edit" ? `${editor.template.id}:${editor.template.version}` : "new-form"}
        initialValue={editor.mode === "edit" ? editor.template : undefined}
        saveDraft={saveEditorDraft}
        onSendForApproval={sendEditorForApproval}
        onSaved={(form) => { void editorSaved(form); }}
        onCancel={() => setEditor(null)}
        reusableTemplates={reusableTemplates}
        loadReusableTemplate={loadFormTemplateRevision}
      />
    </div> : activeTab === "Templates" ? <div className="forms-template-layout">
      <div className="forms-library">
        <div className="forms-toolbar">
          <label className="forms-search"><span>Search templates</span><input type="search" value={query.search ?? ""} placeholder="Name, code or purpose" onChange={(event) => changeQuery({ search: event.target.value || undefined })}/></label>
          <label><span>Status</span><select value={query.status ?? ""} onChange={(event) => changeQuery({ status: event.target.value ? event.target.value as LifecycleStatus : undefined })}><option value="">All states</option><option value="DRAFT">Draft</option><option value="PENDING_APPROVAL">Pending approval</option><option value="ACTIVE">Active</option><option value="PAUSED">Paused</option><option value="RETIRED">Retired</option><option value="REJECTED">Rejected</option></select></label>
          <div className="forms-layout-toggle" aria-label="Template layout"><button type="button" aria-pressed={layout === "table"} onClick={() => setLayout("table")}>Table</button><button type="button" aria-pressed={layout === "grid"} onClick={() => setLayout("grid")}>Cards</button></div>
          <details className="forms-more-filters"><summary>More filters</summary><div><label><span>Owner ID</span><input value={query.owner ?? ""} onChange={(event) => changeQuery({ owner: event.target.value || undefined })}/></label><label><span>Program ID</span><input value={query.program ?? ""} onChange={(event) => changeQuery({ program: event.target.value || undefined })}/></label><label><span>Approved use</span><input value={query.use ?? ""} onChange={(event) => changeQuery({ use: event.target.value || undefined })}/></label><label><span>Tag</span><input value={query.tag ?? ""} onChange={(event) => changeQuery({ tag: event.target.value || undefined })}/></label></div></details>
        </div>

        <div className="forms-saved-views" aria-label="Saved form views">
          <span>Saved views</span>
          {savedViews.length ? savedViews.map((view) => <span className="forms-saved-view" key={view.id}><button type="button" onClick={() => applySavedView(view)}>{view.name}</button><button type="button" aria-label={`Delete saved view ${view.name}`} disabled={busy === `delete-view:${view.id}`} onClick={() => void removeSavedView(view.id)}>×</button></span>) : <small>None yet</small>}
          <button type="button" className="forms-link-button" onClick={() => setSaveViewOpen((open) => !open)}>Save current view</button>
          {saveViewOpen && <form className="forms-save-view" onSubmit={submitSavedView}><label><span>View name</span><input value={savedViewName} onChange={(event) => setSavedViewName(event.target.value)} maxLength={120}/></label><button type="submit" disabled={!savedViewName.trim() || busy === "save-view"}>Save</button></form>}
        </div>

        <div className="forms-secondary-actions"><button type="button" onClick={() => setStarterOpen((open) => !open)}>Use a starter template</button>{starterOpen && <div className="forms-starters">{starters.length ? starters.map((starter) => <article key={`${starter.code}:${starter.catalog_version}`}><div><strong>{starter.template.name}</strong><span>v{starter.catalog_version} · published {starter.published_on}</span><p>{starter.reference_label}</p></div><button type="button" disabled={busy === `starter:${starter.code}`} onClick={() => void useStarter(starter)}>Create governed draft</button></article>) : <p>No starter templates are currently available.</p>}</div>}</div>

        {recent.length > 0 && <section className="forms-recent" aria-labelledby="forms-recent-title"><div><h2 id="forms-recent-title">Recently updated</h2><span>Latest bounded server results</span></div><ol>{recent.map((item) => <li key={item.template.id}><button type="button" onClick={() => choose(item.template.id)}><strong>{item.template.name}</strong><span>{statusLabel(item.template.status)} · v{item.template.version}</span></button></li>)}</ol></section>}

        {selectedItems.length > 0 && <div className="forms-bulk" role="status"><strong>{selectedItems.length} selected</strong>{bulkTransition ? <button type="button" disabled={busy === "bulk-transition"} onClick={() => void runBulkTransition()}>Send {selectedItems.length} for approval</button> : <span>Bulk approval is available only when every selected row is an approval-ready draft with the same permitted lifecycle action.</span>}<button type="button" onClick={() => setSelectedIDs(new Set())}>Clear selection</button></div>}

        {state === "loading" ? <div className="forms-loading" aria-live="polite" aria-busy="true">Loading form templates…</div> : state === "unavailable" ? <div className="forms-empty"><h2>Forms are temporarily unavailable</h2><p>The library could not be read. No template state was changed.</p><button type="button" onClick={() => void refresh()}>Retry</button></div> : page.items.length === 0 ? <div className="forms-empty"><h2>{query.search ? `No form templates match ‘${query.search}’ in this legal entity.` : "No form templates are available in this legal entity yet."}</h2><p>{query.search ? "Change the search or filters, or start a new governed draft." : "Create a template from scratch or copy the reviewed starter into a draft."}</p><div><button className="forms-primary" type="button" onClick={openCreate}>Create form template</button><button type="button" onClick={() => setStarterOpen(true)}>Use a starter template</button></div></div> : layout === "table" ? <TemplateTable items={page.items} selectedIDs={selectedIDs} targetID={targetID} onToggle={(id) => toggleSelected(id, selectedIDs, setSelectedIDs)} onOpen={choose}/> : <TemplateGrid items={page.items} selectedIDs={selectedIDs} targetID={targetID} onToggle={(id) => toggleSelected(id, selectedIDs, setSelectedIDs)} onOpen={choose}/>}

        {page.next_cursor && state === "live" && <button className="forms-load-more" type="button" disabled={busy === "load-more"} onClick={() => void loadMore()}>{busy === "load-more" ? "Loading…" : "Load more"}</button>}
      </div>

      <aside className="forms-detail" aria-label="Selected form template">
        {selected ? <TemplateDetail item={selected} busy={busy} onEdit={() => openEdit(selected.template)} onTransition={(to) => void transition(selected, to)}/> : targetID ? <><span className="forms-detail-kicker">Selected template</span><h2>Template is outside this bounded page</h2><p>Clear filters or search for the template to inspect its latest and active revision state.</p><button type="button" onClick={() => { changeQuery({ search: undefined, status: undefined, owner: undefined, program: undefined, use: undefined, tag: undefined }); choose(undefined); }}>Clear filters</button></> : <><span className="forms-detail-kicker">Template detail</span><h2>Select a template</h2><p>Inspect the latest stored revision separately from the active revision available for reuse.</p></>}
      </aside>
    </div> : <FutureFormsTab tab={activeTab as Exclude<FormsTab, "Templates">}/>} 
  </section>;
}

function TemplateTable({ items, selectedIDs, targetID, onToggle, onOpen }: { items: FormLibraryItem[]; selectedIDs: Set<string>; targetID?: string; onToggle: (id: string) => void; onOpen: (id?: string) => void }) {
  return <div className="forms-table-wrap"><table className="forms-table"><thead><tr><th><span className="sr-only">Select</span></th><th>Template</th><th>Latest revision</th><th>Reusable revision</th><th>Owner</th><th>Updated</th><th><span className="sr-only">Open</span></th></tr></thead><tbody>{items.map((item) => <tr key={item.template.id} className={targetID === item.template.id ? "selected" : ""}><td><input type="checkbox" aria-label={`Select ${item.template.name}`} checked={selectedIDs.has(item.template.id)} onChange={() => onToggle(item.template.id)}/></td><td><strong>{item.template.name}</strong><span>{item.template.code}</span></td><td><StatusPill status={item.template.status}/><span>v{item.template.version}</span></td><td>{item.active_version ? <><StatusPill status={item.active_status ?? "ACTIVE"}/><span>v{item.active_version}</span></> : <span className="forms-muted">Not active</span>}</td><td>{item.template.responsible_team || item.template.owner_principal_id || "Not assigned"}</td><td>{formatDate(item.template.updated_at)}</td><td><button type="button" aria-label={`Open ${item.template.name}`} onClick={() => onOpen(item.template.id)}>Open</button></td></tr>)}</tbody></table></div>;
}

function TemplateGrid({ items, selectedIDs, targetID, onToggle, onOpen }: { items: FormLibraryItem[]; selectedIDs: Set<string>; targetID?: string; onToggle: (id: string) => void; onOpen: (id?: string) => void }) {
  return <div className="forms-grid">{items.map((item) => <article key={item.template.id} className={targetID === item.template.id ? "selected" : ""}><div className="forms-card-top"><input type="checkbox" aria-label={`Select ${item.template.name}`} checked={selectedIDs.has(item.template.id)} onChange={() => onToggle(item.template.id)}/><StatusPill status={item.template.status}/></div><button className="forms-card-open" type="button" onClick={() => onOpen(item.template.id)}><strong>{item.template.name}</strong><span>{item.template.code} · latest v{item.template.version}</span><p>{item.template.purpose}</p></button><div className="forms-card-meta"><span>{item.active_version ? `Reusable v${item.active_version}` : "No active revision"}</span><span>{formatDate(item.template.updated_at)}</span></div></article>)}</div>;
}

function TemplateDetail({ item, busy, onEdit, onTransition }: { item: FormLibraryItem; busy: string | null; onEdit: () => void; onTransition: (to: LifecycleStatus) => void }) {
  const form = item.template;
  const approvalReady = form.status === "DRAFT" && isTemplateApprovalReady(form);
  return <><span className="forms-detail-kicker">{form.code}</span><h2>{form.name}</h2><p>{form.purpose}</p><div className="forms-detail-state"><div><span>Latest stored</span><strong><StatusPill status={form.status}/> v{form.version}</strong></div><div><span>Reusable now</span><strong>{item.active_version ? <><StatusPill status={item.active_status ?? "ACTIVE"}/> v{item.active_version}</> : "None"}</strong></div></div><dl><div><dt>Scoring</dt><dd>{form.scoring_mode || "NONE"}</dd></div><div><dt>Fields</dt><dd>{form.fields.length}</dd></div><div><dt>Owner</dt><dd>{form.owner_principal_id || form.responsible_team || "Not assigned"}</dd></div><div><dt>Next review</dt><dd>{form.next_review_at ? formatDate(form.next_review_at) : "Not scheduled"}</dd></div></dl>{form.tags?.length ? <div className="forms-tags">{form.tags.map((tag) => <span key={tag}>{tag}</span>)}</div> : null}<div className="forms-detail-actions">{form.status === "DRAFT" && <><button type="button" onClick={onEdit}>Edit draft</button><button className="forms-primary" type="button" disabled={busy !== null || !approvalReady} onClick={() => onTransition("PENDING_APPROVAL")}>Send for approval</button></>}{form.status === "DRAFT" && !approvalReady && <small className="forms-muted">Open the editor to resolve approval-quality checks before submission.</small>}{form.status === "PENDING_APPROVAL" && <><button className="forms-primary" type="button" disabled={busy !== null} onClick={() => onTransition("ACTIVE")}>Approve and activate</button><button type="button" disabled={busy !== null} onClick={() => onTransition("REJECTED")}>Reject</button></>}{form.status === "ACTIVE" && <button type="button" disabled={busy !== null} onClick={() => onTransition("RETIRED")}>Retire revision</button>}</div></>;
}

function FutureFormsTab({ tab }: { tab: Exclude<FormsTab, "Templates"> }) {
  if (tab === "Imports") return <div className="forms-future"><span className="forms-detail-kicker">Imports</span><h2>Turn governed source documents into reviewable form proposals</h2><p>The current Imports workspace remains the authoritative document intake path. Form-template proposal review is enabled in Tranche 3; no extracted content is silently converted into an active form.</p><button type="button" onClick={() => { window.location.hash = "#imports"; }}>Open Imports</button></div>;
  const messages: Record<Exclude<FormsTab, "Templates" | "Imports">, [string, string]> = {
    "Sent forms": ["No form distributions yet", "Distribution records, recipients and protected delivery become available when the Tranche 2 distribution APIs are deployed."],
    Responses: ["No distribution responses yet", "Responses will appear here from immutable submission revisions; legacy evidence requests remain in Work until that migration is proven."],
    Communications: ["Communication configuration is not enabled yet", "Invitation, reminder and branding configuration arrives with governed Tranche 2 communications and maker-checker activation."],
  };
  const [title, detail] = messages[tab];
  return <div className="forms-future"><span className="forms-detail-kicker">{tab}</span><h2>{title}</h2><p>{detail}</p></div>;
}

function StatusPill({ status }: { status: LifecycleStatus }) {
  return <span className={`forms-status status-${status.toLowerCase().replaceAll("_", "-")}`}>{statusLabel(status)}</span>;
}

function statusLabel(status: LifecycleStatus) {
  return status.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function formatDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Unknown";
  return new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" }).format(date);
}

function toggleSelected(id: string, selected: Set<string>, setSelected: (value: Set<string>) => void) {
  const next = new Set(selected);
  if (next.has(id)) next.delete(id); else next.add(id);
  setSelected(next);
}

function readFormsQuery(hash: string, fallbackSearch?: string): FormTemplateQuery {
  const raw = hash.includes("?") ? hash.slice(hash.indexOf("?") + 1) : "";
  const params = new URLSearchParams(raw);
  const limitValue = Number(params.get("limit") || "25");
  return {
    search: params.get("search") || fallbackSearch || undefined,
    status: (params.get("status") || undefined) as LifecycleStatus | undefined,
    owner: params.get("owner") || undefined,
    program: params.get("program") || undefined,
    use: params.get("use") || undefined,
    tag: params.get("tag") || undefined,
    limit: Number.isFinite(limitValue) && limitValue >= 1 && limitValue <= 100 ? limitValue : 25,
  };
}

function writeFormsLocation(query: FormTemplateQuery, targetID?: string, replace = true) {
  const params = new URLSearchParams();
  if (query.search?.trim()) params.set("search", query.search.trim());
  if (query.status) params.set("status", query.status);
  if (query.owner?.trim()) params.set("owner", query.owner.trim());
  if (query.program?.trim()) params.set("program", query.program.trim());
  if (query.use?.trim()) params.set("use", query.use.trim());
  if (query.tag?.trim()) params.set("tag", query.tag.trim());
  if (query.limit && query.limit !== 25) params.set("limit", String(query.limit));
  const encoded = params.toString();
  const hash = `#forms${targetID ? `/${encodeURIComponent(targetID)}` : ""}${encoded ? `?${encoded}` : ""}`;
  if (replace) window.history.replaceState(null, "", hash); else window.history.pushState(null, "", hash);
}
