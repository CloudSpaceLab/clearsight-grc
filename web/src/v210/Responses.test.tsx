import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Responses } from "./Responses";
import { ApiError } from "../http";

const responses = vi.hoisted(() => ({ loadCompletedResponses: vi.fn(), loadCompletedResponse: vi.fn() }));
const review = vi.hoisted(() => ({ loadResponseReviews: vi.fn(), recordResponseReview: vi.fn() }));
const documents = vi.hoisted(() => ({ loadDocuments: vi.fn(), documentContentURL: vi.fn() }));
vi.mock("../formsDistributionApi", () => responses);
vi.mock("../responseReviewApi", () => review);
vi.mock("../submittedDocumentApi", () => documents);

const summary = { id: "revision-1", title: "Access review", form_template_id: "form-1", form_template_version: 3, revision: 2, current: true, completed_at: "2026-11-20T10:00:00Z", submitted_by_display_name: "Ada Bello" };
const reviewReceipt = { id: "review-5", response_id: "revision-1", response_revision: 2, version: 5, decision: "REVIEWED" };
beforeEach(() => {
  vi.clearAllMocks();
  responses.loadCompletedResponses.mockResolvedValue({ items: [summary] });
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true, revision: 2 }, answers: [{ field_id: "q1", label: "Is access reviewed?", value: { text: "Yes" } }], documents: [] });
  review.loadResponseReviews.mockResolvedValue({ version: 4, may_review: true, items: [] });
  review.recordResponseReview.mockResolvedValue(reviewReceipt);
});

it("opens Documents scoped to this response revision", async () => {
  render(<Responses id="revision-1"/>);
  const link = await screen.findByRole("link", { name: "View response documents" });
  expect(link.getAttribute("href")).toBe("#/documents?response_revision_id=revision-1");
});

it("retains the response distribution in the Documents return link", async () => {
  const previous = window.location.hash;
  window.history.replaceState(null, "", "#/forms/responses/revision-1?distribution_id=distribution-1");
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, distribution_id: "distribution-1" }, revision: { id: "revision-1", current: true, revision: 2 }, answers: [], documents: [] });
  try {
    render(<Responses id="revision-1"/>);
    const link = await screen.findByRole("link", { name: "View response documents" });
    const query = new URLSearchParams(link.getAttribute("href")!.split("?")[1]);
    expect(query.get("response_revision_id")).toBe("revision-1");
    expect(query.get("return_to")).toBe("#/forms/responses/revision-1?distribution_id=distribution-1");
  } finally { window.history.replaceState(null, "", previous || "#/forms/responses"); }
});

it("discovers historical responses and identifies the recorded submitter", async () => {
  render(<Responses/>);
  expect(screen.getByPlaceholderText("Form title")).toBeTruthy();
  expect(await screen.findByText(/Ada Bello/)).toBeTruthy();
  expect(responses.loadCompletedResponses).toHaveBeenCalledWith(expect.objectContaining({ current_only: false, limit: 25 }));
  expect(screen.getByRole("link", { name: "Access review" }).getAttribute("href")).toBe("#/forms/responses/revision-1");
});

it("returns to the filtered Documents view when opened from Documents", async () => {
  const previous = window.location.hash;
  window.history.replaceState(null, "", `#/forms/responses/revision-1?return_to=${encodeURIComponent("#/documents?view=NEEDS_REFRESH&q=policy")}`);
  try {
    render(<Responses id="revision-1"/>);
    expect((await screen.findByRole("link", { name: "← Documents" })).getAttribute("href")).toBe("#/documents?view=NEEDS_REFRESH&q=policy");
  } finally { window.history.replaceState(null, "", previous || "#/forms/responses"); }
});

it("keeps the Documents return link when the response is unavailable", async () => {
  const previous = window.location.hash;
  window.history.replaceState(null, "", `#/forms/responses/revision-1?return_to=${encodeURIComponent("#/documents?kind=PDF")}`);
  responses.loadCompletedResponse.mockRejectedValueOnce(new Error("unavailable"));
  try {
    render(<Responses id="revision-1"/>);
    expect((await screen.findByRole("link", { name: "View documents" })).getAttribute("href")).toBe("#/documents?kind=PDF");
  } finally { window.history.replaceState(null, "", previous || "#/forms/responses"); }
});

