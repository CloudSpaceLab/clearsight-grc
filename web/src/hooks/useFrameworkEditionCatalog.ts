import { useCallback, useEffect, useRef, useState } from "react";
import { loadFrameworkEditionPage } from "../programOperationsApi";
import type { FrameworkEditionOption } from "../programOperationsApi";

type CatalogState = "idle" | "loading" | "ready" | "error";

// Keep a bounded, server-cursor-backed list. Search in the consuming UI
// applies only to these loaded editions, never to an implied global catalog.
export function useFrameworkEditionCatalog(enabled: boolean, jurisdiction: string) {
  const requestID = useRef(0);
  const [items, setItems] = useState<FrameworkEditionOption[]>([]);
  const [nextAfterID, setNextAfterID] = useState("");
  const [state, setState] = useState<CatalogState>("idle");

  const load = useCallback(async (cursor = "", append = false) => {
    const current = ++requestID.current;
    if (!append) {
      setItems([]);
      setNextAfterID("");
    }
    setState("loading");
    try {
      const page = await loadFrameworkEditionPage(jurisdiction, cursor);
      if (current !== requestID.current) return;
      setItems((previous) => append ? [
        ...previous,
        ...page.items.filter((item) => !previous.some((existing) => existing.id === item.id)),
      ] : page.items);
      setNextAfterID(page.next_after_id);
      setState("ready");
    } catch {
      if (current === requestID.current) setState("error");
    }
  }, [jurisdiction]);

  useEffect(() => {
    if (enabled) void load();
    else {
      requestID.current += 1;
      setItems([]);
      setNextAfterID("");
      setState("idle");
    }
    return () => { requestID.current += 1; };
  }, [enabled, load]);

  return {
    items,
    nextAfterID,
    state,
    retry: () => void load(items.length ? nextAfterID : "", items.length > 0),
    loadMore: () => { if (state !== "loading" && nextAfterID) void load(nextAfterID, true); },
  };
}
