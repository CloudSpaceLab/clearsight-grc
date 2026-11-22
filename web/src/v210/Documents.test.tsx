import { act, fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { Documents } from "./Documents";

const documents = vi.hoisted(() => ({ loadDocuments: vi.fn(), documentContentURL: vi.fn() }));
const http = vi.hoisted(() => ({ requestBlob: vi.fn(), apiErrorKind: vi.fn((_error: unknown): string => "unknown") }));
vi.mock("../submittedDocumentApi", () => documents);
vi.mock("../http", () => http);

beforeEach(() => { vi.clearAllMocks(); window.history.replaceState(null, "", "#/documents"); documents.documentContentURL.mockReturnValue("/protected/document"); });

it("browses file types in the document explorer using the scoped server filter", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  const navigation = await screen.findByRole("navigation", { name: "File types" });
  fireEvent.click(within(navigation).getByRole("button", { name: "PDF" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ file_kind: "PDF" }), expect.any(AbortSignal)));
  expect(window.location.hash).toContain("kind=PDF");
});

it.each([
  ["relationship_id", "relationship-1", "Vendor"],
  ["response_revision_id", "response-1", "Response"],
])("keeps the exact %s scope when filtering and loading another page", async (key, id, label) => {
  window.history.replaceState(null, "", `#/documents?${key}=${id}`);
  const file = { id: "1", file_name: "Scoped.pdf", form_title: "Review", current: true, artifact_status: "AVAILABLE", [key]: id };
  documents.loadDocuments.mockResolvedValueOnce({ items: [file], next_cursor: "next" }).mockResolvedValueOnce({ items: [], next_cursor: undefined }).mockResolvedValue({ items: [] });
  render(<Documents/>);
  expect(await screen.findByText(`${label} documents`)).toBeTruthy();
  expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ [key]: id }), expect.any(AbortSignal));
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ [key]: id, cursor: "next" })));
  fireEvent.change(screen.getByLabelText("File type"), { target: { value: "PDF" } });
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ [key]: id, file_kind: "PDF" }), expect.any(AbortSignal)));
  expect(window.location.hash).toContain(`${key}=${id}`);
});

