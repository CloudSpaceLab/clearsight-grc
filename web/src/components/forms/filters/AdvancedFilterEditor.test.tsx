import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AdvancedFilterEditor } from "./AdvancedFilterEditor";

describe("AdvancedFilterEditor", () => {
  it("builds only a bounded typed expression and applies it explicitly", async () => {
    const onApply = vi.fn();
    render(<AdvancedFilterEditor onApply={onApply} onClose={vi.fn()}/>);

    expect(screen.getByRole("dialog", { name: "Advanced form filters" })).toBeTruthy();
    expect(screen.getByLabelText("2 of 12 filter nodes used")).toBeTruthy();
    expect((screen.getByRole("button", { name: "Apply filters" }) as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: /Condition 1 field/ }));
    expect(within(await screen.findByRole("listbox")).getAllByRole("option").map((option) => option.textContent)).toEqual(["Status", "Approved use", "Tag"]);
    fireEvent.keyDown(screen.getByRole("listbox"), { key: "Escape" });

    fireEvent.click(screen.getByRole("button", { name: /Advanced filter match mode/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Any condition" }));
    fireEvent.click(screen.getByRole("button", { name: /Condition 1 Status value/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Active" }));
    fireEvent.click(screen.getByRole("button", { name: "+ Condition" }));
    fireEvent.click(screen.getByRole("button", { name: /Condition 2 field/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Tag" }));
    fireEvent.change(screen.getByLabelText("Condition 2 Tag value"), { target: { value: "third-party" } });

    expect(screen.getByLabelText("3 of 12 filter nodes used")).toBeTruthy();
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

  it("preserves a legacy opaque filter without exposing its internal identifier", async () => {
    render(<AdvancedFilterEditor
      expression={{ kind: "condition", field: "owner", operator: "is", value: "principal-secret-id" }}
      onApply={vi.fn()}
      onClose={vi.fn()}
    />);

    expect(screen.getByLabelText("Condition 1 Owner value").textContent).toBe("Selected owner");
    expect(document.body.textContent).not.toContain("principal-secret-id");
    fireEvent.click(screen.getByRole("button", { name: /Condition 1 field/ }));
    expect(within(await screen.findByRole("listbox")).getAllByRole("option").map((option) => option.textContent)).toEqual([
      "Owner",
      "Status",
      "Approved use",
      "Tag",
    ]);
  });
});
