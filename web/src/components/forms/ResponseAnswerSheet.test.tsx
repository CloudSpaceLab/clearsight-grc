import axe from "axe-core";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { ResponseAssessmentDetail } from "../../formAssessmentApi";
import type { FormScoreProfile, FormTemplateField, FormTemplateSection } from "../../monitoringTypes";
import type { CaptureAnswerValue } from "../../types";
import { AnswerSheetSummary, ResponseAnswerSheet, answerAttention, groupAnswerFields, type AnswerSheetItem } from "./ResponseAnswerSheet";

function fieldItem(id: string, label: string, type: FormTemplateField["type"], sectionID?: string): AnswerSheetItem {
  return { field: { id, label, type, required: false, section_id: sectionID } };
}

function answered(item: AnswerSheetItem, answer: CaptureAnswerValue = { text: "Yes" }): AnswerSheetItem {
  return { ...item, answer };
}

async function expectNoViolations(container: HTMLElement) {
  const results = await axe.run(container, { rules: { "color-contrast": { enabled: false } } });
  expect(results.violations.map((violation) => violation.id)).toEqual([]);
}

function groupOf(title: string) {
  const heading = screen.getByRole("heading", { name: title, level: 4 });
  const group = heading.closest(".response-answer-sheet__group");
  expect(group).not.toBeNull();
  return group as HTMLElement;
}

const scoreProfile: FormScoreProfile = {
  version: "risk4",
  mode: "RISK",
  direction: "HIGH_IS_POOR",
  contributions: [],
  rules: [],
  bands: [
    { band: "LOW", from: 0, through: 39 },
    { band: "MODERATE", from: 40, through: 69 },
    { band: "HIGH", from: 70, through: 89 },
    { band: "CRITICAL", from: 90, through: 100 },
  ],
};

const detail: ResponseAssessmentDetail = {
  response_id: "response-1",
  form_template_id: "form-1",
  form_template_version: 1,
  version: 0,
  current: true,
  state: "ASSESSED",
  required_count: 0,
  reviewed_required_count: 0,
  reviewed_count: 0,
  fields: [],
  automatic_score: { state: "FINAL", mode: "RISK", direction: "HIGH_IS_POOR", raw_score: 20, adverse_score: 20, coverage: 1, calculated_at: "2026-09-08T10:00:00Z", contribution_results: [], rule_results: [] },
  assessed_score: { state: "FINAL", mode: "RISK", direction: "HIGH_IS_POOR", coverage: 1, final: true },
  score_profile: scoreProfile,
};

const automaticDetail: ResponseAssessmentDetail = {
  ...detail,
  automatic_score: { state: "FINAL", mode: "RISK", direction: "HIGH_IS_POOR", raw_score: 80, adverse_score: 80, coverage: 1, calculated_at: "2026-09-08T10:00:00Z", contribution_results: [{ id: "access-review-gap", outcome: "PASS", points: 80, weight: 50 }], rule_results: [] },
  score_profile: {
    ...scoreProfile,
    contributions: [{ id: "access-review-gap", label: "Access review gap", weight: 50, match_points: 80, non_match_points: 0, missing: "INDETERMINATE", predicate: { field_id: "control", operator: "EQUALS", values: ["No"] } }],
  },
};

const sections: FormTemplateSection[] = [
  { id: "branch-details", title: "Branch details", help: "Confirm the branch that owns the register." },
  { id: "nov-2025", title: "November 2025 register" },
];

const groupedFields: AnswerSheetItem[] = [
  answered(fieldItem("branch", "Branch", "short_text", "branch-details"), { text: "Lagos" }),
  answered(fieldItem("directorate", "Directorate", "short_text"), { text: "Retail" }),
  fieldItem("cash_overage_value", "Cash overage value", "currency", "branch-details"),
  answered(fieldItem("reporting_date", "Reporting date", "date", "nov-2025"), { text: "2026-09-01" }),
];

const mixed12: AnswerSheetItem[] = [
  ...Array.from({ length: 10 }, (_, index) => answered(fieldItem(`answered-${index}`, `Answered field ${index}`, "short_text"), { text: `value-${index}` })),
  { field: { id: "followup_owner", label: "Follow-up owner", type: "short_text", required: true }, answer: {} },
  { field: { id: "supporting_document", label: "Supporting evidence", type: "file", required: false }, answer: {} },
];

const assessedFields: AnswerSheetItem[] = [
  ...mixed12,
  { field: { id: "oversight", label: "Oversight review", type: "yes_no", required: true }, answer: { text: "No" }, decision: { points: 85 } },
  { field: { id: "control", label: "Privileged access reviews", type: "yes_no", required: true }, answer: { text: "No" } },
];

