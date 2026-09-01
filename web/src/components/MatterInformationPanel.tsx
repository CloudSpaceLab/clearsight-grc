import { useState } from "react";
import type { FormEvent } from "react";
import { apiErrorKind } from "../http";
import { changeMatterContext } from "../matterOperationsApi";
import type { MatterContextChangeKind, MatterOperation } from "../matterOperationsApi";
import type { MatterAggregate } from "../types";
import { matterOperationControlID } from "./matterHandoff";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onReload: () => void;
};

type ActiveChange =
  | { type: "fact"; key: string; label: string; current: unknown }
  | { type: "missing"; label: string }
  | { type: "contradiction"; label: string }
  | { type: "add-fact" }
  | { type: "add-missing" }
  | { type: "add-contradiction" }
  | null;

function humanize(value: string) {
  return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function display(value: unknown) {
  if (value === null || value === undefined || value === "") return "Not recorded";
  return typeof value === "object" ? JSON.stringify(value) : String(value);
}

function slug(value: string) {
  return value.trim().toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_+|_+$/g, "");
}

function lines(value: string) {
  return value.split(/\r?\n/).map((item) => item.trim()).filter(Boolean);
}

function scalar(value: string, current?: unknown) {
  if (typeof current === "number" && value.trim() !== "" && Number.isFinite(Number(value))) return Number(value);
  if (typeof current === "boolean" && ["true", "false"].includes(value.trim().toLowerCase())) return value.trim().toLowerCase() === "true";
  return value.trim();
}

