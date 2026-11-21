import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FreshApp } from "../v210/FreshApp";

const api = vi.hoisted(() => ({ loadContext: vi.fn(), loadSessionStatus: vi.fn(), resolveAuthority: vi.fn() }));
const forms = vi.hoisted(() => ({ loadFormTemplateItem: vi.fn(), createLibraryFormRevision: vi.fn(), transitionFormTemplateRevision: vi.fn() }));
vi.mock("../api", () => api);
vi.mock("../formsApi", () => forms);

it("retains the saved draft when the subsequent approval transition is unconfirmed", async () => {
  window.location.hash = "#/forms/form-1/edit";
  api.loadSessionStatus.mockResolvedValue({ authenticated: true, demo_login_available: false });
  api.loadContext.mockResolvedValue({ tenant: { id: "bank", name: "Bank" }, legal_entity: { id: "entity", name: "Bank" }, actor: { id: "maker", name: "Ada" }, capabilities: {} });
  api.resolveAuthority.mockResolvedValue({ principal: { id: "maker" } });
  const original = { id: "form-1", code: "VENDOR", name: "Vendor review", purpose: "Review vendor controls", scoring_mode: "NONE", presentation: { default_mode: "AUTOMATIC", allow_mode_switch: false }, sections: [{ id: "identity", title: "Identity" }], fields: [{ id: "name", section_id: "identity", label: "Registered name", type: "short_text", required: true }], status: "DRAFT", version: 1 };
  let persisted = original;
  const operations = [{ command: "forms.template.revise", can_act: true }, { command: "forms.template.transition", can_act: true, allowed_targets: ["PENDING_APPROVAL"] }];
  forms.loadFormTemplateItem.mockImplementation(async () => ({ template: persisted, authority_available: true, operations }));
  forms.createLibraryFormRevision.mockImplementation(async (_id, _version, input) => { persisted = { ...original, ...input, status: "DRAFT", version: 2 }; return persisted; });
  forms.transitionFormTemplateRevision.mockRejectedValue(new Error("approval service unavailable"));
  render(<FreshApp/>);
  fireEvent.change(await screen.findByLabelText("Form name"), { target: { value: "Updated vendor review" } });
  fireEvent.click(screen.getByRole("button", { name: "Save draft" }));
  fireEvent.click(await screen.findByRole("link", { name: "View draft" }));
  expect(await screen.findByRole("heading", { name: "Updated vendor review" })).toBeTruthy();
  expect(screen.getByText("Version 2")).toBeTruthy();
  fireEvent.click(await screen.findByRole("button", { name: "Send for approval" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Form status unknown");
  expect(screen.getByRole("heading", { name: "Updated vendor review" })).toBeTruthy();
  expect(screen.getByText("Version 2")).toBeTruthy();
  expect(forms.createLibraryFormRevision).toHaveBeenCalledTimes(1);
  expect(forms.createLibraryFormRevision).toHaveBeenCalledWith("form-1", 1, expect.objectContaining({ name: "Updated vendor review" }));
  expect(forms.transitionFormTemplateRevision).toHaveBeenCalledOnce();
  expect(forms.transitionFormTemplateRevision).toHaveBeenCalledWith("form-1", 2, "PENDING_APPROVAL");
  expect(screen.queryByRole("button", { name: "Send for approval" })).toBeNull();
});
