import { afterEach, describe, expect, it, vi } from "vitest";
import { documentContentURL, loadDocuments, previewKind } from "./submittedDocumentApi";

afterEach(() => vi.unstubAllGlobals());
describe("submitted document access", () => {
  it("sends bounded file-type filters and exact vendor scope to the server", async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [], next_cursor: "next" })));
    vi.stubGlobal("fetch", fetcher);
    expect(await loadDocuments({ file_kind: "PDF", query: "annual report", relationship_id: "vendor/1", current_only: false })).toEqual({ items: [], next_cursor: "next" });
    const url = new URL(fetcher.mock.calls[0]![0], "https://example.test");
    expect(url.pathname).toBe("/api/v1/forms/documents");
    expect(Object.fromEntries(url.searchParams)).toEqual({ file_kind: "PDF", query: "annual report", relationship_id: "vendor/1", current_only: "false", limit: "25" });
    expect(fetcher.mock.calls[0]![1].credentials).toBe("include");
  });
  it("encodes the exact document occurrence in protected content links", () => {
    expect(documentContentURL({ submission_id: "s/1", field_id: "q?2", artifact_id: "a#3", response_revision_id: "r/4" }, true))
      .toBe("/api/v1/forms/documents/s%2F1/q%3F2/a%233/content?response_revision_id=r%2F4&download=true");
  });
  it("previews only supported available media and never trusts a file extension", () => {
    expect(previewKind({ media_type: "application/pdf", artifact_status: "AVAILABLE" })).toBe("pdf");
    expect(previewKind({ media_type: "image/png", artifact_status: "AVAILABLE" })).toBe("image");
    for (const media_type of ["text/html", "image/svg+xml", "application/msword", "application/octet-stream"]) {
      expect(previewKind({ media_type, artifact_status: "AVAILABLE" })).toBeUndefined();
    }
    expect(previewKind({ media_type: "application/pdf", artifact_status: "QUARANTINED" })).toBeUndefined();
  });
  it("previews server-eligible unscanned samples but rejects blocked states even with a forged flag", () => {
    expect(previewKind({ media_type: "application/pdf", artifact_status: "STORED_UNSCANNED", demo_preview_available: true })).toBe("pdf");
    for (const artifact_status of ["QUARANTINED", "DELETED", "UNKNOWN"]) {
      expect(previewKind({ media_type: "image/png", artifact_status, demo_preview_available: true })).toBeUndefined();
    }
    expect(previewKind({ media_type: "image/png", artifact_status: "STORED_UNSCANNED" })).toBeUndefined();
  });
  it("previews explicitly allowed unscanned demo files without a sample-preview capability", () => {
    expect(previewKind({ media_type: "image/png", artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: true })).toBe("image");
    for (const artifact_status of ["QUARANTINED", "DELETED", "UNKNOWN"]) {
      expect(previewKind({ media_type: "image/png", artifact_status, demo_unscanned_allowed: true })).toBeUndefined();
    }
    expect(previewKind({ media_type: "image/png", artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: false })).toBeUndefined();
  });
});
