import axe from "axe-core";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { SelectField, type SelectOption } from "./index";

const options = [
  { id: "OPEN", label: "Responses open" },
  { id: "LOCKED", label: "Responses locked" },
  { id: "COMPLETED", label: "Completed" },
] as const satisfies readonly SelectOption<string>[];

describe("SelectField", () => {
  it("opens a themed listbox and selects a bounded option", async () => {
    const change = vi.fn();
    render(<SelectField label="Status" placeholder="All states" options={options} onChange={change}/>);
    const trigger = screen.getByRole("button", { name: /Status/ });

    fireEvent.click(trigger);
    expect(await screen.findByRole("listbox")).toBeTruthy();
    fireEvent.click(screen.getByRole("option", { name: "Responses locked" }));

    expect(change).toHaveBeenCalledWith("LOCKED");
    await waitFor(() => expect(document.activeElement).toBe(trigger));
  });

  it("supports keyboard End selection and Escape cancellation", async () => {
    function Harness() {
      const [value, setValue] = useState<string>();
      return <SelectField label="Status" value={value} placeholder="All states" options={options} onChange={setValue}/>;
    }
    render(<Harness/>);
    const trigger = screen.getByRole("button", { name: /Status/ });

    fireEvent.keyDown(trigger, { key: "ArrowDown" });
    const listbox = await screen.findByRole("listbox");
    fireEvent.keyDown(listbox, { key: "End" });
    const completedOption = screen.getByRole("option", { name: "Completed" });
    expect(document.activeElement).toBe(completedOption);
    fireEvent.keyDown(completedOption, { key: "Enter" });
    expect(await screen.findByRole("button", { name: /Completed/ })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Completed/ }));
    fireEvent.keyDown(await screen.findByRole("listbox"), { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("listbox")).toBeNull());
    await waitFor(() => expect(document.activeElement).toBe(trigger));
  });

  it("portals the option list into the fixed workspace overlay root", async () => {
    render(<main data-testid="workspace"><main data-testid="canvas"><SelectField label="Status" placeholder="All states" options={options} onChange={() => undefined}/></main><div id="cs-overlay-root" data-testid="overlay-root"/></main>);
    fireEvent.click(screen.getByRole("button", { name: /Status/ }));
    const listbox = await screen.findByRole("listbox");
    expect(listbox.closest("main")).toBe(screen.getByTestId("workspace"));
    expect(screen.getByTestId("overlay-root").contains(listbox)).toBe(true);
    expect(screen.getByTestId("canvas").contains(listbox)).toBe(false);
  });

  it("keeps a dialog as the option list dismissal boundary", async () => {
    render(<div role="dialog" aria-label="Assignment"><SelectField label="Owner" placeholder="Choose owner" options={options} onChange={() => undefined}/></div>);
    fireEvent.click(screen.getByRole("button", { name: /Owner/ }));
    const listbox = await screen.findByRole("listbox");
    expect(listbox.closest('[role="dialog"]')).toBe(screen.getByRole("dialog", { name: "Assignment" }));
  });

  it("has no representative semantic accessibility violations while open", async () => {
    render(<main><SelectField label="Status" placeholder="All states" options={options} onChange={() => undefined}/><div id="cs-overlay-root"/></main>);
    fireEvent.click(screen.getByRole("button", { name: /Status/ }));
    await screen.findByRole("listbox");

    const results = await axe.run(document.body, { rules: { "color-contrast": { enabled: false } } });
    expect(results.violations.map((violation) => violation.id)).toEqual([]);
  });

  it("keeps the document scroll container stable while the option list is open", async () => {
    render(<SelectField label="Status" placeholder="All states" options={options} onChange={() => undefined}/>);

    fireEvent.click(screen.getByRole("button", { name: /Status/ }));
    await screen.findByRole("listbox");

    expect(document.documentElement.style.overflow).not.toBe("hidden");
    expect(document.documentElement.style.scrollbarGutter).toBe("");
  });

  it("restores an initial positioning scroll without closing a selected option list", async () => {
    let scrollY = 0;
    const originalScrollY = Object.getOwnPropertyDescriptor(window, "scrollY");
    Object.defineProperty(window, "scrollY", { configurable: true, get: () => scrollY });
    const scrollTo = vi.spyOn(window, "scrollTo").mockImplementation((optionsOrX?: ScrollToOptions | number, y?: number) => {
      scrollY = typeof optionsOrX === "number" ? y ?? 0 : optionsOrX?.top ?? 0;
    });

    try {
      render(<main><SelectField label="Response type" value="COMPLETED" placeholder="Choose response type" options={options} onChange={() => undefined}/><div id="cs-overlay-root"/></main>);
      fireEvent.click(screen.getByRole("button", { name: /Completed Response type/ }));
      await screen.findByRole("listbox");

      scrollY = 11;
      fireEvent.scroll(document);

      await waitFor(() => expect(screen.getByRole("listbox")).toBeTruthy());
      expect(scrollTo).toHaveBeenCalledWith(expect.objectContaining({ top: 0 }));
    } finally {
      scrollTo.mockRestore();
      if (originalScrollY) Object.defineProperty(window, "scrollY", originalScrollY);
      else Reflect.deleteProperty(window, "scrollY");
    }
  });

  it("still closes an open option list when the user scrolls after positioning completes", async () => {
    let scrollY = 0;
    let now = 1_000;
    const originalScrollY = Object.getOwnPropertyDescriptor(window, "scrollY");
    Object.defineProperty(window, "scrollY", { configurable: true, get: () => scrollY });
    const performanceNow = vi.spyOn(performance, "now").mockImplementation(() => now);

    try {
      render(<main><SelectField label="Response type" value="COMPLETED" placeholder="Choose response type" options={options} onChange={() => undefined}/><div id="cs-overlay-root"/></main>);
      fireEvent.click(screen.getByRole("button", { name: /Completed Response type/ }));
      await screen.findByRole("listbox");

      now += 500;
      scrollY = 80;
      fireEvent.scroll(document);

      await waitFor(() => expect(screen.queryByRole("listbox")).toBeNull());
    } finally {
      performanceNow.mockRestore();
      if (originalScrollY) Object.defineProperty(window, "scrollY", originalScrollY);
      else Reflect.deleteProperty(window, "scrollY");
    }
  });
});
