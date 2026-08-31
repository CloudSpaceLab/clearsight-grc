import { useState } from "react";
import type { FormEvent } from "react";
import { apiErrorKind } from "../http";
import { loadResponsePackageHistory } from "../api";
import type { MatterOperation } from "../matterOperationsApi";
import { addResponsePackage, recordMatterDecision, transitionResponsePackage } from "../continuityCommands";
import type { MatterAggregate, ResponseHistoryPage } from "../types";
import { matterResponsePresentation } from "./matterResponsePresentation";
import { Button, FocusedSheet, Notice, SelectField, TextArea } from "./ui";

type Props = {
  aggregate: MatterAggregate;
  operations: MatterOperation[];
  onUpdated: (value: MatterAggregate) => void | Promise<void>;
  onReload: () => void;
};

type Active = { kind: "decision"; decisionID?: string } | { kind: "response-add" } | { kind: "response-status"; responseID: string } | null;
type HistoryState = { state: "loading" | "live" | "unavailable"; page?: ResponseHistoryPage };

function operationsFor(operations: MatterOperation[], command: string, subresourceID?: string) {
  const matches = operations.filter((operation) => operation.command === command && (subresourceID === undefined || operation.subresource_id === subresourceID));
  return matches.find((operation) => operation.can_act) ?? matches[0];
}

function lines(value: string) {
  return value.split("\n").map((item) => item.trim()).filter(Boolean);
}

function statusLabel(value: string) {
  switch (value) {
    case "PROPOSED": return "Proposed";
    case "IN_REVIEW": return "In review";
    case "CONDITIONALLY_APPROVED": return "Approved with conditions";
    case "ACKNOWLEDGED": return "Acknowledged";
    default: return value.replaceAll("_", " ").toLowerCase().replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
  }
}

