import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FormCreator } from "./FormCreator";
import { ApiError } from "../http";

it("focuses a new question and navigates between questions from the outline", async () => {
  render(<FormCreator context={context}/>);
  const first = await screen.findByLabelText("Question 1");
  fireEvent.change(first, { target: { value: "Who owns the control?" } });
  fireEvent.click(screen.getByRole("button", { name: "Add question" }));
  expect(document.activeElement).toBe(screen.getByLabelText("Question 2"));
  fireEvent.click(screen.getByRole("button", { name: "Go to question 1: Who owns the control?" }));
  expect(document.activeElement).toBe(first);
});

it("focuses invalid choice options without discarding the question", async () => {
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Access" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Choose controls" } });
  fireEvent.change(screen.getByLabelText("Answer type 1"), { target: { value: "single_select" } });
  fireEvent.change(screen.getByLabelText("Options 1"), { target: { value: "Access\nAccess" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Question 1 needs 2 to 50 distinct options.");
  expect(document.activeElement).toBe(screen.getByLabelText("Options 1"));
  expect((screen.getByLabelText("Question 1") as HTMLInputElement).value).toBe("Choose controls");
});

const form = vi.hoisted(() => ({ createLibraryFormDraft: vi.fn() }));
const api = vi.hoisted(() => ({ resolveAuthority: vi.fn() }));
vi.mock("../formsApi", () => form);
vi.mock("../api", () => api);
const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank" }, actor: { id: "maker", name: "Ada" }, mode: "live" };

beforeEach(() => { vi.clearAllMocks(); api.resolveAuthority.mockResolvedValue({ principal: { id: "maker" } }); });

it("creates a draft with the entered questions and verified authority", async () => {
  form.createLibraryFormDraft.mockResolvedValue({ id: "form-1", name: "Security review", status: "DRAFT", version: 1 });
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Security review" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review service access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Is access reviewed?" } });
  fireEvent.change(screen.getByLabelText("Answer type 1"), { target: { value: "yes_no" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByRole("heading", { name: "Draft created" })).toBeTruthy();
  expect(form.createLibraryFormDraft).toHaveBeenCalledWith(expect.objectContaining({ name: "Security review", purpose: "Review service access", sections: [{ id: "general", title: "Questions" }], fields: [expect.objectContaining({ label: "Is access reviewed?", type: "yes_no", required: true })] }));
  expect(api.resolveAuthority).toHaveBeenCalledWith(expect.objectContaining({ object_type: "LEGAL_ENTITY", object_id: "entity", decision_type: "forms.template.create" }));
});

it("does not offer creation when authority cannot be verified", async () => {
  api.resolveAuthority.mockRejectedValue(new Error("unavailable"));
  render(<FormCreator context={context}/>);
  expect(await screen.findByText("Form creation unavailable. Check your role or try again.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
});

it("does not retry an uncertain creation", async () => {
  form.createLibraryFormDraft.mockRejectedValue(new Error("connection lost"));
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Security review" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Owner?" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByRole("heading", { name: "Draft status unknown" })).toBeTruthy();
  expect(screen.getByText("Owner?")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
});

it("rejects a question label exceeding the backend byte limit", async () => {
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Security review" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "é".repeat(101) } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByText("Question 1 is too long. Shorten the question.")).toBeTruthy();
  expect(form.createLibraryFormDraft).not.toHaveBeenCalled();
});

it("creates a single-choice question with distinct answer options", async () => {
  form.createLibraryFormDraft.mockResolvedValue({ id: "form-1", name: "Security review", status: "DRAFT", version: 1 });
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Security review" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Review frequency?" } });
  fireEvent.change(screen.getByLabelText("Answer type 1"), { target: { value: "single_select" } });
  fireEvent.change(screen.getByLabelText("Options 1"), { target: { value: "Monthly\nQuarterly" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  await screen.findByRole("heading", { name: "Draft created" });
  expect(form.createLibraryFormDraft).toHaveBeenCalledWith(expect.objectContaining({ fields: [expect.objectContaining({ type: "single_select", options: ["Monthly", "Quarterly"] })] }));
});

it("rejects duplicate choice options before saving", async () => {
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Security review" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Review frequency?" } });
  fireEvent.change(screen.getByLabelText("Answer type 1"), { target: { value: "single_select" } });
  fireEvent.change(screen.getByLabelText("Options 1"), { target: { value: "Monthly\nMonthly" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(form.createLibraryFormDraft).not.toHaveBeenCalled();
});

it("retains choice options when the save outcome is unknown", async () => {
  form.createLibraryFormDraft.mockRejectedValue(new Error("connection lost"));
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Security review" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Review frequency?" } });
  fireEvent.change(screen.getByLabelText("Answer type 1"), { target: { value: "single_select" } });
  fireEvent.change(screen.getByLabelText("Options 1"), { target: { value: "Monthly\nQuarterly" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByRole("heading", { name: "Draft status unknown" })).toBeTruthy();
  expect(screen.getByText(/Monthly\s+Quarterly/)).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Save draft" })).toBeNull();
});

it("preserves entered questions after a definite validation rejection", async () => {
  form.createLibraryFormDraft.mockRejectedValue(new ApiError(422, "Check the form details."));
  render(<FormCreator context={context}/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Security review" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
  fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Owner?" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  expect(await screen.findByText("Check the form details.")).toBeTruthy();
  expect((screen.getByLabelText("Question 1") as HTMLInputElement).value).toBe("Owner?");
  expect(screen.getByRole("button", { name: "Save draft" })).toBeTruthy();
});