it.each(["relationship_id=one&relationship_id=two", "response_revision_id=", "relationship_id=one&response_revision_id=two"])("rejects invalid document scope %s without loading all documents", async (query) => {
  window.history.replaceState(null, "", `#/documents?${query}`);
  render(<Documents/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(documents.loadDocuments).not.toHaveBeenCalled();
});

it("clears scoped documents when a later page contains a different relationship", async () => {
  window.history.replaceState(null, "", "#/documents?relationship_id=relationship-1");
  const file = { id: "1", file_name: "Scoped.pdf", form_title: "Review", current: true, artifact_status: "AVAILABLE", relationship_id: "relationship-1" };
  documents.loadDocuments.mockResolvedValueOnce({ items: [file], next_cursor: "next" }).mockResolvedValueOnce({ items: [{ ...file, id: "2", relationship_id: "relationship-2" }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Scoped.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect(await screen.findByText("Documents unavailable.")).toBeTruthy();
  expect(screen.queryByText("Scoped.pdf")).toBeNull();
});

it("rejects a mismatched response page and does not show its contents", async () => {
  window.history.replaceState(null, "", "#/documents?response_revision_id=revision-1");
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Other.pdf", response_revision_id: "revision-2" }] });
  render(<Documents/>);
  expect(await screen.findByText("Documents unavailable.")).toBeTruthy();
  expect(screen.queryByText("Other.pdf")).toBeNull();
});

it("allows only the matching vendor assessment as a return destination", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [] });
  window.history.replaceState(null, "", `#/documents?relationship_id=rel-1&return_to=${encodeURIComponent("#/vendors/rel-2/assessments/assess-1")}`);
  render(<Documents/>);
  expect((await screen.findByRole("link", { name: "← Vendor" })).getAttribute("href")).toBe("#/vendors/rel-1");
  const valid = "#/vendors/rel-1/assessments/assess-1";
  window.history.replaceState(null, "", `#/documents?relationship_id=rel-1&return_to=${encodeURIComponent(valid)}`);
  fireEvent(window, new PopStateEvent("popstate"));
  await vi.waitFor(() => expect(screen.getByRole("link", { name: "← Vendor" }).getAttribute("href")).toBe(valid));
});

it("preserves the response distribution when returning to a scoped response", async () => {
  const destination = "#/forms/responses/revision-1?distribution_id=distribution-1";
  window.history.replaceState(null, "", `#/documents?response_revision_id=revision-1&return_to=${encodeURIComponent(destination)}`);
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  expect((await screen.findByRole("link", { name: "← Response" })).getAttribute("href")).toBe(destination);
});

it("preserves safe response filters when returning to a scoped response", async () => {
  const destination = "#/forms/responses/revision-1?response_concern=HIGH&response_sort=CONCERN_DESC";
  window.history.replaceState(null, "", `#/documents?response_revision_id=revision-1&return_to=${encodeURIComponent(destination)}`);
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  expect((await screen.findByRole("link", { name: "← Response" })).getAttribute("href")).toBe(destination);
});

it("preserves response submission dates when returning to a scoped response", async () => {
  const destination = "#/forms/responses/revision-1?response_from=2026-11-01&response_through=2026-11-22&response_subject=PROGRAM";
  window.history.replaceState(null, "", `#/documents?response_revision_id=revision-1&return_to=${encodeURIComponent(destination)}`);
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  expect((await screen.findByRole("link", { name: "← Response" })).getAttribute("href")).toBe(destination);
});

it("rejects an untrusted response return query", async () => {
  const destination = "#/forms/responses/revision-1?response_concern=HIGH&return_to=https://example.test";
  window.history.replaceState(null, "", `#/documents?response_revision_id=revision-1&return_to=${encodeURIComponent(destination)}`);
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  expect((await screen.findByRole("link", { name: "← Response" })).getAttribute("href")).toBe("#/forms/responses/revision-1");
});

it("hides a previous vendor's documents immediately when the scope changes", async () => {
  window.history.replaceState(null, "", "#/documents?relationship_id=rel-1");
  documents.loadDocuments.mockResolvedValueOnce({ items: [{ id: "1", file_name: "First.pdf", form_title: "Review", relationship_id: "rel-1", current: true, artifact_status: "AVAILABLE" }] }).mockReturnValueOnce(new Promise(() => {}));
  render(<Documents/>);
  expect(await screen.findByRole("button", { name: "Open First.pdf" })).toBeTruthy();
  window.history.replaceState(null, "", "#/documents?relationship_id=rel-2");
  fireEvent(window, new PopStateEvent("popstate"));
  expect(screen.queryByText("First.pdf")).toBeNull();
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ relationship_id: "rel-2" }), expect.any(AbortSignal)));
});

it("clears the document list and selected detail when access is revoked during pagination", async () => {
  documents.loadDocuments.mockResolvedValueOnce({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }], next_cursor: "next" }).mockRejectedValueOnce(new Error("forbidden"));
  http.apiErrorKind.mockReturnValueOnce("forbidden");
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect(await screen.findByText("Documents unavailable.")).toBeTruthy();
  expect(screen.queryByText("Report.pdf")).toBeNull();
  expect(screen.queryByRole("button", { name: "Load more" })).toBeNull();
});

it("does not append a document page when the server repeats its cursor", async () => {
  const file = { id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" };
  documents.loadDocuments.mockResolvedValueOnce({ items: [file], next_cursor: "next" }).mockResolvedValueOnce({ items: [file], next_cursor: "next" });
  render(<Documents/>);
  await screen.findByRole("button", { name: "Open Report.pdf" });
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect(await screen.findByText("More documents unavailable.")).toBeTruthy();
  expect(screen.getAllByRole("button", { name: "Open Report.pdf" })).toHaveLength(1);
});

it("shows the stored assessment review date in document details", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", review: { source: "VENDOR_ASSESSMENT", status: "VALIDATED", reviewed_by: "principal", reviewed_at: "2026-11-19T12:00:00Z" } }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  expect(screen.getByText("Assessment document review")).toBeTruthy();
  expect(screen.getByText("Validated")).toBeTruthy();
  expect(screen.getByText("Reviewed").nextElementSibling?.querySelector("time")?.getAttribute("datetime")).toBe("2026-11-19T12:00:00Z");
  expect(screen.getByText("Reviewed by").nextElementSibling?.textContent).toBe("Person ID: principal");
});