it("rejects external return destinations", async () => {
  const previous = window.location.hash;
  window.history.replaceState(null, "", `#/forms/responses/revision-1?return_to=${encodeURIComponent("https://example.com/")}`);
  try {
    render(<Responses id="revision-1"/>);
    expect((await screen.findByRole("link", { name: "← Responses" })).getAttribute("href")).toBe("#/forms/responses");
  } finally { window.history.replaceState(null, "", previous || "#/forms/responses"); }
});

it("shows the exact answers and records an authorized versioned review", async () => {
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("Is access reviewed?")).toBeTruthy();
  expect(screen.getByText("Yes")).toBeTruthy();
  fireEvent.change(screen.getByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  await vi.waitFor(() => expect(review.recordResponseReview).toHaveBeenCalledWith("revision-1", { expected_version: 4, decision: "REVIEWED", comment: "" }));
});

it("does not permit review of an earlier revision or without reviewer authority", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, current: false }, revision: { id: "revision-1", current: false }, answers: [], documents: [] });
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: true, items: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("Earlier response")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("does not offer review without current authority", async () => {
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: false, items: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("Review unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("does not present a link recipient as a verified submitter", async () => {
  responses.loadCompletedResponses.mockResolvedValue({ items: [{ ...summary, submitted_by_display_name: undefined, submission_recipient: { type: "EXTERNAL_AUDIENCE", label: "Vendor contact" } }] });
  render(<Responses/>);
  expect(await screen.findByText(/Submitter not identified/)).toBeTruthy();
  expect(screen.queryByText(/Vendor contact/)).toBeNull();
});

it("blocks review until the submitted answers can be loaded", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true } });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("Answers unavailable. Reload before reviewing.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("blocks review when the response documents are incomplete", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [], documents: [], documents_next_cursor: "next" });
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("More documents must be loaded before review.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Load more documents" }));
  expect(await screen.findByRole("button", { name: "Save review" })).toBeTruthy();
  expect(documents.loadDocuments).toHaveBeenCalledWith({ response_revision_id: "revision-1", current_only: false, cursor: "next", limit: 100 });
});

it("shows the structured document information in a response", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [{ field_id: "q2", label: "Assurance report", value: { document: { artifact_id: "a", document_type: "SOC 2", reference: "ABC-123", issued_by: "Auditor Ltd", issued_on: "2026-01-01", expires_on: "2026-12-31" } } }], documents: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText(/SOC 2/)).toBeTruthy();
  expect(screen.getByText(/ABC-123/)).toBeTruthy();
  expect(screen.getByText(/Auditor Ltd/)).toBeTruthy();
  expect(screen.getByText(/Jan 1, 2026/)).toBeTruthy();
  expect(screen.getByText(/Dec 31, 2026/)).toBeTruthy();
});

it("keeps exact numerical values and multiline text without inventing currency", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [
    { field_id: "a", label: "Explanation", type: "long_text", value: { text: "First line\nSecond line" } },
    { field_id: "b", label: "Amount", type: "currency", value: { text: "9007199254740993.0100" } },
    { field_id: "c", label: "Rate", type: "percentage", value: { text: "12.3450" } },
    { field_id: "d", label: "Unrecognized rate", type: "percentage", value: { text: "unavailable" } },
  ], documents: [] });
  render(<Responses id="revision-1"/>);
  expect((await screen.findByText("Explanation")).nextElementSibling?.textContent).toBe("First line\nSecond line");
  expect(screen.getByText("Amount").nextElementSibling?.textContent).toBe("9007199254740993.0100");
  expect(screen.getByText("Rate").nextElementSibling?.textContent).toBe("12.3450%");
  expect(screen.getByText("Unrecognized rate").nextElementSibling?.textContent).toBe("unavailable");
});

it("formats valid date answers and preserves invalid recorded dates", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [
    { field_id: "a", label: "Review date", type: "date", value: { text: "2025-11-28" } },
    { field_id: "b", label: "Invalid date", type: "date", value: { text: "2025-02-30" } },
  ], documents: [] });
  render(<Responses id="revision-1"/>);
  expect((await screen.findByText("Review date")).nextElementSibling?.textContent).toBe("Nov 28, 2025");
  expect(screen.getByText("Invalid date").nextElementSibling?.textContent).toBe("2025-02-30");
});

