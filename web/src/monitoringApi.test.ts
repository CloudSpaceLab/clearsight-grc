import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { requestJSON } from "./http";
import { createMonitoringLinkedIssue } from "./monitoringApi";

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
});
