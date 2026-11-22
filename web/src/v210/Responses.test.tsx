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
  sessionStorage.clear();
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

it("retains response filters through scoped Documents", async () => {
  const previous = window.location.hash;
  window.history.replaceState(null, "", "#/forms/responses/revision-1?response_concern=HIGH&response_sort=CONCERN_DESC");
  try {
    render(<Responses id="revision-1"/>);
    const link = await screen.findByRole("link", { name: "View response documents" });
    const query = new URLSearchParams(link.getAttribute("href")!.split("?")[1]);
    expect(query.get("return_to")).toBe("#/forms/responses/revision-1?response_concern=HIGH&response_sort=CONCERN_DESC");
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
    render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
    expect((await screen.findByRole("link", { name: "← Documents" })).getAttribute("href")).toBe("#/documents?view=NEEDS_REFRESH&q=policy");
  } finally { window.history.replaceState(null, "", previous || "#/forms/responses"); }
});

it("keeps the Documents return link when the response is unavailable", async () => {
  const previous = window.location.hash;
  window.history.replaceState(null, "", `#/forms/responses/revision-1?return_to=${encodeURIComponent("#/documents?kind=PDF")}`);
  responses.loadCompletedResponse.mockRejectedValueOnce(new Error("unavailable"));
  try {
    render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
    expect((await screen.findByRole("link", { name: "View documents" })).getAttribute("href")).toBe("#/documents?kind=PDF");
  } finally { window.history.replaceState(null, "", previous || "#/forms/responses"); }
});

it("rejects external return destinations", async () => {
  const previous = window.location.hash;
  window.history.replaceState(null, "", `#/forms/responses/revision-1?return_to=${encodeURIComponent("https://example.com/")}`);
  try {
    render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
    expect((await screen.findByRole("link", { name: "← Responses" })).getAttribute("href")).toBe("#/forms/responses");
  } finally { window.history.replaceState(null, "", previous || "#/forms/responses"); }
});

it("shows the exact answers and records an authorized versioned review", async () => {
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Is access reviewed?")).toBeTruthy();
  expect(screen.getByText("Yes")).toBeTruthy();
  fireEvent.change(screen.getByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  await vi.waitFor(() => expect(review.recordResponseReview).toHaveBeenCalledWith("revision-1", { expected_version: 4, decision: "REVIEWED", comment: "" }, expect.any(String)));
});

it("does not permit review of an earlier revision or without reviewer authority", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, current: false }, revision: { id: "revision-1", current: false }, answers: [], documents: [] });
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: true, items: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Earlier response")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("does not offer review without current authority", async () => {
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: false, items: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Answers unavailable. Reload before reviewing.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("blocks review when the response documents are incomplete", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [], documents: [], documents_next_cursor: "next" });
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("More documents must be loaded before review.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Load more documents" }));
  expect(await screen.findByRole("button", { name: "Save review" })).toBeTruthy();
  expect(documents.loadDocuments).toHaveBeenCalledWith({ response_revision_id: "revision-1", current_only: false, cursor: "next", limit: 100 });
});

it("shows the structured document information in a response", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [{ field_id: "q2", label: "Assurance report", value: { document: { artifact_id: "a", document_type: "SOC 2", reference: "ABC-123", issued_by: "Auditor Ltd", issued_on: "2026-01-01", expires_on: "2026-12-31" } } }], documents: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect((await screen.findByText("Signature")).nextElementSibling?.textContent).toBe("Signature submitted");
  expect(screen.getByText("Photo").nextElementSibling?.textContent).toBe("Photo submitted");
  expect(screen.getByText("Files").nextElementSibling?.textContent).toBe("2 documents submitted");
  expect(screen.getByText("Missing file").nextElementSibling?.textContent).toBe("No answer");
});

