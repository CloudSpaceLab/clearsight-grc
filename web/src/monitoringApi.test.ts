import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { requestJSON } from "./http";
import { createFormMonitoringCheck, createMonitoringLinkedIssue, loadCollectionSummaries, transitionMonitoringCheck, updateCollectionPolicy } from "./monitoringApi";

vi.mock("./api", () => ({ loadContext: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

describe("monitoring API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadContext).mockResolvedValue({
      tenant: { id: "tenant-1", name: "Clear Bank" }, legal_entity: { id: "entity-1", name: "Clear Bank Nigeria" },
      actor: { id: "reviewer-1", name: "Control assurance reviewer" }, mode: "demo",
    });
    vi.mocked(requestJSON).mockResolvedValue({ matter: { id: "matter-1", reference: "MAT-0001" }, created: true });
  });

  it("creates an issue from the exact result without sending an actor or Program identity", async () => {
    await createMonitoringLinkedIssue("result / 1");

    const [, path, init] = vi.mocked(requestJSON).mock.calls[0]!;
    const body = JSON.parse(String(init?.body));
    expect(path).toBe("/api/v1/monitoring-results/result%20%2F%201/linked-issue?tenant_id=tenant-1");
    expect(init?.method).toBe("POST");
    expect(body).toEqual({});
    expect(body).not.toHaveProperty("actor_id");
    expect(body).not.toHaveProperty("reviewer_principal_id");
    expect(body).not.toHaveProperty("program_id");
  });

  it("binds replacement approval to the current check revision shown to the reviewer", async () => {
    await transitionMonitoringCheck("replacement-1", 2, "ACTIVE", { id: "current-1", version: 4 });

    const [, path, init] = vi.mocked(requestJSON).mock.calls[0]!;
    expect(path).toBe("/api/v1/monitoring-checks/replacement-1/transition?tenant_id=tenant-1");
    expect(JSON.parse(String(init?.body))).toEqual({
      expected_version: 2,
      expected_current_id: "current-1",
      expected_current_version: 4,
      to: "ACTIVE",
    });
  });

  it("sends the approved collection policy with a new form check", async () => {
    const form = {
      id: "form-1", tenant_id: "tenant-1", code: "VENDOR", name: "Vendor review", purpose: "Confirm safeguards.", fields: [],
      status: "ACTIVE" as const, is_current: true, version: 2, created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z",
    };
    const policy = { validity_months: 12, renewal_window_days: 30, reminder_count: 3 };
    await createFormMonitoringCheck("program/1", form, policy);
    const [, path, init] = vi.mocked(requestJSON).mock.calls[0]!;
    expect(path).toBe("/api/v1/programs/program%2F1/monitoring-checks?tenant_id=tenant-1");
    expect(JSON.parse(String(init?.body)).collection_policy).toEqual(policy);
  });

  it("updates a policy by expected version and loads bounded Program summaries", async () => {
    const policy = { validity_months: 12, renewal_window_days: 30, reminder_count: 3 };
    await updateCollectionPolicy("check/1", 4, policy);
    const first = vi.mocked(requestJSON).mock.calls[0]!;
    expect(JSON.parse(String(first[2]?.body))).toEqual({ expected_version: 4, collection_policy: policy });

    vi.mocked(requestJSON).mockResolvedValueOnce({ items: [] });
    await loadCollectionSummaries("program/1");
    expect(vi.mocked(requestJSON).mock.calls[1]?.[1]).toBe("/api/v1/programs/program%2F1/collection-summaries?limit=100&tenant_id=tenant-1");
  });
});
