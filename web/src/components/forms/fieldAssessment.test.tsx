import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { FormTemplate } from "../../monitoringTypes";
import { buildCreateInput, draftFromTemplate, duplicateSection } from "./formAuthoring";
import { evaluateQuality } from "./formQuality";
import { FormFieldPropertyEditor } from "./FormFieldPropertyEditor";
import { FormInspector } from "./builder/FormInspector";
import { FieldAssessmentEditor } from "./FieldAssessmentEditor";

vi.mock("../../identityAccessApi", () => ({ loadIdentityAccessOverview: vi.fn().mockResolvedValue({ roles: [{ id: "r1", code: "RISK_REVIEWER", name: "Risk reviewer", capabilities: [] }] }) }));

const template = {
  id: "form", tenant_id: "bank", code: "SECURITY", name: "Security review", purpose: "Review control evidence.",
  status: "DRAFT", is_current: true, version: 4, created_at: "2026-09-08T09:00:00Z", updated_at: "2026-09-08T09:00:00Z",
  scoring_mode: "RISK", sections: [{ id: "section_1", title: "Evidence" }],
  fields: [{ id: "report", section_id: "section_1", label: "Vulnerability test", type: "file", required: true, accepted_formats: ["application/pdf"], assessment: { mode: "MANUAL", required: true, weight: 50, reviewer_role: "RISK_REVIEWER", rubric: [{ id: "sufficient", label: "Evidence sufficient", points: 0 }, { id: "gap", label: "Evidence incomplete", points: 100 }] } }],
} satisfies FormTemplate;

