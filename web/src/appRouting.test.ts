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

describe("Program section routes", () => {
  it("parses a Program and its selected section", () => {
    expect(parseRoute("#programs/program-1/monitoring")).toEqual({ view: "programs", target: { programID: "program-1", programSection: "monitoring" } });
  });

  it("builds a Program section hash", () => {
    expect(routeHash("programs", { programID: "program-1", programSection: "history" }, "matters")).toBe("#programs/program-1/history");
  });

  it("keeps the Program id and falls back to Overview for unknown sections", () => {
    expect(parseRoute("#programs/program-1/unknown")).toEqual({ view: "programs", target: { programID: "program-1", programSection: "overview" } });
  });
});