it("distinguishes confirmation, missing and unknown checkbox or attestation answers", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [
    { field_id: "a", label: "Confirmed check", type: "CHECKBOX", value: { text: "yes" } },
    { field_id: "b", label: "Declined attestation", type: "attestation", value: { text: "false" } },
    { field_id: "c", label: "Missing check", type: "checkbox", value: {} },
    { field_id: "d", label: "Unrecognized check", type: "checkbox", value: { text: "pending" } },
  ], documents: [] });
  render(<Responses id="revision-1"/>);
  expect((await screen.findByText("Confirmed check")).nextElementSibling?.textContent).toBe("Confirmed");
  expect(screen.getByText("Declined attestation").nextElementSibling?.textContent).toBe("Not confirmed");
  expect(screen.getByText("Missing check").nextElementSibling?.textContent).toBe("No answer");
  expect(screen.getByText("Unrecognized check").nextElementSibling?.textContent).toBe("pending");
});

it("distinguishes submitted signatures, photos, document counts and missing documents", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [
    { field_id: "a", label: "Signature", type: "signature", value: { artifact_ids: ["signature-1"] } },
    { field_id: "b", label: "Photo", type: "photo", value: { artifact_ids: ["photo-1"] } },
    { field_id: "c", label: "Files", type: "file", value: { artifact_ids: ["file-1", "file-2"] } },
    { field_id: "d", label: "Missing file", type: "file", value: {} },
  ], documents: [] });
  render(<Responses id="revision-1"/>);
  expect((await screen.findByText("Signature")).nextElementSibling?.textContent).toBe("Signature submitted");
  expect(screen.getByText("Photo").nextElementSibling?.textContent).toBe("Photo submitted");
  expect(screen.getByText("Files").nextElementSibling?.textContent).toBe("2 documents submitted");
  expect(screen.getByText("Missing file").nextElementSibling?.textContent).toBe("No answer");
});

it("limits review comments by UTF-8 bytes", async () => {
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.change(screen.getByLabelText("Comment (optional)"), { target: { value: "é".repeat(2001) } });
  expect(screen.getByText("Comment must be 4,000 bytes or fewer.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Save review" }) as HTMLButtonElement).disabled).toBe(true);
  expect(review.recordResponseReview).not.toHaveBeenCalled();
});

it("preserves entered review details after a definite conflict and reload", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(409, "Changed"));
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "Please provide the latest report." } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("The response or review changed. Reload before saving.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  expect((await screen.findByLabelText("Reason for changes") as HTMLTextAreaElement).value).toBe("Please provide the latest report.");
});

it("keeps a rejected review editable", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(422, "Invalid"));
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.change(screen.getByLabelText("Comment (optional)"), { target: { value: "Needs clarification" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review not saved. Check the decision and comment.")).toBeTruthy();
  expect((screen.getByLabelText("Comment (optional)") as HTMLTextAreaElement).value).toBe("Needs clarification");
});

it("reports a definite authorization denial instead of an unknown outcome", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.change(screen.getByLabelText("Comment (optional)"), { target: { value: "Review note" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("You cannot review this response. Reload to check access.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  expect((await screen.findByLabelText("Comment (optional)") as HTMLTextAreaElement).value).toBe("Review note");
});

it("reports a committed review when the following history refresh fails", async () => {
  review.loadResponseReviews.mockResolvedValueOnce({ version: 4, may_review: true, items: [] }).mockRejectedValueOnce(new Error("connection lost"));
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review saved. Reload to see review history.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it.each([
  { response_id: "other", response_revision: 2, version: 5, decision: "REVIEWED" },
  { response_id: "revision-1", response_revision: 2, version: 6, decision: "REVIEWED" },
  { response_id: "revision-1", response_revision: 1, version: 5, decision: "REVIEWED" },
  { response_id: "revision-1", response_revision: 2, version: 5, decision: "CHANGES_REQUESTED" },
  { id: "", response_id: "revision-1", response_revision: 2, version: 5, decision: "REVIEWED" },
  {},
])("does not confirm a review with a mismatched receipt %#", async (receipt) => {
  review.recordResponseReview.mockResolvedValueOnce({ id: "review-5", ...receipt });
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review status unknown. Reload before trying again.")).toBeTruthy();
  expect(screen.queryByText("Review saved.")).toBeNull();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  expect(review.recordResponseReview).toHaveBeenCalledTimes(1);
});

it("requires a reason for changes and does not retry an uncertain save", async () => {
  review.recordResponseReview.mockRejectedValue(new Error("connection lost"));
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "Clarify the review date." } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review status unknown. Reload before trying again.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("distinguishes the recorded access recipient from the actual submitter", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, submission_recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "person-1", label: "Tola Adebayo" } }, revision: { id: "revision-1", current: true, revision: 2 }, answers: [], documents: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("Tola Adebayo")).toBeTruthy();
  expect(screen.getByText("Ada Bello")).toBeTruthy();
  expect(screen.getByText("Access recipient")).toBeTruthy();
});

it("links to the exact previous response without inventing a history", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, distribution_id: "distribution-1" }, revision: { id: "revision-1", current: true, revision: 2, supersedes_revision_id: "prior/1" }, answers: [], documents: [] });
  render(<Responses id="revision-1"/>);
  expect((await screen.findByRole("link", { name: "Previous response" })).getAttribute("href")).toBe("#/forms/responses/prior%2F1?distribution_id=distribution-1");
});

