import { useEffect, useState } from "react";
import { loadActorWorkPage, type ActorWorkPage, type RuntimeContext } from "../api";
import { InternalResponse } from "./InternalResponse";

/** Isolated until internal response authority and document gates are verified. */
export function AssignedRequests({ context, allowOpening = false }: { context: RuntimeContext; allowOpening?: boolean }) {
  const [page, setPage] = useState<ActorWorkPage>();
  const [selected, setSelected] = useState<string>();
  const [status, setStatus] = useState<"loading" | "ready" | "unavailable">("loading");
  const [retry, setRetry] = useState(0);
  useEffect(() => {
    let active = true; setPage(undefined); setSelected(undefined); setStatus("loading");
    loadActorWorkPage("").then((value) => {
      if (!active) return;
      if (!context.actor.id || context.legal_entity.id === "*" || value.scope?.tenant_id !== context.tenant.id || value.scope?.legal_entity_id !== context.legal_entity.id || value.scope?.principal_id !== context.actor.id || value.coverage !== "WORKFLOW_ASSIGNMENTS_ONLY" || !Array.isArray(value.items)) { setStatus("unavailable"); return; }
      setPage(value); setStatus("ready");
    }).catch(() => { if (active) setStatus("unavailable"); });
    return () => { active = false; };
  }, [context.tenant.id, context.legal_entity.id, context.actor.id, retry]);
  const scopeMatches = page?.scope.tenant_id === context.tenant.id && page?.scope.legal_entity_id === context.legal_entity.id && page?.scope.principal_id === context.actor.id;
  if (page && !scopeMatches) return <p role="status">Loading assignments…</p>;
  if (selected && allowOpening && scopeMatches) return <InternalResponse key={selected} requestId={selected}/>;
  return <section><h1>Form assignments</h1>{status === "loading" && <p role="status">Loading assignments…</p>}{status === "unavailable" && <><p role="alert">Assignments unavailable.</p><button onClick={() => setRetry((value) => value + 1)}>Try again</button></>}{status === "ready" && page && <div className="v-panel">{page.items.filter((item) => item.action_target_type === "EVIDENCE_REQUEST" && item.action_target_id).map((item) => <div key={item.id}><strong>{item.title}</strong>{allowOpening ? <button type="button" aria-label={`Open ${item.title}`} onClick={() => setSelected(item.action_target_id)}>Open form</button> : <span>Form response unavailable.</span>}</div>)}{!page.items.some((item) => item.action_target_type === "EVIDENCE_REQUEST" && item.action_target_id) && <p>No form assignments in the first page.</p>}{page.has_more && <p>More assignments are available.</p>}</div>}</section>;
}
