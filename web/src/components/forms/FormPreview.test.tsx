import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { CaptureFormContract, CapturePresentationMode } from "../../types";
import { FormPreview } from "./FormPreview";

function contract(mode: CapturePresentationMode, allowSwitch = true): CaptureFormContract {
  return {
    presentation: { default_mode: mode, allow_mode_switch: allowSwitch },
    sections: [{ id: "service", title: "Service" }, { id: "owner", title: "Owner" }],
    fields: [
      { id: "description", section_id: "service", label: "Service description", type: "short_text", required: false },
      { id: "owner", section_id: "owner", label: "Review owner", type: "short_text", required: false },
    ],
  };
}

describe("form preview entry", () => {
  it.each(["AUTOMATIC", "CLASSIC", "WIZARD"] as const)("opens %s with questions and a single layout control", (mode) => {
    render(<FormPreview contract={contract(mode)}/>);
    expect(screen.getByRole("textbox", { name: "Service description" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Preview Classic" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Preview Wizard" })).toBeNull();
    expect(screen.getAllByRole("group", { name: "Question layout" })).toHaveLength(1);
    if (mode === "WIZARD") expect(screen.getByText("Step 1 of 2")).toBeTruthy();
    else expect(screen.getByRole("textbox", { name: "Review owner" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Show one section at a time" }));
    expect(screen.getByText("Step 1 of 2")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Show all questions" }));
    expect(screen.getByRole("textbox", { name: "Review owner" })).toBeTruthy();
  });

  it("respects a fixed respondent layout without hiding the questions", () => {
    render(<FormPreview contract={contract("WIZARD", false)}/>);
    expect(screen.getByRole("textbox", { name: "Service description" })).toBeTruthy();
    expect(screen.getByText("Step 1 of 2")).toBeTruthy();
    expect(screen.queryByRole("group", { name: "Question layout" })).toBeNull();
    expect(screen.queryByRole("group", { name: "Preview layout" })).toBeNull();
  });
});
