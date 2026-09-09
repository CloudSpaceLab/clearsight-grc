import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiError, requestJSON, requestVoid } from "./http";

afterEach(() => vi.unstubAllGlobals());

describe("request recovery messages", () => {
  it("does not claim a write failed when the connection drops", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new TypeError("Failed to fetch")));
    await expect(requestVoid("", "/review", { method: "POST" })).rejects.toMatchObject({
      code: "connection_lost", kind: "unavailable", message: "Connection lost. Check the record before trying again.",
    });
  });
  it("keeps a domain conflict message and code", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: "assessment_changed", message: "Assessment changed. Reload before saving." } }), { status: 409 })));
    await expect(requestJSON("", "/review")).rejects.toMatchObject({ code: "assessment_changed", kind: "conflict", message: "Assessment changed. Reload before saving." });
  });
  it("reads the flat error envelope emitted by the API", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: "form_proposal_source_changed", message: "The source document changed. Reload the import and create a new proposal." }), { status: 409 })));
    await expect(requestJSON("", "/proposal")).rejects.toMatchObject({ code: "form_proposal_source_changed", kind: "conflict", message: "The source document changed. Reload the import and create a new proposal." });
  });
  it("gives recovery for a non-JSON service failure", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("upstream error", { status: 503 })));
    await expect(requestJSON("", "/review")).rejects.toMatchObject({ message: "Service unavailable. Try again." });
  });
  it("preserves cancellation without presenting a connection failure", async () => {
    const abort = new DOMException("Aborted", "AbortError");
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(abort));
    await expect(requestJSON("", "/review")).rejects.toBe(abort);
    expect(abort).not.toBeInstanceOf(ApiError);
  });
});
