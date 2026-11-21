import { useEffect, useRef, useState } from "react";
import { transitionFormTemplateRevision } from "../formsApi";
import { loadContext, resolveAuthority } from "../api";
import type { FormLibraryItem } from "../formsTypes";
import type { LifecycleStatus } from "../monitoringTypes";

const labels: Partial<Record<LifecycleStatus, string>> = { PENDING_APPROVAL: "Send for approval", ACTIVE: "Approve form", REJECTED: "Reject form", PAUSED: "Pause form", RETIRED: "Retire form" };

export function FormActions({ item, onReload, onSaved }: { item: FormLibraryItem; onReload: () => void; onSaved?: (result: { id: string; version: number; status: LifecycleStatus }) => void }) {
  const [busy, setBusy] = useState(false); const [state, setState] = useState<"ready" | "unknown" | "saved">("ready"); const lock = useRef(false);
  const action = item.authority_available ? item.operations?.find((operation) => operation.command === "forms.template.transition" && operation.can_act) : undefined;
  const targets = action?.allowed_targets?.filter((target) => labels[target]) ?? [];
  const [serviceAuthority, setServiceAuthority] = useState<"checking" | "allowed" | "denied">("checking");
  useEffect(() => {
    let active = true; setServiceAuthority("checking");
    if (!action) return;
    Promise.all([loadContext(), item.template.status === "PENDING_APPROVAL" ? Promise.resolve(undefined) : resolveAuthority({ object_type: "FORM_TEMPLATE", object_id: item.template.id, responsibility: "ACCOUNTABLE_OWNER", decision_type: "forms.template.transition", materiality: 3 })]).then(([context, route]) => {
      const human = Boolean(context.actor.id) && context.actor.kind?.toUpperCase() !== "SERVICE";
      if (active) setServiceAuthority(human && (!route || route.principal?.id === context.actor.id || route.candidate_principals?.some((person) => person.id === context.actor.id)) ? "allowed" : "denied");
    }).catch(() => { if (active) setServiceAuthority("denied"); });
    return () => { active = false; };
  }, [item.template.id, item.template.version, item.template.status, Boolean(action)]);
  async function transition(target: LifecycleStatus) {
    if (lock.current || state !== "ready" || serviceAuthority !== "allowed" || !targets.includes(target)) return;
    lock.current = true; setBusy(true);
    try {
      let result;
      try {
        result = await transitionFormTemplateRevision(item.template.id, item.template.version, target);
        if (result.id !== item.template.id || result.status !== target || result.version !== item.template.version + 1) throw new Error("unconfirmed");
      } catch { setState("unknown"); return; }
      setState("saved");
      // A confirmed decision remains confirmed even if a local refresh fails.
      try { onSaved?.(result); } catch { /* retain the confirmed status */ }
      try { onReload(); } catch { /* retain the confirmed status */ }
    }
    finally { setBusy(false); }
  }
  if (state === "unknown") return <div><p role="alert">Form status unknown. Reload before trying again.</p><button onClick={onReload}>Reload form</button></div>;
  if (state === "saved") return <p role="status">Form status updated.</p>;
  if (!targets.length) return null;
  if (serviceAuthority === "checking") return <p role="status">Checking form access…</p>;
  if (serviceAuthority === "denied") return <p>Form action unavailable. Check your role.</p>;
  return <div className="v-form-actions">{targets.map((target) => <button key={target} className={target === "REJECTED" ? undefined : "v-primary"} disabled={busy} onClick={() => void transition(target)}>{target === "ACTIVE" && item.template.status === "PAUSED" ? "Resume form" : labels[target]}</button>)}</div>;
}
