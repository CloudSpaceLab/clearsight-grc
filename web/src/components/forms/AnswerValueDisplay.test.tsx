import axe from "axe-core";
import { render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Responses } from "../../v210/Responses";

const responses = vi.hoisted(() => ({ loadCompletedResponses: vi.fn(), loadCompletedResponse: vi.fn() }));
const review = vi.hoisted(() => ({ loadResponseReviews: vi.fn(), recordResponseReview: vi.fn() }));
const documents = vi.hoisted(() => ({ loadDocuments: vi.fn(), documentContentURL: vi.fn(() => "/protected/document") }));
vi.mock("../../formsDistributionApi", () => responses);
vi.mock("../../responseReviewApi", () => review);
vi.mock("../../submittedDocumentApi", () => documents);

const summary = { id: "response-1", title: "Assessment", revision: 1, current: true, completed_at: "2026-11-20T10:00:00Z", submitted_by_display_name: "Ada Bello" };
const answer = (field_id: string, label: string, type: string, value?: unknown) => ({ field_id, label, type, value });
function display(label: string) { return screen.getByText(label).nextElementSibling; }
beforeEach(() => {
  vi.clearAllMocks();
  review.loadResponseReviews.mockResolvedValue({ version: 1, may_review: false, items: [] });
  documents.documentContentURL.mockReturnValue("/protected/document");
});
function response(answers: unknown[], submittedDocuments: unknown[] = []) {
  responses.loadCompletedResponse.mockResolvedValue({ response: summary, revision: { id: "response-1", current: true, revision: 1 }, answers, documents: submittedDocuments });
}

it("preserves text, exact numbers, dates and choices from the recorded response", async () => {
  response([
    answer("a", "Branch", "short_text", { text: "Lagos Island" }),
    answer("b", "Notes", "LONG_TEXT", { text: "First line\nSecond line" }),
    answer("c", "Count", "integer", { text: "1250000" }),
    answer("d", "Amount", "currency", { text: "9007199254740993.0100" }),
    answer("e", "Rate", "percentage", { text: "12.3450" }),
    answer("f", "Unrecognized", "integer", { text: "unavailable" }),
    answer("g", "Review date", "date", { text: "2025-11-28" }),
    answer("h", "Selection", "single_select", { text: "Retail Banking" }),
    answer("i", "Events", "multi_select", { values: ["Fraud attempt", "Operational outage"] }),
  ]);
  render(<Responses id="response-1"/>);
  expect(await screen.findByText("Lagos Island")).toBeTruthy();
  expect(display("Notes")?.textContent).toBe("First line\nSecond line");
  expect(display("Count")?.textContent).toBe("1250000");
  expect(display("Amount")?.textContent).toBe("9007199254740993.0100");
  expect(display("Rate")?.textContent).toBe("12.3450%");
  expect(display("Unrecognized")?.textContent).toBe("unavailable");
  expect(display("Review date")?.textContent).toBe(new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric", timeZone: "UTC" }).format(new Date("2025-11-28")));
  expect(display("Selection")?.textContent).toBe("Retail Banking");
  expect(display("Events")?.textContent).toBe("Fraud attempt, Operational outage");
});

it("distinguishes confirmation, missing answers and submitted documents", async () => {
  response([
    answer("a", "Access review", "yes_no", { values: ["Yes"] }),
    answer("b", "Confirmation", "checkbox", { text: "true" }),
    answer("c", "Attestation", "attestation", { text: "false" }),
    answer("d", "Files", "file", { artifact_ids: ["a1", "a2"] }),
    answer("e", "Signature", "signature", { artifact_ids: ["s1"] }),
    answer("f", "Missing", "short_text"),
  ], [{ id: "document-1", file_name: "KRI report.pdf", artifact_status: "AVAILABLE" }]);
  render(<Responses id="response-1"/>);
  expect(await screen.findByText("Yes")).toBeTruthy();
  expect(display("Confirmation")?.textContent).toBe("Confirmed");
  expect(display("Attestation")?.textContent).toBe("Not confirmed");
  expect(display("Files")?.textContent).toBe("2 documents submitted");
  expect(display("Signature")?.textContent).toBe("Signature submitted");
  expect(display("Missing")?.textContent).toBe("No answer");
  expect(screen.getByText("KRI report.pdf")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Download" }).getAttribute("href")).toBe("/protected/document");
});

it("shows recorded document metadata and expiry", async () => {
  response([answer("a", "Certificate", "vendor_document", { document: { artifact_id: "d1", document_type: "Certificate", reference: "REF-42", expires_on: "2026-03-01" } })]);
  render(<Responses id="response-1"/>);
  expect(await screen.findByText("Type: Certificate")).toBeTruthy();
  expect(screen.getByText("Reference: REF-42")).toBeTruthy();
  expect(screen.getByText(/Expires:/)).toBeTruthy();
});

it("keeps the answer workspace accessible", async () => {
  response([answer("a", "Events", "multi_select", { values: ["Option A", "Option B"] })]);
  const view = render(<Responses id="response-1"/>);
  expect(await screen.findByText("Option A, Option B")).toBeTruthy();
  const results = await axe.run(view.container, { rules: { "color-contrast": { enabled: false } } });
  expect(results.violations.map((violation) => violation.id)).toEqual([]);
});
