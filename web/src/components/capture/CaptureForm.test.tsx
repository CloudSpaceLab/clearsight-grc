import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import type { CaptureAnswerValue, CaptureAnswers, CaptureFormContract, CapturePresentationMode } from "../../types";
import { CaptureForm } from "./CaptureForm";
import { CaptureWorkspaceRecoveryProvider, type CaptureWorkspaceRecoveryUI } from "./CaptureWorkspaceRecoveryContext";

function Harness({
  contract,
  initialAnswers = {},
  initialMode = "CLASSIC",
  onReview = vi.fn(),
  onBeforeSectionNavigation,
  recoveryUI,
}: {
  contract: CaptureFormContract;
  initialAnswers?: CaptureAnswers;
  initialMode?: CapturePresentationMode;
  onReview?: () => void;
  onBeforeSectionNavigation?: () => Promise<boolean>;
  recoveryUI?: CaptureWorkspaceRecoveryUI;
}) {
  const [answers, setAnswers] = useState(initialAnswers);
  const [mode, setMode] = useState(initialMode);
  const form = <CaptureForm
    contract={contract}
    answers={answers}
    attachments={{}}
    mode={mode}
    external
    uploadingField={null}
    onAnswer={(fieldID: string, value: CaptureAnswerValue) => setAnswers((current) => ({ ...current, [fieldID]: value }))}
    onUpload={vi.fn()}
    onRemoveAttachment={vi.fn()}
    onModeChange={setMode}
    onBeforeSectionNavigation={onBeforeSectionNavigation}
    onReview={onReview}
  />;
  return recoveryUI ? <CaptureWorkspaceRecoveryProvider value={recoveryUI}>{form}</CaptureWorkspaceRecoveryProvider> : form;
}

