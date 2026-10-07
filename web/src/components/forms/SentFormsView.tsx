import { useEffect, useState, type ReactNode } from "react";
import { loadDistribution, loadDistributionPage, transitionDistribution, type Distribution, type DistributionDetail, type DistributionQuery } from "../../formsDistributionApi";
import { ApiError } from "../../http";
import { ActionLink, Button, EmptyState, FocusedSheet, Notice, Surface } from "../ui";
import { DistributionChangePanel } from "./DistributionChangePanel";
import { DistributionComposer } from "./DistributionComposer";
import { readFormsHashParams, updateFormsHashParams } from "./formsLocation";
import { SentFormDetail } from "./sent/SentFormDetail";
import { SentFormsFilters } from "./sent/SentFormsFilters";
import { SentFormsTable } from "./sent/SentFormsTable";
import { distributionStatusLabel } from "./sent/distributionPresentation";

type ListState = "loading" | "live" | "sign-in-required" | "error";
type DetailState = "idle" | "loading" | "live" | "error";

export function SentFormsView() {
  const wideDetail = useMediaQuery("(min-width: 1180px)");
  const [query, setQuery] = useState<DistributionQuery>(() => readQuery());
  const [listState, setListState] = useState<ListState>("loading");
  const [items, setItems] = useState<Distribution[]>([]);
  const [nextCursor, setNextCursor] = useState<string>();
  const [selectedID, setSelectedID] = useState<string | undefined>(() => readDistributionTarget());
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
  useEffect(() => {
    const syncFromLocation = () => {
      setQuery(readQuery());
      setSelectedID(readDistributionTarget());
    };
    window.addEventListener("hashchange", syncFromLocation);
    window.addEventListener("popstate", syncFromLocation);
    return () => {
      window.removeEventListener("hashchange", syncFromLocation);
      window.removeEventListener("popstate", syncFromLocation);
    };
  }, []);

  async function refresh() {
    setListState("loading");
    setError(undefined);
    try {
      const page = await loadDistributionPage({ ...query, cursor: undefined, limit: query.limit ?? 25 });
      setItems(page.items);
      setNextCursor(page.next_cursor);
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
      setItems((current) => appendUniqueByID(current, page.items));
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

  function selectDistribution(id: string) {
    setSelectedID(id);
    writeDistributionTarget(id);
  }

  function closeSelectedDistribution() {
    setSelectedID(undefined);
    writeDistributionTarget(undefined);
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
    setComposerOpen(false);
    setItems((current) => [value.distribution, ...current.filter((item) => item.id !== value.distribution.id)]);
    selectDistribution(value.distribution.id);
    setDetail(value);
    setDetailState("live");
  }}/>;
  if (changeMode && detail) return <DistributionChangePanel mode={changeMode} detail={detail} onCancel={() => setChangeMode(undefined)} onSaved={(value, resultNotice) => {
    setChangeMode(undefined);
    setItems((current) => [value.distribution, ...current.filter((item) => item.id !== detail.distribution.id && item.id !== value.distribution.id)]);
    selectDistribution(value.distribution.id);
    setDetail(value);
    setDetailState("live");
    setNotice(resultNotice);
  }}/>;

  const selectedItem = items.find((item) => item.id === selectedID);
  const selectedTitle = selectedItem?.title ?? detail?.distribution.title;
  const detailContent = selectedID
    ? renderDetailState(detailState, detail, detailError, busy, lifecycle, () => setChangeMode("amend"), () => setChangeMode("supersede"))
    : <><p className="forms-sent-detail__type">Distribution detail</p><h3>Select a sent form</h3></>;

  return <section className="forms-sent" aria-labelledby="sent-forms-title">
    <header className="forms-sent__heading"><div><p>Sender workspace</p><h2 id="sent-forms-title">Sent forms</h2></div><Button variant="primary" onPress={() => setComposerOpen(true)}>Send form</Button></header>
    {notice && <Notice tone="success">{notice}</Notice>}
    <SentFormsFilters query={query} resultCount={listState === "live" ? items.length : undefined} onChange={updateQuery} onClear={clearFilters}/>
    <div className="forms-sent__results" aria-live="polite">
      {listState === "loading" && <Surface><p role="status" aria-label="Loading sent forms matching the current filters">Loading sent forms…</p></Surface>}
      {listState === "sign-in-required" && <EmptyState population="Sent forms matching the current filters" title="Sign in to review sent forms" description="Session expired." action={<ActionLink href="/">Sign in again</ActionLink>}/>}
      {listState === "error" && <EmptyState population="Sent forms matching the current filters" title="Sent forms could not be loaded" description={error ?? "Retry."} action={<Button onPress={() => void refresh()}>Try again</Button>}/>}
      {listState === "live" && items.length === 0 && !selectedID && <EmptyState population="Sent forms matching the current filters" title="No sent forms match these filters" description="Change filters."/>}
      {listState === "live" && (items.length > 0 || selectedID) && <div className={`forms-sent__layout${wideDetail ? "" : " forms-sent__layout--single"}`}>
        {items.length > 0
          ? <SentFormsTable items={items} selectedID={selectedID} nextCursor={nextCursor} loadingMore={busy === "more"} onSelect={selectDistribution} onLoadMore={() => void loadMore()}/>
          : <EmptyState population="Sent forms matching the current filters" title="No sent forms match these filters" description="The selected sent form is outside this filtered list."/>}
        {wideDetail && <aside className="forms-sent__detail" aria-label={selectedTitle ? `${selectedTitle} details` : "Selected distribution"}>{detailContent}</aside>}
      </div>}
    </div>
    {!wideDetail && selectedID && <FocusedSheet label={`${selectedTitle ?? "Sent form"} details`} panelClassName="forms-sent__detail-sheet" onClose={closeSelectedDistribution}>{detailContent}</FocusedSheet>}
  </section>;
}

function renderDetailState(state: DetailState, detail: DistributionDetail | undefined, detailError: string | undefined, busy: string | undefined, lifecycle: (action: "lock" | "reopen" | "revoke") => Promise<void>, onAmend: () => void, onSupersede: () => void): ReactNode {
  if (state === "loading") return <p role="status" aria-label="Loading the selected sent form">Loading sent form…</p>;
  if (state === "error") return <Notice tone="error">{detailError} Select the sent form again to retry.</Notice>;
  if (state === "live" && detail) return <SentFormDetail detail={detail} error={detailError} busy={busy} onLifecycle={(action) => void lifecycle(action)} onAmend={onAmend} onSupersede={onSupersede}/>;
  return null;
}

function useMediaQuery(query: string) {
  const [matches, setMatches] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const media = window.matchMedia(query);
    const update = () => setMatches(media.matches);
    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [query]);
  return matches;
}