it("limits review comments by UTF-8 bytes", async () => {
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.change(screen.getByLabelText("Comment (optional)"), { target: { value: "é".repeat(2001) } });
  expect(screen.getByText("Comment must be 4,000 bytes or fewer.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Save review" }) as HTMLButtonElement).disabled).toBe(true);
  expect(review.recordResponseReview).not.toHaveBeenCalled();
});

it("preserves entered review details after a definite conflict and reload", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(409, "Changed"));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "Please provide the latest report." } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("The response or review changed. Reload before saving.")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  expect((await screen.findByLabelText("Reason for changes") as HTMLTextAreaElement).value).toBe("Please provide the latest report.");
});

it("keeps a rejected review editable", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(422, "Invalid"));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.change(screen.getByLabelText("Comment (optional)"), { target: { value: "Needs clarification" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review not saved. Check the decision and comment.")).toBeTruthy();
  expect((screen.getByLabelText("Comment (optional)") as HTMLTextAreaElement).value).toBe("Needs clarification");
});

it("reports a definite authorization denial instead of an unknown outcome", async () => {
  review.recordResponseReview.mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.change(screen.getByLabelText("Comment (optional)"), { target: { value: "Review note" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("You cannot review this response. Reload to check access.")).toBeTruthy();
  expect((screen.getByLabelText("Comment (optional)") as HTMLTextAreaElement).value).toBe("Review note");
  expect((screen.getByRole("button", {name:"Retry review"}) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Reload response" }));
  expect((await screen.findByLabelText("Comment (optional)") as HTMLTextAreaElement).value).toBe("Review note");
});

it("reports a committed review when the following history refresh fails", async () => {
  review.loadResponseReviews.mockResolvedValueOnce({ version: 4, may_review: true, items: [] }).mockRejectedValueOnce(new Error("connection lost"));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "REVIEWED" } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review status unknown. Retrying will submit the same decision.")).toBeTruthy();
  expect(screen.queryByText("Review saved.")).toBeNull();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  expect(review.recordResponseReview).toHaveBeenCalledTimes(1);
});

it("requires a reason for changes and does not retry an uncertain save", async () => {
  review.recordResponseReview.mockRejectedValue(new Error("connection lost"));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.change(await screen.findByLabelText("Review decision"), { target: { value: "CHANGES_REQUESTED" } });
  fireEvent.change(screen.getByLabelText("Reason for changes"), { target: { value: "Clarify the review date." } });
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(await screen.findByText("Review status unknown. Retrying will submit the same decision.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("distinguishes the recorded access recipient from the actual submitter", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, submission_recipient: { type: "INTERNAL_PRINCIPAL", principal_id: "person-1", label: "Tola Adebayo" } }, revision: { id: "revision-1", current: true, revision: 2 }, answers: [], documents: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Tola Adebayo")).toBeTruthy();
  expect(screen.getByText("Ada Bello")).toBeTruthy();
  expect(screen.getByText("Access recipient")).toBeTruthy();
});

it("links to the exact previous response without inventing a history", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, distribution_id: "distribution-1" }, revision: { id: "revision-1", current: true, revision: 2, supersedes_revision_id: "prior/1" }, answers: [], documents: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect((await screen.findByRole("link", { name: "Previous response" })).getAttribute("href")).toBe("#/forms/responses/prior%2F1?distribution_id=distribution-1");
});

it("identifies a response as earlier when its source revision is no longer current", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: false }, answers: [], documents: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("Earlier response")).toBeTruthy();
  expect(screen.getByText("Review is available for the current response.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("sends only one review while submission is in progress", async () => {
  let finish!: (value: unknown) => void;
  review.recordResponseReview.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  const decision = await screen.findByLabelText("Review decision") as HTMLSelectElement;
  fireEvent.change(decision, { target: { value: "REVIEWED" } });
  const comment = screen.getByLabelText("Comment (optional)") as HTMLTextAreaElement;
  fireEvent.click(screen.getByRole("button", { name: "Save review" }));
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
  expect((screen.getByRole("button", { name: "Saving…" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.getByText("Saving review…")).toBeTruthy();
  expect(review.recordResponseReview).toHaveBeenCalledTimes(1);
  await act(async () => { finish(reviewReceipt); });
});

it("clears stale document loading state when the response reloads", async () => {
  let finish!: (value: unknown) => void;
  const payload = { response: summary, revision: { id: "revision-1", current: true }, answers: undefined, documents: [], documents_next_cursor: "next" };
  responses.loadCompletedResponse.mockResolvedValue(payload);
  documents.loadDocuments.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
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
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load earlier reviews" }));
  expect(await screen.findByText("Review unavailable.")).toBeTruthy();
  expect(screen.queryByText(/Risk Officer/)).toBeNull();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("clears loaded documents if access is denied when loading the next page", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "revision-1", current: true }, answers: [], documents: [{ id: "doc-1", file_name: "Sensitive report.pdf" }], documents_next_cursor: "next" });
  documents.loadDocuments.mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more documents" }));
  expect(await screen.findByText("Documents unavailable. Reload before reviewing.")).toBeTruthy();
  expect(screen.queryByText("Sensitive report.pdf")).toBeNull();
  expect(screen.queryByRole("button", { name: "Save review" })).toBeNull();
});

it("distinguishes response submission and review times on the same day", async () => {
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: false, items: [{ id: "r1", decision: "REVIEWED", reviewer_display_name: "Risk Officer", reviewed_at: "2026-11-20T11:42:00Z", comment: "" }] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect((await screen.findByText(/10:00.*UTC/)).textContent).toContain("20 Nov 2026");
  expect(screen.getByText(/11:42.*UTC/).textContent).toContain("Risk Officer");
});

it("does not invent times when a timestamp is missing or invalid", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "invalid" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: false, items: [{ id: "r1", decision: "REVIEWED", reviewed_at: "", comment: "" }] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect((await screen.findAllByText(/time unavailable/)).length).toBe(2);
  expect(screen.queryByText(/UTC/)).toBeNull();
});

it("does not report a normalized time for an impossible calendar date", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "2026-02-30T10:00:00Z" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("time unavailable")).toBeTruthy();
});

