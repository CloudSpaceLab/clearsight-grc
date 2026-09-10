import { vendorWorkFilters } from "../vendorFormPresentation";
import { useEffect, useMemo, useRef, useState } from "react";
import "../vendors.css";
import { resolveAuthority } from "../api";
import { apiErrorKind } from "../http";
import { loadFormTemplates } from "../monitoringApi";
import { prepareVendorCollection, type PrepareVendorCollectionInput, type PrepareVendorCollectionResult } from "../vendorCollectionApi";
import type { FormTemplate } from "../monitoringTypes";
import { applyVendorAssessmentResponse, cancelVendorAssessment, completeVendorAssessment, createVendorAssessmentDeficiency, loadCurrentVendorAssessment, loadVendorAssessment, reissueVendorAssessmentRequest, requestVendorAssessmentClarification, retryVendorAssessmentSetup, reviewVendorAssessmentDocument, sendVendorAssessmentRequest, startVendorAssessment, startVendorAssessmentReview, vendorAssessmentDocumentURL } from "../vendorAssessmentApi";
import type { ApplyVendorAssessmentResponseInput, CompleteVendorAssessmentInput, CreateVendorAssessmentDeficiencyInput, CurrentVendorAssessment, ReviewVendorAssessmentDocumentInput, StartVendorAssessmentInput, VendorAssessment, VendorAssessmentApplicationResult, VendorAssessmentClarificationInput, VendorAssessmentFormOption, VendorAssessmentReviewView, VendorAssessmentSendOutcome, VendorAssessmentSetupRetryOutcome } from "../vendorAssessmentTypes";
import { createVendorRelationship, loadVendorRelationship, loadVendorRelationships, updateVendorRelationship } from "../vendorApi";
import { normalizeRegisteredAddress, normalizeWebsiteDomain, validateWebsiteDomain, vendorIdentityLimits } from "../vendorIdentity";
import type { CreateVendorRelationshipInput, VendorCriticality, VendorIdentityPresentation, VendorPrivacyRole, VendorRelationshipAggregate } from "../vendorTypes";
import { VendorDueDiligence } from "./VendorDueDiligence";
import { VendorWorkPanel } from "./VendorWorkPanel";
import { VendorBrandIcon, vendorBrandLabel } from "./VendorBrandIcon";
import { VendorActivationPanel } from "./VendorActivationPanel";
import { VendorIdentityEditor } from "./VendorIdentityEditor";
import { VendorFormReadiness } from "./VendorFormReadiness";
import { VendorComplianceOverview } from "./VendorComplianceOverview";
import { VendorPortfolio } from "./VendorPortfolio";
import { ActionLink, Button, Notice, SelectField, StatusBadge, TextField, TextArea, Tabs } from "./ui";
import { DocumentBrowser } from "./documents/DocumentBrowser";
import { VendorFormsPanel, VendorResponseHistory } from "./VendorFormsPanel";
import { VendorFormRequest, type VendorRequestTarget } from "./VendorFormRequest";
import { loadVendorFormSummaries, type VendorFormSummary, type VendorFormsFilter } from "../vendorFormsApi";
import "./vendor-forms.css";

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
  onOpenForms?: () => void;
};

type LoadState = "loading" | "live" | "unavailable";
type VendorSection = "OVERVIEW" | "FORMS" | "DOCUMENTS" | "DUE_DILIGENCE" | "HISTORY";
const vendorSections = [{ id: "OVERVIEW", label: "Overview" }, { id: "FORMS", label: "Forms" }, { id: "DOCUMENTS", label: "Documents" }, { id: "DUE_DILIGENCE", label: "Due diligence" }, { id: "HISTORY", label: "History" }] satisfies Array<{ id: VendorSection; label: string }>;

type FormValues = {
  legalName: string;
  tradingName: string;
  registrationRef: string;
  jurisdiction: string;
  websiteDomain: string;
  registeredAddress: string;
  serviceName: string;
  criticality?: VendorCriticality;
  privacyRole?: VendorPrivacyRole;
  sourceID: string;
  externalRef: string;
  effectiveFrom: string;
  renewalAt: string;
};

const emptyForm: FormValues = {
  legalName: "", tradingName: "", registrationRef: "", jurisdiction: "", websiteDomain: "", registeredAddress: "", serviceName: "",
  criticality: undefined, privacyRole: undefined, sourceID: "", externalRef: "", effectiveFrom: "", renewalAt: "",
};

function focusGuideTarget(target: HTMLElement | null) {
  if (!isGuideTargetAvailable(target)) return false;
  for (let ancestor = target.parentElement; ancestor; ancestor = ancestor.parentElement) { if (ancestor instanceof HTMLDetailsElement) ancestor.open = true; }
  const reducedMotion = typeof window.matchMedia === "function" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  target.scrollIntoView?.({ behavior: reducedMotion ? "auto" : "smooth", block: "center" });
  if (!target.hasAttribute("tabindex") && !(target instanceof HTMLButtonElement) && !(target instanceof HTMLInputElement)) target.setAttribute("tabindex", "-1");
  target.focus({ preventScroll: true });
  return document.activeElement === target;
}

function isGuideTargetAvailable(target: HTMLElement | null): target is HTMLElement {
  if (!target || target.hidden || target.closest("[hidden], [inert], [aria-hidden='true']")) return false;
  if (target instanceof HTMLButtonElement && target.disabled) return false;
  const style = window.getComputedStyle?.(target);
  return style?.display !== "none" && style?.visibility !== "hidden";
}

function firstVisiblePrimaryAction(selector: string) {
  return [...document.querySelectorAll<HTMLElement>(`${selector} button.primary-button:not(:disabled), ${selector} button.cs-button--primary:not(:disabled)`)].find(isGuideTargetAvailable) ?? null;
}

