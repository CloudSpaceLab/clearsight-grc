import { describe, expect, it } from "vitest";
import { normalizeProgramAggregate } from "./programAggregate";

describe("Program aggregate normalization", () => {
  it("converts nullable collection fields from a newly created Program to empty arrays", () => {
    const value = normalizeProgramAggregate({
      state_label: "Setup in progress",
      program: {
        id: "program-1", tenant_id: "bank-1", code: "MOBILE", name: "Mobile banking", type: "CHANNEL", status: "DRAFT",
        owning_function: "Digital Banking", scope: {}, effective_from: "2026-08-17T00:00:00Z", created_at: "2026-08-17T00:00:00Z", updated_at: "2026-08-17T00:00:00Z", version: 1,
      },
      requirements: null,
      applicability: null,
      control_objectives: null,
      control_implementations: null,
      requirement_control_links: null,
      evidence_contracts: null,
      evidence_assessments: null,
      triggers: null,
    });

    expect(value.requirements).toEqual([]);
    expect(value.control_implementations).toEqual([]);
    expect(value.evidence_contracts).toEqual([]);
  });
});
