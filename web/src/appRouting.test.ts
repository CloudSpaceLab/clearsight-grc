import { describe, expect, it } from "vitest";
import { parseRoute, routeHash } from "./appRouting";

describe("vendor routes", () => {
  it("parses a vendor relationship target", () => {
    expect(parseRoute("#vendors/relationship-1")).toEqual({ view: "vendors", target: { vendorRelationshipID: "relationship-1" } });
  });

  it("builds a vendor relationship hash", () => {
    expect(routeHash("vendors", { vendorRelationshipID: "relationship-1" }, "matters")).toBe("#vendors/relationship-1");
  });
});
