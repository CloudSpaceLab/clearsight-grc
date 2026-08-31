import { useEffect, useState } from "react";
import { loadDistribution, loadDistributionPage, transitionDistribution, type Distribution, type DistributionDetail, type DistributionQuery } from "../../formsDistributionApi";
import { ApiError } from "../../http";
import { ActionLink, Button, EmptyState, Notice, Surface } from "../ui";
import { DistributionChangePanel } from "./DistributionChangePanel";
import { DistributionComposer } from "./DistributionComposer";
import { SentFormDetail } from "./sent/SentFormDetail";
import { SentFormsFilters } from "./sent/SentFormsFilters";
import { SentFormsTable } from "./sent/SentFormsTable";
import { distributionStatusLabel } from "./sent/distributionPresentation";

type ListState = "loading" | "live" | "sign-in-required" | "error";
type DetailState = "idle" | "loading" | "live" | "error";

export function SentFormsView() {
  const [query, setQuery] = useState<DistributionQuery>(() => readQuery());
  const [listState, setListState] = useState<ListState>("loading");
  const [items, setItems] = useState<Distribution[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [selectedID, setSelectedID] = useState<string>();
  const [detailState, setDetailState] = useState<DetailState>("idle");
  const [detail, setDetail] = useState<DistributionDetail>();
  const [composerOpen, setComposerOpen] = useState(false);
  const [changeMode, setChangeMode] = useState<"amend" | "supersede">();
  const [busy, setBusy] = useState<string>();
  const [error, setError] = useState<string>();
  const [detailError, setDetailError] = useState<string>();
  const [notice, setNotice] = useState<string>();

  useEffect(() => { void refresh(); }, [query.status, query.due_state, query.subject_type, query.subject_id, query.owner, query.limit]);
  useEffect(() => {
    if (selectedID) void loadSelectedDetail(selectedID);
    else { setDetail(undefined); setDetailState("idle"); setDetailError(undefined); }
  }, [selectedID]);

  async function refresh() {
    setListState("loading");
    setError(undefined);
    try {
      const page = await loadDistributionPage({ ...query, cursor: undefined, limit: query.limit ?? 25 });
      setItems(page.items);
      setNextCursor(page.next_cursor);
      setSelectedID((current) => current && page.items.some((value) => value.id === current) ? current : undefined);
      setListState("live");
    } catch (cause) {
      setError(message(cause, "Sent forms could not be loaded for the current filters."));
      setListState(cause instanceof ApiError && cause.status === 401 ? "sign-in-required" : "error");
    }
  }

  async function loadSelectedDetail(id: string) {
    setDetailState("loading");
    setDetailError(undefined);
    try {
      setDetail(await loadDistribution(id));
      setDetailState("live");
    } catch (cause) {
      setDetail(undefined);
      setDetailError(message(cause, "Distribution details could not be loaded."));
      setDetailState("error");
    }
  }

  async function loadMore() {
    if (!nextCursor || busy === "more") return;
    setBusy("more");
    try {
      const page = await loadDistributionPage({ ...query, cursor: nextCursor, limit: query.limit ?? 25 });
      setItems((current) => [...current, ...page.items]);
      setNextCursor(page.next_cursor);
    } catch (cause) {
      setError(message(cause, "More sent forms could not be loaded."));
    } finally {
      setBusy(undefined);
    }
  }

  function updateQuery(patch: Partial<DistributionQuery>) {
    const next = { ...query, ...patch, cursor: undefined };
    setQuery(next);
    writeQuery(next);
  }

  function clearFilters() {
    updateQuery({ status: undefined, due_state: undefined, subject_type: undefined, subject_id: undefined, owner: undefined });
  }

  async function lifecycle(action: "lock" | "reopen" | "revoke") {
    if (!detail || busy) return;
    setBusy(action);
    setDetailError(undefined);
    try {
      const updated = await transitionDistribution(detail.distribution.id, detail.distribution.version, action);
      setDetail(updated);
      setDetailState("live");
      setItems((current) => current.map((value) => value.id === updated.distribution.id ? updated.distribution : value));
      setNotice(`${distributionStatusLabel[updated.distribution.status]}. The sent-form change was confirmed.`);
    } catch (cause) {
      setDetailError(message(cause, "The sent-form state could not be changed."));
    } finally {
      setBusy(undefined);
    }
  }

  if (composerOpen) return <DistributionComposer onCancel={() => setComposerOpen(false)} onCreated={(value) => {
    setComposerOpen(false); setItems((current) => [value.distribution, ...current]); setSelectedID(value.distribution.id); setDetail(value); setDetailState("live");
  }}/>;
  if (changeMode && detail) return <DistributionChangePanel mode={changeMode} detail={detail} onCancel={() => setChangeMode(undefined)} onSaved={(value, resultNotice) => {
    setChangeMode(undefined); setItems((current) => [value.distribution, ...current.filter((item) => item.id !== detail.distribution.id && item.id !== value.distribution.id)]); setSelectedID(value.distribution.id); setDetail(value); setDetailState("live"); setNotice(resultNotice);
  }}/>;

  return <section className="forms-sent" aria-labelledby="sent-forms-title">
    <header className="forms-sent__heading"><div><p>Sender workspace</p><h2 id="sent-forms-title">Sent forms</h2><p>Track each sent form, its recipients, response status, access method and deadline.</p></div><Button variant="primary" onPress={() => setComposerOpen(true)}>Send form</Button></header>
    {notice && <Notice tone="success">{notice}</Notice>}
    <SentFormsFilters query={query} resultCount={listState === "live" ? items.length : undefined} onChange={updateQuery} onClear={clearFilters}/>
    <div className="forms-sent__results" aria-live="polite">
      {listState === "loading" && <Surface><p role="status" aria-label="Loading sent forms matching the current filters">Loading sent forms matching the current filters…</p></Surface>}
      {listState === "sign-in-required" && <EmptyState population="Sent forms matching the current filters" title="Sign in to review sent forms" description="Your session ended before this sent-form list could be loaded." action={<ActionLink href="/">Sign in again</ActionLink>}/>}
      {listState === "error" && <EmptyState population="Sent forms matching the current filters" title="Sent forms could not be loaded" description={error ?? "The current sent-form query could not be completed."} action={<Button onPress={() => void refresh()}>Try again</Button>}/>}
      {listState === "live" && items.length === 0 && <EmptyState population="Sent forms matching the current filters" title="No sent forms match these filters" description="Change the filters or send a form to an exact recipient and subject." action={<Button variant="primary" onPress={() => setComposerOpen(true)}>Send form</Button>}/>}
      {listState === "live" && items.length > 0 && <div className="forms-sent__layout">
        <SentFormsTable items={items} selectedID={selectedID} nextCursor={nextCursor} loadingMore={busy === "more"} onSelect={setSelectedID} onLoadMore={() => void loadMore()}/>
        <aside className="forms-sent__detail" aria-label="Selected distribution">
          {!selectedID && <><p className="forms-sent-detail__type">Distribution detail</p><h3>Select a sent form</h3><p>Open one sent form to review recipient progress, deadline and access method.</p></>}
          {selectedID && detailState === "loading" && <p role="status" aria-label="Loading the selected sent form">Loading the selected sent form…</p>}
          {selectedID && detailState === "error" && <Notice tone="error">{detailError} Select the sent form again to retry.</Notice>}
          {selectedID && detailState === "live" && detail && <SentFormDetail detail={detail} error={detailError} busy={busy} onLifecycle={(action) => void lifecycle(action)} onAmend={() => setChangeMode("amend")} onSupersede={() => setChangeMode("supersede")}/>}
        </aside>
      </div>}
    </div>
  </section>;
}

function readQuery(): DistributionQuery {
  const params = new URLSearchParams(window.location.search);
  const status = params.get("dist_status") as DistributionQuery["status"] | null;
  const due = params.get("dist_due") as DistributionQuery["due_state"] | null;
  return { status: status || undefined, due_state: due || undefined, subject_type: params.get("dist_subject_type") || undefined, subject_id: params.get("dist_subject_id") || undefined, owner: params.get("dist_owner") || undefined, limit: 25 };
}

function writeQuery(query: DistributionQuery) {
  const url = new URL(window.location.href);
  const set = (key: string, value?: string) => value ? url.searchParams.set(key, value) : url.searchParams.delete(key);
  set("dist_status", query.status); set("dist_due", query.due_state); set("dist_subject_type", query.subject_type?.trim()); set("dist_subject_id", query.subject_id?.trim()); set("dist_owner", query.owner?.trim());
  window.history.replaceState(null, "", `${url.pathname}${url.search}${url.hash}`);
}

function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
