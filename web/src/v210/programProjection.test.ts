import { expect, it } from "vitest";
import { programCount, programStatus } from "./programProjection";

const current = { program: { version: 5 }, program_version: 5, assessed_program_version: 5, projection_version: 3, projection_stale: false, state_generated_at: "2026-11-20T10:00:00Z", overall_state: "CURRENT", state_label: "Up to date" } as any;

it("does not present stale or earlier assessments as current", () => {
  expect(programStatus({ ...current, projection_stale: true })).toBe("Out of date");
  expect(programStatus({ ...current, assessed_program_version: 4 })).toBe("Out of date");
  expect(programStatus({ ...current, program_version: 4 })).toBe("Out of date");
});

it("requires version, freshness and state provenance before showing current status", () => {
  for (const field of ["projection_stale", "program_version", "assessed_program_version", "projection_version", "state_generated_at", "overall_state", "state_label"]) {
    expect(programStatus({ ...current, [field]: undefined }), field).toBe("Unknown");
  }
  expect(programStatus({ ...current, state_generated_at: "invalid" })).toBe("Unknown");
  expect(programStatus(current)).toBe("Up to date");
});

it("keeps missing or invalid counts unknown while retaining real zero", () => {
  for (const value of [undefined, null, -1, NaN, 1.5]) expect(programCount(value)).toBe("Unknown");
  expect(programCount(0)).toBe("0");
  expect(programCount(8)).toBe("8");
});
