import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import { FocusedSheet } from "./FocusedSheet";

it("keeps the legacy import compatible with the shared labelled dialog", () => {
  render(<FocusedSheet label="Review evidence" onClose={() => undefined}><p>Evidence detail</p></FocusedSheet>);
  const dialog = screen.getByRole("dialog", { name: "Review evidence" });
  expect(document.body.contains(dialog)).toBe(true);
  expect(dialog.textContent).toContain("Evidence detail");
});
