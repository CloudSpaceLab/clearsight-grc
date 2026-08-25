import { describe, expect, it } from "vitest";
import runtimeContract from "../../api/runtime.openapi.json";
import { matterOperationalCoverage } from "./operationalCoverage";

const operationCommands = {
  createMatter: "matter.create",
  updateMatterDetails: "matter.details.update",
  changeMatterContext: "matter.context.change",
  assignMatter: "matter.assign",
  transitionMatter: "matter.transition",
  addMatterLink: "matter.link",
  addMatterDecision: "matter.decision.record",
  addMatterAction: "matter.action.add",
  updateMatterAction: "matter.action.update",
  assignMatterAction: "matter.action.assign",
  transitionMatterAction: "matter.action.transition",
  addMatterVerificationContract: "matter.outcome.define",
  recordMatterVerificationResult: "matter.outcome.record",
  addMatterResponse: "matter.response.add",
  transitionMatterResponse: "matter.response.transition",
} as const;

describe("Matter operational UI coverage", () => {
  it("maps every executable Matter material command to a tested UI surface", () => {
    const materialOperationIDs = Object.entries(runtimeContract.paths)
      .filter(([path]) => path === "/api/v1/matters" || path.startsWith("/api/v1/matters/{id}"))
      .flatMap(([, methods]) => Object.values(methods))
      .filter((operation) => operation["x-clearsight-route-class"] === "MATERIAL_COMMAND")
      .map((operation) => operation.operationId);
    const commands = materialOperationIDs.map((operationID) => operationCommands[operationID as keyof typeof operationCommands]).sort();
    expect(commands).not.toContain(undefined);
    expect(Object.keys(matterOperationalCoverage).sort()).toEqual(commands);
    for (const entry of Object.values(matterOperationalCoverage)) {
      expect(entry.surface).not.toBe("");
      expect(entry.states.length).toBeGreaterThan(0);
      expect(entry.testedBy.length).toBeGreaterThan(0);
    }
  });
});
