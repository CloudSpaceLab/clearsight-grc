import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FocusedSheet } from "./FocusedSheet";

it("keeps focused work outside inert application content and traps keyboard focus", () => {
  const onClose = vi.fn();
  const view = render(<div className="app-shell">
    <aside className="sidebar"><button type="button">Navigation</button></aside>
    <main><button type="button">Workspace action</button></main>
    <FocusedSheet label="Review evidence" onClose={onClose}>
      <button type="button">First action</button>
      <button type="button">Last action</button>
    </FocusedSheet>
  </div>);

  const dialog = screen.getByRole("dialog", { name: "Review evidence" });
  expect(document.body.contains(dialog)).toBe(true);
  expect(dialog.closest(".app-shell")).toBeNull();
  expect(document.querySelector("main")?.hasAttribute("inert")).toBe(true);
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Close" }));

  screen.getByRole("button", { name: "Last action" }).focus();
  fireEvent.keyDown(dialog, { key: "Tab" });
  expect(document.activeElement).toBe(screen.getByRole("button", { name: "Close" }));
  fireEvent.keyDown(dialog, { key: "Escape" });
  expect(onClose).toHaveBeenCalledTimes(1);

  view.unmount();
  expect(document.querySelector(".panel-backdrop")).toBeNull();
});
