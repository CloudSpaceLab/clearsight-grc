import { afterEach, describe, expect, it, vi } from "vitest";
import { createVendorRelationship, loadVendorIdentity, loadVendorRelationships, removeApprovedVendorLogo, updateVendorIdentity, updateVendorRelationship, uploadApprovedVendorLogo, vendorBrandURL } from "./vendorApi";

const aggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Payments", business_owner_principal_id: "owner", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
};

describe("vendor API", () => {
  afterEach(() => { vi.unstubAllGlobals(); vi.unstubAllEnvs(); });

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
    await createVendorRelationship({ legal_name: "Acme", website_domain: "acme.example", registered_address: "1 Marina Road\nLagos", service_name: "Payments", criticality: "IMPORTANT", privacy_role: "PROCESSOR" });
    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    const init = call[1] as RequestInit;
    expect(call[0]).toBe("/api/v1/vendors");
    expect(JSON.parse(String(init.body))).toEqual({ legal_name: "Acme", website_domain: "acme.example", registered_address: "1 Marina Road\nLagos", service_name: "Payments", criticality: "IMPORTANT", privacy_role: "PROCESSOR" });
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

  it("loads and updates the exact shared vendor identity", async () => {
    const identity = { vendor: { ...aggregate.vendor, website_domain: "acme.example", version: 4 }, brand: { state: "PENDING", version: 2, event_version: 2 } };
    const fetchMock = vi.fn().mockImplementation(async () => new Response(JSON.stringify(identity), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await loadVendorIdentity("vendor/1");
    await updateVendorIdentity("vendor/1", { expected_version: 4, legal_name: "Acme", website_domain: "acme.example" });

    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/v1/vendor-identities/vendor%2F1");
    expect(fetchMock.mock.calls[1]?.[0]).toBe("/api/v1/vendor-identities/vendor%2F1");
    expect(fetchMock.mock.calls[1]?.[1]).toEqual(expect.objectContaining({ method: "PUT" }));
    expect(JSON.parse(String((fetchMock.mock.calls[1]?.[1] as RequestInit).body))).toEqual({ expected_version: 4, legal_name: "Acme", website_domain: "acme.example" });
  });

  it("uploads and removes an approved logo with the independent brand version", async () => {
    const identity = { vendor: aggregate.vendor, brand: { state: "APPROVED_LOGO", version: 3, event_version: 3 } };
    const fetchMock = vi.fn().mockImplementation(async () => new Response(JSON.stringify(identity), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const file = new File([new Uint8Array([137, 80, 78, 71])], "logo.png", { type: "image/png" });

    await uploadApprovedVendorLogo("vendor-1", file, 2);
    await removeApprovedVendorLogo("vendor-1", 3);

    expect(fetchMock.mock.calls[0]?.[0]).toBe("/api/v1/vendor-identities/vendor-1/brand");
    expect(fetchMock.mock.calls[0]?.[1]).toEqual(expect.objectContaining({ method: "PUT", body: file }));
    expect(new Headers((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).get("Content-Type")).toBe("image/png");
    expect(new Headers((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).get("If-Match")).toBe("\"2\"");
    expect(new Headers((fetchMock.mock.calls[0]?.[1] as RequestInit).headers).get("Idempotency-Key")).toMatch(/^vendor-brand-/);
    expect(fetchMock.mock.calls[1]?.[1]).toEqual(expect.objectContaining({ method: "DELETE" }));
    expect(new Headers((fetchMock.mock.calls[1]?.[1] as RequestInit).headers).get("If-Match")).toBe("\"3\"");
    expect(new Headers((fetchMock.mock.calls[1]?.[1] as RequestInit).headers).get("Idempotency-Key")).toMatch(/^vendor-brand-/);
  });

  it("constructs only a same-origin versioned brand URL", () => {
    expect(vendorBrandURL("vendor/1", "opaque/token?value")).toBe("/api/v1/vendor-identities/vendor%2F1/brand?version=opaque%2Ftoken%3Fvalue");
    expect(vendorBrandURL("vendor-1", "https://vendor.example/logo.png")).toBeUndefined();
    expect(vendorBrandURL("vendor-1", "//vendor.example/logo.png")).toBeUndefined();
    expect(vendorBrandURL("vendor-1", "")).toBeUndefined();
  });

  it("keeps brand images on the current origin when the JSON API uses a configured base", async () => {
    vi.stubEnv("VITE_API_BASE_URL", "https://api.vendor-platform.example");
    vi.resetModules();
    const configured = await import("./vendorApi");
    expect(configured.vendorBrandURL("vendor/1", "opaque-token")).toBe("/api/v1/vendor-identities/vendor%2F1/brand?version=opaque-token");
  });

  it("infers a supported media type when the browser leaves the file type empty", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => new Response(JSON.stringify({ vendor: aggregate.vendor, brand: { state: "APPROVED_LOGO", version: 1, event_version: 1 } }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    for (const [name, mediaType] of [["approved.png", "image/png"], ["approved.jpg", "image/jpeg"], ["approved.webp", "image/webp"], ["approved.ico", "image/x-icon"]] as const) {
      await uploadApprovedVendorLogo("vendor-1", new File([new Uint8Array([0, 0, 1, 0])], name), 0, `vendor-brand-${name}`);
      const call = fetchMock.mock.calls.at(-1);
      expect(new Headers((call?.[1] as RequestInit).headers).get("Content-Type")).toBe(mediaType);
    }
  });
});
