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

describe("submitted field assessment", () => {
  it("presents semantic vendor responses as service groups with truthful metrics", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, required_count: 2, fields: [
      { may_review: true, field: { id: "requirement_1_response", section_id: "service_1", label: "Certification missing", type: "long_text", description: "Vendor response · Service: Moneytor GetPaid application", assessment: { mode: "MANUAL", required: true, weight: 20, reviewer_role: "REVIEWER", rubric: [] } }, answer: { text: "Audit underway" } },
      { field: { id: "requirement_1_evidence", section_id: "service_1", label: "Current certificate", type: "vendor_document", description: "Supporting evidence · Service: Moneytor GetPaid application" }, answer: {} },
      { may_review: true, field: { id: "requirement_2_response", section_id: "service_2", label: "PCI-DSS certificate expired", type: "long_text", description: "Vendor response · Service: Payment Terminal Service Provider (PTSP)", assessment: { mode: "MANUAL", required: true, weight: 20, reviewer_role: "REVIEWER", rubric: [] } }, answer: {} },
      { field: { id: "requirement_2_evidence", section_id: "service_2", label: "Current PCI-DSS certificate", type: "vendor_document", description: "Supporting evidence · Service: Payment Terminal Service Provider (PTSP)" }, answer: { artifact_ids: ["doc-1"] } },
    ] });
    render(<ResponseAssessment responseID="response-2" answersOnly/>);
    expect(await screen.findByRole("region", { name: "Vendor responses" })).toBeTruthy();
    expect(screen.getByRole("group", { name: "Requirements" }).textContent).toContain("2");
    expect(screen.getByRole("group", { name: "Vendor responses received" }).textContent).toContain("1");
    expect(screen.getByRole("group", { name: "Evidence received" }).textContent).toContain("1");
    expect(screen.getAllByText("No document received")).toHaveLength(1);
    expect(screen.getByRole("region", { name: "Moneytor GetPaid application" })).toBeTruthy();
    expect(screen.queryByText("Existing form rules")).toBeNull();
  });

  it("keeps supporting documents inside each requirement while bank reviewers assess the vendor response", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, fields: [
      { may_review: true, field: { id: "requirement_1_response", section_id: "service_1", label: "Certification missing", type: "long_text", description: "Vendor response · Service: Moneytor GetPaid application", assessment: { mode: "MANUAL", required: true, weight: 20, reviewer_role: "REVIEWER", rubric: [] } }, answer: { text: "Audit underway" } },
      { field: { id: "requirement_1_evidence", section_id: "service_1", label: "Current certificate", type: "vendor_document", description: "Supporting evidence · Service: Moneytor GetPaid application" }, answer: {} },
    ] });
    render(<ResponseAssessment responseID="response-2"/>);
    expect(await screen.findByRole("heading", { name: "Internal assessment" })).toBeTruthy();
    expect(screen.getByRole("article", { name: "Certification missing" })).toBeTruthy();
    expect(screen.queryByRole("article", { name: "Current certificate" })).toBeNull();
    expect(screen.getByText("Vendor response")).toBeTruthy();
    expect(screen.queryByText("Submitted answer and evidence")).toBeNull();
  });

  it("keeps the submitted result and document access available when the assessment read fails", async () => {
    api.loadResponseAssessment.mockRejectedValue(new ApiError(503, "Assessment unavailable"));
    render(<ResponseAssessment responseID="response-2" submissionScore={assessment.automatic_score as never}/>);
    await screen.findByText("Assessment unavailable");
    expect(screen.getByText("20% risk")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "View submitted documents" }));
    expect(screen.getByRole("region", { name: "Submitted documents response-2" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save assessment" })).toBeNull();
  });

  it("withholds review while response currency cannot be checked without clearing its draft", async () => {
    const view = render(<ResponseAssessment responseID="response-2"/>);
    await screen.findByLabelText("Rationale for Vulnerability test");
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "Keep this rationale" } });
    view.rerender(<ResponseAssessment responseID="response-2" current={null}/>);
    expect(screen.queryByRole("button", { name: "Save assessment" })).toBeNull();
    expect(screen.queryByText(/Historical response\. Review/)).toBeNull();
    expect(screen.getByText(/Response currency is unavailable/)).toBeTruthy();
    view.rerender(<ResponseAssessment responseID="response-2" current/>);
    expect((screen.getByLabelText("Rationale for Vulnerability test") as HTMLTextAreaElement).value).toBe("Keep this rationale");
  });
  it("immediately removes review controls when the parent learns this revision is historical", async () => {
    const view = render(<ResponseAssessment responseID="response-2"/>);
    await screen.findByLabelText("Rationale for Vulnerability test");
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "Retained judgement" } });
    view.rerender(<ResponseAssessment responseID="response-2" current={false}/>);
    expect(screen.queryByRole("button", { name: "Save assessment" })).toBeNull();
    expect(screen.queryByLabelText("Rationale for Vulnerability test")).toBeNull();
    expect(api.loadResponseAssessment).toHaveBeenCalledTimes(1);
  });
  it("presents review not required without permission warnings or empty review scores", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, state: "NOT_REQUIRED", may_review: false, required_count: 0, reviewed_required_count: 0, fields: [], automatic_score: { state: "NOT_CONFIGURED", coverage: 0 }, assessed_score: { state: "NOT_CONFIGURED", coverage: 0 } });
    render(<ResponseAssessment responseID="response-2"/>);
    await screen.findByText("Review not required");
    expect(screen.queryByText(/Review permission is unavailable/)).toBeNull();
    expect(screen.queryByText("0 of 0 required fields reviewed")).toBeNull();
    expect(screen.queryByRole("region", { name: "Automatic result" })).toBeNull();
    expect(screen.queryByRole("region", { name: "Reviewed result" })).toBeNull();
  });
  it("requires explicit permission for the response and each field before offering judgement controls", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, fields: assessment.fields.map((item) => ({ ...item, may_review: false })) });
    const view = render(<ResponseAssessment responseID="response-2"/>);
    await screen.findByText("1 required field awaiting review");
    expect(screen.queryByRole("button", { name: /Decision for/ })).toBeNull();
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, may_review: undefined });
    view.rerender(<ResponseAssessment responseID="response-3"/>);
    await screen.findByText("1 required field awaiting review");
    expect(screen.queryByRole("button", { name: "Save assessment" })).toBeNull();
  });
  it("shows unavailable totals without converting null scores to numbers", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, automatic_score: { ...assessment.automatic_score, raw_score: null, adverse_score: null, coverage: null } });
    render(<ResponseAssessment responseID="response-2"/>);
    const result = await screen.findByRole("region", { name: "Automatic result" });
    expect(within(result).getByText("Score unavailable")).toBeTruthy();
    expect(within(result).getByText("Coverage: Not available")).toBeTruthy();
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

  it("keeps a committed assessment successful when its parent refresh fails", async () => {
    api.recordResponseAssessment.mockResolvedValue({ ...assessment, version: 1 });
    render(<ResponseAssessment responseID="response-2" onUpdated={() => { throw new Error("Parent refresh failed"); }}/>);
    await screen.findByText("1 required field awaiting review");
    fireEvent.click(screen.getByRole("button", { name: /Decision for Vulnerability test/ }));
    fireEvent.click(screen.getByRole("option", { name: /Evidence incomplete/ }));
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "Scope missing." } });
    fireEvent.click(screen.getByRole("button", { name: "Save assessment" }));
    expect(await screen.findByText(/Assessment saved/)).toBeTruthy();
    expect(screen.queryByText("Parent refresh failed")).toBeNull();
  });
  it("shows provisional bank coverage separately from the submitted score and reviews exact documents", async () => {
    render(<ResponseAssessment responseID="response-2"/>);
    expect(await screen.findByText("1 required field awaiting review")).toBeTruthy();
    expect(screen.getByText("20% risk")).toBeTruthy();
    expect(screen.getByText("Provisional")).toBeTruthy();
    expect(screen.getByText("0 of 1 required fields reviewed")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "View submitted documents" }));
    expect(screen.getByRole("region", { name: "Submitted documents response-2" })).toBeTruthy();
  });

  it("requires a rationale, retains failed input, and submits only bank decisions with version", async () => {
    api.recordResponseAssessment.mockRejectedValueOnce(new ApiError(503, "Review service unavailable"));
    api.recordResponseAssessment.mockResolvedValueOnce({ ...assessment, version: 1, state: "ASSESSED", reviewed_count: 1, reviewed_required_count: 1 });
    const updated = vi.fn();
    render(<ResponseAssessment responseID="response-2" onUpdated={updated}/>);
    await screen.findByText("1 required field awaiting review");
    fireEvent.click(screen.getByRole("button", { name: /Decision for Vulnerability test/ }));
    fireEvent.click(screen.getByRole("option", { name: /Evidence incomplete/ }));
    expect(screen.getByRole("button", { name: "Save assessment" }).hasAttribute("disabled")).toBe(true);
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "The report excludes the payment service." } });
    fireEvent.click(screen.getByRole("button", { name: "Save assessment" }));
    expect(await screen.findByText(/Review service unavailable/)).toBeTruthy();
    expect((screen.getByLabelText("Rationale for Vulnerability test") as HTMLTextAreaElement).value).toBe("The report excludes the payment service.");
    fireEvent.click(screen.getByRole("button", { name: "Save assessment" }));
    await waitFor(() => expect(updated).toHaveBeenCalledTimes(1));
    expect(api.recordResponseAssessment).toHaveBeenLastCalledWith("response-2", { expected_version: 0, decisions: [{ field_id: "test", outcome_id: "gap", rationale: "The report excludes the payment service." }] });
  });

  it("requires a fresh read after a conflict without discarding the draft judgement", async () => {
    api.recordResponseAssessment.mockRejectedValue(new ApiError(409, "Assessment changed"));
    render(<ResponseAssessment responseID="response-2"/>);
    await screen.findByText("1 required field awaiting review");
    fireEvent.click(screen.getByRole("button", { name: /Decision for Vulnerability test/ }));
    fireEvent.click(screen.getByRole("option", { name: /Evidence incomplete/ }));
    fireEvent.change(screen.getByLabelText("Rationale for Vulnerability test"), { target: { value: "Scope missing." } });
    fireEvent.click(screen.getByRole("button", { name: "Save assessment" }));
    expect(await screen.findByRole("button", { name: "Reload assessment" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Save assessment" }).hasAttribute("disabled")).toBe(true);
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, version: 2 });
    fireEvent.click(screen.getByRole("button", { name: "Reload assessment" }));
    await waitFor(() => expect(api.loadResponseAssessment).toHaveBeenCalledTimes(2));
    expect((await screen.findByLabelText("Rationale for Vulnerability test") as HTMLTextAreaElement).value).toBe("Scope missing.");
  });

  it("keeps historical response decisions read-only and passes semantic accessibility checks", async () => {
    api.loadResponseAssessment.mockResolvedValue({ ...assessment, current: false });
    const view = render(<ResponseAssessment responseID="response-1"/>);
    expect(await screen.findByText(/Historical response\. Review/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Save assessment" })).toBeNull();
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