it("does not normalize an impossible clock time", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "2026-11-20T24:00:00Z" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText("time unavailable")).toBeTruthy();
});

it("converts recorded offsets to the labelled UTC time", async () => {
  responses.loadCompletedResponse.mockResolvedValue({ response: { ...summary, completed_at: "2026-11-20T10:15:30+01:00" }, revision: { id: "revision-1", current: true }, answers: [], documents: [] });
  render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
  expect(await screen.findByText(/09:15:30 UTC/)).toBeTruthy();
});

it("applies concern and sort on the server and keeps them when loading more", async () => {
 responses.loadCompletedResponses.mockResolvedValueOnce({items:[summary],next_cursor:"page-2"}).mockResolvedValueOnce({items:[summary],next_cursor:"filtered-2"}).mockResolvedValueOnce({items:[summary],next_cursor:"sorted-2"}).mockResolvedValueOnce({items:[]});
 render(<Responses/>); await screen.findByText(/Ada Bello/);
 fireEvent.change(screen.getByLabelText("Concern"),{target:{value:"HIGH"}});
 await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({bands:["HIGH"],sort:"COMPLETED_DESC"})));
 fireEvent.change(screen.getByLabelText("Sort responses"),{target:{value:"CONCERN_DESC"}});
 await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({bands:["HIGH"],sort:"CONCERN_DESC"})));
 fireEvent.click(await screen.findByRole("button",{name:"Load more"}));
 await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({bands:["HIGH"],sort:"CONCERN_DESC",cursor:"sorted-2"})));
});

