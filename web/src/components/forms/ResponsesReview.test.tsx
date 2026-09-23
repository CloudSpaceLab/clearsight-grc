import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "../../http";
import { ResponsesView } from "./ResponsesView";

const api = vi.hoisted(() => ({ loadCompletedResponses: vi.fn(), loadCompletedResponse: vi.fn(), loadResponseRevisions: vi.fn() }));
const assessment = vi.hoisted(() => ({ loadResponseAssessment: vi.fn(), recordResponseAssessment: vi.fn() }));
const templateApi = vi.hoisted(() => ({ loadFormTemplateRevision: vi.fn() }));
vi.mock("../../formsDistributionApi", () => api);
vi.mock("../../formAssessmentApi", () => assessment);
vi.mock("../../formsApi", () => templateApi);
vi.mock("../documents/DocumentBrowser", () => ({ DocumentBrowser: () => <p>Submitted documents</p> }));
const response = { id: "response", distribution_id: "distribution", title: "Security review", subject_type: "VENDOR", subject_id: "vendor", revision: 1, form_template_version: 2, current: true, completed_at: "2026-09-08T10:00:00Z", score: { state: "NOT_CONFIGURED", coverage: 0 } };
beforeEach(() => {
  vi.clearAllMocks(); window.history.replaceState(null, "", "/#forms");
  api.loadCompletedResponses.mockResolvedValue({ items: [response] });
  api.loadCompletedResponse.mockResolvedValue({ response, revision: { achieved_assurance: "EMAIL_VERIFIED" } });
  api.loadResponseRevisions.mockResolvedValue({ items: [] });
  templateApi.loadFormTemplateRevision.mockReset().mockResolvedValue({ sections: [], fields: [] });
  assessment.loadResponseAssessment.mockResolvedValue({ response_id: "response", form_template_id: "security-form", form_template_version: 2, version: 3, current: true, may_review: true, state: "AWAITING_REVIEW", required_count: 1, reviewed_required_count: 0, fields: [{ may_review: true, field: { id: "report", label: "Test coverage", type: "short_text", section_id: "reporting", assessment: { mode: "MANUAL", required: true, weight: 100, rubric: [{ id: "gap", label: "Scope missing", points: 80 }] } }, answer: { text: "Payment service omitted" } }, { may_review: true, field: { id: "followup", label: "Follow-up owner", type: "short_text", section_id: "reporting" }, answer: {} }] });
});

it("retains judgement, rationale and conflict across response sections and viewport changes", async () => {
  assessment.recordResponseAssessment.mockRejectedValue(new ApiError(409, "Assessment changed"));
  templateApi.loadFormTemplateRevision.mockResolvedValueOnce({ sections: [{ id: "reporting", title: "Reporting details" }], fields: [] });
  render(<ResponsesView/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Security review response" }));
  expect(await screen.findByText("Payment service omitted")).toBeTruthy();
  expect(await screen.findByRole("heading", { name: "Reporting details", level: 4 })).toBeTruthy();
  expect(screen.getByText("1 of 2 fields answered")).toBeTruthy();
  expect(screen.getByText("1 field needs attention")).toBeTruthy();
  expect(screen.queryByLabelText("Rationale for Test coverage")).toBeNull();
  fireEvent.click(screen.getByRole("tab", { name: "Review" }));
  fireEvent.click(await screen.findByRole("button", { name: /Decision for Test coverage/ }));
  fireEvent.click(screen.getByRole("option", { name: "Scope missing · 80 points" }));
  fireEvent.change(screen.getByLabelText("Rationale for Test coverage"), { target: { value: "The report excludes payment processing." } });
  const input = screen.getByLabelText("Rationale for Test coverage");
  fireEvent.click(screen.getByRole("button", { name: "Save assessment" }));
  await screen.findByText("Assessment changed");
  for (const section of ["Documents", "History", "Answers"]) { fireEvent.click(screen.getByRole("tab", { name: section })); expect(screen.queryByRole("button", { name: "Save assessment" })).toBeNull(); }
  fireEvent.resize(window);
  fireEvent.click(screen.getByRole("button", { name: "Answers Response section" }));
  fireEvent.click(screen.getByRole("option", { name: "Review" }));
  expect(screen.getByLabelText("Rationale for Test coverage")).toBe(input);
  expect((input as HTMLTextAreaElement).value).toBe("The report excludes payment processing.");
  expect(screen.getByText("Assessment changed")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Save assessment" }).hasAttribute("disabled")).toBe(true);
  expect(assessment.loadResponseAssessment).toHaveBeenCalledTimes(2);
}, 20000);

it("retries failed response history without reloading or discarding the selected response", async () => {
  api.loadResponseRevisions.mockRejectedValueOnce(new Error("History unavailable")).mockResolvedValueOnce({ items: [{ id: "earlier", revision: 1, current: true, created_at: "2026-09-08T10:00:00Z", achieved_assurance: "EMAIL_VERIFIED" }] });
  render(<ResponsesView/>);
  fireEvent.click(await screen.findByRole("button", { name: "Review Security review response" }));
  fireEvent.click(await screen.findByRole("tab", { name: "History" }));
  fireEvent.click(await screen.findByRole("button", { name: "Retry version history" }));
  await screen.findByText("Revision 1 · Current");
  await waitFor(() => expect(api.loadResponseRevisions).toHaveBeenCalledTimes(2));
  expect(api.loadCompletedResponse).toHaveBeenCalledTimes(1);
});
