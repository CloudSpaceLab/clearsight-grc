import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
// @ts-ignore Vitest executes this CSS source regression in Node.
import { readFileSync } from "node:fs";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createFormTemplate } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { FormBuilder } from "./FormBuilder";

vi.mock("../monitoringApi", () => ({ createFormTemplate: vi.fn() }));

const builderWorkspaceCSS = readFileSync("src/form-builder-workspace.css", "utf8");

const savedForm: FormTemplate = {
  id: "form-1",
  tenant_id: "bank-1",
  code: "VENDOR-DUE-DILIGENCE",
  name: "Vendor due diligence",
  purpose: "Collect information required for the vendor review.",
  scoring_mode: "NONE",
  presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true },
  sections: [{ id: "section_1", title: "Vendor profile" }],
  fields: [],
  status: "DRAFT",
  is_current: false,
  version: 1,
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
};

function selectOverview() {
  fireEvent.click(screen.getByRole("button", { name: "Overview" }));
}

function selectQuestion(index = 0) {
  const questions = screen.getAllByLabelText("Question");
  const question = questions[index];
  if (!question) throw new Error(`Question ${index + 1} is missing`);
  fireEvent.focus(question);
  return question;
}

function selectedResponseType() {
  return screen.getByRole("button", { name: /Inspector response type/ });
}

function chooseSelect(label: RegExp, option: string) {
  fireEvent.click(screen.getByRole("button", { name: label }));
  fireEvent.click(screen.getByRole("option", { name: option }));
}

function completeBase() {
  fireEvent.change(screen.getByLabelText("Form name"), { target: { value: "Vendor due diligence" } });
  fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Collect information required for the vendor review." } });
  selectOverview();
  fireEvent.change(screen.getByLabelText("Code"), { target: { value: "vendor due diligence" } });
  const question = selectQuestion();
  fireEvent.change(question, { target: { value: "Primary contact" } });
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(createFormTemplate).mockResolvedValue(savedForm);
});

