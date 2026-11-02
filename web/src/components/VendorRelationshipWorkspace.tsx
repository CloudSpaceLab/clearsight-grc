import { useEffect, useRef, useState, type ReactNode } from "react";
import type { PrepareVendorCollectionInput, PrepareVendorCollectionResult } from "../vendorCollectionApi";
import { vendorAssessmentDocumentURL } from "../vendorAssessmentApi";
import type { cancelVendorAssessment, createVendorAssessmentDeficiency, reissueVendorAssessmentRequest, requestVendorAssessmentClarification, reviewVendorAssessmentDocument, sendVendorAssessmentRequest } from "../vendorAssessmentApi";
import type { ApplyVendorAssessmentResponseInput, CompleteVendorAssessmentInput, CurrentVendorAssessment, StartVendorAssessmentInput, VendorAssessment, VendorAssessmentApplicationResult, VendorAssessmentFormOption, VendorAssessmentReviewView, VendorAssessmentSendOutcome, VendorAssessmentSetupRetryOutcome } from "../vendorAssessmentTypes";
import type { VendorPrivacyRole, VendorRelationshipAggregate } from "../vendorTypes";
import type { VendorFormSummary, VendorFormsFilter } from "../vendorFormsApi";
import { VendorBrandIcon, vendorBrandLabel } from "./VendorBrandIcon";
import { VendorComplianceOverview } from "./VendorComplianceOverview";
import { VendorDueDiligence } from "./VendorDueDiligence";
import { VendorActivationPanel } from "./VendorActivationPanel";
import { VendorPortfolio } from "./VendorPortfolio";
import { VendorFormsPanel, VendorResponseHistory } from "./VendorFormsPanel";
import { VendorWorkPanel } from "./VendorWorkPanel";
import { DocumentBrowser } from "./documents/DocumentBrowser";
import { Button, StatusBadge, Tabs } from "./ui";
import "./vendor-relationship-workspace.css";

type LoadState = "loading" | "live" | "unavailable";
export type VendorSection = "OVERVIEW" | "FORMS" | "DOCUMENTS" | "DUE_DILIGENCE" | "ISSUES" | "HISTORY";
type RecordSection = "OVERVIEW" | "DUE_DILIGENCE" | "EVIDENCE" | "ISSUES" | "HISTORY";
type EvidenceSection = "FORMS" | "DOCUMENTS";
const recordSections = [
  { id: "OVERVIEW", label: "Overview" },
  { id: "DUE_DILIGENCE", label: "Due diligence" },
  { id: "EVIDENCE", label: "Evidence" },
  { id: "ISSUES", label: "Issues & actions" },
  { id: "HISTORY", label: "Activity" },
] satisfies Array<{ id: RecordSection; label: string }>;
const evidenceSections = [{ id: "FORMS", label: "Forms" }, { id: "DOCUMENTS", label: "Documents" }] satisfies Array<{ id: EvidenceSection; label: string }>;