describe("field assessment authoring", () => {
  it("uses assessment weights for automatic compliance fields and requires an automatic rule", () => {
    const draft = draftFromTemplate(template);
    draft.scoringMode = "COMPLIANCE";
    draft.sections[0]!.weight = 100;
    draft.fields = [{ ...draft.fields[0]!, type: "yes_no", accepted_formats: undefined, options: ["Yes", "No"], scoring: { weight: 25, answer_scores: { Yes: 100, No: 0 } }, assessment: { mode: "AUTOMATIC", required: false, weight: 100 } }];
    expect(evaluateQuality(draft)).toEqual([]);
    draft.fields[0]!.scoring = undefined;
    expect(evaluateQuality(draft)).toContainEqual(expect.objectContaining({ fieldID: "report", message: expect.stringMatching(/automatic rule/i) }));
  });
  it("rejects retained advanced contributions for manual fields and requires one direct bank-confirmed contribution", () => {
    const draft = draftFromTemplate(template);
    draft.scoreProfile = { version: "profile4", mode: "RISK", direction: "HIGH_IS_POOR", bands: [{ band: "LOW", from: 0, through: 24 }, { band: "MODERATE", from: 25, through: 49 }, { band: "HIGH", from: 50, through: 74 }, { band: "CRITICAL", from: 75, through: 100 }], contributions: [{ id: "combined", label: "Report present", predicate: { operator: "AND", children: [{ field_id: "report", operator: "ANSWERED" }] }, weight: 100, match_points: 0, non_match_points: 100, missing: "INDETERMINATE" }] };
    expect(evaluateQuality(draft)).toContainEqual(expect.objectContaining({ id: "assessment-profile-conflict:report", fieldID: "report" }));
    draft.fields[0]!.assessment!.mode = "AUTOMATIC_REVIEW";
    expect(evaluateQuality(draft)).toContainEqual(expect.objectContaining({ id: "assessment-direct-rule:report", fieldID: "report" }));
    draft.scoreProfile.contributions[0]!.predicate = { field_id: "report", operator: "ANSWERED" };
    expect(evaluateQuality(draft)).toEqual([]);
  });
  it("removes incompatible bank and automatic settings when changing field assessment mode", () => {
    const onChange = vi.fn();
    const field = { ...draftFromTemplate(template).fields[0]!, scoring: { weight: 100, answer_scores: { Yes: 0, No: 100 } } };
    render(<FieldAssessmentEditor field={field} onChange={onChange}/>);
    fireEvent.click(screen.getByRole("button", { name: /How this field is assessed/ }));
    fireEvent.click(screen.getByRole("option", { name: "Not scored" }));
    expect(onChange).toHaveBeenLastCalledWith({ scoring: undefined, assessment: { mode: "NONE", required: false, weight: 50, reviewer_role: undefined, rubric: undefined } });
  });

  it("requires a reviewer responsibility and validates compliance manual weights", () => {
    const draft = draftFromTemplate(template);
    draft.fields[0]!.assessment!.reviewer_role = undefined;
    expect(evaluateQuality(draft)).toContainEqual(expect.objectContaining({ fieldID: "report", message: expect.stringMatching(/reviewer responsibility/i) }));
    draft.fields[0]!.assessment!.reviewer_role = "RISK_REVIEWER";
    draft.fields[0]!.assessment!.weight = 100;
    draft.scoringMode = "COMPLIANCE";
    draft.sections[0]!.weight = 100;
    expect(evaluateQuality(draft)).toEqual([]);
  });
  it("links an invalid assessment to its field and keeps valid rubrics out of the overview", () => {
    const onSelectField = vi.fn();
    const draft = draftFromTemplate(template);
    draft.fields[0]!.assessment!.reviewer_role = undefined;
    render(<FormInspector draft={draft} selection={{ kind: "overview" }} onSelectField={onSelectField} onPatch={() => {}} onScoringMode={() => {}} onSectionsChange={() => {}} onFieldChange={() => {}} onFieldTypeChange={() => {}} onFieldConstraint={() => {}} onFieldScoringToggle={() => {}} onMoveSection={() => {}} onDuplicateSection={() => {}} onRemoveSection={() => {}} onMoveField={() => {}} onRemoveField={() => {}}/>);
    expect(screen.getByText("Assessment and scoring")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Edit assessment" }));
    expect(onSelectField).toHaveBeenCalledWith("report");
  });
  it("retains imported assessment configuration through save and keeps copies independent", () => {
    const draft = draftFromTemplate(template);
    expect(buildCreateInput(draft).fields[0]!.assessment).toEqual(template.fields[0]!.assessment);
    draft.fields[0]!.assessment!.rubric![0]!.label = "Updated outcome";
    expect(template.fields[0]!.assessment.rubric[0]!.label).toBe("Evidence sufficient");
    const copied = duplicateSection("section_1", draft.sections, draft.fields, 2, 2)!;
    copied.fields[0]!.assessment!.rubric![0]!.label = "Copy outcome";
    expect(draft.fields[0]!.assessment!.rubric![0]!.label).toBe("Updated outcome");
  });

  it("does not add assessment modes to unchanged legacy fields", () => {
    const legacy = { ...template, fields: [{ id: "control", section_id: "section_1", type: "yes_no" as const, label: "Control in place", required: true, options: ["Yes", "No"], scoring: { weight: 100, answer_scores: { Yes: 0, No: 100 } } }] };
    const input = buildCreateInput(draftFromTemplate(legacy));
    expect(input.fields[0]!.assessment).toBeUndefined();
    expect(input.fields[0]!.scoring).toMatchObject(legacy.fields[0]!.scoring);
  });

  it("blocks approval for an empty rubric and out-of-range outcome", () => {
    const draft = draftFromTemplate(template);
    draft.fields[0]!.assessment!.rubric = [];
    expect(evaluateQuality(draft)).toContainEqual(expect.objectContaining({ fieldID: "report", blocking: true, message: expect.stringMatching(/rubric/i) }));
    draft.fields[0]!.assessment!.rubric = [{ id: "gap", label: "Gap", points: 101 }];
    expect(evaluateQuality(draft)).toContainEqual(expect.objectContaining({ fieldID: "report", blocking: true, message: expect.stringMatching(/0.100/) }));
  });

  it("offers assessment modes on document fields in the existing inspector", async () => {
    const onChange = vi.fn();
    render(<FormFieldPropertyEditor field={draftFromTemplate(template).fields[0]!} index={0} scoringMode="RISK" sections={template.sections} earlierFields={[]} onChange={onChange} onTypeChange={() => {}} onConstraint={() => {}} onScoringToggle={() => {}} onMove={() => {}} onRemove={() => {}} removable={false} first last/>);
    fireEvent.click(screen.getByRole("button", { name: /How this field is assessed/ }));
    for (const mode of ["Not scored", "Review", "Automatic rules", "Automatic rules, then review"]) expect(screen.getByRole("option", { name: mode })).toBeTruthy();
    fireEvent.click(screen.getByRole("option", { name: "Automatic rules, then review" }));
    expect(onChange).toHaveBeenCalledWith(expect.objectContaining({ assessment: expect.objectContaining({ mode: "AUTOMATIC_REVIEW", rubric: template.fields[0]!.assessment.rubric }) }));
    fireEvent.click(screen.getByRole("button", { name: /Reviewer responsibility/ }));
    await waitFor(() => expect(screen.getByRole("option", { name: "Risk reviewer" })).toBeTruthy());
  });
});
