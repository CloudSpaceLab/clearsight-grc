import { describe, expect, it } from "vitest";
import { peopleAccessHash, peopleAccessRoute } from "./peopleAccessRouting";

describe("People and access routes", () => {
  it("keeps existing access and authority entries and gives each source tab its own destination", () => {
    expect(peopleAccessRoute("#configure/access").view).toBe("people");
    expect(peopleAccessRoute("#configure/access/roles").view).toBe("roles");
    expect(peopleAccessRoute("#configure/authority").view).toBe("policies");
    expect(peopleAccessRoute("#configure/authority/delegations").view).toBe("delegations");
  });

  it("preserves selected identities through tabs, including reserved characters", () => {
    const selections = { person: "person/with?reserved&characters", role: "risk-officer", policy: "policy:1", delegation: "delegation:1" };
    const hash = peopleAccessHash("roles", selections);
    expect(peopleAccessRoute(hash)).toEqual({ view: "roles", area: "roles", selections });
    expect(peopleAccessRoute(peopleAccessHash("people", selections)).selections.person).toBe(selections.person);
    expect(peopleAccessRoute(peopleAccessHash("people", { ...selections, person: undefined })).selections.person).toBeUndefined();
  });

  it("retains organization operations and routing checks as secondary destinations", () => {
    for (const area of ["positions", "reporting", "directory", "escalation"] as const) {
      expect(peopleAccessRoute(peopleAccessHash("organization", {}, area))).toEqual({ view: "organization", area, selections: {} });
    }
    expect(peopleAccessRoute("#configure/authority/checks").view).toBe("checks");
    expect(peopleAccessHash("organization", {}, "people")).toBe("#configure/access/positions");
  });

  it("does not carry unsupported query keys or oversized identities into routes", () => {
    const route = peopleAccessRoute(`#configure/access/unknown?tenant=other&person=${"x".repeat(257)}&role=`);
    expect(route).toEqual({ view: "people", area: "people", selections: {} });
    expect(peopleAccessHash(route.view, route.selections)).toBe("#configure/access");
  });
});
