import { useState } from "react";
import type { VendorFormRow, VendorFormSummary, VendorFormsPage } from "./vendorFormsApi";
import type { VendorAssessment, VendorAssessmentConclusion } from "./vendorAssessmentTypes";
import type { ResponseAssessmentDetail } from "./formAssessmentApi";
import { VendorComplianceOverview } from "./components/VendorComplianceOverview";
import { Button, Notice } from "./components/ui";

// Imported only by evidenceMain. These labelled records never reach the customer build.
export const vendorComplianceEvidenceStates = ["empty", "gaps", "incomplete", "reused", "expired", "partially-replaced", "awaiting-review", "satisfactory", "conditional", "adverse", "unavailable", "pagination", "restricted", "freshness-unknown"] as const;
export type VendorComplianceEvidenceState = typeof vendorComplianceEvidenceStates[number];
const observedAt = "2026-09-09T12:00:00Z";
const relationshipID = "sample-overview-card-processing";
let recovered = false;
const baseRow: VendorFormRow = {
  request_id: "sample-compliance-request", relationship_id: relationshipID, response_id: "sample-compliance-response", form_template_id: "sample-third-party-risk", form_template_version: 3,
  title: "Sample · Third Party Risk Compliance", purpose: "Review assurance documents and contractual safeguards for card processing.", response_state: "SUBMITTED", current: true, response_currency: "CURRENT", outdated: false,
  // This deadline is past on purpose: request expiry must not make a submitted response outdated.
  deadline: "2026-08-31T16:00:00Z", updated_at: observedAt, submitted_at: "2026-09-07T10:00:00Z", required_count: 5, answered_required: 5, missing_fields: [], required_reviews: 0, completed_reviews: 0, assessment_state: "ASSESSED", attention_items: [],
};
function assessment(conclusion: VendorAssessmentConclusion): VendorAssessment {
  return { id: "sample-overview-assessment", tenant_id: "sample-bank", legal_entity_id: "sample-entity", relationship_id: relationshipID, review_kind: "PERIODIC", source_trigger: "Sample annual review", stable_episode_key: "sample-review-2026", status: "COMPLETED", form_template_id: baseRow.form_template_id, form_template_version: 3, review_due_at: "2026-09-15T16:00:00Z", started_by_principal_id: "sample-vendor-owner", started_at: "2026-09-01T10:00:00Z", completed_at: "2026-09-08T12:00:00Z", reviewer_principal_id: "sample-independent-reviewer", conclusion, next_review_recommended_at: "2027-09-08T12:00:00Z", version: 4, created_at: "2026-09-01T10:00:00Z", updated_at: observedAt };
}
export function vendorComplianceScenario(state: string) {
  const summary: VendorFormSummary = { relationship_id: relationshipID, outstanding_forms: 0, overdue_forms: 0, submitted_forms: 1, awaiting_review: 0, unassessed_forms: 0, assessed_forms: 1, highest_concern: "LOW", outdated_forms: 0, freshness_unknown_forms: 0, partially_replaced_forms: 0, observed_at: observedAt };
  let rows: VendorFormRow[] = [structuredClone(baseRow)];
  let review: VendorAssessment | null = null;
  if (state === "empty" || state === "unavailable") { rows = []; Object.assign(summary, { submitted_forms: 0, assessed_forms: 0, highest_concern: undefined }); }
  if (state === "gaps") {
    rows[0]!.title = "Sample · Card processing service checks";
    rows[0]!.attention_items = [
      { field_id: "iso27001", label: "ISO 27001 certificate", state: "MISSING", source: "RESPONSE" },
      { field_id: "pci", label: "PCI DSS attestation", state: "EXPIRED", source: "RESPONSE" },
      { rule_id: "audit-rights", label: "Contractual audit rights", state: "GAP", source: "RESPONSE" },
      { rule_id: "vapt", label: "Vulnerability and penetration testing", state: "GAP", source: "REVIEW" },
    ];
    Object.assign(rows[0]!, { outdated: true, assessment_state: "AWAITING_REVIEW", required_reviews: 3, completed_reviews: 1 });
    Object.assign(summary, { outdated_forms: 1, highest_concern: undefined, awaiting_review: 1, unassessed_forms: 1, assessed_forms: 0 });
  }
  if (state === "incomplete") {
    rows[0] = { ...baseRow, response_id: undefined, response_state: "IN_PROGRESS", submitted_at: undefined, deadline: "2026-09-06T16:00:00Z", answered_required: 3, assessment_state: undefined, missing_fields: [{ id: "iso22301", label: "ISO 22301 certificate" }, { id: "vapt", label: "Vulnerability and penetration test report" }] };
    Object.assign(summary, { outstanding_forms: 1, overdue_forms: 1, submitted_forms: 0, assessed_forms: 0, highest_concern: undefined });
  }
  if (state === "reused") {
    rows[0] = { ...baseRow, response_id: undefined, response_state: "NO_VENDOR_ACTION", submitted_at: undefined, required_count: 3, answered_required: 0, held_required: 3, required_reviews: 3, assessment_state: "AWAITING_REVIEW" };
    Object.assign(summary, { submitted_forms: 0, assessed_forms: 0, awaiting_review: 1, highest_concern: undefined });
  }
  if (state === "expired") { rows[0]!.attention_items = [{ field_id: "pci", label: "PCI DSS attestation", state: "EXPIRED", source: "RESPONSE" }]; rows[0]!.outdated = true; summary.outdated_forms = 1; }
  if (state === "partially-replaced") { rows[0]!.response_currency = "PARTIALLY_REPLACED"; rows[0]!.outdated = true; summary.outdated_forms = 1; summary.partially_replaced_forms = 1; }
  if (state === "awaiting-review") { rows[0]!.required_reviews = 3; rows[0]!.completed_reviews = 1; rows[0]!.assessment_state = "AWAITING_REVIEW"; Object.assign(summary, { awaiting_review: 1, unassessed_forms: 1, assessed_forms: 0, highest_concern: undefined }); }
  if (state === "satisfactory") review = assessment("SATISFACTORY");
  if (state === "conditional") review = assessment("SATISFACTORY_WITH_CONDITIONS");
  if (state === "adverse") { review = assessment("UNSATISFACTORY"); summary.highest_concern = "HIGH"; }
  if (state === "pagination") { Object.assign(summary, { submitted_forms: 2, assessed_forms: 2 }); review = assessment("SATISFACTORY"); }
  if (state === "freshness-unknown") { rows[0]!.outdated = null; summary.freshness_unknown_forms = 1; review = assessment("SATISFACTORY"); }
  if (state === "restricted") { rows[0]!.title = "Sample · Authorized payment-service review"; Object.assign(summary, { submitted_forms: 1, assessed_forms: 0, unassessed_forms: 1, highest_concern: undefined }); }
  const page: VendorFormsPage = { items: rows, observed_at: observedAt, ...(state === "pagination" ? { next_cursor: "sample-overview-page-2" } : {}) };
  return { summary, assessment: review, page };
}
function responseAssessment(state: string, id: string): ResponseAssessmentDetail {
  const current = state !== "partially-replaced";
  const auditField = { id: "audit-rights", section_id: "assurance", label: "Does the contract include audit rights?", type: "yes_no" as const, required: true, options: ["Yes", "No"], assessment: { mode: "NONE" as const, required: false, weight: 100 } };
  if (state !== "gaps") return { response_id: id, form_template_id: baseRow.form_template_id, form_template_version: 3, version: 1, current, state: "NOT_REQUIRED", required_count: 0, reviewed_required_count: 0, reviewed_count: 0, may_review: false, fields: [{ field: auditField, answer: { text: "Yes" }, may_review: false }] };
  const review = { mode: "MANUAL" as const, required: true, weight: 100, rubric: [{ id: "missing", label: "Evidence does not meet the requirement", points: 100 }, { id: "accepted", label: "Evidence accepted", points: 0 }] };
  return { response_id: id, form_template_id: baseRow.form_template_id, form_template_version: 3, version: 1, current, state: "IN_REVIEW", required_count: 3, reviewed_required_count: 1, reviewed_count: 1, may_review: false,
    fields: [
      { field: auditField, answer: { text: "No" }, may_review: false },
      { field: { id: "iso27001", label: "ISO 27001 certificate", type: "file", required: false, assessment: review }, answer: {}, may_review: false },
      { field: { id: "pci", label: "PCI DSS attestation", type: "file", required: false, assessment: review }, answer: { text: "Attestation expired on 1 September 2026." }, may_review: false },
      { field: { id: "vapt", label: "Vulnerability and penetration testing", type: "file", required: false, assessment: review }, answer: { text: "Testing report submitted for independent review." }, may_review: false, decision: { id: "sample-vapt-review", field_id: "vapt", outcome_id: "missing", points: 100, rationale: "The submitted report excludes the card processing service.", reviewer_id: "sample-independent-reviewer", reviewer_display_name: "Sample independent reviewer", assessed_at: observedAt } },
    ],
    score_profile: { version: "sample-service-checks-v3", mode: "COMPLIANCE", direction: "HIGH_IS_POOR", contributions: [], rules: [{ id: "audit-rights", label: "Contractual audit rights", predicate: { field_id: "audit-rights", operator: "EQUALS", values: ["No"] }, effect: { kind: "FLOOR", value: 100 } }], bands: [{ band: "LOW", from: 0, through: 49 }, { band: "HIGH", from: 50, through: 100 }] },
    automatic_score: { mode: "COMPLIANCE", direction: "HIGH_IS_POOR", state: "FINAL", profile_version: "sample-service-checks-v3", raw_score: 100, adverse_score: 100, band: "HIGH", coverage: 100, final: true, calculated_at: observedAt, rule_results: [{ id: "audit-rights", matched: true, outcome: "MATCHED", effect: "FLOOR", value: 100 }] },
  };
}

