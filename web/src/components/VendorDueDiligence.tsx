import { useEffect, useLayoutEffect, useMemo, useState } from "react";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import type {
  CompleteVendorAssessmentInput,
  CancelVendorAssessmentInput,
  CreateVendorAssessmentDeficiencyInput,
  ReissueVendorAssessmentRequestInput,
  ReviewVendorAssessmentDocumentInput,
  SendVendorAssessmentRequestInput,
  StartVendorAssessmentInput,
  VendorAssessment,
  VendorAssessmentClarificationInput,
  VendorAssessmentClarificationOutcome,
  VendorAssessmentConclusion,
  VendorAssessmentDeficiencyOutcome,
  VendorAssessmentDocument,
  VendorAssessmentFormOption,
  VendorAssessmentReviewKind,
  VendorAssessmentReviewView,
  VendorAssessmentSendOutcome,
  VendorAssessmentSetupRetryOutcome,
  VendorAssessmentApplicationResult,
  ApplyVendorAssessmentResponseInput,
} from "../vendorAssessmentTypes";
import { VendorResponseReview } from "./forms/VendorResponseReview";
import { VendorEvidenceChecklist } from "./VendorEvidenceChecklist";
import type { PrepareVendorCollectionInput, PrepareVendorCollectionResult, VendorCollection } from "../vendorCollectionApi";
import { Button, CheckboxField, FocusedSheet, Notice, SelectField, StatusBadge, TextArea, TextField, type StatusTone } from "./ui";
import "./vendor-due-diligence.css";

type ViewState = "live" | "loading" | "unavailable";

