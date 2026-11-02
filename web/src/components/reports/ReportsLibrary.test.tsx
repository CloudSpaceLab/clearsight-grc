import { useState } from "react";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReportRun } from "../../reportingTypes";
import { ReportsLibrary, type ReportsLibraryProps } from "./ReportsLibrary";
import type { ReportArea } from "./reportRunPresentation";

const readyRun: ReportRun = {
  id: "report-ready", tenant_id: "tenant-1", legal_entity_id: "entity-1",
  definition_id: "vendor-template", definition_version: 3, definition_code: "VENDOR-PORTFOLIO",
  definition_checksum: "a".repeat(64), scope_kind: "LEGAL_ENTITY", requested_by_ref: "operator-1",
  as_of: "2026-10-20T08:00:00Z", filter: { kind: "group", operator: "and", children: [] },
  dataset: "VENDORS", format: "XLSX", status: "READY", attempt_count: 1, row_count: 18,
  created_at: "2026-10-20T08:01:00Z", completed_at: "2026-10-20T08:02:00Z",
  expires_at: "2099-10-21T08:00:00Z",
  source_boundary: { captured_at: "2026-10-20T08:00:00Z", projection_version: "vendors.v1",
    source_high_water: {}, population: 18, population_complete: false },
};
const failedRun: ReportRun = {
  ...readyRun, id: "report-failed", definition_id: "program-template", definition_code: "PROGRAM-HEALTH",
  dataset: "PROGRAMS", status: "FAILED", row_count: 0, failure_code: "row_limit_exceeded",
  created_at: "2026-10-19T08:01:00Z", completed_at: "2026-10-19T08:02:00Z",
};

function fixtures(overrides: Partial<ReportsLibraryProps> = {}): ReportsLibraryProps {
  return {
    runs: [readyRun, failedRun], definitionsByID: new Map(), state: "live", query: "", area: "ALL",
    scopeLabel: "Meridian Nigeria", onQueryChange: vi.fn(), onAreaChange: vi.fn(), onLoadMore: vi.fn(),
    onRetry: vi.fn(), onInspect: vi.fn(), onDownload: vi.fn(), ...overrides,
  };
}

function LibraryWithFilters({ props }: { props: ReportsLibraryProps }) {
  const [query, setQuery] = useState("");
  const [area, setArea] = useState<ReportArea>("ALL");
  return <ReportsLibrary {...props} query={query} onQueryChange={setQuery} area={area} onAreaChange={setArea}/>;
}

