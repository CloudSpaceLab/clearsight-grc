import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { loadActorWorkPage, type ActorWorkScope } from "../api";
import { apiErrorKind } from "../http";
import type { AttentionItem } from "../types";
import { workDeadlineFilters, workSourceFilters, type WorkDeadlineDrill } from "./workSourceFilters";
import { validWorkPage } from "./workSourceReceipt";
import type { WorkDueWindow } from "./workQueuePresentation";

type SearchResult = {
  queryKey?: string;
  state: "idle" | "loading" | "live" | "unavailable";
  items: AttentionItem[];
  asOf?: string;
  cursor?: string;
  hasMore: boolean;
  loadingMore: boolean;
  loadMoreFailed: boolean;
};

const emptyResult: SearchResult = {
  state: "idle", items: [], hasMore: false, loadingMore: false, loadMoreFailed: false,
};

// Title search and deadline drills share the authorized source reader.
// Neither represents the mixed Today/admin queue or completed work.
export function useActorWorkSearch({ search, scopeKey, scope, enabled, status, dueWindow = "ALL", deadline, onAccessLost }: {
  search: string;
  scopeKey: string;
  scope?: ActorWorkScope;
  enabled: boolean;
  status?: string;
  dueWindow?: WorkDueWindow;
  deadline?: WorkDeadlineDrill;
  onAccessLost?: () => void;
}) {
  const { tenantID = "", legalEntityID = "", principalID = "" } = scope ?? {};
  const expectedScope = useMemo(() => ({ tenantID, legalEntityID, principalID }), [tenantID, legalEntityID, principalID]);
  const term = search.trim();
  const active = Boolean(enabled && scopeKey && tenantID && legalEntityID && principalID && (term || deadline));
  const deadlineKind = deadline?.kind;
  const deadlineAsOf = deadline?.asOf;
  const filters = useMemo(() => ({
    ...workSourceFilters(status, deadlineKind ? "ALL" : dueWindow, Date.now()),
    ...(deadlineKind && deadlineAsOf ? workDeadlineFilters({ kind: deadlineKind, asOf: deadlineAsOf }) : {}),
  }), [status, dueWindow, scopeKey, term, deadlineKind, deadlineAsOf]);
  const queryKey = JSON.stringify([active, scopeKey, expectedScope, term, filters]);
  const loadPage = useCallback((cursor?: string) => {
    if (Object.keys(filters).length) return loadActorWorkPage(term, cursor, filters);
    return cursor ? loadActorWorkPage(term, cursor) : loadActorWorkPage(term);
  }, [term, filters]);
  const generation = useRef(0);
  const [retryToken, setRetryToken] = useState(0);
  const [result, setResult] = useState<SearchResult>({ ...emptyResult, queryKey });
  if (result.queryKey !== queryKey) {
    setResult({ ...emptyResult, queryKey, state: active ? "loading" : "idle" });
  }
  const rejectReceipt = useCallback(() => {
    setResult({ ...emptyResult, queryKey, state: "unavailable" });
    onAccessLost?.();
  }, [queryKey, onAccessLost]);

  useEffect(() => {
    const current = ++generation.current;
    if (!active) return;
    setResult({ ...emptyResult, queryKey, state: "loading" });
    const timer = window.setTimeout(() => {
      void loadPage().then((page) => {
        if (generation.current !== current) return;
        if (!validWorkPage(page, expectedScope)) { rejectReceipt(); return; }
        setResult({
          queryKey, state: "live", items: page.items, asOf: page.as_of,
          cursor: page.next_cursor, hasMore: page.has_more,
          loadingMore: false, loadMoreFailed: false,
        });
      }).catch((error) => {
        if (generation.current !== current) return;
        setResult({ ...emptyResult, queryKey, state: "unavailable" });
        if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) onAccessLost?.();
      });
    }, term ? 220 : 0);
    return () => {
      window.clearTimeout(timer);
      generation.current++;
    };
  }, [active, term, queryKey, loadPage, retryToken, expectedScope, rejectReceipt, onAccessLost]);

  const loadMore = useCallback(() => {
    if (!active || result.queryKey !== queryKey || result.state !== "live" || !result.hasMore ||
        !result.cursor || result.loadingMore) return;
    const current = generation.current;
    const cursor = result.cursor;
    setResult((previous) => ({ ...previous, loadingMore: true, loadMoreFailed: false }));
    void loadPage(cursor).then((page) => {
      if (generation.current !== current) return;
      if (!validWorkPage(page, expectedScope) || page.as_of !== result.asOf || page.next_cursor === cursor) {
        rejectReceipt();
        return;
      }
      setResult((previous) => {
        const existing = new Set(previous.items.map((item) => item.id));
        return {
          ...previous,
          items: [...previous.items, ...page.items.filter((item) => !existing.has(item.id))],
          cursor: page.next_cursor, hasMore: page.has_more,
          loadingMore: false, loadMoreFailed: false,
        };
      });
    }).catch((error) => {
      if (generation.current !== current) return;
      if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) rejectReceipt();
      else setResult((previous) => ({ ...previous, loadingMore: false, loadMoreFailed: true }));
    });
  }, [result, active, queryKey, loadPage, expectedScope, rejectReceipt]);

  const visible = result.queryKey === queryKey ? result : { ...emptyResult, state: active ? "loading" as const : "idle" as const };
  return { ...visible, loadMore, retry: () => setRetryToken((value) => value + 1) };
}
