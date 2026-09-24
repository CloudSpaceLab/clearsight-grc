import type { ProcessingActivity, ProcessingActivityPageWire, ProcessingActivityResponse, ProcessingActivityHistoryResponse, RegisterSummary } from "./ropaTypes";

const tenantID = "bank-demo";
const legalEntityID = "bank-ng";
const primaryActivityID = "ropa-activity-customer-account-opening";

const customerAccountOpening: ProcessingActivity = {
  id: primaryActivityID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-CUSTOMER-ACCOUNT-OPENING",
  name: "Customer account opening",
  description: "Sample record: collect and verify identity and contact details when a customer opens a personal account.",
  status: "OPEN",
  purpose: "Open and verify customer accounts for retail and small-business customers.",
  lawful_basis: "",
  controller: "Meridian Trust Bank Nigeria",
  processor: "Retail Banking Operations",
  automated_decision_making: false,
  data_subject_categories: "",
  personal_data_categories: "Name; date of birth; address; identification document",
  security_measures: "Encryption at rest and in transit; role-based access; masked reports",
  retention_period: "Seven years after account closure",
  start_date: "2024-02-12",
  next_review_date: "2026-10-02",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 3,
  created_at: "2026-08-14T09:00:00Z",
  updated_at: "2026-09-20T15:30:00Z",
  recipients: [
    { recipient: "Core banking platform", recipient_kind: "INTERNAL", is_cross_border: false, transfer_basis: "NOT_APPLICABLE" },
    { recipient: "Customer onboarding support provider", recipient_kind: "EXTERNAL", country_code: "GB", is_cross_border: true, transfer_basis: "STANDARD_CONTRACT_CLAUSES" },
  ],
  systems: [{ system_name: "Customer onboarding portal", system_kind: "APPLICATION" }],
  reviews: [{ id: "sample-review-account-opening-2026", created_at: "2026-08-14T09:00:00Z", due_date: "2026-10-02" }],
};

const loanApplication: ProcessingActivity = {
  id: "ropa-activity-loan-application",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-LOAN-APPLICATION",
  name: "Loan application assessment",
  description: "Sample record: collect applicant information so the credit team can assess a loan application and proposed terms.",
  status: "OPEN",
  purpose: "Assess personal and business loan applications.",
  lawful_basis: "",
  controller: "Meridian Trust Bank Nigeria",
  processor: "Credit Risk Operations",
  automated_decision_making: true,
  data_subject_categories: "Customers; prospective customers",
  personal_data_categories: "Name; employment history; income; credit history",
  security_measures: "Encryption at rest; restricted credit-team access",
  retention_period: "Seven years after loan closure or withdrawal",
  start_date: "2024-05-06",
  next_review_date: "2026-10-18",
  owner_principal_id: "Aisha Bello · Retail Credit Operations",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 5,
  created_at: "2026-08-10T10:00:00Z",
  updated_at: "2026-09-18T11:20:00Z",
  data_categories: [
    { category: "Financial profile", sensitivity: "SENSITIVE_BY_NATURE" },
    { category: "Identity", sensitivity: "DIRECT_PERSONAL" },
  ],
  systems: [{ system_name: "Credit decisioning platform", system_kind: "APPLICATION" }],
  reviews: [{ id: "sample-review-loan-2026", created_at: "2026-08-10T10:00:00Z", due_date: "2026-10-18" }],
};

