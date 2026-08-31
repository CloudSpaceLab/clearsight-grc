import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AdvancedFilterEditor } from "./AdvancedFilterEditor";

describe("AdvancedFilterEditor", () => {
  it("builds only a bounded typed expression and applies it explicitly", () => {
    const onApply = vi.fn();
    render(<AdvancedFilterEditor onApply={onApply} onClose={vi.fn()}/>);

    expect(screen.getByRole("dialog", { name: "Advanced form filters" })).toBeInTheDocument();
    expect(screen.getByLabelText("1 of 12 filter nodes used")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Apply filters" })).toBeDisabled();

    fireEvent.change(screen.getByLabelText("Advanced filter match mode"), { target: { value: "or" } });
    fireEvent.change(screen.getByLabelText("Condition 1 Status value"), { target: { value: "ACTIVE" } });
    fireEvent.click(screen.getByRole("button", { name: "+ Condition" }));
    fireEvent.change(screen.getByLabelText("Condition 2 field"), { target: { value: "tag" } });
    fireEvent.change(screen.getByLabelText("Condition 2 Tag value"), { target: { value: "third-party" } });

    expect(screen.getByLabelText("3 of 12 filter nodes used")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Apply filters" }));
    expect(onApply).toHaveBeenCalledWith({
      kind: "group",
      operator: "or",
      children: [
        { kind: "condition", field: "status", operator: "is", value: "ACTIVE" },
        { kind: "condition", field: "tag", operator: "is", value: "third-party" },
      ],
    });
  });
});