it("shows the recorded form score and distinguishes provisional and unavailable results", async () => {
 responses.loadCompletedResponses.mockResolvedValue({items:[{...summary,score:{state:"PROVISIONAL",mode:"COMPLIANCE",raw_score:42,band:"HIGH"}},{...summary,id:"failed",title:"Failed score",score:{state:"FAILED",mode:"COMPLIANCE",raw_score:99}},{...summary,id:"invalid",title:"Invalid score",score:{state:"FINAL",mode:"COMPLIANCE",raw_score:999}}]});
 render(<Responses/>); expect(await screen.findByText("Form compliance score: 42 / 100 · Provisional" )).toBeTruthy();
 expect(screen.getAllByText("Score unavailable")).toHaveLength(2); expect(screen.queryByText(/999/)).toBeNull();
});

it("preserves the response filters in the detail link", async () => {
 render(<Responses/>); await screen.findByText(/Ada Bello/); fireEvent.change(screen.getByLabelText("Concern"),{target:{value:"HIGH"}});
 const link=await screen.findByRole("link",{name:"Access review"}); expect(link.getAttribute("href")).toContain("response_concern=HIGH");
});

it("restores response filters from a detail return link", async () => {
 window.history.replaceState(null,"","#/forms/responses/revision-1?response_concern=HIGH&response_sort=CONCERN_DESC&response_search=Access");
 const view=render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
 expect((await screen.findByRole("link",{name:"← Responses"})).getAttribute("href")).toBe("#/forms/responses?response_search=Access&response_concern=HIGH&response_sort=CONCERN_DESC");
 view.unmount(); window.history.replaceState(null,"","#/forms/responses?response_search=Access&response_concern=HIGH&response_sort=CONCERN_DESC");
 render(<Responses/>); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({search:"Access",bands:["HIGH"],sort:"CONCERN_DESC"})));
 window.history.replaceState(null,"","#/forms/responses");
});

it("rejects a repeated response cursor without appending the page", async () => {
 responses.loadCompletedResponses.mockResolvedValueOnce({items:[summary],next_cursor:"same"}).mockResolvedValueOnce({items:[{...summary,id:"duplicate",title:"Duplicate result"}],next_cursor:"same"});
 render(<Responses/>); fireEvent.click(await screen.findByRole("button",{name:"Load more"}));
 expect(await screen.findByText("More responses unavailable.")).toBeTruthy(); expect(screen.queryByText("Duplicate result")).toBeNull(); expect(screen.getByRole("button",{name:"Load more"})).toBeTruthy();
});

it.each([{state:"FINAL",final:false},{state:"PROVISIONAL",final:true}])("does not present a score with contradictory finality %s",async (contradiction)=>{
 responses.loadCompletedResponses.mockResolvedValue({items:[{...summary,score:{...contradiction,mode:"COMPLIANCE",raw_score:42}}]}); render(<Responses/>); expect(await screen.findByText("Score unavailable")).toBeTruthy(); expect(screen.queryByText(/42 \/ 100/)).toBeNull();
});

it("rejects an overlong route search without querying or truncating it", async () => {
 const old=window.location.hash; window.history.replaceState(null,"",`#/forms/responses?response_search=${encodeURIComponent("é".repeat(101))}`);
 render(<Responses/>); expect(await screen.findByText("Search is too long. Shorten the form title.")).toBeTruthy(); expect(responses.loadCompletedResponses).not.toHaveBeenCalled();
 fireEvent.change(screen.getByLabelText("Find a response"),{target:{value:"Access"}}); fireEvent.click(screen.getByRole("button",{name:"Search"})); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenCalledWith(expect.objectContaining({search:"Access"}))); window.history.replaceState(null,"",old||"#/forms/responses");
});
it("rejects an overlong typed search without issuing a query",async()=>{
 render(<Responses/>); await screen.findByText(/Ada Bello/); responses.loadCompletedResponses.mockClear(); fireEvent.change(screen.getByLabelText("Find a response"),{target:{value:"é".repeat(101)}}); fireEvent.click(screen.getByRole("button",{name:"Search"})); expect(await screen.findByText("Search is too long. Shorten the form title.")).toBeTruthy(); expect(responses.loadCompletedResponses).not.toHaveBeenCalled();
});
it("shows only a stored valid concern band",async()=>{
 responses.loadCompletedResponses.mockResolvedValue({items:[{...summary,score:{state:"FINAL",final:true,mode:"COMPLIANCE",raw_score:42,band:"HIGH"}},{...summary,id:"unknown",title:"Unknown",score:{state:"FINAL",final:false,mode:"COMPLIANCE",raw_score:42,band:"CRITICAL"}},{...summary,id:"missing",title:"Missing"}]}); render(<Responses/>); expect(await screen.findByText("High concern")).toBeTruthy(); expect(screen.getAllByText("Concern unknown")).toHaveLength(2); expect(screen.queryByText("Critical concern")).toBeNull();
});