export function MatterInformationPanel({ aggregate, operations, onUpdated, onReload }: Props) {
  const operation = operations.find((candidate) => candidate.command === "matter.context.change");
  const [active, setActive] = useState<ActiveChange>(null);
  const [label, setLabel] = useState("");
  const [key, setKey] = useState("");
  const [value, setValue] = useState("");
  const [evidence, setEvidence] = useState("");
  const [rationale, setRationale] = useState("");
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);

  function start(next: Exclude<ActiveChange, null>) {
    setActive(next); setError(""); setConflict(false); setNotice(""); setRationale(""); setEvidence("");
    if (next.type === "fact") { setLabel(next.label); setKey(next.key); setValue(display(next.current)); return; }
    if (next.type === "missing" || next.type === "contradiction") { setLabel(next.label); setKey(slug(next.label)); setValue(""); return; }
    setLabel(""); setKey(""); setValue("");
  }

  function handleError(cause: unknown) {
    const isConflict = apiErrorKind(cause) === "conflict";
    setConflict(isConflict);
    setError(isConflict ? "This issue changed since you opened it. Your entries have been kept." : cause instanceof Error && cause.message ? cause.message : "The information could not be changed.");
  }

  async function submit(event: FormEvent, forcedKind?: MatterContextChangeKind) {
    event.preventDefault();
    if (!active) return;
    let kind: MatterContextChangeKind;
    if (forcedKind) kind = forcedKind;
    else if (active.type === "fact") kind = "CORRECT_FACT";
    else if (active.type === "missing") kind = "RESOLVE_MISSING";
    else if (active.type === "contradiction") kind = "RESOLVE_CONTRADICTION";
    else if (active.type === "add-fact") kind = "ADD_FACT";
    else if (active.type === "add-missing") kind = "ADD_MISSING";
    else kind = "ADD_CONTRADICTION";
    const businessLabel = (active.type === "fact" || active.type === "missing" || active.type === "contradiction") ? active.label : label.trim();
    const businessKey = (active.type === "fact") ? active.key : key.trim() || slug(businessLabel);
    setSaving(true); setError(""); setConflict(false); setNotice("");
    try {
      const updated = await changeMatterContext(aggregate.matter.id, aggregate.matter.version, {
        kind,
        key: ["ADD_MISSING", "ADD_CONTRADICTION", "RESOLVE_CONTRADICTION"].includes(kind) ? undefined : businessKey,
        label: businessLabel,
        value: ["RETIRE_FACT", "ADD_MISSING", "ADD_CONTRADICTION", "RESOLVE_CONTRADICTION"].includes(kind) ? undefined : scalar(value, active.type === "fact" ? active.current : undefined),
        evidenceReferences: lines(evidence),
        rationale: rationale.trim(),
      });
      await onUpdated(updated);
      setActive(null); setNotice(kind === "RESOLVE_MISSING" ? "Missing information recorded." : "Issue information updated.");
    } catch (cause) { handleError(cause); } finally { setSaving(false); }
  }

  const facts = Object.entries(aggregate.matter.known_facts);
  return <article className="matter-record-panel matter-information-panel" id="matter-operation-matter.context.change">
    <div className="matter-record-section-heading">
      <div><span className="eyebrow">Evidence and facts</span><h2>Recorded information</h2></div>
      {operation?.can_act && !active && <div className="matter-panel-actions"><button id={aggregate.matter.missing_facts.length === 0 ? matterOperationControlID(operation) : undefined} className="secondary-button" type="button" onClick={() => start({ type: "add-fact" })}>Add recorded fact</button><button className="secondary-button" type="button" onClick={() => start({ type: "add-missing" })}>Add missing information</button></div>}
    </div>
    {facts.length ? <dl className="matter-record-facts">{facts.map(([factKey, factValue]) => { const factLabel = humanize(factKey); return <div key={factKey}><dt>{factLabel}</dt><dd><span>{display(factValue)}</span>{operation?.can_act && <button className="text-button" type="button" aria-label={`Edit ${factLabel}`} onClick={() => start({ type: "fact", key: factKey, label: factLabel, current: factValue })}>Edit</button>}</dd></div>; })}</dl> : <p>No facts have been recorded for this issue.</p>}
    {aggregate.matter.missing_facts.length > 0 && <section className="matter-record-attention" aria-labelledby="missing-information-heading"><strong id="missing-information-heading">Information still needed</strong><ul className="matter-information-list">{aggregate.matter.missing_facts.map((item, index) => { const itemLabel = display(item); return <li key={`${index}-${itemLabel}`}><span>{itemLabel}</span>{operation?.can_act && <button id={index === 0 ? matterOperationControlID(operation) : undefined} className="secondary-button" type="button" aria-label={`Add information for ${itemLabel}`} onClick={() => start({ type: "missing", label: itemLabel })}>Add information</button>}</li>; })}</ul></section>}
    {aggregate.matter.contradictions.length > 0 && <section className="matter-record-attention"><strong>Contradictions to resolve</strong><ul className="matter-information-list">{aggregate.matter.contradictions.map((item, index) => { const itemLabel = display(item); return <li key={`${index}-${itemLabel}`}><span>{itemLabel}</span>{operation?.can_act && <button className="secondary-button" type="button" aria-label={`Resolve contradiction: ${itemLabel}`} onClick={() => start({ type: "contradiction", label: itemLabel })}>Resolve</button>}</li>; })}</ul></section>}
    {operation?.can_act && !active && <button className="text-button matter-add-contradiction" type="button" onClick={() => start({ type: "add-contradiction" })}>Record a contradiction</button>}
    {active && <form className="matter-operation-form" onSubmit={(event) => void submit(event)}>
      {active.type === "add-fact" && <><label><span>Fact name</span><input value={label} onChange={(event) => { setLabel(event.target.value); setKey(slug(event.target.value)); }} required/></label><label><span>Fact key</span><input value={key} onChange={(event) => setKey(event.target.value)} pattern="[a-z0-9_]+" required/></label></>}
      {(active.type === "add-missing" || active.type === "add-contradiction") && <label className="wide"><span>{active.type === "add-missing" ? "Information needed" : "Contradiction to record"}</span><input value={label} onChange={(event) => setLabel(event.target.value)} required/></label>}
      {(active.type === "fact" || active.type === "add-fact") && <label className="wide"><span>{active.type === "fact" ? "Updated value" : "Fact value"}</span><input value={value} onChange={(event) => setValue(event.target.value)} required/></label>}
      {active.type === "missing" && <label className="wide"><span>Information to record</span><input value={value} onChange={(event) => setValue(event.target.value)} required/></label>}
      {(active.type === "fact" || active.type === "add-fact" || active.type === "missing") && <label className="wide"><span>Evidence references (optional)</span><textarea value={evidence} onChange={(event) => setEvidence(event.target.value)} rows={2} placeholder="One artifact, source or link per line"/></label>}
      <label className="wide"><span>{active.type === "fact" ? "Reason for this correction" : active.type === "missing" ? "Reason this information resolves the gap" : active.type === "add-missing" ? "Why this information is needed" : active.type === "contradiction" ? "Reason this contradiction is resolved" : active.type === "add-contradiction" ? "Why these sources conflict" : "Reason for this addition"}</span><textarea value={rationale} onChange={(event) => setRationale(event.target.value)} rows={2} required/></label>
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide">
        <button className="primary-button" type="submit" disabled={saving}>{saving ? "Saving…" : active.type === "fact" ? `Save ${active.label}` : active.type === "missing" ? "Record missing information" : active.type === "add-missing" ? "Add missing item" : active.type === "contradiction" ? "Resolve contradiction" : active.type === "add-contradiction" ? "Record contradiction" : "Add fact"}</button>
        {active.type === "fact" && <button className="secondary-button" type="button" disabled={saving} onClick={(event) => void submit(event as unknown as FormEvent, "RETIRE_FACT")}>Retire {active.label}</button>}
        <button className="text-button" type="button" onClick={() => setActive(null)}>Cancel</button>
      </div>
    </form>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
    {!operation?.can_act && operation?.reason && <p className="matter-operation-reason">{operation.reason}</p>}
  </article>;
}
