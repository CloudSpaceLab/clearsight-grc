import { useEffect, useMemo, useRef, useState } from "react";
import { apiErrorKind } from "../http";
import { loadFormTemplates } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { cancelVendorAssessment, completeVendorAssessment, createVendorAssessmentDeficiency, loadCurrentVendorAssessment, loadVendorAssessment, reissueVendorAssessmentRequest, requestVendorAssessmentClarification, retryVendorAssessmentSetup, reviewVendorAssessmentDocument, sendVendorAssessmentRequest, startVendorAssessment, startVendorAssessmentReview, vendorAssessmentDocumentURL } from "../vendorAssessmentApi";
import type { CompleteVendorAssessmentInput, CreateVendorAssessmentDeficiencyInput, CurrentVendorAssessment, ReviewVendorAssessmentDocumentInput, StartVendorAssessmentInput, VendorAssessment, VendorAssessmentClarificationInput, VendorAssessmentFormOption, VendorAssessmentReviewView, VendorAssessmentSendOutcome, VendorAssessmentSetupRetryOutcome } from "../vendorAssessmentTypes";
import { createVendorRelationship, loadVendorRelationship, loadVendorRelationships, updateVendorRelationship } from "../vendorApi";
import type { CreateVendorRelationshipInput, VendorCriticality, VendorPrivacyRole, VendorRelationshipAggregate } from "../vendorTypes";
import { VendorDueDiligence } from "./VendorDueDiligence";
import { VendorWorkPanel } from "./VendorWorkPanel";

type Props = {
  organizationName: string;
  legalEntityName: string;
  targetID?: string;
  guideIntent?: { id: number; type: "open-vendor-due-diligence" | "open-vendor-work" | "open-vendor-next-action" };
  onGuideIntentCompleted?: (id: number) => void;
  onGuideIntentFailed?: (id: number) => void;
  onTarget?: (id?: string) => void;
  onOpenRequest?: (requestID: string) => void;
  onOpenMatter?: (matterID: string) => void;
};

type LoadState = "loading" | "live" | "unavailable";

type FormValues = {
  legalName: string;
  tradingName: string;
  registrationRef: string;
  jurisdiction: string;
  serviceName: string;
  criticality: VendorCriticality;
  privacyRole: VendorPrivacyRole;
  sourceID: string;
  externalRef: string;
  effectiveFrom: string;
  renewalAt: string;
};

const emptyForm: FormValues = {
  legalName: "", tradingName: "", registrationRef: "", jurisdiction: "", serviceName: "",
  criticality: "STANDARD", privacyRole: "NONE", sourceID: "", externalRef: "", effectiveFrom: "", renewalAt: "",
};

function focusGuideTarget(target: HTMLElement | null) {
  if (!target) return false;
  target.scrollIntoView?.({ behavior: "smooth", block: "center" });
  if (!target.hasAttribute("tabindex") && !(target instanceof HTMLButtonElement) && !(target instanceof HTMLInputElement)) target.setAttribute("tabindex", "-1");
  target.focus({ preventScroll: true });
  return document.activeElement === target;
}

function firstVisiblePrimaryAction(selector: string) {
  return [...document.querySelectorAll<HTMLElement>(`${selector} button.primary-button:not(:disabled)`)].find((element) => !element.hidden && !element.closest("[hidden], [aria-hidden='true']")) ?? null;
}

