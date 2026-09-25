import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ReportingPage } from "./ReportingPage";
import type { ReportDefinition, ReportDefinitionRevision, ReportFilterFieldDefinition, ReportRun } from "../reportingTypes";

const fields: ReportFilterFieldDefinition[] = [
  { field: "status", label: "Processing activity status", dataset: "PROCESSING_ACTIVITIES", operators: ["is"], indexed: true },
  { field: "name", label: "Activity name contains", dataset: "PROCESSING_ACTIVITIES", operators: ["contains"], indexed: false },
  { field: "overall_state", label: "Calculated Program state", dataset: "PROGRAMS", operators: ["is"], indexed: true },
];

function definition(overrides: Partial<ReportDefinition> = {}): ReportDefinition {
  return {
    id: "definition-active",
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    code: "ROPA-OPEN-EXCEPTIONS",
    name: "Processing activities with open exceptions",
    description: "Reports activities that still need a required closure fact.",
    dataset: "PROCESSING_ACTIVITIES",
    scope_kind: "LEGAL_ENTITY",
    format: "CSV",
    filter: { kind: "group", operator: "and", children: [] },
    status: "ACTIVE",
    current_version: 3,
    effective: true,
    checksum: "a".repeat(64),
    maker_id: "Amina Yusuf · proposer",
    reviewer_id: "Tunde Adebayo · reviewer",
    checker_id: "Ngozi Eze · authorizer",
    reviewer_note: "Scope and filter checked.",
    effective_from: "2026-09-20T08:00:00Z",
    approved_at: "2026-09-20T08:00:00Z",
    created_at: "2026-09-18T08:00:00Z",
    updated_at: "2026-09-20T08:00:00Z",
    version: 7,
    ...overrides,
  };
}

const active = definition();
const pending = definition({
  id: "definition-pending",
  code: "ROPA-PENDING",
  name: "Quarterly processing review",
  status: "PENDING_REVIEW",
  effective: false,
  current_version: 1,
  maker_id: "Amina Yusuf · proposer",
  reviewer_id: undefined,
  checker_id: undefined,
  submitted_at: "2026-09-23T08:00:00Z",
  version: 2,
});
const reviewed = definition({
  id: "definition-reviewed",
  code: "ROPA-REVIEWED",
  name: "Cross-border transfer review",
  status: "REVIEWED",
  effective: false,
  current_version: 2,
  maker_id: "Amina Yusuf · proposer",
  reviewer_id: "Tunde Adebayo · reviewer",
  checker_id: undefined,
  reviewer_note: "The cross-border scope is ready for activation.",
  submitted_at: "2026-09-22T08:00:00Z",
  version: 3,
});

const readyRun: ReportRun = {
  id: "run-ready",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  definition_id: active.id,
  definition_version: 3,
  definition_code: active.code,
  definition_checksum: active.checksum,
  scope_kind: "LEGAL_ENTITY",
  requested_by_ref: "privacy-performer",
  as_of: "2026-09-24T08:30:00Z",
  filter: active.filter!,
  dataset: active.dataset,
  format: "CSV",
  status: "READY",
  attempt_count: 1,
  row_count: 42,
  created_at: "2026-09-24T08:31:00Z",
  completed_at: "2026-09-24T08:32:00Z",
  expires_at: "2026-10-01T08:31:00Z",
  source_boundary: {
    captured_at: "2026-09-24T08:30:00Z",
    projection_version: "ropa-report-v7",
    source_high_water: { processing_activities: "2026-09-24T08:29:00Z", matters: "2026-09-24T08:28:00Z" },
    population: 42,
    population_complete: true,
  },
};
const failedRun: ReportRun = {
  id: "run-failed",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  definition_id: active.id,
  definition_version: 3,
  definition_code: active.code,
  definition_checksum: active.checksum,
  scope_kind: "LEGAL_ENTITY",
  requested_by_ref: "privacy-performer",
  as_of: "2026-09-24T07:30:00Z",
  filter: active.filter!,
  dataset: active.dataset,
  format: "CSV",
  status: "FAILED",
  attempt_count: 1,
  row_count: 0,
  failure_code: "row_limit_exceeded",
  created_at: "2026-09-24T07:31:00Z",
  completed_at: "2026-09-24T07:32:00Z",
  expires_at: "2026-10-01T07:31:00Z",
  source_boundary: {
    captured_at: "2026-09-24T07:30:00Z",
    projection_version: "ropa-report-v7",
    source_high_water: { processing_activities: "2026-09-24T07:29:00Z" },
    population: 10001,
    population_complete: true,
  },
};

const history: ReportDefinitionRevision[] = [
  {
    definition_id: active.id,
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    version: 3,
    base_version: 2,
    dataset: active.dataset,
    scope_kind: active.scope_kind,
    format: active.format,
    filter: active.filter!,
    checksum: active.checksum,
    maker_id: "Amina Yusuf · proposer",
    created_at: "2026-09-18T08:00:00Z",
    reviewed_by: "Tunde Adebayo · reviewer",
    reviewed_at: "2026-09-19T08:00:00Z",
    approved_by: "Ngozi Eze · authorizer",
    approved_at: "2026-09-20T08:00:00Z",
    decision: "APPROVED",
    decision_note: "Scope and filter checked.",
  },
];

const api = vi.hoisted(() => ({
  listReportFilterFields: vi.fn(),
  listReportDefinitions: vi.fn(),
  listReportRuns: vi.fn(),
  getReportDefinitionHistory: vi.fn(),
  createReportDefinition: vi.fn(),
  transitionReportDefinition: vi.fn(),
  createReportRun: vi.fn(),
  downloadReportRun: vi.fn(),
}));