it("keeps document upload, response submission and assessment review provenance distinct", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, artifact_status: "AVAILABLE", uploaded_at: "2026-11-16T08:00:00Z", uploaded_by: "upload-1", submitted_at: "2026-11-17T09:00:00Z", submitted_by: "response-2", review: { source: "VENDOR_ASSESSMENT", status: "VALIDATED", reviewed_by: "review-3", reviewed_at: "2026-11-18T10:00:00Z" } }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  for (const [label, id] of [["Uploaded by", "upload-1"], ["Submitted by", "response-2"], ["Reviewed by", "review-3"]] as const) expect(screen.getByText(label).nextElementSibling?.textContent).toBe(`Person ID: ${id}`);
  for (const [label, instant] of [["Uploaded", "2026-11-16T08:00:00Z"], ["Submitted", "2026-11-17T09:00:00Z"], ["Reviewed", "2026-11-18T10:00:00Z"]] as const) expect(screen.getByText(label).nextElementSibling?.querySelector("time")?.getAttribute("datetime")).toBe(instant);
});

it("rejects filename searches exceeding the API's UTF-8 byte limit", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  await screen.findByText("No documents match this view.");
  const calls = documents.loadDocuments.mock.calls.length;
  fireEvent.change(screen.getByLabelText("Find a document"), { target: { value: "é".repeat(101) } });
  fireEvent.submit(screen.getByLabelText("Find a document").closest("form")!);
  expect(screen.getByRole("alert").textContent).toContain("Search is too long. Shorten the document name.");
  expect(documents.loadDocuments).toHaveBeenCalledTimes(calls);
});

it("restores an overlong route search without sending it and clears the error on navigation", async () => {
  const tooLong = "é".repeat(101);
  window.history.replaceState(null, "", `#/documents?q=${encodeURIComponent(tooLong)}`);
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Search is too long");
  expect(documents.loadDocuments).not.toHaveBeenCalled();
  window.history.replaceState(null, "", "#/documents?q=policy");
  fireEvent(window, new PopStateEvent("popstate"));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ query: "policy" }), expect.any(AbortSignal)));
  expect(screen.queryByRole("alert")).toBeNull();
});

it("does not silently truncate an overlong search from a shared link", async () => {
  window.history.replaceState(null, "", `#/documents?q=${"a".repeat(201)}`);
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Search is too long");
  expect(documents.loadDocuments).not.toHaveBeenCalled();
});

it("restores the document view, search and file type from the route", async () => {
  window.history.replaceState({ retained: true }, "", "#/documents?view=NEEDS_REFRESH&q=policy&kind=PDF");
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ view: "NEEDS_REFRESH", query: "policy", file_kind: "PDF" }), expect.any(AbortSignal)));
  expect((screen.getByLabelText("Find a document") as HTMLInputElement).value).toBe("policy");
  expect((screen.getByLabelText("File type") as HTMLSelectElement).value).toBe("PDF");
});

it("preserves route filters and history state as the document view changes", async () => {
  window.history.replaceState({ retained: true }, "", "#/documents");
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  fireEvent.change(screen.getByLabelText("Find a document"), { target: { value: "policy" } });
  fireEvent.submit(screen.getByLabelText("Find a document").closest("form")!);
  fireEvent.change(screen.getByLabelText("File type"), { target: { value: "PDF" } });
  fireEvent.click(screen.getByRole("button", { name: "Needs refresh" }));
  expect(window.location.hash).toContain("q=policy");
  expect(window.location.hash).toContain("kind=PDF");
  expect(window.location.hash).toContain("view=NEEDS_REFRESH");
  expect(window.history.state).toEqual({ retained: true });
});

it("restores route defaults when returning to an unfiltered document route", async () => {
  window.history.replaceState(null, "", "#/documents?view=NEEDS_REFRESH&q=policy&kind=PDF");
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  window.history.replaceState(null, "", "#/documents");
  fireEvent(window, new PopStateEvent("popstate"));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ view: undefined, query: undefined, file_kind: undefined }), expect.any(AbortSignal)));
  expect((screen.getByLabelText("Find a document") as HTMLInputElement).value).toBe("");
});

it("restores the exact form filter when returning to documents", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Vendor review", form_template_id: "form/1", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  const first = render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "View form documents" }));
  expect(window.location.hash).toContain("form=form%2F1");
  first.unmount();
  render(<Documents/>);
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ form_template_id: "form/1" }), expect.any(AbortSignal)));
  expect(await screen.findByText("Form: Vendor review")).toBeTruthy();
});

it("switches between list and grid without losing the selected document", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", file_kind: "PDF" }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "Grid view" }));
  expect(screen.getByRole("button", { name: "Grid view" }).getAttribute("aria-pressed")).toBe("true");
  expect(screen.getByRole("heading", { name: "Report.pdf" })).toBeTruthy();
  expect(screen.getByRole("button", { name: "Open Report.pdf" }).getAttribute("aria-pressed")).toBe("true");
  fireEvent.click(screen.getByRole("button", { name: "List view" }));
  expect(screen.getByRole("button", { name: "List view" }).getAttribute("aria-pressed")).toBe("true");
});

