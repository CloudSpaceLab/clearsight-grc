import { describe, expect, it } from "vitest";
import runtimeContract from "../../api/runtime.openapi.json";
import { matterOperationalCoverage, programOperationalCoverage } from "./operationalCoverage";

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

const programOperationCommands = {
  createProgram: "program.create",
  updateProgramDetails: "program.details.update",
  assignProgram: "program.assign",
  transitionProgram: "program.transition",
  addProgramRequirement: "program.requirement.add",
  supersedeProgramRequirement: "program.requirement.supersede",
  determineProgramApplicability: "program.applicability.decide",
  addProgramControlObjective: "program.control-objective.add",
  addProgramControlImplementation: "program.safeguard.add",
  linkProgramRequirementControl: "program.coverage.link",
  addProgramEvidenceContract: "program.evidence.define",
  recordProgramEvidenceAssessment: "program.evidence.assess",
  applyProgramTrigger: "program.trigger.apply",
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

describe("Program operational UI coverage", () => {
  it("maps every executable Program material command and supported ongoing workflow", () => {
    const materialOperationIDs = Object.entries(runtimeContract.paths)
      .filter(([path]) => path === "/api/v1/programs" || path.startsWith("/api/v1/programs/{id}"))
      .flatMap(([, methods]) => Object.values(methods))
      .filter((operation) => operation["x-clearsight-route-class"] === "MATERIAL_COMMAND")
      .map((operation) => operation.operationId);
    const commands = materialOperationIDs.map((operationID) => programOperationCommands[operationID as keyof typeof programOperationCommands]);
    expect(commands).not.toContain(undefined);
    for (const command of commands) expect(programOperationalCoverage).toHaveProperty(command);
    for (const command of ["program.review.accept", "monitoring.form.create", "monitoring.check.create", "monitoring.check.transition", "monitoring.collection.start", "monitoring.source.evaluate"]) {
      expect(programOperationalCoverage).toHaveProperty(command);
    }
    for (const entry of Object.values(programOperationalCoverage)) {
      expect(entry.surface).not.toBe("");
      expect(entry.states.length).toBeGreaterThan(0);
      expect(entry.testedBy.length).toBeGreaterThan(0);
    }
  });
});