it("applies the inclusive UTC submission dates to the server query and next page", async () => {
 responses.loadCompletedResponses.mockResolvedValue({items:[summary],next_cursor:"page-2"});
 render(<Responses/>); await screen.findByText(/Ada Bello/);
 fireEvent.change(screen.getByLabelText("Submitted from (UTC)"),{target:{value:"2026-11-01"}});
 fireEvent.change(screen.getByLabelText("Submitted through (UTC)"),{target:{value:"2026-11-30"}});
 await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({completed_from:"2026-11-01T00:00:00Z",completed_until:"2026-11-30T23:59:59.999999Z"})));
 fireEvent.click(await screen.findByRole("button",{name:"Load more"})); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({cursor:"page-2",completed_from:"2026-11-01T00:00:00Z",completed_until:"2026-11-30T23:59:59.999999Z"})));
 expect(screen.getByRole("link",{name:"Access review"}).getAttribute("href")).toContain("response_from=2026-11-01");
});
it("rejects a reversed submission range without querying",async()=>{
 render(<Responses/>); await screen.findByText(/Ada Bello/); fireEvent.change(screen.getByLabelText("Submitted from (UTC)"),{target:{value:"2026-12-01"}}); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({completed_from:"2026-12-01T00:00:00Z"}))); responses.loadCompletedResponses.mockClear();
 fireEvent.change(screen.getByLabelText("Submitted through (UTC)"),{target:{value:"2026-11-01"}}); expect(await screen.findByText("Check the submission dates.")).toBeTruthy(); expect(responses.loadCompletedResponses).not.toHaveBeenCalled();
});

it("blocks malformed date filters from the link until cleared",async()=>{
 const old=window.location.hash; window.history.replaceState(null,"","#/forms/responses?response_from=2026-02-30"); render(<Responses/>);
 expect(await screen.findByText("Check the submission dates.")).toBeTruthy(); expect(responses.loadCompletedResponses).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"Clear dates"})); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenCalledWith(expect.not.objectContaining({completed_from:expect.anything()}))); window.history.replaceState(null,"",old||"#/forms/responses");
});
it("restores date filters from a response detail link",async()=>{
 const old=window.location.hash; window.history.replaceState(null,"","#/forms/responses/revision-1?response_from=2026-11-01&response_through=2026-11-30"); render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
 expect((await screen.findByRole("link",{name:"← Responses"})).getAttribute("href")).toBe("#/forms/responses?response_from=2026-11-01&response_through=2026-11-30"); window.history.replaceState(null,"",old||"#/forms/responses");
});

