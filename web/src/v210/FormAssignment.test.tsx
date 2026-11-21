import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FormAssignment } from "./FormAssignment";

const api = vi.hoisted(() => ({ loadContext: vi.fn(), loadProgramSummaries: vi.fn() }));
const forms = vi.hoisted(() => ({ loadFormTemplateItem: vi.fn() }));
const distributions = vi.hoisted(() => ({ loadRecipientCandidates: vi.fn(), createDistribution: vi.fn(), loadDistributionPage: vi.fn() }));
const http = vi.hoisted(() => ({ requestJSON: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../formsApi", () => forms);
vi.mock("../formsDistributionApi", () => distributions);
vi.mock("../http", () => http);

beforeEach(() => {
  vi.clearAllMocks();
  api.loadContext.mockResolvedValue({ tenant: { id: "bank" }, legal_entity: { id: "entity" }, actor: { id: "owner", kind: "PERSON" } });
  api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "program-1", name: "Bank assurance" } }] });
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-1", name: "Access review" }, active_version: 2, active_status: "ACTIVE" });
  http.requestJSON.mockResolvedValue({ principal: { id: "owner" } });
  distributions.loadRecipientCandidates.mockResolvedValue({ items: [{ principal_id: "person-1", display_name: "Ada Bello", context_label: "Risk" }] });
  distributions.createDistribution.mockResolvedValue({ distribution: { id: "distribution-1", status: "OPEN", form_template_id: "form-1", form_template_version: 2, subject_type: "PROGRAM", subject_id: "program-1" }, recipients: [{ principal_id: "person-1", role: "TO", type: "INTERNAL_PRINCIPAL" }] });
});

async function completeForm() {
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "program-1" } });
  await screen.findByRole("button", { name: "Create assignment" });
  fireEvent.change(screen.getByLabelText("Find respondent"), { target: { value: "Ada" } });
  fireEvent.click(screen.getByRole("button", { name: "Search people" }));
  fireEvent.click(await screen.findByRole("button", { name: /Ada Bello/ }));
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Confirm access reviews" } });
  fireEvent.change(screen.getByLabelText("Response deadline"), { target: { value: "2099-01-01T10:00" } });
}

it("assigns an active form to an exact named respondent and Program", async () => {
  render(<FormAssignment formId="form-1"/>);
  await completeForm();
  fireEvent.click(await screen.findByRole("button", { name: "Create assignment" }));
  expect(await screen.findByRole("heading", { name: "Form request created" })).toBeTruthy();
  const [input] = distributions.createDistribution.mock.calls[0]!;
  expect(input).toMatchObject({ form_template_id: "form-1", form_template_version: 2, subject_type: "PROGRAM", subject_id: "program-1", recipients: [{ role: "TO", type: "INTERNAL_PRINCIPAL", principal_id: "person-1" }] });
  expect(input).not.toHaveProperty("actor_id");
  expect(http.requestJSON).toHaveBeenCalledWith(expect.any(String), "/api/v1/authority/resolve", expect.objectContaining({ method: "POST" }));
  expect(JSON.parse(http.requestJSON.mock.calls[0]![2].body)).toMatchObject({ object_type: "LEGAL_ENTITY", object_id: "entity", decision_type: "forms.distribution.create", responsibility: "ACCOUNTABLE_OWNER" });
});

it("does not create an assignment without a response deadline", async () => {
  render(<FormAssignment formId="form-1"/>);
  await completeForm();
  fireEvent.change(screen.getByLabelText("Response deadline"), { target: { value: "" } });
  fireEvent.click(await screen.findByRole("button", { name: "Create assignment" }));
  expect(distributions.createDistribution).not.toHaveBeenCalled();
});

it("does not offer assignment when authority cannot be confirmed", async () => {
  http.requestJSON.mockResolvedValue({ principal: { id: "someone-else" } });
  render(<FormAssignment formId="form-1"/>);
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "program-1" } });
  expect(await screen.findByText("Assignment unavailable. Check your role or contact an administrator.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Create assignment" })).toBeNull();
});

it("does not search the people directory before assignment authority is confirmed", async () => {
  http.requestJSON.mockResolvedValue({ principal: { id: "someone-else" } });
  render(<FormAssignment formId="form-1"/>);
  fireEvent.change(await screen.findByLabelText("Find respondent"), { target: { value: "Ada" } });
  expect((screen.getByRole("button", { name: "Search people" }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.change(screen.getByLabelText("Program"), { target: { value: "program-1" } });
  expect(await screen.findByText("Assignment unavailable. Check your role or contact an administrator.")).toBeTruthy();
  expect((screen.getByRole("button", { name: "Search people" }) as HTMLButtonElement).disabled).toBe(true);
  expect(distributions.loadRecipientCandidates).not.toHaveBeenCalled();
});

it("discards a pending people search when the selected Program changes", async () => {
  api.loadProgramSummaries.mockResolvedValue({ items: [{ program: { id: "program-1", name: "Bank assurance" } }, { program: { id: "program-2", name: "Risk review" } }] });
  let finish!: (value: unknown) => void;
  distributions.loadRecipientCandidates.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<FormAssignment formId="form-1"/>);
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "program-1" } });
  await screen.findByRole("button", { name: "Create assignment" });
  fireEvent.change(screen.getByLabelText("Find respondent"), { target: { value: "Ada" } });
  fireEvent.click(screen.getByRole("button", { name: "Search people" }));
  fireEvent.change(screen.getByLabelText("Program"), { target: { value: "program-2" } });
  await act(async () => finish({ items: [{ principal_id: "person-1", display_name: "Ada Bello" }] }));
  expect(screen.queryByRole("button", { name: /Ada Bello/ })).toBeNull();
  expect(screen.queryByText(/Respondent:/)).toBeNull();
});

