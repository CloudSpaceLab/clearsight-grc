import { useEffect, useState } from "react";
import { loadMatterSummaries } from "../api";
import type { MatterSummary } from "../summaryTypes";
import type { ProgramAggregate } from "../types";
import { MatterSetupWorkspace } from "./MatterSetupWorkspace";

type Props = { aggregate: ProgramAggregate; onOpenMatter: (matterID: string) => void };

export function ProgramIssuesPanel({ aggregate, onOpenMatter }: Props) {
  const [items, setItems] = useState<MatterSummary[]>([]);
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [creating, setCreating] = useState(false);

  async function reload() {
    setState("loading");
    try {
      const page = await loadMatterSummaries({ status: "OPEN", programID: aggregate.program.id, limit: 20 });
      setItems(page.items); setState("live");
    } catch { setState("unavailable"); }
  }

  useEffect(() => { void reload(); }, [aggregate.program.id]);

  return <article className="program-record-panel program-wide-panel" id="program-issues-panel">
    <div className="program-panel-heading"><div><span className="eyebrow">Issues and changes</span><h2>Linked issues and changes</h2></div><div className="program-panel-actions"><button className="secondary-button" type="button" onClick={() => setCreating((value) => !value)}>{creating ? "Close issue form" : "Record new issue"}</button></div></div>
    <p>These open items are linked to this Program. Open an item to assign work, record decisions and check the outcome.</p>
    {state === "loading" && <p aria-live="polite">Loading linked issues and changes…</p>}
    {state === "unavailable" && <div className="inline-error"><p>Linked issues could not be loaded.</p><button className="secondary-button" type="button" onClick={() => void reload()}>Try again</button></div>}
    {state === "live" && (items.length ? <div className="program-issue-list">{items.map((item) => <section className="program-issue-card" key={item.matter.id}>
      <div><span>{item.type_label} · {item.status_label}</span><h3>{item.matter.title}</h3><p>{item.matter.summary}</p><small>{item.next_action} · priority {item.matter.priority}</small></div>
      <button className="secondary-button" type="button" onClick={() => onOpenMatter(item.matter.id)}>Open {item.matter.reference}</button>
    </section>)}</div> : <div className="program-empty-state"><strong>No open linked issues</strong><p>The current query found no open issues or changes linked to this Program. Record one if a gap, exception, request or change needs assigned work.</p></div>)}
    {creating && <MatterSetupWorkspace initialProgramID={aggregate.program.id} onClose={() => setCreating(false)} onCreated={(value) => onOpenMatter(value.matter.id)}/>} 
  </article>;
}
