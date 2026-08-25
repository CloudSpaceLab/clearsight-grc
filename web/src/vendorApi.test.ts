import { afterEach, describe, expect, it, vi } from "vitest";
import { createVendorRelationship, loadVendorRelationships, updateVendorRelationship } from "./vendorApi";

const aggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Payments", business_owner_principal_id: "owner", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
};

describe("vendor API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("loads a bounded vendor relationship search", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [aggregate] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await loadVendorRelationships({ search: "payment services", limit: 25 });
    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendors?search=payment+services&limit=25");
  });

  it("does not send browser identity fields when creating a relationship", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(aggregate), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);
    await createVendorRelationship({ legal_name: "Acme", service_name: "Payments", criticality: "IMPORTANT", privacy_role: "PROCESSOR" });
    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    const init = call[1] as RequestInit;
    expect(call[0]).toBe("/api/v1/vendors");
    expect(JSON.parse(String(init.body))).toEqual({ legal_name: "Acme", service_name: "Payments", criticality: "IMPORTANT", privacy_role: "PROCESSOR" });
  });

  it("uses the route id and expected version for updates", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(aggregate), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    await updateVendorRelationship("relationship/1", { expected_version: 4, service_name: "Payments", criticality: "CRITICAL", privacy_role: "PROCESSOR" });
    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendors/relationship%2F1");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({
      expected_version: 4, service_name: "Payments", criticality: "CRITICAL", privacy_role: "PROCESSOR",
    });
  });
});
