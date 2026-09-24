import type { ProcessingActivity, ProcessingActivityPageWire, ProcessingActivityResponse, ProcessingActivityHistoryResponse, RegisterSummary } from "./ropaTypes";

const tenantID = "bank-demo";
const legalEntityID = "bank-ng";
const primaryActivityID = "ropa-activity-customer-account-opening";

const sampleSystems = {
  finacle: "Finacle Treasury",
  fincore: "Fincore/Coligo",
  ringo: "Ringo sms",
  bvn: "BVN Link Portal/Matching System",
  entrust: "Entrust/Entrust Middleware (Credential Security)",
  fortiProxy: "FortiProxy, Analyzer, Manager, Gate (Firewall Management)",
  falcon: "Falcon (AD Security)",
  checkmarx: "Checkmarx (Software testing/code scanning)",
  card: "Card Management Portal/Instant card",
  softToken: "Soft Token",
  qradar: "Log management review (Qradar)",
  virusScan: "Virus Scan and Update",
  fim: "File Integrity Monitoring Review",
  patching: "Patching Process",
  accessReview: "User Access Control Review",
  penetration: "Internal & External Penetration Testing",
  cloudspacePOS: "Cloudspace OEM — POS Support/PTSP",
  cloudspaceMontgomery: "Cloudspace OEM — Montgomery Vault Services",
} as const;

