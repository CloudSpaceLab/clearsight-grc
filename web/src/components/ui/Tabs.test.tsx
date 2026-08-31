import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { Tabs, type TabItem } from "./index";

const items = [
  { id: "TEMPLATES", label: "Templates" },
  { id: "SENT", label: "Sent forms" },
  { id: "RESPONSES", label: "Responses" },
] as const satisfies readonly TabItem<string>[];

function Harness() {
  const [selected, setSelected] = useState<(typeof items)[number]["id"]>("TEMPLATES");
  return <Tabs ariaLabel="Forms views" items={items} selectedKey={selected} onSelectionChange={setSelected}>
    {(key) => <p>{key === "TEMPLATES" ? "Template library" : key === "SENT" ? "Sent-form distributions" : "Submitted responses"}</p>}
  </Tabs>;
}

describe("Tabs", () => {
  it("owns selected-tab and tab-panel semantics", () => {
    render(<Harness/>);
    expect(screen.getByRole("tab", { name: "Templates" }).getAttribute("aria-selected")).toBe("true");
    expect(screen.getByRole("tabpanel").textContent).toContain("Template library");
  });

  it("uses roving focus with automatic keyboard activation", () => {
    render(<Harness/>);
    const first = screen.getByRole("tab", { name: "Templates" });
    first.focus();
    fireEvent.keyDown(first, { key: "ArrowRight" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "Sent forms" }));
    expect(screen.getByRole("tabpanel").textContent).toContain("Sent-form distributions");

    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "End" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name: "Responses" }));
    expect(screen.getByRole("tabpanel").textContent).toContain("Submitted responses");

    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "Home" });
    expect(document.activeElement).toBe(first);
  });

  it("uses one selected indicator owned by the selected tab", () => {
    render(<Harness/>);
    const selected = screen.getByRole("tab", { name: "Templates" });
    expect(selected.querySelectorAll(".cs-tabs__indicator")).toHaveLength(1);
    expect(document.querySelectorAll(".cs-tabs__indicator")).toHaveLength(1);
  });
});