it("opens the same document from the details action and the title", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", file_kind: "PDF" }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "View details Report.pdf" }));
  expect(screen.getByRole("heading", { name: "Report.pdf" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Close document details" }));
  fireEvent.click(screen.getByRole("button", { name: "Open Report.pdf" }));
  expect(screen.getByRole("heading", { name: "Report.pdf" })).toBeTruthy();
});

it("does not offer downloads for files awaiting a scan", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: false }] });
  render(<Documents/>);
  expect(await screen.findByText("File unavailable")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Download" })).toBeNull();
  expect(http.requestBlob).not.toHaveBeenCalled();
});

it("reports a protected download failure", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  http.requestBlob.mockRejectedValue(new Error("forbidden"));
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Download" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Could not download Report.pdf");
  expect(http.requestBlob).toHaveBeenCalledWith("", "/protected/document");
});

it("opens exact document details and its response review", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", field_label: "Security report", current: true, submitted_at: "2026-11-20", submitted_by: "person-1", artifact_status: "AVAILABLE", file_kind: "PDF", form_template_version: 3, response_revision_id: "response/1", review: { status: "SUBMITTED", reviewed_by: "" } }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  expect(screen.getByRole("heading", { name: "Report.pdf" })).toBeTruthy();
  expect(screen.getByText("Security report")).toBeTruthy();
  expect(screen.getByText("Submitted by")).toBeTruthy();
  expect(screen.getByText("Submitted by").nextElementSibling?.textContent).toBe("Person ID: person-1");
  expect(screen.getByRole("link", { name: "Open response" }).getAttribute("href")).toBe("#/forms/responses/response%2F1?return_to=%23%2Fdocuments");
});

it("requests the selected file type from the authorized inventory", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  fireEvent.change(await screen.findByLabelText("File type"), { target: { value: "PDF" } });
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ file_kind: "PDF" }), expect.any(AbortSignal)));
});

it("identifies a validated assessment document without implying a universal approval", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", review: { status: "VALIDATED", reviewed_by: "person-2" } }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  expect(screen.getByText("Assessment document review")).toBeTruthy();
  expect(screen.getByText("Validated")).toBeTruthy();
});

it("requests the exact protected document before previewing", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", media_type: "application/pdf" }] });
  http.requestBlob.mockResolvedValue({ blob: new Blob(["sample"], { type: "application/pdf" }) });
  const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:document-test");
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  const { unmount } = render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  expect(await screen.findByTitle("Preview: Report.pdf")).toBeTruthy();
  expect(documents.documentContentURL).toHaveBeenCalledWith(expect.objectContaining({ id: "1" }));
  unmount();
  expect(revoke).toHaveBeenCalledWith("blob:document-test");
  create.mockRestore(); revoke.mockRestore();
});

