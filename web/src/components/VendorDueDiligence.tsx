import { useEffect, useLayoutEffect, useMemo, useState } from "react";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import type {
  CompleteVendorAssessmentInput,
  ReissueVendorAssessmentRequestInput,
  SendVendorAssessmentRequestInput,
  StartVendorAssessmentInput,
  VendorAssessment,
  VendorAssessmentClarificationInput,
  VendorAssessmentConclusion,
  VendorAssessmentDocument,
  VendorAssessmentFormOption,
  VendorAssessmentReviewView,
  VendorAssessmentSendOutcome,
} from "../vendorAssessmentTypes";
import "./vendor-due-diligence.css";

type ViewState = "live" | "loading" | "unavailable";

type Props = {
  relationship: VendorRelationshipAggregate;
  assessment?: VendorAssessment | null;
  form?: VendorAssessmentFormOption;
  review?: VendorAssessmentReviewView;
  reviewState?: ViewState;
  requestOutcome?: VendorAssessmentSendOutcome;
  requestOutcomeKind?: "initial" | "replacement";
  viewState?: ViewState;
  defaultReviewDueDate?: string;
  setupFailure?: string;
  sourceStatus?: { state: "CURRENT" | "STALE" | "UNAVAILABLE"; detail: string };
  clarificationFields?: { id: string; label: string }[];
  onStart?: (input: StartVendorAssessmentInput) => Promise<VendorAssessment | void> | VendorAssessment | void;
  onRetrySetup?: (assessmentID: string) => Promise<void> | void;
  onRefresh?: () => Promise<void> | void;
  onRefreshReview?: (assessmentID: string) => Promise<void> | void;
  onSend?: (input: SendVendorAssessmentRequestInput) => Promise<VendorAssessmentSendOutcome>;
  onReissue?: (input: ReissueVendorAssessmentRequestInput) => Promise<VendorAssessmentSendOutcome>;
  onOpenRequest?: (requestID: string) => void;
  onStartReview?: (assessmentID: string, expectedVersion: number) => Promise<VendorAssessment | void> | VendorAssessment | void;
  onRequestClarification?: (assessmentID: string, input: VendorAssessmentClarificationInput) => Promise<VendorAssessment | void> | VendorAssessment | void;
  onCreateDeficiency?: (assessmentID: string) => void;
  onReviewDocument?: (assessmentID: string, document: VendorAssessmentDocument, decision: "VALIDATE" | "REJECT") => void;
  onComplete?: (assessmentID: string, input: CompleteVendorAssessmentInput) => Promise<VendorAssessment | void> | VendorAssessment | void;
};

type ActionPanel = "start" | "send" | "reissue" | "clarification" | "conclusion" | null;

const conclusionOptions: { value: VendorAssessmentConclusion; label: string }[] = [
  { value: "SATISFACTORY", label: "Satisfactory" },
  { value: "SATISFACTORY_WITH_CONDITIONS", label: "Satisfactory with conditions" },
  { value: "UNSATISFACTORY", label: "Unsatisfactory" },
  { value: "INCONCLUSIVE", label: "Inconclusive" },
];