it("filters responses by subject on the server and retains it across pagination",async()=>{
 responses.loadCompletedResponses.mockResolvedValueOnce({items:[summary],next_cursor:"initial"}).mockResolvedValueOnce({items:[summary],next_cursor:"next"}).mockResolvedValueOnce({items:[]}); render(<Responses/>); await screen.findByText(/Ada Bello/);
 fireEvent.change(screen.getByLabelText("Subject"),{target:{value:"MATTER"}}); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({subject_type:"MATTER"})));
 expect((await screen.findByRole("link",{name:"Access review"})).getAttribute("href")).toContain("response_subject=MATTER"); fireEvent.click(screen.getByRole("button",{name:"Load more"})); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({subject_type:"MATTER",cursor:"next"})));
});
it("blocks an unsupported subject in a link rather than broadening the query",async()=>{
 const old=window.location.hash; window.history.replaceState(null,"","#/forms/responses?response_subject=OTHER"); render(<Responses/>); expect(await screen.findByText("Choose an available subject." )).toBeTruthy(); expect(responses.loadCompletedResponses).not.toHaveBeenCalled(); fireEvent.click(screen.getByRole("button",{name:"Clear subject"})); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenCalled()); window.history.replaceState(null,"",old||"#/forms/responses");
});

it("retains the selected subject through response and previous-version navigation",async()=>{
 const old=window.location.hash; window.history.replaceState(null,"","#/forms/responses/revision-1?response_subject=MATTER&response_from=2026-11-01"); responses.loadCompletedResponse.mockResolvedValue({response:{...summary,distribution_id:"distribution"},revision:{id:"revision-1",revision:2,current:true,supersedes_revision_id:"previous"},answers:[],documents:[]});
 render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
 expect((await screen.findByRole("link",{name:"← Responses"})).getAttribute("href")).toContain("response_subject=MATTER");
 const previous=(screen.getByRole("link",{name:"Previous response"}) as HTMLAnchorElement).getAttribute("href")??""; expect(previous).toContain("response_subject=MATTER"); expect(previous).toContain("distribution_id=distribution"); window.history.replaceState(null,"",old||"#/forms/responses");
});
it("discards an older page when subject changes",async()=>{
 let finish!: (value: unknown)=>void; responses.loadCompletedResponses.mockResolvedValueOnce({items:[summary],next_cursor:"old"}).mockReturnValueOnce(new Promise(resolve=>{finish=resolve})).mockResolvedValueOnce({items:[{...summary,id:"new",title:"New scope"}]});
 render(<Responses/>); fireEvent.click(await screen.findByRole("button",{name:"Load more"})); fireEvent.change(screen.getByLabelText("Subject"),{target:{value:"PROGRAM"}}); expect(await screen.findByText("New scope")).toBeTruthy(); await act(async()=>{finish({items:[{...summary,id:"stale",title:"Stale scope"}]});}); expect(screen.queryByText("Stale scope")).toBeNull();
});
it("does not broaden a link with duplicate subject filters",async()=>{
 const old=window.location.hash; window.history.replaceState(null,"","#/forms/responses?response_subject=MATTER&response_subject=PROGRAM"); render(<Responses/>); expect(await screen.findByText("Choose an available subject.")).toBeTruthy(); expect(responses.loadCompletedResponses).not.toHaveBeenCalled(); window.history.replaceState(null,"",old||"#/forms/responses");
});

it.each(["response_concern=HIGH&response_concern=LOW","response_concern=UNKNOWN","response_sort=UNKNOWN","response_sort=CONCERN_DESC&response_sort=COMPLETED_DESC","response_from=2026-01-01&response_from=2026-02-01","response_through=2026-01-01&response_through=2026-02-01"])("does not broaden a malformed filter link: %s",async(query)=>{
 const old=window.location.hash; window.history.replaceState(null,"",`#/forms/responses?${query}`); render(<Responses/>); expect(await screen.findByText("Check the response filters.")).toBeTruthy(); expect(responses.loadCompletedResponses).not.toHaveBeenCalled(); fireEvent.click(screen.getByRole("button",{name:"Clear filters"})); await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenCalled()); window.history.replaceState(null,"",old||"#/forms/responses");
});

it("removes malformed filter parameters when clearing them",async()=>{
 const old=window.location.hash;window.history.replaceState(null,"","#/forms/responses?response_concern=UNKNOWN"); render(<Responses/>);fireEvent.click(await screen.findByRole("button",{name:"Clear filters"}));await vi.waitFor(()=>expect(responses.loadCompletedResponses).toHaveBeenCalled()); expect(window.location.hash).not.toContain("response_concern");window.history.replaceState(null,"",old||"#/forms/responses");
});

