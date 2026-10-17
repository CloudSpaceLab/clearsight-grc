import { describe, expect, it } from "vitest";
import type { ProgramAggregate } from "./types";
import { programOverviewMetrics } from "./programOverviewMetrics";

const aggregate = {
  program: { id: "program-1", version: 5 },
  requirements: [{ id: "a", status: "APPROVED" }, { id: "b", status: "PROPOSED" }],
  evidence_contracts: [{ id: "e", status: "ACTIVE" }, { id: "retired", status: "RETIRED" }],
} as unknown as ProgramAggregate;

const policy = {
  last_revision: 1,
  active: { approved_at: "2026-10-15T09:00:00Z", definition: {
    revision: "r1", categories: [{ code: "REQUIREMENTS", weight_bps: 7000 }, { code: "VENDORS", weight_bps: 3000 }],
  } },
};

const calculation = {
  policy_revision: "r1", state: "FINAL" as const, coverage_bps: 7500,
  categories: [{ code: "REQUIREMENTS", weight_bps: 7000, eligible_count: 8, assessed_count: 6,
    unknown_count: 2, excluded_count: 0, coverage_bps: 7500 }],
};
const snapshot = { program_version: 5, generated_at: "2026-10-17T10:00:00Z", open_matter_count: 0, calculation };

describe("Program overview measures", () => {
  it("uses native approved requirements and active evidence checks without inferring conclusions", () => {
    const measures = programOverviewMetrics(aggregate);
    expect(measures.map((item) => item.label)).toEqual([
      "Approved requirements", "Assessed requirements", "Open issues", "Active evidence checks",
    ]);
    expect(measures.map((item) => item.value)).toEqual([1, "—", "—", 1]);
    expect(measures.map((item) => item.quality)).toEqual(["current", "unknown", "unknown", "current"]);
  });

  it("counts a verified zero separately from unknown, with source provenance", () => {
    const result = programOverviewMetrics({
      ...aggregate, program: { ...aggregate.program, calculation_policy: policy },
      current_state: { ...snapshot, calculation: { ...calculation, coverage_bps: 10000,
        categories: [{ ...calculation.categories[0]!, eligible_count: 0, assessed_count: 0, unknown_count: 0, coverage_bps: 10000 }] } },
    } as ProgramAggregate);
    expect(result[1]).toMatchObject({ value: 0, quality: "current" });
    expect(result[2]).toMatchObject({ value: 0, quality: "current" });
    expect(result[2]?.meta).toContain("Calculated 2026-10-17");
  });

  it("marks incomplete and provisional assessment coverage without fabricating supported counts", () => {
    const result = programOverviewMetrics({
      ...aggregate, program: { ...aggregate.program, calculation_policy: policy },
      current_state: snapshot,
    } as ProgramAggregate);
    expect(result[1]).toMatchObject({ value: 6, quality: "partial", detail: "6 of 8 eligible" });
    expect(result[1]?.label).not.toContain("Supported");
    const provisional = programOverviewMetrics({
      ...aggregate, program: { ...aggregate.program, calculation_policy: policy },
      current_state: { ...snapshot, calculation: { ...calculation, state: "PROVISIONAL" } },
    } as ProgramAggregate);
    expect(provisional[1]).toMatchObject({ qualityLabel: "Provisional" });
  });

  it("marks older Program snapshots stale and retains their exact basis", () => {
    const result = programOverviewMetrics({
      ...aggregate, program: { ...aggregate.program, calculation_policy: policy },
      current_state: { ...snapshot, program_version: 4 },
    } as ProgramAggregate);
    expect(result[1]).toMatchObject({ value: 6, quality: "stale" });
    expect(result[2]).toMatchObject({ value: 0, quality: "stale" });
    expect(result[2]?.meta).toContain("Program v4");
  });

  it("rejects missing approval, conflicting revision and inconsistent denominators", () => {
    const invalid = [
      { ...aggregate, current_state: snapshot },
      { ...aggregate, program: { ...aggregate.program, calculation_policy: policy }, current_state: { ...snapshot,
        calculation: { ...calculation, policy_revision: "r0" } } },
      { ...aggregate, program: { ...aggregate.program, calculation_policy: policy }, current_state: { ...snapshot,
        calculation: { ...calculation, categories: [{ ...calculation.categories[0]!, assessed_count: 9 }] } } },
    ] as ProgramAggregate[];
    for (const value of invalid) expect(programOverviewMetrics(value)[1]).toMatchObject({ value: "—", quality: "unknown" });
  });

  it("renders unknown for malformed policy and category arrays", () => {
    const malformed = [
      { ...aggregate, program: { ...aggregate.program, calculation_policy: { active: { approved_at: "2026-10-15", definition: { revision: "r1" } } } }, current_state: snapshot },
      { ...aggregate, program: { ...aggregate.program, calculation_policy: policy }, current_state: { ...snapshot, calculation: { ...calculation, categories: null } } },
    ] as unknown as ProgramAggregate[];
    for (const input of malformed) expect(programOverviewMetrics(input)[1]).toMatchObject({ value: "—", quality: "unknown" });
  });

  it("does not treat invalid timestamps or counts as current posture", () => {
    const result = programOverviewMetrics({
      ...aggregate, current_state: { ...snapshot, generated_at: "invalid", open_matter_count: -2 },
    } as ProgramAggregate);
    expect(result[2]).toMatchObject({ value: "—", quality: "unknown" });
  });
});