const allAnswered: AnswerSheetItem[] = [
  answered(fieldItem("branch", "Branch", "short_text", "branch-details"), { text: "Lagos" }),
  answered(fieldItem("directorate", "Directorate", "short_text"), { text: "Retail" }),
  answered(fieldItem("reporting_date", "Reporting date", "date"), { text: "2026-09-01" }),
];

describe("answerAttention", () => {
  it("flags missing answers and missing evidence without an assessment context", () => {
    expect(answerAttention({ field: { id: "owner", label: "Follow-up owner", type: "short_text", required: true }, answer: {} })).toBe("MISSING_ANSWER");
    expect(answerAttention({ field: { id: "certificate", label: "Certificate", type: "file", required: false }, answer: {} })).toBe("MISSING_EVIDENCE");
    expect(answerAttention({ field: { id: "signature", label: "Director signature", type: "signature", required: false }, answer: {} })).toBe("MISSING_EVIDENCE");
    expect(answerAttention({ field: { id: "branch", label: "Branch", type: "short_text", required: false }, answer: { text: "Lagos" } })).toBeUndefined();
  });

  it("classifies POOR only with an assessment context and only at or above the concern threshold", () => {
    const poor: AnswerSheetItem = { field: { id: "oversight", label: "Oversight review", type: "yes_no", required: true }, answer: { text: "No" }, decision: { points: 75 } };
    expect(answerAttention(poor, detail)).toBe("POOR");
    expect(answerAttention(poor)).toBeUndefined();
    expect(answerAttention({ ...poor, decision: { points: 70 } }, detail)).toBe("POOR");
    expect(answerAttention({ ...poor, decision: { points: 40 } }, detail)).toBeUndefined();
    expect(answerAttention({ field: { id: "branch", label: "Branch", type: "short_text", required: false }, answer: { text: "Lagos" } }, detail)).toBeUndefined();
    const automaticPoor: AnswerSheetItem = { field: { id: "control", label: "Privileged access reviews", type: "yes_no", required: true }, answer: { text: "No" } };
    expect(answerAttention(automaticPoor, automaticDetail)).toBe("POOR");
    expect(answerAttention(automaticPoor)).toBeUndefined();
    expect(answerAttention({ field: { id: "owner", label: "Follow-up owner", type: "short_text", required: true }, answer: {} }, detail)).toBe("MISSING_ANSWER");
  });
});

describe("groupAnswerFields", () => {
  it("groups fields under sections in template order and pulls unmatched fields into Other answers", () => {
    const groups = groupAnswerFields(groupedFields, sections);
    expect(groups.map((group) => [group.id, group.title, group.fields.map((item) => item.field.id)])).toEqual([
      ["branch-details", "Branch details", ["branch", "cash_overage_value"]],
      ["nov-2025", "November 2025 register", ["reporting_date"]],
      [undefined, "Other answers", ["directorate"]],
    ]);
    expect(groups[0]?.help).toBe("Confirm the branch that owns the register.");
  });

  it("falls back to a single All answers group without sections", () => {
    expect(groupAnswerFields(groupedFields)).toEqual([{ title: "All answers", fields: groupedFields }]);
    expect(groupAnswerFields(groupedFields, [])).toEqual([{ title: "All answers", fields: groupedFields }]);
  });
});