describe("CaptureForm", () => {
  it("uses native or purpose-built controls and applies the relevant limits", () => {
    const contract: CaptureFormContract = {
      presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
      sections: [{ id: "company", title: "Company details" }],
      fields: [
        { id: "name", section_id: "company", label: "Registered name", type: "short_text", required: true, constraints: { min_length: 2, max_length: 120 } },
        { id: "summary", section_id: "company", label: "Service description", type: "long_text", required: false, constraints: { max_length: 600 } },
        { id: "email", section_id: "company", label: "Security contact email", type: "email", required: true },
        { id: "phone", section_id: "company", label: "Security contact phone", type: "telephone", required: false },
        { id: "website", section_id: "company", label: "Company website", type: "url", required: false },
        { id: "staff", section_id: "company", label: "Security staff", type: "integer", required: true, constraints: { minimum: 1, maximum: 500, step: 1 } },
        { id: "loss", section_id: "company", label: "Expected loss", type: "decimal", required: false, constraints: { maximum: 1000000, step: 0.01, decimal_precision: 2 } },
        { id: "coverage", section_id: "company", label: "Control coverage", type: "percentage", required: true, constraints: { minimum: 0, maximum: 100, step: 0.1 } },
        { id: "spend", section_id: "company", label: "Annual spend", type: "currency", required: true, constraints: { currency: "NGN", minimum: 0, step: 0.01 } },
        { id: "expiry", section_id: "company", label: "Certificate expiry", type: "date", required: true, constraints: { min_date: "2026-08-26", max_date: "2030-08-26" } },
        { id: "handles_data", section_id: "company", label: "Handles customer data", type: "yes_no", required: true },
        { id: "tier", section_id: "company", label: "Service tier", type: "single_select", required: true, options: ["Critical", "Important", "Standard", "Limited", "Other"] },
        { id: "regions", section_id: "company", label: "Processing regions", type: "multi_select", required: true, options: ["Nigeria", "Ghana", "Kenya"], constraints: { min_selections: 1, max_selections: 2 } },
        { id: "confirmed", section_id: "company", label: "Details confirmed", type: "checkbox", required: true },
        { id: "attest", section_id: "company", label: "Authorized response", type: "attestation", required: true, attestation: "I confirm that I am authorized to submit this response." },
        { id: "policy", section_id: "company", label: "Security policy", type: "file", required: false, accepted_formats: ["application/pdf"], constraints: { max_files: 1, max_file_bytes: 5000000 } },
        { id: "photo", section_id: "company", label: "Data centre photo", type: "photo", required: false, accepted_formats: ["image/jpeg"] },
        { id: "signature", section_id: "company", label: "Authorized signature", type: "signature", required: true },
        { id: "certificate", section_id: "company", label: "ISO certificate", type: "vendor_document", required: true, accepted_formats: ["application/pdf"] },
      ],
    };

    render(<Harness contract={contract}/>);
    expect(screen.getByLabelText(/Registered name/).getAttribute("minlength")).toBe("2");
    expect(screen.getByLabelText(/Registered name/).getAttribute("maxlength")).toBe("120");
    expect(screen.getByLabelText("Service description").tagName).toBe("TEXTAREA");
    expect(screen.getByLabelText(/Security contact email/).getAttribute("type")).toBe("email");
    expect(screen.getByLabelText(/Security contact phone/).getAttribute("type")).toBe("tel");
    expect(screen.getByLabelText(/Company website/).getAttribute("type")).toBe("url");
    expect(screen.getByRole("spinbutton", { name: /Security staff/ }).getAttribute("step")).toBe("1");
    expect(screen.getByRole("spinbutton", { name: /Expected loss/ }).getAttribute("max")).toBe("1000000");
    expect(screen.getByRole("spinbutton", { name: /Control coverage/ }).getAttribute("max")).toBe("100");
    const spend = screen.getByRole("spinbutton", { name: /Annual spend/ });
    expect(document.getElementById(spend.getAttribute("aria-describedby") ?? "")?.textContent).toMatch(/NGN/);
    expect(screen.getByLabelText(/Certificate expiry/).getAttribute("min")).toBe("2026-08-26");
    expect(screen.getByRole("radio", { name: "Yes" })).toBeTruthy();
    expect(screen.getByRole("combobox", { name: /Service tier/ })).toBeTruthy();
    expect(screen.getByRole("group", { name: /Processing regions/ }).textContent).toMatch(/Select 1 to 2/);
    expect(screen.getByRole("checkbox", { name: /Details confirmed/ })).toBeTruthy();
    expect(screen.getByRole("checkbox", { name: /I confirm that I am authorized/ })).toBeTruthy();
    expect(screen.getByLabelText(/Security policy/).getAttribute("accept")).toBe("application/pdf");
    expect(screen.getByLabelText(/Data centre photo/).getAttribute("capture")).toBe("environment");
    expect(screen.getByRole("button", { name: "Add signature" })).toBeTruthy();
    expect(screen.getByLabelText(/ISO certificate file/).getAttribute("accept")).toBe("application/pdf");
    expect(screen.getByLabelText(/Document type/)).toBeTruthy();
    expect(screen.getByLabelText(/Expiry date/)).toBeTruthy();
  });

  it("matches every supported visibility operator for fields and sections", () => {
    const contract: CaptureFormContract = {
      presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
      sections: [
        { id: "main", title: "Main" },
        { id: "conditional", title: "Conditional section", condition: { field_id: "trigger", operator: "IN", values: ["A"] } },
      ],
      fields: [
        { id: "trigger", section_id: "main", label: "Trigger", type: "multi_select", required: false, options: ["A", "B"] },
        { id: "equals", section_id: "main", label: "Equals field", type: "short_text", required: false, condition: { field_id: "trigger", operator: "EQUALS", values: ["A"] } },
        { id: "not_equals", section_id: "main", label: "Not equals field", type: "short_text", required: false, condition: { field_id: "trigger", operator: "NOT_EQUALS", values: ["Z"] } },
        { id: "in", section_id: "main", label: "In field", type: "short_text", required: false, condition: { field_id: "trigger", operator: "IN", values: ["B", "C"] } },
        { id: "not_in", section_id: "main", label: "Not in field", type: "short_text", required: false, condition: { field_id: "trigger", operator: "NOT_IN", values: ["Z"] } },
        { id: "answered", section_id: "main", label: "Answered field", type: "short_text", required: false, condition: { field_id: "trigger", operator: "ANSWERED" } },
        { id: "section_field", section_id: "conditional", label: "Conditional section field", type: "short_text", required: false },
      ],
    };
    render(<Harness contract={contract} initialAnswers={{ trigger: { values: ["A", "B"] } }}/>);
    for (const label of ["Equals field", "Not equals field", "In field", "Not in field", "Answered field", "Conditional section field"]) expect(screen.getByRole("textbox", { name: label })).toBeTruthy();
  });

  it("shows Classic section links and lets the respondent switch modes without losing answers", () => {
    const contract = wizardContract(true);
    render(<Harness contract={contract}/>);
    expect(screen.getByRole("navigation", { name: "Request sections" })).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", { name: /Registered name/ }), { target: { value: "Acme Processing Limited" } });
    fireEvent.click(screen.getByRole("button", { name: "Show one section at a time" }));
    expect(screen.getByText("Step 1 of 2")).toBeTruthy();
    expect((screen.getByRole("textbox", { name: /Registered name/ }) as HTMLInputElement).value).toBe("Acme Processing Limited");
  });

  it("links an accessible error summary, preserves Back navigation, and requires review after the last Wizard section", async () => {
    const onReview = vi.fn();
    render(<Harness contract={wizardContract()} initialMode="WIZARD" onReview={onReview}/>);
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByRole("alert").textContent).toContain("Registered name is required");
    fireEvent.change(screen.getByRole("textbox", { name: /Registered name/ }), { target: { value: "Acme Processing Limited" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(screen.getByText("Step 2 of 2")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect((screen.getByRole("textbox", { name: /Registered name/ }) as HTMLInputElement).value).toBe("Acme Processing Limited");
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    fireEvent.change(screen.getByRole("textbox", { name: /Evidence note/ }), { target: { value: "Policy attached separately" } });
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    expect(onReview).toHaveBeenCalledTimes(1);
  });

  it("waits for a successful draft save before moving in either Wizard direction", async () => {
    const save = vi.fn().mockResolvedValue(true);
    render(<Harness contract={wizardContract()} initialMode="WIZARD" onBeforeSectionNavigation={save}/>);
    fireEvent.change(screen.getByRole("textbox", { name: /Registered name/ }), { target: { value: "Acme Processing Limited" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    await screen.findByText("Step 2 of 2");
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    await screen.findByText("Step 1 of 2");
    expect(save).toHaveBeenCalledTimes(2);
  });

  it("keeps the current Wizard section and values when draft saving fails", async () => {
    render(<Harness contract={wizardContract()} initialMode="WIZARD" onBeforeSectionNavigation={vi.fn().mockResolvedValue(false)}/>);
    const name = screen.getByRole("textbox", { name: /Registered name/ }) as HTMLInputElement;
    fireEvent.change(name, { target: { value: "Acme Processing Limited" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    await waitFor(() => expect(screen.getByText("Step 1 of 2")).toBeTruthy());
    expect(name.value).toBe("Acme Processing Limited");
  });

  it("restores the encrypted Wizard page and records both forward and back page changes", async () => {
    const onPageChange = vi.fn();
    const save = vi.fn().mockResolvedValue(true);
    render(<Harness
      contract={wizardContract()}
      initialMode="WIZARD"
      initialAnswers={{ note: { text: "Recovered note" } }}
      onBeforeSectionNavigation={save}
      recoveryUI={{ initialPage: 1, filesToReselect: [], onPageChange }}
    />);

    expect(screen.getByText("Step 2 of 2")).toBeTruthy();
    expect((screen.getByRole("textbox", { name: /Evidence note/ }) as HTMLTextAreaElement).value).toBe("Recovered note");
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    await screen.findByText("Step 1 of 2");
    expect(onPageChange).toHaveBeenCalledWith(0);
  });

  it("shows the exact file reselection instruction beside an unsynced recovered file", () => {
    const contract: CaptureFormContract = {
      presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
      sections: [{ id: "evidence", title: "Evidence" }],
      fields: [{ id: "proof", section_id: "evidence", label: "Proof", type: "file", required: true }],
    };
    render(<Harness contract={contract} recoveryUI={{ initialPage: 0, filesToReselect: ["proof"], onPageChange: vi.fn() }}/>);
    expect(screen.getByText("Reselect file to upload")).toBeTruthy();
  });

  it("renders recovered text as inert input content rather than executable HTML", () => {
    const contract: CaptureFormContract = {
      presentation: { default_mode: "CLASSIC", allow_mode_switch: false },
      sections: [{ id: "main", title: "Main" }],
      fields: [{ id: "name", section_id: "main", label: "Registered name", type: "short_text", required: false }],
    };
    const hostile = '<img src="x" onerror="globalThis.__captureXss = true">';
    render(<Harness contract={contract} initialAnswers={{ name: { text: hostile } }}/>);
    expect((screen.getByRole("textbox", { name: /Registered name/ }) as HTMLInputElement).value).toBe(hostile);
    expect(document.querySelector('img[src="x"]')).toBeNull();
  });
});

function wizardContract(allowModeSwitch = false): CaptureFormContract {
  return {
    presentation: { default_mode: allowModeSwitch ? "CLASSIC" : "WIZARD", allow_mode_switch: allowModeSwitch },
    sections: [{ id: "company", title: "Company" }, { id: "evidence", title: "Evidence" }],
    fields: [
      { id: "name", section_id: "company", label: "Registered name", type: "short_text", required: true },
      { id: "note", section_id: "evidence", label: "Evidence note", type: "long_text", required: false },
    ],
  };
}
