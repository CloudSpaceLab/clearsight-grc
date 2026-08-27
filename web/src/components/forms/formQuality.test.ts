import { describe, expect, it } from "vitest";
import type { FormDraft } from "./formAuthoring";
import { evaluateDraftValidity, evaluateQuality } from "./formQuality";

function complianceDraft(): FormDraft {
  return {
    code: "VENDOR-COMPLIANCE",
    name: "Vendor compliance",
    purpose: "Collect current vendor compliance evidence.",
    scoringMode: "COMPLIANCE",
    presentation: "AUTOMATIC",
    allowModeSwitch: false,
    sections: [{ id: "identity", title: "Vendor identity", weight: 80 }],
    fields: [{
      id: "registration",
      section_id: "identity",
      label: "Registration verified",
      type: "yes_no",
      required: true,
      options: ["Yes", "No"],
      scoring: { weight: 80, answer_scores: { Yes: 100, No: 0 } },
      collection_intent: "CAPTURE",
      browser_cache_policy: "ALLOWED",
    }],
  };
}

describe("form quality contract parity", () => {
  it("defers compliance allocation completeness for drafts but not approval", () => {
    const draft = complianceDraft();
    expect(evaluateDraftValidity(draft).filter((issue) => issue.blocking)).toEqual([]);
    expect(evaluateQuality(draft).map((issue) => issue.message)).toContain("20% remains to allocate in Vendor identity");
  });

  it("still blocks structurally invalid compliance weights while drafting", () => {
    const draft = complianceDraft();
    draft.sections[0]!.weight = 150;
    expect(evaluateDraftValidity(draft).map((issue) => issue.message)).toContain("Vendor identity requires a whole-number weight from 0–100%.");
  });

  it("blocks malformed record targets before persistence", () => {
    const draft = complianceDraft();
    draft.sections[0]!.weight = 100;
    draft.fields[0]!.scoring!.weight = 100;
    draft.fields[0]!.type = "vendor_document";
    draft.fields[0]!.options = undefined;
    draft.fields[0]!.scoring = undefined;
    draft.fields[0]!.accepted_formats = ["application/pdf"];
    draft.fields[0]!.constraints = { max_files: 1, max_file_bytes: 10 * 1024 * 1024 };
    draft.fields[0]!.collection_intent = "REPLACE_HELD_DOCUMENT";
    draft.fields[0]!.record_target = { key: "bad target?", required_subject_type: "VENDOR_RELATIONSHIP" };
    expect(evaluateDraftValidity(draft).map((issue) => issue.message)).toContain("Registration verified requires a valid bounded record target.");
  });
});
