import axe from "axe-core";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../../http";
import { ResponseAssessment } from "./ResponseAssessment";

const api = vi.hoisted(() => ({ loadResponseAssessment: vi.fn(), recordResponseAssessment: vi.fn() }));
vi.mock("../../formAssessmentApi", () => api);
vi.mock("../documents/DocumentBrowser", () => ({ DocumentBrowser: ({ responseRevisionID }: { responseRevisionID: string }) => <section aria-label={`Submitted documents ${responseRevisionID}`}/> }));

const assessment = {
  response_id: "response-2", form_template_id: "form-1", form_template_version: 4, version: 0, current: true, may_review: true,
  state: "AWAITING_REVIEW", required_count: 1, reviewed_required_count: 0, reviewed_count: 0,
  fields: [{ may_review: true, field: { id: "test", label: "Vulnerability test", type: "file", required: true, assessment: { mode: "MANUAL", required: true, weight: 100, rubric: [{ id: "gap", label: "Evidence incomplete", points: 100 }, { id: "pass", label: "Evidence sufficient", points: 0 }] } }, answer: { artifact_ids: ["artifact-1"] } }],
  automatic_score: { state: "FINAL", mode: "RISK", direction: "HIGH_IS_POOR", raw_score: 20, adverse_score: 20, coverage: 1, calculated_at: "2026-09-08T10:00:00Z", contribution_results: [], rule_results: [] },
  assessed_score: { state: "PROVISIONAL", mode: "RISK", direction: "HIGH_IS_POOR", coverage: 0, final: false },
};

beforeEach(() => { vi.clearAllMocks(); api.loadResponseAssessment.mockResolvedValue(structuredClone(assessment)); });

