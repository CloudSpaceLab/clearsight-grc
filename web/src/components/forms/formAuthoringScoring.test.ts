import { describe, expect, it } from "vitest";
import type { FormTemplate } from "../../monitoringTypes";
import { buildCreateInput, draftFromTemplate } from "./formAuthoring";

describe("advanced scoring authoring contract", () => {
  it("round-trips the exact score profile without converting it to legacy field weights", () => {
    const template = {
      id: "form-1", tenant_id: "bank", code: "VENDOR-CERT", name: "Vendor certification", purpose: "Collect certification evidence.",
      status: "DRAFT", is_current: true, version: 4, created_at: "2026-09-01T09:00:00Z", updated_at: "2026-09-01T09:00:00Z",
      scoring_mode: "RISK", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "section_1", title: "Certification" }],
      fields: [{ id: "certified", section_id: "section_1", label: "Current certification", type: "yes_no", required: true, options: ["Yes", "No"] }],
      score_profile: { version: "risk-v4", mode: "RISK", direction: "HIGH_IS_POOR", contributions: [{ id: "cert", label: "Certification gap", weight: 100, predicate: { field_id: "certified", operator: "EQUALS", values: ["No"] }, match_points: 100, non_match_points: 0, missing: "INDETERMINATE" }], rules: [{ id: "critical-cert", label: "Missing certification", predicate: { field_id: "certified", operator: "EQUALS", values: ["No"] }, effect: { kind: "FLOOR", value: 75 } }], bands: [{ band: "LOW", from: 0, through: 24 }, { band: "MODERATE", from: 25, through: 49 }, { band: "HIGH", from: 50, through: 74 }, { band: "CRITICAL", from: 75, through: 100 }] },
    } satisfies FormTemplate;
    const input = buildCreateInput(draftFromTemplate(template));
    expect(input.score_profile).toEqual(template.score_profile);
    expect(input.fields[0]?.scoring).toBeUndefined();
  });
});
