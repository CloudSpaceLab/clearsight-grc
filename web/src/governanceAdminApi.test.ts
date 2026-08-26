import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createGovernancePolicyDraft,
  createGovernanceDelegation,
  loadGovernanceInventory,
  searchGovernanceDelegationCandidates,
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

  it("loads bounded responsibility-scoped delegation candidates with safe labels", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ items: [{ principal_id: "person-1", display_name: "Ada Okafor", context_label: "Risk assurance lead", can_give: true, can_receive: true }], has_more: false }));
    vi.stubGlobal("fetch", fetchMock);

    const page = await searchGovernanceDelegationCandidates("REVIEWER", "risk assurance");

    expect(page.items[0]?.display_name).toBe("Ada Okafor");
    expect(fetchMock.mock.calls[0]![0]).toBe("/api/v1/governance/delegation-candidates?responsibility=REVIEWER&limit=50&q=risk+assurance");
  });

  it("creates a routing policy draft from business fields without actor, tenant, or raw JSON input", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: "policy-1", version: 1 }, 201));
    vi.stubGlobal("fetch", fetchMock);

    await createGovernancePolicyDraft({
      legalEntityId: "entity-1", code: "PAYMENT_REVIEW", name: "Payment review route", responsibility: "REVIEWER",
      roleCode: "CONTROL_ASSURANCE", objectType: "MATTER", decisionType: "matter.outcome.record", minMateriality: 2,
      priority: 100, effectiveFrom: "2026-09-01T00:00:00.000Z",
    });

    const [, init] = fetchMock.mock.calls[0]!;
    const body = JSON.parse(String(init?.body));
    expect(body).toEqual({
      legal_entity_id: "entity-1", code: "PAYMENT_REVIEW", name: "Payment review route", effective_from: "2026-09-01T00:00:00.000Z",
      definition: { rules: [{ id: "payment-review-route", legal_entity_id: "entity-1", object_type: "MATTER", responsibility: "REVIEWER", decision_type: "matter.outcome.record", min_materiality: 2, priority: 100, selector: { kind: "ROLE", ref: "CONTROL_ASSURANCE" } }] },
    });
    expect(body).not.toHaveProperty("tenant_id");
    expect(body).not.toHaveProperty("maker_id");
  });

  it("expands the combined Program and issue scope without creating a global object wildcard", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: "policy-1", version: 1 }, 201));
    vi.stubGlobal("fetch", fetchMock);

    await createGovernancePolicyDraft({
      legalEntityId: "entity-1", code: "REVIEW", name: "Review route", responsibility: "REVIEWER",
      roleCode: "CONTROL_ASSURANCE", objectType: "*", minMateriality: 0, priority: 100,
    });

    const body = JSON.parse(String(fetchMock.mock.calls[0]![1]?.body));
    expect(body.definition.rules).toEqual([
      expect.objectContaining({ id: "review-route-program", object_type: "PROGRAM" }),
      expect.objectContaining({ id: "review-route-matter", object_type: "MATTER" }),
    ]);
    expect(body.definition.rules.some((rule: Record<string, unknown>) => !rule.object_type || rule.object_type === "*")).toBe(false);
  });

  it("sends only the record version and rationale for lifecycle actions", async () => {
    const fetchMock = vi.fn().mockImplementation(() => Promise.resolve(jsonResponse({ id: "record-1", version: 3 })));
    vi.stubGlobal("fetch", fetchMock);

    await transitionGovernancePolicy("policy/1", "retire", 2, "Superseded");
    await transitionGovernancePolicy("policy/2", "reject", 6, "Narrow the legal-entity scope");
    await transitionGovernanceDelegation("delegation/1", "approve", 4);

    expect(fetchMock.mock.calls[0]![0]).toBe("/api/v1/governance/policies/policy%2F1/retire");
    expect(JSON.parse(String(fetchMock.mock.calls[0]![1]?.body))).toEqual({ expected_version: 2, rationale: "Superseded" });
    expect(fetchMock.mock.calls[1]![0]).toBe("/api/v1/governance/policies/policy%2F2/reject");
    expect(JSON.parse(String(fetchMock.mock.calls[1]![1]?.body))).toEqual({ expected_version: 6, rationale: "Narrow the legal-entity scope" });
    expect(fetchMock.mock.calls[2]![0]).toBe("/api/v1/governance/delegations/delegation%2F1/approve");
    expect(JSON.parse(String(fetchMock.mock.calls[2]![1]?.body))).toEqual({ expected_version: 4 });
  });
});

function jsonResponse(value: unknown, status = 200) {
  return new Response(JSON.stringify(value), { status, headers: { "Content-Type": "application/json" } });
}
