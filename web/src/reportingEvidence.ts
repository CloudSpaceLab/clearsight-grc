import type { ReportDefinition, ReportDefinitionRevision, ReportFilterExpression, ReportFilterFieldDefinition, ReportRun } from "./reportingTypes";

const tenantID = "bank-demo";
const legalEntityID = "bank-ng";
const activeDefinitionID = "report-active";
const pendingDefinitionID = "report-pending";
const reviewedDefinitionID = "report-reviewed";
const readyRunID = "report-ready";
const failedRunID = "report-failed";

const makerID = "Amina Yusuf · proposer";
const reviewerID = "Tunde Adebayo · reviewer";
const authorizerID = "Ngozi Eze · authorizer";
const performerID = "Kemi Adebayo · report performer";

const fields: ReportFilterFieldDefinition[] = [
  { field: "status", label: "Processing activity status", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: true },
  { field: "lawful_basis", label: "Lawful basis", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: true },
  { field: "owner_principal_id", label: "Named owner", dataset: "PROCESSING_ACTIVITIES", operators: ["is", "is_not"], indexed: true },
  { field: "program_id", label: "Related program", dataset: "PROCESSING_ACTIVITIES", operators: ["is", "is_not"], indexed: true },
  { field: "matter_id", label: "Related issue or change", dataset: "PROCESSING_ACTIVITIES", operators: ["is", "is_not"], indexed: true },
  { field: "automated_decision_making", label: "Automated decision making", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: false },
  { field: "cross_border_transfer", label: "Cross-border transfer", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: true },
  { field: "review_overdue", label: "Review overdue", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: true },
  { field: "missing_lawful_basis", label: "Lawful basis not recorded", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: false },
  { field: "missing_owner", label: "Owner not recorded", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: false },
  { field: "missing_data_subjects", label: "Data subject categories not recorded", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: false },
  { field: "name", label: "Activity name contains", dataset: "PROCESSING_ACTIVITIES", operators: ["contains"], indexed: false },
  { field: "status", label: "Program status", dataset: "PROGRAMS", operators: ["is"], indexed: true },
  { field: "owner_principal_id", label: "Program owner", dataset: "PROGRAMS", operators: ["is", "is_not"], indexed: true },
  { field: "overall_state", label: "Calculated Program state", dataset: "PROGRAMS", operators: ["is"], indexed: true },
  { field: "jurisdiction", label: "Program jurisdiction", dataset: "PROGRAMS", operators: ["is"], indexed: false },
  { field: "has_open_matters", label: "Open issues and changes", dataset: "PROGRAMS", operators: ["is"], indexed: true },
  { field: "status", label: "Issue or change status", dataset: "MATTER_EXCEPTIONS", operators: ["is"], indexed: true },
  { field: "owner_principal_id", label: "Issue or change owner", dataset: "MATTER_EXCEPTIONS", operators: ["is", "is_not"], indexed: true },
  { field: "matter_type", label: "Issue or change type", dataset: "MATTER_EXCEPTIONS", operators: ["is"], indexed: true },
  { field: "priority", label: "Priority", dataset: "MATTER_EXCEPTIONS", operators: ["is"], indexed: true },
  { field: "due_condition", label: "Due condition", dataset: "MATTER_EXCEPTIONS", operators: ["is"], indexed: true },
  { field: "program", label: "Related Program", dataset: "MATTER_EXCEPTIONS", operators: ["is", "is_not"], indexed: true },
  { field: "latest_verification_result", label: "Latest outcome result", dataset: "MATTER_EXCEPTIONS", operators: ["is"], indexed: true },
];

const emptyFilter: ReportFilterExpression = { kind: "group", operator: "and", children: [] };
const activeFilter: ReportFilterExpression = { kind: "group", operator: "and", children: [{ kind: "condition", field: "missing_lawful_basis", operator: "is", value: "true" }] };

const activeDefinition: ReportDefinition = {
  id: activeDefinitionID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "ROPA-OPEN-EXCEPTIONS",
  name: "Processing activities with open exceptions",
  description: "Sample data: processing activities that still need a lawful basis, named owner, data-subject category or completed review.",
  dataset: "PROCESSING_ACTIVITY_EXCEPTIONS",
  scope_kind: "LEGAL_ENTITY",
  format: "CSV",
  filter: activeFilter,
  status: "ACTIVE",
  current_version: 3,
  effective: true,
  checksum: "a".repeat(64),
  maker_id: makerID,
  reviewer_id: reviewerID,
  checker_id: authorizerID,
  reviewer_note: "Sample privacy review completed.",
  effective_from: "2026-09-20T08:00:00Z",
  submitted_at: "2026-09-18T08:00:00Z",
  approved_at: "2026-09-20T08:00:00Z",
  created_at: "2026-09-17T08:00:00Z",
  updated_at: "2026-09-20T08:00:00Z",
  version: 7,
};

