import { afterEach, describe, expect, it, vi } from "vitest";
import { endVendorRelationshipLink, linkVendorRelationship, loadVendorRelationshipLinks } from "./vendorLinkApi";

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

  it("ends a link with its current version and reason", async () => {
    const ended = { id: "link-1", relationship_id: "relationship-1", target_type: "PROGRAM", target_id: "program-1", purpose_code: "SERVICE_SUPPORT", purpose_label: "Service support", state: "ENDED", version: 2 };
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify(ended), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await endVendorRelationshipLink("relationship/1", "link/1", { expected_version: 1, reason: "The vendor no longer supports this Program." });

    const call = fetchMock.mock.calls[0];
    if (!call) throw new Error("fetch was not called");
    expect(call[0]).toBe("/api/v1/vendors/relationship%2F1/links/link%2F1/end");
    expect(JSON.parse(String((call[1] as RequestInit).body))).toEqual({ expected_version: 1, reason: "The vendor no longer supports this Program." });
  });
});
