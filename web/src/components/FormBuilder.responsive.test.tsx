import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FormBuilder } from "./FormBuilder";

describe("FormBuilder responsive panes", () => {
  it("keeps the canvas primary and opens Outline and Settings as focused sheets", async () => {
    render(<FormBuilder programID="program-1" onSaved={vi.fn()} onCancel={vi.fn()}/>);

    const panes = screen.getByRole("navigation", { name: "Builder panes" });
    expect(within(panes).getByText("Canvas").getAttribute("aria-current")).toBe("page");

    fireEvent.click(within(panes).getByRole("button", { name: "Outline" }));
    const outline = within(await screen.findByRole("dialog", { name: "Form outline" }));
    expect(outline.getByRole("navigation", { name: "Form outline" })).toBeTruthy();
    fireEvent.click(outline.getByRole("button", { name: "Overview" }));
    expect(screen.queryByRole("dialog", { name: "Form outline" })).toBeNull();

    fireEvent.click(within(panes).getByRole("button", { name: "Settings" }));
    const settings = within(await screen.findByRole("dialog", { name: "Form settings" }));
    expect(settings.getByLabelText("Code")).toBeTruthy();
    fireEvent.click(settings.getByRole("button", { name: "Close form settings" }));
    expect(screen.queryByRole("dialog", { name: "Form settings" })).toBeNull();
  });
});
