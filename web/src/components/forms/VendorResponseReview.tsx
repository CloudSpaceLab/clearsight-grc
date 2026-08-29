import { useEffect, useMemo, useState } from "react";
import type { VendorRelationshipAggregate } from "../../vendorTypes";
import type { ApplyVendorAssessmentResponseInput, VendorAssessmentApplicationResult, VendorAssessmentFieldApplicationDecision, VendorAssessmentReviewAnswer, VendorAssessmentReviewView } from "../../vendorAssessmentTypes";

type Props = {
  relationship: VendorRelationshipAggregate;
  review: VendorAssessmentReviewView;
  onApply: (assessmentID: string, revisionID: string, input: ApplyVendorAssessmentResponseInput) => Promise<VendorAssessmentApplicationResult>;
  onApplied?: (result: VendorAssessmentApplicationResult) => void;
};

type DraftDecision = { decision: "" | "ACCEPT" | "REJECT"; rationale: string };

export function VendorResponseReview({ relationship, review, onApply, onApplied }: Props) {
  const governed = useMemo(() => review.answers.filter((answer) => answer.visibility === "VISIBLE" && answer.baseline), [review.answers]);
  const [decisions, setDecisions] = useState<Record<string, DraftDecision>>({});
  const [receipt, setReceipt] = useState<VendorAssessmentApplicationResult["receipt"] | undefined>(review.application_receipt);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const conflicts = governed.filter((answer) => identityTarget(answer) && answer.baseline!.record_version !== relationship.vendor.version);
  const unresolved = governed.filter((answer) => {
    const draft = decisions[answer.field_id];
    if (!draft?.decision || !draft.rationale.trim()) return true;
    return draft.decision === "ACCEPT" && documentTarget(answer) && !review.documents.some((document) => document.field_id === answer.field_id && document.status === "VALIDATED");
  });
  const ready = governed.length > 0 && unresolved.length === 0 && conflicts.length === 0 && Boolean(review.response) && !receipt;

  useEffect(() => setReceipt(review.application_receipt), [review.application_receipt]);

  function update(fieldID: string, change: Partial<DraftDecision>) {
    setDecisions((current) => ({ ...current, [fieldID]: { decision: current[fieldID]?.decision ?? "", rationale: current[fieldID]?.rationale ?? "", ...change } }));
  }

  async function apply() {
    if (!ready || !review.response || busy) return;
    setBusy(true); setError("");
    const input: ApplyVendorAssessmentResponseInput = {
      expected_assessment_version: review.assessment.version,
      expected_submission_revision: 1,
      decisions: governed.map((answer): VendorAssessmentFieldApplicationDecision => ({ field_id: answer.field_id, decision: decisions[answer.field_id]!.decision as "ACCEPT" | "REJECT", rationale: decisions[answer.field_id]!.rationale.trim() })),
    };
    try {
      const result = await onApply(review.assessment.id, review.response.submission_id, input);
      setReceipt(result.receipt);
      onApplied?.(result);
    } catch {
      setError("These held records changed or the response could not be applied. Reload the vendor and response before reviewing the decisions again.");
    } finally { setBusy(false); }
  }

  if (!governed.length) return null;
  if (receipt) return <section className="vdd-application-receipt" aria-labelledby="response-application-title"><span className="eyebrow">Application receipt</span><h3 id="response-application-title">Reviewed changes recorded</h3><p>{receipt.accepted_field_ids.length} accepted and {receipt.rejected_field_ids.length} rejected field decisions were recorded on {formatDateTime(receipt.applied_at)}.</p><dl><div><dt>Reviewer</dt><dd>{receipt.actor_principal_id}</dd></div><div><dt>Vendor version</dt><dd>{receipt.prior_vendor_version} → {receipt.result_vendor_version}</dd></div><div><dt>Assessment version</dt><dd>{receipt.result_assessment_version}</dd></div></dl></section>;

  return <section className="vdd-response-application" aria-labelledby="response-application-title">
    <div className="vdd-review-group-heading"><div><span className="eyebrow">Held-record review</span><h3 id="response-application-title">Decide which vendor changes to apply</h3><p>Submission did not update the vendor record. Compare the requested value, submitted response and current held version before deciding each field.</p></div></div>
    {conflicts.length > 0 && <div className="vdd-alert" role="alert"><strong>{conflicts.length} held {conflicts.length === 1 ? "record has" : "records have"} changed</strong><span>Reload the vendor and response before applying any decision.</span></div>}
    <div className="vdd-comparison-list">{governed.map((answer) => {
      const draft = decisions[answer.field_id] ?? { decision: "", rationale: "" };
      const conflict = conflicts.includes(answer);
      const documentBlocked = draft.decision === "ACCEPT" && documentTarget(answer) && !review.documents.some((document) => document.field_id === answer.field_id && document.status === "VALIDATED");
      return <article key={answer.field_id} aria-label={`Apply decision: ${answer.label}`} className={conflict ? "has-conflict" : undefined}>
        <div className="vdd-comparison-heading"><h4>{answer.label}</h4>{conflict && <span>Held version changed</span>}</div>
        <dl className="vdd-value-comparison"><div><dt>Requested held value</dt><dd>{answer.baseline!.display_value || "Not recorded"}<small>{answer.baseline!.source_label} · version {answer.baseline!.record_version}</small></dd></div><div><dt>Submitted response</dt><dd>{answerValue(answer)}<small>{assuranceLabel(answer)}</small></dd></div><div><dt>Current bank record</dt><dd>{currentHeldValue(answer, relationship)}<small>{identityTarget(answer) ? `Vendor version ${relationship.vendor.version}` : `Requested version ${answer.baseline!.record_version}`}</small></dd></div></dl>
        <fieldset disabled={conflict || busy}><legend>Decision</legend><label><input type="radio" name={`decision-${answer.field_id}`} checked={draft.decision === "ACCEPT"} onChange={() => update(answer.field_id, { decision: "ACCEPT" })}/>Accept submitted value</label><label><input type="radio" name={`decision-${answer.field_id}`} checked={draft.decision === "REJECT"} onChange={() => update(answer.field_id, { decision: "REJECT" })}/>Keep current value</label></fieldset>
        <label className="vdd-field"><span>Decision rationale</span><textarea rows={2} maxLength={2000} value={draft.rationale} disabled={conflict || busy} onChange={(event) => update(answer.field_id, { rationale: event.target.value })}/></label>
        {documentBlocked && <p className="vdd-inline-warning">Validate the replacement document before accepting it.</p>}
      </article>;
    })}</div>
    {error && <p className="vdd-error" role="alert">{error}</p>}
    <div className="vdd-panel-actions"><button type="button" className="primary-button" disabled={!ready || busy} onClick={() => void apply()}>{busy ? "Applying decisions…" : "Apply reviewed changes"}</button>{!ready && !conflicts.length && <small>Choose accept or reject and record a rationale for every held-record field.</small>}</div>
  </section>;
}