type Props = {
  relationship: VendorRelationshipAggregate;
  accountableOwnerLabel?: string;
  assessment?: VendorAssessment | null;
  form?: VendorAssessmentFormOption;
  forms?: VendorAssessmentFormOption[];
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
  onRetrySetup?: (assessmentID: string, expectedVersion: number) => Promise<VendorAssessmentSetupRetryOutcome>;
  onRefresh?: () => Promise<void> | void;
  onRefreshReview?: (assessmentID: string) => Promise<void> | void;
  onSend?: (input: SendVendorAssessmentRequestInput) => Promise<VendorAssessmentSendOutcome>;
  onPrepare?: (input: PrepareVendorCollectionInput) => Promise<PrepareVendorCollectionResult>;
  onReissue?: (input: ReissueVendorAssessmentRequestInput) => Promise<VendorAssessmentSendOutcome>;
  onOpenRequest?: (requestID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  onStartReview?: (assessmentID: string, expectedVersion: number) => Promise<VendorAssessment | void> | VendorAssessment | void;
  onRequestClarification?: (assessmentID: string, input: VendorAssessmentClarificationInput) => Promise<VendorAssessmentClarificationOutcome>;
  onCreateDeficiency?: (assessmentID: string, input: CreateVendorAssessmentDeficiencyInput) => Promise<VendorAssessmentDeficiencyOutcome>;
  onOpenDocument?: (assessmentID: string, requestID: string, artifactID: string) => void;
  onReviewDocument?: (assessmentID: string, artifactID: string, input: ReviewVendorAssessmentDocumentInput) => Promise<VendorAssessmentReviewView>;
  onComplete?: (assessmentID: string, input: CompleteVendorAssessmentInput) => Promise<VendorAssessment | void> | VendorAssessment | void;
  onCancelAssessment?: (assessmentID: string, input: CancelVendorAssessmentInput) => Promise<VendorAssessment | void> | VendorAssessment | void;
  onSetUpForm?: () => void;
  onOpenForms?: () => void;
  onApplyResponse?: (assessmentID: string, revisionID: string, input: ApplyVendorAssessmentResponseInput) => Promise<VendorAssessmentApplicationResult>;
};

type ActionPanel = "start" | "send" | "reissue" | "clarification" | "deficiency" | "document" | "conclusion" | "cancelAssessment" | null;
type AssessmentStartMode = "initial" | "restart" | "reassessment";

const linkExpiryOptions = [{ id: "60", label: "1 hour" }, { id: "1440", label: "24 hours" }, { id: "10080", label: "7 days" }];

const conclusionOptions: { value: VendorAssessmentConclusion; label: string }[] = [
  { value: "SATISFACTORY", label: "Satisfactory" },
  { value: "SATISFACTORY_WITH_CONDITIONS", label: "Satisfactory with conditions" },
  { value: "UNSATISFACTORY", label: "Unsatisfactory" },
  { value: "INCONCLUSIVE", label: "Inconclusive" },
];

export function VendorDueDiligence({
  relationship,
  accountableOwnerLabel = "Current owner unavailable",
  assessment,
  form,
  forms = [],
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
  onPrepare,
  onReissue,
  onOpenRequest,
  onOpenMatter,
  onStartReview,
  onRequestClarification,
  onCreateDeficiency,
  onOpenDocument,
  onReviewDocument,
  onComplete,
  onCancelAssessment,
  onSetUpForm,
  onOpenForms,
  onApplyResponse,
}: Props) {
  const [panel, setPanel] = useState<ActionPanel>(null);
  const [localAssessment, setLocalAssessment] = useState<VendorAssessment | null | undefined>(assessment);
  const [localOutcome, setLocalOutcome] = useState<VendorAssessmentSendOutcome>();
  const [localOutcomeKind, setLocalOutcomeKind] = useState<"initial" | "replacement">("initial");
  const [reviewDueDate, setReviewDueDate] = useState(defaultReviewDueDate);
  const [reviewKind, setReviewKind] = useState<Extract<VendorAssessmentReviewKind, "PERIODIC" | "TRIGGERED">>("PERIODIC");
  const [reviewReference, setReviewReference] = useState("");
  const availableForms = useMemo(() => forms.length ? forms : form ? [form] : [], [forms, form]);
  const [selectedFormKey, setSelectedFormKey] = useState(form ? `${form.id}:${form.version}` : "");
  const [scopeKind, setScopeKind] = useState<"FULL" | "FOCUSED">("FULL");
  const [selectedFieldIDs, setSelectedFieldIDs] = useState<string[]>([]);
  const [applicationComplete, setApplicationComplete] = useState(false);
  const [recipient, setRecipient] = useState("");
  const [collection, setCollection] = useState<VendorCollection>();
  const [preparedDeadline, setPreparedDeadline] = useState("");
  const [responseDueDate, setResponseDueDate] = useState("");
  const [invitationMinutes, setInvitationMinutes] = useState(1440);
  const [selectedClarificationFields, setSelectedClarificationFields] = useState<string[]>([]);
  const [clarificationMessage, setClarificationMessage] = useState("");
  const [clarificationRecipient, setClarificationRecipient] = useState("");
  const [clarificationDueDate, setClarificationDueDate] = useState("");
  const [clarificationInvitationMinutes, setClarificationInvitationMinutes] = useState(1440);
  const [clarificationOutcome, setClarificationOutcome] = useState<VendorAssessmentClarificationOutcome>();
  const [deficiencyKey, setDeficiencyKey] = useState("");
  const [deficiencyTitle, setDeficiencyTitle] = useState("");
  const [deficiencySummary, setDeficiencySummary] = useState("");
  const [deficiencyDueDate, setDeficiencyDueDate] = useState("");
  const [selectedDocument, setSelectedDocument] = useState<VendorAssessmentDocument>();
  const [documentDecision, setDocumentDecision] = useState<"VALIDATE" | "REJECT">("VALIDATE");
  const [documentType, setDocumentType] = useState("");
  const [documentEvidenceClass, setDocumentEvidenceClass] = useState<ReviewVendorAssessmentDocumentInput["evidence_class"]>("VENDOR_SUPPLIED");
  const [documentValidUntil, setDocumentValidUntil] = useState("");
  const [conclusion, setConclusion] = useState<VendorAssessmentConclusion | "">("");
  const [rationale, setRationale] = useState("");
  const [uncertainty, setUncertainty] = useState("");
  const [nextReviewDate, setNextReviewDate] = useState("");
  const [cancellationReason, setCancellationReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  useLayoutEffect(() => {
    setLocalAssessment(assessment);
    setLocalOutcome(undefined);
    setLocalOutcomeKind("initial");
    setClarificationOutcome(undefined);
    setPanel(null);
    setError("");
    setNotice("");
    setConclusion("");
    setRationale("");
    setUncertainty("");
    setNextReviewDate("");
    setCancellationReason("");
    setScopeKind("FULL");
    setSelectedFieldIDs([]);
    setApplicationComplete(false);
  }, [assessment?.id]);

  useEffect(() => {
    setLocalAssessment(assessment);
    setPanel(null);
    if (assessment?.status !== "COLLECTING") setClarificationOutcome(undefined);
  }, [assessment]);

  useEffect(() => setApplicationComplete(Boolean(review?.application_receipt)), [assessment?.id, review?.application_receipt?.id]);

  const effectiveOutcome = localOutcome ?? requestOutcome;
  const effectiveOutcomeKind = localOutcome ? localOutcomeKind : requestOutcomeKind;
  const effectiveAssessment = localAssessment && (!effectiveOutcome || localAssessment.version >= effectiveOutcome.assessment.version) ? localAssessment : effectiveOutcome?.assessment ?? localAssessment;
  const currentCollection = collection?.assessment_id === effectiveAssessment?.id && collection?.request_id === effectiveAssessment?.current_request_id ? collection : undefined;
  const status = effectiveAssessment?.status;
  const requestID = effectiveAssessment?.current_request_id ?? effectiveOutcome?.request.id;
  const dueDate = effectiveAssessment?.review_due_at;
  const minimumFutureDate = tomorrowUTC();
  const availableClarificationFields = clarificationFields.length ? clarificationFields : review?.answers
    .filter((answer) => answer.visibility === "VISIBLE")
    .map((answer) => ({ id: answer.field_id, label: answer.label })) ?? [];

  const statusCopy = useMemo(() => assessmentStatusCopy(effectiveAssessment, setupFailure), [effectiveAssessment, setupFailure]);
  const startMode = assessmentStartMode(relationship.relationship.status, effectiveAssessment);
  const selectedForm = availableForms.find((value) => `${value.id}:${value.version}` === selectedFormKey) ?? availableForms[0];
  const responseRequiresApplication = Boolean(onApplyResponse && review?.answers.some((answer) => answer.baseline));

  useEffect(() => {
    if (!availableForms.length) { setSelectedFormKey(""); return; }
    if (!availableForms.some((value) => `${value.id}:${value.version}` === selectedFormKey)) setSelectedFormKey(`${availableForms[0]!.id}:${availableForms[0]!.version}`);
  }, [availableForms, selectedFormKey]);

  function openPanel(next: ActionPanel) {
    setPanel(next);
    setError("");
    setNotice("");
  }

  async function startAssessment(event: React.FormEvent) {
    event.preventDefault();
    if (!selectedForm || !onStart || !reviewDueDate || Date.parse(endOfDay(reviewDueDate)) <= Date.now() || (startMode === "reassessment" && (!reviewReference.trim() || (scopeKind === "FOCUSED" && selectedFieldIDs.length === 0)))) {
      setError(startMode === "reassessment" ? "Choose a review type, enter the review reference and set a future due date. A focused review must include at least one held record." : "Choose an active collection form and review due date before starting due diligence.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      const episode = startMode === "restart" && effectiveAssessment
        ? { review_kind: "ONBOARDING" as const, restart_assessment_id: effectiveAssessment.id }
        : startMode === "reassessment"
          ? { review_kind: reviewKind, source_trigger: reviewReference.trim() }
          : {};
      const result = await onStart({
        relationship_version: relationship.relationship.version,
        ...episode,
        form_template_id: selectedForm.id,
        form_template_version: selectedForm.version,
        review_due_at: endOfDay(reviewDueDate),
        ...(startMode === "reassessment" ? { scope_kind: scopeKind, selected_field_ids: scopeKind === "FOCUSED" ? selectedFieldIDs : [] } : {}),
      });
      if (result) setLocalAssessment(result);
      setPanel(null);
      setReviewReference("");
      setNotice(startMode === "reassessment" ? "Reassessment started. The review record is being prepared." : startMode === "restart" ? "Due diligence restarted. The new review record is being prepared." : "Due diligence started. The review record is being prepared.");
    } catch {
      setError(startMode === "reassessment" ? "The reassessment could not be started. Check the relationship, review reference and form version, then try again." : "Due diligence could not be started. Check the current relationship and form versions, then try again.");
    } finally {
      setBusy(false);
    }
  }

  async function sendRequest(event: React.FormEvent) {
    event.preventDefault();
    const preparing = Boolean(onPrepare && !effectiveAssessment?.current_request_id);
    const prepared = Boolean(onPrepare && effectiveAssessment?.current_request_id);
    const deadlineValue = prepared ? currentCollection?.deadline || preparedDeadline : responseDueDate ? endOfDay(responseDueDate) : "";
    const deadline = Date.parse(deadlineValue);
    if (!effectiveAssessment || !onSend || (!prepared && !validEmail(recipient)) || !Number.isFinite(deadline) || deadline <= Date.now() || (!prepared && deadline > Date.parse(effectiveAssessment.review_due_at))) {
      setError("Enter a valid vendor contact email and a future response date no later than the assessment due date.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (preparing && onPrepare) {
        const outcome = await onPrepare({ expected_version: effectiveAssessment.version, audience: recipient.trim().toLowerCase(), deadline: deadlineValue });
        setLocalAssessment(outcome.assessment);
        setPreparedDeadline(outcome.request.deadline ?? deadlineValue);
        setPanel(null);
        setNotice("Request prepared. Review evidence already held before sending the outstanding items to the vendor.");
        return;
      }
      const outcome = await onSend({
        expected_version: Math.max(effectiveAssessment.version, currentCollection?.assessment_version ?? 0),
        audience: prepared ? "" : recipient.trim().toLowerCase(),
        deadline: deadlineValue,
        invitation_ttl_minutes: invitationMinutes,
      });
      setRecipient("");
      setLocalOutcome(outcome);
      setLocalOutcomeKind("initial");
      setPanel(null);
      if (outcome.state === "DELIVERED") setNotice(`The request was sent. The response is due ${formatDate(outcome.request.deadline)}.`);
    } catch {
      if (preparing || prepared) setError(preparing ? "The request could not be prepared. Check the contact and deadline, then try again." : "The prepared request was not sent. Reload the checklist and try again.");
      else { setRecipient(""); setPanel(null); setError("The request was not sent. Re-enter the vendor contact email before trying again."); }
    } finally {
      setBusy(false);
    }
  }

  async function reissueRequest(event: React.FormEvent) {
    event.preventDefault();
    if (!effectiveAssessment || !onReissue || !validEmail(recipient)) {
      setError("Enter a valid vendor contact email before sending another link.");
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
      if (outcome.state === "DELIVERED") setNotice("Another link was sent. Earlier links remain available until their printed expiry unless you cancel the request.");
    } catch {
      setPanel(null);
      setError("The new link was not sent. Re-enter the vendor contact email before trying again.");
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

  async function retrySetup() {
    if (!effectiveAssessment || !onRetrySetup) return;
    await run(async () => {
      const outcome = await onRetrySetup(effectiveAssessment.id, effectiveAssessment.version);
      setLocalAssessment(outcome.assessment);
      setNotice("Assessment setup queued. Review setup will continue in the background.");
    }, "Assessment setup could not be queued. Reload the assessment before trying again.");
  }

  async function submitClarification(event: React.FormEvent) {
    event.preventDefault();
    const audience = clarificationRecipient.trim().toLowerCase();
    setClarificationRecipient("");
    const deadline = clarificationDueDate ? Date.parse(endOfDay(clarificationDueDate)) : Number.NaN;
    if (!effectiveAssessment || !onRequestClarification || selectedClarificationFields.length === 0 || !clarificationMessage.trim() || clarificationMessage.trim().length > 2000 || !validEmail(audience) || !Number.isFinite(deadline) || deadline <= Date.now() || deadline > Date.parse(effectiveAssessment.review_due_at)) {
      setError("Select the response fields, enter a valid vendor contact email and set a future due date no later than the assessment due date.");
      return;
    }
    await run(async () => {
      const outcome = await onRequestClarification(effectiveAssessment.id, {
        expected_version: effectiveAssessment.version,
        request_fields: selectedClarificationFields,
        message: clarificationMessage.trim(),
        audience,
        deadline: endOfDay(clarificationDueDate),
        invitation_ttl_minutes: clarificationInvitationMinutes,
      });
      setLocalAssessment(outcome.assessment);
      setClarificationOutcome(outcome);
      setClarificationMessage("");
      setClarificationDueDate("");
      setSelectedClarificationFields([]);
      setPanel(null);
      if (outcome.state === "DELIVERED") setNotice("Clarification sent. The assessment will return to review when the vendor responds.");
      else if (outcome.state === "LINK_CREATED_EMAIL_NOT_SENT") setNotice("Clarification request created. Email delivery did not complete.");
      else setNotice("Clarification request prepared. Secure vendor access was not issued.");
    }, "The clarification request was not sent. Re-enter the vendor contact email before trying again.");
  }

  async function submitDeficiency(event: React.FormEvent) {
    event.preventDefault();
    const triggerKey = deficiencyKey.trim().toLowerCase();
    const dueAt = deficiencyDueDate ? Date.parse(endOfDay(deficiencyDueDate)) : Number.NaN;
    if (!effectiveAssessment || !onCreateDeficiency || !/^[a-z0-9][a-z0-9._:-]{0,79}$/.test(triggerKey) || !deficiencyTitle.trim() || deficiencyTitle.trim().length > 200 || !deficiencySummary.trim() || deficiencySummary.trim().length > 2000 || !Number.isFinite(dueAt) || dueAt <= Date.now()) {
      setError("Enter a valid finding reference, title, details and future action due date.");
      return;
    }
    await run(async () => {
      const outcome = await onCreateDeficiency(effectiveAssessment.id, {
        expected_version: effectiveAssessment.version,
        trigger_key: triggerKey,
        title: deficiencyTitle.trim(),
        summary: deficiencySummary.trim(),
        due_at: endOfDay(deficiencyDueDate),
      });
      setLocalAssessment(outcome.assessment);
      setDeficiencyKey("");
      setDeficiencyTitle("");
      setDeficiencySummary("");
      setDeficiencyDueDate("");
      setPanel(null);
      const reference = outcome.matter.matter.reference || outcome.matter.matter.title;
      setNotice(`Finding ${reference} is open and linked to this assessment.`);
    }, "The finding could not be recorded. Your entries remain on this screen; reload the assessment before trying again.");
  }

  function openDocumentReview(document: VendorAssessmentDocument, decision: "VALIDATE" | "REJECT") {
    setSelectedDocument(document);
    setDocumentDecision(decision);
    setDocumentType(document.document_type);
    setDocumentEvidenceClass(decision === "VALIDATE" ? "BANK_VALIDATED" : document.evidence_class as ReviewVendorAssessmentDocumentInput["evidence_class"]);
    setDocumentValidUntil(document.expires_on?.slice(0, 10) ?? "");
    openPanel("document");
  }

  async function submitDocumentDecision(event: React.FormEvent) {
    event.preventDefault();
    if (!effectiveAssessment || !selectedDocument || !onReviewDocument || !documentType.trim() || documentType.trim().length > 128) {
      setError("Enter the document type and evidence class before recording this decision.");
      return;
    }
    await run(async () => {
      const refreshed = await onReviewDocument(effectiveAssessment.id, selectedDocument.artifact_id, {
        expected_version: effectiveAssessment.version,
        field_id: selectedDocument.field_id,
        decision: documentDecision,
        document_type: documentType.trim(),
        evidence_class: documentEvidenceClass,
        valid_until: documentValidUntil || undefined,
      });
      setLocalAssessment(refreshed.assessment);
      setSelectedDocument(undefined);
      setPanel(null);
      setNotice(documentDecision === "VALIDATE" ? "Document accepted." : "Document rejected.");
    }, "The document decision could not be recorded. Reload the response before trying again.");
  }

  async function submitConclusion(event: React.FormEvent) {
    event.preventDefault();
    if (!effectiveAssessment || !onComplete || !conclusion || !rationale.trim() || (nextReviewDate && Date.parse(startOfDay(nextReviewDate)) <= Date.now())) {
      setError("Select a conclusion and record the assessment basis before completing the review.");
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

  async function cancelAssessment(event: React.FormEvent) {
    event.preventDefault();
    if (!effectiveAssessment || !onCancelAssessment || !cancellationReason.trim()) {
      setError("Enter why this assessment is being cancelled.");
      return;
    }
    await run(async () => {
      const result = await onCancelAssessment(effectiveAssessment.id, { expected_version: effectiveAssessment.version, reason: cancellationReason.trim() });
      if (result) setLocalAssessment(result);
      setCancellationReason("");
      setPanel(null);
      setNotice("Assessment cancelled. The vendor relationship was not changed.");
    }, "The assessment could not be cancelled. Your reason remains on this screen; reload the assessment before trying again.");
  }

  async function copyCaptureLink() {
    const captureURL = effectiveOutcome?.capture_url;
    if (!captureURL) return;
    try {
      await navigator.clipboard.writeText(captureURL);
      setNotice(effectiveOutcomeKind === "replacement" ? "New secure link copied." : "Secure link copied.");
    } catch {
      setError("The secure link could not be copied. Use the request status to retry delivery.");
    }
  }

  async function copyClarificationLink() {
    if (!clarificationOutcome?.capture_url) return;
    try {
      await navigator.clipboard.writeText(clarificationOutcome.capture_url);
      setNotice("Clarification link copied.");
    } catch {
      setError("The clarification link could not be copied. Use the request status to retry delivery.");
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
    return <section className="vdd-workspace" aria-label="Due diligence" tabIndex={-1}><div className="vdd-state" aria-live="polite" aria-busy="true">Loading due diligence for {relationship.relationship.service_name}…</div></section>;
  }

  if (viewState === "unavailable") {
    return <section className="vdd-workspace" aria-label="Due diligence" tabIndex={-1}><div className="vdd-state vdd-state-error" role="alert"><h2>Due diligence is unavailable</h2><p>The current assessment for {relationship.relationship.service_name} could not be loaded. Try again before starting or changing the review.</p>{onRefresh && <Button type="button" variant="secondary" onPress={() => void onRefresh()}>Try again</Button>}</div></section>;
  }

  return <section className="vdd-workspace" aria-labelledby="vdd-title" tabIndex={-1}>
    <header className="vdd-header">
      <div><span className="eyebrow">{relationship.relationship.service_name}</span><h2 id="vdd-title" tabIndex={-1}>Due diligence</h2><p>{statusCopy.description}</p></div>
      <StatusBadge tone={dueDiligenceTone(statusCopy.tone)}>{statusCopy.label}</StatusBadge>
    </header>

    <div className="vdd-scope" aria-label="Assessment scope">
      <div><span>Vendor</span><strong>{relationship.vendor.legal_name}</strong></div>
      <div><span>Service</span><strong>{relationship.relationship.service_name}</strong></div>
      <div><span>Accountable owner</span><strong>{accountableOwnerLabel}</strong></div>
      <div><span>Review due</span><strong>{formatDate(dueDate)}</strong></div>
    </div>

    {sourceStatus && sourceStatus.state !== "CURRENT" && <Notice tone="warning"><strong>{sourceStatus.state === "STALE" ? "Vendor source is out of date" : "Vendor source is unavailable"}</strong> {sourceStatus.detail}</Notice>}
    {setupFailure && status === "SETUP_PENDING" && <Notice tone="error"><strong>Review setup needs attention</strong> {setupFailure}</Notice>}
    {effectiveOutcome?.state === "REQUEST_READY_INVITATION_NOT_ISSUED" && <Notice tone="error"><strong>The request is ready, but secure access was not issued</strong> {effectiveOutcome.recovery ?? "Retry invitation creation for this request."}</Notice>}
    {effectiveOutcome?.state === "LINK_CREATED_EMAIL_NOT_SENT" && <Notice tone="error"><strong>Email delivery did not complete</strong> {effectiveOutcome.recovery ?? "Copy the secure link or review delivery status."}</Notice>}
    {clarificationOutcome?.state === "REQUEST_READY_INVITATION_NOT_ISSUED" && <Notice tone="error"><strong>Clarification access was not issued</strong> {clarificationOutcome.recovery ?? "Review the clarification request before retrying secure access."}</Notice>}
    {clarificationOutcome?.state === "LINK_CREATED_EMAIL_NOT_SENT" && <Notice tone="error"><strong>Clarification email delivery did not complete</strong> {clarificationOutcome.recovery ?? "Use the returned secure link or review delivery status."}</Notice>}
    {notice && <Notice tone="success">{notice}</Notice>}
    {error && panel !== "document" && <Notice tone="error">{error}</Notice>}

    {effectiveAssessment && needsReviewView(status) && reviewState === "loading" && <div className="vdd-review-state" aria-live="polite" aria-busy="true">Loading the submitted response and supporting documents…</div>}
    {effectiveAssessment && needsReviewView(status) && reviewState === "unavailable" && <Notice tone="error"><strong>Vendor response is unavailable</strong> The submitted answers and documents could not be loaded. Reload them before starting or completing the review. {onRefreshReview && <Button type="button" variant="secondary" onPress={() => void onRefreshReview(effectiveAssessment.id)}>Reload vendor response</Button>}</Notice>}

    {onPrepare && effectiveAssessment && !["SETUP_PENDING", "CANCELLED"].includes(effectiveAssessment.status) && <VendorEvidenceChecklist key={effectiveAssessment.id} assessmentID={effectiveAssessment.id} assessmentVersion={effectiveAssessment.version} relationshipID={relationship.relationship.id} onLoaded={setCollection} onChanged={async () => { await onRefresh?.(); }} documents={review?.documents} onReviewDocument={status === "UNDER_REVIEW" && !panel && onReviewDocument ? openDocumentReview : undefined} onOpenDocument={onOpenDocument ? (document) => { const request = document.request_id || review?.response?.request_id; if (request) onOpenDocument(effectiveAssessment.id, request, document.artifact_id); } : undefined}/>}

    {reviewState === "live" && review && <ReviewSummary review={review} assessment={effectiveAssessment} documents={currentCollection ? review.documents.filter((document) => !currentCollection.fields.some((field) => field.field_id === document.field_id)) : review.documents} onOpenMatter={onOpenMatter} onOpenDocument={!panel && onOpenDocument ? onOpenDocument : undefined} onReviewDocument={!panel && onReviewDocument ? openDocumentReview : undefined} onCreateDeficiency={!panel && onCreateDeficiency ? () => openPanel("deficiency") : undefined}/>}

    {status === "UNDER_REVIEW" && reviewState === "live" && review && onApplyResponse && <VendorResponseReview relationship={relationship} review={review} onApply={onApplyResponse} onApplied={(result) => { setLocalAssessment(result.review.assessment); setApplicationComplete(true); }}/>}

    {panel === "start" && startMode && <StartPanel mode={startMode} forms={availableForms} selectedForm={selectedForm} onSelectedForm={setSelectedFormKey} scopeKind={scopeKind} selectedFieldIDs={selectedFieldIDs} onScopeKind={setScopeKind} onSelectedFieldIDs={setSelectedFieldIDs} reviewDueDate={reviewDueDate} reviewKind={reviewKind} reviewReference={reviewReference} minimumDate={minimumFutureDate} busy={busy} onReviewDueDate={setReviewDueDate} onReviewKind={setReviewKind} onReviewReference={setReviewReference} onCancel={() => setPanel(null)} onSubmit={startAssessment}/>}
    {panel === "send" && effectiveAssessment && <SendPanel preparing={Boolean(onPrepare && !effectiveAssessment.current_request_id)} prepared={Boolean(onPrepare && effectiveAssessment.current_request_id)} preparedDeadline={collection?.deadline || preparedDeadline} audienceHint={collection?.audience_hint} recipient={recipient} responseDueDate={responseDueDate} invitationMinutes={invitationMinutes} minimumDate={minimumFutureDate} reviewDueAt={effectiveAssessment.review_due_at} busy={busy} onRecipient={setRecipient} onResponseDueDate={setResponseDueDate} onInvitationMinutes={setInvitationMinutes} onCancel={() => { if (!onPrepare) setRecipient(""); setPanel(null); }} onSubmit={sendRequest}/>}
    {panel === "reissue" && effectiveAssessment && <ReissuePanel recipient={recipient} invitationMinutes={invitationMinutes} busy={busy} onRecipient={setRecipient} onInvitationMinutes={setInvitationMinutes} onCancel={() => { setRecipient(""); setPanel(null); }} onSubmit={reissueRequest}/>}
    {panel === "clarification" && effectiveAssessment && <ClarificationPanel fields={availableClarificationFields} selected={selectedClarificationFields} message={clarificationMessage} recipient={clarificationRecipient} dueDate={clarificationDueDate} invitationMinutes={clarificationInvitationMinutes} minimumDate={minimumFutureDate} reviewDueAt={effectiveAssessment.review_due_at} busy={busy} onSelected={setSelectedClarificationFields} onMessage={setClarificationMessage} onRecipient={setClarificationRecipient} onDueDate={setClarificationDueDate} onInvitationMinutes={setClarificationInvitationMinutes} onCancel={() => { setClarificationRecipient(""); setPanel(null); }} onSubmit={submitClarification}/>}
    {panel === "deficiency" && <DeficiencyPanel triggerKey={deficiencyKey} title={deficiencyTitle} summary={deficiencySummary} dueDate={deficiencyDueDate} minimumDate={minimumFutureDate} busy={busy} onTriggerKey={setDeficiencyKey} onTitle={setDeficiencyTitle} onSummary={setDeficiencySummary} onDueDate={setDeficiencyDueDate} onCancel={() => setPanel(null)} onSubmit={submitDeficiency}/>}
    {panel === "document" && selectedDocument && <FocusedSheet label="Review document" isDismissable={!busy} onClose={() => { setSelectedDocument(undefined); setPanel(null); }}><div className="vdd-document-sheet">{error && <Notice tone="error">{error}</Notice>}<DocumentDecisionPanel document={selectedDocument} decision={documentDecision} documentType={documentType} evidenceClass={documentEvidenceClass} validUntil={documentValidUntil} busy={busy} onDecision={(value) => { setDocumentDecision(value); setDocumentEvidenceClass(value === "VALIDATE" ? "BANK_VALIDATED" : selectedDocument.evidence_class as ReviewVendorAssessmentDocumentInput["evidence_class"]); }} onDocumentType={setDocumentType} onEvidenceClass={setDocumentEvidenceClass} onValidUntil={setDocumentValidUntil} onCancel={() => { setSelectedDocument(undefined); setPanel(null); }} onSubmit={submitDocumentDecision}/></div></FocusedSheet>}
    {panel === "conclusion" && <ConclusionPanel conclusion={conclusion} rationale={rationale} uncertainty={uncertainty} nextReviewDate={nextReviewDate} minimumDate={minimumFutureDate} busy={busy} onConclusion={setConclusion} onRationale={setRationale} onUncertainty={setUncertainty} onNextReviewDate={setNextReviewDate} onCancel={() => setPanel(null)} onSubmit={submitConclusion}/>}
    {panel === "cancelAssessment" && <CancelAssessmentPanel reason={cancellationReason} busy={busy} onReason={setCancellationReason} onCancel={() => setPanel(null)} onSubmit={cancelAssessment}/>}

    {!panel && <div className="vdd-actions">
      {status === "COLLECTING" ? <><Button type="button" variant="primary" onPress={() => requestID && onOpenRequest?.(requestID)} isDisabled={!requestID || !onOpenRequest}>Review request status</Button>{clarificationOutcome?.capture_url && <Button type="button" variant="secondary" onPress={() => void copyClarificationLink()}>Copy clarification link</Button>}{effectiveOutcome?.state === "LINK_CREATED_EMAIL_NOT_SENT" && effectiveOutcome.capture_url ? <Button type="button" variant="secondary" onPress={() => void copyCaptureLink()}>{effectiveOutcomeKind === "replacement" ? "Copy new link" : "Copy secure link"}</Button> : <Button type="button" variant="secondary" onPress={() => openPanel("reissue")} isDisabled={!onReissue}>{effectiveOutcomeKind === "replacement" && effectiveOutcome?.state === "REQUEST_READY_INVITATION_NOT_ISSUED" ? "Retry new link" : "Send another link"}</Button>}</>
        : effectiveOutcome?.state === "LINK_CREATED_EMAIL_NOT_SENT" && effectiveOutcome.capture_url ? <Button type="button" variant="primary" onPress={() => void copyCaptureLink()}>Copy secure link</Button>
        : effectiveOutcome?.state === "REQUEST_READY_INVITATION_NOT_ISSUED" ? <Button type="button" variant="primary" onPress={() => openPanel("send")} isDisabled={!onSend}>Retry invitation creation</Button>
          : startMode ? availableForms.length ? <Button type="button" variant="primary" onPress={() => openPanel("start")} isDisabled={!onStart}>{startActionLabel(startMode)}</Button> : <><Button type="button" variant="primary" onPress={onSetUpForm} isDisabled={!onSetUpForm}>Use a starter template</Button>{onOpenForms && <Button type="button" variant="secondary" onPress={onOpenForms}>Open Forms</Button>}</>
            : status === "SETUP_PENDING" ? setupFailure ? <Button type="button" variant="primary" onPress={() => void retrySetup()} isDisabled={!onRetrySetup || busy}>{busy ? "Queuing setup…" : "Retry due diligence setup"}</Button> : <Button type="button" variant="primary" onPress={() => void onRefresh?.()} isDisabled={!onRefresh}>View setup status</Button>
              : collection?.can_start_review && ["READY_TO_SEND", "COLLECTING"].includes(status ?? "") ? <Button type="button" variant="primary" onPress={() => void startReview()} isDisabled={!onStartReview || busy}>Review existing evidence</Button>
              : status === "READY_TO_SEND" ? <Button type="button" variant="primary" onPress={() => openPanel("send")} isDisabled={!onSend}>{onPrepare && !effectiveAssessment?.current_request_id ? "Prepare request" : "Send due diligence request"}</Button>
                : status === "SUBMITTED" ? <Button type="button" variant="primary" onPress={() => void startReview()} isDisabled={!onStartReview || busy || reviewState !== "live"}>{busy ? "Opening review…" : "Review vendor response"}</Button>
                    : status === "UNDER_REVIEW" && (!responseRequiresApplication || applicationComplete) ? <Button type="button" variant="primary" onPress={() => openPanel("conclusion")} isDisabled={!onComplete || reviewState !== "live"}>Record assessment conclusion</Button>
                      : null}
      {status === "UNDER_REVIEW" && onRequestClarification && <Button type="button" variant="secondary" onPress={() => openPanel("clarification")}>Request clarification</Button>}
      {effectiveAssessment && !["COMPLETED", "CANCELLED"].includes(effectiveAssessment.status) && onCancelAssessment && <Button type="button" variant="secondary" onPress={() => openPanel("cancelAssessment")}>Cancel assessment</Button>}
    </div>}

    {!availableForms.length && !effectiveAssessment && <p className="vdd-limitation">No active due-diligence form was found in this legal entity. Create a new form draft or open Forms to review and approve an existing template before starting this vendor review.</p>}
  </section>;
}

function CancelAssessmentPanel({ reason, busy, onReason, onCancel, onSubmit }: { reason: string; busy: boolean; onReason: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Assessment status</span><h3>Cancel assessment</h3><p>Cancel this assessment when the current review should stop. The vendor relationship and recorded evidence remain available.</p></div>
    <TextArea label="Reason for cancellation" rows={4} maxLength={2000} value={reason} onChange={(value) => onReason(value)} isRequired/>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Keep assessment</Button><Button type="submit" variant="primary" isDisabled={busy || !reason.trim()}>{busy ? "Cancelling…" : "Cancel assessment"}</Button></div>
  </form>;
}

function StartPanel({ mode, forms, selectedForm, onSelectedForm, scopeKind, selectedFieldIDs, onScopeKind, onSelectedFieldIDs, reviewDueDate, reviewKind, reviewReference, minimumDate, busy, onReviewDueDate, onReviewKind, onReviewReference, onCancel, onSubmit }: { mode: AssessmentStartMode; forms: VendorAssessmentFormOption[]; selectedForm?: VendorAssessmentFormOption; onSelectedForm: (value: string) => void; scopeKind: "FULL" | "FOCUSED"; selectedFieldIDs: string[]; onScopeKind: (value: "FULL" | "FOCUSED") => void; onSelectedFieldIDs: (value: string[]) => void; reviewDueDate: string; reviewKind: Extract<VendorAssessmentReviewKind, "PERIODIC" | "TRIGGERED">; reviewReference: string; minimumDate: string; busy: boolean; onReviewDueDate: (value: string) => void; onReviewKind: (value: Extract<VendorAssessmentReviewKind, "PERIODIC" | "TRIGGERED">) => void; onReviewReference: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  const action = startActionLabel(mode);
  const refreshFields = selectedForm?.fields?.filter((field) => field.collection_intent && field.collection_intent !== "CAPTURE" && field.target_key) ?? [];
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Assessment setup</span><h3>{action}</h3><p>{mode === "reassessment" ? "Use the review reference that identifies this review. Known vendor and service details will remain available to the form." : mode === "restart" ? "Start a new onboarding review while preserving the cancelled assessment and its history." : "The selected form will use the current vendor and service details as known context."}</p></div>
    <SelectField label="Active collection form" value={selectedForm ? `${selectedForm.id}:${selectedForm.version}` : undefined} placeholder="Select an active form" options={forms.map((item) => ({ id: `${item.id}:${item.version}`, label: `${item.name} · version ${item.version}` }))} onChange={(value) => { onSelectedForm(value ?? ""); onSelectedFieldIDs([]); }}/>
    <dl className="vdd-preview"><div><dt>Collection form</dt><dd>{selectedForm?.name ?? "No active form"}</dd></div><div><dt>Form version</dt><dd>{selectedForm ? `Version ${selectedForm.version}` : "Not available"}</dd></div><div><dt>Response layout</dt><dd>{selectedForm ? presentationLabel(selectedForm.presentation) : "Not available"}</dd></div></dl>
    {mode === "reassessment" && <div className="vdd-form-grid"><SelectField label="Review type" value={reviewKind} placeholder="Select review type" allowsEmpty={false} options={[{ id: "PERIODIC", label: "Scheduled review" }, { id: "TRIGGERED", label: "Event or change" }]} onChange={(value) => { if (value) onReviewKind(value); }}/><TextField label="Review reference" value={reviewReference} maxLength={128} onChange={(value) => onReviewReference(value)} isRequired/></div>}
    {mode === "reassessment" && <fieldset className="vdd-fieldset"><legend>Information to request</legend><label><input type="radio" name="assessment-scope" checked={scopeKind === "FULL"} onChange={() => { onScopeKind("FULL"); onSelectedFieldIDs([]); }}/><span>Full form</span></label><label><input type="radio" name="assessment-scope" checked={scopeKind === "FOCUSED"} disabled={refreshFields.length === 0} onChange={() => { onScopeKind("FOCUSED"); onSelectedFieldIDs(refreshFields.map((field) => field.id)); }}/><span>Selected held records only</span></label>{scopeKind === "FOCUSED" && <div className="vdd-refresh-fields">{refreshFields.map((field) => <CheckboxField key={field.id} label={field.label} description={humanizeStatus(field.collection_intent ?? "CAPTURE")} isSelected={selectedFieldIDs.includes(field.id)} onChange={(selected) => onSelectedFieldIDs(selected ? [...selectedFieldIDs, field.id] : selectedFieldIDs.filter((id) => id !== field.id))}/>)}</div>}</fieldset>}
    <TextField label="Review due date" type="date" min={minimumDate} value={reviewDueDate} onChange={(value) => onReviewDueDate(value)} isRequired/>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Cancel</Button><Button type="submit" variant="primary" isDisabled={busy || !selectedForm || (mode === "reassessment" && (!reviewReference.trim() || (scopeKind === "FOCUSED" && selectedFieldIDs.length === 0)))}>{busy ? "Starting…" : action}</Button></div>
  </form>;
}

function SendPanel({ preparing = false, prepared = false, preparedDeadline, audienceHint, recipient, responseDueDate, invitationMinutes, minimumDate, reviewDueAt, busy, onRecipient, onResponseDueDate, onInvitationMinutes, onCancel, onSubmit }: { preparing?: boolean; prepared?: boolean; preparedDeadline?: string; audienceHint?: string; recipient: string; responseDueDate: string; invitationMinutes: number; minimumDate: string; reviewDueAt: string; busy: boolean; onRecipient: (value: string) => void; onResponseDueDate: (value: string) => void; onInvitationMinutes: (value: number) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Vendor evidence request</span><h3>{preparing ? "Prepare request" : "Send due diligence request"}</h3><p>{preparing ? "Confirm the vendor contact and deadline, then check documents already held. Preparing the request does not send an email." : prepared ? `Send the outstanding items to the prepared contact${audienceHint ? ` (${audienceHint})` : ""}. Documents already linked remain received.` : "The vendor contact receives access to this request only. The email address is cleared from this screen after the send attempt."}</p></div>
    <div className="vdd-form-grid">
      {prepared ? <p>Response due {formatDate(preparedDeadline)}</p> : <><div className="vdd-wide"><TextField label="Vendor contact email" type="email" inputMode="email" autoComplete="email" value={recipient} onChange={(value) => onRecipient(value)} isRequired/></div><TextField label="Response due date" type="date" value={responseDueDate} min={minimumDate} max={reviewDueAt.slice(0, 10)} onChange={(value) => onResponseDueDate(value)} isRequired/></>}
      {!preparing && <SelectField label="Secure link valid for" value={String(invitationMinutes)} placeholder="Choose link expiry" allowsEmpty={false} options={linkExpiryOptions} onChange={(value) => { if (value) onInvitationMinutes(Number(value)); }}/>}
    </div>
    <p className="vdd-limitation">Sending the request starts evidence collection. A submitted response still requires review.</p>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Cancel</Button><Button type="submit" variant="primary" isDisabled={busy}>{busy ? preparing ? "Preparing…" : "Sending…" : preparing ? "Prepare request" : "Send due diligence request"}</Button></div>
  </form>;
}

function ReissuePanel({ recipient, invitationMinutes, busy, onRecipient, onInvitationMinutes, onCancel, onSubmit }: { recipient: string; invitationMinutes: number; busy: boolean; onRecipient: (value: string) => void; onInvitationMinutes: (value: number) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Vendor request access</span><h3>Send another link</h3><p>Each link remains available until its printed expiry. Cancel the request when all active links must stop working.</p></div>
    <div className="vdd-form-grid">
      <div className="vdd-wide"><TextField label="Vendor contact email" type="email" inputMode="email" autoComplete="email" value={recipient} onChange={(value) => onRecipient(value)} isRequired/></div>
      <SelectField label="New link valid for" value={String(invitationMinutes)} placeholder="Choose link expiry" allowsEmpty={false} options={linkExpiryOptions} onChange={(value) => { if (value) onInvitationMinutes(Number(value)); }}/>
    </div>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Cancel</Button><Button type="submit" variant="primary" isDisabled={busy}>{busy ? "Sending…" : "Send another link"}</Button></div>
  </form>;
}

function ClarificationPanel({ fields, selected, message, recipient, dueDate, invitationMinutes, minimumDate, reviewDueAt, busy, onSelected, onMessage, onRecipient, onDueDate, onInvitationMinutes, onCancel, onSubmit }: { fields: { id: string; label: string }[]; selected: string[]; message: string; recipient: string; dueDate: string; invitationMinutes: number; minimumDate: string; reviewDueAt: string; busy: boolean; onSelected: (value: string[]) => void; onMessage: (value: string) => void; onRecipient: (value: string) => void; onDueDate: (value: string) => void; onInvitationMinutes: (value: number) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Vendor follow-up</span><h3>Request clarification</h3><p>Select only the submitted fields that the vendor must update or support.</p></div>
    {fields.length ? <fieldset className="vdd-fieldset"><legend>Fields requiring clarification</legend>{fields.map((field) => <CheckboxField key={field.id} label={field.label} isSelected={selected.includes(field.id)} onChange={(checked) => onSelected(checked ? [...selected, field.id] : selected.filter((id) => id !== field.id))}/>)}</fieldset> : <p className="vdd-limitation">No response fields are available for clarification. Reload the submitted response before creating a request.</p>}
    <TextArea label="What the vendor must provide" rows={4} maxLength={2000} value={message} onChange={(value) => onMessage(value)} isRequired/>
    <div className="vdd-form-grid">
      <div className="vdd-wide"><TextField label="Vendor contact email" type="email" inputMode="email" autoComplete="email" maxLength={254} value={recipient} onChange={(value) => onRecipient(value)} isRequired/></div>
      <TextField label="Response due date" type="date" min={minimumDate} max={reviewDueAt.slice(0, 10)} value={dueDate} onChange={(value) => onDueDate(value)} isRequired/>
      <SelectField label="Secure link valid for" value={String(invitationMinutes)} placeholder="Choose link expiry" allowsEmpty={false} options={linkExpiryOptions} onChange={(value) => { if (value) onInvitationMinutes(Number(value)); }}/>
    </div>
    <p className="vdd-limitation">The email address is used only for this secure invitation and is cleared from this screen after every attempt.</p>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Cancel</Button><Button type="submit" variant="primary" isDisabled={busy || fields.length === 0}>{busy ? "Creating request…" : "Send clarification request"}</Button></div>
  </form>;
}

function DeficiencyPanel({ triggerKey, title, summary, dueDate, minimumDate, busy, onTriggerKey, onTitle, onSummary, onDueDate, onCancel, onSubmit }: { triggerKey: string; title: string; summary: string; dueDate: string; minimumDate: string; busy: boolean; onTriggerKey: (value: string) => void; onTitle: (value: string) => void; onSummary: (value: string) => void; onDueDate: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Review finding</span><h3>Record a vendor finding</h3><p>Use one stable reference for the same evidence gap. Reusing the reference opens the existing finding instead of creating a duplicate.</p></div>
    <div className="vdd-form-grid">
      <TextField label="Finding reference" value={triggerKey} maxLength={80} pattern="[a-z0-9][a-z0-9._:-]{0,79}" placeholder="security-test-report" onChange={(value) => onTriggerKey(value.toLowerCase())} isRequired/>
      <TextField label="Action due date" type="date" min={minimumDate} value={dueDate} onChange={(value) => onDueDate(value)} isRequired/>
      <div className="vdd-wide"><TextField label="Finding title" value={title} maxLength={200} onChange={(value) => onTitle(value)} isRequired/></div>
      <div className="vdd-wide"><TextArea label="Finding details" rows={4} maxLength={2000} value={summary} onChange={(value) => onSummary(value)} isRequired/></div>
    </div>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Cancel</Button><Button type="submit" variant="primary" isDisabled={busy}>{busy ? "Recording…" : "Record finding"}</Button></div>
  </form>;
}

function DocumentDecisionPanel({ document, decision, documentType, evidenceClass, validUntil, busy, onDecision, onDocumentType, onEvidenceClass, onValidUntil, onCancel, onSubmit }: { document: VendorAssessmentDocument; decision: "VALIDATE" | "REJECT"; documentType: string; evidenceClass: ReviewVendorAssessmentDocumentInput["evidence_class"]; validUntil: string; busy: boolean; onDecision: (value: "VALIDATE" | "REJECT") => void; onDocumentType: (value: string) => void; onEvidenceClass: (value: ReviewVendorAssessmentDocumentInput["evidence_class"]) => void; onValidUntil: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Document decision</span><h3>Review document</h3><p>{document.file_name}</p></div>
    <div className="vdd-form-grid">
      <SelectField label="Decision" value={decision} placeholder="Choose a decision" isDisabled={busy} allowsEmpty={false} options={[{ id: "VALIDATE", label: "Accept" }, { id: "REJECT", label: "Reject" }]} onChange={(value) => { if (value) onDecision(value); }}/>
      <div className="vdd-wide"><TextField label="Document type" value={documentType} maxLength={128} onChange={(value) => onDocumentType(value)} isRequired/></div>
      <SelectField label="Evidence class" value={evidenceClass} placeholder="Choose evidence class" allowsEmpty={false} options={[{ id: "VENDOR_SUPPLIED", label: "Vendor supplied" }, { id: "BANK_VALIDATED", label: "Validated by reviewer" }, { id: "OFFICIAL_SOURCE", label: "Official source" }]} onChange={(value) => { if (value) onEvidenceClass(value); }}/>
      <TextField label="Valid until" type="date" value={validUntil} onChange={(value) => onValidUntil(value)}/>
    </div>
    <p className="vdd-limitation">{decision === "VALIDATE" ? "Validation confirms this document can support the current assessment. It does not approve the vendor relationship." : "Rejection records that this document cannot support the current assessment."}</p>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Cancel</Button><Button type="submit" variant="primary" isDisabled={busy}>{busy ? "Recording…" : decision === "VALIDATE" ? "Record validation" : "Record rejection"}</Button></div>
  </form>;
}

function ConclusionPanel({ conclusion, rationale, uncertainty, nextReviewDate, minimumDate, busy, onConclusion, onRationale, onUncertainty, onNextReviewDate, onCancel, onSubmit }: { conclusion: VendorAssessmentConclusion | ""; rationale: string; uncertainty: string; nextReviewDate: string; minimumDate: string; busy: boolean; onConclusion: (value: VendorAssessmentConclusion | "") => void; onRationale: (value: string) => void; onUncertainty: (value: string) => void; onNextReviewDate: (value: string) => void; onCancel: () => void; onSubmit: (event: React.FormEvent) => void }) {
  return <form className="vdd-panel" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">Reviewer conclusion</span><h3>Record assessment conclusion</h3><p>Base the conclusion on the submitted response, reviewed documents and recorded findings.</p></div>
    <div className="vdd-form-grid">
      <SelectField label="Conclusion" value={conclusion || undefined} placeholder="Select a conclusion" isRequired options={conclusionOptions.map((option) => ({ id: option.value, label: option.label }))} onChange={(value) => onConclusion(value ?? "")}/>
      <TextField label="Recommended next review" type="date" min={minimumDate} value={nextReviewDate} onChange={(value) => onNextReviewDate(value)}/>
      <div className="vdd-wide"><TextArea label="Assessment basis" rows={5} value={rationale} onChange={(value) => onRationale(value)} isRequired/></div>
      <div className="vdd-wide"><TextArea label="Remaining uncertainty" rows={3} value={uncertainty} onChange={(value) => onUncertainty(value)}/></div>
    </div>
    <p className="vdd-limitation">This conclusion completes the assessment review. It does not activate or approve the vendor relationship.</p>
    <div className="vdd-panel-actions"><Button type="button" variant="secondary" onPress={onCancel} isDisabled={busy}>Cancel</Button><Button type="submit" variant="primary" isDisabled={busy || !conclusion || !rationale.trim()}>{busy ? "Recording…" : "Record assessment conclusion"}</Button></div>
  </form>;
}

function ReviewSummary({ review, assessment, documents = review.documents, onOpenMatter, onOpenDocument, onReviewDocument, onCreateDeficiency }: { review: VendorAssessmentReviewView; assessment?: VendorAssessment | null; documents?: VendorAssessmentDocument[]; onOpenMatter?: (matterID: string) => void; onOpenDocument?: (assessmentID: string, requestID: string, artifactID: string) => void; onReviewDocument?: (document: VendorAssessmentDocument, decision: "VALIDATE" | "REJECT") => void; onCreateDeficiency?: () => void }) {
  const criticalResponses = new Map(review.provisional_score?.critical_failures?.map((failure) => [failure.field_id, failure.outcome]) ?? []);
  return <section className="vdd-review" aria-label="Vendor response review">
    <div className="vdd-review-header"><div><h3>Vendor response</h3>{review.response ? <p>Submitted {formatDate(review.response.submitted_at)} · {review.response.answer_count} {itemLabel(review.response.answer_count, "answer")} · {review.response.artifact_count} {itemLabel(review.response.artifact_count, "document")}</p> : <p>No submitted response summary is available.</p>}</div><div className="vdd-review-metrics"><span>{review.coverage.answered_required} of {review.coverage.required_fields} required answers received</span>{review.provisional_score?.score !== undefined && <span>Provisional score: {formatScore(review.provisional_score.score)} of 100 · Form version {review.assessment.form_template_version}</span>}</div></div>
    <div className="vdd-review-group"><h4>Submitted answers</h4>{review.answers.length ? <dl className="vdd-answer-list">{review.answers.map((answer) => <ReviewAnswer key={answer.field_id} answer={answer} criticalResponse={criticalResponses.get(answer.field_id)}/>)}</dl> : <p>No answers were submitted for this form version.</p>}</div>
    {documents.length > 0 && <div className="vdd-review-group"><h4>Supporting documents</h4>{documents.map((document) => <ReviewDocument key={`${document.field_id}:${document.artifact_id}`} document={document} assessment={assessment} requestID={document.request_id || review.response?.request_id} onOpenDocument={onOpenDocument} onReviewDocument={onReviewDocument}/>)}</div>}
    <div className="vdd-review-group"><div className="vdd-review-group-heading"><h4>Findings</h4>{assessment?.status === "UNDER_REVIEW" && onCreateDeficiency && <Button type="button" variant="secondary" onPress={onCreateDeficiency}>Record finding</Button>}</div>{review.matters.length ? <ul>{review.matters.map((finding) => <li key={finding.matter_id}><strong>{finding.title}</strong><span>{humanizeStatus(finding.status)}</span>{onOpenMatter && <Button type="button" variant="quiet" onPress={() => onOpenMatter(finding.matter_id)}>Open finding</Button>}</li>)}</ul> : <p>No findings are linked to this assessment.</p>}</div>
    {assessment?.status === "COMPLETED" && <div className="vdd-review-group"><h4>Recorded conclusion</h4><dl className="vdd-conclusion"><div><dt>Conclusion</dt><dd>{conclusionLabel(assessment.conclusion)}</dd></div><div><dt>Assessment basis</dt><dd>{assessment.conclusion_rationale || "No assessment basis was recorded."}</dd></div>{assessment.conclusion_uncertainty && <div><dt>Remaining uncertainty</dt><dd>{assessment.conclusion_uncertainty}</dd></div>}<div><dt>Completed</dt><dd>{formatDate(assessment.completed_at)}</dd></div></dl></div>}
  </section>;
}

function ReviewAnswer({ answer, criticalResponse }: { answer: VendorAssessmentReviewView["answers"][number]; criticalResponse?: string }) {
  const conditionalOmission = answer.visibility === "CONDITIONALLY_OMITTED";
  const sourceValue = answer.provenance?.source_value?.text;
  const linkedDocument = !answer.value ? answer.collection_resolution?.source.file_name : undefined;
  const missingRequired = answer.required && !answer.value && !linkedDocument;
  return <div>
    <dt>{answer.label}{answer.required ? " · Required" : ""}</dt>
    <dd className={conditionalOmission || missingRequired ? "vdd-answer-limitation" : undefined}>{conditionalOmission ? "Not requested because its condition was not met" : linkedDocument ? `Document linked: ${linkedDocument}` : missingRequired ? "Required response missing" : `Vendor response: ${formatAnswer(answer.value)}`}</dd>
    {sourceValue && <dd>Source value: {sourceValue}</dd>}
    {criticalResponse && <dd className="vdd-critical-response">Critical response: {humanizeResponseOutcome(criticalResponse)}</dd>}
    {answer.provenance && <dd className="vdd-provenance">{provenanceLabel(answer.provenance)}</dd>}
    {answer.provenance?.validations?.map((validation, index) => <dd className="vdd-validation" key={`${answer.field_id}-validation-${index}`}>{validationLabel(validation)}</dd>)}
  </div>;
}

function ReviewDocument({ document, assessment, requestID, onOpenDocument, onReviewDocument }: { document: VendorAssessmentDocument; assessment?: VendorAssessment | null; requestID?: string; onOpenDocument?: (assessmentID: string, requestID: string, artifactID: string) => void; onReviewDocument?: (document: VendorAssessmentDocument, decision: "VALIDATE" | "REJECT") => void }) {
  const actionable = documentIsActionable(document);
  const unavailableReason = actionable && requestID ? "" : documentRecovery(document, requestID);
  return <article className="vdd-document" aria-label={document.file_name}>
    <div>
      <strong>{document.file_name}</strong>
      <span>{document.document_type.replaceAll("_", " ")} · {formatBytes(document.size_bytes)}{document.expires_on ? ` · Expires ${formatDate(document.expires_on)}` : ""}</span>
      <span className="vdd-evidence-status">{artifactStatusLabel(document.artifact_status)} · {evidenceClassLabel(document.evidence_class)}</span>
      {document.status && <span className="vdd-document-decision">{documentDecisionLabel(document.status)} · {evidenceClassLabel(document.evidence_class)}</span>}
      {unavailableReason && <span className="vdd-document-recovery">{unavailableReason}</span>}
    </div>
    {assessment?.status === "UNDER_REVIEW" && (onOpenDocument || onReviewDocument) && <div>
      {onOpenDocument && <Button type="button" variant="quiet" onPress={() => requestID && onOpenDocument(assessment.id, requestID, document.artifact_id)} isDisabled={!actionable || !requestID}>Open document</Button>}
      {onReviewDocument && <Button type="button" variant="quiet" onPress={() => onReviewDocument(document, "VALIDATE")} isDisabled={!actionable}>Validate document</Button>}
      {onReviewDocument && <Button type="button" variant="quiet" onPress={() => onReviewDocument(document, "REJECT")} isDisabled={!actionable}>Reject document</Button>}
    </div>}
  </article>;
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

function validationLabel(validation: NonNullable<NonNullable<VendorAssessmentReviewView["answers"][number]["provenance"]>["validations"]>[number]) {
  const state = validation.state ?? "UNKNOWN";
  const status = state === "CURRENT" ? "Validation current"
    : state === "STALE" ? "Validation out of date"
      : state === "PARTIAL" ? "Validation incomplete"
        : state === "NOT_FOUND" ? "Source did not confirm this value"
          : state === "AMBIGUOUS" ? "Source returned more than one match"
            : state === "SCHEMA_DRIFT" ? "Validation mapping changed"
              : state === "UNAVAILABLE" ? "Validation source unavailable"
                : state === "INVALID" ? "Source did not validate this value"
                  : `Validation result: ${humanizeStatus(state)}`;
  const source = validation.binding_name || validation.source_id;
  const checked = validation.receipt?.observed_at ? `Checked ${formatDate(validation.receipt.observed_at)}` : "";
  return [status, source, checked].filter(Boolean).join(" · ");
}

function documentIsActionable(document: VendorAssessmentDocument) {
  return document.artifact_status === "AVAILABLE" && document.status !== "REJECTED" && document.status !== "EXPIRED";
}

function documentRecovery(document: VendorAssessmentDocument, requestID?: string) {
  if (!requestID) return "The submitted request could not be confirmed. Reload the vendor response before opening this document.";
  if (document.status === "REJECTED") return "This document was rejected. Request a replacement before reviewing it again.";
  if (document.status === "EXPIRED") return "This document has expired. Request a current document before using it in the review.";
  switch (document.artifact_status) {
    case "STORED_UNSCANNED": return "The security scan is pending. Open and review actions will be available after the scan completes.";
    case "QUARANTINED": return "This document is quarantined. Wait for a clean replacement before reviewing it.";
    case "DELETED": return "This document is unavailable. Request a replacement before continuing the review.";
    case "AVAILABLE": return "";
    default: return "This document is unavailable for review. Reload the response or request a replacement.";
  }
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
    case "BANK_VALIDATED": return "Validated evidence";
    case "OFFICIAL_SOURCE": return "Official source evidence";
    default: return humanizeStatus(value);
  }
}

function documentDecisionLabel(value: string) {
  switch (value) {
    case "SUBMITTED": return "Awaiting document decision";
    case "VALIDATED": return "Validated";
    case "REJECTED": return "Rejected";
    case "EXPIRED": return "Expired";
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

function humanizeResponseOutcome(value: string) {
  return value.includes("_") || value === value.toUpperCase() ? humanizeStatus(value) : value;
}

function assessmentStatusCopy(assessment?: VendorAssessment | null, setupFailure?: string) {
  if (!assessment) return { label: "Not started", tone: "neutral", description: "No due diligence review has been started for this vendor relationship." };
  switch (assessment.status) {
    case "SETUP_PENDING": return setupFailure
      ? { label: "Setup failed", tone: "danger", description: "The assessment exists, but its review record still needs to be prepared." }
      : { label: "Setup in progress", tone: "pending", description: "The assessment exists while its review record is being prepared." };
    case "READY_TO_SEND": return { label: "Ready to send", tone: "information", description: "The assessment is ready for the vendor contact and response deadline." };
    case "COLLECTING": return { label: "Awaiting response", tone: "pending", description: "The vendor request is open. Submission will not complete the review." };
    case "SUBMITTED": return { label: "Response received", tone: "information", description: "The vendor response was submitted and now requires review." };
    case "UNDER_REVIEW": return { label: "Under review", tone: "pending", description: "Review the response, documents and findings before recording a conclusion." };
    case "COMPLETED": return { label: conclusionLabel(assessment.conclusion), tone: assessment.conclusion === "UNSATISFACTORY" ? "danger" : "complete", description: "The assessment conclusion is recorded. The vendor relationship status remains separate." };
    case "CANCELLED": return assessment.review_kind === "ONBOARDING"
      ? { label: "Cancelled", tone: "neutral", description: "This onboarding assessment was cancelled. The relationship remains available for a new review." }
      : { label: "Cancelled", tone: "neutral", description: "This reassessment was cancelled. Start a new review with a new review reference when required." };
  }
}

function dueDiligenceTone(tone: string): StatusTone {
  if (tone === "complete") return "success";
  if (tone === "pending") return "warning";
  if (tone === "information") return "info";
  if (tone === "danger") return "error";
  return "neutral";
}

function assessmentStartMode(relationshipStatus: VendorRelationshipAggregate["relationship"]["status"], assessment?: VendorAssessment | null): AssessmentStartMode | null {
  const terminal = !assessment || assessment.status === "COMPLETED" || assessment.status === "CANCELLED";
  if (!terminal) return null;
  if (["ACTIVE", "RESTRICTED", "SUSPENDED"].includes(relationshipStatus)) return "reassessment";
  if (assessment?.status === "CANCELLED" && assessment.review_kind === "ONBOARDING" && ["PROPOSED", "UNDER_REVIEW"].includes(relationshipStatus)) return "restart";
  if (!assessment && ["PROPOSED", "UNDER_REVIEW"].includes(relationshipStatus)) return "initial";
  return null;
}

function startActionLabel(mode: AssessmentStartMode) {
  if (mode === "reassessment") return "Start reassessment";
  if (mode === "restart") return "Restart due diligence";
  return "Start due diligence";
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
