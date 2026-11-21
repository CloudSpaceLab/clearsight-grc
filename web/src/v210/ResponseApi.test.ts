import { expect, it, vi } from "vitest";
import { loadCompletedResponse } from "../formsDistributionApi";
import { uploadInternalDocument } from "./internalResponseApi";

it("retains the authorized submitted answers returned by the exact response endpoint", async () => {
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ response: { id: "revision-1" }, revision: { id: "revision-1" }, submission_id: "submission-1", answers: [{ field_id: "q1", label: "Is access reviewed?", type: "yes_no", value: { text: "Yes" } }], documents: [], documents_next_cursor: "next" }), { status: 200, headers: { "Content-Type": "application/json" } }));
  try { const result = await loadCompletedResponse("revision-1"); expect(result.answers?.[0]?.label).toBe("Is access reviewed?"); expect(result.answers?.[0]?.value.text).toBe("Yes"); expect(result.documents_next_cursor).toBe("next"); }
  finally { fetchMock.mockRestore(); }
});

it("uploads an internal document to the exact workspace without identity fields", async () => {
  const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ id: "artifact", request_id: "req/1" }), { status: 201, headers: { "Content-Type": "application/json" } }));
  try {
    const file = new File(["document"], "report.txt", { type: "text/plain" });
    await uploadInternalDocument("req/1", file, "field-1");
    const [url, init] = fetchMock.mock.calls[0]!;
    expect(url).toBe("/api/v1/forms/requests/req%2F1/workspace/documents");
    expect(init?.credentials).toBe("include");
    const body = init?.body as FormData;
    expect([...body.keys()].sort()).toEqual(["field_id", "file"]);
    expect(body.get("field_id")).toBe("field-1");
  } finally { fetchMock.mockRestore(); }
});