export function VendorsWorkspace({ organizationName, legalEntityName, targetID, guideIntent, onGuideIntentCompleted, onGuideIntentFailed, onTarget, onOpenRequest, onOpenMatter }: Props) {
  const [records, setRecords] = useState<VendorRelationshipAggregate[]>([]);
  const [selected, setSelected] = useState<VendorRelationshipAggregate | null>(null);
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [mode, setMode] = useState<"browse" | "create" | "edit">("browse");
  const [query, setQuery] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [nextCursor, setNextCursor] = useState("");
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreError, setLoadMoreError] = useState(false);
  const [form, setForm] = useState<FormValues>(emptyForm);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [notice, setNotice] = useState("");
  const [formError, setFormError] = useState("");
  const [saving, setSaving] = useState(false);
  const [existingVendorSource, setExistingVendorSource] = useState<VendorRelationshipAggregate>();
  const [vendorCandidates, setVendorCandidates] = useState<VendorRelationshipAggregate[]>([]);
  const [candidateState, setCandidateState] = useState<"idle" | "loading" | "ready" | "failed">("idle");
  const [forms, setForms] = useState<FormTemplate[]>([]);
  const [formState, setFormState] = useState<LoadState>("loading");
  const [assessment, setAssessment] = useState<VendorAssessment | null>(null);
  const [assessmentSetup, setAssessmentSetup] = useState<CurrentVendorAssessment["setup"]>();
  const [assessmentState, setAssessmentState] = useState<LoadState>("loading");
  const [review, setReview] = useState<VendorAssessmentReviewView>();
  const [reviewState, setReviewState] = useState<LoadState>("loading");
  const [requestOutcome, setRequestOutcome] = useState<VendorAssessmentSendOutcome>();
  const [requestOutcomeKind, setRequestOutcomeKind] = useState<"initial" | "replacement">("initial");
  const assessmentLoadID = useRef(0);
  const formLoadID = useRef(0);
  const registerLoadID = useRef(0);
  const acknowledgedGuideIntentID = useRef<number | undefined>(undefined);

  useEffect(() => {
    setQuery("");
    setSubmittedQuery("");
    void refresh(targetID, "");
  }, [targetID]);

  useEffect(() => {
    if (guideIntent) void refresh(targetID, "", guideIntent);
  }, [guideIntent]);

  useEffect(() => {
    if (!guideIntent || state !== "live") return;
    if (acknowledgedGuideIntentID.current === guideIntent.id) return;
    if (!selected) {
      if (mode === "create") {
        const form = document.getElementById("vendor-legal-name");
        if (focusGuideTarget(form as HTMLElement | null)) acknowledgeGuideIntent(guideIntent.id);
      }
      return;
    }
    if ((guideIntent.type === "open-vendor-due-diligence" || guideIntent.type === "open-vendor-next-action") && (assessmentState === "loading" || reviewState === "loading" || (!assessment && formState === "loading"))) return;
    if (guideIntent.type === "open-vendor-next-action") {
      const workspace = document.querySelector<HTMLElement>(".vendors-workspace");
      if (!workspace) return;
      const focusNextAction = () => {
        const dueDiligenceAction = assessment?.status === "COMPLETED" ? null : firstVisiblePrimaryAction(".vdd-workspace");
        if (dueDiligenceAction && focusGuideTarget(dueDiligenceAction)) return acknowledgeGuideIntent(guideIntent.id);
        const vendorWork = workspace.querySelector<HTMLElement>(".vendor-work-panel");
        if (!vendorWork || vendorWork.getAttribute("aria-busy") === "true") return false;
        const vendorWorkAction = firstVisiblePrimaryAction(".vendor-work-panel");
        if (vendorWorkAction && focusGuideTarget(vendorWorkAction)) return acknowledgeGuideIntent(guideIntent.id);
        if (focusGuideTarget(workspace)) return acknowledgeGuideIntent(guideIntent.id);
        return false;
      };
      if (focusNextAction()) return;
      const observer = new MutationObserver(() => { if (focusNextAction()) observer.disconnect(); });
      observer.observe(workspace, { childList: true, subtree: true, attributes: true, attributeFilter: ["aria-busy", "disabled", "hidden"] });
      return () => observer.disconnect();
    }
    const target = guideIntent.type === "open-vendor-due-diligence"
      ? document.getElementById("vdd-title") ?? document.querySelector<HTMLElement>(".vdd-workspace")
      : document.querySelector<HTMLElement>(".vendor-work-panel") ?? document.querySelector<HTMLElement>(".vendors-workspace");
    if (!target) return;
    if (focusGuideTarget(target)) acknowledgeGuideIntent(guideIntent.id);

    function acknowledgeGuideIntent(id: number) {
      acknowledgedGuideIntentID.current = id;
      onGuideIntentCompleted?.(id);
      return true;
    }
  }, [guideIntent, state, mode, selected?.relationship.id, assessment, assessmentState, reviewState, formState, onGuideIntentCompleted]);

  useEffect(() => {
    void refreshForms();
    return () => { formLoadID.current += 1; };
  }, []);

  async function refreshForms() {
    const loadID = ++formLoadID.current;
    setFormState("loading");
    try {
      const values = await loadFormTemplates();
      if (loadID !== formLoadID.current) return;
      setForms(values);
      setFormState("live");
    } catch {
      if (loadID !== formLoadID.current) return;
      setForms([]);
      setFormState("unavailable");
    }
  }

  useEffect(() => {
    if (!selected) {
      assessmentLoadID.current += 1;
      setAssessment(null);
      setAssessmentSetup(undefined);
      setAssessmentState("loading");
      setReview(undefined);
      setReviewState("loading");
      setRequestOutcome(undefined);
      setRequestOutcomeKind("initial");
      return;
    }
    void refreshAssessment(selected.relationship.id);
  }, [selected?.relationship.id]);

  const activeVendorForm = useMemo(() => selectActiveVendorForm(forms), [forms]);

  async function refreshAssessment(relationshipID: string) {
    const loadID = ++assessmentLoadID.current;
    setAssessmentState("loading");
    setRequestOutcome(undefined);
    setRequestOutcomeKind("initial");
    setReview(undefined);
    setReviewState("loading");
    try {
      const current = await loadCurrentVendorAssessment(relationshipID);
      if (loadID !== assessmentLoadID.current) return;
      setAssessment(current.assessment);
      setAssessmentSetup(current.setup);
      setAssessmentState("live");
      if (current.assessment && needsReviewView(current.assessment.status)) {
        try {
          const value = await loadVendorAssessment(current.assessment.id);
          if (loadID !== assessmentLoadID.current) return;
          setReview(value);
          setReviewState("live");
        } catch {
          if (loadID !== assessmentLoadID.current) return;
          setReview(undefined);
          setReviewState("unavailable");
        }
      } else {
        setReviewState("live");
      }
    } catch (error) {
      if (loadID !== assessmentLoadID.current) return;
      if (apiErrorKind(error) === "not_found") {
        setAssessment(null);
        setAssessmentSetup(undefined);
        setAssessmentState("live");
        setReviewState("live");
      } else {
        setAssessment(null);
        setAssessmentSetup(undefined);
        setAssessmentState("unavailable");
        setReviewState("unavailable");
      }
    }
  }

  async function startAssessment(input: StartVendorAssessmentInput) {
    if (!selected) return;
    const value = await startVendorAssessment(selected.relationship.id, input);
    setAssessment(value);
    setAssessmentSetup(undefined);
    setAssessmentState("live");
    setReview(undefined);
    setReviewState("live");
    return value;
  }

  async function sendAssessmentRequest(input: Parameters<typeof sendVendorAssessmentRequest>[1]) {
    if (!assessment) throw new Error("No current assessment");
    const outcome = await sendVendorAssessmentRequest(assessment.id, input);
    setAssessment(outcome.assessment);
    setRequestOutcome(outcome);
    setRequestOutcomeKind("initial");
    return outcome;
  }

  async function reissueAssessmentRequest(input: Parameters<typeof reissueVendorAssessmentRequest>[1]) {
    if (!assessment) throw new Error("No current assessment");
    const outcome = await reissueVendorAssessmentRequest(assessment.id, input);
    setAssessment(outcome.assessment);
    setRequestOutcome(outcome);
    setRequestOutcomeKind("replacement");
    return outcome;
  }

  async function retryAssessmentSetup(assessmentID: string, expectedVersion: number): Promise<VendorAssessmentSetupRetryOutcome> {
    const outcome = await retryVendorAssessmentSetup(assessmentID, { expected_version: expectedVersion });
    setAssessment(outcome.assessment);
    setAssessmentSetup(outcome.setup);
    setAssessmentState("live");
    return outcome;
  }

  async function refreshReview(assessmentID: string) {
    const loadID = ++assessmentLoadID.current;
    setReviewState("loading");
    try {
      const value = await loadVendorAssessment(assessmentID);
      if (loadID !== assessmentLoadID.current) return;
      setReview(value);
      setAssessment(value.assessment);
      setReviewState("live");
    } catch {
      if (loadID !== assessmentLoadID.current) return;
      setReview(undefined);
      setReviewState("unavailable");
    }
  }

  async function beginAssessmentReview(assessmentID: string, expectedVersion: number) {
    const value = await startVendorAssessmentReview(assessmentID, { expected_version: expectedVersion });
    setAssessment(value);
    setReview((current) => current ? { ...current, assessment: value } : current);
    return value;
  }

  async function requestAssessmentClarification(assessmentID: string, input: VendorAssessmentClarificationInput) {
    const outcome = await requestVendorAssessmentClarification(assessmentID, input);
    setAssessment(outcome.assessment);
    setReview((current) => current ? { ...current, assessment: outcome.assessment } : current);
    return outcome;
  }

  async function recordAssessmentDeficiency(assessmentID: string, input: CreateVendorAssessmentDeficiencyInput) {
    const outcome = await createVendorAssessmentDeficiency(assessmentID, input);
    setAssessment(outcome.assessment);
    try {
      const refreshed = await loadVendorAssessment(assessmentID);
      setReview(refreshed);
      setAssessment(refreshed.assessment);
      setReviewState("live");
    } catch {
      setReview(undefined);
      setReviewState("unavailable");
    }
    return outcome;
  }

  async function decideAssessmentDocument(assessmentID: string, artifactID: string, input: ReviewVendorAssessmentDocumentInput) {
    const refreshed = await reviewVendorAssessmentDocument(assessmentID, artifactID, input);
    setReview(refreshed);
    setAssessment(refreshed.assessment);
    setReviewState("live");
    return refreshed;
  }

  async function finishAssessmentReview(assessmentID: string, input: CompleteVendorAssessmentInput) {
    const value = await completeVendorAssessment(assessmentID, input);
    setAssessment(value);
    setReview((current) => current ? { ...current, assessment: value } : current);
    return value;
  }

  async function cancelAssessment(assessmentID: string, input: Parameters<typeof cancelVendorAssessment>[1]) {
    const value = await cancelVendorAssessment(assessmentID, input);
    setAssessment(value);
    setReview((current) => current ? { ...current, assessment: value } : current);
    return value;
  }

  async function refresh(requestedID?: string, search = submittedQuery, intent?: Props["guideIntent"]) {
    const loadID = ++registerLoadID.current;
    setState("loading");
    setLoadMoreError(false);
    try {
      const page = await loadVendorRelationships({ ...(search ? { search } : {}), limit: 50 });
      if (loadID !== registerLoadID.current) return;
      let next = page.items ?? [];
      let exact = requestedID ? next.find((item) => item.relationship.id === requestedID) : undefined;
      if (requestedID && !exact) {
        exact = await loadVendorRelationship(requestedID);
        if (loadID !== registerLoadID.current) return;
        next = [exact, ...next];
      }
      setRecords(next);
      setNextCursor(page.next_cursor ?? "");
      const preserved = selected ? next.find((item) => item.relationship.id === selected.relationship.id) : undefined;
      const nextSelected = exact ?? preserved ?? (intent ? next[0] : undefined);
      setSelected(nextSelected ?? null);
      if (nextSelected) setMode("browse");
      if (intent && next.length === 0) setMode("create");
      setState("live");
    } catch {
      if (loadID !== registerLoadID.current) return;
      setRecords([]);
      setSelected(null);
      setState("unavailable");
      if (intent) onGuideIntentFailed?.(intent.id);
    }
  }

  async function loadMoreRelationships() {
    if (!nextCursor || loadingMore) return;
    setLoadingMore(true);
    setLoadMoreError(false);
    try {
      const page = await loadVendorRelationships({ ...(submittedQuery ? { search: submittedQuery } : {}), cursor: nextCursor, limit: 50 });
      setRecords((current) => page.items.reduce((items, item) => items.some((existing) => existing.relationship.id === item.relationship.id) ? items : [...items, item], current));
      setNextCursor(page.next_cursor ?? "");
    } catch {
      setLoadMoreError(true);
    } finally {
      setLoadingMore(false);
    }
  }

  function searchRelationships(event: React.FormEvent) {
    event.preventDefault();
    const search = query.trim();
    setSubmittedQuery(search);
    setSelected(null);
    onTarget?.();
    void refresh(undefined, search);
  }

  function choose(record: VendorRelationshipAggregate) {
    setSelected(record); setMode("browse"); setNotice(""); setFormError(""); onTarget?.(record.relationship.id);
  }

  function startCreate() {
    setMode("create"); setForm(emptyForm); setSelected(null); setExistingVendorSource(undefined); setVendorCandidates([]); setCandidateState("idle"); setFieldErrors({}); setFormError(""); setNotice(""); onTarget?.();
  }

  function startEdit() {
    if (!selected) return;
    const { vendor, relationship } = selected;
    setForm({
      legalName: vendor.legal_name, tradingName: vendor.trading_name ?? "", registrationRef: vendor.registration_ref ?? "", jurisdiction: vendor.jurisdiction ?? "",
      serviceName: relationship.service_name, criticality: relationship.criticality, privacyRole: relationship.privacy_role,
      sourceID: vendor.source_id ?? "", externalRef: vendor.external_ref ?? "", effectiveFrom: dateInput(relationship.effective_from), renewalAt: dateInput(relationship.renewal_at),
    });
    setMode("edit"); setFieldErrors({}); setFormError(""); setNotice("");
  }

  function cancelForm() {
    setMode("browse"); setExistingVendorSource(undefined); setVendorCandidates([]); setCandidateState("idle"); setFieldErrors({}); setFormError("");
  }

  function setValue<K extends keyof FormValues>(key: K, value: FormValues[K]) {
    setForm((current) => ({ ...current, [key]: value }));
    if (key === "legalName" && existingVendorSource) setExistingVendorSource(undefined);
    setFieldErrors((current) => ({ ...current, [key]: "" }));
  }

  async function findExistingVendor() {
    const search = form.legalName.trim();
    if (search.length < 2) return;
    setCandidateState("loading");
    try {
      const page = await loadVendorRelationships({ search, limit: 10 });
      setVendorCandidates(page.items);
      setCandidateState("ready");
    } catch {
      setVendorCandidates([]);
      setCandidateState("failed");
    }
  }

  function useExistingVendor(candidate: VendorRelationshipAggregate) {
    setExistingVendorSource(candidate);
    setForm((current) => ({ ...current, legalName: candidate.vendor.legal_name, tradingName: candidate.vendor.trading_name ?? "", registrationRef: candidate.vendor.registration_ref ?? "", jurisdiction: candidate.vendor.jurisdiction ?? "", sourceID: "", externalRef: "" }));
    setCandidateState("idle");
    setVendorCandidates([]);
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const errors: Record<string, string> = {};
    if (mode === "create" && !form.legalName.trim()) errors.legalName = "Enter the vendor's legal name.";
    if (!form.serviceName.trim()) errors.serviceName = "Enter the service supplied to this legal entity.";
    if ((form.sourceID.trim() && !form.externalRef.trim()) || (!form.sourceID.trim() && form.externalRef.trim())) errors.sourceID = "Enter both source system and source reference, or leave both blank.";
    if (form.effectiveFrom && form.renewalAt && form.renewalAt < form.effectiveFrom) errors.renewalAt = "Renewal date cannot be before the effective date.";
    setFieldErrors(errors);
    if (Object.values(errors).some(Boolean)) return;

    const relationshipInput = {
      service_name: form.serviceName.trim(), criticality: form.criticality, privacy_role: form.privacyRole,
      effective_from: apiDate(form.effectiveFrom), renewal_at: apiDate(form.renewalAt),
    };
    setSaving(true); setFormError(""); setNotice("");
    try {
      let saved: VendorRelationshipAggregate;
      if (mode === "edit" && selected) {
        saved = await updateVendorRelationship(selected.relationship.id, { ...relationshipInput, expected_version: selected.relationship.version });
        setNotice("Vendor relationship updated.");
      } else {
        const input: CreateVendorRelationshipInput = {
          ...relationshipInput, existing_relationship_id: existingVendorSource?.relationship.id, legal_name: form.legalName.trim(), trading_name: optional(form.tradingName),
          registration_ref: optional(form.registrationRef), jurisdiction: optional(form.jurisdiction),
          source_id: optional(form.sourceID), external_ref: optional(form.externalRef),
        };
        saved = await createVendorRelationship(input);
        setNotice("Vendor relationship added.");
      }
      setRecords((current) => [saved, ...current.filter((item) => item.relationship.id !== saved.relationship.id)]);
      setSelected(saved); setMode("browse"); onTarget?.(saved.relationship.id);
    } catch (error) {
      const kind = apiErrorKind(error);
      if (kind === "conflict") setFormError("This record changed. Your entries are still here; reload the record before saving again.");
      else if (kind === "validation") setFormError("Check the required vendor and service fields. Your entries are still here.");
      else if (kind === "forbidden" || kind === "unauthorized") setFormError("Your current role cannot make this vendor change. Your entries are still here.");
      else setFormError("The vendor change could not be saved. Your entries are still here; try again.");
    } finally {
      setSaving(false);
    }
  }

  const workspaceClass = `vendors-workspace${mode !== "browse" ? " is-form" : selected ? " has-selection" : ""}`;
  return <div className={workspaceClass} tabIndex={-1}>
    <header className="topbar vendors-topbar">
      <div><span className="eyebrow">{organizationName} · {legalEntityName}</span><h1>Vendors</h1><p>Manage vendors and the services they supply to {legalEntityName}. Review each relationship&apos;s owner, criticality and due-diligence status.</p></div>
      {mode !== "create" && <button type="button" className={selected ? "secondary-button" : "primary-button"} onClick={startCreate}>Add vendor</button>}
    </header>

    {notice && <p className="vendor-notice" role="status">{notice}</p>}
    {state === "loading" && <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading vendor relationships for {legalEntityName}…</div>}
    {state === "unavailable" && <section className="vendor-state" role="alert"><h2>Vendor records are unavailable</h2><p>The vendor register for {legalEntityName} could not be loaded. Try again before adding or changing a record.</p><button className="secondary-button" type="button" onClick={() => void refresh(targetID, "")}>Try again</button></section>}
    {state === "live" && <div className="vendor-layout">
      <section className="vendor-register" aria-label={`Vendor relationships for ${legalEntityName}`}>
        <div className="vendor-register-header"><div><h2>Vendor register</h2><p>{submittedQuery ? `Showing ${records.length} matching ${records.length === 1 ? "relationship" : "relationships"}` : `Showing ${records.length} ${records.length === 1 ? "relationship" : "relationships"} in this legal entity`}</p>{nextCursor && <small>More relationships are available.</small>}</div></div>
        <form className="vendor-search" onSubmit={searchRelationships}><label><span>Search vendors and services</span><input type="search" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Name, service or reference"/></label><button type="submit" className="secondary-button">Search vendors</button></form>
        {records.length > 0 ? <div className="vendor-list">{records.map((record) => <button type="button" key={record.relationship.id} aria-label={`${record.vendor.legal_name}, ${record.relationship.service_name}`} aria-current={selected?.relationship.id === record.relationship.id ? "true" : undefined} className={selected?.relationship.id === record.relationship.id ? "vendor-row selected" : "vendor-row"} onClick={() => choose(record)}>
          <span className="vendor-row-icon" aria-hidden="true">V</span><span className="vendor-row-main"><strong>{record.vendor.legal_name}</strong><span>Service: {record.relationship.service_name}</span></span><span className={`vendor-criticality criticality-${record.relationship.criticality.toLowerCase()}`}>{humanize(record.relationship.criticality)}</span>
        </button>)}</div> : submittedQuery ? <div className="vendor-empty"><h3>No vendor relationships match this search.</h3><p>No legal name, service, registration or source reference matched “{submittedQuery}” in {legalEntityName}.</p><button type="button" className="secondary-button" onClick={() => { setQuery(""); setSubmittedQuery(""); void refresh(undefined, ""); }}>Clear search</button></div> : <div className="vendor-empty"><h3>No vendor relationships found for {legalEntityName}.</h3><p>Add the first vendor and the service it supplies. Use <strong>Add vendor</strong> above; the signed-in actor becomes the initial accountable owner.</p></div>}
        {nextCursor && <button type="button" className="secondary-button" disabled={loadingMore} onClick={() => void loadMoreRelationships()}>{loadingMore ? "Loading…" : "Load more vendors"}</button>}
        {loadMoreError && <p role="alert" className="inline-error">More vendor relationships could not be loaded. The current results remain available.</p>}
      </section>

      <section className="vendor-focus" aria-label="Selected vendor relationship">
        {(mode === "create" || mode === "edit") ? <VendorForm mode={mode} form={form} errors={fieldErrors} formError={formError} saving={saving} existingVendor={existingVendorSource} candidates={vendorCandidates} candidateState={candidateState} onFindExisting={findExistingVendor} onUseExisting={useExistingVendor} onUseDifferent={() => { setExistingVendorSource(undefined); setForm((current) => ({ ...current, legalName: "", tradingName: "", registrationRef: "", jurisdiction: "" })); }} onChange={setValue} onCancel={cancelForm} onSubmit={submit}/> : selected ? <VendorDetail
          record={selected}
          assessment={assessment}
          assessmentSetup={assessmentSetup}
          assessmentState={assessmentState}
          review={review}
          reviewState={reviewState}
          form={activeVendorForm}
          formState={formState}
          requestOutcome={requestOutcome}
          requestOutcomeKind={requestOutcomeKind}
          onBack={() => { setSelected(null); onTarget?.(); }}
          onEdit={startEdit}
          onRefreshAssessment={() => refreshAssessment(selected.relationship.id)}
          onRefreshForms={refreshForms}
          onStartAssessment={startAssessment}
          onSendAssessmentRequest={sendAssessmentRequest}
          onReissueAssessmentRequest={reissueAssessmentRequest}
          onRetryAssessmentSetup={retryAssessmentSetup}
          onRefreshReview={refreshReview}
          onStartAssessmentReview={beginAssessmentReview}
          onRequestAssessmentClarification={requestAssessmentClarification}
          onCreateAssessmentDeficiency={recordAssessmentDeficiency}
          onReviewAssessmentDocument={decideAssessmentDocument}
          onCompleteAssessmentReview={finishAssessmentReview}
          onCancelAssessment={cancelAssessment}
          onOpenRequest={onOpenRequest}
          onOpenMatter={onOpenMatter}
        /> : records.length > 0 ? <div className="vendor-selection"><h2>Select a vendor</h2><p>Choose a relationship to review its service, accountable owner, source and current record version.</p></div> : null}
      </section>
    </div>}
  </div>;
}