describe("FormBuilder", () => {
  it("keeps the complete outline in one sticky scroll pane", () => {
    expect(builderWorkspaceCSS).toMatch(/\.form-builder-outline-shell\s*\{[^}]*position:\s*sticky;[^}]*overflow:\s*auto;/s);
    expect(builderWorkspaceCSS).not.toMatch(/\.form-builder-outline,\s*\.form-builder-inspector\s*\{[^}]*position:\s*sticky;/s);
  });

  it("offers every approved response type without exposing internal field codes", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    const responseType = selectedResponseType();
    fireEvent.click(responseType);
    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual([
      "Short answer", "Long answer", "Email address", "Telephone number", "Web address",
      "Whole number", "Decimal number", "Percentage", "Currency amount", "Date", "Yes or No",
      "Select one", "Select several", "Checkbox", "Attestation", "File", "Photo", "Signature", "Vendor document",
    ]);
    expect(screen.queryByText("short_text")).toBeNull();
  });

  it("shows only limits that apply to the selected response type and uses native dates", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    selectQuestion();
    expect(screen.getByLabelText("Minimum characters")).toBeTruthy();
    chooseSelect(/Inspector response type/, "Date");
    expect(screen.getByLabelText("Earliest date").getAttribute("type")).toBe("date");
    expect(screen.getByLabelText("Latest date").getAttribute("type")).toBe("date");
    chooseSelect(/Inspector response type/, "File");
    expect(screen.getByRole("group", { name: "Accepted files" })).toBeTruthy();
    expect(screen.getByLabelText("Maximum file size (MB)")).toBeTruthy();
    expect(screen.queryByLabelText("Minimum characters")).toBeNull();
  });

  it("deduplicates pasted choices before persistence", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    completeBase();
    chooseSelect(/Inspector response type/, "Select one");
    fireEvent.change(screen.getByLabelText("Choices"), { target: { value: "Nigeria\nGhana\nnigeria\n\nKenya" } });
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
    expect(vi.mocked(createFormTemplate).mock.calls[0]?.[1].fields[0]?.options).toEqual(["Nigeria", "Ghana", "Kenya"]);
  });

  it("duplicates a section with regenerated field keys and rewritten internal conditions", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    completeBase();
    chooseSelect(/Inspector response type/, "Yes or No");
    fireEvent.click(screen.getByRole("button", { name: "+ Question" }));
    const second = selectQuestion(1);
    fireEvent.change(second, { target: { value: "Explain the answer" } });
    fireEvent.click(screen.getByText("Logic"));
    chooseSelect(/Show this question when/, "Primary contact");
    fireEvent.change(screen.getByLabelText("Condition value"), { target: { value: "Yes" } });
    fireEvent.click(screen.getByRole("button", { name: "1Questions" }));
    fireEvent.click(screen.getByText("Section actions"));
    fireEvent.click(screen.getByRole("button", { name: "Duplicate Questions" }));
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
    const input = vi.mocked(createFormTemplate).mock.calls[0]![1];
    expect(input.sections.map((section) => section.id)).toEqual(["section_1", "section_2"]);
    expect(input.fields.map((field) => field.id)).toEqual(["question_1", "question_2", "question_3", "question_4"]);
    expect(input.fields[3]?.condition?.field_id).toBe("question_3");
  });

  it("reorders questions with pointer drag while retaining keyboard move actions", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    completeBase();
    fireEvent.click(screen.getByRole("button", { name: "+ Add question" }));
    const second = selectQuestion(1);
    fireEvent.change(second, { target: { value: "Registration number" } });

    const handle = screen.getByTitle("Drag question 1 to reorder");
    const secondCard = screen.getAllByLabelText("Question")[1]?.closest("article");
    expect(secondCard).not.toBeNull();
    const elementFromPoint = Object.getOwnPropertyDescriptor(document, "elementFromPoint");
    Object.defineProperty(document, "elementFromPoint", { configurable: true, value: vi.fn(() => secondCard) });
    fireEvent.pointerDown(handle, { button: 0, pointerId: 7, clientX: 10, clientY: 10 });
    fireEvent.pointerMove(handle, { pointerId: 7, clientX: 10, clientY: 30 });
    fireEvent.pointerUp(handle, { pointerId: 7, clientX: 10, clientY: 30 });
    if (elementFromPoint) Object.defineProperty(document, "elementFromPoint", elementFromPoint);
    else Reflect.deleteProperty(document, "elementFromPoint");

    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
    expect(vi.mocked(createFormTemplate).mock.calls[0]?.[1].fields.map((field) => field.label)).toEqual([
      "Registration number",
      "Primary contact",
    ]);
    fireEvent.click(screen.getByLabelText("Question 1 actions"));
    expect(screen.getAllByRole("button", { name: "Move down" }).some((button) => !(button as HTMLButtonElement).disabled)).toBe(true);
  });

  it("duplicates a question from the compact actions menu with a regenerated key", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    completeBase();
    fireEvent.click(screen.getByLabelText("Question 1 actions"));
    fireEvent.click(screen.getByRole("button", { name: "Duplicate question" }));
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
    expect(vi.mocked(createFormTemplate).mock.calls[0]?.[1].fields.map(({ id, label }) => ({ id, label }))).toEqual([
      { id: "question_1", label: "Primary contact" },
      { id: "question_2", label: "Primary contact copy" },
    ]);
  });

  it("blocks a reorder that would place a conditional question before its source", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    completeBase();
    chooseSelect(/Inspector response type/, "Yes or No");
    fireEvent.click(screen.getByRole("button", { name: "+ Add question" }));
    const dependent = selectQuestion(1);
    fireEvent.change(dependent, { target: { value: "Explain the answer" } });
    fireEvent.click(screen.getByText("Logic"));
    chooseSelect(/Show this question when/, "Primary contact");
    fireEvent.change(screen.getByLabelText("Condition value"), { target: { value: "Yes" } });

    const handle = screen.getByTitle("Drag question 2 to reorder");
    const firstCard = screen.getAllByLabelText("Question")[0]?.closest("article");
    const elementFromPoint = Object.getOwnPropertyDescriptor(document, "elementFromPoint");
    Object.defineProperty(document, "elementFromPoint", { configurable: true, value: vi.fn(() => firstCard) });
    fireEvent.pointerDown(handle, { button: 0, pointerId: 8, clientX: 10, clientY: 30 });
    fireEvent.pointerMove(handle, { pointerId: 8, clientX: 10, clientY: 10 });
    fireEvent.pointerUp(handle, { pointerId: 8, clientX: 10, clientY: 10 });
    if (elementFromPoint) Object.defineProperty(document, "elementFromPoint", elementFromPoint);
    else Reflect.deleteProperty(document, "elementFromPoint");

    expect(screen.getByRole("alert").textContent).toContain("must remain after the question it depends on");
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
    const fields = vi.mocked(createFormTemplate).mock.calls[0]?.[1].fields ?? [];
    expect(fields.map((field) => field.label)).toEqual(["Primary contact", "Explain the answer"]);
    expect(fields[1]?.condition?.field_id).toBe("question_1");
  });

  it("inserts a section from an exact active template revision with regenerated keys", async () => {
    const reusable: FormTemplate = {
      ...savedForm,
      id: "shared-template",
      name: "Shared vendor controls",
      version: 3,
      status: "ACTIVE",
      is_current: true,
      sections: [{ id: "security", title: "Security" }],
      fields: [
        { id: "mfa", section_id: "security", label: "Is MFA required?", type: "yes_no", required: true, options: ["Yes", "No"] },
        { id: "details", section_id: "security", label: "Describe the exception", type: "long_text", required: false, condition: { field_id: "mfa", operator: "EQUALS", values: ["No"] } },
      ],
    };
    const loadReusableTemplate = vi.fn().mockResolvedValue(reusable);
    render(<FormBuilder
      programID="program-1"
      onSaved={vi.fn()}
      onCancel={vi.fn()}
      reusableTemplates={[{ id: "shared-template", name: "Shared vendor controls", code: "SHARED", version: 3 }]}
      loadReusableTemplate={loadReusableTemplate}
    />);
    completeBase();
    fireEvent.click(screen.getByText("Reuse approved section"));
    chooseSelect(/Active template revision/, "Shared vendor controls · active v3");
    await waitFor(() => expect(loadReusableTemplate).toHaveBeenCalledWith("shared-template", 3));
    fireEvent.click(await screen.findByRole("button", { name: "Insert section" }));
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
    const input = vi.mocked(createFormTemplate).mock.calls[0]![1];
    expect(input.sections[1]).toMatchObject({ id: "section_2", title: "Security" });
    expect(input.fields[1]).toMatchObject({ id: "question_2", section_id: "section_2", label: "Is MFA required?" });
    expect(input.fields[2]?.condition?.field_id).toBe("question_2");
  });

  it("saves incomplete compliance allocation as a draft while blocking approval", async () => {
    const complianceForm: FormTemplate = {
      ...savedForm,
      scoring_mode: "COMPLIANCE",
      sections: [{ id: "identity", title: "Vendor identity", weight: 100 }],
      fields: [{
        id: "registration",
        section_id: "identity",
        label: "Registration verified",
        type: "yes_no",
        required: true,
        options: ["Yes", "No"],
        scoring: { weight: 80, answer_scores: { Yes: 100, No: 0 } },
      }],
    };
    const saveDraft = vi.fn().mockResolvedValue(complianceForm);
    const sendForApproval = vi.fn();
    render(<FormBuilder initialValue={complianceForm} saveDraft={saveDraft} onSendForApproval={sendForApproval} onSaved={vi.fn()} onCancel={vi.fn()} allowIncompleteComplianceDraft/>);
    fireEvent.click(screen.getByRole("button", { name: /Review/ }));
    const allocationIssue = screen.getByText("20% remains to allocate in Vendor identity");
    expect(allocationIssue).toBeTruthy();
    expect((screen.getByRole("button", { name: "Send for approval", hidden: true }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(within(allocationIssue.closest("li")!).getByRole("button", { name: "Fix →" }));
    await waitFor(() => expect(document.activeElement).toBe(screen.getByLabelText("Section 1 title")));
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(saveDraft).toHaveBeenCalledTimes(1));
    expect(sendForApproval).not.toHaveBeenCalled();
  });

  it("adds an explicit required sign-off without making it an implicit contract rule", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    fireEvent.click(screen.getByRole("button", { name: /Review/ }));
    expect(screen.getByText("No required sign-off yet")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Add →" }));
    expect(screen.getByDisplayValue("Required sign-off")).toBeTruthy();
    expect(screen.getByDisplayValue(/I confirm that the information provided/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /Review/ }));
    expect(screen.getByText("Required sign-off included")).toBeTruthy();
  });

  it("persists governed record-target, cache and file limits through the shared contract", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    completeBase();
    chooseSelect(/Inspector response type/, "Vendor document");
    fireEvent.click(screen.getByText(/Data handling/));
    chooseSelect(/Collection purpose/, "Replace a held document");
    fireEvent.change(screen.getByLabelText("Record target key"), { target: { value: "vendor.registration_document" } });
    fireEvent.click(screen.getByText("Technical recovery"));
    chooseSelect(/Browser recovery/, "Do not cache in browser");
    fireEvent.change(screen.getByLabelText("Maximum file size (MB)"), { target: { value: "10" } });
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
    expect(vi.mocked(createFormTemplate).mock.calls[0]?.[1].fields[0]).toMatchObject({
      collection_intent: "REPLACE_HELD_DOCUMENT",
      record_target: { key: "VENDOR.REGISTRATION_DOCUMENT", required_subject_type: "VENDOR_RELATIONSHIP" },
      browser_cache_policy: "NO_BROWSER_CACHE",
      constraints: { max_files: 1, max_file_bytes: 10 * 1024 * 1024 },
    });
  });

  it("saves the exact normalized contract and uses the shared capture renderer", async () => {
    const onSaved = vi.fn();
    render(<FormBuilder programID="program-1" onSaved={onSaved} onCancel={vi.fn()}/>);
    completeBase();
    chooseSelect(/Inspector response type/, "Email address");
    selectOverview();
    chooseSelect(/Default layout/, "Show one section at a time");
    fireEvent.click(screen.getByLabelText("Allow respondents to switch layouts"));
    fireEvent.click(screen.getByRole("button", { name: "Preview" }));
    fireEvent.click(screen.getByRole("button", { name: "Preview Classic" }));
    expect(screen.getByLabelText("Primary contact *").getAttribute("type")).toBe("email");
    fireEvent.click(screen.getByRole("button", { name: "Preview Wizard" }));
    expect(screen.getByText("Step 1 of 1")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Close form preview" }));
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledWith("program-1", expect.objectContaining({
      code: "VENDOR-DUE-DILIGENCE",
      name: "Vendor due diligence",
      purpose: "Collect information required for the vendor review.",
      scoring_mode: "NONE",
      presentation: { default_mode: "WIZARD", allow_mode_switch: true },
      fields: [expect.objectContaining({ type: "email", label: "Primary contact" })],
    })));
    expect(onSaved).toHaveBeenCalledWith(savedForm);
  });

  it("requires an explicit governed handoff before sending an unchanged draft for approval", async () => {
    const saveDraft = vi.fn().mockResolvedValue({ ...savedForm, version: 2 });
    const transition = vi.fn().mockResolvedValue({ ...savedForm, status: "PENDING_APPROVAL" });
    render(<FormBuilder initialValue={{ ...savedForm, fields: [{ id: "q", section_id: "section_1", label: "Question", type: "short_text", required: true }] }} saveDraft={saveDraft} onSendForApproval={transition} onSaved={vi.fn()} onCancel={vi.fn()}/>);

    fireEvent.click(screen.getByRole("button", { name: "Send for approval" }));
    const handoff = within(await screen.findByRole("dialog", { name: "Send form for approval" }));
    expect(handoff.getByText("Ready for independent review")).toBeTruthy();
    expect(handoff.getByText(/This does not activate the form/)).toBeTruthy();
    expect(handoff.getByText("Activation still requires a separate approver.")).toBeTruthy();
    expect(transition).not.toHaveBeenCalled();
    expect(saveDraft).not.toHaveBeenCalled();

    fireEvent.click(handoff.getByRole("button", { name: "Send for approval" }));
    await waitFor(() => expect(transition).toHaveBeenCalledTimes(1));
    expect(saveDraft).not.toHaveBeenCalled();
  });

  it("keeps Save draft as the only primary form action in Program authoring", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    const primaryActions = Array.from(document.querySelectorAll("button.primary-button"));
    expect(primaryActions).toHaveLength(1);
    expect(primaryActions[0]?.textContent).toBe("Save draft");
  });
});
