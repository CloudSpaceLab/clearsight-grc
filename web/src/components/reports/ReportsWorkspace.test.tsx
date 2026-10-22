import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReportDefinition, ReportRun } from "../../reportingTypes";
import { ReportsWorkspace } from "./ReportsWorkspace";

vi.mock("../insights/RiskLossInsights", () => ({
  RiskLossInsights: ({ period, organizationScopeID }: {
    period: { start_date: string; end_date: string }; organizationScopeID?: string;
  }) => <section aria-label="Risk and loss analysis">{period.start_date} to {period.end_date} · {organizationScopeID ?? "Legal entity"}</section>,
}));

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
  const loadOwners = vi.fn();
  const createRun = vi.fn();
  const downloadRun = vi.fn();
  const onOpenInsights = vi.fn();
  const loadMetricBasis = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    loadDefinitions.mockResolvedValue([vendorDefinition, programDefinition]);
    loadRunPage.mockResolvedValue({ items: [readyRun, failedRun] });
    loadOwners.mockResolvedValue([
      { principal_id: "owner-1", display_name: "Amina Okafor" },
      { principal_id: "owner-2", display_name: "Tunde Bello" },
    ]);
    createRun.mockResolvedValue({
      ...readyRun,
      id: "run-queued",
      status: "QUEUED",
      row_count: 0,
      completed_at: undefined,
      created_at: "2026-09-25T09:00:00Z",
    });
    downloadRun.mockResolvedValue({ blob: new Blob(["report"]), filename: "vendor-portfolio.xlsx" });
    loadMetricBasis.mockResolvedValue({
      scope_kind: "LEGAL_ENTITY", scope_id: "entity-1",
      reporting_period: { start_date: "2026-07-01", end_date: "2026-09-28" },
    });
  });

  function renderWorkspace(initialView: "overview" | "library" = "library") {
    const rendered = render(<ReportsWorkspace
      organizationName="Meridian Trust Bank"
      legalEntityName="Meridian Trust Bank Nigeria"
      legalEntityID="entity-1"
      loadMetricBasis={loadMetricBasis}
      loadDefinitions={loadDefinitions}
      loadRunPage={loadRunPage}
      loadOwners={loadOwners}
      createRun={createRun}
      downloadRun={downloadRun}
      onOpenInsights={onOpenInsights}
    />);
    if (initialView === "library") fireEvent.click(screen.getByRole("tab", { name: "Reports" }));
    return rendered;
  }

  it("opens the V2 Overview, then the governed report library", async () => {
    renderWorkspace("overview");
    expect(screen.getByRole("region", { name: "Reports Overview" })).toBeTruthy();
    expect(screen.getByRole("region", { name: "Reporting measures" })).toBeTruthy();
    expect(screen.getByRole("complementary", { name: "Reporting context" })).toBeTruthy();
    expect(screen.getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["Overview", "Explore", "Reports", "Schedules", "Build report"]);
    fireEvent.click(screen.getByRole("tab", { name: "Reports" }));

    expect(await screen.findByRole("heading", { name: "Reports & insights" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Reports & insights" }).closest("header")?.className).toContain("cs-workspace-header");

    const records = await screen.findByRole("list", { name: "Generated report records" });
    expect(within(records).getByRole("button", { name: /Select Vendor portfolio, Ready/ }).getAttribute("aria-pressed")).toBe("true");
    expect(within(records).getByRole("button", { name: /Select Program health, Failed/ })).toBeTruthy();
    const inspector = screen.getByRole("complementary", { name: "Selected report" });
    expect(within(inspector).getByRole("heading", { name: "Vendor portfolio" })).toBeTruthy();
    expect(within(inspector).getByRole("button", { name: "Download report" })).toHaveProperty("disabled", false);
    expect(screen.getByText("2 matching · 2 loaded")).toBeTruthy();
    expect(screen.getByText("Reports for Meridian Trust Bank Nigeria")).toBeTruthy();
    expect(loadRunPage).toHaveBeenCalledWith({ limit: 50 }, expect.any(AbortSignal));
    expect(loadDefinitions).toHaveBeenCalledWith(true, expect.any(AbortSignal));
  });

  it("reuses loaded runs in the approved Overview and routes to existing Insights", async () => {
    renderWorkspace();
    await screen.findByRole("list", { name: "Generated report records" });
    fireEvent.click(screen.getByRole("tab", { name: "Overview" }));
    const measures = screen.getByRole("region", { name: "Reporting measures" });
    expect(within(measures).getAllByText("Reports ready")).toHaveLength(1);
    expect(within(measures).getAllByText("Requirements assessed")).toHaveLength(1);
    const recent = screen.getByRole("table", { name: "Recent reports" });
    expect(within(recent).getByRole("row", { name: /Vendor portfolio/ })).toBeTruthy();
    expect(within(recent).getByRole("row", { name: /Program health/ })).toBeTruthy();
    expect(within(measures).getByLabelText(/Reports ready: 1/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Explore analysis" }));
    const analysis = await screen.findByRole("region", { name: "Risk and loss analysis" });
    expect(analysis.textContent).toContain("2026-07-01 to 2026-09-28");
    expect(loadMetricBasis).toHaveBeenCalledWith(undefined, undefined);
    fireEvent.click(screen.getByRole("button", { name: "Open Insights workspace" }));
    expect(onOpenInsights).toHaveBeenCalledExactlyOnceWith({ start_date: "2026-07-01", end_date: "2026-09-28" });
    fireEvent.click(screen.getByRole("tab", { name: "Overview" }));
    fireEvent.click(screen.getByRole("button", { name: "View all" }));
    expect(screen.getByRole("list", { name: "Generated report records" })).toBeTruthy();
  });

  it("does not represent an unavailable report population as zero", async () => {
    loadRunPage.mockRejectedValueOnce(new Error("Report API unavailable"));
    renderWorkspace();
    fireEvent.click(screen.getByRole("tab", { name: "Overview" }));
    expect(await screen.findByText("Report totals unavailable.")).toBeTruthy();
    expect(screen.getByRole("button", { name: /Reports ready: —.*Unavailable/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /Reports ready: 0/ })).toBeNull();
  });

  it("rejects metric bundles from a different verified legal entity", async () => {
    loadMetricBasis.mockResolvedValue({
      scope_kind: "LEGAL_ENTITY", scope_id: "entity-other",
      reporting_period: { start_date: "2026-07-01", end_date: "2026-09-28" },
    });
    renderWorkspace();
    fireEvent.click(screen.getByRole("tab", { name: "Explore" }));
    expect(await screen.findByText("Reporting metrics are unavailable for the selected scope.")).toBeTruthy();
    expect(screen.queryByRole("region", { name: "Risk and loss analysis" })).toBeNull();
  });

  it("uses only the selected group scope and rejects a legal-entity substitute", async () => {
    loadMetricBasis.mockResolvedValue({
      scope_kind: "LEGAL_ENTITY", scope_id: "entity-1",
      reporting_period: { start_date: "2026-07-01", end_date: "2026-09-28" },
    });
    render(<ReportsWorkspace legalEntityID="entity-1" organizationScopeID="group-1"
      loadMetricBasis={loadMetricBasis} loadDefinitions={loadDefinitions} loadRunPage={loadRunPage}/>);
    fireEvent.click(screen.getByRole("tab", { name: "Explore" }));
    expect(await screen.findByText("Reporting metrics are unavailable for the selected scope.")).toBeTruthy();
    expect(loadMetricBasis).toHaveBeenCalledWith(undefined, "group-1");
  });

  it("keeps the source Schedules destination honest and the Build report workflow operational", async () => {
    renderWorkspace("overview");
    fireEvent.click(screen.getByRole("tab", { name: "Schedules" }));
    expect(screen.getByRole("region", { name: "Report schedules" })).toBeTruthy();
    expect(screen.getByText(/Recurring report delivery is unavailable/)).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: "Build report" }));
    expect(screen.getByRole("region", { name: "Build report" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Build report" })).toBeTruthy();
  });

  it("generates a vendor report from a ready saved setup", async () => {
    renderWorkspace();
    await screen.findByRole("list", { name: "Generated report records" });

    fireEvent.click(screen.getByRole("button", { name: "New report" }));
    const dialog = await screen.findByRole("dialog", { name: "Generate report" });
    expect(within(dialog).getAllByText("Vendor portfolio")).not.toHaveLength(0);
    expect(within(dialog).getAllByText("Overview")).not.toHaveLength(0);
    expect(within(dialog).getByLabelText("Start date")).toBeTruthy();
    expect(within(dialog).getByLabelText("End date")).toBeTruthy();

    fireEvent.change(within(dialog).getByLabelText("Start date"), { target: { value: "2026-09-01" } });
    fireEvent.change(within(dialog).getByLabelText("End date"), { target: { value: "2026-09-30" } });
    await waitFor(() => expect(loadOwners).toHaveBeenCalledWith("VENDORS", expect.any(AbortSignal)));
    fireEvent.click(within(dialog).getByRole("button", { name: /Assigned owner/ }));
    fireEvent.click(screen.getByRole("option", { name: "Amina Okafor" }));

    fireEvent.click(within(dialog).getByRole("button", { name: "Generate report" }));
    await waitFor(() => expect(createRun).toHaveBeenCalledWith(
      vendorDefinition.id,
      vendorDefinition.current_version,
      { start_date: "2026-09-01", end_date: "2026-09-30", owner_principal_id: "owner-1" },
    ));
    expect(await screen.findByText(/Vendor portfolio queued/)).toBeTruthy();
    expect(screen.getByRole("button", { name: /Select Vendor portfolio, Queued/ })).toBeTruthy();
  });

  it("explains why a ready report cannot be downloaded", async () => {
    downloadRun.mockRejectedValueOnce(new Error("Report download access is not assigned to your account."));
    renderWorkspace();

    await screen.findByRole("list", { name: "Generated report records" });
    fireEvent.click(within(screen.getByRole("complementary", { name: "Selected report" })).getByRole("button", { name: "Download report" }));

    expect(await screen.findByText("Report download access is not assigned to your account.")).toBeTruthy();
    expect(downloadRun).toHaveBeenCalledExactlyOnceWith(readyRun.id);
  });

  it("keeps generated report history usable when saved setups fail to load", async () => {
    loadDefinitions.mockRejectedValueOnce(new Error("setup endpoint unavailable"));

    renderWorkspace();

    const records = await screen.findByRole("list", { name: "Generated report records" });
    expect(within(records).getByRole("button", { name: /Select Vendor Portfolio, Ready/i })).toBeTruthy();
    expect(screen.getByText(/Saved setups unavailable/i)).toBeTruthy();
    expect(screen.queryByText(/Generated reports could not be loaded/i)).toBeNull();
    expect(screen.getByRole("button", { name: "New report" })).toHaveProperty("disabled", true);
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
    const records = await screen.findByRole("list", { name: "Generated report records" });
    expect(screen.getByText("2 matching · 2 loaded · More available")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Load older reports" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Load older reports" }));
    await waitFor(() => expect(loadRunPage).toHaveBeenLastCalledWith({ limit: 50, cursor: "older-page" }));

    const vendorRecords = within(records).getAllByRole("button", { name: /Select Vendor portfolio/ });
    expect(vendorRecords).toHaveLength(2);
    expect(within(records).getAllByRole("listitem")).toHaveLength(3);
    expect(screen.getByText("3 matching · 3 loaded")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Load older reports" })).toBeNull();
  });

  it("shows recorded digests without claiming downloaded bytes were verified", async () => {
    const run: ReportRun = {
      ...readyRun,
      data_sha256: "ab".repeat(32),
      manifest_sha256: "cd".repeat(32),
      source_boundary: { ...readyRun.source_boundary, population_complete: true },
    };
    loadRunPage.mockResolvedValueOnce({ items: [run] });
    renderWorkspace();
    await screen.findByRole("list", { name: "Generated report records" });
    fireEvent.click(within(screen.getByRole("complementary", { name: "Selected report" })).getByRole("button", { name: "View report details" }));
    const dialog = await screen.findByRole("dialog", { name: "Report details" });
    const source = within(dialog).getByRole("region", { name: "Source snapshot" });
    expect(within(source).getByText("Population complete")).toBeTruthy();
    fireEvent.click(within(source).getByText("Run identifiers and digests"));
    expect(within(source).getByText("ab".repeat(32))).toBeTruthy();
    expect(within(source).getByText("cd".repeat(32))).toBeTruthy();
    expect(within(source).getByText("run-ready")).toBeTruthy();
  });

  it("shows concise report details from the central library", async () => {
    renderWorkspace();
    await screen.findByRole("list", { name: "Generated report records" });
    fireEvent.click(within(screen.getByRole("complementary", { name: "Selected report" })).getByRole("button", { name: "View report details" }));

    const dialog = await screen.findByRole("dialog", { name: "Report details" });
    expect(within(dialog).getByRole("heading", { name: "Vendor portfolio" })).toBeTruthy();
    const source = within(dialog).getByRole("region", { name: "Source snapshot" });
    expect(within(dialog).getAllByText("18")).toHaveLength(2);
    expect(within(source).getByText("Population incomplete")).toBeTruthy();
    expect(within(source).getByText("2")).toBeTruthy();
    expect(within(dialog).getByText("XLSX")).toBeTruthy();
    expect(within(dialog).queryByText("vendor-relationship-report.v1")).toBeNull();
  });

  it("counts only downloadable files as ready and disables expired downloads", async () => {
    loadRunPage.mockResolvedValueOnce({ items: [
      readyRun,
      { ...readyRun, id: "run-expired", expires_at: "2000-01-01T00:00:00Z" },
      { ...readyRun, id: "run-unknown-expiry", expires_at: "not-a-date" },
    ] });
    renderWorkspace("overview");
    expect(await screen.findByRole("button", { name: /Reports ready: 1/ })).toBeTruthy();
    fireEvent.click(screen.getByRole("tab", { name: "Reports" }));
    const records = await screen.findByRole("list", { name: "Generated report records" });
    fireEvent.click(within(records).getByRole("button", { name: /Select Vendor portfolio, Expired/ }));
    const inspector = screen.getByRole("complementary", { name: "Selected report" });
    const expiredDownload = within(inspector).getByRole("button", { name: "File expired" });
    expect(expiredDownload).toHaveProperty("disabled", true);
    fireEvent.click(expiredDownload);
    expect(downloadRun).not.toHaveBeenCalled();
    expect(within(inspector).getByText("File expired. Generate a new report to download.")).toBeTruthy();
  });

  it("refreshes the metric basis and report history from the Overview", async () => {
    renderWorkspace("overview");
    const context = screen.getByRole("complementary", { name: "Reporting context" });
    await within(context).findByText("2026-07-01 – 2026-09-28");
    await screen.findByRole("table", { name: "Recent reports" });
    loadMetricBasis.mockResolvedValueOnce({
      scope_kind: "LEGAL_ENTITY", scope_id: "entity-1",
      reporting_period: { start_date: "2026-07-02", end_date: "2026-09-29" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(await within(context).findByText("2026-07-02 – 2026-09-29")).toBeTruthy();
    expect(loadMetricBasis).toHaveBeenCalledTimes(2);
    expect(loadRunPage).toHaveBeenCalledTimes(2);
    expect(loadDefinitions).toHaveBeenCalledTimes(2);
    expect(within(context).queryByText("2026-07-01 – 2026-09-28")).toBeNull();
  });
});