function readQuery(): DistributionQuery {
  const params = readFormsHashParams(window.location.hash);
  const status = params.get("dist_status") as DistributionQuery["status"] | null;
  const due = params.get("dist_due") as DistributionQuery["due_state"] | null;
  return {
    status: status || undefined,
    due_state: due || undefined,
    subject_type: params.get("dist_subject_type") || undefined,
    subject_id: params.get("dist_subject_id") || undefined,
    owner: params.get("dist_owner") || undefined,
    limit: 25,
  };
}

function readDistributionTarget() {
  return readFormsHashParams(window.location.hash).get("distribution") || undefined;
}

function writeQuery(query: DistributionQuery) {
  updateFormsHashParams((params) => {
    setParam(params, "dist_status", query.status);
    setParam(params, "dist_due", query.due_state);
    setParam(params, "dist_subject_type", query.subject_type?.trim());
    setParam(params, "dist_subject_id", query.subject_id?.trim());
    setParam(params, "dist_owner", query.owner?.trim());
  });
}

function writeDistributionTarget(id?: string) {
  updateFormsHashParams((params) => setParam(params, "distribution", id));
}

function setParam(params: URLSearchParams, key: string, value?: string) {
  if (value) params.set(key, value);
  else params.delete(key);
}

function appendUniqueByID<T extends { id: string }>(current: T[], incoming: T[]) {
  if (incoming.length === 0) return current;
  const seen = new Set(current.map((item) => item.id));
  return [...current, ...incoming.filter((item) => !seen.has(item.id))];
}

function message(cause: unknown, fallback: string) { return cause instanceof Error ? cause.message : fallback; }
