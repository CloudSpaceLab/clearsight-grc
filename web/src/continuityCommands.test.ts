import { beforeEach, describe, expect, it, vi } from "vitest";
import { loadContext } from "./api";
import { createMatter } from "./continuityCommands";
import { requestJSON } from "./http";

vi.mock("./api", () => ({ loadContext: vi.fn(), resolveAuthority: vi.fn() }));
vi.mock("./http", () => ({ requestJSON: vi.fn() }));

describe("continuity commands", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(loadContext).mockResolvedValue({
      tenant: { id: "tenant-1", name: "Clear Bank" }, legal_entity: { id: "entity-1", name: "Clear Bank Nigeria" },
      actor: { id: "actor-1", name: "Risk owner" }, mode: "demo",
    });
    vi.mocked(requestJSON).mockResolvedValue({});
  });

  it("maps work creation to the actor-bound Matter command", async () => {
    await createMatter({
      type: "CONTROL_GAP", priority: 4, title: "Face verification is unavailable",
      summary: "The mobile channel did not return a successful result.", affectedArea: "Mobile banking",
      knownInformation: "The status check failed.", missingInformation: ["Confirm SDK version"],
      dueAt: "2026-09-30T22:59:59.999Z", programID: "program-mobile",
    });

    expect(requestJSON).toHaveBeenCalledTimes(1);
    const [base, path, init] = vi.mocked(requestJSON).mock.calls[0]!;
    expect(base).toBe("");
    expect(path).toBe("/api/v1/matters?tenant_id=tenant-1");
    expect(init?.method).toBe("POST");
    expect(JSON.parse(String(init?.body))).toEqual({
      tenant_id: "tenant-1", type: "CONTROL_GAP", priority: 4, title: "Face verification is unavailable",
      summary: "The mobile channel did not return a successful result.", scope: { access: "INTERNAL", area: "Mobile banking" },
      known_facts: { notes: "The status check failed." }, missing_facts: ["Confirm SDK version"], contradictions: [],
      owner_principal_id: "actor-1", due_at: "2026-09-30T22:59:59.999Z", program_id: "program-mobile",
    });
  });
});
