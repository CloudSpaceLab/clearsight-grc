import { beforeEach, expect, it, vi } from "vitest";
import { requestJSON } from "./http";
import { loadEvidenceSources } from "./api";

vi.mock("./http", () => ({ requestJSON: vi.fn(), requestVoid: vi.fn() }));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(requestJSON)
    .mockResolvedValueOnce({ tenant: { id: "bank" }, legal_entity: { id: "*" }, actor: { id: "admin" }, mode: "memory" })
    .mockResolvedValueOnce({ items: [] });
});

it("binds evidence source reads to the record's exact legal entity", async () => {
  await loadEvidenceSources("entity/one");
  expect(vi.mocked(requestJSON).mock.calls[1]?.[1]).toBe("/api/v1/evidence/sources?tenant_id=bank&legal_entity_id=entity%2Fone&limit=50");
});
