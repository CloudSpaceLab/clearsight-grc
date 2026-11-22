import { useEffect, useState } from "react";
import { loadMatter, type RuntimeContext } from "../api";
import type { MatterAggregate } from "../types";
import { MatterResponses } from "./MatterResponses";
import { MatterOutcomes } from "./MatterOutcomes";

export function MatterWorkDetail({ id, context }: { id: string; context: RuntimeContext }) {
  const [matter, setMatter] = useState<MatterAggregate>();
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [attempt, setAttempt] = useState(0);
  const tenant = context.tenant.id, entity = context.legal_entity.id, actor = context.actor.id;

  useEffect(() => {
    let active = true;
    setMatter(undefined); setState("loading");
    if (!id || !tenant || !entity || entity === "*" || !actor) { setState("error"); return () => { active = false; }; }
    loadMatter(id).then((value) => {
      if (!active) return;
      if (!value?.matter || value.matter.id !== id || !value.scope || value.scope.tenant_id !== tenant || value.scope.legal_entity_id !== entity || value.scope.principal_id !== actor || !value.scope.record_tenant_id || !value.scope.record_legal_entity_id || value.scope.record_tenant_id !== value.matter.tenant_id || value.scope.record_legal_entity_id !== value.matter.legal_entity_id) { setState("error"); return; }
      setMatter(value); setState("ready");
    }).catch(() => { if (active) setState("error"); });
    return () => { active = false; };
  }, [id, tenant, entity, actor, attempt]);

  return <section className="v-matter-work"><a className="v-back" href="#/work">← Work &amp; approvals</a>
    {state === "loading" && <p role="status">Loading issue…</p>}
    {state === "error" && <div className="v-panel"><p role="alert">Issue unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></div>}
    {state === "ready" && matter && <><header className="v-page-heading"><div><p className="v-eyebrow">{matter.matter.reference || "Issue"}</p><h1>{matter.matter.title}</h1></div><span className="v-status">{matter.status_label || "Status unavailable"}</span></header>
      <section className="v-panel"><h2>Issue details</h2><p>{matter.matter.summary || "Summary not recorded."}</p><dl><div><dt>Owner</dt><dd>{matter.owner_display_name || (matter.matter.owner_principal_id ? "Owner name unavailable" : "Owner not recorded")}</dd></div></dl></section>
      <section className="v-panel">{matter.matter.status === "DECISION_REQUIRED" || matter.decisions?.length ? <><h2>Decision</h2><p>Decision unavailable.</p></> : <><h2>Issue work</h2><p>Issue actions unavailable.</p></>}</section>
      <MatterOutcomes key={`${tenant}:${entity}:${actor}:${id}:${matter.matter.version}`} aggregate={matter} onRefresh={() => setAttempt((value) => value + 1)} onAccessRevoked={() => { setMatter(undefined); setState("error"); }}/>
      <MatterResponses key={`${tenant}:${entity}:${actor}:${id}`} matterID={id}/>
    </>}
  </section>;
}
