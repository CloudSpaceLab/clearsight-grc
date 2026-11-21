import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Documents } from "../../v210/Documents";
import type { DocumentOccurrence } from "../../submittedDocumentApi";

const inventory = vi.hoisted(() => ({ loadDocuments: vi.fn() }));
vi.mock("../../submittedDocumentApi", async (original) => ({ ...await original<typeof import("../../submittedDocumentApi")>(), loadDocuments: inventory.loadDocuments }));
const imageFile: DocumentOccurrence = { id: "image", artifact_id: "a", request_id: "r", submission_id: "s", field_id: "f", form_template_version: 1, form_title: "Vendor review", field_label: "Certificate", file_name: "certificate.png", media_type: "image/png", file_kind: "IMAGE", size_bytes: 5, sha256: "sample", artifact_status: "AVAILABLE", uploaded_at: "2026-09-01", submitted_at: "2026-09-02", current: true };

beforeEach(() => { window.history.replaceState(null, "", "#/documents"); inventory.loadDocuments.mockReset(); inventory.loadDocuments.mockResolvedValue({ items: [imageFile] }); });
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });
async function open() { fireEvent.click(await screen.findByRole("button", { name: "Open certificate.png" })); }

describe("protected Documents preview", () => {
  it("uses protected, no-store fetch and revokes the object URL on close", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response("image", { headers: { "Content-Type": "image/png" } }));
    vi.stubGlobal("fetch", fetcher);
    const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:protected");
    const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
    render(<Documents/>); await open();
    expect(await screen.findByRole("img", { name: "Preview: certificate.png" })).toBeTruthy();
    expect(fetcher).toHaveBeenCalledWith("/api/v1/forms/documents/s/f/a/content", expect.objectContaining({ credentials: "include", cache: "no-store", signal: expect.any(AbortSignal) }));
    fireEvent.click(screen.getByRole("button", { name: "Close document details" }));
    expect(revoke).toHaveBeenCalledWith("blob:protected");
    expect(create).toHaveBeenCalledTimes(1);
  });
  it.each([["wrong type", "wrong", "text/html"], ["truncated content", "bad", "image/png"]])("rejects %s and allows a fresh retry", async (_name, content, type) => {
    const fetcher = vi.fn().mockResolvedValueOnce(new Response(content, { headers: { "Content-Type": type } })).mockResolvedValueOnce(new Response("image", { headers: { "Content-Type": "image/png" } }));
    vi.stubGlobal("fetch", fetcher);
    const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:retry");
    vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
    render(<Documents/>); await open();
    expect(await screen.findByText("Preview unavailable. Try again.")).toBeTruthy();
    expect(create).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Preview document" }));
    expect(await screen.findByRole("img", { name: "Preview: certificate.png" })).toBeTruthy();
    expect(create).toHaveBeenCalledTimes(1);
  });
  it("rejects invalid present size metadata instead of treating it as unknown", async () => {
    inventory.loadDocuments.mockResolvedValue({ items: [{ ...imageFile, size_bytes: -1 }] });
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response("image", { headers: { "Content-Type": "image/png" } })));
    const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:invalid");
    render(<Documents/>); await open();
    expect(await screen.findByText("Preview unavailable. Try again.")).toBeTruthy();
    expect(create).not.toHaveBeenCalled();
  });
  it.each(["QUARANTINED", "DELETED", "UNKNOWN", "STORED_UNSCANNED"])("does not fetch or offer download for %s without demo permission", async (status) => {
    inventory.loadDocuments.mockResolvedValue({ items: [{ ...imageFile, artifact_status: status, demo_preview_available: status !== "STORED_UNSCANNED", demo_unscanned_allowed: status !== "STORED_UNSCANNED" }] });
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    render(<Documents/>); await open();
    expect(fetcher).not.toHaveBeenCalled();
    expect(screen.queryByRole("button", { name: "Download" })).toBeNull();
    expect(screen.queryByRole("img", { name: "Preview: certificate.png" })).toBeNull();
  });
  it.each([false, true])("shows the demo warning and allows a preview with review permission flag %s", async (allowed) => {
    inventory.loadDocuments.mockResolvedValue({ items: [{ ...imageFile, artifact_status: "STORED_UNSCANNED", demo_preview_available: true, demo_unscanned_allowed: allowed }] });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("image", { headers: { "Content-Type": "image/png" } })));
    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:demo"); vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
    render(<Documents/>); await open();
    expect(screen.getAllByText(/No antivirus scan was performed/).length).toBeGreaterThan(0);
    expect(await screen.findByRole("img", { name: "Preview: certificate.png" })).toBeTruthy();
    expect(Boolean(screen.queryByRole("button", { name: "Download" }))).toBe(allowed);
  });
  it.each([false, true])("explains unsupported demo document previews with download permission %s", async (allowed) => {
    inventory.loadDocuments.mockResolvedValue({ items: [{ ...imageFile, file_name: "register.xlsx", media_type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", file_kind: "SPREADSHEET", artifact_status: "STORED_UNSCANNED", demo_preview_available: true, demo_unscanned_allowed: allowed }] });
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    render(<Documents/>); fireEvent.click(await screen.findByRole("button", { name: "Open register.xlsx" }));
    expect(screen.getByText(allowed ? "Preview unavailable for this file type. Download the document." : "Preview unavailable for this file type.")).toBeTruthy();
    expect(fetcher).not.toHaveBeenCalled();
    expect(Boolean(screen.queryByRole("button", { name: "Download" }))).toBe(allowed);
  });
  it("offers protected PDF download without preview when the browser cannot display PDFs", async () => {
    const descriptor = Object.getOwnPropertyDescriptor(navigator, "pdfViewerEnabled");
    Object.defineProperty(navigator, "pdfViewerEnabled", { configurable: true, value: false });
    inventory.loadDocuments.mockResolvedValue({ items: [{ ...imageFile, file_name: "certificate.pdf", media_type: "application/pdf", file_kind: "PDF" }] });
    const fetcher = vi.fn(); vi.stubGlobal("fetch", fetcher);
    try {
      render(<Documents/>); fireEvent.click(await screen.findByRole("button", { name: "Open certificate.pdf" }));
      expect(screen.getByText("PDF preview unavailable in this browser. Download the document.")).toBeTruthy();
      expect(screen.getByRole("button", { name: "Download certificate.pdf" })).toBeTruthy();
      expect(screen.queryByTitle("Preview: certificate.pdf")).toBeNull(); expect(fetcher).not.toHaveBeenCalled();
    } finally { if (descriptor) Object.defineProperty(navigator, "pdfViewerEnabled", descriptor); else Reflect.deleteProperty(navigator, "pdfViewerEnabled"); }
  });
  it("aborts a closed preview and does not allocate a late object URL", async () => {
    let finish!: (value: Response) => void;
    const fetcher = vi.fn().mockReturnValue(new Promise<Response>((resolve) => { finish = resolve; })); vi.stubGlobal("fetch", fetcher);
    const create = vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:late");
    render(<Documents/>); await open();
    fireEvent.click(screen.getByRole("button", { name: "Close document details" }));
    finish(new Response("image", { headers: { "Content-Type": "image/png" } }));
    await waitFor(() => expect(fetcher.mock.calls[0]![1].signal.aborted).toBe(true));
    expect(create).not.toHaveBeenCalled();
  });
  it("revokes an image the browser cannot decode and permits retry", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response("image", { headers: { "Content-Type": "image/png" } })));
    vi.spyOn(URL, "createObjectURL").mockReturnValueOnce("blob:bad-image").mockReturnValueOnce("blob:retry-image");
    const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
    render(<Documents/>); await open();
    fireEvent.error(await screen.findByRole("img", { name: "Preview: certificate.png" }));
    expect(revoke).toHaveBeenCalledWith("blob:bad-image");
    expect(screen.getByText("Preview unavailable. Try again.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Preview document" }));
    expect((await screen.findByRole("img", { name: "Preview: certificate.png" })).getAttribute("src")).toBe("blob:retry-image");
  });
});
