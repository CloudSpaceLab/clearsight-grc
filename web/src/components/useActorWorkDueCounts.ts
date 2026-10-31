import { useCallback, useEffect, useRef, useState } from "react";
import { loadActorWorkDueCounts, type ActorWorkDueCounts, type ActorWorkScope } from "../api";
import { validWorkCounts } from "./workSourceReceipt";

type CountState = { key: string; value?: ActorWorkDueCounts; unavailable: boolean };

// The response is source-qualified, not a full Work/exception snapshot. State
// is bound to the actor, entity and source invalidation token before render.
export function useActorWorkDueCounts(scope: ActorWorkScope | undefined, scopeKey?: string) {
  const principalID = scope?.principalID ?? "";
  const tenantID = scope?.tenantID ?? "";
  const legalEntityID = scope?.legalEntityID ?? "";
  const key = scopeKey && principalID && tenantID && legalEntityID
    ? `${scopeKey}\u0000${tenantID}\u0000${principalID}\u0000${legalEntityID}` : "";
  const [state, setState] = useState<CountState>({ key, unavailable: false });
  const generation = useRef(0);
  // Discard the previous visit during render, so A → B → A cannot expose A's
  // prior counts before its new effect has read the source again.
  if (state.key !== key) setState({ key, unavailable: false });
  const invalidate = useCallback(() => {
    generation.current++;
    setState({ key, unavailable: true });
  }, [key]);

  useEffect(() => {
    const current = ++generation.current;
    if (!key) return;
    void loadActorWorkDueCounts().then((value) => {
      if (generation.current !== current) return;
      setState(validWorkCounts(value, { principalID, tenantID, legalEntityID })
        ? { key, value, unavailable: false }
        : { key, unavailable: true });
    }).catch(() => {
      if (generation.current === current) setState({ key, unavailable: true });
    });
    return () => { generation.current++; };
  }, [key, principalID, tenantID, legalEntityID]);

  return {
    value: state?.key === key ? state.value : undefined,
    unavailable: Boolean(key && state?.key === key && state.unavailable),
    invalidate,
  };
}