export function MatterDecisionResponsePanel({ aggregate, operations, onUpdated, onReload }: Props) {
  const initialDecision = operationsFor(operations, "matter.decision.record");
  const responseAdd = operationsFor(operations, "matter.response.add");
  const [active, setActive] = useState<Active>(null);
  const [decisionType, setDecisionType] = useState("");
  const [options, setOptions] = useState("");
  const [selectedOption, setSelectedOption] = useState("");
  const [target, setTarget] = useState("");
  const [rationale, setRationale] = useState("");
  const [purpose, setPurpose] = useState("");
  const [audience, setAudience] = useState("");
  const [manifest, setManifest] = useState("");
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [conflict, setConflict] = useState(false);
  const [responseHistories, setResponseHistories] = useState<Record<string, HistoryState>>({});

  async function openResponseHistory(responseID: string) {
    if (responseHistories[responseID]?.state === "loading" || responseHistories[responseID]?.state === "live") return;
    setResponseHistories((current) => ({ ...current, [responseID]: { state: "loading" } }));
    try {
      const page = await loadResponsePackageHistory(aggregate.matter.id, responseID, 20);
      setResponseHistories((current) => ({ ...current, [responseID]: { state: "live", page } }));
    } catch {
      setResponseHistories((current) => ({ ...current, [responseID]: { state: "unavailable" } }));
    }
  }

  function beginDecision(decisionID?: string) {
    const current = aggregate.decisions.find((decision) => decision.id === decisionID);
    const operation = operationsFor(operations, "matter.decision.record", decisionID);
    setDecisionType(current?.type ?? "");
    setOptions("");
    setSelectedOption(current?.selected_option ?? "");
    setTarget(operation?.allowed_targets?.[0] ?? "PROPOSED");
    setRationale("");
    setError("");
    setNotice("");
    setActive({ kind: "decision", decisionID });
  }

  function beginResponseAdd() {
    setPurpose(""); setAudience(""); setManifest(""); setError(""); setNotice(""); setActive({ kind: "response-add" });
  }

  function beginResponseStatus(responseID: string) {
    const operation = operationsFor(operations, "matter.response.transition", responseID);
    setTarget(operation?.allowed_targets?.[0] ?? ""); setRationale(""); setError(""); setNotice(""); setActive({ kind: "response-status", responseID });
  }

  function handleError(cause: unknown) {
    const isConflict = apiErrorKind(cause) === "conflict";
    setConflict(isConflict);
    setError(isConflict ? "This issue changed since you opened it. Your entries have been kept." : cause instanceof Error && cause.message ? cause.message : "The decision or response could not be recorded.");
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!active) return;
    setSaving(true); setError(""); setConflict(false); setNotice("");
    try {
      let updated: MatterAggregate;
      if (active.kind === "decision") {
        updated = await recordMatterDecision(aggregate.matter.id, aggregate.matter.version, {
          type: decisionType.trim(), status: target, options: lines(options),
          selectedOption: selectedOption.trim() || undefined, rationale: rationale.trim(),
        });
      } else if (active.kind === "response-add") {
        updated = await addResponsePackage(aggregate.matter.id, aggregate.matter.version, {
          purpose: purpose.trim(), audience: audience.trim(), manifest: { references: lines(manifest) },
        });
      } else {
        updated = await transitionResponsePackage(aggregate.matter.id, active.responseID, aggregate.matter.version, target, rationale.trim());
      }
      const completedKind = active.kind;
      await onUpdated(updated);
      if (completedKind === "response-add") setResponseHistories({});
      if (completedKind === "response-status" && active.kind === "response-status") setResponseHistories((current) => { const next = { ...current }; delete next[active.responseID]; return next; });
      setActive(null);
      setNotice(completedKind === "decision" ? "Decision recorded." : completedKind === "response-add" ? "Response package created." : "Response status updated.");
    } catch (cause) { handleError(cause); } finally { setSaving(false); }
  }

  const currentDecision = active?.kind === "decision" ? aggregate.decisions.find((decision) => decision.id === active.decisionID) : undefined;
  const decisionOperation = active?.kind === "decision" ? operationsFor(operations, "matter.decision.record", active.decisionID) : undefined;
  const responseOperation = active?.kind === "response-status" ? operationsFor(operations, "matter.response.transition", active.responseID) : undefined;
  const responsePresentation = matterResponsePresentation(responseOperation);
  const activeResponse = active?.kind === "response-status" ? aggregate.response_packages.find((response) => response.id === active.responseID) : undefined;
  const decisionNeedsSelection = ["APPROVED", "CONDITIONALLY_APPROVED", "REJECTED"].includes(target);

  return <article className="matter-record-panel matter-decision-response-panel" id="matter-operation-matter.decision.record">
    <div className="matter-record-section-heading"><div><span className="eyebrow">Decisions and responses</span><h2>Review and external handling</h2></div><div className="matter-panel-actions">
      {!active && initialDecision?.can_act && !initialDecision.subresource_id && <button className="secondary-button" type="button" onClick={() => beginDecision()}>{initialDecision.label}</button>}
      {!active && responseAdd?.can_act && <button className="secondary-button" type="button" onClick={beginResponseAdd}>{responseAdd.label}</button>}
    </div></div>

    <section className="matter-governance-section" aria-labelledby="matter-decision-history"><h3 id="matter-decision-history">Decision history</h3>
      {aggregate.decisions.length ? <div className="matter-governance-list">{aggregate.decisions.map((decision) => {
        const operation = operationsFor(operations, "matter.decision.record", decision.id);
        return <div className="matter-governance-row" key={decision.id}><div><strong>{decision.type}</strong><span>{statusLabel(decision.status)}</span><p>{decision.rationale}</p>{decision.selected_option && <small>Selected option: {decision.selected_option}</small>}{operation?.assigned_to && <small>Current responsibility: {operation.assigned_to.display_name}</small>}</div>{!active && operation?.can_act && <button className="secondary-button" type="button" aria-label={`Continue ${decision.type} decision`} onClick={() => beginDecision(decision.id)}>{operation.label}</button>}</div>;
      })}</div> : <p>No decision has been recorded for this issue.</p>}
    </section>

    <section className="matter-governance-section" aria-labelledby="matter-response-history"><h3 id="matter-response-history">Response packages</h3>
      {aggregate.response_packages.length ? <div className="matter-governance-list">{aggregate.response_packages.map((response) => {
        const operation = operationsFor(operations, "matter.response.transition", response.id);
        const history = responseHistories[response.id];
        const presentation = matterResponsePresentation(operation);
        return <div className="matter-governance-row" key={response.id}><div><strong>{response.purpose}</strong><span>{statusLabel(response.status)} · {response.audience}</span>{operation?.assigned_to && <small>Current responsibility: {operation.assigned_to.display_name}</small>}<details onToggle={(event) => { if (event.currentTarget.open) void openResponseHistory(response.id); }}><summary>View response status history</summary>{history?.state === "loading" && <p role="status">Loading response status history…</p>}{history?.state === "unavailable" && <p role="alert">Response status history could not be loaded. Close and reopen this history to retry.</p>}{history?.page && <><p>Showing {history.page.items.length} stored transitions{history.page.has_more ? "; older transitions are not shown" : ""}. Loaded {new Date(history.page.generated_at).toLocaleString()}.</p><ol>{history.page.items.map((item) => <li key={`${item.matter_version}-${item.occurred_at}`}><strong>{statusLabel(item.status)}</strong><span>{new Date(item.occurred_at).toLocaleString()} · {item.actor_label}</span><small>Issue version {item.matter_version}</small></li>)}</ol></>}</details></div>{!active && operation?.can_act && <Button variant="secondary" aria-label={`${presentation.action} for ${response.purpose}`} onPress={() => beginResponseStatus(response.id)}>{presentation.action}</Button>}</div>;
      })}</div> : <p>No response package has been prepared for this issue.</p>}
    </section>

    {active && active.kind !== "response-status" && <form className="matter-operation-form" onSubmit={submit}>
      {active.kind === "decision" && <>
        <label><span>Decision type</span><input value={decisionType} onChange={(event) => setDecisionType(event.target.value)} readOnly={Boolean(currentDecision)} required/></label>
        {(decisionOperation?.allowed_targets?.length ?? 0) > 1 && <label><span>Decision outcome</span><select value={target} onChange={(event) => setTarget(event.target.value)}>{decisionOperation?.allowed_targets?.map((value) => <option key={value} value={value}>{statusLabel(value)}</option>)}</select></label>}
        {!currentDecision && <label className="wide"><span>Options considered</span><textarea rows={3} value={options} onChange={(event) => setOptions(event.target.value)} placeholder="One option per line" required/></label>}
        <label className="wide"><span>{decisionNeedsSelection ? "Selected option" : "Recommended option"}</span><input value={selectedOption} onChange={(event) => setSelectedOption(event.target.value)} required={decisionNeedsSelection || !currentDecision}/></label>
        <label className="wide"><span>Decision rationale</span><textarea rows={3} value={rationale} onChange={(event) => setRationale(event.target.value)} required/></label>
      </>}
      {active.kind === "response-add" && <>
        <label className="wide"><span>Response purpose</span><input value={purpose} onChange={(event) => setPurpose(event.target.value)} required/></label>
        <label><span>Audience</span><input value={audience} onChange={(event) => setAudience(event.target.value)} required/></label>
        <label className="wide"><span>Included records</span><textarea rows={3} value={manifest} onChange={(event) => setManifest(event.target.value)} placeholder="One artifact or record reference per line" required/></label>
      </>}
      {error && <div className="matter-form-error wide" role="alert"><span>{error}</span>{conflict && <button className="secondary-button" type="button" onClick={onReload}>Reload current issue</button>}</div>}
      <div className="matter-form-actions wide"><button className="primary-button" type="submit" disabled={saving || (active.kind === "decision" && !target)}>{saving ? "Saving…" : active.kind === "decision" ? "Record decision" : "Create response package"}</button><button className="text-button" type="button" onClick={() => setActive(null)}>Cancel</button></div>
    </form>}
    {active?.kind === "response-status" && <FocusedSheet label={responsePresentation.sheet} closeLabel={`Close ${responsePresentation.sheet.toLowerCase()}`} onClose={() => setActive(null)}>
      <div className="cs-sheet-heading"><span className="eyebrow">{responseOperation?.responsibility ? statusLabel(responseOperation.responsibility) : "Response handling"}</span><h2>{responsePresentation.sheet}</h2><p>Review the exact response package and record only the action assigned to your current responsibility.</p></div>
      <form className="cs-sheet-form" onSubmit={submit}>
        <dl className="cs-sheet-facts"><div><dt>Response</dt><dd>{activeResponse?.purpose ?? "Selected response"}</dd></div><div><dt>Current state</dt><dd>{statusLabel(activeResponse?.status ?? "UNKNOWN")}</dd></div><div><dt>Audience</dt><dd>{activeResponse?.audience ?? "Not recorded"}</dd></div><div><dt>Next state</dt><dd>{target ? statusLabel(target) : "No permitted state available"}</dd></div></dl>
        {(responseOperation?.allowed_targets?.length ?? 0) > 1 && <SelectField label="Next response status" value={target || undefined} placeholder="Select the permitted response state" allowsEmpty={false} isRequired options={(responseOperation?.allowed_targets ?? []).map((value) => ({ id: value, label: statusLabel(value) }))} onChange={(value) => setTarget(value ?? "")}/>}
        <Notice tone="info">{responsePresentation.consequence}</Notice>
        <TextArea label={responsePresentation.rationaleLabel} value={rationale} onChange={setRationale} rows={4} isRequired description="Record the evidence, review basis or reference that supports this step."/>
        {error && <Notice tone="error"><span>{error}</span>{conflict && <Button variant="secondary" onPress={onReload}>Reload current issue</Button>}</Notice>}
        <div className="cs-sheet-actions"><Button type="button" variant="quiet" isDisabled={saving} onPress={() => setActive(null)}>Cancel</Button><Button type="submit" variant="primary" isDisabled={!target || !rationale.trim()} isLoading={saving}>{responsePresentation.submit}</Button></div>
      </form>
    </FocusedSheet>}
    {notice && <p className="inline-success" role="status">{notice}</p>}
  </article>;
}