function identityTarget(answer: VendorAssessmentReviewAnswer) { return answer.baseline?.target_key.startsWith("VENDOR.IDENTITY."); }
function documentTarget(answer: VendorAssessmentReviewAnswer) { return answer.baseline?.target_key.startsWith("VENDOR.DOCUMENT."); }
function answerValue(answer: VendorAssessmentReviewAnswer) { if (answer.value?.document) return [answer.value.document.document_type, answer.value.document.reference, answer.value.document.expires_on ? `expires ${answer.value.document.expires_on}` : ""].filter(Boolean).join(" · "); return answer.value?.text ?? answer.value?.values?.join(", ") ?? "Not provided"; }
function assuranceLabel(answer: VendorAssessmentReviewAnswer) { const origin = answer.provenance?.origin?.replaceAll("_", " ").toLowerCase(); return origin ? `${origin[0]?.toUpperCase()}${origin.slice(1)}` : "Respondent supplied"; }
function currentHeldValue(answer: VendorAssessmentReviewAnswer, relationship: VendorRelationshipAggregate) { switch (answer.baseline?.target_key) { case "VENDOR.IDENTITY.LEGAL_NAME": return relationship.vendor.legal_name; case "VENDOR.IDENTITY.TRADING_NAME": return relationship.vendor.trading_name || "Not recorded"; case "VENDOR.IDENTITY.REGISTRATION_REFERENCE": return relationship.vendor.registration_ref || "Not recorded"; case "VENDOR.IDENTITY.JURISDICTION": return relationship.vendor.jurisdiction || "Not recorded"; case "VENDOR.IDENTITY.REGISTERED_ADDRESS": return relationship.vendor.registered_address || "Not recorded"; case "VENDOR.IDENTITY.WEBSITE_DOMAIN": return relationship.vendor.website_domain || "Not recorded"; default: return answer.baseline?.display_value || "Not recorded"; } }
function formatDateTime(value: string) { const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed)) : value; }
