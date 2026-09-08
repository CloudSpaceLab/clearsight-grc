import { afterEach, describe, expect, it, vi } from "vitest";
import { loadResponseAssessment, recordResponseAssessment } from "./formAssessmentApi";

afterEach(() => vi.unstubAllGlobals());
describe("bank assessment requests", () => {
  it("reads an encoded exact response and writes only the approved decision contract", async () => {
    const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ response_id: "response/a", version: 3 }) });
    vi.stubGlobal("fetch", fetcher);
    await loadResponseAssessment("response/a");
    expect(fetcher.mock.calls[0]![0]).toBe("/api/v1/forms/responses/response%2Fa/assessment");
    await recordResponseAssessment("response/a", { expected_version: 3, decisions: [{ field_id: "proof", outcome_id: "gap", rationale: "Scope missing" }] });
    const init = fetcher.mock.calls[1]![1];
    expect(init.credentials).toBe("include");
    expect(JSON.parse(init.body)).toEqual({ expected_version: 3, decisions: [{ field_id: "proof", outcome_id: "gap", rationale: "Scope missing" }] });
  });
});
