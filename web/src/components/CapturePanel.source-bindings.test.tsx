import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CaptureRequest } from "../types";
import { CapturePanel } from "./CapturePanel";

const request: CaptureRequest = {
  id: "source-request",
  title: "Confirm branch information",
  purpose: "Keep the resilience review current.",
  why_you: "You manage the branch.",
  status: "READY",
  sensitivity: "INTERNAL",
  estimated_minutes: 2,
  deadline: "2027-08-15T12:00:00Z",
  known_facts: {},
  version: 1,
  fields: [{
    id: "branch",
    label: "Branch name",
    type: "text",
    required: true,
    bindings: [{ binding_id: "branch-binding", binding_version: 3, mode: "PREFILL" }],
    source_resolutions: [{
      mode: "PREFILL",
      binding_id: "branch-binding",
      binding_version: 3,
      binding_name: "Authoritative branch register",
      source_id: "source-1",
      state: "CURRENT",
      value: { kind: "STRING", text: "Enugu Main" },
      receipt: { source_id: "source-1", binding_id: "branch-binding", binding_version: "3", observed_at: "2026-08-15T10:00:00Z", count: 1, completeness: "COMPLETE" },
    }],
  }],
};

describe("CapturePanel connected-source provenance", () => {
  it("prefills the exact source value, exposes its origin, and labels respondent correction", () => {
    render(<CapturePanel request={request} onSubmit={async () => ({ submitted_at: "2026-08-15T12:30:00Z" })}/>);
    const input = screen.getByRole("textbox", { name: /Branch name/ }) as HTMLInputElement;
    expect(input.value).toBe("Enugu Main");
    expect(screen.getByText(/Prefilled from Authoritative branch register/)).toBeTruthy();

    fireEvent.change(input, { target: { value: "Nsukka Branch" } });
    expect(screen.getByText(/Corrected by you · source value was Enugu Main/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Review and submit" }));
    expect(screen.getByText(/Respondent correction · source value: Enugu Main/)).toBeTruthy();
  });
});
