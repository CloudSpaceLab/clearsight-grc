import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
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
  it("offers an opt-in controlled compact selector without empty or duplicate selection", async () => {
    const change = vi.fn();
    const { rerender } = render(<Tabs ariaLabel="Forms views" compactLabel="Forms section" items={items} selectedKey="TEMPLATES" onSelectionChange={change}>{(key) => <p>{key}</p>}</Tabs>);
    fireEvent.click(screen.getByRole("button", { name: "Templates Forms section" }));
    await screen.findByRole("listbox");
    expect(screen.getAllByRole("option").map((option) => option.textContent)).toEqual(items.map((item) => item.label));
    fireEvent.click(screen.getByRole("option", { name: "Templates" }));
    expect(change).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Templates Forms section" }));
    fireEvent.click(await screen.findByRole("option", { name: "Sent forms" }));
    expect(change).toHaveBeenCalledExactlyOnceWith("SENT");
    rerender(<Tabs ariaLabel="Forms views" compactLabel="Forms section" items={items} selectedKey="SENT" onSelectionChange={change}>{(key) => <p>{key}</p>}</Tabs>);
    expect(screen.getByRole("button", { name: "Sent forms Forms section" })).toBeTruthy();
    expect(screen.getByRole("tab", { name: "Sent forms", selected: true })).toBeTruthy();
  });

  it("retains one editor and a named panel when compact navigation hides the tab list", async () => {
    const { rerender } = render(<Tabs ariaLabel="Forms views" compactLabel="Forms section" items={items} selectedKey="TEMPLATES" onSelectionChange={() => undefined}>{() => <input aria-label="Draft title" defaultValue=""/>}</Tabs>);
    const editor = screen.getByRole("textbox", { name: "Draft title" });
    fireEvent.change(editor, { target: { value: "Unsaved review" } });
    const panel = screen.getByRole("tabpanel", { name: "Templates" });
    screen.getByRole("tablist").style.display = "none";
    expect(screen.queryByRole("tablist")).toBeNull();
    expect(screen.getByRole("button", { name: "Templates Forms section" })).toBeTruthy();
    expect(screen.getByRole("tabpanel", { name: "Templates" })).toBe(panel);
    fireEvent.resize(window);
    rerender(<Tabs ariaLabel="Forms views" compactLabel="Forms section" items={items} selectedKey="TEMPLATES" onSelectionChange={() => undefined}>{() => <input aria-label="Draft title" defaultValue=""/>}</Tabs>);
    await waitFor(() => expect(screen.getAllByRole("textbox")).toHaveLength(1));
    expect(screen.getByRole("textbox")).toBe(editor);
    expect((editor as HTMLInputElement).value).toBe("Unsaved review");
    expect(screen.getByRole("tabpanel", { name: "Templates" })).toBe(panel);
  });

  it("owns selected-tab and tab-panel semantics", () => {
    render(<Harness/>);
    expect(screen.queryByRole("button", { name: /Forms section/ })).toBeNull();
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

  it("keeps the selected tab linked to its mounted panel after switching", () => {
    render(<Harness/>);
    for (const name of ["Sent forms", "Responses", "Templates"]) {
      const tab = screen.getByRole("tab", { name });
      fireEvent.click(tab);
      const panel = screen.getByRole("tabpanel");
      expect(document.getElementById(tab.getAttribute("aria-controls") ?? "")).toBe(panel);
      expect(panel.getAttribute("aria-labelledby")).toBe(tab.id);
    }
  });
});