function VendorDetail({ record, assessment, assessmentSetup, assessmentState, review, reviewState, form, formState, requestOutcome, requestOutcomeKind, onBack, onEdit, onRefreshAssessment, onRefreshForms, onStartAssessment, onSendAssessmentRequest, onReissueAssessmentRequest, onRetryAssessmentSetup, onRefreshReview, onStartAssessmentReview, onRequestAssessmentClarification, onCreateAssessmentDeficiency, onReviewAssessmentDocument, onCompleteAssessmentReview, onCancelAssessment, onOpenRequest, onOpenMatter }: {
  record: VendorRelationshipAggregate;
  assessment: VendorAssessment | null;
  assessmentSetup?: CurrentVendorAssessment["setup"];
  assessmentState: LoadState;
  review?: VendorAssessmentReviewView;
  reviewState: LoadState;
  form?: VendorAssessmentFormOption;
  formState: LoadState;
  requestOutcome?: VendorAssessmentSendOutcome;
  requestOutcomeKind: "initial" | "replacement";
  onBack: () => void;
  onEdit: () => void;
  onRefreshAssessment: () => Promise<void>;
  onRefreshForms: () => Promise<void>;
  onStartAssessment: (input: StartVendorAssessmentInput) => Promise<VendorAssessment | void>;
  onSendAssessmentRequest: (input: Parameters<typeof sendVendorAssessmentRequest>[1]) => Promise<VendorAssessmentSendOutcome>;
  onReissueAssessmentRequest: (input: Parameters<typeof reissueVendorAssessmentRequest>[1]) => Promise<VendorAssessmentSendOutcome>;
  onRetryAssessmentSetup: (assessmentID: string, expectedVersion: number) => Promise<VendorAssessmentSetupRetryOutcome>;
  onRefreshReview: (assessmentID: string) => Promise<void>;
  onStartAssessmentReview: (assessmentID: string, expectedVersion: number) => Promise<VendorAssessment>;
  onRequestAssessmentClarification: typeof requestVendorAssessmentClarification;
  onCreateAssessmentDeficiency: typeof createVendorAssessmentDeficiency;
  onReviewAssessmentDocument: typeof reviewVendorAssessmentDocument;
  onCompleteAssessmentReview: (assessmentID: string, input: CompleteVendorAssessmentInput) => Promise<VendorAssessment>;
  onCancelAssessment: (assessmentID: string, input: Parameters<typeof cancelVendorAssessment>[1]) => Promise<VendorAssessment>;
  onOpenRequest?: (requestID: string) => void;
  onOpenMatter?: (matterID: string) => void;
}) {
  const { vendor, relationship } = record;
  const effectiveAssessmentState = assessmentState === "live" && !assessment && formState === "loading" ? "loading" : assessmentState;
  const setupFailure = assessmentSetup?.state === "FAILED" ? setupFailureText(assessmentSetup.failure_code) : undefined;
  return <>
  <article className="vendor-detail">
    <button type="button" className="text-button vendor-mobile-back" onClick={onBack}>← Back to vendor register</button>
    <div className="vendor-detail-heading"><div><span className="eyebrow">{humanize(relationship.status)} relationship</span><h2>{vendor.legal_name}</h2><p>{vendor.trading_name ? `Trading as ${vendor.trading_name}` : "No trading name recorded"}</p></div><button type="button" className="secondary-button" onClick={onEdit}>Edit vendor relationship</button></div>
    <div className="vendor-service-callout"><span>Service supplied</span><strong>{relationship.service_name}</strong><small>{humanize(relationship.criticality)} criticality · {privacyLabel(relationship.privacy_role)}</small></div>
    <dl className="vendor-facts">
      <Fact label="Accountable owner" value={relationship.business_owner_principal_id}/><Fact label="Jurisdiction" value={vendor.jurisdiction || "Not recorded"}/>
      <Fact label="Registration reference" value={vendor.registration_ref || "Not recorded"}/><Fact label="Effective date" value={formatDate(relationship.effective_from)}/>
      <Fact label="Renewal date" value={formatDate(relationship.renewal_at)}/><Fact label="Source" value={vendor.source_id && vendor.external_ref ? `${vendor.source_id} · ${vendor.external_ref}` : "Entered directly"}/>
      <Fact label="Record updated" value={formatDateTime(relationship.updated_at)}/><Fact label="Current record" value={`Version ${relationship.version}`}/>
    </dl>
    <div className="vendor-boundary-note"><strong>Relationship status</strong><p>{humanize(relationship.status)} for {relationship.service_name}. Review the due-diligence status below before the relationship moves to its next decision.</p></div>
  </article>
  {assessmentState === "live" && !assessment && formState === "unavailable" ? <section className="vdd-workspace" aria-label="Due diligence" tabIndex={-1}><div className="vdd-state vdd-state-error" role="alert"><h2>Due-diligence forms are unavailable</h2><p>Approved collection forms could not be loaded for {relationship.service_name}. Try again before starting the assessment.</p><button type="button" className="secondary-button" onClick={() => void onRefreshForms()}>Reload forms</button></div></section> : <VendorDueDiligence
    relationship={record}
    assessment={assessment}
    review={review}
    reviewState={reviewState}
    form={form}
    requestOutcome={requestOutcome}
    requestOutcomeKind={requestOutcomeKind}
    viewState={effectiveAssessmentState}
    defaultReviewDueDate={recommendedReviewDate()}
    setupFailure={setupFailure}
    onRefresh={onRefreshAssessment}
    onStart={onStartAssessment}
    onSend={onSendAssessmentRequest}
    onReissue={onReissueAssessmentRequest}
    onRetrySetup={onRetryAssessmentSetup}
    onRefreshReview={onRefreshReview}
    onStartReview={onStartAssessmentReview}
    clarificationFields={review?.answers.filter((answer) => answer.visibility === "VISIBLE").map((answer) => ({ id: answer.field_id, label: answer.label }))}
    onRequestClarification={onRequestAssessmentClarification}
    onCreateDeficiency={onCreateAssessmentDeficiency}
    onOpenDocument={(assessmentID, requestID, artifactID) => window.open(vendorAssessmentDocumentURL(assessmentID, requestID, artifactID), "_blank", "noopener,noreferrer")}
    onReviewDocument={onReviewAssessmentDocument}
    onComplete={onCompleteAssessmentReview}
    onCancelAssessment={onCancelAssessment}
    onOpenRequest={onOpenRequest}
    onOpenMatter={onOpenMatter}
  />}
  <VendorWorkPanel relationshipID={relationship.id} onOpenRequest={onOpenRequest}/>
  </>;
}

