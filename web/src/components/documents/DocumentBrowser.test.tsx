import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadDocuments, type DocumentOccurrence } from "../../submittedDocumentApi";
import { DocumentBrowser } from "./DocumentBrowser";

vi.mock("../../submittedDocumentApi", async (original) => ({ ...await original<typeof import("../../submittedDocumentApi")>(), loadDocuments: vi.fn() }));
const file: DocumentOccurrence = {
  id: "occurrence-1", artifact_id: "artifact-1", request_id: "request-1", submission_id: "submission-1", field_id: "certificate",
  form_template_version: 1, form_title: "Vendor due diligence", field_label: "Insurance certificate", file_name: "Insurance 2026.docx",
  media_type: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", file_kind: "WORD", size_bytes: 4096,
  sha256: "sample-digest", artifact_status: "AVAILABLE", uploaded_at: "2026-09-01T10:00:00Z", submitted_at: "2026-09-02T12:00:00Z", current: true,
};
beforeEach(() => { vi.mocked(loadDocuments).mockReset().mockResolvedValue({ items: [file] }); });

describe("DocumentBrowser", () => {
  it("labels the current page count in singular or plural from the returned rows", async () => {
    render(<DocumentBrowser scopeLabel="Vendor documents" relationshipID="vendor-1"/>);
    expect(await screen.findByText("1 file on this page")).toBeTruthy();
    vi.mocked(loadDocuments).mockResolvedValue({ items: [file, { ...file, id: "occurrence-2" }], next_cursor: "next-page" });
    fireEvent.click(screen.getByRole("button", { name: "Refresh files" }));
    expect(await screen.findByText("2 files on this page · More available")).toBeTruthy();
  });

  it("uses the compact type selector to reset pagination and preserve exact response scope and search", async () => {
    vi.mocked(loadDocuments).mockResolvedValue({ items: [file], next_cursor: "page-2" });
    render(<DocumentBrowser scopeLabel="Response documents" relationshipID="vendor-1" responseRevisionID="revision-1"/>);
    await screen.findByRole("row", { name: /Insurance 2026.docx/ });
    fireEvent.change(screen.getByRole("searchbox", { name: "Search file names" }), { target: { value: "Insurance" } });
    fireEvent.click(await screen.findByRole("button", { name: "Next page" }));
    await waitFor(() => expect(loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "page-2" }), expect.any(AbortSignal)));
    fireEvent.click(screen.getByRole("button", { name: "All files File type" }));
    await screen.findByRole("listbox");
    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual(["All files", "PDF files", "Images", "Word documents", "Spreadsheets", "Other files"]);
    fireEvent.click(screen.getByRole("option", { name: "Word documents" }));
    await waitFor(() => expect(loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ file_kind: "WORD", query: "Insurance", cursor: undefined, relationship_id: "vendor-1", response_revision_id: "revision-1", current_only: false, limit: 25 }), expect.any(AbortSignal)));
    expect(loadDocuments).toHaveBeenCalledTimes(4);
    expect(screen.getByRole("button", { name: "Word documents" }).getAttribute("aria-pressed")).toBe("true");
    expect(screen.getByRole("button", { name: "Word documents File type" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Previous page" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "PDF files" }));
    expect(screen.getByRole("button", { name: "PDF files File type" })).toBeTruthy();
  });

  it("keeps selection and preview on resize or choosing the current type", async () => {
    render(<DocumentBrowser scopeLabel="Vendor documents" relationshipID="vendor-1"/>);
    const row = await screen.findByRole("row", { name: /Insurance 2026.docx/ });
    expect(row.querySelector('[data-label="Name"]')?.getAttribute("data-mobile-layout")).toBe("full-width");
    fireEvent.focus(row);
    fireEvent.click(screen.getByRole("button", { name: "All files File type" }));
    fireEvent.click(await screen.findByRole("option", { name: "All files" }));
    expect(row.getAttribute("aria-selected")).toBe("true");
    fireEvent.keyDown(row, { key: " " });
    const preview = await screen.findByRole("dialog", { name: "Preview Insurance 2026.docx" });
    fireEvent.resize(window);
    expect(screen.getByRole("dialog", { name: "Preview Insurance 2026.docx" })).toBe(preview);
    expect(loadDocuments).toHaveBeenCalledTimes(1);
  });

  it("opens Word file details without claiming an unsupported inline preview", async () => {
    render(<DocumentBrowser scopeLabel="Vendor documents" relationshipID="vendor-1"/>);
    const row = await screen.findByRole("row", { name: /Insurance 2026.docx/ });
    fireEvent.keyDown(row, { key: " " });
    expect(await screen.findByRole("dialog", { name: "Preview Insurance 2026.docx" })).toBeTruthy();
    expect(screen.getByText("Inline preview is not available for this file type. Download the file to view it in its application.")).toBeTruthy();
    expect(screen.getByRole("link", { name: "Download file" }).getAttribute("href")).toContain("/submission-1/certificate/artifact-1/content?download=true");
    expect(screen.getAllByText("Not recorded").length).toBeGreaterThan(0);
    expect(document.querySelector("iframe, img")).toBeNull();
  });

  it("filters file types and filenames on the server while preserving vendor scope", async () => {
    render(<DocumentBrowser scopeLabel="Vendor documents" relationshipID="vendor-1"/>);
    await screen.findByRole("row", { name: /Insurance 2026.docx/ });
    fireEvent.click(screen.getByRole("button", { name: "PDF files" }));
    await waitFor(() => expect(loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ file_kind: "PDF", relationship_id: "vendor-1" }), expect.any(AbortSignal)));
    fireEvent.change(screen.getByRole("searchbox", { name: "Search file names" }), { target: { value: "audit" } });
    await waitFor(() => expect(loadDocuments).toHaveBeenLastCalledWith(expect.objectContaining({ query: "audit", file_kind: "PDF" }), expect.any(AbortSignal)));
  });

  it("shows inspection restrictions without an open or download link", async () => {
    vi.mocked(loadDocuments).mockResolvedValue({ items: [{ ...file, artifact_status: "QUARANTINED" }] });
    render(<DocumentBrowser scopeLabel="Submitted documents"/>);
    fireEvent.keyDown(await screen.findByRole("row", { name: /Insurance 2026.docx/ }), { key: "Enter" });
    expect(await screen.findByText("This file is quarantined. It cannot be previewed or downloaded.")).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Download file" })).toBeNull();
  });

  it("offers recovery after a failed inventory read without reporting an empty population", async () => {
    vi.mocked(loadDocuments).mockRejectedValue(new Error("offline"));
    render(<DocumentBrowser scopeLabel="Submitted documents"/>);
    expect(await screen.findByText("Documents could not be loaded.")).toBeTruthy();
    expect(screen.queryByText("No matching documents")).toBeNull();
    vi.mocked(loadDocuments).mockResolvedValue({ items: [] });
    fireEvent.click(screen.getByRole("button", { name: "Reload documents" }));
    expect(await screen.findByText("No matching documents")).toBeTruthy();
  });
});