it("shows the verified recorded form result on the exact response",async()=>{
 const score={state:"FINAL",final:true,mode:"COMPLIANCE",direction:"LOW_IS_POOR",raw_score:42,band:"HIGH",coverage:0.9,profile_version:"policy-1"};
 responses.loadCompletedResponse.mockResolvedValue({response:{...summary,score},revision:{id:"revision-1",revision:2,current:true,score},answers:[],documents:[]}); render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);
 expect(await screen.findByText("Form compliance score: 42 / 100 · Final")).toBeTruthy(); expect(screen.getByRole("heading",{name:"Submitted result"})).toBeTruthy(); expect(screen.getByText("High concern")).toBeTruthy();
});
it("does not present conflicting response and revision scores",async()=>{
 const score={state:"FINAL",final:true,mode:"COMPLIANCE",raw_score:42,band:"HIGH"}; responses.loadCompletedResponse.mockResolvedValue({response:{...summary,score},revision:{id:"revision-1",revision:2,current:true,score:{...score,raw_score:90}},answers:[],documents:[]}); render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); expect(await screen.findByText("Score unavailable")).toBeTruthy(); expect(screen.queryByText(/42 \/ 100/)).toBeNull(); expect(screen.queryByText(/90 \/ 100/)).toBeNull();
});
it("distinguishes an explicitly unscored response from a missing score",async()=>{
 const score={state:"NOT_CONFIGURED",final:false}; responses.loadCompletedResponse.mockResolvedValueOnce({response:{...summary,score},revision:{id:"revision-1",revision:2,current:true,score},answers:[],documents:[]}).mockResolvedValueOnce({response:summary,revision:{id:"revision-1",revision:2,current:true},answers:[],documents:[]}); const view=render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); expect(await screen.findByText("Not scored")).toBeTruthy();view.unmount();render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>);expect(await screen.findByText("Score unavailable")).toBeTruthy();
});

it.each([
 {name:"missing revision score", revisionScore:undefined, revisionNumber:2},
 {name:"missing finality", revisionScore:{state:"FINAL",mode:"COMPLIANCE",raw_score:42}, revisionNumber:2, summaryScore:{state:"FINAL",mode:"COMPLIANCE",raw_score:42}},
 {name:"different revision", revisionScore:{state:"FINAL",final:true,mode:"COMPLIANCE",raw_score:42}, revisionNumber:1},
 {name:"different scoring profile", revisionScore:{state:"FINAL",final:true,mode:"COMPLIANCE",raw_score:42,profile_version:"other"}, revisionNumber:2},
])("keeps an unverified form result unavailable: $name",async(test)=>{
 const summaryScore=test.summaryScore??{state:"FINAL",final:true,mode:"COMPLIANCE",raw_score:42}; responses.loadCompletedResponse.mockResolvedValue({response:{...summary,score:summaryScore},revision:{id:"revision-1",revision:test.revisionNumber,current:true,score:test.revisionScore},answers:[],documents:[]}); render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); expect(await screen.findByText("Score unavailable")).toBeTruthy(); expect(screen.queryByText(/42 \/ 100/)).toBeNull();
});

it("does not call a contradictory score unscored",async()=>{
 const score={state:"NOT_CONFIGURED",final:false,mode:"COMPLIANCE"}; responses.loadCompletedResponse.mockResolvedValue({response:{...summary,score},revision:{id:"revision-1",revision:2,current:true,score},answers:[],documents:[]}); render(<Responses id="revision-1" reviewScope={{tenant:"tenant",entity:"entity",actor:"reviewer"}}/>); expect(await screen.findByText("Score unavailable")).toBeTruthy(); expect(screen.queryByText("Not scored")).toBeNull();
});
