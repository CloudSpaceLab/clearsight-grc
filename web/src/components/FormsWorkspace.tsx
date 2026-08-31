import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties, FormEvent } from "react";
import "../forms.css";
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
  FormTemplatePage,
  FormTemplateProposal,
  FormTemplateQuery,
  ReusableFormTemplateRef,
  SavedFormView,
  StarterTemplate,
} from "../formsTypes";
import type { CreateFormTemplateInput, FormTemplate as MonitoringFormTemplate, LifecycleStatus } from "../monitoringTypes";
import { FormBuilder } from "./FormBuilder";
import { FormAIComposer } from "./forms/FormAIComposer";
import { FormProposalReview } from "./forms/FormProposalReview";
import { FormsBrandHeader } from "./forms/FormsBrandHeader";
import { FormsEmptyState } from "./forms/FormsEmptyState";
import { FormsTabContent } from "./forms/FormsTabContent";
import { NewFormLauncher } from "./forms/creation/NewFormLauncher";
import { TemplateDetailDrawer } from "./forms/dashboard/TemplateDetailDrawer";
import { TemplateLibraryTable } from "./forms/dashboard/TemplateLibraryTable";
import { FilterBar } from "./forms/filters/FilterBar";
import { FormStatusScopes } from "./forms/filters/FormStatusScopes";
import { defaultFormsAccent, loadFormsAppearance, type FormsAppearance } from "./forms/formsAppearance";
import { preserveLibraryRevisionMetadata } from "./forms/formRevisionInput";
import { isTemplateApprovalReady } from "./forms/formQuality";
import { clearedFormsQuery, readFormsQuery, writeFormsLocation } from "./forms/formsLocation";

const tabs = ["Templates", "Sent forms", "Responses", "Imports", "Communications"] as const;
const libraryRevalidationIntervalMs = 30_000;
type FormsTab = typeof tabs[number];
type LoadState = "loading" | "live" | "unavailable";
type EditorState = { mode: "create" } | { mode: "edit"; template: FormTemplate; canSendForApproval: boolean };

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  appearanceScope?: string;
  targetID?: string;
  initialSearch?: string;
  onTarget?: (id?: string) => void;
};

