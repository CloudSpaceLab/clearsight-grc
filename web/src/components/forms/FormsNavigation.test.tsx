import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { FormsNavigation, type FormsTab } from "./FormsNavigation";

describe("FormsNavigation", () => {
  it("uses one automatically activated tab contract for all five peer sections", () => {
    const changed = vi.fn();
    function Harness() {
      const [active, setActive] = useState<FormsTab>("Templates");
      return <FormsNavigation activeTab={active} onChange={(tab) => { changed(tab); setActive(tab); }}><p>{active} workspace</p></FormsNavigation>;
    }
    render(<Harness/>);
    expect(screen.getAllByRole("tab")).toHaveLength(5);
    const templates = screen.getByRole("tab", { name: "Templates" });
    templates.focus();
    fireEvent.keyDown(templates, { key: "ArrowRight" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "Sent forms" }));
    expect(changed).toHaveBeenCalledWith("Sent forms");
    expect(document.querySelectorAll(".cs-tabs__indicator")).toHaveLength(1);
    expect(screen.getByRole("tabpanel").textContent).toContain("Sent forms workspace");
  });
});
