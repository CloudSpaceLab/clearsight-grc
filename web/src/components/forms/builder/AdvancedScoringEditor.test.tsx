import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FormTemplateField } from "../../../monitoringTypes";
import { AdvancedScoringEditor } from "./AdvancedScoringEditor";

const api = vi.hoisted(() => ({ previewFormScore: vi.fn() }));
vi.mock("../../../formPoliciesApi", () => api);

const fields: FormTemplateField[] = [
  { id: "certified", section_id: "section_1", label: "Current certification", type: "yes_no", required: true, options: ["Yes", "No"] },
  { id: "coverage", section_id: "section_1", label: "Control coverage", type: "percentage", required: true },
];

describe("AdvancedScoringEditor", () => {
  it("adds a bounded contribution through labelled controls instead of raw JSON", () => {
    const changed = vi.fn();
    render(<AdvancedScoringEditor mode="RISK" fields={[...fields]} onChange={changed}/>);

    fireEvent.click(screen.getByRole("button", { name: "Add score contribution" }));

    const profile = changed.mock.calls.at(-1)?.[0];
    expect(profile.mode).toBe("RISK");
    expect(profile.direction).toBe("HIGH_IS_POOR");
    expect(profile.contributions).toHaveLength(1);
    expect(screen.queryByText(/JSON|script/i)).toBeNull();
  });

  it("builds a bounded cross-field group and limits operators to the selected response type", () => {
    const changed = vi.fn();
    render(<AdvancedScoringEditor
      mode="RISK"
      fields={fields}
      profile={{ version: "risk-v1", mode: "RISK", direction: "HIGH_IS_POOR", contributions: [{ id: "cert", label: "Certification gap", weight: 100, predicate: { field_id: "certified", operator: "EQUALS", values: ["No"] }, match_points: 100, non_match_points: 0, missing: "INDETERMINATE" }], bands: [{ band: "LOW", from: 0, through: 24 }, { band: "MODERATE", from: 25, through: 49 }, { band: "HIGH", from: 50, through: 74 }, { band: "CRITICAL", from: 75, through: 100 }] }}
      onChange={changed}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Answer equals Contribution condition operator" }));
    expect(screen.queryByRole("option", { name: "Number is at least" })).toBeNull();
    fireEvent.click(screen.getByRole("option", { name: "Answer equals" }));

    fireEvent.click(screen.getByRole("button", { name: "One condition Contribution condition match" }));
    fireEvent.click(screen.getByRole("option", { name: "All conditions" }));
    expect(changed.mock.calls.at(-1)?.[0].contributions[0].predicate).toMatchObject({ operator: "AND", children: [{ field_id: "certified" }, { field_id: "coverage" }] });
  });

  it("previews an exact stored revision on the server and presents its concern meaning", async () => {
    api.previewFormScore.mockResolvedValue({ raw_score: 78, adverse_score: 78, coverage: 1, final: true, band: "CRITICAL", contribution_results: [], rule_results: [] });
    render(<AdvancedScoringEditor
      mode="RISK"
      fields={[...fields]}
      templateRevision={{ id: "form/a", version: 4 }}
      profile={{ version: "risk-v4", mode: "RISK", direction: "HIGH_IS_POOR", contributions: [{ id: "cert", label: "Certification gap", weight: 100, predicate: { field_id: "certified", operator: "EQUALS", values: ["No"] }, match_points: 100, non_match_points: 0, missing: "INDETERMINATE" }], bands: [{ band: "LOW", from: 0, through: 24 }, { band: "MODERATE", from: 25, through: 49 }, { band: "HIGH", from: 50, through: 74 }, { band: "CRITICAL", from: 75, through: 100 }] }}
      onChange={() => undefined}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Choose an answer Current certification preview answer" }));
    fireEvent.click(screen.getByRole("option", { name: "No" }));
    fireEvent.click(screen.getByRole("button", { name: "Preview score" }));

    await waitFor(() => expect(api.previewFormScore).toHaveBeenCalledWith("form/a", 4, { certified: { text: "No" } }));
    expect(await screen.findByText("Critical concern")).toBeTruthy();
    expect(screen.getByText("78 risk score")).toBeTruthy();
  });
});