beforeEach(() => {
  vi.clearAllMocks();
  api.listReportFilterFields.mockResolvedValue({ fields });
  api.listReportDefinitions.mockResolvedValue([active, pending, reviewed]);
  api.listReportRuns.mockResolvedValue([readyRun, failedRun]);
  api.getReportDefinitionHistory.mockResolvedValue(history);
  api.createReportDefinition.mockResolvedValue(active);
  api.transitionReportDefinition.mockResolvedValue(active);
  api.createReportRun.mockResolvedValue(readyRun);
  api.downloadReportRun.mockResolvedValue({ blob: new Blob(["id,name"]), filename: "ROPA-OPEN-EXCEPTIONS.csv" });
});

function renderPage() {
  return render(<ReportingPage
    organizationName="Meridian Trust Bank"
    legalEntityName="Meridian Trust Bank Nigeria"
    loadFilterFields={api.listReportFilterFields}
    loadDefinitions={api.listReportDefinitions}
    loadRuns={api.listReportRuns}
    loadDefinitionHistory={api.getReportDefinitionHistory}
    createDefinition={api.createReportDefinition}
    transitionDefinition={api.transitionReportDefinition}
    createRun={api.createReportRun}
    downloadRun={api.downloadReportRun}
  />);
}

describe("ReportingPage", () => {
  it("lists definitions with human governance states and separate decision roles", async () => {
    renderPage();
    expect(await screen.findByRole("heading", { name: "Reports" })).toBeTruthy();
    const table = screen.getByRole("table", { name: "Report definitions" });
    const activeRow = within(table).getByRole("row", { name: /Processing activities with open exceptions/ });
    expect(within(activeRow).getByText("Active")).toBeTruthy();
    expect(within(activeRow).getByText("Whole legal entity")).toBeTruthy();
    expect(within(activeRow).getByText("Processing activities")).toBeTruthy();
    expect(within(activeRow).getByText("Proposer")).toBeTruthy();
    expect(within(activeRow).getByText("Reviewer")).toBeTruthy();
    expect(within(activeRow).getByText("Authorizer")).toBeTruthy();
    expect(screen.queryByText("PENDING_REVIEW")).toBeNull();
    expect(screen.queryByText("ACTIVE")).toBeNull();
  });

  it("disables run for a definition waiting for review and names the review step", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "Reports" });
    fireEvent.click(screen.getByRole("row", { name: /Quarterly processing review/ }));
    const panel = screen.getByRole("region", { name: "Selected report definition" });
    expect(within(panel).getByRole("button", { name: "Run report" })).toHaveProperty("disabled", true);
    expect(within(panel).getByText(/reviewer must complete the report review/i)).toBeTruthy();
    expect(within(panel).getByText(/authoris/i)).toBeTruthy();
  });

  it("distinguishes a reviewed definition from one still waiting for review", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "Reports" });
    fireEvent.click(screen.getByRole("row", { name: /Cross-border transfer review/ }));
    const panel = screen.getByRole("region", { name: "Selected report definition" });
    expect(within(panel).getByRole("button", { name: "Run report" })).toHaveProperty("disabled", true);
    expect(within(panel).getByText(/authorizer must activate this report/i)).toBeTruthy();
    expect(within(panel).queryByText(/reviewer must complete the report review/i)).toBeNull();
  });

  it("shows run freshness, source boundary and bounded-stop failure instead of a short successful report", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "Reports" });
    const table = screen.getByRole("table", { name: "Report runs" });
    const readyRow = within(table).getByRole("row", { name: /Ready/ });
    expect(within(readyRow).getByText(/As of/)).toBeTruthy();
    expect(within(readyRow).getByText(/Generated/)).toBeTruthy();
    expect(within(readyRow).getByText("42 rows")).toBeTruthy();
    expect(within(readyRow).getByText("ropa-report-v7")).toBeTruthy();
    expect(within(readyRow).getAllByText(/Processing Activities/i).length).toBeGreaterThan(0);
    expect(within(readyRow).getByText("Complete population")).toBeTruthy();

    const failedRow = within(table).getByRole("row", { name: /Failed at row limit/ });
    expect(within(failedRow).getByText("0 rows · no file produced")).toBeTruthy();
    expect(within(failedRow).getByText(/10,000-row ceiling/i)).toBeTruthy();
    expect(within(failedRow).getByText(/source projection version/i)).toBeTruthy();
    expect(within(failedRow).getByText(/source high-water/i)).toBeTruthy();
  });

  it("shows the selected definition's decision history with proposer, reviewer and authorizer", async () => {
    renderPage();
    await screen.findByRole("heading", { name: "Reports" });
    const historyRegion = screen.getByRole("region", { name: "Report definition history" });
    expect(await within(historyRegion).findByText(/proposed by/i)).toBeTruthy();
    expect(within(historyRegion).getByText(/reviewed by/i)).toBeTruthy();
    expect(within(historyRegion).getByText(/authorised by/i)).toBeTruthy();
    expect(within(historyRegion).getByText("Scope and filter checked.")).toBeTruthy();
  });

  it("names the checked population and next action in the empty state", async () => {
    api.listReportDefinitions.mockResolvedValue([]);
    renderPage();
    expect(await screen.findByText("No report definitions are recorded in this legal entity yet")).toBeTruthy();
    expect(screen.getByText(/next valid action/i)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Define a report" })).toBeTruthy();
  });
});