function Fact({ label, value }: { label: string; value: string }) { return <div><dt>{label}</dt><dd>{value}</dd></div>; }

function needsReviewView(status: VendorAssessment["status"]) {
  return status === "SUBMITTED" || status === "UNDER_REVIEW" || status === "COMPLETED";
}

function selectActiveVendorForm(forms: FormTemplate[]): VendorAssessmentFormOption | undefined {
  const current = forms
    .filter((form) => form.code.trim().toUpperCase() === "VENDOR-DUE-DILIGENCE" && form.status === "ACTIVE" && form.is_current)
    .sort((left, right) => right.version - left.version)[0];
  if (!current) return undefined;
  return { id: current.id, version: current.version, name: current.name, presentation: current.presentation?.default_mode ?? "AUTOMATIC" };
}

function setupFailureText(code?: string) {
  switch (code) {
    case "ASSESSMENT_READ_FAILED": return "The assessment could not be reopened for setup. Retry setup from the current assessment version.";
    case "RELATIONSHIP_READ_FAILED": return "The vendor relationship could not be read during setup. Confirm that the relationship is still available, then retry setup.";
    case "MATTER_CREATE_FAILED": return "The review work item could not be created. Retry assessment setup; no duplicate review will be created.";
    case "ASSESSMENT_SETUP_FAILED": return "The review work item exists, but assessment setup could not be completed. Retry setup to continue the same review.";
    case "ATTEMPTS_EXHAUSTED": return "Assessment setup stopped after repeated attempts. Retry setup to queue another controlled attempt.";
    case "AUTHORITY_ROUTE_UNAVAILABLE": return "No current accountable owner can authorize assessment setup. Correct the relationship authority route, then retry setup.";
    default: return "Assessment setup could not be completed. Retry setup from the current assessment version.";
  }
}

