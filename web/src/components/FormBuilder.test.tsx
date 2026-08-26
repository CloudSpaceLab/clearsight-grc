import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createFormTemplate } from "../monitoringApi";
import type { FormTemplate } from "../monitoringTypes";
import { FormBuilder } from "./FormBuilder";

vi.mock("../monitoringApi", () => ({ createFormTemplate: vi.fn() }));

const savedForm: FormTemplate = {
  id: "form-1",
  tenant_id: "bank-1",
  code: "VENDOR-DUE-DILIGENCE",
  name: "Vendor due diligence",
  purpose: "Collect information required for the vendor review.",
  presentation: { default_mode: "AUTOMATIC", allow_mode_switch: true },
  sections: [{ id: "section_1", title: "Vendor profile" }],
  fields: [],
  status: "DRAFT",
  is_current: false,
  version: 1,
  created_at: "2026-08-26T00:00:00Z",
  updated_at: "2026-08-26T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(createFormTemplate).mockResolvedValue(savedForm);
});

describe("FormBuilder", () => {
  it("offers every approved response type without exposing internal field codes", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);

    const responseType = screen.getByLabelText("Response type");
    expect(within(responseType).getAllByRole("option").map((option) => option.textContent)).toEqual([
      "Short answer", "Long answer", "Email address", "Telephone number", "Web address",
      "Whole number", "Decimal number", "Percentage", "Currency amount", "Date", "Yes or No",
      "Select one", "Select several", "Checkbox", "Attestation", "File", "Photo", "Signature", "Vendor document",
    ]);
    expect(screen.queryByText("short_text")).toBeNull();
  });

  it("shows only the limits that apply to the selected response type", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);

    expect(screen.getByRole("group", { name: "Response limits" })).toBeTruthy();
    expect(screen.getByLabelText("Minimum characters")).toBeTruthy();
    expect(screen.queryByRole("group", { name: "Accepted files" })).toBeNull();

    fireEvent.change(screen.getByLabelText("Response type"), { target: { value: "file" } });

    expect(screen.getByRole("group", { name: "Accepted files" })).toBeTruthy();
    expect(screen.getByLabelText("Maximum files")).toBeTruthy();
    expect(screen.getByLabelText("Maximum file size (MB)")).toBeTruthy();
    expect(screen.queryByLabelText("Minimum characters")).toBeNull();
  });

  it("adds and reorders sections while preserving their assigned questions", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);

    fireEvent.change(screen.getByLabelText("Section title"), { target: { value: "Vendor profile" } });
    fireEvent.click(screen.getByRole("button", { name: "Add section" }));
    const sectionTitles = screen.getAllByLabelText("Section title");
    fireEvent.change(sectionTitles[1]!, { target: { value: "Information security" } });
    fireEvent.click(screen.getByRole("button", { name: "Move Information security up" }));

    expect(screen.getAllByLabelText("Section title").map((input) => (input as HTMLInputElement).value)).toEqual([
      "Information security", "Vendor profile",
    ]);
  });

  it("limits conditions to earlier questions and saves the shared form contract", async () => {
    const onSaved = vi.fn();
    render(<FormBuilder programID="program-1" onSaved={onSaved} onCancel={vi.fn()}/>);

    fireEvent.change(screen.getByLabelText("Form name"), { target: { value: "Vendor due diligence" } });
    fireEvent.change(screen.getByLabelText("Code"), { target: { value: "vendor due diligence" } });
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Collect information required for the vendor review." } });
    fireEvent.change(screen.getByLabelText("Section title"), { target: { value: "Vendor profile" } });
    fireEvent.change(screen.getByLabelText("Question"), { target: { value: "Does the vendor hold cyber insurance?" } });
    fireEvent.click(screen.getByRole("button", { name: "Add Yes/No question" }));
    const questions = screen.getAllByLabelText("Question");
    fireEvent.change(questions[1]!, { target: { value: "What is the policy number?" } });

    const conditions = screen.getAllByLabelText("Show this question when") as HTMLSelectElement[];
    expect(within(conditions[0]!).getAllByRole("option").map((option) => option.textContent)).toEqual(["Always shown"]);
    expect(within(conditions[1]!).getByRole("option", { name: "Does the vendor hold cyber insurance?" })).toBeTruthy();
    fireEvent.change(conditions[1]!, { target: { value: "question_1" } });
    fireEvent.change(screen.getByLabelText("Condition value"), { target: { value: "Yes" } });
    fireEvent.change(screen.getByLabelText("Default layout"), { target: { value: "WIZARD" } });
    fireEvent.click(screen.getByLabelText("Allow respondents to switch layouts"));
    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledWith("program-1", {
      code: "VENDOR-DUE-DILIGENCE",
      name: "Vendor due diligence",
      purpose: "Collect information required for the vendor review.",
      presentation: { default_mode: "WIZARD", allow_mode_switch: true },
      sections: [{ id: "section_1", title: "Vendor profile" }],
      fields: [
        expect.objectContaining({ id: "question_1", section_id: "section_1", type: "short_text", label: "Does the vendor hold cyber insurance?" }),
        expect.objectContaining({ id: "question_2", section_id: "section_1", type: "yes_no", options: ["Yes", "No"], condition: { field_id: "question_1", operator: "EQUALS", values: ["Yes"] } }),
      ],
    }));
    expect(onSaved).toHaveBeenCalledWith(savedForm);
  });

  it("uses the shared capture renderer for Classic and Wizard previews", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);

    fireEvent.change(screen.getByLabelText("Question"), { target: { value: "Primary contact email" } });
    fireEvent.change(screen.getByLabelText("Response type"), { target: { value: "email" } });
    fireEvent.click(screen.getByRole("button", { name: "Preview Classic" }));

    expect(screen.getByRole("heading", { name: "Questions" })).toBeTruthy();
    expect(screen.getByLabelText("Primary contact email *").getAttribute("type")).toBe("email");

    fireEvent.click(screen.getByRole("button", { name: "Preview Wizard" }));
    expect(screen.getByText("Step 1 of 1")).toBeTruthy();
  });

  it("saves the form draft while its unanswered preview is open", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);
    fireEvent.change(screen.getByLabelText("Form name"), { target: { value: "Vendor due diligence" } });
    fireEvent.change(screen.getByLabelText("Code"), { target: { value: "VENDOR-DUE-DILIGENCE" } });
    fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Collect information required for the vendor review." } });
    fireEvent.change(screen.getByLabelText("Question"), { target: { value: "Primary contact email" } });
    fireEvent.click(screen.getByRole("button", { name: "Preview Classic" }));

    fireEvent.click(screen.getByRole("button", { name: "Save draft" }));

    await waitFor(() => expect(createFormTemplate).toHaveBeenCalledTimes(1));
  });

  it("keeps Save draft as the only primary form action", () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);

    const primaryActions = Array.from(document.querySelectorAll("button.primary-button"));
    expect(primaryActions).toHaveLength(1);
    expect(primaryActions[0]?.textContent).toBe("Save draft");
  });
});