it("clears a selected respondent when searching for another Program", async () => {
  render(<FormAssignment formId="form-1"/>);
  await completeForm();
  expect(screen.getByText("Ada Bello")).toBeTruthy();
  fireEvent.change(screen.getByLabelText("Find a program"), { target: { value: "Risk" } });
  fireEvent.click(screen.getByRole("button", { name: "Search programs" }));
  expect(screen.queryByText("Ada Bello")).toBeNull();
  expect((screen.getByRole("button", { name: "Search people" }) as HTMLButtonElement).disabled).toBe(true);
});

it("prevents duplicate creation when the command outcome is unknown", async () => {
  distributions.createDistribution.mockRejectedValue(new Error("connection lost"));
  distributions.loadDistributionPage.mockResolvedValue({ items: [] });
  render(<FormAssignment formId="form-1"/>);
  await completeForm();
  fireEvent.click(await screen.findByRole("button", { name: "Create assignment" }));
  expect(await screen.findByRole("heading", { name: "Assignment status unknown" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Create assignment" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Check assignments" }));
  await waitFor(() => expect(distributions.loadDistributionPage).toHaveBeenCalledWith({ subject_type: "PROGRAM", subject_id: "program-1", limit: 20 }));
});

it("locks the assignment target while creation is in flight", async () => {
  distributions.createDistribution.mockReturnValue(new Promise(() => {}));
  render(<FormAssignment formId="form-1"/>);
  await completeForm();
  fireEvent.click(await screen.findByRole("button", { name: "Create assignment" }));
  expect((screen.getByLabelText("Program") as HTMLSelectElement).disabled).toBe(true);
  expect((screen.getByLabelText("Find respondent") as HTMLInputElement).disabled).toBe(true);
  expect((screen.getByLabelText("Find a program") as HTMLInputElement).disabled).toBe(true);
});

it("does not show an earlier form's creation result after the selected form changes", async () => {
  let finish!: (value: unknown) => void;
  distributions.createDistribution.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  const view = render(<FormAssignment formId="form-1"/>);
  await completeForm();
  fireEvent.click(await screen.findByRole("button", { name: "Create assignment" }));
  forms.loadFormTemplateItem.mockResolvedValue({ template: { id: "form-2", name: "New form" }, active_version: 3, active_status: "ACTIVE" });
  view.rerender(<FormAssignment formId="form-2"/>);
  expect(await screen.findByRole("heading", { name: "Assign New form" })).toBeTruthy();
  await act(async () => finish({ distribution: { id: "distribution-1", status: "OPEN", form_template_id: "form-1", form_template_version: 2, subject_type: "PROGRAM", subject_id: "program-1" }, recipients: [{ principal_id: "person-1", role: "TO", type: "INTERNAL_PRINCIPAL" }] }));
  expect(screen.queryByRole("heading", { name: "Form request created" })).toBeNull();
  expect(screen.getByRole("heading", { name: "Assign New form" })).toBeTruthy();
});

it("does not offer a respondent from an outdated search", async () => {
  let finish!: (value: unknown) => void;
  distributions.loadRecipientCandidates.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
  render(<FormAssignment formId="form-1"/>);
  fireEvent.change(await screen.findByLabelText("Program"), { target: { value: "program-1" } });
  await screen.findByRole("button", { name: "Create assignment" });
  const search = await screen.findByLabelText("Find respondent");
  fireEvent.change(search, { target: { value: "Ada" } });
  fireEvent.click(screen.getByRole("button", { name: "Search people" }));
  fireEvent.change(search, { target: { value: "Tola" } });
  await act(async () => { finish({ items: [{ principal_id: "person-1", display_name: "Ada Bello" }] }); });
  expect(screen.queryByRole("button", { name: /Ada Bello/ })).toBeNull();
  expect((screen.getByRole("button", { name: "Search people" }) as HTMLButtonElement).disabled).toBe(false);
});


it("does not offer programs from an outdated search", async () => {
  let finish!: (value: unknown) => void;
  render(<FormAssignment formId="form-1"/>);
  const search = await screen.findByLabelText("Find a program");
  api.loadProgramSummaries.mockReturnValueOnce(new Promise((resolve) => { finish = resolve; }));
  fireEvent.change(search, { target: { value: "Bank" } });
  fireEvent.click(screen.getByRole("button", { name: "Search programs" }));
  fireEvent.change(search, { target: { value: "Risk" } });
  await act(async () => { finish({ items: [{ program: { id: "old", name: "Bank assurance" } }] }); });
  expect(screen.queryByRole("option", { name: "Bank assurance" })).toBeNull();
  expect((screen.getByRole("button", { name: "Search programs" }) as HTMLButtonElement).disabled).toBe(false);
});
