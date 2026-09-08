import axe from "axe-core";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ResponsesView } from "./ResponsesView";
vi.mock("../documents/DocumentBrowser", () => ({ DocumentBrowser: ({ responseRevisionID }: { responseRevisionID?: string }) => <section aria-label={`Documents for ${responseRevisionID}`}/> }));
vi.mock("./ResponseAssessment", () => ({ ResponseAssessment: ({ responseID }: { responseID: string }) => <section aria-label={`Bank assessment for ${responseID}`}/> }));

it("opens documents belonging to the selected submitted version", async () => {
  render(<ResponsesView/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Vendor certification refresh response" }));
  fireEvent.click(await screen.findByRole("tab", { name: "Review" }));
  expect(await screen.findByRole("region", { name: "Bank assessment for response-a" })).toBeTruthy();
  fireEvent.click(await screen.findByRole("tab", { name: "Documents" }));
  expect(screen.getByRole("region", { name: "Documents for response-a" })).toBeTruthy();
});

const distributionApi = vi.hoisted(() => ({
  loadCompletedResponses: vi.fn(),
  loadCompletedResponse: vi.fn(),
  loadResponseRevisions: vi.fn(),
}));
const subjects = vi.hoisted(() => ({ loadVendorRelationship: vi.fn(), loadProgram: vi.fn(), loadMatter: vi.fn() }));
vi.mock("../../vendorApi", () => ({ loadVendorRelationship: subjects.loadVendorRelationship }));
vi.mock("../../api", () => ({ loadProgram: subjects.loadProgram, loadMatter: subjects.loadMatter }));
vi.mock("../../formsDistributionApi", () => distributionApi);

const completedResponse = {
  id: "response-a", distribution_id: "distribution-a", form_template_id: "form-a", form_template_version: 4,
  title: "Vendor certification refresh", subject_type: "VENDOR", subject_id: "vendor-a", revision: 2,
  current: true, state: "FINAL", completed_at: "2026-09-01T09:30:00Z",
  score: { mode: "COMPLIANCE", direction: "LOW_IS_POOR", raw_score: 42, adverse_score: 58, band: "HIGH", coverage: 0.9, final: true, state: "FINAL", profile_version: "iso-v2", profile_checksum: "checksum", evaluator_version: "advanced-v1", calculated_at: "2026-09-01T09:30:00Z", contribution_results: [], rule_results: [] },
} as const;

beforeEach(() => {
  Object.values(subjects).forEach((mock) => mock.mockReset().mockRejectedValue(new Error("Subject unavailable")));
  window.history.replaceState(null, "", "/#forms");
  for (const value of Object.values(distributionApi)) value.mockReset();
  distributionApi.loadCompletedResponses.mockResolvedValue({ items: [completedResponse] });
  distributionApi.loadResponseRevisions.mockResolvedValue({ items: [
    { id: "response-a-1", revision: 1, achieved_assurance: "LINK_POSSESSION", scored_weight_coverage: 80, state: "FINAL", current: false, created_at: "2026-08-20T09:30:00Z" },
    { id: "response-a", revision: 2, achieved_assurance: "EMAIL_VERIFIED", scored_weight_coverage: 90, state: "FINAL", current: true, created_at: "2026-09-01T09:30:00Z" },
  ] });
  distributionApi.loadCompletedResponse.mockResolvedValue({
    response: completedResponse,
    revision: { id: "response-a", revision: 2, achieved_assurance: "EMAIL_VERIFIED", scored_weight_coverage: 90, state: "FINAL", current: true, created_at: "2026-09-01T09:30:00Z", score: completedResponse.score },
  });
});

describe("completed response portfolio", () => {
  it("opens an exact response outside the first page and names only its separately permitted subject", async () => {
    window.history.replaceState(null, "", "/#forms?section=responses&response=response-a");
    distributionApi.loadCompletedResponses.mockResolvedValue({ items: [] });
    subjects.loadVendorRelationship.mockResolvedValue({ relationship: { id: "vendor-a", service_name: "Payment hosting" } });
    distributionApi.loadCompletedResponse.mockResolvedValue({ response: { ...completedResponse, subject_type: "VENDOR_RELATIONSHIP" }, revision: { achieved_assurance: "EMAIL_VERIFIED" } });
    render(<ResponsesView/>);
    expect(await screen.findByText("Vendor service · Payment hosting")).toBeTruthy();
    expect(subjects.loadVendorRelationship).toHaveBeenCalledExactlyOnceWith("vendor-a");
    expect(distributionApi.loadCompletedResponse).toHaveBeenCalledExactlyOnceWith("response-a");
    expect(subjects.loadProgram).not.toHaveBeenCalled();
    expect(subjects.loadMatter).not.toHaveBeenCalled();
  });
  it("filters by familiar subject choices and keeps unconfigured coverage unavailable", async () => {
    distributionApi.loadCompletedResponses.mockResolvedValue({ items: [{ ...completedResponse, score: { state: "NOT_CONFIGURED", coverage: 0 } }] });
    render(<ResponsesView/>);
    await screen.findByText("Vendor certification refresh");
    expect(screen.queryByText("vendor-a")).toBeNull();
    expect(screen.queryByRole("row", { name: /vendor-a/ })).toBeNull();
    expect(screen.queryByText("0%")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /Subject type/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Vendor services" }));
    await waitFor(() => expect(distributionApi.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({ subject_type: "VENDOR_RELATIONSHIP" })));
  });
  it("requests completed responses by concern and explains compliance meaning", async () => {
    render(<ResponsesView/>);

    expect(await screen.findByText("Vendor certification refresh")).toBeTruthy();
    expect(distributionApi.loadCompletedResponses).toHaveBeenCalledWith(expect.objectContaining({ sort: "CONCERN_DESC", current_only: true, limit: 25 }));
    expect(screen.getByText("42% compliance")).toBeTruthy();
    expect(screen.getByText("Below required level")).toBeTruthy();
    expect(screen.getByText("High concern")).toBeTruthy();
  });

  it("sends concern and date filters to the server", async () => {
    render(<ResponsesView/>);
    expect(await screen.findByText("Vendor certification refresh")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Concern/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Critical" }));
    fireEvent.change(screen.getByLabelText("Completed from"), { target: { value: "2026-08-01" } });

    await waitFor(() => expect(distributionApi.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({ bands: ["CRITICAL"], completed_from: "2026-08-01T00:00:00.000Z" })));
  });

  it("keeps a failed score reviewable and offers one row action", async () => {
    distributionApi.loadCompletedResponses.mockResolvedValue({ items: [{ ...completedResponse, score: { ...completedResponse.score, raw_score: undefined, adverse_score: undefined, band: undefined, state: "FAILED", failure_code: "SCORE_EVALUATION_FAILED" } }] });
    render(<ResponsesView/>);

    expect(await screen.findByRole("button", { name: "Review Vendor certification refresh response" })).toBeTruthy();
    expect(screen.getAllByText("Score unavailable").length).toBeGreaterThan(0);
    expect(screen.getByRole("cell", { name: /The response is complete and can still be reviewed/ })).toBeTruthy();
    expect(screen.getAllByRole("button", { name: "Review Vendor certification refresh response" })).toHaveLength(1);
  });

  it("opens a focused review sheet without mutation controls and passes semantic checks", async () => {
    const view = render(<ResponsesView/>);
    fireEvent.click(await screen.findByRole("button", { name: "Review Vendor certification refresh response" }));

    expect(await screen.findByRole("dialog", { name: "Review Vendor certification refresh response" })).toBeTruthy();
    expect(screen.getByText("Email verified")).toBeTruthy();
    expect(distributionApi.loadResponseRevisions).toHaveBeenCalledWith("distribution-a");
    fireEvent.click(screen.getByRole("tab", { name: "History" }));
    expect(screen.getByRole("region", { name: "Version history" })).toBeTruthy();
    expect(screen.getByText("Revision 1")).toBeTruthy();
    expect(screen.getByText("Revision 2 · Current")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Edit response" })).toBeNull();
    const results = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });
});
