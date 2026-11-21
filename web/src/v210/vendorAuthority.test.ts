import { expect, it, vi } from "vitest";
import { checkVendorAuthority } from "./vendorAuthority";
const api = vi.hoisted(() => ({ loadContext: vi.fn() }));
const http = vi.hoisted(() => ({ requestJSON: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../http", () => http);

it("checks the signed-in reviewer against the exact assessment route", async () => {
  api.loadContext.mockResolvedValue({ tenant: { id: "t" }, legal_entity: { id: "e" }, actor: { id: "reviewer", kind: "HUMAN" } });
  http.requestJSON.mockResolvedValue({ principal: { id: "other" }, candidate_principals: [{ id: "reviewer" }] });
  expect(await checkVendorAuthority("review", "assessment-1")).toBe(true);
  const request = http.requestJSON.mock.calls[0]?.[2];
  expect(request).toBeDefined();
  expect(JSON.parse(request.body)).toEqual({ tenant_id: "t", legal_entity_id: "e", object_type: "THIRD_PARTY_ASSESSMENT", object_id: "assessment-1", responsibility: "REVIEWER", decision_type: "thirdparty.assessment.review", materiality: 3 });
});

it("does not authorize a principal outside the current route", async () => {
  api.loadContext.mockResolvedValue({ tenant: { id: "t" }, legal_entity: { id: "e" }, actor: { id: "other", kind: "HUMAN" } });
  http.requestJSON.mockResolvedValue({ principal: { id: "reviewer" }, candidate_principals: [] });
  expect(await checkVendorAuthority("review", "assessment-1")).toBe(false);
});

it("requires both HTTP and service authority routes to start an assessment", async () => {
  api.loadContext.mockResolvedValue({ tenant: { id: "t" }, legal_entity: { id: "e" }, actor: { id: "person", kind: "HUMAN" } });
  http.requestJSON.mockReset().mockResolvedValueOnce({ principal: { id: "person" } }).mockResolvedValueOnce({ principal: { id: "other" } });
  expect(await checkVendorAuthority("start", "rel")).toBe(false);
  expect(http.requestJSON.mock.calls.map((call) => JSON.parse(call[2].body).object_type)).toEqual(["VENDOR_RELATIONSHIP", "THIRD_PARTY_RELATIONSHIP"]);
});

it.each([
  ["send", "ACCOUNTABLE_OWNER", "thirdparty.assessment.send_request"],
  ["complete", "REVIEWER", "thirdparty.assessment.complete"],
  ["document", "REVIEWER", "thirdparty.assessment.document.review"],
  ["setup", "ACCOUNTABLE_OWNER", "thirdparty.assessment.setup.retry"],
  ["reissue", "ACCOUNTABLE_OWNER", "thirdparty.assessment.reissue_request"],
  ["clarification", "REVIEWER", "thirdparty.assessment.clarification.request"],
] as const)("checks the exact %s command policy", async (action, responsibility, decision_type) => {
  api.loadContext.mockResolvedValue({ tenant: { id: "t" }, legal_entity: { id: "e" }, actor: { id: "person", kind: "HUMAN" } });
  http.requestJSON.mockReset().mockResolvedValue({ principal: { id: "person" } });
  expect(await checkVendorAuthority(action, "a")).toBe(true);
  expect(JSON.parse(http.requestJSON.mock.calls[0]?.[2].body)).toMatchObject({ object_type: "THIRD_PARTY_ASSESSMENT", object_id: "a", responsibility, decision_type, materiality: 3 });
});

it("checks the activation authorizer route at materiality four", async () => {
  api.loadContext.mockResolvedValue({ tenant: { id: "t" }, legal_entity: { id: "e" }, actor: { id: "person", kind: "HUMAN" } });
  http.requestJSON.mockReset().mockResolvedValue({ principal: { id: "person" } });
  expect(await checkVendorAuthority("activate", "rel")).toBe(true);
  expect(JSON.parse(http.requestJSON.mock.calls[0]?.[2].body)).toMatchObject({ object_type: "VENDOR_RELATIONSHIP", object_id: "rel", responsibility: "AUTHORIZER", decision_type: "thirdparty.relationship.activate", materiality: 4 });
});