const customerServiceChannel: ProcessingActivity = {
  id: "ropa-activity-customer-service-channel",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-CUSTOMER-SERVICE-CHANNEL",
  name: "Customer service channel monitoring",
  description: "Sample record: retain customer contact details and service interactions to investigate complaints and support account servicing.",
  status: "OPEN",
  purpose: "Support customer servicing and complaint investigation.",
  lawful_basis: "Legitimate interests",
  controller: "Meridian Trust Bank Nigeria",
  processor: "Customer Experience Operations",
  automated_decision_making: false,
  data_subject_categories: "Customers",
  personal_data_categories: "Name; telephone number; email address; contact history",
  security_measures: "Encryption at rest; masked contact details in reports",
  retention_period: "Five years after the last customer interaction",
  start_date: "2023-11-20",
  next_review_date: "2026-09-18",
  owner_principal_id: "Chinedu Nwosu · Customer Experience",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 4,
  created_at: "2026-08-12T08:30:00Z",
  updated_at: "2026-09-17T16:45:00Z",
  data_categories: [
    { category: "Contact details", sensitivity: "DIRECT_PERSONAL" },
    { category: "Interaction history", sensitivity: "INDIRECT_PERSONAL" },
  ],
  recipients: [{ recipient: "Customer Experience Operations", recipient_kind: "INTERNAL", is_cross_border: false, transfer_basis: "NOT_APPLICABLE" }],
  systems: [{ system_name: "Contact centre recording archive", system_kind: "DATABASE" }],
  reviews: [{ id: "sample-review-service-channel-2026", created_at: "2026-08-12T08:30:00Z", due_date: "2026-09-18" }],
};