export function VendorRelationshipWorkspace({ record, formSummary, summaryState, section, onSectionChange, assessment, assessmentSetup, assessmentState, review, reviewState, form, forms, formState, requestOutcome, requestOutcomeKind, onBack, onEdit, onEditIdentity, onRefreshAssessment, onRefreshForms, onSetUpForm, onOpenForms, onStartAssessment, onPrepareAssessmentRequest, onSendAssessmentRequest, onReissueAssessmentRequest, onRetryAssessmentSetup, onRefreshReview, onStartAssessmentReview, onRequestAssessmentClarification, onCreateAssessmentDeficiency, onReviewAssessmentDocument, onCompleteAssessmentReview, onCancelAssessment, onApplyAssessmentResponse, onOpenRequest, onOpenMatter, accountableOwnerLabel, onActivated, onRefreshed, onRequestForm, onFormWorkUpdated, formsRefreshKey, formsFilter }: {
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
  const root = useRef<HTMLElement>(null);
  const pendingReviewFocus = useRef(false);
  const [lastEvidenceSection, setLastEvidenceSection] = useState<EvidenceSection>(section === "DOCUMENTS" ? "DOCUMENTS" : "FORMS");
  const evidenceSection = section === "FORMS" || section === "DOCUMENTS" ? section : lastEvidenceSection;
  const selectedSection: RecordSection = section === "FORMS" || section === "DOCUMENTS" ? "EVIDENCE" : section;
  useEffect(() => { if (section === "FORMS" || section === "DOCUMENTS") setLastEvidenceSection(section); }, [section]);
  function revealDueDiligence() {
    const target = root.current?.querySelector<HTMLElement>(".vdd-workspace");
    if (!target) return false;
    target.setAttribute("tabindex", "-1");
    target.focus({ preventScroll: true });
    const reducedMotion = typeof window.matchMedia === "function" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    target.scrollIntoView?.({ behavior: reducedMotion ? "auto" : "smooth", block: "start" });
    return true;
  }
  useEffect(() => {
    if (section !== "DUE_DILIGENCE" || !pendingReviewFocus.current) return;
    if (revealDueDiligence()) pendingReviewFocus.current = false;
  }, [section]);
  function openDueDiligence() {
    if (section === "DUE_DILIGENCE") revealDueDiligence();
    else { pendingReviewFocus.current = true; onSectionChange("DUE_DILIGENCE"); }
  }
  function selectSection(value: RecordSection) { onSectionChange(value === "EVIDENCE" ? evidenceSection : value); }
  const effectiveAssessmentState = assessmentState === "live" && !assessment && formState === "loading" ? "loading" : assessmentState;
  const setupFailure = assessmentSetup?.state === "FAILED" ? setupFailureText(assessmentSetup.failure_code) : undefined;
  const assessmentLabel = assessmentState === "loading" ? "Loading" : assessmentState === "unavailable" ? "Unavailable" : assessment ? humanize(assessment.status) : "Not started";
  const summaryAvailable = summaryState === "live" && !!formSummary;
  const summaryLabel = summaryState === "loading" ? "Loading" : "Unavailable";
  const reviewLabel = relationship.status === "PROPOSED" || relationship.status === "UNDER_REVIEW" ? "Review onboarding" : "Review due diligence";
  const nextDescription = assessmentState === "loading" ? "The current due-diligence assessment is loading."
    : assessmentState === "unavailable" ? "Due diligence could not be loaded. Open the review to retry."
      : !assessment ? "Open due diligence to check the approved forms and start the relationship review."
        : assessment.status === "COMPLETED" ? "The assessment is complete. Check activation requirements separately before changing the relationship status."
          : assessment.status === "CANCELLED" ? "The assessment was cancelled. Open due diligence to start a new review."
          : assessment.status === "COLLECTING" ? "Review the request status and the evidence received for this assessment."
            : assessment.status === "SUBMITTED" || assessment.status === "UNDER_REVIEW" ? "Review the submitted evidence and outstanding assessment decisions."
              : "Open the current assessment to continue due diligence.";

  return <article className="vendor-relationship-workspace" ref={root}>
    <div className="vendor-relationship-back"><Button variant="quiet" onPress={onBack}>Back to vendor register</Button></div>
    <header className="vendor-relationship-header">
      <div className="vendor-relationship-identity"><VendorBrandIcon vendorID={vendor.id} legalName={vendor.legal_name} brand={record.brand} size="detail" decorative/>
        <div className="vendor-relationship-title"><span className="eyebrow">Vendor relationship</span><h1>{vendor.legal_name}</h1><p>{relationship.service_name}</p></div>
      </div>
      <div className="vendor-relationship-actions"><StatusBadge tone={relationship.status === "RESTRICTED" || relationship.status === "SUSPENDED" ? "warning" : "neutral"}>{humanize(relationship.status)}</StatusBadge><Button onPress={onEditIdentity}>Edit vendor details</Button><Button onPress={onEdit}>Edit vendor relationship</Button><Button onPress={openDueDiligence}>{reviewLabel}</Button></div>
    </header>
    <div className="vendor-relationship-navigation"><Tabs ariaLabel="Vendor sections" compactLabel="Vendor section" retainVisitedPanels items={recordSections} selectedKey={selectedSection} onSelectionChange={selectSection}>{(current) => <div className="vendor-relationship-grid">
      <div className="vendor-relationship-primary">
        {current === "OVERVIEW" && <section className="vendor-relationship-overview" aria-label="Vendor overview">
          <section className="vendor-relationship-card vendor-relationship-status" aria-label="Relationship status">
            <header><div><h2>Review status</h2><p>{summaryAvailable ? `Forms checked ${formatDate(formSummary.observed_at)}` : summaryState === "loading" ? "Loading form totals…" : "Form totals unavailable"}</p></div><StatusBadge>{humanize(relationship.status)}</StatusBadge></header>
            <dl className="vendor-relationship-steps">
              <Fact label="Relationship" value={humanize(relationship.status)}/>
              <Fact label="Due diligence" value={assessmentLabel}/>
              <Fact label="Submitted forms" value={summaryAvailable ? String(formSummary.submitted_forms) : summaryLabel}/>
              <Fact label="Awaiting review" value={summaryAvailable ? String(formSummary.awaiting_review) : summaryLabel}/>
            </dl>
            {assessmentState === "live" && assessment?.review_due_at && <p className="vendor-relationship-caption">Assessment review due {formatDate(assessment.review_due_at)}</p>}
          </section>
          <section className="vendor-relationship-card vendor-relationship-requirements" aria-label="Review requirements">
            <VendorComplianceOverview heading="Review requirements" relationshipID={relationship.id} serviceName={relationship.service_name} summary={formSummary} summaryState={summaryState} assessment={assessment} assessmentState={assessmentState} refreshKey={formsRefreshKey} onOpenForms={() => onSectionChange("FORMS")} onOpenDueDiligence={openDueDiligence} onRequestForm={onRequestForm} onUpdated={onFormWorkUpdated} onOpenRequest={onOpenRequest}/>
          </section>
        </section>}
        {current === "EVIDENCE" && <section className="vendor-relationship-card vendor-relationship-evidence" aria-label="Vendor evidence">
          <Tabs ariaLabel="Evidence views" retainVisitedPanels items={evidenceSections} selectedKey={evidenceSection} onSelectionChange={onSectionChange}>{(evidence) => evidence === "FORMS" ? <><VendorFormsPanel relationshipID={relationship.id} serviceName={relationship.service_name} onRequestForm={onRequestForm} onUpdated={onFormWorkUpdated} onOpenHistory={() => onSectionChange("HISTORY")} onOpenDueDiligence={openDueDiligence} refreshKey={formsRefreshKey} initialFilter={formsFilter} onOpenRequest={onOpenRequest}/><details className="vendor-linked-work"><summary>Linked vendor work</summary><VendorWorkPanel relationshipID={relationship.id} onOpenRequest={onOpenRequest}/></details></> : <DocumentBrowser scopeLabel={`${vendor.legal_name} · ${relationship.service_name}`} relationshipID={relationship.id}/>}</Tabs>
        </section>}
        {current === "ISSUES" && <section className="vendor-relationship-issues" aria-label="Issues and actions"><VendorPortfolio records={[record]} hasMore={false} onOpenMatter={onOpenMatter} detail refreshKey={formsRefreshKey}/></section>}
        {current === "HISTORY" && <section className="vendor-relationship-card vendor-relationship-activity" aria-label="Vendor activity"><VendorResponseHistory relationshipID={relationship.id} serviceName={relationship.service_name} onUpdated={onFormWorkUpdated} active={section === "HISTORY"} refreshKey={formsRefreshKey}/></section>}
        {current === "DUE_DILIGENCE" && <div className="vendor-relationship-diligence">
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

        </div>}
      </div>
      <aside className="vendor-relationship-context" aria-label="Relationship context">
        <section className="vendor-relationship-card vendor-relationship-facts"><h2>Relationship</h2>
          <dl><Fact label="Accountable owner" value={accountableOwnerLabel}/><Fact label="Service criticality" value={humanize(relationship.criticality)}/><Fact label="Privacy role" value={privacyLabel(relationship.privacy_role)}/><Fact label="Renewal date" value={formatDate(relationship.renewal_at)}/><Fact label="Jurisdiction" value={vendor.jurisdiction || "Not recorded"}/></dl>
        </section>
        <section className="vendor-relationship-card vendor-relationship-next"><h2>Next action</h2><p>{nextDescription}</p>{current !== "DUE_DILIGENCE" && <Button onPress={openDueDiligence}>Open due diligence</Button>}</section>
        <details className="vendor-relationship-card vendor-relationship-record-details"><summary>Vendor details and record history</summary>
          {vendor.trading_name && <p>Trading as {vendor.trading_name}</p>}
          <dl><Fact label="Logo" value={vendorBrandLabel(record.brand)}/><Fact label="Registration reference" value={vendor.registration_ref || "Not recorded"}/><Fact label="Website domain" value={vendor.website_domain || "Not recorded"}/><Fact label="Registered address" value={vendor.registered_address || "Not recorded"}/><Fact label="Effective date" value={formatDate(relationship.effective_from)}/><Fact label="Source" value={vendor.source_id && vendor.external_ref ? `${vendor.source_id} · ${vendor.external_ref}` : "Entered directly"}/><Fact label="Relationship updated" value={formatDateTime(relationship.updated_at)}/><Fact label="Relationship version" value={`Version ${relationship.version}`}/><Fact label="Vendor details version" value={`Vendor version ${vendor.version}`}/></dl>
        </details>
      </aside>
    </div>}</Tabs></div>
  </article>;
}

function Fact({ label, value }: { label: string; value: ReactNode }) { return <div><dt>{label}</dt><dd>{value}</dd></div>; }
function humanize(value: string) { return value.toLowerCase().replaceAll("_", " ").replace(/(^|\s)\S/g, (letter) => letter.toUpperCase()); }
function privacyLabel(value: VendorPrivacyRole) { return value === "NONE" ? "No processing role" : humanize(value); }
function formatDate(value?: string) { if (!value) return "Not recorded"; const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(parsed)) : "Not recorded"; }
function formatDateTime(value: string) { const parsed = Date.parse(value); return Number.isFinite(parsed) ? new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(parsed)) : "Update time unavailable"; }

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

