import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { requestJSON } from "./http";
import { createFormMonitoringCheck, loadCollectionSummaries, updateCollectionPolicy } from "./monitoringApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

const form = {
  id: "form-1", tenant_id: "bank-1", code: "VENDOR", name: "Vendor review", purpose: "Confirm safeguards.", fields: [],
  status: "ACTIVE" as const, is_current: true, version: 2, created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z",
};
const policy = { validity_months: 12, renewal_window_days: 30, reminder_count: 3 };

describe("monitoring collection API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadContext).mockResolvedValue({ tenant: { id: "bank-1" } } as Awaited<ReturnType<typeof loadContext>>);
    vi.mocked(requestJSON).mockResolvedValue({ items: [] });
  });

  it("creates a form check with the approved collection policy", async () => {
    await createFormMonitoringCheck("program/1", form, policy);
    expect(requestJSON).toHaveBeenCalledWith("", "/api/v1/programs/program%2F1/monitoring-checks?tenant_id=bank-1", expect.objectContaining({
      method: "POST",
      body: expect.any(String),
    }));
    const init = vi.mocked(requestJSON).mock.calls[0]?.[2] as RequestInit;
    expect(JSON.parse(String(init.body)).collection_policy).toEqual(policy);
  });

  it("updates policy by expected version and loads one bounded Program summary", async () => {
    await updateCollectionPolicy("check/1", 4, policy);
    let init = vi.mocked(requestJSON).mock.calls[0]?.[2] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({ expected_version: 4, collection_policy: policy });

    vi.mocked(requestJSON).mockResolvedValueOnce({ items: [] });
    await loadCollectionSummaries("program/1");
    expect(vi.mocked(requestJSON).mock.calls[1]?.[1]).toBe("/api/v1/programs/program%2F1/collection-summaries?limit=100&tenant_id=bank-1");
  });
});