function recommendedReviewDate() {
  const value = new Date();
  value.setUTCDate(value.getUTCDate() + 30);
  return value.toISOString().slice(0, 10);
}

function VendorForm({ mode, form, errors, formError, saving, existingVendor, candidates, candidateState, onFindExisting, onUseExisting, onUseDifferent, onChange, onCancel, onSubmit }: {
  mode: "create" | "edit";
  form: FormValues;
  errors: Record<string, string>;
  formError: string;
  saving: boolean;
  existingVendor?: VendorRelationshipAggregate;
  candidates: VendorRelationshipAggregate[];
  candidateState: "idle" | "loading" | "ready" | "failed";
  onFindExisting: () => Promise<void>;
  onUseExisting: (candidate: VendorRelationshipAggregate) => void;
  onUseDifferent: () => void;
  onChange: <K extends keyof FormValues>(key: K, value: FormValues[K]) => void;
  onCancel: () => void;
  onSubmit: (event: React.FormEvent) => void;
}) {
  return <form className="vendor-form" onSubmit={onSubmit} noValidate>
    <div><span className="eyebrow">{mode === "create" ? "New relationship" : "Current relationship"}</span><h2>{mode === "create" ? "Add a vendor and service" : "Edit vendor relationship"}</h2><p>{mode === "create" ? "Record the organization and the service it supplies. Your verified identity will be recorded as the initial accountable owner." : "Update the supplied service, criticality, privacy role or dates using the current relationship version."}</p></div>
    {formError && <div className="vendor-form-error" role="alert">{formError}</div>}
    <div className="vendor-form-grid">
      {mode === "create" ? <>
        {existingVendor ? <div className="vendor-identity-note">
          <span>Existing vendor selected</span><strong>{existingVendor.vendor.legal_name}</strong>
          <small>{[existingVendor.vendor.registration_ref, existingVendor.vendor.jurisdiction].filter(Boolean).join(" · ") || "No registration or jurisdiction recorded"}</small>
          <p>A separate service relationship will be created without duplicating the vendor identity.</p>
          <button type="button" className="text-button" onClick={onUseDifferent}>Use a different vendor</button>
        </div> : <div className="vendor-existing-search">
          <Field label="Legal name" required error={errors.legalName}><input id="vendor-legal-name" value={form.legalName} onChange={(event) => onChange("legalName", event.target.value)} aria-invalid={Boolean(errors.legalName)}/></Field>
          <button type="button" className="secondary-button" disabled={candidateState === "loading" || form.legalName.trim().length < 2} onClick={() => void onFindExisting()}>{candidateState === "loading" ? "Searching…" : "Find existing vendor"}</button>
          {candidateState === "failed" && <p role="alert">Existing vendors could not be searched. You can retry or continue only if this is a new vendor.</p>}
          {candidateState === "ready" && candidates.length === 0 && <p>No existing vendor matched this name. Continue with the new vendor details.</p>}
          {candidateState === "ready" && candidates.length > 0 && <section className="vendor-match-list" aria-label="Possible vendor matches"><h3>Possible vendor matches</h3><p>Select an existing vendor, or continue only when this is a different organization.</p>{candidates.map((candidate) => <button type="button" className="vendor-match" key={candidate.relationship.id} aria-label={`Use ${candidate.vendor.legal_name} for a new service relationship`} onClick={() => onUseExisting(candidate)}><strong>{candidate.vendor.legal_name}</strong><span>{candidate.relationship.service_name}</span><small>{candidate.vendor.registration_ref || candidate.vendor.external_ref || "No reference recorded"}</small></button>)}</section>}
        </div>}
        {!existingVendor && <><Field label="Trading name"><input id="vendor-trading-name" value={form.tradingName} onChange={(event) => onChange("tradingName", event.target.value)}/></Field>
        <Field label="Registration reference"><input id="vendor-registration" value={form.registrationRef} onChange={(event) => onChange("registrationRef", event.target.value)}/></Field>
        <Field label="Jurisdiction"><input id="vendor-jurisdiction" value={form.jurisdiction} onChange={(event) => onChange("jurisdiction", event.target.value)} placeholder="For example, Nigeria"/></Field></>}
      </> : <div className="vendor-identity-note">
        <span>Vendor legal details</span><strong>{form.legalName}</strong>
        <small>{[form.registrationRef, form.jurisdiction].filter(Boolean).join(" · ") || "No registration or jurisdiction recorded"}</small>
        <p>These details are shared across the bank and cannot be changed from this service relationship.</p>
      </div>}
      <Field label="Service supplied" required error={errors.serviceName} wide><input id="vendor-service" value={form.serviceName} onChange={(event) => onChange("serviceName", event.target.value)} aria-invalid={Boolean(errors.serviceName)}/></Field>
      <Field label="Criticality" required><select id="vendor-criticality" value={form.criticality} onChange={(event) => onChange("criticality", event.target.value as VendorCriticality)}><option value="STANDARD">Standard</option><option value="IMPORTANT">Important</option><option value="CRITICAL">Critical</option></select></Field>
      <Field label="Privacy role" required><select id="vendor-privacy-role" value={form.privacyRole} onChange={(event) => onChange("privacyRole", event.target.value as VendorPrivacyRole)}><option value="NONE">No processing role</option><option value="PROCESSOR">Processor</option><option value="JOINT_CONTROLLER">Joint controller</option></select></Field>
      {mode === "create" && !existingVendor && <><Field label="Source system" error={errors.sourceID}><input id="vendor-source" value={form.sourceID} onChange={(event) => onChange("sourceID", event.target.value)} placeholder="For example, procurement"/></Field><Field label="Source reference"><input id="vendor-external-ref" value={form.externalRef} onChange={(event) => onChange("externalRef", event.target.value)}/></Field></>}
      <Field label="Effective date"><input id="vendor-effective" type="date" value={form.effectiveFrom} onChange={(event) => onChange("effectiveFrom", event.target.value)}/></Field>
      <Field label="Renewal date" error={errors.renewalAt}><input id="vendor-renewal" type="date" value={form.renewalAt} onChange={(event) => onChange("renewalAt", event.target.value)} aria-invalid={Boolean(errors.renewalAt)}/></Field>
    </div>
    <div className="vendor-form-actions"><button type="button" className="secondary-button" onClick={onCancel} disabled={saving}>Cancel</button><button type="submit" className="primary-button" disabled={saving}>{saving ? "Saving…" : mode === "create" ? "Add vendor relationship" : "Save vendor relationship"}</button></div>
  </form>;
}

function Field({ label, required, error, wide, children }: { label: string; required?: boolean; error?: string; wide?: boolean; children: React.ReactElement<{ id?: string }> }) {
  const id = children.props.id;
  return <label className={wide ? "vendor-field wide" : "vendor-field"} htmlFor={id}><span className={required ? "required" : undefined}>{label}</span>{children}{error && <small role="alert">{error}</small>}</label>;
}

function optional(value: string) { return value.trim() || undefined; }
function apiDate(value: string) { return value ? `${value}T00:00:00Z` : undefined; }
function dateInput(value?: string) { return value?.slice(0, 10) ?? ""; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
function privacyLabel(value: VendorPrivacyRole) { return value === "NONE" ? "No processing role" : humanize(value); }
function formatDate(value?: string) { if (!value) return "Not recorded"; const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : "Not recorded"; }
function formatDateTime(value: string) { const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed)) : "Update time unavailable"; }
