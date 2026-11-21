import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "../../http";
import { Responses } from "../../v210/Responses";

const api = vi.hoisted(() => ({ loadResponseReviews: vi.fn(), recordResponseReview: vi.fn(), loadCompletedResponse: vi.fn(), loadCompletedResponses: vi.fn(), loadDocuments: vi.fn(), documentContentURL: vi.fn() }));
vi.mock("../../responseReviewApi", () => ({ loadResponseReviews: api.loadResponseReviews, recordResponseReview: api.recordResponseReview }));
vi.mock("../../formsDistributionApi", () => ({ loadCompletedResponse: api.loadCompletedResponse, loadCompletedResponses: api.loadCompletedResponses }));
vi.mock("../../submittedDocumentApi", () => ({ loadDocuments: api.loadDocuments, documentContentURL: api.documentContentURL }));

const summary = { id: "response", title: "Access review", form_template_id: "form", form_template_version: 1, revision: 1, current: true, completed_at: "2026-11-20T10:00:00Z" };
beforeEach(() => {
  sessionStorage.clear();
  vi.clearAllMocks();
  api.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "response", current: true, revision: 1 }, answers: [], documents: [] });
  api.loadResponseReviews.mockResolvedValue({ version: 0, may_review: true, items: [] });
});

it("reviews an unscored response and displays the recorded reviewer", async () => {
  api.recordResponseReview.mockResolvedValue({ id: "review", response_id: "response", response_revision: 1, version: 1, decision: "REVIEWED" });
  api.loadResponseReviews.mockResolvedValueOnce({ version: 0, may_review: true, items: [] }).mockResolvedValueOnce({ version: 1, may_review: true, items: [{ id: "review", version: 1, decision: "REVIEWED", reviewer_id: "person", reviewer_display_name: "Ada Okafor", reviewed_at: "2026-11-20T12:00:00Z", comment: "" }] });
  render(<Responses id="response" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  await waitFor(() => expect(api.recordResponseReview).toHaveBeenCalledWith("response", { expected_version: 0, decision: "REVIEWED", comment: "" }, expect.any(String)));
  expect(await screen.findByText(/Ada Okafor/)).toBeTruthy();
});

it("requires a reason for changes and retains it after a conflict", async () => {
  api.recordResponseReview.mockRejectedValue(new ApiError(409, "Review changed"));
  render(<Responses id="response" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  const reason = screen.getByLabelText("Reason for changes") as HTMLTextAreaElement;
  expect(reason.required).toBe(true);
  fireEvent.submit(reason.closest("form")!);
  expect(api.recordResponseReview).not.toHaveBeenCalled();
  fireEvent.change(reason, { target: { value: "Add the reporting period." } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("The response or review changed. Reload before saving.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  expect((await screen.findByLabelText("Reason for changes") as HTMLTextAreaElement).value).toBe("Add the reporting period.");
});

it("does not offer a decision without a current response and reviewer permission", async () => {
  api.loadResponseReviews.mockResolvedValue({ version: 0, may_review: false, items: [] });
  const view = render(<Responses id="response" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Review unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  api.loadResponseReviews.mockResolvedValue({ version: 0, may_review: true, items: [] });
  api.loadCompletedResponse.mockResolvedValue({ response: { ...summary, current: false }, revision: { id: "response", revision: 1, current: false }, answers: [], documents: [] });
  view.unmount();
  render(<Responses id="response" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Review is available for the current response.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("reports a committed decision if refreshing history fails", async () => {
  api.recordResponseReview.mockResolvedValue({ id: "review", response_id: "response", response_revision: 1, version: 1, decision: "REVIEWED" });
  api.loadResponseReviews.mockResolvedValueOnce({ version: 0, may_review: true, items: [] }).mockRejectedValueOnce(new Error("History unavailable"));
  render(<Responses id="response" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review saved. Reload to see review history.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});
