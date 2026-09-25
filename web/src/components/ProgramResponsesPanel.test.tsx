import axe from "axe-core";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ProgramResponsesPanel } from "./ProgramResponsesPanel";

const api = vi.hoisted(() => ({ loadCompletedResponses: vi.fn(), loadCompletedResponse: vi.fn() }));
vi.mock("../formsDistributionApi", async (original) => ({ ...(await original<typeof import("../formsDistributionApi")>()), loadCompletedResponses: api.loadCompletedResponses, loadCompletedResponse: api.loadCompletedResponse }));
vi.mock("./forms/ResponseAssessment", () => ({ ResponseAssessment: ({ responseID }: { responseID: string }) => <section aria-label={`Assess ${responseID}`}/> }));
vi.mock("./documents/DocumentBrowser", () => ({ DocumentBrowser: ({ responseRevisionID, scopeLabel }: { responseRevisionID?: string; scopeLabel?: string }) => <section aria-label={`Documents for ${responseRevisionID ?? "none"}`}><p>{scopeLabel}</p></section> }));

const response = {
  id: "response-1", distribution_id: "distribution-1", form_template_id: "form-1", form_template_version: 4,
  title: "Branch incident self-assessment", subject_type: "PROGRAM", subject_id: "program-1", revision: 2,
  current: true, state: "FINAL", completed_at: "2026-09-01T09:30:00Z",
  score: { mode: "COMPLIANCE", direction: "LOW_IS_POOR", raw_score: 42, adverse_score: 58, band: "HIGH", coverage: 0.9, final: true, state: "FINAL", profile_version: "iso-v2", profile_checksum: "checksum", evaluator_version: "advanced-v1", calculated_at: "2026-09-01T09:30:00Z", contribution_results: [], rule_results: [] },
};
const earlier = {
  id: "response-2", distribution_id: "distribution-1", form_template_id: "form-1", form_template_version: 4,
  title: "Branch data controls check", subject_type: "PROGRAM", subject_id: "program-1", revision: 1,
  current: false, state: "FINAL", completed_at: "2026-08-20T09:30:00Z",
  score: { mode: "COMPLIANCE", direction: "LOW_IS_POOR", raw_score: 90, adverse_score: 10, band: "LOW", coverage: 1, final: true, state: "FINAL", calculated_at: "2026-08-20T09:30:00Z", contribution_results: [], rule_results: [] },
};

beforeEach(() => {
  vi.clearAllMocks();
  window.history.replaceState(null, "", "/#programs/program-1/evidence-results");
  api.loadCompletedResponses.mockResolvedValue({ items: [response] });
  api.loadCompletedResponse.mockResolvedValue({
    response,
    revision: { id: "response-1", revision: 2, achieved_assurance: "EMAIL_VERIFIED", scored_weight_coverage: 90, state: "FINAL", current: true, created_at: "2026-09-01T09:30:00Z", score: response.score },
  });
});

describe("Program submitted data panel", () => {
  it("lists completed responses for the Program and loads the next page on demand", async () => {
    api.loadCompletedResponses
      .mockResolvedValueOnce({ items: [response], next_cursor: "next-page" })
      .mockResolvedValueOnce({ items: [earlier], next_cursor: undefined });
    const view = render(<ProgramResponsesPanel programID="program-1"/>);

    expect(await screen.findByText("Branch incident self-assessment")).toBeTruthy();
    expect(screen.getByText("42% compliance")).toBeTruthy();
    expect(screen.getByText("Below required level")).toBeTruthy();
    expect(screen.getByText("High concern")).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Assessment result" })).toBeTruthy();
    expect(screen.queryByRole("columnheader", { name: "Coverage" })).toBeNull();
    expect(api.loadCompletedResponses).toHaveBeenCalledWith({ subject_type: "PROGRAM", subject_id: "program-1", current_only: true, sort: "COMPLETED_DESC", limit: 20, cursor: undefined });

    fireEvent.click(screen.getByRole("button", { name: "Load more responses" }));
    expect(await screen.findByText("Branch data controls check")).toBeTruthy();
    expect(api.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({ subject_id: "program-1", cursor: "next-page" }));

    const results = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });

  it("names the checked population and the next action when no data is collected", async () => {
    api.loadCompletedResponses.mockResolvedValue({ items: [] });
    render(<ProgramResponsesPanel programID="program-1"/>);

    expect(await screen.findByText("No data collected yet")).toBeTruthy();
    expect(screen.getByText("Completed responses for this Program")).toBeTruthy();
    expect(screen.getByText("No completed responses are recorded for this Program. Start a form collection or add a collection check in Data collection to collect responses.")).toBeTruthy();
    expect(screen.queryByRole("row")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Open Data collection" }));
    expect(window.location.hash).toBe("#programs/program-1/monitoring");
  });

  it("keeps submitted data retryable when the response list cannot be loaded", async () => {
    api.loadCompletedResponses.mockRejectedValueOnce(new Error("response list unavailable")).mockResolvedValue({ items: [response] });
    render(<ProgramResponsesPanel programID="program-1"/>);

    expect(await screen.findByText("Submitted data could not be loaded")).toBeTruthy();
    expect(screen.getByText("Completed responses cannot be reviewed while the response list is unavailable. Retry the list before reviewing evidence.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Retry submitted data" }));
    expect(await screen.findByText("Branch incident self-assessment")).toBeTruthy();
    expect(api.loadCompletedResponses).toHaveBeenCalledTimes(2);
  });

  it("opens a review sheet with Answers, Documents and Review tabs and passes semantic checks", async () => {
    const view = render(<ProgramResponsesPanel programID="program-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Review Branch incident self-assessment response" }));

    const dialog = await screen.findByRole("dialog", { name: "Review Branch incident self-assessment response" });
    expect(api.loadCompletedResponse).toHaveBeenCalledWith("response-1");
    expect(within(dialog).getByRole("tab", { name: "Answers" })).toBeTruthy();
    expect(within(dialog).getByRole("tab", { name: "Documents" })).toBeTruthy();
    expect(within(dialog).getByRole("tab", { name: "Review" })).toBeTruthy();
    expect(within(dialog).getByRole("region", { name: "Assess response-1" })).toBeTruthy();
    expect(within(dialog).getByText("Email verified")).toBeTruthy();
    expect(within(dialog).getByText("Scoring coverage")).toBeTruthy();
    expect(within(dialog).getByText("90%")).toBeTruthy();

    fireEvent.click(within(dialog).getByRole("tab", { name: "Review" }));
    expect(await within(dialog).findByRole("region", { name: "Assess response-1" })).toBeTruthy();

    const results = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });

  it("shows documents scoped to the selected response revision and closes the sheet", async () => {
    render(<ProgramResponsesPanel programID="program-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Review Branch incident self-assessment response" }));

    const dialog = await screen.findByRole("dialog", { name: "Review Branch incident self-assessment response" });
    fireEvent.click(within(dialog).getByRole("tab", { name: "Documents" }));
    expect(await within(dialog).findByRole("region", { name: "Documents for response-1" })).toBeTruthy();
    expect(within(dialog).getByText("Branch incident self-assessment · Revision 2")).toBeTruthy();

    fireEvent.click(within(dialog).getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(screen.getByRole("button", { name: "Review Branch incident self-assessment response" })).toBeTruthy();
  });
});
