import { useState } from "react";
import { loadVendorFormSummaries, requestVendorForms, type VendorFormSummary, type VendorRequestInput, type VendorRequestReceipt, type VendorRequestSettings } from "../vendorFormsApi";
import { DistributionComposer } from "./forms/DistributionComposer";
import { Button, FocusedSheet, Notice, TextField } from "./ui";
import "./vendor-forms.css";

export type VendorRequestTarget = { relationshipID: string; vendorName: string; serviceName: string };
export function VendorFormRequest({ targets, onClose, onUpdated }: { targets: VendorRequestTarget[]; onClose: () => void; onUpdated: () => void }) {
  const [recipients, setRecipients] = useState<Record<string, string>>({});
  const [input, setInput] = useState<VendorRequestInput>();
  const [summaries, setSummaries] = useState<VendorFormSummary[]>([]);
  const [summaryError, setSummaryError] = useState(false);
  const [receipt, setReceipt] = useState<VendorRequestReceipt>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();
  const [attempted, setAttempted] = useState(false);
  const [phase, setPhase] = useState<"setup" | "preview" | "receipt">("setup");
  const ready = targets.length > 0 && targets.length <= 50 && targets.every((target) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(recipients[target.relationshipID]?.trim() ?? ""));

  async function preview(settings: VendorRequestSettings) {
    const request: VendorRequestInput = { ...settings, batch_id: crypto.randomUUID(), targets: targets.map((target) => ({ relationship_id: target.relationshipID, recipient: { type: "EXTERNAL_AUDIENCE", role: "TO", address: recipients[target.relationshipID]!.trim().toLowerCase() } })) };
    setInput(request); setReceipt(undefined); setAttempted(false); setError(undefined); setSummaryError(false); setSummaries([]);
    try { const page = await loadVendorFormSummaries(targets.map((target) => target.relationshipID), settings.form_template_id); setSummaries(page.items); }
    catch { setSummaryError(true); }
    setPhase("preview");
  }
  async function dispatch() {
    if (!input || busy) return;
    const unfinished = input.targets.filter((target) => !receipt?.items.some((item) => item.relationship_id === target.relationship_id && item.status === "CREATED"));
    if (!unfinished.length) return;
    setBusy(true); setError(undefined); setAttempted(true);
    try {
      const result = await requestVendorForms({ ...input, targets: unfinished });
      setReceipt((previous) => ({ batch_id: input.batch_id, items: [...(previous?.items ?? []).filter((item) => !result.items.some((next) => next.relationship_id === item.relationship_id)), ...result.items] }));
      setPhase("receipt");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Vendor request receipts could not be loaded. Retry to check this request attempt."); }
    finally { setBusy(false); }
    onUpdated();
  }
  const pending = input?.targets.filter((target) => !receipt?.items.some((item) => item.relationship_id === target.relationship_id && item.status === "CREATED")) ?? [];
  return <FocusedSheet label="Request vendor forms" size="wide" onClose={onClose}>
    <div className="vendor-form-request">
      <div hidden={phase !== "setup"}><DistributionComposer scopedDelivery={{ label: `${targets.length} selected vendor ${targets.length === 1 ? "service" : "services"}`, ready, submitLabel: "Preview vendor requests", onSubmit: preview, recipients: <section className="vendor-form-request__recipients" aria-label="Vendor recipients"><h3>Who must respond</h3><p>Each service receives a separate request. Enter the permitted recipient for each vendor.</p>{targets.map((target) => <div key={target.relationshipID}><strong>{target.vendorName} · {target.serviceName}</strong><TextField label={`Email for ${target.vendorName} · ${target.serviceName}`} type="email" value={recipients[target.relationshipID] ?? ""} onChange={(address) => setRecipients((current) => ({ ...current, [target.relationshipID]: address }))}/></div>)}</section> }}/></div>
      {phase !== "setup" && input && <>
        <header><h2>{phase === "preview" ? "Review vendor requests" : "Vendor request receipts"}</h2><p>{input.title} · Form revision {input.form_template_version}</p><p>{input.purpose}</p></header>
        <dl className="cs-sheet-facts"><div><dt>Response deadline</dt><dd>{new Date(input.deadline).toLocaleString()}</dd></div><div><dt>Access expires</dt><dd>{new Date(input.route_expires_at).toLocaleString()}</dd></div></dl>
        {summaryError && <Notice tone="warning">Existing requests for this form could not be checked. Review the vendor’s current requests before sending another.</Notice>}
        {error && <Notice tone="error">{error} The same request attempt will be reused when you retry.</Notice>}
        <ul className="vendor-form-request__targets">{targets.map((target) => {
          const result = receipt?.items.find((item) => item.relationship_id === target.relationshipID);
          const summary = summaries.find((item) => item.relationship_id === target.relationshipID);
          return <li key={target.relationshipID}><strong>{target.vendorName} · {target.serviceName}</strong><span>Recipient: {input.targets.find((item) => item.relationship_id === target.relationshipID)?.recipient.address}</span>
            {phase === "preview" && <span>{summary ? `${summary.outstanding_forms} outstanding requests for this form` : "Existing request count unavailable"}</span>}
            {summary && summary.outstanding_forms > 0 && phase === "preview" && <span>Review the outstanding requests; their purpose, revision or deadline may differ.</span>}
            {result?.status === "CREATED" && <span>{result.distribution_state ? `Request saved · ${result.distribution_state.toLowerCase().replaceAll("_", " ")}.` : "Request created."} Check delivery status in Forms and responses.</span>}
            {result?.status === "FAILED" && <span role="alert">{result.error || "Request could not be created."}</span>}
            {result?.status === "PREPARED" && <span role="alert">Request saved; delivery setup needs retry. {result.error}</span>}
            {phase === "receipt" && !result && <span>No receipt returned. Retry this target to check its request.</span>}
          </li>;
        })}</ul>
        <Notice tone="info">Creating a request does not mean the email was delivered, the vendor responded or the evidence was accepted.</Notice>
        <div className="vendor-form-request__actions">{!attempted && <Button onPress={() => setPhase("setup")}>Edit request details</Button>}{pending.length > 0 ? <Button variant="primary" isLoading={busy} onPress={() => void dispatch()}>{phase === "receipt" ? "Retry failed vendor requests" : attempted ? "Retry vendor requests" : "Create and dispatch vendor requests"}</Button> : <Button variant="primary" onPress={onClose}>Return to vendor forms</Button>}</div>
      </>}
    </div>
  </FocusedSheet>;
}
