import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { QuestionOutline } from "./QuestionOutline";

const show = vi.fn(function (this: HTMLDialogElement) { this.setAttribute("open", ""); });
beforeEach(() => {
  show.mockClear();
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: show });
  Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: function (this: HTMLDialogElement) { this.removeAttribute("open"); } });
});

it("opens an accessible outline sheet and returns focus when dismissed", () => {
  render(<><QuestionOutline questions={[{ id: "q", label: "Access review" }]}/><input id="v-authoring-question-0" aria-label="Question 1"/></>);
  const opener = screen.getByRole("button", { name: "Show questions" });
  opener.focus(); fireEvent.click(opener);
  expect(show).toHaveBeenCalledOnce();
  const dialog = screen.getByRole("dialog", { name: "Questions" });
  expect(dialog.contains(document.activeElement)).toBe(true);
  expect(document.body.style.overflow).toBe("hidden");
  fireEvent(dialog, new Event("cancel", { bubbles: true, cancelable: true }));
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(document.body.style.overflow).toBe("");
  expect(document.activeElement).toBe(opener);
});

it("focuses the chosen question when the sheet closes", () => {
  render(<><QuestionOutline questions={[{ id: "q", label: "Access review" }]}/><input id="v-authoring-question-0" aria-label="Question 1"/></>);
  fireEvent.click(screen.getByRole("button", { name: "Show questions" }));
  const dialog = screen.getByRole("dialog", { name: "Questions" });
  fireEvent.click(dialog.querySelector("nav button")!);
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(document.activeElement).toBe(screen.getByLabelText("Question 1"));
});

it("moves focus to the workspace when the sheet is replaced by the desktop outline", () => {
  const originalWidth = window.innerWidth;
  Object.defineProperty(window, "innerWidth", { configurable: true, value: 390 });
  try {
    render(<main id="v-main" tabIndex={-1}><QuestionOutline questions={[]}/></main>);
    fireEvent.click(screen.getByRole("button", { name: "Show questions" }));
    expect(screen.getByRole("dialog", { name: "Questions" })).toBeTruthy();
    Object.defineProperty(window, "innerWidth", { configurable: true, value: 1024 });
    fireEvent(window, new Event("resize"));
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(document.activeElement?.id).toBe("v-main");
  } finally { Object.defineProperty(window, "innerWidth", { configurable: true, value: originalWidth }); }
});

it("dismisses only clicks outside the sheet, not empty sheet content", () => {
  render(<QuestionOutline questions={[]}/>);
  const opener = screen.getByRole("button", { name: "Show questions" });
  fireEvent.click(opener);
  const dialog = screen.getByRole("dialog", { name: "Questions" });
  vi.spyOn(dialog, "getBoundingClientRect").mockReturnValue({ left: 10, right: 390, top: 0, bottom: 900, x: 10, y: 0, width: 380, height: 900, toJSON: () => ({}) });
  fireEvent.click(dialog, { clientX: 100, clientY: 500 });
  expect(screen.getByRole("dialog", { name: "Questions" })).toBeTruthy();
  fireEvent.click(dialog, { clientX: 2, clientY: 500 });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(document.activeElement).toBe(opener);
});
