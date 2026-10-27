import { useEffect, useState } from "react";
import { loadActorWorkDueCounts, type ActorWorkDueCounts } from "../api";

type ActorCountScope = { principalID: string; legalEntityID: string };
type CountState = { key: string; value?: ActorWorkDueCounts; unavailable: boolean };

function validSourceCounts(value: ActorWorkDueCounts, scope: ActorCountScope): boolean {
  const { total, overdue, due_soon: dueSoon, undated } = value;
  return value.coverage === "WORKFLOW_ASSIGNMENTS_ONLY" &&
    value.count_basis === "SOURCE_SNAPSHOT" &&
    value.scope?.principal_id === scope.principalID &&
    value.scope?.legal_entity_id === scope.legalEntityID &&
    Number.isFinite(Date.parse(value.as_of)) &&
    [total, overdue, dueSoon, undated].every((count) => Number.isSafeInteger(count) && count >= 0) &&
    overdue + dueSoon + undated <= total;
}

// The response is source-qualified, not a full Work/exception snapshot. State
// is bound to the actor, entity and source invalidation token before render.
export function useActorWorkDueCounts(scope: ActorCountScope | undefined, scopeKey?: string) {
  const principalID = scope?.principalID ?? "";
  const legalEntityID = scope?.legalEntityID ?? "";
  const key = scopeKey && principalID && legalEntityID
    ? `${scopeKey}\u0000${principalID}\u0000${legalEntityID}` : "";
  const [state, setState] = useState<CountState>();

  useEffect(() => {
    if (!key) return;
    let current = true;
    void loadActorWorkDueCounts().then((value) => {
      if (!current) return;
      setState(validSourceCounts(value, { principalID, legalEntityID })
        ? { key, value, unavailable: false }
        : { key, unavailable: true });
    }).catch(() => {
      if (current) setState({ key, unavailable: true });
    });
    return () => { current = false; };
  }, [key, principalID, legalEntityID]);

  return {
    value: state?.key === key ? state.value : undefined,
    unavailable: Boolean(key && state?.key === key && state.unavailable),
  };
}
