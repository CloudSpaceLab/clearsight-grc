import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FocusedDialog } from "./FocusedDialog";

describe("FocusedDialog", () => {
  it("renders a centered modal contract and closes from the labelled control", () => {
    const onClose = vi.fn();
    render(<FocusedDialog label="Create a form" onClose={onClose}><p>Creation choices</p></FocusedDialog>);

    const dialog = screen.getByRole("dialog", { name: "Create a form" });
    expect(dialog.closest(".cs-dialog")?.classList.contains("cs-dialog--default")).toBe(true);
    const overlay = dialog.closest(".cs-dialog__overlay");
    expect(overlay?.classList.contains("panel-backdrop")).toBe(false);
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