const pendingDefinition: ReportDefinition = {
  id: pendingDefinitionID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "ROPA-QUARTERLY-ISSUES",
  name: "Quarterly issue and change review",
  description: "Sample data: open issues and changes that need an overdue-obligation review.",
  dataset: "MATTER_EXCEPTIONS",
  scope_kind: "LEGAL_ENTITY",
  format: "NDJSON",
  filter: { kind: "group", operator: "and", children: [{ kind: "condition", field: "due_condition", operator: "is", value: "OVERDUE" }] },
  status: "PENDING_REVIEW",
  current_version: 1,
  effective: false,
  checksum: "b".repeat(64),
  maker_id: makerID,
  submitted_at: "2026-09-23T08:00:00Z",
  created_at: "2026-09-22T08:00:00Z",
  updated_at: "2026-09-23T08:00:00Z",
  version: 2,
};

const reviewedDefinition: ReportDefinition = {
  id: reviewedDefinitionID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  code: "ROPA-PROGRAM-HEALTH",
  name: "Program health review",
  description: "Sample data: current Program operating status and calculated attention state.",
  dataset: "PROGRAMS",
  scope_kind: "LEGAL_ENTITY",
  format: "CSV",
  filter: { kind: "group", operator: "and", children: [{ kind: "condition", field: "overall_state", operator: "is", value: "AT_RISK" }] },
  status: "REVIEWED",
  current_version: 2,
  effective: false,
  checksum: "c".repeat(64),
  maker_id: makerID,
  reviewer_id: reviewerID,
  reviewer_note: "Sample review completed; activation remains with the authorizer.",
  submitted_at: "2026-09-22T08:00:00Z",
  created_at: "2026-09-21T08:00:00Z",
  updated_at: "2026-09-23T08:00:00Z",
  version: 3,
};

const definitions = [activeDefinition, pendingDefinition, reviewedDefinition];

const activeHistory: ReportDefinitionRevision = {
  definition_id: activeDefinitionID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  version: 3,
  base_version: 2,
  dataset: activeDefinition.dataset,
  scope_kind: activeDefinition.scope_kind,
  format: activeDefinition.format,
  filter: activeFilter,
  checksum: activeDefinition.checksum,
  maker_id: makerID,
  created_at: "2026-09-17T08:00:00Z",
  reviewed_by: reviewerID,
  reviewed_at: "2026-09-19T08:00:00Z",
  approved_by: authorizerID,
  approved_at: "2026-09-20T08:00:00Z",
  decision: "APPROVED",
  decision_note: "Sample privacy review and authorisation completed.",
};

const pendingHistory: ReportDefinitionRevision = {
  definition_id: pendingDefinitionID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  version: 1,
  base_version: 0,
  dataset: pendingDefinition.dataset,
  scope_kind: pendingDefinition.scope_kind,
  format: pendingDefinition.format,
  filter: pendingDefinition.filter!,
  checksum: pendingDefinition.checksum,
  maker_id: makerID,
  created_at: "2026-09-22T08:00:00Z",
  decision: "PROPOSED",
  decision_note: "Sample proposal submitted for independent review.",
};

const reviewedHistory: ReportDefinitionRevision = {
  definition_id: reviewedDefinitionID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  version: 2,
  base_version: 1,
  dataset: reviewedDefinition.dataset,
  scope_kind: reviewedDefinition.scope_kind,
  format: reviewedDefinition.format,
  filter: reviewedDefinition.filter!,
  checksum: reviewedDefinition.checksum,
  maker_id: makerID,
  created_at: "2026-09-21T08:00:00Z",
  reviewed_by: reviewerID,
  reviewed_at: "2026-09-23T08:00:00Z",
  decision: "REVIEWED",
  decision_note: "Sample review completed; activation remains with the authorizer.",
};

const histories: Record<string, ReportDefinitionRevision[]> = {
  [activeDefinitionID]: [activeHistory],
  [pendingDefinitionID]: [pendingHistory],
  [reviewedDefinitionID]: [reviewedHistory],
};

const readyRun: ReportRun = {
  id: readyRunID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  definition_id: activeDefinitionID,
  definition_version: 3,
  definition_code: activeDefinition.code,
  definition_checksum: activeDefinition.checksum,
  scope_kind: activeDefinition.scope_kind,
  requested_by_ref: performerID,
  as_of: "2026-09-24T08:30:00Z",
  filter: activeFilter,
  dataset: activeDefinition.dataset,
  format: activeDefinition.format,
  status: "READY",
  attempt_count: 1,
  row_count: 42,
  data_sha256: "d".repeat(64),
  manifest_sha256: "e".repeat(64),
  created_at: "2026-09-24T08:31:00Z",
  completed_at: "2026-09-24T08:32:00Z",
  expires_at: "2026-10-01T08:31:00Z",
  source_boundary: {
    captured_at: "2026-09-24T08:30:00Z",
    projection_version: "ropa-report-v3",
    source_high_water: { activities: "2026-09-24T08:29:00Z", matters: "2026-09-24T08:28:00Z" },
    population: 42,
    population_complete: true,
  },
};

