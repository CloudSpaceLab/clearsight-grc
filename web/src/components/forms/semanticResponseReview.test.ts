import { describe, expect, it } from "vitest";
import type { ResponseFieldAssessment } from "../../formAssessmentApi";
import { fieldAssessmentLabel } from "./fieldAssessment";
import { buildSemanticResponseReview } from "./semanticResponseReview";

function field(id: string, description: string, answer?: ResponseFieldAssessment["answer"], decision?: ResponseFieldAssessment["decision"]): ResponseFieldAssessment {
  return { field: { id, section_id: id.startsWith("requirement_1") ? "service_1" : "service_2", label: id.includes("response") ? "Security certification missing" : "Current certification", type: id.includes("response") ? "long_text" : "vendor_document", required: false, description, assessment: id.includes("response") ? { mode: "MANUAL", required: true, weight: 20, reviewer_role: "REVIEWER", rubric: [] } : undefined }, answer: answer ?? {}, decision };
}

describe("semantic response review", () => {
  it("groups requirements by service and calculates stored response states", () => {
    const fields = [
      field("requirement_1_response", "Vendor response · Service: Moneytor GetPaid application", { text: "Audit underway" }),
      field("requirement_1_evidence", "Supporting evidence · Service: Moneytor GetPaid application"),
      field("requirement_2_response", "Vendor response · Service: Payment Terminal Service Provider (PTSP)"),
      field("requirement_2_evidence", "Supporting evidence · Service: Payment Terminal Service Provider (PTSP)", { artifact_ids: ["artifact-1"] }),
    ];
    expect(buildSemanticResponseReview(fields)).toMatchObject({
      metrics: { requirements: 2, answered: 1, evidenceReceived: 1, awaitingReview: 2 },
      services: [{ name: "Moneytor GetPaid application" }, { name: "Payment Terminal Service Provider (PTSP)" }],
    });
  });

  it("does not invent an implementation label for an unconfigured field", () => {
    expect(fieldAssessmentLabel({ id: "note", label: "Note", type: "long_text", required: false })).toBe("");
  });
});
