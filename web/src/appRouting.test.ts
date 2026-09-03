import { describe, expect, it } from "vitest";
import { parseRoute, routeHash } from "./appRouting";

describe("workspace routes", () => {
  it("keeps filter queries out of record targets", () => {
    expect(parseRoute("#work/matters/matter%2F1?status=OPEN&priority=4")).toEqual({ view: "work", workTab: "matters", target: { matterID: "matter/1" } });
    expect(parseRoute("#programs/program%2F1?overall_state=CURRENT")).toEqual({ view: "programs", target: { programID: "program/1", programSection: "overview" } });
  });

  it("parses and builds vendor relationship targets", () => {
    expect(parseRoute("#vendors/relationship-1")).toEqual({ view: "vendors", target: { vendorRelationshipID: "relationship-1" } });
    expect(routeHash("vendors", { vendorRelationshipID: "relationship-1" }, "matters")).toBe("#vendors/relationship-1");
  });

  it("keeps the Forms search query separate from the selected exact template", () => {
    expect(parseRoute("#forms/template%2F1?search=vendor&status=ACTIVE")).toEqual({ view: "forms", target: { formTemplateID: "template/1" } });
    expect(routeHash("forms", { formTemplateID: "template/1" }, "matters")).toBe("#forms/template%2F1");
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