it("offers download without requesting a PDF preview when the browser cannot display PDFs", async () => {
  const original = Object.getOwnPropertyDescriptor(navigator, "pdfViewerEnabled");
  Object.defineProperty(navigator, "pdfViewerEnabled", { configurable: true, value: false });
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "pdf", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", media_type: "application/pdf" }] });
  try {
    render(<Documents/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
    expect(screen.getByText("PDF preview unavailable in this browser. Download the document.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Download" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Download Report.pdf" })).toBeTruthy();
    expect(screen.queryByTitle("Preview: Report.pdf")).toBeNull();
    expect(http.requestBlob).not.toHaveBeenCalled();
    http.requestBlob.mockRejectedValueOnce(new Error("Unavailable"));
    fireEvent.click(screen.getByRole("button", { name: "Download Report.pdf" }));
    expect(await screen.findByText("Could not download Report.pdf. Try again.")).toBeTruthy();
    expect(documents.documentContentURL).toHaveBeenCalledWith(expect.objectContaining({ id: "pdf" }), true);
  } finally {
    if (original) Object.defineProperty(navigator, "pdfViewerEnabled", original);
    else Reflect.deleteProperty(navigator, "pdfViewerEnabled");
  }
});

it("does not offer a restricted PDF download when browser preview is unavailable", async () => {
  const original = Object.getOwnPropertyDescriptor(navigator, "pdfViewerEnabled");
  Object.defineProperty(navigator, "pdfViewerEnabled", { configurable: true, value: false });
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "pdf", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "STORED_UNSCANNED", demo_preview_available: true, demo_unscanned_allowed: false, media_type: "application/pdf" }] });
  try {
    render(<Documents/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
    expect(screen.getByText("PDF preview unavailable in this browser.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: /download/i })).toBeNull();
    expect(http.requestBlob).not.toHaveBeenCalled();
  } finally {
    if (original) Object.defineProperty(navigator, "pdfViewerEnabled", original);
    else Reflect.deleteProperty(navigator, "pdfViewerEnabled");
  }
});

it.each([
  ["wrong media type", new Blob(["hello"], { type: "text/html" })],
  ["truncated content", new Blob(["bad"], { type: "application/pdf" })],
])("does not present %s as a document preview and permits retry", async (_label, invalid) => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "file", file_name: "Report.pdf", media_type: "application/pdf", size_bytes: 5, form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  http.requestBlob.mockResolvedValueOnce({ blob: invalid }).mockResolvedValueOnce({ blob: new Blob(["hello"], { type: "application/pdf" }) });
  const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:verified");
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  try {
    render(<Documents/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
    expect(await screen.findByText("Preview unavailable. Try again.")).toBeTruthy();
    expect(create).not.toHaveBeenCalled();
    expect(screen.queryByTitle("Preview: Report.pdf")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Preview document" }));
    expect(await screen.findByTitle("Preview: Report.pdf")).toBeTruthy();
    expect(create).toHaveBeenCalledTimes(1);
  } finally { create.mockRestore(); revoke.mockRestore(); }
});

it("allows a valid media type when the document size is unavailable", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "file", file_name: "Report.pdf", media_type: "application/pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  http.requestBlob.mockResolvedValue({ blob: new Blob(["hello"], { type: "application/pdf" }) });
  const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:verified");
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  try {
    render(<Documents/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
    expect(await screen.findByTitle("Preview: Report.pdf")).toBeTruthy();
    expect(http.requestBlob).toHaveBeenCalledWith("", "/protected/document", expect.objectContaining({ cache: "no-store", signal: expect.any(AbortSignal) }));
  } finally { create.mockRestore(); revoke.mockRestore(); }
});

it("normalizes the stored media type in the same way as the document endpoint", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "file", file_name: "Report.pdf", media_type: "Application/PDF; charset=binary", size_bytes: 5, form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  http.requestBlob.mockResolvedValue({ blob: new Blob(["hello"], { type: "application/pdf" }) });
  const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:verified");
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  try {
    render(<Documents/>);
    fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
    expect(await screen.findByTitle("Preview: Report.pdf")).toBeTruthy();
  } finally { create.mockRestore(); revoke.mockRestore(); }
});

it("does not request a preview for an unscanned document without permission", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "STORED_UNSCANNED", media_type: "application/pdf" }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  expect(http.requestBlob).not.toHaveBeenCalled();
  expect(screen.queryByTitle("Preview: Report.pdf")).toBeNull();
});

it("allows a failed protected preview to be retried", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", media_type: "application/pdf" }] });
  http.requestBlob.mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce({ blob: new Blob(["sample"], { type: "application/pdf" }) });
  const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:retry-test");
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  const { unmount } = render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Preview unavailable");
  fireEvent.click(screen.getByRole("button", { name: "Preview document" }));
  expect(await screen.findByTitle("Preview: Report.pdf")).toBeTruthy();
  unmount(); create.mockRestore(); revoke.mockRestore();
});

it("ignores a delayed preview after selecting a different document", async () => {
  const file = (id: string) => ({ id, file_name: `${id}.pdf`, form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", media_type: "application/pdf" });
  documents.loadDocuments.mockResolvedValue({ items: [file("first"), file("second")] });
  let finishFirst!: (value: { blob: Blob }) => void;
  http.requestBlob.mockImplementationOnce(() => new Promise((resolve) => { finishFirst = resolve; })).mockResolvedValueOnce({ blob: new Blob(["second"], { type: "application/pdf" }) });
  const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:second");
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  const { unmount } = render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open first.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "Open second.pdf" }));
  expect(await screen.findByTitle("Preview: second.pdf")).toBeTruthy();
  finishFirst({ blob: new Blob(["first"], { type: "application/pdf" }) });
  await vi.waitFor(() => expect(create).toHaveBeenCalledTimes(1));
  expect(screen.queryByTitle("Preview: first.pdf")).toBeNull();
  unmount(); create.mockRestore(); revoke.mockRestore();
});