const customerAccountOpening: ProcessingActivity = {
  id: primaryActivityID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-CUSTOMER-ACCOUNT-OPENING",
  name: "Customer account opening",
  description: "Sample data: collect and verify identity and contact details when a customer opens a personal account. The record is linked to the Digital Identity & KYC reference obligation; confirm the current applicability with the privacy office.",
  status: "OPEN",
  purpose: "Open and verify customer accounts and digital identity records",
  lawful_basis: "Contract",
  controller: "Meridian Trust Bank",
  processor: "Retail Banking Operations",
  automated_decision_making: false,
  data_subject_categories: "Customers; Prospective customers",
  personal_data_categories: "Name; date of birth; address; identification document",
  security_measures: "Encryption at rest and in transit; role-based access; BVN matching and soft-token authentication",
  retention_period: "Seven years after account closure",
  start_date: "2024-02-12",
  next_review_date: "2027-01-22",
  owner_principal_id: "Somto · BVN and Digital Identity",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 3,
  created_at: "2026-08-14T09:00:00Z",
  updated_at: "2026-09-20T15:30:00Z",
  data_categories: [
    { category: "Identity", sensitivity: "DIRECT_PERSONAL" },
    { category: "Contact", sensitivity: "DIRECT_PERSONAL" },
  ],
  systems: [
    { system_name: sampleSystems.bvn, system_kind: "APPLICATION" },
    { system_name: sampleSystems.softToken, system_kind: "APPLICATION" },
  ],
  reviews: [{ id: "sample-review-account-opening-2026", created_at: "2026-07-26T09:00:00Z", due_date: "2026-08-25", completed_at: "2026-08-18T09:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "Aminat Yusuf · Independent privacy reviewer" }],
};

const loanApplication: ProcessingActivity = {
  id: "ropa-activity-loan-application",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-LOAN-APPLICATION",
  name: "Loan application assessment",
  description: "Sample data: collect applicant financial information so the credit team can assess a loan application and proposed terms. The lawful basis remains open for confirmation in the supplied reference material.",
  status: "OPEN",
  purpose: "Assess personal and business loan applications",
  lawful_basis: "",
  controller: "Meridian Trust Bank",
  processor: "Credit Risk Operations",
  automated_decision_making: true,
  data_subject_categories: "Customers; prospective customers",
  personal_data_categories: "Name; employment history; income; credit history",
  security_measures: "Encryption at rest; restricted credit-team access; BVN matching and soft-token authentication",
  retention_period: "Seven years after loan closure or withdrawal",
  start_date: "2024-05-06",
  next_review_date: "2026-12-23",
  owner_principal_id: "Godspower · Fincore/Coligo",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 5,
  created_at: "2026-08-10T10:00:00Z",
  updated_at: "2026-09-18T11:20:00Z",
  data_categories: [
    { category: "Financial profile", sensitivity: "SENSITIVE_BY_NATURE" },
    { category: "Identity", sensitivity: "DIRECT_PERSONAL" },
  ],
  systems: [
    { system_name: sampleSystems.fincore, system_kind: "APPLICATION" },
    { system_name: sampleSystems.bvn, system_kind: "APPLICATION" },
    { system_name: sampleSystems.softToken, system_kind: "APPLICATION" },
  ],
  reviews: [{ id: "sample-review-loan-2026", created_at: "2026-07-26T09:00:00Z", due_date: "2026-08-25", completed_at: "2026-08-18T09:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "Aminat Yusuf · Independent privacy reviewer" }],
};

const customerServiceChannel: ProcessingActivity = {
  id: "ropa-activity-customer-service-channel",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-CUSTOMER-SERVICE-CHANNEL",
  name: "Customer service channel monitoring",
  description: "Sample data: record customer contact details and service interactions to investigate complaints and support account servicing. Ringo sms carries customer notifications, while the card service is used when the interaction concerns a card.",
  status: "OPEN",
  purpose: "Support customer servicing, card servicing and complaint investigation",
  lawful_basis: "Legitimate interests",
  controller: "Meridian Trust Bank",
  processor: "Customer Experience Operations",
  automated_decision_making: false,
  data_subject_categories: "Customers",
  personal_data_categories: "Name; telephone number; email address; contact and card-service history",
  security_measures: "Encryption at rest; masked contact details in reports; role-based access",
  retention_period: "Five years after the last customer interaction",
  start_date: "2023-11-20",
  next_review_date: "2026-09-17",
  owner_principal_id: "Tobi · Ringo sms and Card Management",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 4,
  created_at: "2026-08-12T08:30:00Z",
  updated_at: "2026-09-17T16:45:00Z",
  data_categories: [
    { category: "Contact details", sensitivity: "DIRECT_PERSONAL" },
    { category: "Interaction history", sensitivity: "INDIRECT_PERSONAL" },
  ],
  recipients: [{ recipient: "Customer Experience Operations", recipient_kind: "INTERNAL", is_cross_border: false, transfer_basis: "NOT_APPLICABLE" }],
  systems: [
    { system_name: sampleSystems.ringo, system_kind: "APPLICATION" },
    { system_name: sampleSystems.card, system_kind: "APPLICATION" },
  ],
  reviews: [{ id: "sample-review-service-channel-2026", created_at: "2026-07-26T09:00:00Z", due_date: "2026-08-25", completed_at: "2026-08-18T09:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "Aminat Yusuf · Independent privacy reviewer" }],
};

const archivedCustomerRecords: ProcessingActivity = {
  id: "ropa-activity-archived-customer-records",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-ARCHIVED-CUSTOMER-RECORDS",
  name: "Archived customer records migration",
  description: "Sample data: retain a historical customer-record extract from Finacle Treasury for controlled retrieval after routine processing and migration ended.",
  status: "CLOSED",
  purpose: "Controlled retrieval of historic customer records",
  lawful_basis: "Legal obligation",
  controller: "Meridian Trust Bank",
  processor: "Records Management",
  automated_decision_making: false,
  data_subject_categories: "Customers",
  personal_data_categories: "Name; account number; transaction history; address",
  security_measures: "Read-only archive storage; restricted retrieval approval",
  retention_period: "Ten years after the applicable records period",
  start_date: "2022-03-01",
  end_date: "2026-08-24",
  owner_principal_id: "Tobi · Finacle Treasury",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 6,
  created_at: "2026-08-05T09:00:00Z",
  updated_at: "2026-08-31T17:00:00Z",
  data_categories: [
    { category: "Account identifiers", sensitivity: "DIRECT_PERSONAL" },
    { category: "Transaction history", sensitivity: "SENSITIVE_BY_NATURE" },
  ],
  systems: [{ system_name: sampleSystems.finacle, system_kind: "APPLICATION" }],
  reviews: [{ id: "sample-review-archive-2026", created_at: "2026-07-26T09:00:00Z", due_date: "2026-08-25", completed_at: "2026-08-18T09:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "Aminat Yusuf · Independent privacy reviewer" }],
};

const paymentsAndTreasury: ProcessingActivity = {
  id: "ropa-activity-payments-treasury-operations",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-PAYMENTS-TREASURY-OPERATIONS",
  name: "Payments and treasury operations",
  description: "Sample data: process POS, card and treasury instructions through the bank's Nigerian payment estate. Cloudspace OEM provides POS Support/PTSP and Montgomery Vault Services. The open third-party register findings are no adopted information security management standard such as ISO 27001, no VAPT, no right-to-audit clause in the SLA, and no certificate of compliance with ISO 27001 and ISO 22301 for the PTSP. The workplan names Ebube for POS Support/PTSP and Hakeem for the register action. Review the open evidence and contract gaps before relying on this service.",
  status: "OPEN",
  purpose: "Process and monitor domestic card, POS and treasury transactions for Payment Systems, Instant Payments, Outsourcing Governance and Operational Resilience reference obligations",
  lawful_basis: "Contract",
  controller: "Meridian Trust Bank",
  processor: "Cloudspace OEM / POS Business",
  automated_decision_making: false,
  data_subject_categories: "Customers; Cardholders; Service providers",
  personal_data_categories: "Account and card identifiers; transaction data; merchant and terminal identifiers",
  security_measures: "Tokenisation; encryption in transit and at rest; role-based access; domestic processor oversight",
  retention_period: "Seven years after the applicable payment record period",
  start_date: "2023-04-01",
  next_review_date: "2026-03-31",
  owner_principal_id: "Hakeem · POS Business",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 2,
  created_at: "2026-08-06T09:00:00Z",
  updated_at: "2026-09-20T12:00:00Z",
  data_categories: [
    { category: "Payment transaction data", sensitivity: "SENSITIVE_BY_NATURE" },
    { category: "Account and card identifiers", sensitivity: "DIRECT_PERSONAL" },
    { category: "Merchant and terminal identifiers", sensitivity: "INDIRECT_PERSONAL" },
  ],
  recipients: [{ recipient: "Cloudspace OEM", recipient_kind: "EXTERNAL", is_cross_border: false, transfer_basis: "NOT_APPLICABLE" }],
  systems: [
    { system_name: sampleSystems.finacle, system_kind: "APPLICATION" },
    { system_name: sampleSystems.fincore, system_kind: "APPLICATION" },
    { system_name: sampleSystems.card, system_kind: "APPLICATION" },
    { system_name: sampleSystems.cloudspacePOS, system_kind: "THIRD_PARTY" },
    { system_name: sampleSystems.cloudspaceMontgomery, system_kind: "THIRD_PARTY" },
  ],
  reviews: [{ id: "sample-review-payments-treasury-2026", created_at: "2026-02-02T00:00:00Z", due_date: "2026-03-31", reviewer_principal_id: "Amina Yusuf · Independent privacy reviewer" }],
};

const securityMonitoring: ProcessingActivity = {
  id: "ropa-activity-security-monitoring-control-review",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-SECURITY-MONITORING-CONTROL-REVIEW",
  name: "Security monitoring and control review",
  description: "Sample data: review security events, vulnerability results, access decisions and infrastructure changes. The activity is supported by the Qradar, Checkmarx, Falcon, FortiProxy and Entrust controls listed in the supplied IT risk workplan. The source workplan names Sikiru for Qradar and file-integrity review, Ivason for virus, patching and access review, Ese for Entrust, Fawaz for FortiProxy, Adetutu for Falcon and Ginika for Checkmarx.",
  status: "OPEN",
  purpose: "Monitor security events and review infrastructure and application safeguards under Cybersecurity and User Access Control reference obligations",
  lawful_basis: "Legitimate interests",
  controller: "Meridian Trust Bank",
  processor: "Information Security",
  automated_decision_making: false,
  data_subject_categories: "Staff; Contractors; Customers",
  personal_data_categories: "Security and access events; employee and device identifiers; vulnerability findings",
  security_measures: "Central log review; vulnerability scanning; privileged-access review; firewall and credential monitoring",
  retention_period: "Seven years after the applicable security record period",
  start_date: "2024-09-24",
  next_review_date: "2026-11-23",
  owner_principal_id: "Sikiru · GRC",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 2,
  created_at: "2026-08-07T09:00:00Z",
  updated_at: "2026-09-20T12:00:00Z",
  data_categories: [
    { category: "Security and access events", sensitivity: "INDIRECT_PERSONAL" },
    { category: "Employee and device identifiers", sensitivity: "DIRECT_PERSONAL" },
    { category: "Vulnerability findings", sensitivity: "SENSITIVE_BY_NATURE" },
  ],
  systems: [
    { system_name: sampleSystems.qradar, system_kind: "APPLICATION" },
    { system_name: sampleSystems.virusScan, system_kind: "APPLICATION" },
    { system_name: sampleSystems.fim, system_kind: "APPLICATION" },
    { system_name: sampleSystems.patching, system_kind: "APPLICATION" },
    { system_name: sampleSystems.accessReview, system_kind: "MANUAL" },
    { system_name: sampleSystems.penetration, system_kind: "THIRD_PARTY" },
    { system_name: sampleSystems.checkmarx, system_kind: "APPLICATION" },
    { system_name: sampleSystems.fortiProxy, system_kind: "APPLICATION" },
    { system_name: sampleSystems.falcon, system_kind: "APPLICATION" },
    { system_name: sampleSystems.entrust, system_kind: "APPLICATION" },
  ],
  reviews: [{ id: "sample-review-security-monitoring-2026", created_at: "2026-07-26T09:00:00Z", due_date: "2026-08-25", completed_at: "2026-08-18T09:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "Aminat Yusuf · Independent privacy reviewer" }],
};

const azureUserAccess: ProcessingActivity = {
  id: "ropa-activity-azure-user-access-management",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-AZURE-USER-ACCESS-MANAGEMENT",
  name: "Azure user access management",
  description: "Sample data: Risk ID 72 from Sample IT Risk Exception Register (1).xlsx. The open finding records 160 stale staff and guest accounts active for over three months on the Azure portal, affecting 28 guest users and 133 staff members. The source record was published 29 October 2025, targeted 31 January 2026, and remains OPEN; the closure timeline is exceeded. Control: ISO 27002:2022 A.5.18. This is sample reference data, not legal advice.",
  status: "OPEN",
  purpose: "Review and remediate user and guest access on the Azure portal",
  lawful_basis: "",
  controller: "Meridian Trust Bank",
  processor: "CISO / Security Engineering / Technology Vendor Management",
  automated_decision_making: false,
  data_subject_categories: "Staff; Guests; Prospective customers",
  personal_data_categories: "Staff and guest account records; access and activity logs",
  security_measures: "Conditional access; privileged-access review; activity logging; credential security",
  retention_period: "Seven years after the applicable access-audit period",
  start_date: "2025-10-29",
  next_review_date: "2026-01-31",
  owner_principal_id: "CISO · Meridian Trust Bank",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 2,
  created_at: "2025-10-29T09:00:00Z",
  updated_at: "2026-09-20T12:00:00Z",
  data_categories: [
    { category: "Staff and guest account records", sensitivity: "DIRECT_PERSONAL" },
    { category: "Access and activity logs", sensitivity: "INDIRECT_PERSONAL" },
  ],
  systems: [
    { system_name: "Azure portal", system_kind: "APPLICATION" },
    { system_name: sampleSystems.falcon, system_kind: "APPLICATION" },
    { system_name: sampleSystems.entrust, system_kind: "APPLICATION" },
  ],
  reviews: [{ id: "sample-review-azure-user-access-2026", created_at: "2025-10-29T00:00:00Z", due_date: "2026-01-31", reviewer_principal_id: "Amina Yusuf · Independent privacy reviewer" }],
};

const azureDeviceCompliance: ProcessingActivity = {
  id: "ropa-activity-azure-device-compliance",
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "PA-AZURE-DEVICE-COMPLIANCE",
  name: "Azure device compliance management",
  description: "Sample data: Risk ID 82 from Sample IT Risk Exception Register (1).xlsx. The open finding records 1,315 stale devices, 6,352 uncompliant devices and 8,693 unmanaged devices on the Microsoft Entra device dashboard. The source record was published 29 October 2025, targeted 31 January 2026, and remains OPEN; the closure timeline is exceeded. Owner: CISO / Security Engr / TVM. Control: ISO 27001 A.8.1.1. This is sample reference data, not legal advice.",
  status: "OPEN",
  purpose: "Review and remediate device compliance on the Azure portal",
  lawful_basis: "Legitimate interests",
  controller: "Meridian Trust Bank",
  processor: "CISO / Security Engineering / Technology Vendor Management",
  automated_decision_making: false,
  data_subject_categories: "Staff; Guests; Service providers",
  personal_data_categories: "Device and user identifiers; device compliance status; device activity logs",
  security_measures: "Device compliance policy; conditional access; endpoint monitoring; firewall and credential review",
  retention_period: "Seven years after the applicable device-audit period",
  start_date: "2025-10-29",
  next_review_date: "2026-01-31",
  owner_principal_id: "CISO / Security Engr / TVM",
  required_authority_principal_id: "Amina Yusuf · Head of Data Privacy",
  version: 2,
  created_at: "2025-10-29T09:00:00Z",
  updated_at: "2026-09-20T12:00:00Z",
  data_categories: [
    { category: "Device and user identifiers", sensitivity: "DIRECT_PERSONAL" },
    { category: "Device compliance status", sensitivity: "INDIRECT_PERSONAL" },
    { category: "Device activity logs", sensitivity: "INDIRECT_PERSONAL" },
  ],
  systems: [
    { system_name: "Azure portal", system_kind: "APPLICATION" },
    { system_name: sampleSystems.fortiProxy, system_kind: "APPLICATION" },
    { system_name: sampleSystems.falcon, system_kind: "APPLICATION" },
    { system_name: sampleSystems.entrust, system_kind: "APPLICATION" },
  ],
  reviews: [{ id: "sample-review-azure-device-compliance-2026", created_at: "2025-10-29T00:00:00Z", due_date: "2026-01-31", reviewer_principal_id: "Amina Yusuf · Independent privacy reviewer" }],
};

const activities = [customerAccountOpening, loanApplication, customerServiceChannel, archivedCustomerRecords, paymentsAndTreasury, securityMonitoring, azureUserAccess, azureDeviceCompliance];

const closureBlockersByCode: Record<string, string[]> = {
  "PA-CUSTOMER-ACCOUNT-OPENING": [],
  "PA-LOAN-APPLICATION": ["lawful basis"],
  "PA-CUSTOMER-SERVICE-CHANNEL": [],
  "PA-ARCHIVED-CUSTOMER-RECORDS": [],
  "PA-PAYMENTS-TREASURY-OPERATIONS": ["completed review"],
  "PA-SECURITY-MONITORING-CONTROL-REVIEW": [],
  "PA-AZURE-USER-ACCESS-MANAGEMENT": ["lawful basis", "completed review"],
  "PA-AZURE-DEVICE-COMPLIANCE": ["completed review"],
};

function registerSummary(freshness: "STALE" | "CURRENT"): RegisterSummary {
  const stale = freshness === "STALE";
  return {
    generated_at: stale ? "2026-09-24T07:15:00Z" : "2026-09-24T09:00:00Z",
    projection_version: "ropa-v1",
    freshness,
    source_high_water: stale ? "2026-09-24T07:12:00Z" : "2026-09-24T08:58:00Z",
    coverage: stale ? { population: 8, excluded: 1, unknown: 1 } : { population: 8 },
    counts: {
      total: 8,
      new: 0,
      open: 7,
      closed: 1,
      review_overdue: 4,
      missing_lawful_basis: 2,
      missing_owner: 0,
      no_data_subjects: 0,
      retired: 1,
    },
  };
}

function activityResponse(activityID: string): ProcessingActivityResponse {
  const activity = activities.find((item) => item.id === activityID);
  if (!activity) return { state_label: "In progress", activity: customerAccountOpening, closure_blockers: [] };
  return {
    state_label: activity.status === "CLOSED" ? "Complete" : "In progress",
    activity,
    closure_blockers: closureBlockersByCode[activity.code] ?? [],
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
        payload: { changed_fact: "Next review date", next_review_date: "2027-01-22" },
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
    if (search) {
      const values = [activity.name, activity.code, activity.purpose, activity.description, ...(activity.systems ?? []).map((system) => system.system_name), ...(activity.recipients ?? []).map((recipient) => recipient.recipient)].join(" ").toLowerCase();
      if (!values.includes(search)) return false;
    }
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
      const id = decodeURIComponent(detail[1]!);
      if (!activities.some((activity) => activity.id === id)) return json({ error: { code: "ropa_activity_not_found", message: "The processing activity is not available in this legal entity." } }, 404);
      return json(activityResponse(id));
    }
    return previous(input, init);
  };
}