export function VendorDueDiligence({
  relationship,
  assessment,
  form,
  review,
  reviewState = "live",
  requestOutcome,
  requestOutcomeKind = "initial",
  viewState = "live",
  defaultReviewDueDate = "",
  setupFailure,
  sourceStatus,
  clarificationFields = [],
  onStart,
  onRetrySetup,
  onRefresh,
  onRefreshReview,
  onSend,
  onReissue,
  onOpenRequest,
  onStartReview,
  onRequestClarification,
  onCreateDeficiency,
  onReviewDocument,
  onComplete,
}: Props) {
  const [panel, setPanel] = useState<ActionPanel>(null);
  const [localAssessment, setLocalAssessment] = useState<VendorAssessment | null | undefined>(assessment);
  const [localOutcome, setLocalOutcome] = useState<VendorAssessmentSendOutcome>();
  const [localOutcomeKind, setLocalOutcomeKind] = useState<"initial" | "replacement">("initial");
  const [reviewDueDate, setReviewDueDate] = useState(defaultReviewDueDate);
  const [recipient, setRecipient] = useState("");
  const [responseDueDate, setResponseDueDate] = useState("");
  const [invitationMinutes, setInvitationMinutes] = useState(1440);
  const [selectedClarificationFields, setSelectedClarificationFields] = useState<string[]>([]);
  const [clarificationMessage, setClarificationMessage] = useState("");
  const [clarificationDueDate, setClarificationDueDate] = useState("");
  const [conclusion, setConclusion] = useState<VendorAssessmentConclusion>("SATISFACTORY_WITH_CONDITIONS");
  const [rationale, setRationale] = useState("");
  const [uncertainty, setUncertainty] = useState("");
  const [nextReviewDate, setNextReviewDate] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  useLayoutEffect(() => {
    setLocalAssessment(assessment);
    setLocalOutcome(undefined);
    setLocalOutcomeKind("initial");
    setPanel(null);
    setError("");
    setNotice("");
  }, [assessment?.id]);

  useEffect(() => {
    setLocalAssessment(assessment);
    setPanel(null);
  }, [assessment]);

  const effectiveOutcome = localOutcome ?? requestOutcome;
  const effectiveOutcomeKind = localOutcome ? localOutcomeKind : requestOutcomeKind;
  const effectiveAssessment = effectiveOutcome?.assessment ?? localAssessment;
  const status = effectiveAssessment?.status;
  const requestID = effectiveAssessment?.current_request_id ?? effectiveOutcome?.request.id;
  const dueDate = effectiveAssessment?.review_due_at;
  const minimumFutureDate = tomorrowUTC();

  const statusCopy = useMemo(() => assessmentStatusCopy(effectiveAssessment, setupFailure), [effectiveAssessment, setupFailure]);

  function openPanel(next: ActionPanel) {
    setPanel(next);
    setError("");
    setNotice("");
  }

  async function startAssessment(event: React.FormEvent) {
    event.preventDefault();
    if (!form || !onStart || !reviewDueDate || Date.parse(endOfDay(reviewDueDate)) <= Date.now()) {
      setError("Choose an active collection form and review due date before starting due diligence.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const result = await onStart({
        relationship_version: relationship.relationship.version,
        form_template_id: form.id,
        form_template_version: form.version,
        review_due_at: endOfDay(reviewDueDate),
      });
      if (result) setLocalAssessment(result);
      setPanel(null);
      setNotice("Due diligence started. The review record is being prepared.");
    } catch {
      setError("Due diligence could not be started. Check the current relationship and form versions, then try again.");
    } finally {
      setBusy(false);
    }
  }

  async function sendRequest(event: React.FormEvent) {
    event.preventDefault();
    const deadline = responseDueDate ? Date.parse(endOfDay(responseDueDate)) : Number.NaN;
    if (!effectiveAssessment || !onSend || !validEmail(recipient) || !Number.isFinite(deadline) || deadline <= Date.now() || deadline > Date.parse(effectiveAssessment.review_due_at)) {
      setError("Enter a valid vendor contact email and a future response date no later than the assessment due date.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const outcome = await onSend({
        expected_version: effectiveAssessment.version,
        audience: recipient.trim().toLowerCase(),
        deadline: endOfDay(responseDueDate),
        invitation_ttl_minutes: invitationMinutes,
      });
      setRecipient("");
      setLocalOutcome(outcome);
      setLocalOutcomeKind("initial");
      setPanel(null);
      if (outcome.state === "DELIVERED") setNotice(`The request was sent. The response is due ${formatDate(outcome.request.deadline)}.`);
    } catch {
      setRecipient("");
      setPanel(null);
      setError("The request was not sent. Re-enter the vendor contact email before trying again.");
    } finally {
      setBusy(false);
    }
  }

  async function reissueRequest(event: React.FormEvent) {
    event.preventDefault();
    if (!effectiveAssessment || !onReissue || !validEmail(recipient)) {
      setError("Enter a valid vendor contact email before sending the replacement link.");
      setRecipient("");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const outcome = await onReissue({
        expected_version: effectiveAssessment.version,
        audience: recipient.trim().toLowerCase(),
        invitation_ttl_minutes: invitationMinutes,
      });
      setLocalOutcome(outcome);
      setLocalOutcomeKind("replacement");
      setPanel(null);
      if (outcome.state === "DELIVERED") setNotice("Replacement link sent. Previous access to this request has ended.");
    } catch {
      setPanel(null);
      setError("The replacement link was not sent. Re-enter the vendor contact email before trying again.");
    } finally {
      setRecipient("");
      setBusy(false);
    }
  }

  async function startReview() {
    if (!effectiveAssessment || !onStartReview) return;
    await run(async () => {
      const result = await onStartReview(effectiveAssessment.id, effectiveAssessment.version);
      if (result) setLocalAssessment(result);
      setNotice("Review started. Record document decisions, findings and the assessment conclusion.");
    }, "The vendor response could not be opened for review. Reload the assessment and try again.");
  }

  async function submitClarification(event: React.FormEvent) {
    event.preventDefault();
    if (!effectiveAssessment || !onRequestClarification || selectedClarificationFields.length === 0 || !clarificationMessage.trim() || !clarificationDueDate) {
      setError("Select the missing fields, explain what the vendor must provide and set a response due date.");
      return;
    }
    await run(async () => {
      const result = await onRequestClarification(effectiveAssessment.id, {
        expected_version: effectiveAssessment.version,
        request_fields: selectedClarificationFields,
        message: clarificationMessage.trim(),
        deadline: endOfDay(clarificationDueDate),
      });
      if (result) setLocalAssessment(result);
      setClarificationMessage("");
      setClarificationDueDate("");
      setSelectedClarificationFields([]);
      setPanel(null);
      setNotice("Clarification requested. The assessment remains open until the vendor responds.");
    }, "The clarification request could not be created. Your selected fields remain available; try again.");
  }

  async function submitConclusion(event: React.FormEvent) {
    event.preventDefault();
    if (!effectiveAssessment || !onComplete || !rationale.trim() || (nextReviewDate && Date.parse(startOfDay(nextReviewDate)) <= Date.now())) {
      setError("Record the assessment basis before completing the review.");
      return;
    }
    await run(async () => {
      const input: CompleteVendorAssessmentInput = {
        expected_version: effectiveAssessment.version,
        conclusion,
        rationale: rationale.trim(),
        uncertainty: uncertainty.trim() || undefined,
        next_review_recommended_at: nextReviewDate ? startOfDay(nextReviewDate) : undefined,
      };
      const result = await onComplete(effectiveAssessment.id, input);
      if (result) setLocalAssessment(result);
      setPanel(null);
      setNotice("Assessment conclusion recorded. The vendor relationship status has not changed.");
    }, "The assessment conclusion could not be recorded. Your entries remain on this screen; reload the assessment before trying again.");
  }

  async function copyCaptureLink() {
    const captureURL = effectiveOutcome?.capture_url;
    if (!captureURL) return;
    try {
      await navigator.clipboard.writeText(captureURL);
      setNotice(effectiveOutcomeKind === "replacement" ? "Replacement link copied." : "Secure link copied.");
    } catch {
      setError("The secure link could not be copied. Use the request status to retry delivery.");
    }
  }

  async function run(operation: () => Promise<void>, failure: string) {
    setBusy(true);
    setError("");
    try {
      await operation();
    } catch {
      setError(failure);
    } finally {
      setBusy(false);
    }
  }

  if (viewState === "loading") {
    return <section className="vdd-workspace" aria-label="Due diligence"><div className="vdd-state" aria-live="polite" aria-busy="true">Loading due diligence for {relationship.relationship.service_name}…</div></section>;
  }

  if (viewState === "unavailable") {
    return <section className="vdd-workspace" aria-label="Due diligence"><div className="vdd-state vdd-state-error" role="alert"><h2>Due diligence is unavailable</h2><p>The current assessment for {relationship.relationship.service_name} could not be loaded. Try again before starting or changing the review.</p>{onRefresh && <button type="button" className="secondary-button" onClick={() => void onRefresh()}>Try again</button>}</div></section>;
  }

  return <section className="vdd-workspace" aria-labelledby="vdd-title">
    <header className="vdd-header">
      <div><span className="eyebrow">{relationship.relationship.service_name}</span><h2 id="vdd-title">Due diligence</h2><p>{statusCopy.description}</p></div>
      <span className={`vdd-status vdd-status-${statusCopy.tone}`}>{statusCopy.label}</span>
    </header>

    <div className="vdd-scope" aria-label="Assessment scope">
      <div><span>Vendor</span><strong>{relationship.vendor.legal_name}</strong></div>
      <div><span>Service</span><strong>{relationship.relationship.service_name}</strong></div>
      <div><span>Accountable owner</span><strong>{relationship.relationship.business_owner_principal_id}</strong></div>
      <div><span>Review due</span><strong>{formatDate(dueDate)}</strong></div>
    </div>

    {sourceStatus && sourceStatus.state !== "CURRENT" && <div className="vdd-source-warning" role="status"><strong>{sourceStatus.state === "STALE" ? "Vendor source is out of date" : "Vendor source is unavailable"}</strong><span>{sourceStatus.detail}</span></div>}
    {setupFailure && status === "SETUP_PENDING" && <div className="vdd-alert" role="alert"><strong>Review setup needs attention</strong><span>{setupFailure}</span></div>}
    {effectiveOutcome?.state === "REQUEST_READY_INVITATION_NOT_ISSUED" && <div className="vdd-alert" role="alert"><strong>The request is ready, but secure access was not issued</strong><span>{effectiveOutcome.recovery ?? "Retry invitation creation for this request."}</span></div>}
    {effectiveOutcome?.state === "LINK_CREATED_EMAIL_NOT_SENT" && <div className="vdd-alert" role="alert"><strong>Email delivery did not complete</strong><span>{effectiveOutcome.recovery ?? "Copy the secure link or review delivery status."}</span></div>}
    {notice && <p className="vdd-notice" role="status">{notice}</p>}
    {error && <p className="vdd-error" role="alert">{error}</p>}

    {effectiveAssessment && needsReviewView(status) && reviewState === "loading" && <div className="vdd-review-state" aria-live="polite" aria-busy="true">Loading the submitted response and supporting documents…</div>}
    {effectiveAssessment && needsReviewView(status) && reviewState === "unavailable" && <div className="vdd-alert" role="alert"><strong>Vendor response is unavailable</strong><span>The submitted answers and documents could not be loaded. Reload them before starting or completing the review.</span>{onRefreshReview && <button type="button" className="secondary-button" onClick={() => void onRefreshReview(effectiveAssessment.id)}>Reload vendor response</button>}</div>}

    {reviewState === "live" && review && <ReviewSummary review={review} assessment={effectiveAssessment} onReviewDocument={onReviewDocument} onCreateDeficiency={onCreateDeficiency}/>}

    {panel === "start" && <StartPanel form={form} reviewDueDate={reviewDueDate} minimumDate={minimumFutureDate} busy={busy} onReviewDueDate={setReviewDueDate} onCancel={() => setPanel(null)} onSubmit={startAssessment}/>}
    {panel === "send" && effectiveAssessment && <SendPanel recipient={recipient} responseDueDate={responseDueDate} invitationMinutes={invitationMinutes} minimumDate={minimumFutureDate} reviewDueAt={effectiveAssessment.review_due_at} busy={busy} onRecipient={setRecipient} onResponseDueDate={setResponseDueDate} onInvitationMinutes={setInvitationMinutes} onCancel={() => { setRecipient(""); setPanel(null); }} onSubmit={sendRequest}/>}
    {panel === "reissue" && effectiveAssessment && <ReissuePanel recipient={recipient} invitationMinutes={invitationMinutes} busy={busy} onRecipient={setRecipient} onInvitationMinutes={setInvitationMinutes} onCancel={() => { setRecipient(""); setPanel(null); }} onSubmit={reissueRequest}/>}
    {panel === "clarification" && <ClarificationPanel fields={clarificationFields} selected={selectedClarificationFields} message={clarificationMessage} dueDate={clarificationDueDate} busy={busy} onSelected={setSelectedClarificationFields} onMessage={setClarificationMessage} onDueDate={setClarificationDueDate} onCancel={() => setPanel(null)} onSubmit={submitClarification}/>}
    {panel === "conclusion" && <ConclusionPanel conclusion={conclusion} rationale={rationale} uncertainty={uncertainty} nextReviewDate={nextReviewDate} minimumDate={minimumFutureDate} busy={busy} onConclusion={setConclusion} onRationale={setRationale} onUncertainty={setUncertainty} onNextReviewDate={setNextReviewDate} onCancel={() => setPanel(null)} onSubmit={submitConclusion}/>}

    {!panel && <div className="vdd-actions">
      {status === "COLLECTING" ? <><button type="button" className="primary-button" onClick={() => requestID && onOpenRequest?.(requestID)} disabled={!requestID || !onOpenRequest}>Review request status</button>{effectiveOutcome?.state === "LINK_CREATED_EMAIL_NOT_SENT" && effectiveOutcome.capture_url ? <button type="button" className="secondary-button" onClick={() => void copyCaptureLink()}>{effectiveOutcomeKind === "replacement" ? "Copy replacement link" : "Copy secure link"}</button> : <button type="button" className="secondary-button" onClick={() => openPanel("reissue")} disabled={!onReissue}>{effectiveOutcomeKind === "replacement" && effectiveOutcome?.state === "REQUEST_READY_INVITATION_NOT_ISSUED" ? "Retry replacement link" : "Send replacement link"}</button>}</>
        : effectiveOutcome?.state === "LINK_CREATED_EMAIL_NOT_SENT" && effectiveOutcome.capture_url ? <button type="button" className="primary-button" onClick={() => void copyCaptureLink()}>Copy secure link</button>
        : effectiveOutcome?.state === "REQUEST_READY_INVITATION_NOT_ISSUED" ? <button type="button" className="primary-button" onClick={() => openPanel("send")} disabled={!onSend}>Retry invitation creation</button>
          : !effectiveAssessment ? <button type="button" className="primary-button" onClick={() => openPanel("start")} disabled={!form || !onStart}>Start due diligence</button>
            : status === "SETUP_PENDING" ? setupFailure ? <button type="button" className="primary-button" onClick={() => onRetrySetup?.(effectiveAssessment.id)} disabled={!onRetrySetup}>Retry due diligence setup</button> : <button type="button" className="primary-button" onClick={() => void onRefresh?.()} disabled={!onRefresh}>View setup status</button>
              : status === "READY_TO_SEND" ? <button type="button" className="primary-button" onClick={() => openPanel("send")} disabled={!onSend}>Send due diligence request</button>
                : status === "SUBMITTED" ? <button type="button" className="primary-button" onClick={() => void startReview()} disabled={!onStartReview || busy || reviewState !== "live"}>{busy ? "Opening review…" : "Review vendor response"}</button>
                    : status === "UNDER_REVIEW" ? <button type="button" className="primary-button" onClick={() => openPanel("conclusion")} disabled={!onComplete || reviewState !== "live"}>Record assessment conclusion</button>
                      : null}
      {status === "UNDER_REVIEW" && onRequestClarification && <button type="button" className="secondary-button" onClick={() => openPanel("clarification")}>Request clarification</button>}
    </div>}

    {!form && !effectiveAssessment && <p className="vdd-limitation">No active due diligence form is available for this relationship. Activate a collection form before starting the review.</p>}
  </section>;
}

