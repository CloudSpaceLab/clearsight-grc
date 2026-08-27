import { describe, expect, it } from "vitest";
import type { AuthoringField, AuthoringSection } from "./formAuthoring";
import { duplicateSection, reconcileAuthoringOrder } from "./formAuthoring";

const source: AuthoringField = {
  id: "question_1",
  section_id: "source",
  label: "Source answer",
  type: "yes_no",
  required: true,
  options: ["Yes", "No"],
};

const dependent: AuthoringField = {
  id: "question_2",
  section_id: "dependent",
  label: "Dependent answer",
  type: "short_text",
  required: true,
  condition: { field_id: "question_1", operator: "EQUALS", values: ["Yes"] },
};

it("groups questions by display section order and clears dependencies that become forward references", () => {
  const sections: AuthoringSection[] = [
    { id: "dependent", title: "Dependent", condition: { field_id: "question_1", operator: "EQUALS", values: ["Yes"] } },
    { id: "source", title: "Source" },
  ];
  const reconciled = reconcileAuthoringOrder(sections, [source, dependent]);

  expect(reconciled.fields.map((field) => field.id)).toEqual(["question_2", "question_1"]);
  expect(reconciled.fields[0]?.condition).toBeUndefined();
  expect(reconciled.sections[0]?.condition).toBeUndefined();
});

it("clears field and section conditions when their source question is removed", () => {
  const sections: AuthoringSection[] = [
    { id: "source", title: "Source" },
    { id: "dependent", title: "Dependent", condition: { field_id: "question_1", operator: "ANSWERED" } },
  ];
  const reconciled = reconcileAuthoringOrder(sections, [dependent]);

  expect(reconciled.fields[0]?.condition).toBeUndefined();
  expect(reconciled.sections[1]?.condition).toBeUndefined();
});

describe("section duplication", () => {
  it("keeps persisted scoring data free of editor-only identifiers", () => {
    const field: AuthoringField = {
      ...source,
      scoring: { weight: 100, answer_scores: { Yes: 0, No: 100 }, critical_answers: ["No"] },
    };
    const copied = duplicateSection("source", [{ id: "source", title: "Source" }], [field], 2, 2);

    expect(copied).toBeTruthy();
    expect(copied?.fields[0]?.scoring).toEqual({
      weight: 100,
      answer_scores: { Yes: 0, No: 100 },
      critical_answers: ["No"],
    });
    expect("id" in (copied?.fields[0]?.scoring ?? {})).toBe(false);
  });
});
