import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { DocumentOccurrence } from "../../submittedDocumentApi";
import { DocumentPreview } from "./DocumentPreview";

const imageFile: DocumentOccurrence = { id: "image", artifact_id: "a", request_id: "r", submission_id: "s", field_id: "f", form_template_version: 1,
  form_title: "Vendor review", field_label: "Certificate", file_name: "certificate.png", media_type: "image/png", file_kind: "IMAGE", size_bytes: 5,
  sha256: "sample", artifact_status: "AVAILABLE", uploaded_at: "2026-09-01", submitted_at: "2026-09-02", current: true };
afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });
describe("protected document preview", () => {
  it("offers download instead of a blank frame when PDF viewing is disabled", async () => {
    const prior = Object.getOwnPropertyDescriptor(navigator, "pdfViewerEnabled");
    Object.defineProperty(navigator, "pdfViewerEnabled", { configurable: true, value: false });
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("pdf", { headers: { "Content-Type": "application/pdf" } })));
    try {
      render(<DocumentPreview file={{ ...imageFile, file_name: "certificate.pdf", media_type: "application/pdf", file_kind: "PDF" }} onClose={() => undefined}/>);
      expect(screen.getByText("This browser cannot preview PDFs. Download the file to view it in a PDF application.")).toBeTruthy();
      expect(document.querySelector("iframe")).toBeNull();
      expect(fetch).not.toHaveBeenCalled();
    } finally { if (prior) Object.defineProperty(navigator, "pdfViewerEnabled", prior); else Reflect.deleteProperty(navigator, "pdfViewerEnabled"); }
  });
  it("uses the existing centered Quick Look dialog contract", () => {
    render(<DocumentPreview file={{ ...imageFile, artifact_status: "QUARANTINED" }} onClose={() => undefined}/>);
    expect(screen.getByRole("dialog").closest(".cs-dialog--wide")).not.toBeNull();
  });
  it("does not display a truncated file as a successful preview", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("x", { headers: { "Content-Type": "image/png" } })));
    URL.createObjectURL = vi.fn().mockReturnValue("blob:truncated");
    URL.revokeObjectURL = vi.fn();
    render(<DocumentPreview file={imageFile} onClose={() => undefined}/>);
    expect(await screen.findByRole("button", { name: "Retry preview" })).toBeTruthy();
    expect(URL.createObjectURL).not.toHaveBeenCalled();
  });
  it("loads protected image bytes and releases its temporary URL when closed", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("image", { headers: { "Content-Type": "image/png" } })));
    URL.createObjectURL = vi.fn().mockReturnValue("blob:test-preview");
    URL.revokeObjectURL = vi.fn();
    const view = render(<DocumentPreview file={imageFile} onClose={() => undefined}/>);
    expect(await screen.findByRole("img", { name: "Submitted document: certificate.png" })).toBeTruthy();
    expect(fetch).toHaveBeenCalledWith("/api/v1/forms/documents/s/f/a/content", expect.objectContaining({ credentials: "include", cache: "no-store" }));
    view.unmount();
    expect(URL.revokeObjectURL).toHaveBeenCalledWith("blob:test-preview");
  });
  it("rejects a content-type mismatch and allows a fresh retry", async () => {
    const fetcher = vi.fn().mockResolvedValueOnce(new Response("<script>bad</script>", { headers: { "Content-Type": "text/html" } }))
      .mockResolvedValueOnce(new Response("image", { headers: { "Content-Type": "image/png" } }));
    vi.stubGlobal("fetch", fetcher);
    URL.createObjectURL = vi.fn().mockReturnValue("blob:retry-preview");
    URL.revokeObjectURL = vi.fn();
    render(<DocumentPreview file={imageFile} onClose={() => undefined}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Retry preview" }));
    expect(await screen.findByRole("img")).toBeTruthy();
    expect(URL.createObjectURL).toHaveBeenCalledTimes(1);
  });
  it("does not allocate a preview URL after an in-flight preview is closed", async () => {
    let resolve!: (value: Response) => void;
    vi.stubGlobal("fetch", vi.fn().mockReturnValue(new Promise<Response>((done) => { resolve = done; })));
    URL.createObjectURL = vi.fn(); URL.revokeObjectURL = vi.fn();
    const view = render(<DocumentPreview file={imageFile} onClose={() => undefined}/>);
    view.unmount(); resolve(new Response("image", { headers: { "Content-Type": "image/png" } }));
    await waitFor(() => expect((fetch as ReturnType<typeof vi.fn>).mock.calls[0]![1].signal.aborted).toBe(true));
    expect(URL.createObjectURL).not.toHaveBeenCalled();
  });
});