describe("ReportsLibrary", () => {
  it("selects exact stored runs for the persistent inspector, download and full details", () => {
    const props = fixtures();
    render(<ReportsLibrary {...props}/>);
    const inspector = screen.getByRole("complementary", { name: "Selected report" });
    expect(within(inspector).getByRole("heading", { name: "Vendor Portfolio" })).toBeTruthy();
    expect(within(inspector).getByText("Incomplete")).toBeTruthy();
    expect(within(inspector).getByText("Meridian Nigeria")).toBeTruthy();
    fireEvent.click(within(inspector).getByRole("button", { name: "Download report" }));
    expect(props.onDownload).toHaveBeenCalledWith(readyRun);
    fireEvent.click(within(inspector).getByRole("button", { name: "View report details" }));
    expect(props.onInspect).toHaveBeenLastCalledWith(readyRun);

    fireEvent.click(screen.getByRole("button", { name: /^Select Program Health,/ }));
    expect(within(inspector).getByRole("heading", { name: "Program Health" })).toBeTruthy();
    expect(within(inspector).getByText("Generation stopped: Row Limit Exceeded")).toBeTruthy();
    expect((within(inspector).getByRole("button", { name: "Download report" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(within(inspector).getByRole("button", { name: "View report details" }));
    expect(props.onInspect).toHaveBeenLastCalledWith(failedRun);
  });

  it("uses refreshed record values and clears a selection removed from the loaded population", () => {
    const props = fixtures();
    const view = render(<ReportsLibrary {...props}/>);
    fireEvent.click(screen.getByRole("button", { name: /^Select Program Health,/ }));
    const generated = { ...failedRun, status: "READY" as const, row_count: 12, failure_code: undefined };
    view.rerender(<ReportsLibrary {...props} runs={[readyRun, generated]}/>);
    const inspector = screen.getByRole("complementary", { name: "Selected report" });
    fireEvent.click(within(inspector).getByRole("button", { name: "Download report" }));
    expect(props.onDownload).toHaveBeenCalledWith(generated);
    expect(within(inspector).queryByText(/Generation stopped/)).toBeNull();

    view.rerender(<ReportsLibrary {...props} runs={[readyRun]}/>);
    expect(within(inspector).getByRole("heading", { name: "Vendor Portfolio" })).toBeTruthy();
    expect(within(inspector).queryByRole("heading", { name: "Program Health" })).toBeNull();
  });

  it("excludes expired and unknown-availability files from Ready and never enables their download", () => {
    const expired = { ...readyRun, id: "expired", definition_code: "EXPIRED-REPORT", expires_at: "2000-01-01T00:00:00Z" };
    const unknown = { ...readyRun, id: "unknown", definition_code: "UNKNOWN-REPORT", expires_at: "invalid" };
    const props = fixtures({ runs: [expired, readyRun, unknown] });
    render(<ReportsLibrary {...props}/>);
    fireEvent.click(screen.getByRole("button", { name: /^Select Expired Report,/ }));
    const inspector = screen.getByRole("complementary", { name: "Selected report" });
    expect((within(inspector).getByRole("button", { name: "File expired" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: /^Select Unknown Report,/ }));
    expect((within(inspector).getByRole("button", { name: "Download report" }) as HTMLButtonElement).disabled).toBe(true);
    expect(within(inspector).getByText("File availability is unknown. Refresh reports.")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Report status/ }));
    fireEvent.click(screen.getByRole("option", { name: "Ready" }));
    expect(screen.getByRole("button", { name: /^Select Vendor Portfolio,/ })).toBeTruthy();
    expect(screen.queryByRole("button", { name: /^Select Expired Report,/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /^Select Unknown Report,/ })).toBeNull();
    expect(props.onDownload).not.toHaveBeenCalled();
  });

  it("filters the loaded population and preserves the real older-report recovery action", () => {
    const props = fixtures({ hasMore: true });
    render(<LibraryWithFilters props={props}/>);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search generated reports" }), { target: { value: "program" } });
    expect(screen.getByText("1 matching · 2 loaded · More available")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /^Select Vendor Portfolio,/ })).toBeNull();
    expect(screen.getByRole("button", { name: /^Select Program Health,/ })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /Area/ }));
    fireEvent.click(screen.getByRole("option", { name: "Vendors" }));
    expect(screen.getByRole("heading", { name: "No matching loaded reports" })).toBeTruthy();
    expect(screen.getByText("Clear filters or load older reports.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Load older reports" }));
    expect(props.onLoadMore).toHaveBeenCalledOnce();
  });

  it("does not retain report content or downloadable actions after an unavailable read", () => {
    const props = fixtures();
    const view = render(<ReportsLibrary {...props}/>);
    view.rerender(<ReportsLibrary {...props} state="error" error="Report access is unavailable."/>);
    expect(screen.queryByRole("list", { name: "Generated report records" })).toBeNull();
    expect(screen.queryByRole("heading", { name: "Vendor Portfolio" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Download report" })).toBeNull();
    expect(screen.getByRole("alert")).toBeTruthy();
    expect(screen.getByText("Report access is unavailable.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(props.onRetry).toHaveBeenCalledOnce();
  });

  it("keeps loading distinct from an empty population and prevents actions on updating records", () => {
    const props = fixtures({ state: "loading", runs: [] });
    const view = render(<ReportsLibrary {...props}/>);
    expect(screen.getByText("Loading reports…")).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "No reports yet" })).toBeNull();
    expect(screen.queryByText(/0 loaded/)).toBeNull();
    view.rerender(<ReportsLibrary {...props} runs={[readyRun]}/>);
    const inspector = screen.getByRole("complementary", { name: "Selected report" });
    expect((within(inspector).getByRole("button", { name: "Download report" }) as HTMLButtonElement).disabled).toBe(true);
    expect((within(inspector).getByRole("button", { name: "View report details" }) as HTMLButtonElement).disabled).toBe(true);
  });
});
