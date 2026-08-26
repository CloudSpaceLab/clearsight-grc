import { afterEach, describe, expect, it, vi } from "vitest";
import { linkVendorRelationship, loadVendorRelationshipLinks } from "./vendorLinkApi";

describe("vendor relationship link API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("loads a bounded target-scoped link page", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await loadVendorRelationshipLinks({ target_type: "PROGRAM", target_id: "program/1", cursor: "next page", limit: 25 });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/vendor-links?target_type=PROGRAM&target_id=program%2F1&cursor=next+page&limit=25",
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("links the selected relationship without browser identity fields", async () => {
    const link = {
      id: "link-1", relationship_id: "relationship-1", target_type: "MATTER", target_id: "matter-1",
      purpose_code: "REMEDIATION_SUPPORT", purpose_label: "Supports remediation", state: "ACTIVE", version: 1,
    };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(link), { status: 201 }));
    vi.stubGlobal("fetch", fetchMock);

    await linkVendorRelationship("relationship/1", {
      target_type: "MATTER", target_id: "matter-1", purpose_code: "REMEDIATION_SUPPORT", purpose_label: "Supports remediation",
    });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendors/relationship%2F1/links");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({
      target_type: "MATTER", target_id: "matter-1", purpose_code: "REMEDIATION_SUPPORT", purpose_label: "Supports remediation",
    });
  });
});
