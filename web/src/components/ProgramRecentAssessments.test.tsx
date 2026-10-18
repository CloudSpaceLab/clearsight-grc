import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ProgramRecentAssessments } from "./ProgramRecentAssessments";

const api = vi.hoisted(() => ({ load: vi.fn() }));
vi.mock("../formsDistributionApi", async (original) => ({
  ...(await original<typeof import("../formsDistributionApi")>()),
  loadCompletedResponses: api.load,
}));

const response = {
  id: "response-1", subject_type: "PROGRAM", subject_id: "program-1",
  distribution_id: "distribution-1", form_template_id: "form-1", form_template_version: 4,
  title: "Access review", revision: 2, current: true, state: "FINAL",
  completed_at: "2026-10-15T09:30:00Z",
  score: { state: "FINAL", mode: "RISK", direction: "HIGH_IS_POOR",
    raw_score: 42, band: "HIGH", coverage: 0.9, final: true },
};

describe("approved Program recent assessments", () => {
  beforeEach(() => {
    vi.resetAllMocks();
    vi.spyOn(Date, "now").mockReturnValue(Date.parse("2026-10-18T08:00:00Z"));
    api.load.mockResolvedValue({items: [response]});
  });
  afterEach(() => vi.restoreAllMocks());

  it("shows only bounded Program responses and navigates to the existing Assessments tab", async () => {
    const navigate = vi.fn();
    render(<ProgramRecentAssessments programID="program-1" onNavigate={navigate}/>);
    expect(await screen.findByText("Access review")).toBeTruthy();
    expect(screen.getByRole("heading", {name: "Recent assessments"})).toBeTruthy();
    expect(screen.getByText("Submitted responses · Last 90 days")).toBeTruthy();
    expect(screen.getByText("Form revision 4 · Response revision 2")).toBeTruthy();
    expect(api.load).toHaveBeenCalledWith({
      subject_type: "PROGRAM", subject_id: "program-1", current_only: true,
      sort: "COMPLETED_DESC", completed_from: new Date(Date.parse("2026-10-18T08:00:00Z") - 90 * 86_400_000).toISOString(),
      limit: 3,
    });
    fireEvent.click(screen.getByRole("button", {name: "View assessments"}));
    expect(navigate).toHaveBeenCalledWith("assessments");
  });

  it("applies 7 and 30 day periods to submitted activity without changing Program posture", async () => {
    render(<ProgramRecentAssessments programID="program-1" onNavigate={vi.fn()}/>);
    await screen.findByText("Access review");
    fireEvent.change(screen.getByRole("combobox", {name: "Assessment activity period"}), {target: {value: "7"}});
    await waitFor(() => expect(api.load).toHaveBeenLastCalledWith(expect.objectContaining({
      subject_id: "program-1",
      completed_from: new Date(Date.parse("2026-10-18T08:00:00Z") - 7 * 86_400_000).toISOString(),
      limit: 3,
    })));
    fireEvent.change(screen.getByRole("combobox", {name: "Assessment activity period"}), {target: {value: "30"}});
    await waitFor(() => expect(api.load).toHaveBeenLastCalledWith(expect.objectContaining({
      subject_id: "program-1",
      completed_from: new Date(Date.parse("2026-10-18T08:00:00Z") - 30 * 86_400_000).toISOString(),
    })));
  });

  it("never presents an unrelated Program's submission as a recent assessment", async () => {
    api.load.mockResolvedValueOnce({items: [response, {...response, id: "response-other", subject_id: "program-2"}]});
    render(<ProgramRecentAssessments programID="program-1" onNavigate={vi.fn()}/>);
    expect(await screen.findByText("Recent submitted responses are unavailable.")).toBeTruthy();
    expect(screen.queryByText("Access review")).toBeNull();
    fireEvent.click(screen.getByRole("button", {name: "Retry assessments"}));
    expect(await screen.findByText("Access review")).toBeTruthy();
  });

  it("discards a previous Program's late read after navigation", async () => {
    let resolveOld!: (value: unknown) => void;
    api.load.mockImplementationOnce(() => new Promise((resolve) => { resolveOld = resolve; }))
      .mockResolvedValueOnce({items: [{...response, subject_id: "program-2", title: "New Program response"}]});
    const view = render(<ProgramRecentAssessments programID="program-1" onNavigate={vi.fn()}/>);
    view.rerender(<ProgramRecentAssessments programID="program-2" onNavigate={vi.fn()}/>);
    expect(await screen.findByText("New Program response")).toBeTruthy();
    resolveOld({items: [response]});
    await waitFor(() => expect(screen.queryByText("Access review")).toBeNull());
  });
  it("distinguishes an empty activity period from an unavailable source", async () => {
    api.load.mockResolvedValueOnce({items: []}).mockRejectedValueOnce(new Error("Not authorized"))
      .mockResolvedValueOnce({items: [response]});
    render(<ProgramRecentAssessments programID="program-1" onNavigate={vi.fn()}/>);
    expect(await screen.findByText("No submitted responses in this period.")).toBeTruthy();
    expect(screen.queryByText("Recent submitted responses are unavailable.")).toBeNull();
    fireEvent.change(screen.getByRole("combobox", {name: "Assessment activity period"}), {target: {value: "30"}});
    expect(await screen.findByText("Recent submitted responses are unavailable.")).toBeTruthy();
    expect(screen.queryByText("No submitted responses in this period.")).toBeNull();
    fireEvent.click(screen.getByRole("button", {name: "Retry assessments"}));
    expect(await screen.findByText("Access review")).toBeTruthy();
    expect(api.load).toHaveBeenCalledTimes(3);
  });

});