describe("AnswerSheetSummary", () => {
  it("renders the answered count, attention headline and criterion chips", () => {
    render(<AnswerSheetSummary fields={mixed12} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByText("10 of 12 fields answered")).toBeTruthy();
    expect(screen.getByText("2 fields need attention")).toBeTruthy();
    expect(screen.getByText("1 no answer")).toBeTruthy();
    expect(screen.getByText("1 no evidence")).toBeTruthy();
  });

  it("renders nothing for zero fields", () => {
    const { container } = render(<AnswerSheetSummary fields={[]} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(container.childElementCount).toBe(0);
  });
});

describe("ResponseAnswerSheet", () => {
  it("renders fields grouped under section headings with per-section mini-counts and a trailing Other answers group", () => {
    render(<ResponseAnswerSheet fields={groupedFields} sections={sections} title="Submitted answers" note="Form revision 4." emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getAllByRole("heading", { level: 4 }).map((heading) => heading.textContent)).toEqual(["Branch details", "November 2025 register", "Other answers"]);
    expect(screen.getByText("1 of 2 answered")).toBeTruthy();
    expect(screen.getAllByText("1 of 1 answered")).toHaveLength(1);
    expect(within(groupOf("Branch details")).getByText("Branch")).toBeTruthy();
    expect(within(groupOf("November 2025 register")).getByText("Reporting date")).toBeTruthy();
    expect(within(groupOf("Other answers")).getByText("Directorate")).toBeTruthy();
    expect(within(groupOf("Other answers")).queryByText("Branch")).toBeNull();
    expect(screen.getByRole("heading", { name: "Submitted answers", level: 3 })).toBeTruthy();
    expect(screen.getByText("Form revision 4.")).toBeTruthy();
  });

  it("falls back to a single All answers group when sections are absent", () => {
    render(<ResponseAnswerSheet fields={groupedFields} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByRole("heading", { name: "All answers", level: 4 })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Branch details" })).toBeNull();
    expect(screen.getByText("Branch")).toBeTruthy();
    expect(screen.queryByText("3 of 4 answered")).toBeNull();
  });

  it("renders section headings at the requested heading level", () => {
    const { unmount } = render(<ResponseAnswerSheet fields={groupedFields} sections={sections} headingLevel="h3" emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByRole("heading", { name: "Branch details", level: 3 })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "November 2025 register", level: 3 })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Branch details", level: 4 })).toBeNull();
    unmount();
    render(<ResponseAnswerSheet fields={groupedFields} sections={sections} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByRole("heading", { name: "Branch details", level: 4 })).toBeTruthy();
  });

  it("shows honest summary counts and attention chips only when items need attention", () => {
    render(<ResponseAnswerSheet fields={mixed12} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByText("10 of 12 fields answered")).toBeTruthy();
    expect(screen.getByText("2 fields need attention")).toBeTruthy();
    expect(screen.getByText("1 no answer")).toBeTruthy();
    expect(screen.getByText("1 no evidence")).toBeTruthy();
    expect(screen.queryByText(/poor results?/)).toBeNull();
  });

  it("hides the attention headline and chips when every field is answered", () => {
    render(<ResponseAnswerSheet fields={allAnswered} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByText("3 of 3 fields answered")).toBeTruthy();
    expect(screen.queryByText(/fields need attention/)).toBeNull();
    expect(screen.queryByText(/no answer/)).toBeNull();
    expect(screen.queryByText(/no evidence/)).toBeNull();
  });

  it("adds poor-result chips only when an assessment context is supplied", () => {
    render(<ResponseAnswerSheet fields={assessedFields} assessmentContext={automaticDetail} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByText("12 of 14 fields answered")).toBeTruthy();
    expect(screen.getByText("4 fields need attention")).toBeTruthy();
    expect(screen.getByText("1 no answer")).toBeTruthy();
    expect(screen.getByText("1 no evidence")).toBeTruthy();
    expect(screen.getByText("2 poor results")).toBeTruthy();
  });

  it("switches between all answers and the needs-attention focus view", () => {
    render(<ResponseAnswerSheet
      fields={[answered(fieldItem("branch", "Branch", "short_text")), { field: { id: "owner", label: "Follow-up owner", type: "short_text", required: true }, answer: {} }, { field: { id: "certificate", label: "Certificate", type: "file", required: false }, answer: {} }]}
      emptyLabel="Not provided"
      evidenceEmptyLabel="Not provided"
    />);
    const group = screen.getByRole("group", { name: "Answer view" });
    const allButton = within(group).getByRole("button", { name: "All answers" });
    const attentionButton = within(group).getByRole("button", { name: "Needs attention" });
    expect(allButton.getAttribute("aria-pressed")).toBe("true");
    expect(attentionButton.getAttribute("aria-pressed")).toBe("false");
    expect(screen.getByText("Branch")).toBeTruthy();

    fireEvent.click(attentionButton);
    expect(allButton.getAttribute("aria-pressed")).toBe("false");
    expect(attentionButton.getAttribute("aria-pressed")).toBe("true");
    expect(screen.queryByText("Branch")).toBeNull();
    expect(screen.getByText("Follow-up owner")).toBeTruthy();
    expect(screen.getByText("Certificate")).toBeTruthy();
    expect(screen.getByText("No answer submitted for this field.")).toBeTruthy();
    expect(screen.getByText("No evidence submitted for this field.")).toBeTruthy();

    fireEvent.click(allButton);
    expect(allButton.getAttribute("aria-pressed")).toBe("true");
    expect(screen.getByText("Branch")).toBeTruthy();
  });

  it("shows poor-result reasons with their concern points in the focus view", () => {
    render(<ResponseAnswerSheet fields={assessedFields} assessmentContext={automaticDetail} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    fireEvent.click(screen.getByRole("button", { name: "Needs attention" }));
    expect(screen.getByText("Poor result · 85 concern points")).toBeTruthy();
    expect(screen.getByText("Poor result · 80 concern points")).toBeTruthy();
  });

  it("never classifies POOR without an assessment context", () => {
    render(<ResponseAnswerSheet fields={[{ field: { id: "oversight", label: "Oversight review", type: "yes_no", required: true }, answer: { text: "No" }, decision: { points: 85 } }]} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    const attentionButton = screen.getByRole("button", { name: "Needs attention" }) as HTMLButtonElement;
    expect(attentionButton.disabled).toBe(true);
    expect(screen.getByText("No fields in this response need attention.")).toBeTruthy();
    expect(screen.queryByText(/Poor result/)).toBeNull();
  });

  it("disables the needs-attention switch and explains when nothing needs attention", () => {
    render(<ResponseAnswerSheet fields={allAnswered} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    const attentionButton = screen.getByRole("button", { name: "Needs attention" }) as HTMLButtonElement;
    expect(attentionButton.disabled).toBe(true);
    expect(screen.getByText("No fields in this response need attention.")).toBeTruthy();
  });

  it("renders only the empty copy without a switch or summary for zero fields", () => {
    const { container } = render(<ResponseAnswerSheet fields={[]} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(screen.getByText("No answer fields were recorded for this submitted response.")).toBeTruthy();
    expect(screen.queryByRole("group", { name: "Answer view" })).toBeNull();
    expect(screen.queryByText(/fields answered/)).toBeNull();
    expect(container.querySelector("h3")).toBeNull();
  });

  it("renders excluded fields without ever flagging them", () => {
    render(<ResponseAnswerSheet
      fields={[{ field: { id: "certificate", label: "Certificate", type: "vendor_document", required: false }, answer: {} }, { field: { id: "owner", label: "Owner", type: "short_text", required: false }, answer: {} }]}
      emptyLabel="Not provided"
      evidenceEmptyLabel="Not provided"
      excludeFromAttention={(item) => item.field.id === "certificate"}
    />);
    expect(screen.getByText("Certificate")).toBeTruthy();
    expect(screen.getByText("0 of 2 fields answered")).toBeTruthy();
    expect(screen.getByText("1 field needs attention")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Needs attention" }));
    expect(screen.getByText("Owner")).toBeTruthy();
    expect(screen.queryByText("Certificate")).toBeNull();
  });

  it("treats all-excluded fields as outside attention", () => {
    render(<ResponseAnswerSheet fields={[{ field: { id: "certificate", label: "Certificate", type: "vendor_document", required: false }, answer: {} }]} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" excludeFromAttention={() => true} />);
    expect(screen.getByText("0 of 1 fields answered")).toBeTruthy();
    const attentionButton = screen.getByRole("button", { name: "Needs attention" }) as HTMLButtonElement;
    expect(attentionButton.disabled).toBe(true);
    expect(screen.getByText("No fields in this response need attention.")).toBeTruthy();
  });

  it("replaces the row value with a custom renderValue", () => {
    render(<ResponseAnswerSheet fields={[answered(fieldItem("branch", "Branch", "short_text"), { text: "Lagos" })]} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" renderValue={() => <span>No upload needed.</span>} />);
    expect(screen.getByText("No upload needed.")).toBeTruthy();
    expect(screen.queryByText("Lagos")).toBeNull();
  });

  it("renders an optional title and note", () => {
    const titled = render(<ResponseAnswerSheet fields={[answered(fieldItem("branch", "Branch", "short_text"))]} title="Submitted answers" note="Form revision 4. These answers cannot be changed." emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(titled.container.querySelector("h3")?.textContent).toBe("Submitted answers");
    expect(screen.getByText("Form revision 4. These answers cannot be changed.")).toBeTruthy();
    expect(screen.getByRole("region", { name: "Submitted answers" })).toBeTruthy();
    titled.unmount();

    const untitled = render(<ResponseAnswerSheet fields={[answered(fieldItem("branch", "Branch", "short_text"))]} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    expect(untitled.container.querySelector("h3")).toBeNull();
    expect(untitled.container.querySelector("section")?.getAttribute("aria-label")).toBeNull();
    expect(screen.queryByRole("region")).toBeNull();
  });

  it("passes axe for grouped, single-group and focus renders", async () => {
    const grouped = render(<ResponseAnswerSheet fields={assessedFields} sections={sections} assessmentContext={automaticDetail} title="Submitted answers" note="Form revision 4." emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    await expectNoViolations(grouped.container);
    grouped.unmount();

    const single = render(<ResponseAnswerSheet fields={allAnswered} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" renderValue={() => <span>No upload needed.</span>} />);
    await expectNoViolations(single.container);
    single.unmount();

    const focus = render(<ResponseAnswerSheet fields={assessedFields} assessmentContext={automaticDetail} emptyLabel="Not provided" evidenceEmptyLabel="Not provided" />);
    fireEvent.click(screen.getByRole("button", { name: "Needs attention" }));
    await expectNoViolations(focus.container);
  });
});