declare global { interface Window { vendorComplianceEvidenceReads?: string[] } }
export function installVendorComplianceEvidence() {
  const fixture = new URLSearchParams(window.location.search).get("fixture") ?? "";
  if (!fixture.startsWith("vendor-compliance-")) return;
  const state = fixture.replace(/^vendor-compliance-(?:app-)?/, "");
  recovered = false; window.vendorComplianceEvidenceReads = [];
  const original = globalThis.fetch.bind(globalThis);
  globalThis.fetch = async (input, init) => {
    const url = new URL(typeof input === "string" ? input : input instanceof URL ? input.href : input.url, window.location.origin);
    const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
    const data = vendorComplianceScenario(state === "unavailable" && recovered ? "empty" : state);
    const forms = /^\/api\/v1\/vendors\/([^/]+)\/forms$/.exec(url.pathname);
    const current = /^\/api\/v1\/vendors\/([^/]+)\/assessments\/current$/.exec(url.pathname);
    const response = /^\/api\/v1\/forms\/responses\/([^/]+)\/assessment$/.exec(url.pathname);
    if (forms || current || response || url.pathname === "/api/v1/vendors/form-summaries") window.vendorComplianceEvidenceReads!.push(`${url.pathname}${url.search}`);
    if (state === "unavailable" && !recovered && (forms || url.pathname === "/api/v1/vendors/form-summaries")) return json({ error: { message: "Sample vendor forms are unavailable. Retry the check." } }, 503);
    if (forms) {
      const items = url.searchParams.has("cursor") ? [{ ...baseRow, request_id: "sample-privacy-request", response_id: "sample-privacy-response", title: "Sample · Privacy and data protection review" }] : data.page.items;
      return json({ ...data.page, items: items.map(row => ({ ...row, relationship_id: decodeURIComponent(forms[1]!) })), next_cursor: url.searchParams.has("cursor") ? undefined : data.page.next_cursor });
    }
    if (url.pathname === "/api/v1/vendors/form-summaries") return json({ items: (url.searchParams.get("relationship_ids") ?? relationshipID).split(",").map(id => ({ ...data.summary, relationship_id: id })) });
    if (current) return json({ assessment: data.assessment ? { ...data.assessment, relationship_id: decodeURIComponent(current[1]!) } : null });
    if (response) return json(responseAssessment(state, decodeURIComponent(response[1]!)));
    return original(input, init);
  };
}
export function VendorComplianceEvidencePage({ state }: { state: string }) {
  const [refreshKey, setRefreshKey] = useState(0);
  const [destination, setDestination] = useState("");
  const data = vendorComplianceScenario(state === "unavailable" && recovered ? "empty" : state);
  const unavailable = state === "unavailable" && !recovered;
  const updated = () => { recovered = true; setRefreshKey(key => key + 1); };
  return <main style={{ maxInlineSize: "64rem", margin: "0 auto", padding: "var(--cs-space-page)", minInlineSize: 0, minBlockSize: "100vh" }}>
    <div id="cs-overlay-root" className="cs-overlay-root" aria-live="off"/>
    <Notice tone="info">Sample data · Vendor service records checked 9 September 2026.</Notice>
    <h1>Sample · Card processing</h1>
    {destination ? <section aria-label="Sample destination"><h2>{destination}</h2><p>Sample destination for the card processing service. No request was sent.</p><Button onPress={() => setDestination("")}>Back to compliance</Button></section> : <VendorComplianceOverview relationshipID={relationshipID} serviceName="Card processing" summary={unavailable ? undefined : data.summary} summaryState={unavailable ? "unavailable" : "live"} assessment={data.assessment} assessmentState={unavailable ? "unavailable" : "live"} refreshKey={refreshKey} onOpenForms={() => setDestination("Vendor forms")} onOpenDueDiligence={() => setDestination("Vendor due diligence")} onRequestForm={() => setDestination("Request vendor form")} onOpenRequest={() => setDestination("Vendor form request")} onUpdated={updated}/>}
  </main>;
}
