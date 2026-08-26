import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createGovernanceDelegation,
  loadGovernanceInventory,
  transitionGovernanceDelegation,
  transitionGovernancePolicy,
} from "./governanceAdminApi";

describe("governance administration API", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("loads bounded policy and delegation inventories", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: "policy-1", legal_entity_id: "entity-1" }] }))
      .mockResolvedValueOnce(jsonResponse({ items: [{ id: "delegation-1", legal_entity_id: "entity-1" }] }));
    vi.stubGlobal("fetch", fetchMock);

    const inventory = await loadGovernanceInventory();

    expect(inventory.policies).toHaveLength(1);
    expect(inventory.delegations).toHaveLength(1);
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toEqual([
      "/api/v1/governance/policies?limit=51",
      "/api/v1/governance/delegations?limit=51",
    ]);
    expect(inventory.policiesAvailable).toBe(true);
    expect(inventory.delegationsAvailable).toBe(true);
  });

  it("creates a whole-entity delegation without browser-supplied actor or tenant fields", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: "delegation-1" }, 201));
    vi.stubGlobal("fetch", fetchMock);

    await createGovernanceDelegation({
      legalEntityId: "entity-1",
      fromPrincipalId: "person-1",
      toPrincipalId: "person-2",
      responsibility: "REVIEWER",
      startsAt: "2026-09-01T08:00:00.000Z",
      endsAt: "2026-09-08T16:00:00.000Z",
      reason: "Annual leave cover",
    });

    const [, init] = fetchMock.mock.calls[0]!;
    expect(JSON.parse(String(init?.body))).toEqual({
      legal_entity_id: "entity-1",
      from_principal_id: "person-1",
      to_principal_id: "person-2",
      responsibility: "REVIEWER",
      scope: { legal_entity_id: "entity-1" },
      starts_at: "2026-09-01T08:00:00.000Z",
      ends_at: "2026-09-08T16:00:00.000Z",
      reason: "Annual leave cover",
    });
    expect(JSON.parse(String(init?.body))).not.toHaveProperty("maker_id");
    expect(JSON.parse(String(init?.body))).not.toHaveProperty("tenant_id");
  });

  it("sends only the record version and rationale for lifecycle actions", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(jsonResponse({ id: "record-1", version: 3 })));
    vi.stubGlobal("fetch", fetchMock);

    await transitionGovernancePolicy("policy/1", "retire", 2, "Superseded");
    await transitionGovernanceDelegation("delegation/1", "approve", 4);

    expect(fetchMock.mock.calls[0]![0]).toBe("/api/v1/governance/policies/policy%2F1/retire");
    expect(JSON.parse(String(fetchMock.mock.calls[0]![1]?.body))).toEqual({ expected_version: 2, rationale: "Superseded" });
    expect(fetchMock.mock.calls[1]![0]).toBe("/api/v1/governance/delegations/delegation%2F1/approve");
    expect(JSON.parse(String(fetchMock.mock.calls[1]![1]?.body))).toEqual({ expected_version: 4 });
  });
});

function jsonResponse(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}