it("identifies a response as earlier when its source revision is no longer current", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: false }, answers: [], documents: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("Earlier response")).toBeTruthy();
  expect(screen.getByText("Review is available for the current response.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("sends only one review while submission is in progress", async () => {
  let finish!: (value: unknown) => void;
  review.recordResponseReview.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<Responses id="revision-1"/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  const form = screen.getByRole("button", { name: "Save review" }).closest("form")!;
  act(() => { fireEvent.submit(form); fireEvent.submit(form); });
  expect(review.recordResponseReview).toHaveBeenCalledTimes(1);
  await act(async () => { finish(reviewReceipt); });
  expect(await screen.findByText("Review saved.")).toBeTruthy();
});


it("locks the review fields while a decision is being saved", async () => {
  let finish!: (value: unknown) => void;
  review.recordResponseReview.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<Responses id="revision-1"/>);
  const decision = await screen.findByLabelText("Review decision") as HTMLSelectElement;
  fireEvent.change(decision, { target: { value: "REVIEWED" } });
  const comment = screen.getByLabelText("Comment (optional)") as HTMLTextAreaElement;
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(decision.disabled).toBe(true);
  expect(comment.disabled).toBe(true);
  await act(async () => { finish(reviewReceipt); });
});

it("clears stale document loading state when the response reloads", async () => {
  let finish!: (value: unknown) => void;
  const payload = { response: summary, revision: { id: "revision-1", current: true }, answers: undefined, documents: [], documents_next_cursor: "next" };
  responses.loadCompletedResponse.mockResolvedValue(payload);
  documents.loadDocuments.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<Responses id="revision-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more documents" }));
  expect((screen.getByRole("button", { name: "Loading…" }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  const loadMore = await screen.findByRole("button", { name: "Load more documents" }) as HTMLButtonElement;
  expect(loadMore.disabled).toBe(false);
  await act(async () => { finish({ items: [{ id: "old", file_name: "Old document" }], next_cursor: undefined }); });
  expect(screen.queryByText("Old document")).toBeNull();
  expect(screen.getByRole("button", { name: "Load more documents" })).toBeTruthy();
});

it("waits for earlier review history before saving a new decision", async () => {
  let finish!: (value: unknown) => void;
  review.loadResponseReviews.mockResolvedValueOnce({ version: 4, may_review: true, items: [], next_before_version: 3 }).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }));
  render(<Responses id="revision-1"/>);
  const decision = await screen.findByLabelText("Review decision");
  fireEvent.change(decision, { target: { value: "REVIEWED" } });
  const save = screen.getByRole("button", { name: "Save review" }) as HTMLButtonElement;
  act(() => { fireEvent.click(screen.getByRole("button", { name: "Load earlier reviews" })); fireEvent.submit(save.closest("form")!); });
  expect(save.disabled).toBe(true);
  expect(review.recordResponseReview).not.toHaveBeenCalled();
  await act(async () => { finish({ version: 4, may_review: true, items: [], next_before_version: undefined }); });
  expect(save.disabled).toBe(false);
});

