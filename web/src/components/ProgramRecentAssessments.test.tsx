import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadCompletedResponses } from "../formsDistributionApi";
import { ProgramRecentAssessments } from "./ProgramRecentAssessments";

vi.mock("../formsDistributionApi", async (original) => ({
  ...(await original<typeof import("../formsDistributionApi")>()),
  loadCompletedResponses: vi.fn(),
}));

const response = {
  id: "response-1", distribution_id: "distribution-1", title: "Access review",
  subject_type: "PROGRAM", subject_id: "program-1", form_template_id: "form-1",
  form_template_version: 2, revision: 1, state: "FINAL" as const,
  current: true, completed_at: "2026-10-12T09:00:00Z",
  score: { state: "NOT_CONFIGURED" as const },
};

describe("approved Program recent assessment activity", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadCompletedResponses).mockResolvedValue({items: [response]});
  });

  it("uses exactly scoped history and opens the existing assessment review action", async () => {
    const onOpenAssessment = vi.fn();
    const onViewAll = vi.fn();
    render(<ProgramRecentAssessments programID="program-1"
      onOpenAssessment={onOpenAssessment} onViewAll={onViewAll}/>);
    const assessment = await screen.findByRole("button", {name: "Review Access review"});
    expect(screen.getByRole("table", {name: "Recent Program assessments"})).toBeTruthy();
    expect(screen.getByRole("columnheader", {name: "Assessment"})).toBeTruthy();
    expect(screen.getByRole("columnheader", {name: "Updated"})).toBeTruthy();
    expect(screen.getByRole("columnheader", {name: "Result"})).toBeTruthy();
    const params = vi.mocked(loadCompletedResponses).mock.calls[0]?.[0];
    expect(params).toMatchObject({subject_type: "PROGRAM", subject_id: "program-1",
      current_only: true, sort: "COMPLETED_DESC", limit: 5});
    expect(params?.completed_from).toEqual(expect.any(String));
    fireEvent.click(assessment);
    expect(onOpenAssessment).toHaveBeenCalledExactlyOnceWith("response-1");
    fireEvent.click(screen.getByRole("button", {name: "View assessments"}));
    expect(onViewAll).toHaveBeenCalledOnce();
  });

  it("changes only the historical period and never infers Program compliance", async () => {
    render(<ProgramRecentAssessments programID="program-1"
      onOpenAssessment={vi.fn()} onViewAll={vi.fn()}/>);
    await screen.findByText("Access review");
    const first = vi.mocked(loadCompletedResponses).mock.calls[0]?.[0]?.completed_from;
    fireEvent.click(screen.getByRole("button", {name: "7 days"}));
    await waitFor(() => expect(loadCompletedResponses).toHaveBeenCalledTimes(2));
    const second = vi.mocked(loadCompletedResponses).mock.calls[1]?.[0]?.completed_from;
    expect(Date.parse(second ?? "")).toBeGreaterThan(Date.parse(first ?? ""));
    expect(screen.getByRole("button", {name: "7 days"}).getAttribute("aria-pressed")).toBe("true");
    expect(screen.queryByRole("progressbar", {name: /compliance/i})).toBeNull();
  });

  it("does not display cross-Program source rows", async () => {
    vi.mocked(loadCompletedResponses).mockResolvedValue({
      items: [response, {...response, id: "foreign", subject_id: "program-2"}],
    });
    const onOpen = vi.fn();
    render(<ProgramRecentAssessments programID="program-1"
      onOpenAssessment={onOpen} onViewAll={vi.fn()}/>);
    expect(await screen.findByText("Recent assessments could not be loaded.")).toBeTruthy();
    expect(screen.queryByRole("button", {name: "Review Access review"})).toBeNull();
    expect(onOpen).not.toHaveBeenCalled();
  });

  it("recovers from an unavailable page without changing the selected period", async () => {
    vi.mocked(loadCompletedResponses).mockRejectedValueOnce(new Error("No access"))
      .mockResolvedValueOnce({items: [response]});
    render(<ProgramRecentAssessments programID="program-1"
      onOpenAssessment={vi.fn()} onViewAll={vi.fn()}/>);
    expect(await screen.findByText("Recent assessments could not be loaded.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", {name: "Retry assessments"}));
    expect(await screen.findByRole("button", {name: "Review Access review"})).toBeTruthy();
    expect(loadCompletedResponses).toHaveBeenCalledTimes(2);
    expect(screen.getByRole("button", {name: "90 days"}).getAttribute("aria-pressed")).toBe("true");
  });

  it("discards an older Program result after navigating to another Program", async () => {
    let resolveOld!: (value: {items: typeof response[]}) => void;
    vi.mocked(loadCompletedResponses).mockImplementationOnce(() =>
      new Promise((resolve) => { resolveOld = resolve; }))
      .mockResolvedValueOnce({items: [{...response, id: "new-response", title: "New Program check", subject_id: "program-2"}]});
    const view = render(<ProgramRecentAssessments programID="program-1"
      onOpenAssessment={vi.fn()} onViewAll={vi.fn()}/>);
    await waitFor(() => expect(loadCompletedResponses).toHaveBeenCalledTimes(1));
    view.rerender(<ProgramRecentAssessments programID="program-2"
      onOpenAssessment={vi.fn()} onViewAll={vi.fn()}/>);
    expect(await screen.findByText("New Program check")).toBeTruthy();
    resolveOld({items: [response]});
    await waitFor(() => expect(screen.queryByText("Access review")).toBeNull());
  });
});
