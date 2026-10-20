import { useCallback, useEffect, useRef, useState } from "react";
import { loadActorWorkPage } from "../api";
import type { AttentionItem } from "../types";

type SearchResult = {
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

// Source search is title-only; it must not be interpreted as a global Work,
// approval or administrator-exception query.
export function useActorWorkSearch(search: string, scopeKey: string, enabled: boolean) {
  const term = enabled ? search.trim() : "";
  const generation = useRef(0);
  const [retryToken, setRetryToken] = useState(0);
  const [result, setResult] = useState<SearchResult>(emptyResult);

  useEffect(() => {
    const current = ++generation.current;
    if (!term) {
      setResult(emptyResult);
      return;
    }
    setResult({ ...emptyResult, state: "loading" });
    const timer = window.setTimeout(() => {
      void loadActorWorkPage(term).then((page) => {
        if (generation.current !== current) return;
        setResult({
          state: "live", items: page.items, asOf: page.as_of,
          cursor: page.next_cursor, hasMore: page.has_more,
          loadingMore: false, loadMoreFailed: false,
        });
      }).catch(() => {
        if (generation.current === current) setResult({ ...emptyResult, state: "unavailable" });
      });
    }, 220);
    return () => {
      window.clearTimeout(timer);
      generation.current++;
    };
  }, [term, scopeKey, retryToken]);

  const loadMore = useCallback(() => {
    if (!term || result.state !== "live" || !result.hasMore ||
        !result.cursor || result.loadingMore) return;
    const current = generation.current;
    const cursor = result.cursor;
    setResult((previous) => ({ ...previous, loadingMore: true, loadMoreFailed: false }));
    void loadActorWorkPage(term, cursor).then((page) => {
      if (generation.current !== current) return;
      setResult((previous) => {
        const existing = new Set(previous.items.map((item) => item.id));
        if (page.as_of !== previous.asOf) {
          return { ...previous, loadingMore: false, loadMoreFailed: true };
        }
        return {
          ...previous,
          items: [...previous.items, ...page.items.filter((item) => !existing.has(item.id))],
          cursor: page.next_cursor, hasMore: page.has_more,
          loadingMore: false, loadMoreFailed: false,
        };
      });
    }).catch(() => {
      if (generation.current === current) {
        setResult((previous) => ({ ...previous, loadingMore: false, loadMoreFailed: true }));
      }
    });
  }, [result, term]);

  return { ...result, loadMore, retry: () => setRetryToken((value) => value + 1) };
}
