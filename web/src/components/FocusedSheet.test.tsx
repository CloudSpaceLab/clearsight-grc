import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FocusedSheet } from "../v210/FocusedSheet";

it("presents a labelled modal sheet and restores body scrolling on close", () => {
  const showModal = vi.fn(function (this: HTMLDialogElement) { this.setAttribute("open", ""); });
  const close = vi.fn(function (this: HTMLDialogElement) { this.removeAttribute("open"); });
  const originalShow = Object.getOwnPropertyDescriptor(HTMLDialogElement.prototype, "showModal");
  const originalClose = Object.getOwnPropertyDescriptor(HTMLDialogElement.prototype, "close");
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: showModal });
  Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: close });
  try {
  const onDismiss = vi.fn();
  const { unmount } = render(<FocusedSheet title="Review evidence" onDismiss={onDismiss}><p>Evidence detail</p></FocusedSheet>);
  const dialog = screen.getByRole("dialog", { name: "Review evidence" });
  expect(document.body.contains(dialog)).toBe(true);
  expect(dialog.textContent).toContain("Evidence detail");
  expect(showModal).toHaveBeenCalledOnce();
  expect(document.body.style.overflow).toBe("hidden");
  fireEvent.click(screen.getByRole("button", { name: "Close review evidence" }));
  expect(onDismiss).toHaveBeenCalledOnce();
  unmount();
  expect(close).toHaveBeenCalledOnce();
  expect(document.body.style.overflow).toBe("");
  } finally {
    if (originalShow) Object.defineProperty(HTMLDialogElement.prototype, "showModal", originalShow);
    else Reflect.deleteProperty(HTMLDialogElement.prototype, "showModal");
    if (originalClose) Object.defineProperty(HTMLDialogElement.prototype, "close", originalClose);
    else Reflect.deleteProperty(HTMLDialogElement.prototype, "close");
  }
});
