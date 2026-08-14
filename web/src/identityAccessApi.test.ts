import { afterEach, describe, expect, it, vi } from "vitest";
import { loadIdentityAccessOverview } from "./identityAccessApi";

describe("loadIdentityAccessOverview", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("normalizes empty server collections instead of exposing null to the Configure view", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      sign_in: { mode: "local", assurance_level: "demo" },
      actor_principal_id: "system-admin",
      can_configure: true,
      can_configure_escalation: true,
      sources: null,
      people: null,
      groups: null,
      roles: null,
      legal_entities: null,
      bindings: null,
      escalation: { pending_timers: 0, escalated_tasks: 0, unresolved_24h: 0, failed_timers: 0 },
      escalation_policies: null,
    }), { status: 200, headers: { "Content-Type": "application/json" } })));

    const overview = await loadIdentityAccessOverview();

    expect(overview.sources).toEqual([]);
    expect(overview.people).toEqual([]);
    expect(overview.groups).toEqual([]);
    expect(overview.roles).toEqual([]);
    expect(overview.legal_entities).toEqual([]);
    expect(overview.bindings).toEqual([]);
    expect(overview.escalation_policies).toEqual([]);
  });
});
