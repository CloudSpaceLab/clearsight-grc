import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

const previous = {
  previous_request_id: "request-previous",
  previous_submission_id: "submission-previous",
  previous_submitted_at: "2026-08-14T10:30:00Z",
};

const request: CaptureRequest = {
  id: "request-renewal",
  title: "Confirm current vendor details",
  purpose: "Keep the vendor review current.",
  why_you: "You submitted the previous response.",
  status: "READY",
  sensitivity: "INTERNAL",
  estimated_minutes: 3,
  deadline: "2027-08-15T12:00:00Z",
  known_facts: {},
  version: 1,
  fields: [
    { id: "contact", label: "Vendor contact", type: "text", required: true },
    { id: "review_date", label: "Certification date", type: "date", required: true },
    { id: "status", label: "Certification status", type: "single_select", required: true, options: ["Current", "Expired"] },
    { id: "certificate", label: "Certificate", type: "file", required: false },
    { id: "signature", label: "Confirmation signature", type: "signature", required: false },
  ],
  previous_responses: {
    contact: { ...previous, value: "Ada Okafor" },
    review_date: { ...previous, value: "2026-08-01" },
    status: { ...previous, value: "Current" },
    certificate: { ...previous, value: "artifact-previous" },
    signature: { ...previous, value: "signature-previous" },
  },
};

describe("CapturePanel previous-response provenance", () => {
  it("fills compatible scalar answers and identifies their submission date", () => {
    render(<CapturePanel request={request}/>);

    expect((screen.getByRole("textbox", { name: /Vendor contact/ }) as HTMLInputElement).value).toBe("Ada Okafor");
    expect((screen.getByLabelText(/Certification date/) as HTMLInputElement).value).toBe("2026-08-01");
    expect((screen.getByRole("radio", { name: "Current" }) as HTMLInputElement).checked).toBe(true);
    expect(screen.getAllByText("From the response submitted on 14 Aug 2026")).toHaveLength(3);

    expect(screen.queryByText("artifact-previous")).toBeNull();
    expect(screen.queryByText("signature-previous")).toBeNull();
  });

  it("labels a changed answer and preserves provenance in review", () => {
    render(<CapturePanel request={request}/>);
    const contact = screen.getByRole("textbox", { name: /Vendor contact/ });

    fireEvent.change(contact, { target: { value: "Chidi Nwosu" } });
    expect(screen.getByText("Changed by you · previous response was Ada Okafor")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    expect(screen.getByText("Respondent correction · previous response: Ada Okafor")).toBeTruthy();
    expect(screen.getAllByText("Previous response · submitted 14 Aug 2026")).toHaveLength(2);
  });

  it("keeps a current governed source value ahead of a previous response", () => {
    const sourceRequest: CaptureRequest = {
      ...request,
      fields: [{
        id: "contact",
        label: "Vendor contact",
        type: "text",
        required: true,
        bindings: [{ binding_id: "vendor-register", binding_version: 2, mode: "PREFILL" }],
        source_resolutions: [{
          mode: "PREFILL",
          binding_id: "vendor-register",
          binding_version: 2,
          binding_name: "Vendor register",
          source_id: "source-vendors",
          state: "CURRENT",
          value: { kind: "STRING", text: "Current register contact" },
        }],
      }],
      previous_responses: { contact: { ...previous, value: "Ada Okafor" } },
    };

    render(<CapturePanel request={sourceRequest}/>);
    expect((screen.getByRole("textbox", { name: /Vendor contact/ }) as HTMLInputElement).value).toBe("Current register contact");
    expect(screen.getByText(/Prefilled from Vendor register/)).toBeTruthy();
    expect(screen.queryByText(/response submitted on 14 Aug 2026/)).toBeNull();
  });
});