describe("submitted field bank assessment", () => {
  it("requires explicit permission for the response and each field before offering judgement controls", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, fields: assessment.fields.map((item) => ({ ...item, may_review: false })) });
    const view = render(<ResponseAssessment responseID="response-2"/>);
    await screen.findByText("1 required field awaiting bank review");
    expect(screen.queryByRole("button", { name: /Bank judgement for/ })).toBeNull();
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, may_review: undefined });
    view.rerender(<ResponseAssessment responseID="response-3"/>);
    await screen.findByText("1 required field awaiting bank review");
    expect(screen.queryByRole("button", { name: "Save bank assessment" })).toBeNull();
  });
  it("shows unavailable totals without converting null scores to numbers", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, automatic_score: { ...assessment.automatic_score, raw_score: null, adverse_score: null, coverage: null } });
    render(<ResponseAssessment responseID="response-2"/>);
    const result = await screen.findByRole("region", { name: "Automatic submission result" });
    expect(within(result).getByText("No score available")).toBeTruthy();
    expect(within(result).getByText("Score coverage unknown")).toBeTruthy();
    expect(within(result).queryByText(/null risk|0% score/)).toBeNull();
  });
  it("shows automatic field contributions and uses the saved concern bands for poor-result filtering", async () => {
    const field = { id: "control", label: "Privileged access reviews", type: "yes_no", required: true };
    api.loadResponseAssessment.mockResolvedValue({ ...assessment,
      fields: [...assessment.fields, { field, answer: { text: "No" } }],
      automatic_score: { ...assessment.automatic_score, contribution_results: [{ id: "access-review-gap", points: 80, weight: 50, outcome: "PASS" }] },
      score_profile: { version: "risk4", mode: "RISK", direction: "HIGH_IS_POOR", contributions: [{ id: "access-review-gap", label: "Access review gap", predicate: { operator: "AND", children: [{ field_id: "control", operator: "EQUALS", values: ["No"] }, { field_id: "test", operator: "ANSWERED" }] }, weight: 50, match_points: 80, non_match_points: 0, missing: "INDETERMINATE" }], bands: [{ band: "LOW", from: 0, through: 39 }, { band: "MODERATE", from: 40, through: 69 }, { band: "HIGH", from: 70, through: 89 }, { band: "CRITICAL", from: 90, through: 100 }] },
    });
    render(<ResponseAssessment responseID="response-2"/>);
    const control = await screen.findByRole("article", { name: "Privileged access reviews" });
    expect(within(control).getByText(/Access review gap/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /Assessment fields/ }));
    fireEvent.click(screen.getByRole("option", { name: "Poor results" }));
    expect(screen.getByText(/at least 70 concern points/)).toBeTruthy();
    expect(screen.getByRole("article", { name: "Privileged access reviews" })).toBeTruthy();
  });

  it("keeps a committed bank assessment successful when its parent refresh fails", async () => {
    api.recordResponseAssessment.mockResolvedValue({ ...assessment, version: 1 });
    render(<ResponseAssessment responseID="response-2" onUpdated={() => { throw new Error("Parent refresh failed"); }}/>);
    await screen.findByText("1 required field awaiting bank review");
    fireEvent.click(screen.getByRole("button", { name: /Bank judgement for Vulnerability test/ }));
    fireEvent.click(screen.getByRole("option", { name: /Evidence incomplete/ }));
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "Scope missing." } });
    fireEvent.click(screen.getByRole("button", { name: "Save bank assessment" }));
    expect(await screen.findByText(/Bank assessment saved/)).toBeTruthy();
    expect(screen.queryByText("Parent refresh failed")).toBeNull();
  });
  it("shows provisional bank coverage separately from the submitted score and reviews exact documents", async () => {
    render(<ResponseAssessment responseID="response-2"/>);
    expect(await screen.findByText("1 required field awaiting bank review")).toBeTruthy();
    expect(screen.getByText("20 risk score")).toBeTruthy();
    expect(screen.getByText("Bank assessment is provisional")).toBeTruthy();
    expect(screen.getByText("0 of 1 required fields reviewed")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review submitted evidence" }));
    expect(screen.getByRole("region", { name: "Submitted documents response-2" })).toBeTruthy();
  });

  it("requires a rationale, retains failed input, and submits only bank decisions with version", async () => {
    api.recordResponseAssessment.mockRejectedValueOnce(new ApiError(503, "Review service unavailable"));
    api.recordResponseAssessment.mockResolvedValueOnce({ ...assessment, version: 1, state: "ASSESSED", reviewed_count: 1, reviewed_required_count: 1 });
    const updated = vi.fn();
    render(<ResponseAssessment responseID="response-2" onUpdated={updated}/>);
    await screen.findByText("1 required field awaiting bank review");
    fireEvent.click(screen.getByRole("button", { name: /Bank judgement for Vulnerability test/ }));
    fireEvent.click(screen.getByRole("option", { name: /Evidence incomplete/ }));
    expect(screen.getByRole("button", { name: "Save bank assessment" }).hasAttribute("disabled")).toBe(true);
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "The report excludes the payment service." } });
    fireEvent.click(screen.getByRole("button", { name: "Save bank assessment" }));
    expect(await screen.findByText(/Review service unavailable/)).toBeTruthy();
    expect((screen.getByLabelText("Rationale for Vulnerability test") as HTMLTextAreaElement).value).toBe("The report excludes the payment service.");
    fireEvent.click(screen.getByRole("button", { name: "Save bank assessment" }));
    await waitFor(() => expect(updated).toHaveBeenCalledTimes(1));
    expect(api.recordResponseAssessment).toHaveBeenLastCalledWith("response-2", { expected_version: 0, decisions: [{ field_id: "test", outcome_id: "gap", rationale: "The report excludes the payment service." }] });
  });

  it("requires a fresh read after a conflict without discarding the draft judgement", async () => {
    api.recordResponseAssessment.mockRejectedValue(new ApiError(409, "Assessment changed"));
    render(<ResponseAssessment responseID="response-2"/>);
    await screen.findByText("1 required field awaiting bank review");
    fireEvent.click(screen.getByRole("button", { name: /Bank judgement for Vulnerability test/ }));
    fireEvent.click(screen.getByRole("option", { name: /Evidence incomplete/ }));
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "Scope missing." } });
    fireEvent.click(screen.getByRole("button", { name: "Save bank assessment" }));
    expect(await screen.findByRole("button", { name: "Reload bank assessment" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Save bank assessment" }).hasAttribute("disabled")).toBe(true);
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, version: 2 });
    fireEvent.click(screen.getByRole("button", { name: "Reload bank assessment" }));
    await waitFor(() => expect(api.loadResponseAssessment).toHaveBeenCalledTimes(2));
    expect((await screen.findByLabelText("Rationale for Vulnerability test") as HTMLTextAreaElement).value).toBe("Scope missing.");
  });

  it("keeps historical response decisions read-only and passes semantic accessibility checks", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, current: false });
    const view = render(<ResponseAssessment responseID="response-1"/>);
    expect(await screen.findByText(/Historical response\. Review/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save bank assessment" })).toBeNull();
    const result = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
    expect(result.violations.map((violation) => violation.id)).toEqual([]);
  });

  it("clears protected answers when the selected response changes to an unavailable record", async () => {
    const first = { ...assessment, fields: [{ ...assessment.fields[0], answer: { text: "Submitted private answer" } }] };
    api.loadResponseAssessment.mockResolvedValueOnce(first).mockRejectedValueOnce(new ApiError(403, "Response access denied"));
    const view = render(<ResponseAssessment responseID="response-2"/>);
    expect(await screen.findByText("Submitted private answer")).toBeTruthy();
    view.rerender(<ResponseAssessment responseID="restricted"/>);
    expect(screen.queryByText("Submitted private answer")).toBeNull();
    expect(await screen.findByText(/Response access denied/)).toBeTruthy();
  });
});
