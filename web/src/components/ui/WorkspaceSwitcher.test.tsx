import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

const items = [
  { id: "programs", label: "Programs" },
  { id: "vendors", label: "Vendors" },
  { id: "privacy", label: "Processing activities" },
] as const;

describe("WorkspaceSwitcher", () => {
  it("marks the current workspace and navigates without duplicating content state", () => {
    const onChange = vi.fn();
    render(<WorkspaceSwitcher ariaLabel="Portfolio lenses" compactLabel="Portfolio lens" items={items} selectedKey="programs" onSelectionChange={onChange}/>);

    expect(screen.getByRole("button", { name: "Programs" }).getAttribute("aria-current")).toBe("page");
    fireEvent.click(screen.getByRole("button", { name: "Vendors" }));
    expect(onChange).toHaveBeenCalledWith("vendors");
  });

  it("does not emit a navigation change for the selected workspace", () => {
    const onChange = vi.fn();
    render(<WorkspaceSwitcher ariaLabel="Portfolio lenses" compactLabel="Portfolio lens" items={items} selectedKey="programs" onSelectionChange={onChange}/>);

    fireEvent.click(screen.getByRole("button", { name: "Programs" }));
    expect(onChange).not.toHaveBeenCalled();
  });
});