export function FormsWorkspace({ organizationName = "Organization", legalEntityName = "Legal entity", appearanceScope, targetID, initialSearch, onTarget }: Props) {
  const appearanceKey = appearanceScope?.trim() || legalEntityName;
  const [activeTab, setActiveTab] = useState<FormsTab>("Templates");
  const [query, setQuery] = useState<FormTemplateQuery>(() => readFormsQuery(window.location.hash, initialSearch));
  const [page, setPage] = useState<FormTemplatePage>({ items: [] });
  const [state, setState] = useState<LoadState>("loading");
  const [revalidating, setRevalidating] = useState(false);
  const [starters, setStarters] = useState<StarterTemplate[]>([]);
  const [reusableTemplates, setReusableTemplates] = useState<ReusableFormTemplateRef[]>([]);
  const [savedViews, setSavedViews] = useState<SavedFormView[]>([]);
  const [selectedIDs, setSelectedIDs] = useState<Set<string>>(new Set());
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [newFormOpen, setNewFormOpen] = useState(false);
  const [aiOpen, setAIOpen] = useState(false);
  const [aiProposal, setAIProposal] = useState<FormTemplateProposal | null>(null);
  const [saveViewOpen, setSaveViewOpen] = useState(false);
  const [savedViewName, setSavedViewName] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [appearance, setAppearance] = useState<FormsAppearance>(() => readAppearance(appearanceKey));
  const loadEpoch = useRef(0);
  const requestAbort = useRef<AbortController | null>(null);
  const lastSuccessfulRefresh = useRef(0);

  const selected = useMemo(() => page.items.find((item) => item.template.id === targetID), [page.items, targetID]);
  const selectedItems = useMemo(() => page.items.filter((item) => selectedIDs.has(item.template.id)), [page.items, selectedIDs]);
  const bulkTransition = selectedItems.length > 0 && selectedItems.every((item) => item.template.status === "DRAFT"
    && isTemplateApprovalReady(item.template)
    && item.authority_available === true
    && item.operations?.some((operation) => operation.command === "forms.template.transition" && operation.can_act && operation.allowed_targets?.includes("PENDING_APPROVAL")))
    ? "PENDING_APPROVAL" as const
    : undefined;
  const customView = hasCustomView(query);
  const autoRevalidationEnabled = activeTab === "Templates"
    && state === "live"
    && !editor
    && !newFormOpen
    && !aiOpen
    && !aiProposal
    && !busy
    && page.items.length <= (query.limit ?? 25);

  useEffect(() => setAppearance(readAppearance(appearanceKey)), [appearanceKey]);

  useEffect(() => {
    const sync = () => {
      invalidatePagedLoad();
      setQuery(readFormsQuery(window.location.hash, initialSearch));
    };
    window.addEventListener("hashchange", sync);
    window.addEventListener("popstate", sync);
    return () => {
      window.removeEventListener("hashchange", sync);
      window.removeEventListener("popstate", sync);
    };
  }, [initialSearch]);

  useEffect(() => {
    const timer = window.setTimeout(() => { void refresh(query); }, query.search ? 220 : 0);
    return () => window.clearTimeout(timer);
  }, [query.search, query.status, query.owner, query.program, query.use, query.tag, query.filter, query.sort, query.limit]);

  useEffect(() => {
    if (!autoRevalidationEnabled) return;
    const revalidateIfStale = () => {
      if (document.visibilityState !== "visible" || requestAbort.current) return;
      if (Date.now() - lastSuccessfulRefresh.current < libraryRevalidationIntervalMs) return;
      void refresh(query);
    };
    const interval = window.setInterval(revalidateIfStale, libraryRevalidationIntervalMs);
    window.addEventListener("focus", revalidateIfStale);
    document.addEventListener("visibilitychange", revalidateIfStale);
    return () => {
      window.clearInterval(interval);
      window.removeEventListener("focus", revalidateIfStale);
      document.removeEventListener("visibilitychange", revalidateIfStale);
    };
  }, [autoRevalidationEnabled, query.search, query.status, query.owner, query.program, query.use, query.tag, query.filter, query.sort, query.limit]);

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

  useEffect(() => () => requestAbort.current?.abort(), []);

  function invalidatePagedLoad() {
    loadEpoch.current += 1;
    requestAbort.current?.abort();
    requestAbort.current = null;
    setRevalidating(false);
    setBusy((current) => current === "load-more" ? null : current);
  }

  async function refresh(nextQuery = query) {
    const epoch = ++loadEpoch.current;
    requestAbort.current?.abort();
    const controller = new AbortController();
    requestAbort.current = controller;
    const preserveCurrentRows = state === "live";
    if (preserveCurrentRows) setRevalidating(true); else setState("loading");
    setError(null);
    try {
      const next = await loadFormTemplatePage(
        { ...nextQuery, cursor: undefined, limit: nextQuery.limit ?? 25 },
        controller.signal,
        { statusFacets: true },
      );
      if (controller.signal.aborted || epoch !== loadEpoch.current) return;
      setPage(next);
      lastSuccessfulRefresh.current = Date.now();
      setState("live");
    } catch (cause) {
      if (controller.signal.aborted || epoch !== loadEpoch.current) return;
      if (preserveCurrentRows) {
        setState("live");
      } else {
        setPage({ items: [] });
        setState("unavailable");
      }
      setError(cause instanceof Error ? cause.message : "Form templates could not be loaded.");
    } finally {
      if (requestAbort.current === controller) {
        requestAbort.current = null;
        setRevalidating(false);
      }
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
    const epoch = loadEpoch.current;
    const cursor = page.next_cursor;
    setBusy("load-more");
    setError(null);
    try {
      const next = await loadFormTemplatePage({ ...query, cursor, limit: query.limit ?? 25 });
      if (epoch !== loadEpoch.current) return;
      setPage((current) => ({ ...current, items: [...current.items, ...next.items], next_cursor: next.next_cursor }));
    } catch (cause) {
      if (epoch !== loadEpoch.current) return;
      setError(cause instanceof Error ? cause.message : "More form templates could not be loaded.");
    } finally {
      if (epoch === loadEpoch.current) setBusy(null);
    }
  }

  function replaceQuery(next: FormTemplateQuery) {
    invalidatePagedLoad();
    setQuery({ ...next, cursor: undefined });
    writeFormsLocation(next, targetID, true);
  }

  function choose(id?: string) {
    onTarget?.(id);
    writeFormsLocation(query, id, Boolean(onTarget));
  }

  function clearFiltersAndTarget() {
    const next = clearedFormsQuery(query);
    invalidatePagedLoad();
    setQuery(next);
    writeFormsLocation(next, targetID, true);
  }

  function openCreate() {
    setNewFormOpen(false);
    setEditor({ mode: "create" });
    setAIOpen(false);
    setAIProposal(null);
    setError(null);
  }

  function openAI() {
    setNewFormOpen(false);
    setEditor(null);
    setAIProposal(null);
    setAIOpen(true);
    setError(null);
  }

  function openImport() {
    setNewFormOpen(false);
    window.location.hash = "#imports";
  }

  function openEdit(item: FormLibraryItem) {
    const canSendForApproval = item.authority_available === true && Boolean(item.operations?.some((operation) =>
      operation.command === "forms.template.transition" && operation.can_act && operation.allowed_targets?.includes("PENDING_APPROVAL"),
    ));
    setEditor({ mode: "edit", template: item.template, canSendForApproval });
    setAIOpen(false);
    setAIProposal(null);
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
      filter: view.filter.expression,
      sort: view.filter.sort ?? "UPDATED_DESC",
      limit: view.filter.limit ?? query.limit ?? 25,
    };
    invalidatePagedLoad();
    setQuery(next);
    writeFormsLocation(next, targetID, true);
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
      setNewFormOpen(false);
      setEditor({ mode: "edit", template: created, canSendForApproval: false });
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
    return createLibraryFormRevision(
      editor.template.id,
      editor.template.version,
      preserveLibraryRevisionMetadata(editor.template, input),
    );
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

  const workspaceStyle = { "--forms-accent": appearance.accentColor } as CSSProperties;
  return <section className="forms-workspace" aria-labelledby="forms-title" style={workspaceStyle}>
    <FormsBrandHeader
      organizationName={organizationName}
      legalEntityName={legalEntityName}
      appearance={appearance}
      action={activeTab === "Templates" && !editor && !aiOpen && !aiProposal
        ? <button className="forms-primary" type="button" onClick={() => setNewFormOpen(true)}>+ New form</button>
        : undefined}
    />

    <nav className="forms-tabs" aria-label="Forms sections">
      {tabs.map((tab) => <button key={tab} type="button" className={activeTab === tab ? "active" : ""} aria-current={activeTab === tab ? "page" : undefined} onClick={() => { setActiveTab(tab); setEditor(null); setNewFormOpen(false); setAIOpen(false); setAIProposal(null); }}>{tab}</button>)}
    </nav>

    {error && <div className="forms-message error" role="alert">{error}</div>}
    {notice && <div className="forms-message" role="status">{notice}<button type="button" aria-label="Dismiss Forms notice" onClick={() => setNotice(null)}>×</button></div>}

    {newFormOpen && <NewFormLauncher
      starters={starters}
      busy={busy}
      onBlank={openCreate}
      onAI={openAI}
      onImport={openImport}
      onUseStarter={(starter) => { void useStarter(starter); }}
      onClose={() => setNewFormOpen(false)}
    />}

    {activeTab === "Templates" && aiProposal ? <div className="forms-proposal-shell">
      <FormProposalReview proposal={aiProposal} onProposalChange={setAIProposal} onDraftCreated={(id) => { choose(id); void refresh(); }}/>
      <div className="forms-authoring-recovery"><button className="secondary-button" type="button" onClick={() => { setAIProposal(null); setAIOpen(true); }}>Revise objective</button><button type="button" className="text-button" onClick={() => { setAIProposal(null); setAIOpen(false); }}>Return to form library</button></div>
    </div> : activeTab === "Templates" && aiOpen ? <div className="forms-proposal-shell">
      <FormAIComposer baseTemplate={selected ? { id: selected.template.id, name: selected.template.name, version: selected.template.version } : undefined} onProposal={(proposal) => { setAIProposal(proposal); setAIOpen(false); }}/>
      <div className="forms-authoring-recovery"><button className="secondary-button" type="button" onClick={openCreate}>Open manual builder</button><button type="button" className="text-button" onClick={() => setAIOpen(false)}>Return to form library</button></div>
    </div> : activeTab === "Templates" && editor ? <div className="forms-editor-shell">
      <FormBuilder
        key={editor.mode === "edit" ? `${editor.template.id}:${editor.template.version}` : "new-form"}
        initialValue={editor.mode === "edit" ? editor.template : undefined}
        saveDraft={saveEditorDraft}
        onSendForApproval={editor.mode === "create" || editor.canSendForApproval ? sendEditorForApproval : undefined}
        onSaved={(form) => { void editorSaved(form); }}
        onCancel={() => setEditor(null)}
        reusableTemplates={reusableTemplates}
        loadReusableTemplate={loadFormTemplateRevision}
        allowIncompleteComplianceDraft
      />
    </div> : activeTab === "Templates" ? <div className="forms-template-layout forms-template-dashboard">
      <div className="forms-library">
        <FilterBar
          query={query}
          onChange={replaceQuery}
          resultCount={state === "live" ? page.total : undefined}
          revalidating={revalidating}
        />
        {state === "live" && <FormStatusScopes query={query} facets={page.facets} onChange={replaceQuery}/>} 

        {(savedViews.length > 0 || customView) && <div className="forms-saved-views" aria-label="Saved form views">
          {savedViews.length > 0 && <><span>Views</span>{savedViews.map((view) => <span className="forms-saved-view" key={view.id}><button type="button" onClick={() => applySavedView(view)}>{view.name}</button><button type="button" aria-label={`Delete saved view ${view.name}`} disabled={busy === `delete-view:${view.id}`} onClick={() => void removeSavedView(view.id)}>×</button></span>)}</>}
          {customView && <button type="button" className="forms-link-button" onClick={() => setSaveViewOpen((open) => !open)}>Save view</button>}
          {saveViewOpen && <form className="forms-save-view" onSubmit={submitSavedView}><label><span>View name</span><input value={savedViewName} onChange={(event) => setSavedViewName(event.target.value)} maxLength={120}/></label><button type="submit" disabled={!savedViewName.trim() || busy === "save-view"}>Save</button></form>}
        </div>}

        {selectedItems.length > 0 && <div className="forms-bulk" role="status"><strong>{selectedItems.length} selected</strong>{bulkTransition ? <button type="button" disabled={busy === "bulk-transition"} onClick={() => void runBulkTransition()}>Send {selectedItems.length} for approval</button> : <span>Bulk approval is available only when every selected row is an approval-ready draft with the same permitted lifecycle action.</span>}<button type="button" onClick={() => setSelectedIDs(new Set())}>Clear selection</button></div>}

        {state === "loading" ? <div className="forms-loading" aria-live="polite" aria-busy="true">Loading form templates…</div>
          : state === "unavailable" ? <FormsEmptyState tone="unavailable" eyebrow="Connection" title="Forms are temporarily unavailable" detail="The library could not be read, and no template state was changed." actions={<button type="button" onClick={() => void refresh()}>Retry</button>}/>
          : page.items.length === 0 ? <FormsEmptyState
            tone={query.search ? "search" : "empty"}
            eyebrow={query.search ? "No matches" : "Start here"}
            title={query.search ? `No templates match “${query.search}”` : "Create your governed form library"}
            detail={query.search ? "Adjust the search or filters, or create a new form without losing your current view." : "Start with a blank form, a proven template, an AI proposal, or an existing source."}
            actions={<><button className="forms-primary" type="button" onClick={() => setNewFormOpen(true)}>+ New form</button>{customView && <button className="text-button" type="button" onClick={clearFiltersAndTarget}>Clear filters</button>}</>}
          />
          : <TemplateLibraryTable items={page.items} selectedIDs={selectedIDs} targetID={targetID} onToggle={(id) => toggleSelected(id, selectedIDs, setSelectedIDs)} onOpen={choose}/>} 

        {page.next_cursor && state === "live" && <button className="forms-load-more" type="button" disabled={busy === "load-more"} onClick={() => void loadMore()}>{busy === "load-more" ? "Loading…" : "Load more"}</button>}
      </div>

      <TemplateDetailDrawer
        item={selected}
        requestedID={targetID}
        busy={busy}
        onClose={() => choose(undefined)}
        onClearFilters={clearFiltersAndTarget}
        onEdit={() => { if (selected) openEdit(selected); }}
        onTransition={(to) => { if (selected) void transition(selected, to); }}
      />
    </div> : <FormsTabContent tab={activeTab as Exclude<FormsTab, "Templates">}/>} 
  </section>;
}

function hasCustomView(query: FormTemplateQuery) {
  return Boolean(query.search || query.status || query.owner || query.program || query.use || query.tag || query.filter || query.sort === "UPDATED_ASC");
}

function toggleSelected(id: string, selected: Set<string>, setSelected: (value: Set<string>) => void) {
  const next = new Set(selected);
  if (next.has(id)) next.delete(id); else next.add(id);
  setSelected(next);
}

function readAppearance(scope: string): FormsAppearance {
  try {
    return loadFormsAppearance(window.localStorage, scope, window.location.href);
  } catch {
    return { accentColor: defaultFormsAccent };
  }
}
