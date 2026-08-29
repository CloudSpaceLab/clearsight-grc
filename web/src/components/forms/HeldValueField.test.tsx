import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { CaptureField } from "../../types";
import { HeldValueField } from "./HeldValueField";

const field: CaptureField = {
  id: "registered_address", label: "Registered address", type: "long_text", required: true,
  collection_intent: "CONFIRM_OR_CORRECT",
  record_baseline: { target_key: "VENDOR.IDENTITY.REGISTERED_ADDRESS", subject_type: "VENDOR_RELATIONSHIP", subject_id: "relationship-1", record_id: "vendor-1", record_version: 4, display_value: "12 Marina Road, Lagos", source_label: "Validated vendor record", observed_or_confirmed_at: "2026-08-01T10:00:00Z" },
};

describe("HeldValueField", () => {
  it("confirms the frozen held value without opening the correction editor", () => {
    const onChange = vi.fn();
    render(<HeldValueField field={field} onChange={onChange} editor={<textarea aria-label="Correction"/>}/>);
    expect(screen.getByText("12 Marina Road, Lagos")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Confirm this is accurate" }));
    expect(onChange).toHaveBeenCalledWith({ text: "12 Marina Road, Lagos" });
    expect(screen.queryByLabelText("Correction")).toBeNull();
  });

  it("opens a focused editor for a correction", () => {
    const onChange = vi.fn();
    render(<HeldValueField field={field} value={{ text: "New address" }} onChange={onChange} editor={<textarea aria-label="Correction"/>}/>);
    expect(screen.getByLabelText("Correction")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Keep current record" }));
    expect(onChange).toHaveBeenCalledWith({ text: "12 Marina Road, Lagos" });
  });

  it("opens a focused editor when the respondent chooses to update", () => {
    render(<HeldValueField field={field} onChange={vi.fn()} editor={<textarea aria-label="Correction"/>}/>);
    fireEvent.click(screen.getByRole("button", { name: "Update this information" }));
    expect(screen.getByLabelText("Correction")).toBeTruthy();
  });
});
