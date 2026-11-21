import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Responses } from "../../v210/Responses";
import { ApiError } from "../../http";

const responses = vi.hoisted(() => ({ loadCompletedResponses: vi.fn(), loadCompletedResponse: vi.fn() }));
const review = vi.hoisted(() => ({ loadResponseReviews: vi.fn(), recordResponseReview: vi.fn() }));
vi.mock("../../formsDistributionApi", () => responses);
vi.mock("../../responseReviewApi", () => review);
vi.mock("../../submittedDocumentApi", () => ({ loadDocuments: vi.fn(), documentContentURL: vi.fn() }));

const summary = { id: "response-2", distribution_id: "distribution-1", title: "Security review", form_template_id: "form-1", form_template_version: 2, revision: 2, current: true, completed_at: "2026-11-20T10:00:00Z", submitted_by_display_name: "Ada Bello" };
const detail = { response: summary, revision: { id: "response-2", revision: 2, current: true, supersedes_revision_id: "response-1" }, answers: [{ field_id: "report", label: "Test coverage", value: { text: "Payment service omitted" } }], documents: [] };
const olderReview = { id: "review-2", response_id: "response-2", version: 2, decision: "REVIEWED", reviewer_id: "reviewer-1", reviewer_display_name: "Risk Officer", reviewed_at: "2026-11-20T10:00:00Z" };
beforeEach(() => {
  vi.resetAllMocks();
  window.history.replaceState(null, "", "#/forms/responses/response-2");
  responses.loadCompletedResponse.mockResolvedValue(detail);
  review.loadResponseReviews.mockResolvedValue({ version: 4, may_review: true, items: [], next_before_version: 3 });
});

it("preserves the review reason on conflict and does not retry the command", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(409, "Changed"));
  render(<Responses id="response-2"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "The report excludes payment processing." } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("The response or review changed. Reload before saving.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  expect((await screen.findByLabelText("Reason for changes") as HTMLTextAreaElement).value).toBe("The report excludes payment processing.");
  expect(review.recordResponseReview).toHaveBeenCalledTimes(1);
});

it("retries earlier review history without reloading the selected response or losing the reason", async () => {
  review.loadResponseReviews.mockResolvedValueOnce({ version: 4, may_review: true, items: [], next_before_version: 3 })
    .mockRejectedValueOnce(new Error("History unavailable"))
    .mockResolvedValueOnce({ version: 4, may_review: true, items: [olderReview] });
  render(<Responses id="response-2"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "Clarify the reporting period." } });
  fireEvent.click(screen.getByRole("button", { name: "Load earlier reviews" }));
  expect(await screen.findByText("Earlier reviews unavailable.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load earlier reviews" }));
  expect(await screen.findByText(/Risk Officer/)).toBeTruthy();
  expect(review.loadResponseReviews).toHaveBeenLastCalledWith("response-2", 3);
  expect(responses.loadCompletedResponse).toHaveBeenCalledTimes(1);
  expect((screen.getByLabelText("Reason for changes") as HTMLTextAreaElement).value).toBe("Clarify the reporting period.");
});

it("carries the original distribution when opening the exact previous revision", async () => {
  render(<Responses id="response-2"/>);
  expect((await screen.findByRole("link", { name: "Previous response" })).getAttribute("href")).toBe("#/forms/responses/response-1?distribution_id=distribution-1");
});

it.each([undefined, "", "*", " ", "bad\u0000scope"])("does not offer an earlier response with invalid distribution %s", async (distribution_id) => {
  responses.loadCompletedResponse.mockResolvedValue({ ...detail, response: { ...summary, distribution_id } });
  render(<Responses id="response-2"/>);
  expect(await screen.findByText("Earlier response unavailable.")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Previous response" })).toBeNull();
});

it("retains the Documents return destination in the previous-response link", async () => {
  window.history.replaceState(null, "", `#/forms/responses/response-2?return_to=${encodeURIComponent("#/documents?view=NEEDS_REFRESH")}`);
  render(<Responses id="response-2"/>);
  const href = (await screen.findByRole("link", { name: "Previous response" })).getAttribute("href") ?? "";
  const params = new URLSearchParams(href.split("?")[1]);
  expect(params.get("distribution_id")).toBe("distribution-1");
  expect(params.get("return_to")).toBe("#/documents?view=NEEDS_REFRESH");
});

it("rejects a previous revision returned from a different distribution, including on retry", async () => {
  window.history.replaceState(null, "", "#/forms/responses/response-1?distribution_id=distribution-1");
  responses.loadCompletedResponse.mockResolvedValue({ ...detail, response: { ...summary, id: "response-1", distribution_id: "other", current: false }, revision: { id: "response-1", current: false } });
  render(<Responses id="response-1"/>);
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Try again" }));
  await waitFor(() => expect(responses.loadCompletedResponse).toHaveBeenCalledTimes(2));
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  expect(screen.queryByText("Payment service omitted")).toBeNull();
  expect(review.recordResponseReview).not.toHaveBeenCalled();
});

it("rejects a detail response for a different revision", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ ...detail, revision: { ...detail.revision, id: "other" } });
  render(<Responses id="response-2"/>);
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("rechecks the distribution receipt when the route changes for the same revision", async () => {
  const view = render(<Responses id="response-2"/>);
  expect(await screen.findByText("Payment service omitted")).toBeTruthy();
  window.history.replaceState(null, "", "#/forms/responses/response-2?distribution_id=other");
  view.rerender(<Responses id="response-2"/>);
  expect(await screen.findByRole("heading", { name: "Response unavailable" })).toBeTruthy();
  expect(screen.queryByText("Payment service omitted")).toBeNull();
});

it("does not offer a decision when the independent reviewer route is unavailable", async () => {
  review.loadResponseReviews.mockResolvedValue({ version: 4, may_review: false, items: [] });
  render(<Responses id="response-2"/>);
  expect(await screen.findByText("Review unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  expect(review.recordResponseReview).not.toHaveBeenCalled();
});