function StartPanel({ form, reviewDueDate, minimumDate, busy, onReviewDueDate, onCancel, onSubmit }: { form?: VendorAssessmentFormOption; reviewDueDate: string; minimumDate: string; busy: boolean; onReviewDueDate: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Assessment setup</span><h3>Start due diligence</h3><p>The selected form will use the current vendor and service details as known context.</p></div>
    <dl className="vdd-preview"><div><dt>Collection form</dt><dd>{form?.name ?? "No active form"}</dd></div><div><dt>Form version</dt><dd>{form ? `Version ${form.version}` : "Not available"}</dd></div><div><dt>Response layout</dt><dd>{form ? presentationLabel(form.presentation) : "Not available"}</dd></div></dl>
    <label className="vdd-field"><span>Review due date</span><input type="date" min={minimumDate} value={reviewDueDate} onChange={(event) => onReviewDueDate(event.target.value)} required/></label>
    <div className="vdd-panel-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={busy}>Cancel</button><button type="submit" className="primary-button" disabled={busy || !form}>{busy ? "Starting…" : "Start due diligence"}</button></div>
  </form>;
}

function SendPanel({ recipient, responseDueDate, invitationMinutes, minimumDate, reviewDueAt, busy, onRecipient, onResponseDueDate, onInvitationMinutes, onCancel, onSubmit }: { recipient: string; responseDueDate: string; invitationMinutes: number; minimumDate: string; reviewDueAt: string; busy: boolean; onRecipient: (value: string) => void; onResponseDueDate: (value: string) => void; onInvitationMinutes: (value: number) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Secure vendor request</span><h3>Send due diligence request</h3><p>The vendor contact receives access to this request only. The email address is cleared from this screen after the send attempt.</p></div>
    <div className="vdd-form-grid">
      <label className="vdd-field vdd-wide"><span>Vendor contact email</span><input type="email" inputMode="email" autoComplete="email" value={recipient} onChange={(event) => onRecipient(event.target.value)} required/></label>
      <label className="vdd-field"><span>Response due date</span><input type="date" value={responseDueDate} min={minimumDate} max={reviewDueAt.slice(0, 10)} onChange={(event) => onResponseDueDate(event.target.value)} required/></label>
      <label className="vdd-field"><span>Secure link valid for</span><select value={invitationMinutes} onChange={(event) => onInvitationMinutes(Number(event.target.value))}><option value={60}>1 hour</option><option value={1440}>24 hours</option><option value={10080}>7 days</option></select></label>
    </div>
    <p className="vdd-limitation">Sending the request starts evidence collection. A submitted response still requires bank review.</p>
    <div className="vdd-panel-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={busy}>Cancel</button><button type="submit" className="primary-button" disabled={busy}>{busy ? "Sending…" : "Send due diligence request"}</button></div>
  </form>;
}

function ReissuePanel({ recipient, invitationMinutes, busy, onRecipient, onInvitationMinutes, onCancel, onSubmit }: { recipient: string; invitationMinutes: number; busy: boolean; onRecipient: (value: string) => void; onInvitationMinutes: (value: number) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Vendor request access</span><h3>Send replacement link</h3><p>Sending a replacement ends access from the previous link and active vendor session.</p></div>
    <div className="vdd-form-grid">
      <label className="vdd-field vdd-wide"><span>Vendor contact email</span><input type="email" inputMode="email" autoComplete="email" value={recipient} onChange={(event) => onRecipient(event.target.value)} required/></label>
      <label className="vdd-field"><span>Replacement link valid for</span><select value={invitationMinutes} onChange={(event) => onInvitationMinutes(Number(event.target.value))}><option value={60}>1 hour</option><option value={1440}>24 hours</option><option value={10080}>7 days</option></select></label>
    </div>
    <div className="vdd-panel-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={busy}>Cancel</button><button type="submit" className="primary-button" disabled={busy}>{busy ? "Sending…" : "Send replacement link"}</button></div>
  </form>;
}

function ClarificationPanel({ fields, selected, message, dueDate, busy, onSelected, onMessage, onDueDate, onCancel, onSubmit }: { fields: { id: string; label: string }[]; selected: string[]; message: string; dueDate: string; busy: boolean; onSelected: (value: string[]) => void; onMessage: (value: string) => void; onDueDate: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Missing information</span><h3>Request clarification</h3><p>Select only the response fields that the vendor must update or support.</p></div>
    {fields.length ? <fieldset className="vdd-fieldset"><legend>Fields requiring clarification</legend>{fields.map((field) => <label key={field.id}><input type="checkbox" checked={selected.includes(field.id)} onChange={(event) => onSelected(event.target.checked ? [...selected, field.id] : selected.filter((id) => id !== field.id))}/><span>{field.label}</span></label>)}</fieldset> : <p className="vdd-limitation">No response fields are available for clarification. Reload the submitted response before creating a request.</p>}
    <label className="vdd-field"><span>What the vendor must provide</span><textarea rows={4} value={message} onChange={(event) => onMessage(event.target.value)} required/></label>
    <label className="vdd-field"><span>Response due date</span><input type="date" value={dueDate} onChange={(event) => onDueDate(event.target.value)} required/></label>
    <div className="vdd-panel-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={busy}>Cancel</button><button type="submit" className="primary-button" disabled={busy || fields.length === 0}>{busy ? "Creating request…" : "Send clarification request"}</button></div>
  </form>;
}

function ConclusionPanel({ conclusion, rationale, uncertainty, nextReviewDate, minimumDate, busy, onConclusion, onRationale, onUncertainty, onNextReviewDate, onCancel, onSubmit }: { conclusion: VendorAssessmentConclusion; rationale: string; uncertainty: string; nextReviewDate: string; minimumDate: string; busy: boolean; onConclusion: (value: VendorAssessmentConclusion) => void; onRationale: (value: string) => void; onUncertainty: (value: string) => void; onNextReviewDate: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Reviewer conclusion</span><h3>Record assessment conclusion</h3><p>Base the conclusion on the submitted response, reviewed documents and recorded findings.</p></div>
    <div className="vdd-form-grid">
      <label className="vdd-field"><span>Conclusion</span><select value={conclusion} onChange={(event) => onConclusion(event.target.value as VendorAssessmentConclusion)}>{conclusionOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select></label>
      <label className="vdd-field"><span>Recommended next review</span><input type="date" min={minimumDate} value={nextReviewDate} onChange={(event) => onNextReviewDate(event.target.value)}/></label>
      <label className="vdd-field vdd-wide"><span>Assessment basis</span><textarea rows={5} value={rationale} onChange={(event) => onRationale(event.target.value)} required/></label>
      <label className="vdd-field vdd-wide"><span>Remaining uncertainty</span><textarea rows={3} value={uncertainty} onChange={(event) => onUncertainty(event.target.value)}/></label>
    </div>
    <p className="vdd-limitation">This conclusion completes the assessment review. It does not activate or approve the vendor relationship.</p>
    <div className="vdd-panel-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={busy}>Cancel</button><button type="submit" className="primary-button" disabled={busy}>{busy ? "Recording…" : "Record assessment conclusion"}</button></div>
  </form>;
}

function ReviewSummary({ review, assessment, onReviewDocument, onCreateDeficiency }: { review: VendorAssessmentReviewView; assessment?: VendorAssessment | null; onReviewDocument?: Props["onReviewDocument"]; onCreateDeficiency?: Props["onCreateDeficiency"] }) {
  const visibleAnswers = review.answers.filter((answer) => answer.visibility === "VISIBLE");
  return <section className="vdd-review" aria-label="Vendor response review">
    <div className="vdd-review-header"><div><h3>Vendor response</h3>{review.response ? <p>Submitted {formatDate(review.response.submitted_at)} · {review.response.answer_count} {itemLabel(review.response.answer_count, "answer")} · {review.response.artifact_count} {itemLabel(review.response.artifact_count, "document")}</p> : <p>No submitted response summary is available.</p>}</div><div className="vdd-review-metrics"><span>{review.coverage.answered_required} of {review.coverage.required_fields} required answers received</span>{review.provisional_score?.score !== undefined && <span>Provisional score: {formatScore(review.provisional_score.score)} of 100 · Form version {review.assessment.form_template_version}</span>}</div></div>
    <div className="vdd-review-group"><h4>Submitted answers</h4>{visibleAnswers.length ? <dl className="vdd-answer-list">{visibleAnswers.map((answer) => <div key={answer.field_id}><dt>{answer.label}{answer.required ? " · Required" : ""}</dt><dd>{formatAnswer(answer.value)}</dd>{answer.provenance && <dd className="vdd-provenance">{provenanceLabel(answer.provenance)}</dd>}</div>)}</dl> : <p>No visible answers were submitted for this form version.</p>}</div>
    {review.documents.length > 0 && <div className="vdd-review-group"><h4>Supporting documents</h4>{review.documents.map((document) => <article className="vdd-document" key={document.artifact_id}><div><strong>{document.file_name}</strong><span>{document.document_type.replaceAll("_", " ")} · {formatBytes(document.size_bytes)}{document.expires_on ? ` · Expires ${formatDate(document.expires_on)}` : ""}</span><span className="vdd-evidence-status">{artifactStatusLabel(document.artifact_status)} · {evidenceClassLabel(document.evidence_class)}</span></div>{assessment?.status === "UNDER_REVIEW" && onReviewDocument && <div><button type="button" className="text-button" onClick={() => onReviewDocument(assessment.id, document, "VALIDATE")}>Validate document</button><button type="button" className="text-button" onClick={() => onReviewDocument(assessment.id, document, "REJECT")}>Reject document</button></div>}</article>)}</div>}
    <div className="vdd-review-group"><div className="vdd-review-group-heading"><h4>Findings</h4>{assessment?.status === "UNDER_REVIEW" && onCreateDeficiency && <button type="button" className="secondary-button" onClick={() => onCreateDeficiency(assessment.id)}>Create finding</button>}</div>{review.matters.length ? <ul>{review.matters.map((finding) => <li key={finding.matter_id}><strong>{finding.title}</strong><span>{humanizeStatus(finding.status)}</span></li>)}</ul> : <p>No findings are linked to this assessment.</p>}</div>
    {assessment?.status === "COMPLETED" && <div className="vdd-review-group"><h4>Recorded conclusion</h4><dl className="vdd-conclusion"><div><dt>Conclusion</dt><dd>{conclusionLabel(assessment.conclusion)}</dd></div><div><dt>Assessment basis</dt><dd>{assessment.conclusion_rationale || "No assessment basis was recorded."}</dd></div>{assessment.conclusion_uncertainty && <div><dt>Remaining uncertainty</dt><dd>{assessment.conclusion_uncertainty}</dd></div>}<div><dt>Completed</dt><dd>{formatDate(assessment.completed_at)}</dd></div></dl></div>}
  </section>;
}

function needsReviewView(status?: VendorAssessment["status"]) {
  return status === "SUBMITTED" || status === "UNDER_REVIEW" || status === "COMPLETED";
}

function formatAnswer(value?: VendorAssessmentReviewView["answers"][number]["value"]) {
  if (!value) return "No answer submitted";
  if (value.text?.trim()) return value.text;
  if (value.values?.length) return value.values.join(", ");
  if (value.document) return value.document.reference || value.document.document_type.replaceAll("_", " ");
  if (value.artifact_ids?.length) return `${value.artifact_ids.length} ${value.artifact_ids.length === 1 ? "file" : "files"} submitted`;
  return "No answer submitted";
}

function provenanceLabel(provenance: NonNullable<VendorAssessmentReviewView["answers"][number]["provenance"]>) {
  const observed = provenance.source_receipt?.observed_at ? formatDate(provenance.source_receipt.observed_at) : "";
  if (!provenance.origin && provenance.source) return provenance.source;
  if (provenance.origin === "SOURCE_PREFILLED") {
    const source = provenance.source_receipt?.source_id || provenance.source || "a connected source";
    return `Prefilled from ${source}${observed ? ` · Observed ${observed}` : ""}`;
  }
  if (provenance.origin === "RESPONDENT_CORRECTED") {
    return `Updated by the vendor${observed ? ` · Source value observed ${observed}` : ""}`;
  }
  if (provenance.origin === "RESPONDENT_ENTERED") return "Entered by the vendor";
  return observed ? `Answer source recorded · Observed ${observed}` : "Answer source recorded";
}

function artifactStatusLabel(value: string) {
  switch (value) {
    case "AVAILABLE": return "Scan complete";
    case "STORED_UNSCANNED": return "Security scan pending";
    case "QUARANTINED": return "Quarantined by security scan";
    case "DELETED": return "File unavailable";
    default: return `Scan status: ${humanizeStatus(value)}`;
  }
}

function evidenceClassLabel(value: string) {
  switch (value) {
    case "VENDOR_SUPPLIED": return "Vendor supplied evidence";
    case "BANK_VALIDATED": return "Bank validated evidence";
    case "OFFICIAL_SOURCE": return "Official source evidence";
    default: return humanizeStatus(value);
  }
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${Math.round(value / 1024)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

function formatScore(value: number) {
  return new Intl.NumberFormat("en-GB", { maximumFractionDigits: 2 }).format(value);
}

function itemLabel(count: number, noun: string) {
  return count === 1 ? noun : `${noun}s`;
}

function humanizeStatus(value: string) {
  return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase());
}

function assessmentStatusCopy(assessment?: VendorAssessment | null, setupFailure?: string) {
  if (!assessment) return { label: "Not started", tone: "neutral", description: "No due diligence review has been started for this vendor relationship." };
  switch (assessment.status) {
    case "SETUP_PENDING": return setupFailure
      ? { label: "Setup failed", tone: "danger", description: "The assessment exists, but its review record still needs to be prepared." }
      : { label: "Setup in progress", tone: "pending", description: "The assessment exists while its review record is being prepared." };
    case "READY_TO_SEND": return { label: "Ready to send", tone: "information", description: "The assessment is ready for the vendor contact and response deadline." };
    case "COLLECTING": return { label: "Awaiting response", tone: "pending", description: "The vendor request is open. Submission will not complete the bank review." };
    case "SUBMITTED": return { label: "Response received", tone: "information", description: "The vendor response was submitted and now requires bank review." };
    case "UNDER_REVIEW": return { label: "Under review", tone: "pending", description: "Review the response, documents and findings before recording a conclusion." };
    case "COMPLETED": return { label: conclusionLabel(assessment.conclusion), tone: assessment.conclusion === "UNSATISFACTORY" ? "danger" : "complete", description: "The assessment conclusion is recorded. The vendor relationship status remains separate." };
    case "CANCELLED": return { label: "Cancelled", tone: "neutral", description: "This assessment was cancelled. A new onboarding review has not been started." };
  }
}

function conclusionLabel(value?: VendorAssessmentConclusion) {
  return conclusionOptions.find((option) => option.value === value)?.label ?? "Completed";
}

function presentationLabel(value: VendorAssessmentFormOption["presentation"]) {
  if (value === "WIZARD") return "Step-by-step form";
  if (value === "CLASSIC") return "Single-page form";
  return "Best layout for the form";
}

function validEmail(value: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value.trim()) && value.trim().length <= 254;
}

function endOfDay(value: string) { return `${value}T23:59:59.000Z`; }
function startOfDay(value: string) { return `${value}T00:00:00.000Z`; }
function tomorrowUTC() { const date = new Date(); date.setUTCDate(date.getUTCDate() + 1); return date.toISOString().slice(0, 10); }

function formatDate(value?: string) {
  if (!value) return "Not set";
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date(parsed)) : "Not set";
}