const archivedCustomerRecords: ProcessingActivity = {
  id: "ropa-activity-archived-customer-records",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-ARCHIVED-CUSTOMER-RECORDS",
  name: "Archived customer records migration",
  description: "Sample record: retain a historical customer-record extract for controlled retrieval after routine processing ended.",
  status: "CLOSED",
  purpose: "Controlled retrieval of historic customer records.",
  lawful_basis: "Legal obligation",
  controller: "Meridian Trust Bank Nigeria",
  processor: "Records Management",
  automated_decision_making: false,
  data_subject_categories: "Customers",
  personal_data_categories: "Name; account number; transaction history; address",
  security_measures: "Read-only archive storage; restricted retrieval approval",
  retention_period: "Ten years after the applicable records period",
  start_date: "2022-03-01",
  end_date: "2026-08-31",
  owner_principal_id: "Records Management",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 6,
  created_at: "2026-08-05T09:00:00Z",
  updated_at: "2026-08-31T17:00:00Z",
  data_categories: [
    { category: "Account identifiers", sensitivity: "DIRECT_PERSONAL" },
    { category: "Transaction history", sensitivity: "SENSITIVE_BY_NATURE" },
  ],
  systems: [{ system_name: "Legacy customer records archive", system_kind: "FILE" }],
  reviews: [{ id: "sample-review-archive-2026", created_at: "2026-08-05T09:00:00Z", due_date: "2026-08-31", completed_at: "2026-08-28T14:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "Aminat Yusuf · Independent privacy reviewer" }],
};

const activities = [customerAccountOpening, loanApplication, customerServiceChannel, archivedCustomerRecords];

function registerSummary(freshness: "STALE" | "CURRENT"): RegisterSummary {
  const stale = freshness === "STALE";
  return {
    generated_at: stale ? "2026-09-24T07:15:00Z" : "2026-09-24T09:00:00Z",
    projection_version: "ropa-v1",
    freshness,
    source_high_water: stale ? "2026-09-24T07:12:00Z" : "2026-09-24T08:58:00Z",
    coverage: stale ? { population: 4, excluded: 1, unknown: 1 } : { population: 4 },
    counts: {
      total: 4,
      new: 0,
      open: 3,
      closed: 1,
      review_overdue: 1,
      missing_lawful_basis: 2,
      missing_owner: 1,
      no_data_subjects: 1,
      retired: 1,
    },
  };
}

function activityResponse(): ProcessingActivityResponse {
  return {
    state_label: "In progress",
    activity: customerAccountOpening,
    closure_blockers: ["lawful basis", "named owner", "data subject category", "completed review"],
  };
}

function activityHistory(): ProcessingActivityHistoryResponse {
  return {
    events: [
      {
        id: "ropa-event-account-opening-created",
        tenant_id: tenantID,
        legal_entity_id: legalEntityID,
        aggregate_type: "PROCESSING_ACTIVITY",
        aggregate_id: primaryActivityID,
        aggregate_version: 1,
        type: "processing_activity.created",
        payload: { source: "Sample processing inventory", recorded_by: "Aisha Bello · Retail Banking Operations" },
        actor_type: "USER",
        actor_id: "sample-owner",
        occurred_at: "2026-08-14T09:00:00Z",
      },
      {
        id: "ropa-event-account-opening-updated",
        tenant_id: tenantID,
        legal_entity_id: legalEntityID,
        aggregate_type: "PROCESSING_ACTIVITY",
        aggregate_id: primaryActivityID,
        aggregate_version: 2,
        type: "processing_activity.updated",
        payload: { changed_fact: "Next review date", next_review_date: "2026-10-02" },
        actor_type: "USER",
        actor_id: "sample-dpo",
        occurred_at: "2026-09-12T10:15:00Z",
      },
      {
        id: "ropa-event-account-opening-transitioned",
        tenant_id: tenantID,
        legal_entity_id: legalEntityID,
        aggregate_type: "PROCESSING_ACTIVITY",
        aggregate_id: primaryActivityID,
        aggregate_version: 3,
        type: "processing_activity.transitioned",
        payload: { from: "NEW", to: "OPEN", reason: "Sample record entered annual review" },
        actor_type: "USER",
        actor_id: "sample-dpo",
        occurred_at: "2026-09-20T15:30:00Z",
      },
    ],
    has_more: false,
  };
}

function page(url: URL): ProcessingActivityPageWire {
  const search = (url.searchParams.get("search") ?? "").trim().toLowerCase();
  const status = (url.searchParams.get("status") ?? "").trim().toUpperCase();
  const includeRetired = url.searchParams.get("include_retired") === "true";
  const rows = activities.filter((activity) => {
    if (!includeRetired && activity.end_date) return false;
    if (status && activity.status !== status) return false;
    if (search && !`${activity.name} ${activity.code} ${activity.purpose} ${activity.description}`.toLowerCase().includes(search)) return false;
    return true;
  });
  return { rows, has_more: false };
}

function currentEvidenceVariant(): "STALE" | "CURRENT" {
  // The registered desktop register capture carries the stale, known-coverage
  // state; the narrow register capture carries the current, unknown-coverage
  // state so both honesty states are visible in the retained evidence.
  const fixture = new URLSearchParams(window.location.search).get("fixture") ?? "";
  if (fixture.includes("unknown") || fixture.includes("current")) return "CURRENT";
  if (fixture.includes("stale")) return "STALE";
  return window.innerWidth > 0 && window.innerWidth <= 760 ? "CURRENT" : "STALE";
}

declare global {
  interface Window {
    ropaEvidenceReads?: string[];
  }
}

// Imported only by the isolated evidence entry. The transport keeps the sample
// population bounded and labelled without changing the production API client.
export function installRopaEvidence() {
  const previous = globalThis.fetch.bind(globalThis);
  const variant = currentEvidenceVariant();
  window.ropaEvidenceReads = [];
  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin);
    const method = (init?.method ?? "GET").toUpperCase();
    const path = url.pathname;
    const recordRead = () => window.ropaEvidenceReads?.push(`${path}${url.search}`);
    const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });

    if (method === "GET" && path === "/api/v1/ropa/dashboard") {
      recordRead();
      return json(registerSummary(variant));
    }
    if (method === "GET" && path === "/api/v1/ropa/processing-activities") {
      recordRead();
      return json(page(url));
    }
    const history = /^\/api\/v1\/ropa\/processing-activities\/([^/]+)\/history$/.exec(path);
    if (method === "GET" && history) {
      recordRead();
      if (decodeURIComponent(history[1]!) !== primaryActivityID) return json({ error: { code: "ropa_activity_not_found", message: "The processing activity is not available in this legal entity." } }, 404);
      return json(activityHistory());
    }
    const detail = /^\/api\/v1\/ropa\/processing-activities\/([^/]+)$/.exec(path);
    if (method === "GET" && detail) {
      recordRead();
      if (decodeURIComponent(detail[1]!) !== primaryActivityID) return json({ error: { code: "ropa_activity_not_found", message: "The processing activity is not available in this legal entity." } }, 404);
      return json(activityResponse());
    }
    return previous(input, init);
  };
}
