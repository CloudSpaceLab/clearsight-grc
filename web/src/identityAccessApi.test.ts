import { afterEach, describe, expect, it, vi } from "vitest";
import { loadIdentityAccessOverview } from "./identityAccessApi";

describe("loadIdentityAccessOverview", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("normalizes empty server collections instead of exposing null to the Configure view", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      sign_in: { mode: "local", assurance_level: "demo" },
      actor_principal_id: "system-admin",
      can_configure: true,
      can_configure_organization: true,
      can_configure_escalation: true,
      sources: null,
      people: null,
      groups: null,
      roles: null,
      legal_entities: null,
      bindings: null,
      organization_scopes: null,
      organization_scope_revisions: null,
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
    expect(overview.organization_scopes).toEqual([]);
    expect(overview.organization_scope_revisions).toEqual([]);
    expect(overview.data_boundary).toMatchObject({
      detail_transfer_mode: "AGGREGATE_ONLY",
      allowed_destination_regions: [],
      version: 0,
      configured: false,
    });
    expect(overview.data_boundary_revisions).toEqual([]);
    expect(overview.escalation_policies).toEqual([]);
  });
});