it("links assessment documents to their exact vendor review", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE", relationship_id: "vendor/1", assessment_id: "assessment/1" }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  expect(screen.getByRole("link", { name: "Open vendor assessment" }).getAttribute("href")).toBe("#/vendors/vendor%2F1/assessments/assessment%2F1?return_to=%23%2Fdocuments");
});

it("includes earlier document occurrences in review history", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [] });
  render(<Documents/>);
  fireEvent.click(screen.getByRole("button", { name: "Assessment review history" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ view: "REVIEW_HISTORY", current_only: false }), expect.any(AbortSignal)));
});

it("filters the inventory by the selected document's exact form and clears the scope", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Vendor review", form_template_id: "form/1", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "View form documents" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ form_template_id: "form/1" }), expect.any(AbortSignal)));
  expect(screen.getByText("Form: Vendor review")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Clear form filter" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ form_template_id: undefined }), expect.any(AbortSignal)));
});

it("warns before offering a permitted unscanned sample download", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Sample.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: true }] });
  render(<Documents/>);
  expect(await screen.findByText("Demo file. No antivirus scan was performed.")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Download" })).toBeTruthy();
});

it("keeps the exact form filter when changing document views and loading more", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Vendor review", form_template_id: "form-1", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }], next_cursor: "cursor-1" });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "View form documents" }));
  fireEvent.click(screen.getByRole("button", { name: "Needs refresh" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ view: "NEEDS_REFRESH", form_template_id: "form-1" }), expect.any(AbortSignal)));
  fireEvent.click(await screen.findByRole("button", { name: "Load more" }));
  await vi.waitFor(() => expect(documents.loadDocuments).toHaveBeenCalledWith(expect.objectContaining({ view: "NEEDS_REFRESH", form_template_id: "form-1", cursor: "cursor-1" })));
});

it("moves focus to document details and returns it to the triggering control", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  render(<Documents/>);
  const open = await screen.findByRole("button", { name: "View details Report.pdf" });
  open.focus();
  fireEvent.click(open);
  expect(document.activeElement).toBe(screen.getByRole("heading", { name: "Report.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "Close document details" }));
  expect(document.activeElement).toBe(open);
});

it("refocuses details when the selected document is opened again", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  render(<Documents/>);
  const open = await screen.findByRole("button", { name: "Open Report.pdf" });
  fireEvent.click(open);
  open.focus();
  fireEvent.click(open);
  expect(document.activeElement).toBe(screen.getByRole("heading", { name: "Report.pdf" }));
});

it("moves focus to the active form filter after leaving document details", async () => {
  documents.loadDocuments.mockResolvedValue({ items: [{ id: "1", file_name: "Report.pdf", form_title: "Review", form_template_id: "form-1", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" }] });
  render(<Documents/>);
  fireEvent.click(await screen.findByRole("button", { name: "Open Report.pdf" }));
  fireEvent.click(screen.getByRole("button", { name: "View form documents" }));
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Clear form filter" }));
});

it("does not duplicate a document page or append it after search changes", async () => {
  let finish!: (value: unknown) => void;
  const file = { id: "1", file_name: "Report.pdf", form_title: "Review", current: true, submitted_at: "2026-11-20", artifact_status: "AVAILABLE" };
  documents.loadDocuments.mockResolvedValueOnce({ items: [file], next_cursor: "old" }).mockReturnValueOnce(new Promise((resolve) => { finish = resolve; })).mockResolvedValueOnce({ items: [{ ...file, id: "new", file_name: "Risk.pdf" }], next_cursor: "new" });
  render(<Documents/>);
  const more = await screen.findByRole("button", { name: "Load more" });
  act(() => { fireEvent.click(more); fireEvent.click(more); });
  expect(documents.loadDocuments).toHaveBeenCalledTimes(2);
  fireEvent.change(screen.getByLabelText("Find a document"), { target: { value: "Risk" } });
  fireEvent.submit(screen.getByLabelText("Find a document").closest("form")!);
  expect(await screen.findByRole("button", { name: "Open Risk.pdf" })).toBeTruthy();
  await act(async () => { finish({ items: [{ ...file, id: "old", file_name: "Old.pdf" }] }); });
  expect(screen.queryByText("Old.pdf")).toBeNull();
  expect((screen.getByRole("button", { name: "Load more" }) as HTMLButtonElement).disabled).toBe(false);
});
