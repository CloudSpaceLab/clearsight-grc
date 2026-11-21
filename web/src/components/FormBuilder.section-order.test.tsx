import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FormEditor } from "../v210/FormEditor";

const forms = vi.hoisted(() => ({ loadFormTemplateItem: vi.fn(), createLibraryFormRevision: vi.fn() }));
const api = vi.hoisted(() => ({ loadContext: vi.fn() }));
vi.mock("../formsApi", () => forms);
vi.mock("../api", () => api);

const fields = [
  { id: "q1", section_id: "first", label: "First A", type: "short_text", required: true },
  { id: "q2", section_id: "first", label: "First B", type: "short_text", required: true },
  { id: "q3", section_id: "second", label: "Second A", type: "short_text", required: true },
  { id: "q4", section_id: "second", label: "Second B", type: "short_text", required: true },
];
const item = { template: { id: "form-order", code: "ORDER", version: 1, status: "DRAFT", name: "Ordering review", purpose: "Review ordering", scoring_mode: "NONE", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "first", title: "First section" }, { id: "second", title: "Second section" }], fields }, authority_available: true, operations: [{ command: "forms.template.revise", can_act: true }] };
beforeEach(() => { vi.clearAllMocks(); forms.loadFormTemplateItem.mockResolvedValue(item); forms.createLibraryFormRevision.mockResolvedValue({ id: "form-order", version: 2, status: "DRAFT" }); api.loadContext.mockResolvedValue({ actor: { id: "person", kind: "PERSON" } }); });

it("moves questions only among siblings in the same section", async () => {
  render(<FormEditor id="form-order"/>);
  expect((await screen.findByRole("button", { name: "Move question 3: Second A up" }) as HTMLButtonElement).disabled).toBe(true);
  expect((screen.getByRole("button", { name: "Move question 4: Second B down" }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "Move question 4: Second B up" }));
  expect(document.activeElement).toBe(screen.getByLabelText("Question 3"));
  expect((document.activeElement as HTMLInputElement).value).toBe("Second B");
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await waitFor(() => expect(forms.createLibraryFormRevision).toHaveBeenCalledTimes(1));
  expect(forms.createLibraryFormRevision.mock.calls[0]![2].fields.map((field: { id: string }) => field.id)).toEqual(["q1", "q2", "q4", "q3"]);
});

it("prevents moving a conditional question before its source", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, fields: [fields[0], { ...fields[1], condition: { field_id: "q1", operator: "ANSWERED" } }, ...fields.slice(2)] } });
  render(<FormEditor id="form-order"/>);
  expect((await screen.findByRole("button", { name: "Move question 2: First B up" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.getByRole("button", { name: "Move question 2: First B up" }).getAttribute("aria-describedby")).toBe("v-order-help");
  expect((screen.getByRole("button", { name: "Move question 1: First A down" }) as HTMLButtonElement).disabled).toBe(true);
});

it("preserves section visibility dependencies when moving questions", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, sections: [item.template.sections[0], { ...item.template.sections[1], condition: { field_id: "q1", operator: "ANSWERED" } }], fields: [fields[0], fields[2], fields[1], fields[3]] } });
  render(<FormEditor id="form-order"/>);
  expect((await screen.findByRole("button", { name: "Move question 1: First A down" }) as HTMLButtonElement).disabled).toBe(true);
});

it("reorders noncontiguous section siblings without changing other sections", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, fields: [fields[0], fields[2], fields[1], fields[3]] } });
  render(<FormEditor id="form-order"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Move question 3: First B up" }));
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await waitFor(() => expect(forms.createLibraryFormRevision).toHaveBeenCalledTimes(1));
  expect(forms.createLibraryFormRevision.mock.calls[0]![2].fields.map((field: { id: string }) => field.id)).toEqual(["q2", "q3", "q1", "q4"]);
});

it("retains scored field configuration while changing question order", async () => {
  const scored = fields.map((field, index) => ({ ...field, scoring: { weight: 25, answer_scores: { Yes: index } } }));
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, scoring_mode: "COMPLIANCE", fields: scored } });
  render(<FormEditor id="form-order"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Move question 4: Second B up" }));
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await waitFor(() => expect(forms.createLibraryFormRevision).toHaveBeenCalledTimes(1));
  expect(forms.createLibraryFormRevision.mock.calls[0]![2].fields).toEqual([scored[0], scored[1], scored[3], scored[2]]);
});

it("retains a new choice question's options when moved", async () => {
  render(<FormEditor id="form-order"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add question" }));
  fireEvent.change(screen.getByLabelText("Question 5"), { target: { value: "Select control" } });
  fireEvent.change(screen.getByLabelText("Answer type 5"), { target: { value: "single_select" } });
  fireEvent.change(screen.getByLabelText("Options 5"), { target: { value: "Access\nMonitoring" } });
  fireEvent.click(screen.getByRole("button", { name: "Move question 5: Select control up" }));
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await waitFor(() => expect(forms.createLibraryFormRevision).toHaveBeenCalledTimes(1));
  const saved = forms.createLibraryFormRevision.mock.calls[0]![2].fields;
  expect(saved.find((field: { label: string }) => field.label === "Select control")).toEqual(expect.objectContaining({ section_id: "first", options: ["Access", "Monitoring"] }));
});

it("distinguishes movement controls for questions with the same wording", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, fields: [fields[0], { ...fields[1], label: fields[0].label }] } });
  render(<FormEditor id="form-order"/>);
  expect(await screen.findByRole("button", { name: "Move question 1: First A down" })).toBeTruthy();
  expect(screen.getByRole("button", { name: "Move question 2: First A up" })).toBeTruthy();
});
