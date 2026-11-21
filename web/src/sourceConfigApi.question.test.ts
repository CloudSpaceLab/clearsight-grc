import { beforeEach, expect, it, vi } from "vitest";
import { createRESTBinding, prepareRESTSource } from "./sourceConfigApi";

const mock = vi.hoisted(() => ({ request: vi.fn(), context: vi.fn() }));
vi.mock("./api", () => ({ loadContext: mock.context }));
vi.mock("./http", () => ({ requestJSON: mock.request }));

beforeEach(() => {
  vi.clearAllMocks();
  mock.context.mockResolvedValue({ tenant: { id: "bank" }, legal_entity: { id: "entity" }, actor: { id: "owner" } });
});

it("creates a typed singleton draft and binds its result without a fake stable key", async () => {
  mock.request.mockResolvedValueOnce({ id: "source", code: "CHECK", name: "Check", expected_freshness_minutes: 60 })
    .mockResolvedValueOnce({ connection_id: "connection", version: 1 })
    .mockResolvedValueOnce({ view_id: "view", version: 1 })
    .mockResolvedValueOnce({ view: { view_id: "view", version: 2, native_schema: [{ name: "result", native_type: "json:boolean", nullable: false }] } })
    .mockResolvedValueOnce({ binding_id: "binding", status: "DRAFT" });
  const prepared = await prepareRESTSource({ code: "CHECK", name: "Check", endpoint: "https://bank.example/status", freshnessMinutes: 60, resultType: "BOOLEAN" });
  const viewRequest = JSON.parse(mock.request.mock.calls[2]![2].body);
  expect(viewRequest.definition).toMatchObject({ singleton: true, result_type: "BOOLEAN", path: "/status" });
  await createRESTBinding(prepared, "result");
  expect(mock.request).toHaveBeenCalledTimes(5);
  const bindingRequest = JSON.parse(mock.request.mock.calls[4]![2].body);
  expect(bindingRequest).toMatchObject({ view_version: 2, operations: ["PAGE"], selected_fields: ["result"], key_fields: [] });
});

it("rejects selecting a different field for the standard result contract", async () => {
  const prepared = { source: { code: "CHECK", name: "Check", expected_freshness_minutes: 60 }, view: { view_id: "view", version: 1 }, resultType: "BOOLEAN" } as Parameters<typeof createRESTBinding>[0];
  await expect(createRESTBinding(prepared, "message")).rejects.toThrow(/result/);
  expect(mock.request).not.toHaveBeenCalled();
});

it("rejects credentials embedded in an endpoint before writing a draft", async () => {
  await expect(prepareRESTSource({ code: "CHECK", name: "Check", endpoint: "https://user:secret@bank.example/status", freshnessMinutes: 60, resultType: "BOOLEAN" })).rejects.toThrow(/credentials/i);
  expect(mock.request).not.toHaveBeenCalled();
});

it("rejects query parameters and fragments for API questions before writing anything", async () => {
  for (const endpoint of ["https://bank.example/status?api_key=secret", "https://bank.example/status#token"]) {
    await expect(prepareRESTSource({ code: "CHECK", name: "Check", endpoint, freshnessMinutes: 60, resultType: "BOOLEAN" })).rejects.toThrow(/query|fragment/i);
  }
  expect(mock.request).not.toHaveBeenCalled();
});

it("does not persist an unnecessary reference connection with the full endpoint", async () => {
  mock.request.mockResolvedValueOnce({ id: "source", code: "CHECK", name: "Check", expected_freshness_minutes: 60 })
    .mockResolvedValueOnce({ connection_id: "connection", version: 1 })
    .mockResolvedValueOnce({ view_id: "view", version: 1 })
    .mockResolvedValueOnce({ view: { view_id: "view", version: 2 } });
  await prepareRESTSource({ code: "CHECK", name: "Check", endpoint: "https://bank.example/status", freshnessMinutes: 60, resultType: "BOOLEAN" });
  expect(JSON.parse(mock.request.mock.calls[0]![2].body)).not.toHaveProperty("endpoint");
});

it("preserves committed draft identifiers when setup fails", async () => {
  mock.request.mockResolvedValueOnce({ id: "source", code: "CHECK", name: "Check", expected_freshness_minutes: 60 })
    .mockResolvedValueOnce({ connection_id: "connection", version: 1 })
    .mockRejectedValueOnce(new Error("network failure and sensitive endpoint"));
  try {
    await prepareRESTSource({ code: "CHECK", name: "Check", endpoint: "https://bank.example/status", freshnessMinutes: 60, resultType: "BOOLEAN" });
    throw new Error("expected failure");
  } catch (error) {
    expect(error).toMatchObject({ phase: "view", committed: { source: { id: "source" }, connection: { connection_id: "connection" } } });
    expect((error as Error).message).not.toContain("sensitive endpoint");
  }
});
