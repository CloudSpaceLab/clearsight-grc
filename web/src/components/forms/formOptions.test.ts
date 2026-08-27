import { describe, expect, it } from "vitest";
import { newDraft, normalizeOptionText } from "./formAuthoring";
import { evaluateDraftValidity } from "./formQuality";

describe("form option authoring", () => {
  it("deduplicates without silently truncating choices above the governed limit", () => {
    const input = Array.from({ length: 51 }, (_, index) => `Choice ${index + 1}`).join("\n");
    const options = normalizeOptionText(`${input}\nchoice 1`);
    expect(options).toHaveLength(51);

    const draft = newDraft();
    draft.code = "OPTIONS";
    draft.name = "Options";
    draft.purpose = "Verify bounded option authoring.";
    draft.fields[0] = { ...draft.fields[0]!, label: "Choice", type: "single_select", options };
    expect(evaluateDraftValidity(draft).map((issue) => issue.message)).toContain("Choice requires 2–50 unique choices.");
  });
});