it("clears stale history loading state when the response reloads", async () => {
  let finish!: (value: unknown) => void;
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: undefined, documents: [] });
  review.loadResponseReviews.mockResolvedValue({ version: 4, may_review: true, items: [], next_before_version: 3 });
  render(<Responses id="revision-1"/>);
  await screen.findByRole("button", { name: "Load earlier reviews" });
  review.loadResponseReviews.mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }));
  fireEvent.click(screen.getByRole("button", { name: "Load earlier reviews" }));
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  const older = await screen.findByRole("button", { name: "Load earlier reviews" }) as HTMLButtonElement;
  expect(older.disabled).toBe(false);
  await act(async () => { finish({ items: [{ id: "stale", decision: "REVIEWED", reviewer_display_name: "Old reviewer", reviewed_at: "2026-01-01" }] }); });
  expect(screen.queryByText(/Old reviewer/)).toBeNull();
});

it("does not duplicate a response page or append it after search changes", async () => {
  let finish!: (value: unknown) => void;
  responses.loadCompletedResponses.mockResolvedValueOnce({ items: [summary], next_cursor: "old" }).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; })).mockResolvedValueOnce({ items: [{ ...summary, id: "new", title: "Risk review" }], next_cursor: "new" });
  render(<Responses/>);
  const more = await screen.findByRole("button", { name: "Load more" });
  act(() => { fireEvent.click(more); fireEvent.click(more); });
  expect(responses.loadCompletedResponses).toHaveBeenCalledTimes(2);
  fireEvent.change(screen.getByLabelText("Find a response"), { target: { value: "Risk" } });
  fireEvent.submit(screen.getByLabelText("Find a response").closest("form")!);
  expect(await screen.findByRole("link", { name: "Risk review" })).toBeTruthy();
  await act(async () => { finish({ items: [{ ...summary, id: "old", title: "Old result" }] }); });
  expect(screen.queryByText("Old result")).toBeNull();
  expect((screen.getByRole("button", { name: "Load more" }) as HTMLButtonElement).disabled).toBe(false);
});

it("clears response results if access is denied while loading another page", async () => {
  responses.loadCompletedResponses.mockResolvedValueOnce({ items: [summary], next_cursor: "next" }).mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<Responses/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more" }));
  expect(await screen.findByText("Responses unavailable.")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Access review" })).toBeNull();
});

it("removes review history and the review action if access is denied when loading earlier reviews", async () => {
  review.loadResponseReviews.mockResolvedValueOnce({ version: 4, may_review: true, items: [{ id: "old-review", decision: "REVIEWED", reviewer_display_name: "Risk Officer", reviewed_at: "2026-11-20T11:42:00Z" }], next_before_version: 3 }).mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<Responses id="revision-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load earlier reviews" }));
  expect(await screen.findByText("Review unavailable.")).toBeTruthy();
  expect(screen.queryByText(/Risk Officer/)).toBeNull();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("clears loaded documents if access is denied when loading the next page", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [], documents: [{ id: "doc-1", file_name: "Sensitive report.pdf" }], documents_next_cursor: "next" });
  documents.loadDocuments.mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<Responses id="revision-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more documents" }));
  expect(await screen.findByText("Documents unavailable. Reload before reviewing.")).toBeTruthy();
  expect(screen.queryByText("Sensitive report.pdf")).toBeNull();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("distinguishes response submission and review times on the same day", async () => {
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: false, items: [{ id: "r1", decision: "REVIEWED", reviewer_display_name: "Risk Officer", reviewed_at: "2026-11-20T11:42:00Z", comment: "" }] });
  render(<Responses id="revision-1"/>);
  expect((await screen.findByText(/10:00.*UTC/)).textContent).toContain("20 Nov 2026");
  expect(screen.getByText(/11:42.*UTC/).textContent).toContain("Risk Officer");
});

it("does not invent times when a timestamp is missing or invalid", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "invalid" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: false, items: [{ id: "r1", decision: "REVIEWED", reviewed_at: "", comment: "" }] });
  render(<Responses id="revision-1"/>);
  expect((await screen.findAllByText(/time unavailable/)).length).toBe(2);
  expect(screen.queryByText(/UTC/)).toBeNull();
});

it("does not report a normalized time for an impossible calendar date", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "2026-02-30T10:00:00Z" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("time unavailable")).toBeTruthy();
});

it("does not normalize an impossible clock time", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "2026-11-20T24:00:00Z" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText("time unavailable")).toBeTruthy();
});

it("converts recorded offsets to the labelled UTC time", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "2026-11-20T10:15:30+01:00" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  render(<Responses id="revision-1"/>);
  expect(await screen.findByText(/09:15:30 UTC/)).toBeTruthy();
});
