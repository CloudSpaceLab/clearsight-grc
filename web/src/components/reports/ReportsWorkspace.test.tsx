import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReportDefinition, ReportRun } from "../../reportingTypes";
import { ReportsWorkspace } from "./ReportsWorkspace";

const vendorDefinition: ReportDefinition = {
  id: "vendor-template",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  code: "VENDOR-PORTFOLIO",
  name: "Vendor portfolio",
  description: "Current vendor relationships for the legal entity.",
  dataset: "VENDORS",
  scope_kind: "LEGAL_ENTITY",
  format: "XLSX",
  filter: { kind: "group", operator: "and", children: [] },
  status: "ACTIVE",
  current_version: 2,
  effective: true,
  checksum: "a".repeat(64),
  maker_id: "maker-1",
  reviewer_id: "reviewer-1",
  checker_id: "checker-1",
  effective_from: "2026-09-20T08:00:00Z",
  created_at: "2026-09-18T08:00:00Z",
  updated_at: "2026-09-20T08:00:00Z",
  version: 4,
};

const programDefinition: ReportDefinition = {
  ...vendorDefinition,
  id: "program-template",
  code: "PROGRAM-HEALTH",
  name: "Program health",
  dataset: "PROGRAMS",
};

const readyRun: ReportRun = {
  id: "run-ready",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  definition_id: vendorDefinition.id,
  definition_version: 2,
  definition_code: vendorDefinition.code,
  definition_checksum: vendorDefinition.checksum,
  scope_kind: "LEGAL_ENTITY",
  requested_by_ref: "principal-1",
  as_of: "2026-09-25T08:00:00Z",
  filter: { kind: "group", operator: "and", children: [] },
  dataset: "VENDORS",
  format: "XLSX",
  status: "READY",
  attempt_count: 1,
  row_count: 18,
  created_at: "2026-09-25T08:01:00Z",
  completed_at: "2026-09-25T08:02:00Z",
  expires_at: "2099-10-02T08:01:00Z",
  source_boundary: {
    captured_at: "2026-09-25T08:00:00Z",
    projection_version: "vendor-relationship-report.v1",
    source_high_water: { vendor_relationships: "2026-09-25T07:59:00Z" },
    population: 18,
    population_complete: false,
  },
};

const failedRun: ReportRun = {
  ...readyRun,
  id: "run-failed",
  definition_id: programDefinition.id,
  definition_code: programDefinition.code,
  dataset: "PROGRAMS",
  format: "CSV",
  status: "FAILED",
  row_count: 0,
  failure_code: "row_limit_exceeded",
  created_at: "2026-09-24T08:01:00Z",
  completed_at: "2026-09-24T08:02:00Z",
};

describe("ReportsWorkspace", () => {
  const loadDefinitions = vi.fn();
  const loadRunPage = vi.fn();
  const createRun = vi.fn();
  const downloadRun = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    loadDefinitions.mockResolvedValue([vendorDefinition, programDefinition]);
    loadRunPage.mockResolvedValue({ items: [readyRun, failedRun] });
    createRun.mockResolvedValue({
      ...readyRun,
      id: "run-queued",
      status: "QUEUED",
      row_count: 0,
      completed_at: undefined,
      created_at: "2026-09-25T09:00:00Z",
    });
    downloadRun.mockResolvedValue({ blob: new Blob(["report"]), filename: "vendor-portfolio.xlsx" });
  });

  function renderWorkspace() {
    return render(<ReportsWorkspace
      organizationName="Meridian Trust Bank"
      legalEntityName="Meridian Trust Bank Nigeria"
      loadDefinitions={loadDefinitions}
      loadRunPage={loadRunPage}
      createRun={createRun}
      downloadRun={downloadRun}
    />);
  }

  it("opens on the generated-report library rather than the template builder", async () => {
    renderWorkspace();

    expect(await screen.findByRole("heading", { name: "Reports" })).toBeTruthy();
    expect(screen.getByText("One place for generated reports across vendors, programs and work.")).toBeTruthy();

    const table = await screen.findByRole("table", { name: "Generated reports" });
    expect(within(table).getByRole("row", { name: /Vendor portfolio/ })).toBeTruthy();
    expect(within(table).getByRole("row", { name: /Program health/ })).toBeTruthy();
    const summary = screen.getByRole("region", { name: "Report library summary" });
    expect(within(summary).getByText("Available files").nextElementSibling?.textContent).toBe("1");
    expect(within(summary).getByText("Failed").nextElementSibling?.textContent).toBe("1");
    expect(loadRunPage).toHaveBeenCalledWith({ limit: 50 }, expect.any(AbortSignal));
    expect(loadDefinitions).toHaveBeenCalledWith(true, expect.any(AbortSignal));
  });

  it("generates a vendor report from an active governed template", async () => {
    renderWorkspace();
    await screen.findByRole("table", { name: "Generated reports" });

    fireEvent.click(screen.getByRole("button", { name: "Generate report" }));
    const dialog = await screen.findByRole("dialog", { name: "Generate report" });
    expect(within(dialog).getByText("Vendor portfolio")).toBeTruthy();
    expect(within(dialog).getByText("XLSX")).toBeTruthy();

    fireEvent.click(within(dialog).getByRole("button", { name: "Generate report" }));
    await waitFor(() => expect(createRun).toHaveBeenCalledWith(vendorDefinition.id, vendorDefinition.current_version));
    expect(await screen.findByText(/Vendor portfolio was queued/)).toBeTruthy();
    expect(screen.getByRole("row", { name: /Vendor portfolio/ })).toBeTruthy();
  });

  it("loads older history pages without duplicating runs already in the library", async () => {
    const olderRun: ReportRun = {
      ...readyRun,
      id: "run-older",
      created_at: "2026-09-20T08:01:00Z",
      completed_at: "2026-09-20T08:02:00Z",
    };
    loadRunPage
      .mockResolvedValueOnce({ items: [readyRun, failedRun], next_cursor: "older-page" })
      .mockResolvedValueOnce({ items: [readyRun, olderRun] });

    renderWorkspace();
    const table = await screen.findByRole("table", { name: "Generated reports" });
    expect(screen.getByRole("button", { name: "Load older reports" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load older reports" }));
    await waitFor(() => expect(loadRunPage).toHaveBeenLastCalledWith({ limit: 50, cursor: "older-page" }));

    const vendorRows = within(table).getAllByRole("row", { name: /Vendor portfolio/ });
    expect(vendorRows).toHaveLength(2);
    expect(screen.queryByRole("button", { name: "Load older reports" })).toBeNull();
  });

  it("shows report details from the central library without entering template governance", async () => {
    renderWorkspace();
    const table = await screen.findByRole("table", { name: "Generated reports" });
    const vendorRow = within(table).getByRole("row", { name: /Vendor portfolio/ });
    fireEvent.click(within(vendorRow).getByRole("button", { name: /View for Vendor portfolio/ }));

    const dialog = await screen.findByRole("dialog", { name: "Report details" });
    expect(within(dialog).getByRole("heading", { name: "Vendor portfolio" })).toBeTruthy();
    expect(within(dialog).getByText("vendor-relationship-report.v1")).toBeTruthy();
    expect(within(dialog).getByText(/18 · bounded source/)).toBeTruthy();
  });
});
