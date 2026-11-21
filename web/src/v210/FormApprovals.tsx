import { useEffect, useRef, useState } from "react";
import { loadFormTemplatePage } from "../formsApi";
import type { FormLibraryItem, FormTemplatePage } from "../formsTypes";
import { apiErrorKind } from "../http";

type PageState = { status: "loading" | "ready" | "error"; items: FormLibraryItem[]; cursor?: string; incomplete: boolean; moreBusy: boolean; moreError: boolean };
const empty = (status: PageState["status"]): PageState => ({ status, items: [], incomplete: false, moreBusy: false, moreError: false });
function actionable(item: FormLibraryItem) {
  return item.template.status === "PENDING_APPROVAL" && item.authority_available === true && item.operations?.some((operation) => operation.command === "forms.template.transition" && operation.can_act && operation.allowed_targets?.some((target) => target === "ACTIVE" || target === "REJECTED"));
}
function append(page: FormTemplatePage) {
  return { items: page.items.filter(actionable), incomplete: page.items.some((item) => item.authority_available !== true), cursor: page.next_cursor };
}

export function FormApprovals() {
  const [state, setState] = useState<PageState>(() => empty("loading"));
  const [attempt, setAttempt] = useState(0);
  const generation = useRef(0); const moreLock = useRef(false);
  useEffect(() => {
    const controller = new AbortController(); const current = ++generation.current;
    moreLock.current = false; setState(empty("loading"));
    loadFormTemplatePage({ status: "PENDING_APPROVAL", limit: 20 }, controller.signal).then((page) => {
      if (current === generation.current) setState({ ...empty("ready"), ...append(page) });
    }).catch(() => { if (!controller.signal.aborted && current === generation.current) setState(empty("error")); });
    return () => { controller.abort(); generation.current++; };
  }, [attempt]);
  async function more() {
    if (!state.cursor || state.moreBusy || moreLock.current) return;
    const current = generation.current; const cursor = state.cursor;
    moreLock.current = true; setState((value) => ({ ...value, moreBusy: true, moreError: false }));
    try {
      const page = await loadFormTemplatePage({ status: "PENDING_APPROVAL", limit: 20, cursor });
      if (current !== generation.current) return;
      if (page.next_cursor === cursor) { setState(empty("error")); return; }
      const next = append(page);
      setState((value) => { const seen = new Set(value.items.map((item) => item.template.id)); return { ...value, items: [...value.items, ...next.items.filter((item) => !seen.has(item.template.id))], cursor: next.cursor, incomplete: value.incomplete || next.incomplete, moreBusy: false }; });
    } catch (error) {
      if (current !== generation.current) return;
      if (["unauthorized", "forbidden"].includes(apiErrorKind(error))) setState(empty("error"));
      else setState((value) => ({ ...value, moreBusy: false, moreError: true }));
    } finally { if (current === generation.current) moreLock.current = false; }
  }
  return <section className="v-panel v-work-list v-form-approvals" aria-labelledby="v-form-approvals-title">
    <div className="v-work-list-heading"><h2 id="v-form-approvals-title">Form approvals</h2></div>
    {state.status === "loading" && <p role="status">Loading form approvals…</p>}
    {state.status === "error" && <><p role="alert">Form approvals unavailable.</p><button type="button" onClick={() => setAttempt((value) => value + 1)}>Try again</button></>}
    {state.status === "ready" && <>
      {state.incomplete && <p role="alert">Form approval access unavailable.</p>}
      {state.items.length > 0 ? <ul>{state.items.map(({ template }) => <li key={template.id}><div><strong>{template.name}</strong><span>Version {template.version}</span></div><span className="v-work-status">Pending approval</span><a href={`#/forms/${encodeURIComponent(template.id)}?approval_version=${template.version}`} aria-label={`Review ${template.name}`}>Review form</a></li>)}</ul> : !state.incomplete && <p className="v-work-empty">{state.cursor ? "No form approvals on this page." : "No forms available for your approval."}</p>}
      {state.moreError && <p role="alert">More forms unavailable. Try again.</p>}
      {state.cursor && <button className="v-work-more" type="button" disabled={state.moreBusy} onClick={() => void more()}>{state.moreBusy ? "Loading…" : "Load more forms"}</button>}
    </>}
  </section>;
}
