import { useCallback, useEffect, useRef, useState } from "react";
import { loadFrameworkEditionPage } from "../programOperationsApi";
import { apiErrorKind } from "../http";
import type { FrameworkEditionOption } from "../programOperationsApi";

type CatalogState = "idle" | "loading" | "ready" | "error" | "denied";

// Keep a bounded, server-cursor-backed list. Search in the consuming UI
// applies only to these loaded editions, never to an implied global catalog.
export function useFrameworkEditionCatalog(enabled: boolean, jurisdiction: string) {
  const requestID = useRef(0);
  const [items, setItems] = useState<FrameworkEditionOption[]>([]);
  const [nextAfterID, setNextAfterID] = useState("");
  const [state, setState] = useState<CatalogState>("idle");
  const [loadedJurisdiction, setLoadedJurisdiction] = useState<string | null>(null);

  const load = useCallback(async (cursor = "", append = false) => {
    const current = ++requestID.current;
    if (!append) {
      setItems([]);
      setNextAfterID("");
      setLoadedJurisdiction(null);
    }
    setState("loading");
    try {
      const page = await loadFrameworkEditionPage(jurisdiction, cursor);
      if (current !== requestID.current) return;
      setItems((previous) => {
        if (!append) return page.items;
        const existing = new Set(previous.map((item) => item.id));
        return [...previous, ...page.items.filter((item) => !existing.has(item.id))];
      });
      setNextAfterID(page.next_after_id);
      setLoadedJurisdiction(jurisdiction);
      setState("ready");
    } catch (cause) {
      if (current !== requestID.current) return;
      const kind = apiErrorKind(cause);
      if (kind === "forbidden" || kind === "unauthorized") {
        setItems([]);
        setNextAfterID("");
        setLoadedJurisdiction(null);
        setState("denied");
      } else setState("error");
    }
  }, [jurisdiction]);

  useEffect(() => {
    if (enabled) void load();
    else {
      requestID.current += 1;
      setItems([]);
      setNextAfterID("");
      setLoadedJurisdiction(null);
      setState("idle");
    }
    return () => { requestID.current += 1; };
  }, [enabled, load]);

  return {
    items,
    nextAfterID,
    state,
    loadedJurisdiction,
    retry: () => {
      const canContinue = loadedJurisdiction === jurisdiction && items.length > 0 && nextAfterID;
      void load(canContinue ? nextAfterID : "", Boolean(canContinue));
    },
    loadMore: () => {
      if (state !== "ready" || loadedJurisdiction !== jurisdiction || !nextAfterID) return;
      void load(nextAfterID, true);
    },
  };
}
