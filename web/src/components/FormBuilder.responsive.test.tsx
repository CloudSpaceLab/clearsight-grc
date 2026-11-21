import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FormCreator } from "../v210/FormCreator";

const api = vi.hoisted(() => ({ resolveAuthority: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../formsApi", () => ({ createLibraryFormDraft: vi.fn() }));
const context = { tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank" }, actor: { id: "maker", name: "Ada" }, mode: "live" };

describe("responsive form authoring", () => {
  it("keeps form details and questions available while the outline opens and closes", async () => {
    api.resolveAuthority.mockResolvedValue({ principal: { id: "maker" } });
    const showModal = Object.getOwnPropertyDescriptor(HTMLDialogElement.prototype, "showModal");
    const close = Object.getOwnPropertyDescriptor(HTMLDialogElement.prototype, "close");
    Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: function (this: HTMLDialogElement) { this.setAttribute("open", ""); } });
    Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: function (this: HTMLDialogElement) { this.removeAttribute("open"); } });
    try {
      render(<FormCreator context={context}/>);
      const name = await screen.findByLabelText("Form name");
      fireEvent.change(name, { target: { value: "Access review" } });
      fireEvent.change(screen.getByLabelText("Purpose"), { target: { value: "Review access" } });
      fireEvent.change(screen.getByLabelText("Question 1"), { target: { value: "Who reviews access?" } });

      fireEvent.click(screen.getByRole("button", { name: "Show questions" }));
      const outline = within(screen.getByRole("dialog", { name: "Questions" }));
      expect(outline.getByRole("navigation", { name: "Question outline" })).toBeTruthy();
      fireEvent.click(outline.getByRole("button", { name: "Go to question 1: Who reviews access?" }));
      expect(screen.queryByRole("dialog", { name: "Questions" })).toBeNull();
      expect(document.activeElement).toBe(screen.getByLabelText("Question 1"));
      expect((name as HTMLInputElement).value).toBe("Access review");
      expect((screen.getByLabelText("Purpose") as HTMLTextAreaElement).value).toBe("Review access");
    } finally {
      if (showModal) Object.defineProperty(HTMLDialogElement.prototype, "showModal", showModal); else Reflect.deleteProperty(HTMLDialogElement.prototype, "showModal");
      if (close) Object.defineProperty(HTMLDialogElement.prototype, "close", close); else Reflect.deleteProperty(HTMLDialogElement.prototype, "close");
    }
  });
});
