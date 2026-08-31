import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CheckboxField, TextArea, TextField } from "./index";

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

  it("keeps numeric constraints in the shared labelled field anatomy", () => {
    render(<TextField label="Maximum files" type="number" value="5" min={1} max={10} step={1} onChange={() => undefined}/>);

    const input = screen.getByLabelText("Maximum files") as HTMLInputElement;
    expect(input.type).toBe("number");
    expect(input.min).toBe("1");
    expect(input.max).toBe("10");
    expect(input.step).toBe("1");
  });

  it("provides one labelled checkbox contract with description and disabled semantics", () => {
    const changed = vi.fn();
    const { rerender } = render(<CheckboxField label="Select vendor review" description="Adds this draft to the bulk action." isSelected={false} onChange={changed}/>);

    const checkbox = screen.getByRole("checkbox", { name: "Select vendor review" });
    fireEvent.click(checkbox);
    expect(changed).toHaveBeenCalledWith(true);
    expect(checkbox.getAttribute("aria-describedby")).toBe(screen.getByText("Adds this draft to the bulk action.").id);

    rerender(<CheckboxField label="Select vendor review" isSelected={false} isDisabled onChange={changed}/>);
    const disabled = screen.getByRole("checkbox", { name: "Select vendor review" });
    expect((disabled as HTMLInputElement).disabled).toBe(true);
  });
});
