import { describe, expect, it, vi } from "vitest";
import type { FormScoreProfile } from "../../monitoringTypes";
import { validateAdvancedScoreProfile, validateFieldContractBounds } from "./formContractQuality";

describe("form contract recovery messages", () => {
  it.each(["", "bad key", "a".repeat(201)])("explains how to correct an invalid record destination %s", (key) => {
    const block = vi.fn();
    validateFieldContractBounds({ id: "name", label: "Registered name", type: "short_text", required: false, record_target: { key, required_subject_type: "VENDOR" } }, 0, block);
    expect(block).toHaveBeenCalledWith("record-target-format:name", "Registered name requires a valid record field and record type. Check the record destination.", { fieldID: "name" });
  });

  it.each([0, 101])("states the real score-weight range for invalid weight %s", (weight) => {
    const block = vi.fn();
    const profile: FormScoreProfile = { version: "v1", mode: "RISK", direction: "HIGH_IS_POOR", contributions: [{ id: "review", label: "Access review", weight, match_points: 100, non_match_points: 0, missing: "INDETERMINATE", predicate: { operator: "ANSWERED", field_id: "reviewed" } }], bands: [] };
    validateAdvancedScoreProfile(profile, [{ id: "reviewed", label: "Review complete", type: "yes_no", required: false }], block);
    expect(block).toHaveBeenCalledWith("score-contribution:review", "Access review requires a label, a weight from 1–100 and points from 0–100.");
  });
});