const failedRun: ReportRun = {
  id: failedRunID,
  tenant_id: tenantID,
  legal_entity_id: legalEntityID,
  definition_id: activeDefinitionID,
  definition_version: 3,
  definition_code: activeDefinition.code,
  definition_checksum: activeDefinition.checksum,
  scope_kind: activeDefinition.scope_kind,
  requested_by_ref: performerID,
  as_of: "2026-09-24T07:30:00Z",
  filter: activeFilter,
  dataset: activeDefinition.dataset,
  format: activeDefinition.format,
  status: "FAILED",
  attempt_count: 1,
  row_count: 0,
  failure_code: "row_limit_exceeded",
  created_at: "2026-09-24T07:31:00Z",
  completed_at: "2026-09-24T07:32:00Z",
  expires_at: "2026-10-01T07:31:00Z",
  source_boundary: {
    captured_at: "2026-09-24T07:30:00Z",
    projection_version: "ropa-report-v3",
    source_high_water: { activities: "2026-09-24T07:29:00Z" },
    population: 10001,
    population_complete: true,
  },
};

declare global {
  interface Window {
    reportingEvidenceReads?: string[];
  }
}

export function installReportingEvidence() {
  const previous = globalThis.fetch.bind(globalThis);
  const boundedStopVariant = new URLSearchParams(window.location.search).get("fixture") === "report-run-failed";
  window.reportingEvidenceReads = [];
  globalThis.fetch = async (input, init) => {
    const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const url = new URL(raw, window.location.origin);
    const method = (init?.method ?? (input instanceof Request ? input.method : "GET")).toUpperCase();
    if (method !== "GET") return previous(input, init);
    const path = url.pathname;
    const recordRead = () => window.reportingEvidenceReads?.push(`${path}${url.search}`);
    const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });

    if (path === "/api/v1/ropa/reports/filter-fields") {
      recordRead();
      return json({ fields });
    }
    if (path === "/api/v1/ropa/reports/definitions") {
      recordRead();
      return json({ items: definitions });
    }
    const definitionHistory = /^\/api\/v1\/ropa\/reports\/definitions\/([^/]+)\/history$/.exec(path);
    if (definitionHistory) {
      recordRead();
      const id = decodeURIComponent(definitionHistory[1]!);
      return histories[id] ? json({ items: histories[id] }) : notFound();
    }
    const definitionDetail = /^\/api\/v1\/ropa\/reports\/definitions\/([^/]+)$/.exec(path);
    if (definitionDetail) {
      recordRead();
      const definition = definitions.find((item) => item.id === decodeURIComponent(definitionDetail[1]!));
      return definition ? json(definition) : notFound();
    }
    if (path === "/api/v1/ropa/reports/runs") {
      recordRead();
      const items = boundedStopVariant ? [failedRun] : [readyRun, failedRun];
      const requestedDefinition = url.searchParams.get("definition_id");
      return json({ items: requestedDefinition ? items.filter((run) => run.definition_id === requestedDefinition) : items });
    }
    const runDetail = /^\/api\/v1\/ropa\/reports\/runs\/([^/]+)$/.exec(path);
    if (runDetail) {
      recordRead();
      const run = [readyRun, failedRun].find((item) => item.id === decodeURIComponent(runDetail[1]!));
      return run ? json(run) : notFound();
    }
    const download = /^\/api\/v1\/ropa\/reports\/runs\/([^/]+)\/download$/.exec(path);
    if (download) {
      recordRead();
      const run = [readyRun, failedRun].find((item) => item.id === decodeURIComponent(download[1]!));
      if (!run) return notFound();
      if (run.status !== "READY") return json({ error: { code: "report_run_not_ready", message: "This report file is not ready to download. Check the run state and try again when it is ready." } }, 409);
      return new Response("id,name\nsample,Customer account opening\n", { status: 200, headers: { "Content-Type": "text/csv; charset=utf-8", "Content-Disposition": "attachment; filename=\"ROPA-OPEN-EXCEPTIONS.csv\"" } });
    }
    return previous(input, init);
  };
}

function notFound() {
  return new Response(JSON.stringify({ error: { code: "report_not_found", message: "This report definition or run is not available in your legal entity." } }), { status: 404, headers: { "Content-Type": "application/json" } });
}
