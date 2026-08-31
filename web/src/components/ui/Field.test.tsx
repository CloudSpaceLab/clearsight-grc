import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { TextArea, TextField } from "./index";

describe("field contracts", () => {
  it("binds the label, description and current validation message", () => {
    render(<TextField
      label="Accountable owner"
      value=""
      onChange={() => undefined}
      description="Use the person responsible for this distribution."
      errorMessage="Choose an accountable owner before sending."
      isInvalid
      isRequired
    />);

    const input = screen.getByLabelText(/Accountable owner/);
    const describedBy = input.getAttribute("aria-describedby") ?? "";
    expect(input.getAttribute("aria-invalid")).toBe("true");
    expect(describedBy).toContain(screen.getByText("Use the person responsible for this distribution.").id);
    expect(describedBy).toContain(screen.getByRole("alert").id);
  });

  it("keeps read-only and disabled meanings distinct", () => {
    render(<><TextField label="Subject ID" value="vendor-10001" onChange={() => undefined} isReadOnly/><TextField label="Owner" value="" onChange={() => undefined} isDisabled/></>);

    expect((screen.getByLabelText("Subject ID") as HTMLInputElement).readOnly).toBe(true);
    expect((screen.getByLabelText("Subject ID") as HTMLInputElement).disabled).toBe(false);
    expect((screen.getByLabelText("Owner") as HTMLInputElement).disabled).toBe(true);
  });

  it("uses the same anatomy for long-form explanations", () => {
    render(<TextArea label="Reason for changing the distribution" value="" onChange={() => undefined} description="Explain what changed and which recipients are affected."/>);

    expect(screen.getByLabelText("Reason for changing the distribution").tagName).toBe("TEXTAREA");
    expect(screen.getByText(/which recipients are affected/)).toBeTruthy();
  });
});
