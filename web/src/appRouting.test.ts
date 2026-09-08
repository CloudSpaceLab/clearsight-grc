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
  it.each(["requirement", "control-objective"] as const)("round-trips an encoded %s target", (kind) => {
    const target = { programID: "program/1 #銀行", programSection: "requirements-controls" as const, programItem: { kind, id: "item/1 #銀行" } };
    const hash = routeHash("programs", target, "matters");
    expect(hash).toBe(`#programs/program%2F1%20%23%E9%8A%80%E8%A1%8C/requirements-controls/${kind}/item%2F1%20%23%E9%8A%80%E8%A1%8C`);
    expect(parseRoute(hash).target).toEqual(target);
  });

  it.each([
    "requirements-controls/requirement", "requirements-controls/unknown/item-1",
    "requirements-controls/requirement//item-1", "requirements-controls/requirement/item-1/extra",
    "requirements-controls/requirement/%ZZ", "overview/requirement/item-1",
  ])("ignores incomplete or unsupported item segments: %s", (path) => {
    expect(parseRoute(`#programs/program-1/${path}`).target).not.toHaveProperty("programItem");
  });

  it("does not carry an item into another section", () => {
    expect(routeHash("programs", { programID: "program-1", programSection: "history", programItem: { kind: "requirement", id: "item-1" } }, "matters")).toBe("#programs/program-1/history");
  });

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
