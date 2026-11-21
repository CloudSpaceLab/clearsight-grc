import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FormEditor } from "./FormEditor";

const forms = vi.hoisted(() => ({ loadFormTemplateItem: vi.fn(), createLibraryFormRevision: vi.fn() }));
const api = vi.hoisted(() => ({ loadContext: vi.fn() }));
vi.mock("../formsApi", () => forms);
vi.mock("../api", () => api);
const item = { template: { id: "form-1", code: "F-1", version: 3, status: "DRAFT", name: "Access review", purpose: "Review access", scoring_mode: "NONE", sensitivity: "INTERNAL", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "section-1", title: "Access" }], fields: [{ id: "q1", section_id: "section-1", label: "Is access reviewed?", type: "yes_no", required: true, description: "Quarterly" }] }, authority_available: true, operations: [{ command: "forms.template.revise", can_act: true }] };
beforeEach(() => { vi.clearAllMocks(); forms.loadFormTemplateItem.mockResolvedValue(item); api.loadContext.mockResolvedValue({ actor: { id: "person", kind: "PERSON" } }); });

it("navigates the question outline and focuses an added question", async () => {
  render(<FormEditor id="form-1"/>);
  const first = await screen.findByLabelText("Question 1");
  fireEvent.click(screen.getByRole("button", { name: "Add question" }));
  expect(document.activeElement).toBe(screen.getByLabelText("Question 2"));
  fireEvent.click(screen.getByRole("button", { name: "Go to question 1: Is access reviewed?" }));
  expect(document.activeElement).toBe(first);
});

it("creates a versioned revision without dropping question settings", async () => {
  forms.createLibraryFormRevision.mockResolvedValue({ id: "form-1", version: 4, status: "DRAFT" });
  render(<FormEditor id="form-1"/>);
  fireEvent.change(await screen.findByLabelText("Question 1"), { target: { value: "Is access reviewed quarterly?" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByRole("heading", { name: "Draft saved" })).toBeTruthy();
  expect(forms.createLibraryFormRevision).toHaveBeenCalledWith("form-1", 3, expect.objectContaining({ fields: [expect.objectContaining({ id: "q1", label: "Is access reviewed quarterly?", description: "Quarterly", required: true })], sections: item.template.sections }));
});

it("adds a question to a chosen section without changing existing questions", async () => {
  forms.createLibraryFormRevision.mockResolvedValue({ id: "form-1", version: 4, status: "DRAFT" });
  render(<FormEditor id="form-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add question" }));
  fireEvent.change(screen.getByLabelText("Question 2"), { target: { value: "When was access reviewed?" } });
  fireEvent.change(screen.getByLabelText("Answer type 2"), { target: { value: "date" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await screen.findByRole("heading", { name: "Draft saved" });
  const draft = forms.createLibraryFormRevision.mock.calls[0]![2];
  expect(draft.fields[0]).toEqual(item.template.fields[0]);
  expect(draft.fields[1]).toEqual(expect.objectContaining({ section_id: "section-1", label: "When was access reviewed?", type: "date", required: true }));
});

it("adds a multiple-choice question with its options", async () => {
  forms.createLibraryFormRevision.mockResolvedValue({ id: "form-1", version: 4, status: "DRAFT" });
  render(<FormEditor id="form-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Add question" }));
  fireEvent.change(screen.getByLabelText("Question 2"), { target: { value: "Which controls apply?" } });
  fireEvent.change(screen.getByLabelText("Answer type 2"), { target: { value: "multi_select" } });
  fireEvent.change(screen.getByLabelText("Options 2"), { target: { value: "Access\nMonitoring" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await screen.findByRole("heading", { name: "Draft saved" });
  expect(forms.createLibraryFormRevision.mock.calls[0]![2].fields[1]).toEqual(expect.objectContaining({ type: "multi_select", options: ["Access", "Monitoring"] }));
});

it("updates an ordinary question's answer type in a new revision", async () => {
  forms.createLibraryFormRevision.mockResolvedValue({ id: "form-1", version: 4, status: "DRAFT" });
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, fields: [{ ...item.template.fields[0], constraints: {}, collection_intent: "CAPTURE", browser_cache_policy: "ALLOWED", options: ["Yes", "No"] }] } });
  render(<FormEditor id="form-1"/>);
  fireEvent.change(await screen.findByLabelText("Answer type 1"), { target: { value: "date" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await screen.findByRole("heading", { name: "Draft saved" });
  expect(forms.createLibraryFormRevision).toHaveBeenCalledWith("form-1", 3, expect.objectContaining({ fields: [expect.objectContaining({ id: "q1", type: "date", description: "Quarterly" })] }));
  expect(forms.createLibraryFormRevision.mock.calls[0]![2].fields[0].options).toBeUndefined();
  expect(forms.createLibraryFormRevision.mock.calls[0]![2].fields[0].browser_cache_policy).toBe("ALLOWED");
});

it("does not change the type of a question used by conditional rules", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, sections: [{ id: "section-1", title: "Access", condition: { field_id: "q1", operator: "EQUALS", values: ["Yes"] } }] } });
  render(<FormEditor id="form-1"/>);
  await screen.findByLabelText("Question 1");
  expect(screen.queryByLabelText("Answer type 1")).toBeNull();
});

it("does not change question types in a scored form", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, template: { ...item.template, scoring_mode: "COMPLIANCE" } });
  render(<FormEditor id="form-1"/>);
  await screen.findByLabelText("Question 1");
  expect(screen.queryByLabelText("Answer type 1")).toBeNull();
  expect(screen.queryByRole("button", { name: "Add question" })).toBeNull();
});

it("does not offer edits without current server authority", async () => {
  forms.loadFormTemplateItem.mockResolvedValue({ ...item, operations: [{ command: "forms.template.revise", can_act: false }] });
  render(<FormEditor id="form-1"/>);
  expect(await screen.findByRole("heading", { name: "Form editing unavailable" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
});

it("does not offer edits to a service identity", async () => {
  api.loadContext.mockResolvedValue({ actor: { id: "service", kind: "SERVICE" } });
  render(<FormEditor id="form-1"/>);
  expect(await screen.findByRole("heading", { name: "Form editing unavailable" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
});

it("prevents duplicate revisions when a response is uncertain", async () => {
  forms.createLibraryFormRevision.mockRejectedValue(new Error("lost connection"));
  render(<FormEditor id="form-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Save draft" }));
  expect(await screen.findByRole("heading", { name: "Draft status unknown" })).toBeTruthy();
  expect(screen.getByText("Is access reviewed?")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
});
