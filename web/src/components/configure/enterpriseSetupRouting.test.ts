import { describe, expect, it } from "vitest";
import { enterpriseSetupHash, enterpriseSetupRoute } from "./enterpriseSetupRouting";

describe("Enterprise setup routes", () => {
  it("intercepts only the existing Organization destination", () => {
    expect(enterpriseSetupRoute("#configure/access/positions")).toEqual({ view: "organization", unit: undefined, selections: {} });
    expect(enterpriseSetupRoute("#/configure/access/positions")).toHaveProperty("view", "organization");
    for (const hash of ["#configure", "#configure/access", "#configure/access/reporting", "#configure/access/directory", "#configure/access/escalation", "#configure/authority", "#configure/access/positions/extra"]) {
      expect(enterpriseSetupRoute(hash)).toBeNull();
    }
  });

  it("restores supported Setup views and defaults unknown values to Organization", () => {
    expect(enterpriseSetupRoute("#configure/access/positions?setup=groups")).toHaveProperty("view", "groups");
    expect(enterpriseSetupRoute("#configure/access/positions?setup=jurisdictions")).toHaveProperty("view", "jurisdictions");
    expect(enterpriseSetupRoute("#configure/access/positions?setup=imports")).toHaveProperty("view", "organization");
  });

  it("preserves the selected unit and safe People identities through Setup tabs", () => {
    const selections = { person: "person/with?reserved&characters", role: "role:1", policy: "policy:1", delegation: "delegation:1" };
    for (const view of ["organization", "groups", "jurisdictions"] as const) {
      const route = enterpriseSetupRoute(enterpriseSetupHash(view, "unit/with?reserved&characters", selections));
      expect(route).toEqual({ view, unit: "unit/with?reserved&characters", selections });
    }
  });

  it("drops unsupported authority parameters, control characters and oversized identities", () => {
    const route = enterpriseSetupRoute(`#configure/access/positions?setup=groups&tenant=other&entity=other&principal=other&unit=${"x".repeat(257)}&person=%00secret&role=reader`)!;
    expect(route).toEqual({ view: "groups", unit: undefined, selections: { role: "reader" } });
    expect(enterpriseSetupHash(route.view, route.unit, route.selections)).toBe("#configure/access/positions?role=reader&setup=groups");
    expect(enterpriseSetupHash("organization", "\ninvalid", { person: "x".repeat(257) })).toBe("#configure/access/positions");
  });

  it("clears an explicit unit without losing the selected People records", () => {
    expect(enterpriseSetupHash("organization", undefined, { person: "person-2" })).toBe("#configure/access/positions?person=person-2");
  });
});
