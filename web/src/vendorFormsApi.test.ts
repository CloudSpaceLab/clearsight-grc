import { afterEach, expect, it, vi } from "vitest";
import { loadVendorForms, loadVendorFormSummaries, requestVendorForms } from "./vendorFormsApi";
afterEach(() => vi.unstubAllGlobals());
it("uses exact vendor scope and bounded summary groups", async () => {
  const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ items: [] }) }); vi.stubGlobal("fetch", fetcher);
  await loadVendorForms("vendor/a", { filter: "AWAITING_REVIEW", cursor: "next" });
  expect(fetcher.mock.calls[0]![0]).toContain("/api/v1/vendors/vendor%2Fa/forms?");
  expect(fetcher.mock.calls[0]![0]).toContain("filter=AWAITING_REVIEW");
  await loadVendorFormSummaries(["r1", "r2"], "form-1");
  expect(fetcher.mock.calls[1]![0]).toContain("relationship_ids=r1%2Cr2");
  expect(fetcher.mock.calls[1]![0]).toContain("form_template_id=form-1");
});
it("keeps each target recipient with its own relationship and preserves retry identity", async () => {
  const fetcher = vi.fn().mockResolvedValue({ ok: true, json: async () => ({ batch_id: "batch", items: [] }) }); vi.stubGlobal("fetch", fetcher);
  const input = { batch_id: "batch", targets: [{ relationship_id: "r1", recipient: { type: "EXTERNAL_AUDIENCE" as const, role: "TO" as const, address: "one@example.com" } }, { relationship_id: "r2", recipient: { type: "EXTERNAL_AUDIENCE" as const, role: "TO" as const, address: "two@example.com" } }], form_template_id: "form", form_template_version: 4, title: "Review security", purpose: "Refresh evidence", deadline: "2099-01-01T12:00:00Z", route_expires_at: "2099-01-01T11:00:00Z", access_policy: "DIRECT_LINK_EMAIL_OTP" as const, estimated_minutes: 15 };
  await requestVendorForms(input);
  expect(JSON.parse(fetcher.mock.calls[0]![1].body)).toEqual(input);
});