export function VendorsWorkspace({ organizationName, legalEntityName, targetID, guideIntent, onGuideIntentCompleted, onGuideIntentFailed, onTarget, onOpenRequest, onOpenMatter, onOpenForms }: Props) {
  const [records, setRecords] = useState<VendorRelationshipAggregate[]>([]);
  const [selected, setSelected] = useState<VendorRelationshipAggregate | null>(null);
  const [accountableOwnerLabel, setAccountableOwnerLabel] = useState("Current owner unavailable");
  const [state, setState] = useState<"loading" | "live" | "unavailable">("loading");
  const [mode, setMode] = useState<"browse" | "create" | "edit" | "edit-identity">("browse");
  const [query, setQuery] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [nextCursor, setNextCursor] = useState("");
  const [loadingMore, setLoadingMore] = useState(false);
  const [loadMoreError, setLoadMoreError] = useState(false);
  const [formSummaries, setFormSummaries] = useState<Map<string, VendorFormSummary>>(new Map());
  const [summaryState, setSummaryState] = useState<LoadState>("loading");
  const [formsRefreshKey, setFormsRefreshKey] = useState(0);
  const [checkedRelationships, setCheckedRelationships] = useState<string[]>([]);
  const [requestTargets, setRequestTargets] = useState<VendorRequestTarget[]>();
  const [workFilter, setWorkFilter] = useState<VendorFormsFilter>();
  const [formsFocus, setFormsFocus] = useState<{ relationshipID: string; filter?: VendorFormsFilter }>();
  const [vendorSection, setVendorSection] = useState<VendorSection>("OVERVIEW");
  const consumedFormsFocus = useRef<typeof formsFocus>(undefined);
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
  const [formSetupOpen, setFormSetupOpen] = useState(false);
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
  const nextActionSearch = useRef<{ intentID: number; tried: Set<string> } | undefined>(undefined);
  const navigationFocus = useRef<"detail" | "register" | undefined>(undefined);
  useEffect(() => {
    if (state !== "live" || !navigationFocus.current) return;
    const target = document.getElementById(navigationFocus.current === "detail" ? "vendor-detail-focus" : "vendor-portfolio-register");
    if (!target) return;
    target.focus({ preventScroll: true });
    target.scrollIntoView?.({ block: "start" });
    navigationFocus.current = undefined;
  }, [selected?.relationship.id, state]);

  const recordIDs = records.map((record) => record.relationship.id).join(",");
  useEffect(() => {
    let active = true;
    const ids = recordIDs ? recordIDs.split(",") : [];
    setCheckedRelationships((current) => current.filter((id) => ids.includes(id)));
    setFormSummaries(new Map());
    if (!ids.length) { setSummaryState("live"); return; }
    setSummaryState("loading");
    const groups = Array.from({ length: Math.ceil(ids.length / 50) }, (_, index) => ids.slice(index * 50, index * 50 + 50));
    void Promise.allSettled(groups.map((group) => loadVendorFormSummaries(group))).then((results) => {
      if (!active) return;
      const summaries = new Map<string, VendorFormSummary>();
      for (const result of results) if (result.status === "fulfilled") for (const value of result.value.items ?? []) summaries.set(value.relationship_id, value);
      setFormSummaries(summaries);
      setSummaryState(results.some((result) => result.status === "rejected") ? "unavailable" : "live");
    });
    return () => { active = false; };
  }, [recordIDs, formsRefreshKey, assessment?.id, assessment?.version]);

  useEffect(() => {
    if (!formsFocus || consumedFormsFocus.current === formsFocus || vendorSection !== "FORMS" || formsFocus.relationshipID !== selected?.relationship.id) return;
    const panel = document.querySelector<HTMLElement>(".vendor-forms-panel");
    if (panel && focusGuideTarget(panel)) consumedFormsFocus.current = formsFocus;
  }, [formsFocus, selected?.relationship.id, vendorSection]);

  useEffect(() => {
    if (!guideIntent) void refresh(targetID, submittedQuery);
  }, [targetID]);

  useEffect(() => {
    if (guideIntent) void refresh(targetID, "", guideIntent);
  }, [guideIntent, targetID]);

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
    const intendedSection = guideIntent.type === "open-vendor-work" || guideIntent.type === "open-vendor-next-action" && assessment?.status === "COMPLETED" ? "FORMS" : "DUE_DILIGENCE";
    if (vendorSection !== intendedSection && !(guideIntent.type === "open-vendor-next-action" && vendorSection === "FORMS")) { setVendorSection(intendedSection); return; }
    if (guideIntent.type === "open-vendor-next-action") {
      const workspace = document.querySelector<HTMLElement>(".vendors-workspace");
      if (!workspace) return;
      const focusNextAction = () => {
        const dueDiligenceAction = assessment?.status === "COMPLETED" ? null : firstVisiblePrimaryAction(".vdd-workspace");
        if (dueDiligenceAction && focusGuideTarget(dueDiligenceAction)) return acknowledgeGuideIntent(guideIntent.id);
        if (vendorSection !== "FORMS") { setVendorSection("FORMS"); return true; }
        const vendorWork = workspace.querySelector<HTMLElement>(".vendor-work-panel");
        if (!vendorWork || vendorWork.getAttribute("aria-busy") === "true") return false;
        const vendorWorkAction = firstVisiblePrimaryAction(".vendor-work-panel");
        if (vendorWorkAction && focusGuideTarget(vendorWorkAction)) return acknowledgeGuideIntent(guideIntent.id);
        const search = nextActionSearch.current?.intentID === guideIntent.id
          ? nextActionSearch.current
          : { intentID: guideIntent.id, tried: new Set<string>() };
        nextActionSearch.current = search;
        search.tried.add(selected.relationship.id);
        const next = records.find((record) => !search.tried.has(record.relationship.id));
        if (next) {
          assessmentLoadID.current += 1;
          setAssessment(null);
          setAssessmentSetup(undefined);
          setAssessmentState("loading");
          setReview(undefined);
          setReviewState("loading");
          setSelected(next);
          setVendorSection("DUE_DILIGENCE");
          setMode("browse");
          return true;
        }
        return failGuideIntent(guideIntent.id);
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
    function failGuideIntent(id: number) {
      acknowledgedGuideIntentID.current = id;
      onGuideIntentFailed?.(id);
      return true;
    }
  }, [guideIntent, state, mode, records, selected?.relationship.id, assessment, assessmentState, reviewState, formState, vendorSection, onGuideIntentCompleted, onGuideIntentFailed]);

  useEffect(() => {
    void refreshForms();
    return () => { formLoadID.current += 1; };
  }, []);

  useEffect(() => {
    let current = true;
    if (!selected) {
      setAccountableOwnerLabel("Current owner unavailable");
      return () => { current = false; };
    }
    setAccountableOwnerLabel("Loading current owner…");
    void resolveAuthority({
      object_type: "THIRD_PARTY_RELATIONSHIP",
      object_id: selected.relationship.id,
      responsibility: "ACCOUNTABLE_OWNER",
      decision_type: "thirdparty.assessment.start",
      materiality: 3,
    }).then((resolution) => {
      if (current) setAccountableOwnerLabel(resolution.principal.display_name.trim() || "Current owner unavailable");
    }).catch(() => {
      if (current) setAccountableOwnerLabel("Current owner unavailable");
    });
    return () => { current = false; };
  }, [selected?.relationship.id]);

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

  const activeVendorForms = useMemo(() => selectActiveVendorForms(forms), [forms]);
  const activeVendorForm = activeVendorForms[0];

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

  async function prepareAssessmentRequest(input: PrepareVendorCollectionInput) {
    if (!assessment) throw new Error("No current assessment");
    const outcome = await prepareVendorCollection(assessment.id, input);
    setAssessment(outcome.assessment);
    setRequestOutcome(undefined);
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

  async function applyAssessmentResponse(assessmentID: string, revisionID: string, input: ApplyVendorAssessmentResponseInput): Promise<VendorAssessmentApplicationResult> {
    const result = await applyVendorAssessmentResponse(assessmentID, revisionID, input);
    setAssessment(result.review.assessment);
    setReview(result.review);
    setReviewState("live");
    return result;
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
      if (nextSelected?.relationship.id !== selected?.relationship.id) setVendorSection("OVERVIEW");
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
    if (mode !== "browse" || !nextCursor || loadingMore) return;
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
    if (mode !== "browse") return;
    const search = query.trim();
    setSubmittedQuery(search);
    setSelected(null);
    onTarget?.();
    void refresh(undefined, search);
  }

  function choose(record: VendorRelationshipAggregate) {
    setVendorSection("OVERVIEW");
    if (mode !== "browse") return;
    navigationFocus.current = "detail";
    setSelected(record); setMode("browse"); setNotice(""); setFormError(""); onTarget?.(record.relationship.id);
  }

  function startCreate() {
    setMode("create"); setForm(emptyForm); setSelected(null); setExistingVendorSource(undefined); setVendorCandidates([]); setCandidateState("idle"); setFieldErrors({}); setFormError(""); setNotice(""); onTarget?.();
  }

  function startEdit() {
    if (!selected) return;
    const { vendor, relationship } = selected;
    setForm({
      legalName: vendor.legal_name, tradingName: vendor.trading_name ?? "", registrationRef: vendor.registration_ref ?? "", jurisdiction: vendor.jurisdiction ?? "", websiteDomain: vendor.website_domain ?? "", registeredAddress: vendor.registered_address ?? "",
      serviceName: relationship.service_name, criticality: relationship.criticality, privacyRole: relationship.privacy_role,
      sourceID: vendor.source_id ?? "", externalRef: vendor.external_ref ?? "", effectiveFrom: dateInput(relationship.effective_from), renewalAt: dateInput(relationship.renewal_at),
    });
    setMode("edit"); setFieldErrors({}); setFormError(""); setNotice("");
  }

  function startIdentityEdit() {
    if (!selected) return;
    setMode("edit-identity"); setFieldErrors({}); setFormError(""); setNotice("");
  }

  function applyVendorPresentation(presentation: VendorIdentityPresentation, message = "", close = false) {
    const update = (item: VendorRelationshipAggregate) => item.vendor.id === presentation.vendor.id ? { ...item, vendor: presentation.vendor, brand: presentation.brand } : item;
    setRecords((current) => current.map(update));
    setSelected((current) => current ? update(current) : current);
    if (message) setNotice(message);
    if (close) setMode("browse");
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
    setForm((current) => ({ ...current, legalName: candidate.vendor.legal_name, tradingName: candidate.vendor.trading_name ?? "", registrationRef: candidate.vendor.registration_ref ?? "", jurisdiction: candidate.vendor.jurisdiction ?? "", websiteDomain: "", registeredAddress: "", sourceID: "", externalRef: "" }));
    setCandidateState("idle");
    setVendorCandidates([]);
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    const errors: Record<string, string> = {};
    if (mode === "create" && !form.legalName.trim()) errors.legalName = "Enter the vendor's legal name.";
    if (mode === "create" && !existingVendorSource) {
      const websiteError = validateWebsiteDomain(form.websiteDomain);
      if (websiteError) errors.websiteDomain = websiteError;
      if ([...form.registeredAddress].length > vendorIdentityLimits.registeredAddress) errors.registeredAddress = "Enter a registered address of 2,000 characters or fewer.";
    }
    if (!form.serviceName.trim()) errors.serviceName = "Enter the service supplied to this legal entity.";
    if (!form.criticality) errors.criticality = "Select the service criticality.";
    if (!form.privacyRole) errors.privacyRole = "Select the vendor's privacy role.";
    if ((form.sourceID.trim() && !form.externalRef.trim()) || (!form.sourceID.trim() && form.externalRef.trim())) errors.sourceID = "Enter both source system and source reference, or leave both blank.";
    if (form.effectiveFrom && form.renewalAt && form.renewalAt < form.effectiveFrom) errors.renewalAt = "Renewal date cannot be before the effective date.";
    setFieldErrors(errors);
    if (Object.values(errors).some(Boolean) || !form.criticality || !form.privacyRole) return;

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
          website_domain: existingVendorSource ? undefined : normalizeWebsiteDomain(form.websiteDomain),
          registered_address: existingVendorSource ? undefined : normalizeRegisteredAddress(form.registeredAddress),
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

  const registerLocked = mode !== "browse";
  const registerLockMessage = mode === "create"
    ? "Add or cancel this vendor relationship before using the register."
    : mode === "edit"
      ? "Save or cancel this vendor relationship before using the register."
      : "Save or cancel these vendor details before using the register.";
  function synchronizeRelationship(relationship: VendorRelationshipAggregate["relationship"]) {
    const update = (item: VendorRelationshipAggregate) => item.relationship.id === relationship.id
      && item.relationship.tenant_id === relationship.tenant_id
      && item.relationship.legal_entity_id === relationship.legal_entity_id
      && item.relationship.version <= relationship.version ? { ...item, relationship } : item;
    setSelected((current) => current ? update(current) : current);
    setRecords((current) => current.map(update));
  }

  const workspaceClass = `vendors-workspace${registerLocked ? " is-form" : selected ? " has-selection" : ""}`;
  const shownRecords = workFilter ? records.filter((record) => vendorSummaryMatches(formSummaries.get(record.relationship.id), workFilter)) : records;
  function requestForms(values: VendorRelationshipAggregate[]) { setRequestTargets(values.map((value) => ({ relationshipID: value.relationship.id, vendorName: value.vendor.legal_name, serviceName: value.relationship.service_name }))); }
  function openFormWork(record: VendorRelationshipAggregate, filter?: VendorFormsFilter) { choose(record); setVendorSection("FORMS"); setFormsFocus({ relationshipID: record.relationship.id, filter }); }
  return <div className={workspaceClass} tabIndex={-1}>
    <header className="topbar vendors-topbar">
      <div>{!selected && <span className="eyebrow">{organizationName} · {legalEntityName}</span>}<h1>Vendors</h1></div>
      {mode === "browse" && <Button id="vendor-add-action" type="button" variant={selected ? "secondary" : "primary"} onPress={startCreate} isDisabled={state !== "live"}>Add vendor</Button>}
    </header>

    {notice && <Notice tone="success">{notice}</Notice>}
    {state === "loading" && <div className="workspace-loading" aria-live="polite" aria-busy="true">Loading vendor relationships for {legalEntityName}…</div>}
    {state === "unavailable" && <section className="vendor-state" role="alert"><h2>Vendor records are unavailable</h2><p>The vendor register for {legalEntityName} could not be loaded. Try again before adding or changing a record.</p><Button  type="button" onPress={() => void refresh(targetID, "")}>Try again</Button></section>}
    {state === "live" && !selected && !registerLocked && <VendorPortfolio records={records} summaries={formSummaries} summaryState={summaryState} hasMore={!!nextCursor} onFilter={(filter) => { setWorkFilter(filter || undefined); document.getElementById("vendor-portfolio-register")?.scrollIntoView?.({ block: "start" }); }}/>}
    {state === "live" && <div className="vendor-layout">
      <section id="vendor-portfolio-register" tabIndex={-1} className="vendor-register" aria-label={`Vendor relationships for ${legalEntityName}`} aria-describedby={registerLocked ? "vendor-register-lock-note" : undefined}>
        <div className="vendor-register-header"><div><h2>Vendor register</h2><p>{submittedQuery ? `Showing ${records.length} matching ${records.length === 1 ? "relationship" : "relationships"}` : `Showing ${records.length} ${records.length === 1 ? "relationship" : "relationships"} in this legal entity`}</p>{nextCursor && <small>More relationships are available.</small>}</div></div>
        {registerLocked && <p id="vendor-register-lock-note" className="vendor-register-lock-note">{registerLockMessage}</p>}
        <form className="vendor-search" onSubmit={searchRelationships}><TextField label="Search vendors and services" type="search" value={query} onChange={setQuery} placeholder="Name, service or reference" isDisabled={registerLocked}/><Button type="submit"  isDisabled={registerLocked}>Search vendors</Button></form>
        {records.length > 0 && <div className="vendor-register-bulk"><SelectField label="Form work in loaded relationships" value={workFilter} placeholder="All loaded relationships" isDisabled={registerLocked} options={vendorWorkFilters} onChange={setWorkFilter}/>{workFilter && <p>{shownRecords.length} of {records.length} loaded relationships match this form-work filter.</p>}
          <Button isDisabled={registerLocked || checkedRelationships.length === 0} onPress={() => requestForms(records.filter((record) => checkedRelationships.includes(record.relationship.id)))}>Request form for {checkedRelationships.length} {checkedRelationships.length === 1 ? "service" : "services"}</Button>
          {checkedRelationships.length === 0 && <small>Select up to 50 vendor services to request the same approved form.</small>}{checkedRelationships.length > 0 && <Button variant="quiet" onPress={() => setCheckedRelationships([])}>Clear selection</Button>}
        </div>}
        {summaryState === "unavailable" && <Notice tone="warning">Some vendor form summaries could not be loaded. Their counts and concern remain unknown. <Button onPress={() => setFormsRefreshKey((value) => value + 1)}>Reload form summaries</Button></Notice>}
        {records.length > 0 ? <div className="vendor-list">{shownRecords.map((record) => <div className="vendor-register-entry" key={record.relationship.id}><label><input type="checkbox" aria-label={`Select ${record.vendor.legal_name} · ${record.relationship.service_name}`} checked={checkedRelationships.includes(record.relationship.id)} disabled={registerLocked || checkedRelationships.length >= 50 && !checkedRelationships.includes(record.relationship.id)} onChange={(event) => setCheckedRelationships((current) => event.target.checked ? [...current, record.relationship.id] : current.filter((id) => id !== record.relationship.id))}/></label><div><button type="button" aria-label={`${record.vendor.legal_name}, ${record.relationship.service_name}`} aria-current={selected?.relationship.id === record.relationship.id ? "true" : undefined} className={selected?.relationship.id === record.relationship.id ? "vendor-row selected" : "vendor-row"} onClick={() => choose(record)} disabled={registerLocked}>
          <VendorBrandIcon vendorID={record.vendor.id} legalName={record.vendor.legal_name} brand={record.brand} decorative/><span className="vendor-row-main"><strong>{record.vendor.legal_name}</strong><span>Service: {record.relationship.service_name}</span></span><span className={`vendor-criticality criticality-${record.relationship.criticality.toLowerCase()}`}>{humanize(record.relationship.criticality)}</span>
        </button><VendorRegisterFormSummary summary={formSummaries.get(record.relationship.id)} label={`${record.vendor.legal_name} · ${record.relationship.service_name}`} loading={summaryState === "loading"} disabled={registerLocked} onOpen={(filter) => openFormWork(record, filter)}/></div></div>)}{shownRecords.length === 0 && <p>No loaded vendor relationships match this form-work filter. Clear the filter or load more vendors.</p>}</div> : submittedQuery ? <div className="vendor-empty"><h3>No vendor relationships match this search.</h3><p>No legal name, service, registration or source reference matched “{submittedQuery}” in {legalEntityName}.</p><Button type="button"  onPress={() => { setQuery(""); setSubmittedQuery(""); void refresh(undefined, ""); }} isDisabled={registerLocked}>Clear search</Button></div> : <div className="vendor-empty"><h3>No vendor relationships found for {legalEntityName}.</h3><p>Add the first vendor and the service it supplies. Use <strong>Add vendor</strong> above; the signed-in actor becomes the initial accountable owner.</p></div>}
        {nextCursor && <Button type="button"  isDisabled={registerLocked || loadingMore} onPress={() => void loadMoreRelationships()}>{loadingMore ? "Loading…" : "Load more vendors"}</Button>}
        {loadMoreError && <p role="alert" className="inline-error">More vendor relationships could not be loaded. The current results remain available.</p>}
      </section>

      <section id="vendor-detail-focus" tabIndex={-1} className="vendor-focus" aria-label="Selected vendor relationship">
        {mode === "edit-identity" && selected ? <VendorIdentityEditor record={selected} onCancel={cancelForm} onIdentitySaved={(presentation) => applyVendorPresentation(presentation, "Vendor details updated.", true)} onBrandSaved={(presentation) => applyVendorPresentation(presentation)} onPresentationReloaded={(presentation) => applyVendorPresentation(presentation)}/> : (mode === "create" || mode === "edit") ? <VendorForm mode={mode} form={form} errors={fieldErrors} formError={formError} saving={saving} existingVendor={existingVendorSource} candidates={vendorCandidates} candidateState={candidateState} onFindExisting={findExistingVendor} onUseExisting={useExistingVendor} onUseDifferent={() => { setExistingVendorSource(undefined); setForm((current) => ({ ...current, legalName: "", tradingName: "", registrationRef: "", jurisdiction: "", websiteDomain: "", registeredAddress: "" })); }} onChange={setValue} onCancel={cancelForm} onSubmit={submit}/> : selected ? <VendorDetail
          record={selected}
          formSummary={formSummaries.get(selected.relationship.id)}
          summaryState={summaryState}
          key={selected.relationship.id}
          section={vendorSection}
          onSectionChange={setVendorSection}
          assessment={assessment}
          assessmentSetup={assessmentSetup}
          assessmentState={assessmentState}
          review={review}
          reviewState={reviewState}
          form={activeVendorForm}
          forms={activeVendorForms}
          formState={formState}
          requestOutcome={requestOutcome}
          requestOutcomeKind={requestOutcomeKind}
          onBack={() => { navigationFocus.current = "register"; setSelected(null); onTarget?.(); }}
          onEdit={startEdit}
          onEditIdentity={startIdentityEdit}
          onRefreshAssessment={() => refreshAssessment(selected.relationship.id)}
          onRefreshForms={refreshForms}
          onSetUpForm={() => setFormSetupOpen(true)}
          onOpenForms={onOpenForms}
          onRequestForm={() => requestForms([selected])}
          onFormWorkUpdated={() => setFormsRefreshKey((value) => value + 1)}
          formsRefreshKey={formsRefreshKey + (assessment?.version ?? 0)}
          formsFilter={formsFocus?.relationshipID === selected.relationship.id ? formsFocus.filter : undefined}
          onStartAssessment={startAssessment}
          onSendAssessmentRequest={sendAssessmentRequest}
          onPrepareAssessmentRequest={prepareAssessmentRequest}
          onReissueAssessmentRequest={reissueAssessmentRequest}
          onRetryAssessmentSetup={retryAssessmentSetup}
          onRefreshReview={refreshReview}
          onStartAssessmentReview={beginAssessmentReview}
          onRequestAssessmentClarification={requestAssessmentClarification}
          onCreateAssessmentDeficiency={recordAssessmentDeficiency}
          onReviewAssessmentDocument={decideAssessmentDocument}
          onCompleteAssessmentReview={finishAssessmentReview}
          onCancelAssessment={cancelAssessment}
          onApplyAssessmentResponse={applyAssessmentResponse}
          onOpenRequest={onOpenRequest}
          onOpenMatter={onOpenMatter}
          accountableOwnerLabel={accountableOwnerLabel}
          onActivated={synchronizeRelationship}
          onRefreshed={synchronizeRelationship}
        /> : records.length > 0 ? <div className="vendor-selection"><h2>Select a vendor</h2><p>Choose a relationship to review its service, accountable owner, source and current record version.</p></div> : null}
      </section>
    </div>}
    {requestTargets && <VendorFormRequest targets={requestTargets} onClose={() => setRequestTargets(undefined)} onUpdated={() => setFormsRefreshKey((value) => value + 1)}/>}
    {formSetupOpen && <VendorFormReadiness
      onClose={() => setFormSetupOpen(false)}
      onReady={(ready) => {
        setForms((current) => [ready, ...current.filter((item) => item.id !== ready.id)]);
        setFormState("live");
        setFormSetupOpen(false);
      }}
    />}
  </div>;
}

function VendorDetail({ record, formSummary, summaryState, section, onSectionChange, assessment, assessmentSetup, assessmentState, review, reviewState, form, forms, formState, requestOutcome, requestOutcomeKind, onBack, onEdit, onEditIdentity, onRefreshAssessment, onRefreshForms, onSetUpForm, onOpenForms, onStartAssessment, onPrepareAssessmentRequest, onSendAssessmentRequest, onReissueAssessmentRequest, onRetryAssessmentSetup, onRefreshReview, onStartAssessmentReview, onRequestAssessmentClarification, onCreateAssessmentDeficiency, onReviewAssessmentDocument, onCompleteAssessmentReview, onCancelAssessment, onApplyAssessmentResponse, onOpenRequest, onOpenMatter, accountableOwnerLabel, onActivated, onRefreshed, onRequestForm, onFormWorkUpdated, formsRefreshKey, formsFilter }: {
  record: VendorRelationshipAggregate;
  formSummary?: VendorFormSummary;
  summaryState: LoadState;
  section: VendorSection;
  onSectionChange: (section: VendorSection) => void;
  assessment: VendorAssessment | null;
  assessmentSetup?: CurrentVendorAssessment["setup"];
  assessmentState: LoadState;
  review?: VendorAssessmentReviewView;
  reviewState: LoadState;
  form?: VendorAssessmentFormOption;
  forms: VendorAssessmentFormOption[];
  formState: LoadState;
  requestOutcome?: VendorAssessmentSendOutcome;
  requestOutcomeKind: "initial" | "replacement";
  onBack: () => void;
  onEdit: () => void;
  onEditIdentity: () => void;
  onRefreshAssessment: () => Promise<void>;
  onRefreshForms: () => Promise<void>;
  onSetUpForm: () => void;
  onOpenForms?: () => void;
  onRequestForm: () => void;
  onFormWorkUpdated: () => void;
  formsRefreshKey: number;
  formsFilter?: VendorFormsFilter;
  onStartAssessment: (input: StartVendorAssessmentInput) => Promise<VendorAssessment | void>;
  onSendAssessmentRequest: (input: Parameters<typeof sendVendorAssessmentRequest>[1]) => Promise<VendorAssessmentSendOutcome>;
  onPrepareAssessmentRequest: (input: PrepareVendorCollectionInput) => Promise<PrepareVendorCollectionResult>;
  onReissueAssessmentRequest: (input: Parameters<typeof reissueVendorAssessmentRequest>[1]) => Promise<VendorAssessmentSendOutcome>;
  onRetryAssessmentSetup: (assessmentID: string, expectedVersion: number) => Promise<VendorAssessmentSetupRetryOutcome>;
  onRefreshReview: (assessmentID: string) => Promise<void>;
  onStartAssessmentReview: (assessmentID: string, expectedVersion: number) => Promise<VendorAssessment>;
  onRequestAssessmentClarification: typeof requestVendorAssessmentClarification;
  onCreateAssessmentDeficiency: typeof createVendorAssessmentDeficiency;
  onReviewAssessmentDocument: typeof reviewVendorAssessmentDocument;
  onCompleteAssessmentReview: (assessmentID: string, input: CompleteVendorAssessmentInput) => Promise<VendorAssessment>;
  onCancelAssessment: (assessmentID: string, input: Parameters<typeof cancelVendorAssessment>[1]) => Promise<VendorAssessment>;
  onApplyAssessmentResponse: (assessmentID: string, revisionID: string, input: ApplyVendorAssessmentResponseInput) => Promise<VendorAssessmentApplicationResult>;
  onOpenRequest?: (requestID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  accountableOwnerLabel: string;
  onActivated: (relationship: VendorRelationshipAggregate["relationship"]) => void;
  onRefreshed: (relationship: VendorRelationshipAggregate["relationship"]) => void;
}) {
  const { vendor, relationship } = record;
  const effectiveAssessmentState = assessmentState === "live" && !assessment && formState === "loading" ? "loading" : assessmentState;
  const setupFailure = assessmentSetup?.state === "FAILED" ? setupFailureText(assessmentSetup.failure_code) : undefined;
  return <>
  <article className="vendor-detail">
    <div className="vendor-mobile-back"><Button variant="quiet" onPress={onBack}>Back to vendor register</Button></div>
    <div className="vendor-detail-heading"><div className="vendor-detail-identity"><VendorBrandIcon vendorID={vendor.id} legalName={vendor.legal_name} brand={record.brand} size="detail"/><div><span className="eyebrow">{humanize(relationship.status)} relationship</span><h2>{vendor.legal_name}</h2></div></div></div>
    <div className="vendor-service-summary"><strong>{relationship.service_name}</strong><span>{humanize(relationship.criticality)} criticality · {privacyLabel(relationship.privacy_role)}</span></div>
    <p className="vendor-current-owner"><span>Accountable owner</span><strong>{accountableOwnerLabel}</strong></p>
  </article>
  <Tabs ariaLabel="Vendor sections" compactLabel="Vendor section" retainVisitedPanels items={vendorSections} selectedKey={section} onSelectionChange={onSectionChange}>{(current) => <>
  {current === "OVERVIEW" && <section className="vendor-detail" aria-label="Vendor overview">
    <VendorComplianceOverview relationshipID={relationship.id} serviceName={relationship.service_name} summary={formSummary} summaryState={summaryState} assessment={assessment} assessmentState={assessmentState} refreshKey={formsRefreshKey} onOpenForms={() => onSectionChange("FORMS")} onOpenDueDiligence={() => onSectionChange("DUE_DILIGENCE")} onRequestForm={onRequestForm} onUpdated={onFormWorkUpdated} onOpenRequest={onOpenRequest}/>
    <details className="vendor-record-details"><summary>Vendor details and record history</summary>
    <div className="vendor-detail-actions"><Button onPress={onEditIdentity}>Edit vendor details</Button><Button onPress={onEdit}>Edit vendor relationship</Button></div>
    {vendor.trading_name && <p>Trading as {vendor.trading_name}</p>}
    <dl className="vendor-facts">
      <Fact label="Renewal date" value={formatDate(relationship.renewal_at)}/>
      <Fact label="Logo" value={vendorBrandLabel(record.brand)}/>
      <Fact label="Jurisdiction" value={vendor.jurisdiction || "Not recorded"}/>
      <Fact label="Registration reference" value={vendor.registration_ref || "Not recorded"}/><Fact label="Website domain" value={vendor.website_domain || "Not recorded"}/><Fact label="Registered address" value={vendor.registered_address || "Not recorded"}/><Fact label="Effective date" value={formatDate(relationship.effective_from)}/>
      <Fact label="Source" value={vendor.source_id && vendor.external_ref ? `${vendor.source_id} · ${vendor.external_ref}` : "Entered directly"}/>
      <Fact label="Relationship updated" value={formatDateTime(relationship.updated_at)}/><Fact label="Relationship version" value={`Version ${relationship.version}`}/><Fact label="Vendor details version" value={`Vendor version ${vendor.version}`}/>
    </dl>
    </details>
  </section>}
  {current === "FORMS" && <><VendorFormsPanel relationshipID={relationship.id} serviceName={relationship.service_name} onRequestForm={onRequestForm} onUpdated={onFormWorkUpdated} onOpenHistory={() => onSectionChange("HISTORY")} onOpenDueDiligence={() => onSectionChange("DUE_DILIGENCE")} refreshKey={formsRefreshKey} initialFilter={formsFilter} onOpenRequest={onOpenRequest}/><details className="vendor-linked-work"><summary>Linked vendor work</summary><VendorWorkPanel relationshipID={relationship.id} onOpenRequest={onOpenRequest}/></details></>}
  {current === "DOCUMENTS" && <DocumentBrowser scopeLabel={`${vendor.legal_name} · ${relationship.service_name}`} relationshipID={relationship.id}/>}
  {current === "HISTORY" && <VendorResponseHistory relationshipID={relationship.id} serviceName={relationship.service_name} onUpdated={onFormWorkUpdated} active={section === "HISTORY"} refreshKey={formsRefreshKey}/>}
  {current === "DUE_DILIGENCE" && <>
  {assessmentState === "live" && !assessment && formState === "unavailable" ? <section className="vdd-workspace" aria-label="Due diligence" tabIndex={-1}><div className="vdd-state vdd-state-error" role="alert"><h2>Due-diligence forms are unavailable</h2><p>Approved collection forms could not be loaded for {relationship.service_name}. Try again before starting the assessment.</p><Button onPress={() => void onRefreshForms()}>Reload forms</Button></div></section> : <VendorDueDiligence
    relationship={record}
    accountableOwnerLabel={accountableOwnerLabel}
    assessment={assessment}
    review={review}
    reviewState={reviewState}
    form={form}
    forms={forms}
    requestOutcome={requestOutcome}
    requestOutcomeKind={requestOutcomeKind}
    viewState={effectiveAssessmentState}
    defaultReviewDueDate={recommendedReviewDate()}
    setupFailure={setupFailure}
    onRefresh={onRefreshAssessment}
    onSetUpForm={onSetUpForm}
    onOpenForms={onOpenForms}
    onStart={onStartAssessment}
    onSend={onSendAssessmentRequest}
    onPrepare={onPrepareAssessmentRequest}
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
    onApplyResponse={onApplyAssessmentResponse}
    onOpenRequest={onOpenRequest}
    onOpenMatter={onOpenMatter}
  />}
  <VendorActivationPanel relationship={relationship} reviewVersion={assessment?.version} onActivated={onActivated} onRefreshed={onRefreshed}/>
  </>}
  </>}</Tabs>
  </>;
}

function Fact({ label, value }: { label: string; value: string }) { return <div><dt>{label}</dt><dd>{value}</dd></div>; }

function VendorRegisterFormSummary({ summary, label, loading, disabled, onOpen }: { summary?: VendorFormSummary; label: string; loading: boolean; disabled: boolean; onOpen: (filter?: VendorFormsFilter) => void }) {
  if (!summary) return <div className="vendor-form-summary">{loading ? "Loading form work…" : "Form work counts unavailable"}</div>;
  return <div className="vendor-form-summary">
    <div className="vendor-form-summary__counts"><ActionLink isDisabled={disabled} aria-label={`${summary.outstanding_forms} outstanding forms for ${label}`} onPress={() => onOpen("AWAITING_VENDOR")}>{summary.outstanding_forms} outstanding {summary.outstanding_forms === 1 ? "form" : "forms"}</ActionLink><span>·</span><ActionLink isDisabled={disabled} aria-label={`${summary.overdue_forms} overdue forms for ${label}`} onPress={() => onOpen("OVERDUE")}>{summary.overdue_forms} overdue</ActionLink></div>
    <ActionLink isDisabled={disabled} aria-label={`${summary.awaiting_review} awaiting review for ${label}`} onPress={() => onOpen("AWAITING_REVIEW")}>{summary.awaiting_review} awaiting review</ActionLink>
    <ActionLink isDisabled={disabled} aria-label={`Highest assessed concern for ${label}`} onPress={() => onOpen(summary.highest_concern === "LOW" ? undefined : summary.highest_concern ? "WITH_RISKS" : "NOT_ASSESSED")}>{summary.highest_concern ? <><StatusBadge tone={summary.highest_concern === "CRITICAL" || summary.highest_concern === "HIGH" ? "error" : summary.highest_concern === "MODERATE" ? "warning" : "neutral"}>{humanize(summary.highest_concern)} concern</StatusBadge> · {summary.assessed_forms} assessed {summary.assessed_forms === 1 ? "form" : "forms"}</> : "No assessed concern recorded"}</ActionLink>
    <span>Checked <time dateTime={summary.observed_at}>{formatDateTime(summary.observed_at)}</time></span>
    {!!summary.partially_replaced_forms && <span>Includes {summary.partially_replaced_forms} partly replaced {summary.partially_replaced_forms === 1 ? "response" : "responses"}; remaining fields need review.</span>}
  </div>;
}

function vendorSummaryMatches(summary: VendorFormSummary | undefined, filter: VendorFormsFilter) {
  if (!summary) return false;
  if (filter === "AWAITING_VENDOR") return summary.outstanding_forms > 0;
  if (filter === "AWAITING_REVIEW") return summary.awaiting_review > 0;
  if (filter === "OVERDUE") return summary.overdue_forms > 0;
  if (filter === "NOT_ASSESSED") return summary.unassessed_forms > 0 || summary.assessed_forms === 0;
  if (filter === "HIGH_RISK") return summary.highest_concern === "HIGH" || summary.highest_concern === "CRITICAL";
  return Boolean(summary.highest_concern && summary.highest_concern !== "LOW");
}

function needsReviewView(status: VendorAssessment["status"]) {
  return status === "SUBMITTED" || status === "UNDER_REVIEW" || status === "COMPLETED";
}

function selectActiveVendorForms(forms: FormTemplate[]): VendorAssessmentFormOption[] {
  return forms.filter((form) => form.status === "ACTIVE" && form.is_current)
    .sort((left, right) => Number(right.code.trim().toUpperCase() === "VENDOR-DUE-DILIGENCE") - Number(left.code.trim().toUpperCase() === "VENDOR-DUE-DILIGENCE") || right.version - left.version)
    .map((current) => ({ id: current.id, version: current.version, name: current.name, presentation: current.presentation?.default_mode ?? "AUTOMATIC", fields: current.fields.map((field) => ({ id: field.id, label: field.label, collection_intent: field.collection_intent, target_key: field.record_target?.key })) }));
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
    <div><span className="eyebrow">{mode === "create" ? "New relationship" : "Current relationship"}</span><h2>{mode === "create" ? "Add a vendor and service" : "Edit vendor relationship"}</h2><p>{mode === "create" ? "Record the organization and its service. You are the initial accountable owner." : "Update the service, criticality, privacy role or dates."}</p></div>
    {formError && <div className="vendor-form-error" role="alert">{formError}</div>}
    <div className="vendor-form-grid">
      {mode === "create" ? <>
        {existingVendor ? <div className="vendor-identity-note">
          <span>Existing vendor selected</span><strong>{existingVendor.vendor.legal_name}</strong>
          <small>{[existingVendor.vendor.registration_ref, existingVendor.vendor.jurisdiction].filter(Boolean).join(" · ") || "No registration or jurisdiction recorded"}</small>
          <p>A separate service relationship will be created without duplicating the vendor identity.</p>
          <Button variant="quiet" onPress={onUseDifferent}>Use a different vendor</Button>
        </div> : <div className="vendor-existing-search">
          <TextField label="Legal name" isRequired errorMessage={errors.legalName} id="vendor-legal-name" value={form.legalName} onChange={(value) => onChange("legalName", value)} isInvalid={Boolean(errors.legalName)}/>
          <Button isDisabled={candidateState === "loading" || form.legalName.trim().length < 2} onPress={() => void onFindExisting()}>{candidateState === "loading" ? "Searching…" : "Find existing vendor"}</Button>
          {candidateState === "failed" && <p role="alert">Existing vendors could not be searched. You can retry or continue only if this is a new vendor.</p>}
          {candidateState === "ready" && candidates.length === 0 && <p>No existing vendor matched this name. Continue with the new vendor details.</p>}
          {candidateState === "ready" && candidates.length > 0 && <section className="vendor-match-list" aria-label="Possible vendor matches"><h3>Possible vendor matches</h3><p>Select an existing vendor, or continue only when this is a different organization.</p>{candidates.map((candidate) => <button type="button" className="vendor-match" key={candidate.relationship.id} aria-label={`Use ${candidate.vendor.legal_name} for a new service relationship`} onClick={() => onUseExisting(candidate)}><strong>{candidate.vendor.legal_name}</strong><span>{candidate.relationship.service_name}</span><small>{candidate.vendor.registration_ref || candidate.vendor.external_ref || "No reference recorded"}</small></button>)}</section>}
        </div>}
        {!existingVendor && <><TextField label="Trading name" id="vendor-trading-name" value={form.tradingName} onChange={(value) => onChange("tradingName", value)}/>
        <TextField label="Registration reference" id="vendor-registration" value={form.registrationRef} onChange={(value) => onChange("registrationRef", value)}/>
        <TextField label="Jurisdiction" id="vendor-jurisdiction" value={form.jurisdiction} onChange={(value) => onChange("jurisdiction", value)} placeholder="For example, Nigeria"/>
        <div className="vendor-field wide"><TextField label="Website" errorMessage={errors.websiteDomain} id="vendor-website" type="url" inputMode="url" autoComplete="url" maxLength={vendorIdentityLimits.websiteInput} value={form.websiteDomain} onChange={(value) => onChange("websiteDomain", value)} placeholder="https://vendor.example" isInvalid={Boolean(errors.websiteDomain)}/></div>
        <div className="vendor-field wide"><TextArea label="Registered address" errorMessage={errors.registeredAddress} id="vendor-registered-address" autoComplete="street-address" maxLength={vendorIdentityLimits.registeredAddress} rows={3} value={form.registeredAddress} onChange={(value) => onChange("registeredAddress", value)} isInvalid={Boolean(errors.registeredAddress)}/></div></>}
      </> : <div className="vendor-identity-note">
        <span>Vendor legal details</span><strong>{form.legalName}</strong>
        <small>{[form.registrationRef, form.jurisdiction].filter(Boolean).join(" · ") || "No registration or jurisdiction recorded"}</small>
        <p>These details are shared across your organization and cannot be changed from this service relationship.</p>
      </div>}
      <div className="vendor-field wide"><TextField label="Service supplied" isRequired errorMessage={errors.serviceName} id="vendor-service" value={form.serviceName} onChange={(value) => onChange("serviceName", value)} isInvalid={Boolean(errors.serviceName)}/></div>
      <SelectField<VendorCriticality> label="Criticality" isRequired allowsEmpty={false} placeholder="Select criticality" value={form.criticality} errorMessage={errors.criticality} isInvalid={!!errors.criticality} options={[{ id: "STANDARD", label: "Standard" }, { id: "IMPORTANT", label: "Important" }, { id: "CRITICAL", label: "Critical" }]} onChange={(value) => { if (value) onChange("criticality", value); }}/>
      <SelectField<VendorPrivacyRole> label="Privacy role" isRequired allowsEmpty={false} placeholder="Select privacy role" value={form.privacyRole} errorMessage={errors.privacyRole} isInvalid={!!errors.privacyRole} options={[{ id: "NONE", label: "No processing role" }, { id: "PROCESSOR", label: "Processor" }, { id: "JOINT_CONTROLLER", label: "Joint controller" }]} onChange={(value) => { if (value) onChange("privacyRole", value); }}/>
      {mode === "create" && !existingVendor && <><TextField label="Source system" errorMessage={errors.sourceID} id="vendor-source" value={form.sourceID} onChange={(value) => onChange("sourceID", value)} placeholder="For example, procurement"/><TextField label="Source reference" id="vendor-external-ref" value={form.externalRef} onChange={(value) => onChange("externalRef", value)}/></>}
      <TextField label="Effective date" id="vendor-effective" type="date" value={form.effectiveFrom} onChange={(value) => onChange("effectiveFrom", value)}/>
      <TextField label="Renewal date" errorMessage={errors.renewalAt} id="vendor-renewal" type="date" value={form.renewalAt} onChange={(value) => onChange("renewalAt", value)} isInvalid={Boolean(errors.renewalAt)}/>
    </div>
    <div className="vendor-form-actions"><Button onPress={onCancel} isDisabled={saving}>Cancel</Button><Button type="submit" variant="primary" isDisabled={saving}>{saving ? "Saving…" : mode === "create" ? "Add vendor relationship" : "Save vendor relationship"}</Button></div>
  </form>;
}

function optional(value: string) { return value.trim() || undefined; }
function apiDate(value: string) { return value ? `${value}T00:00:00Z` : undefined; }
function dateInput(value?: string) { return value?.slice(0, 10) ?? ""; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
function privacyLabel(value: VendorPrivacyRole) { return value === "NONE" ? "No processing role" : humanize(value); }
function formatDate(value?: string) { if (!value) return "Not recorded"; const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : "Not recorded"; }
function formatDateTime(value: string) { const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed)) : "Update time unavailable"; }
