import { afterEach, describe, expect, it, vi } from "vitest";
import { installVendorCollectionEvidence } from "./vendorCollectionEvidence";

afterEach(() => { vi.unstubAllGlobals(); window.history.replaceState(null, "", "/"); });

describe("unscanned collection evidence fixture", () => {
  it.each(["vendor-collection-unscanned-allowed", "vendor-collection-unscanned-blocked"])("keeps the real scan status in %s", async (fixture) => {
    window.history.replaceState(null, "", `/?fixture=${fixture}`);
    vi.stubGlobal("fetch", vi.fn());
    installVendorCollectionEvidence();
    const allowed = fixture.endsWith("allowed");
    const inventory = await (await fetch("/api/v1/forms/documents")).json();
    expect(inventory.items[0]).toMatchObject({ artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: allowed });
    const review = await (await fetch("/api/v1/vendor-assessments/sample-collection-assessment")).json();
    expect(review.documents.find((file: { field_id: string }) => file.field_id === "iso")).toMatchObject({ artifact_status: "STORED_UNSCANNED", demo_unscanned_allowed: allowed });
    const collection = await (await fetch("/api/v1/vendor-assessments/sample-collection-assessment/collection")).json();
    expect(collection.fields.find((field: { field_id: string }) => field.field_id === "iso")).toMatchObject({ collection_state: allowed ? "REUSED" : "MISSING", vendor_action_required: !allowed });
  });
});
