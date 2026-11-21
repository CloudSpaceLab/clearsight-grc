import { fireEvent, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { FormActions } from "./FormActions";

const forms = vi.hoisted(() => ({ transitionFormTemplateRevision: vi.fn() }));
const api = vi.hoisted(() => ({ loadContext: vi.fn(), resolveAuthority: vi.fn() }));
vi.mock("../formsApi", () => forms);
vi.mock("../api", () => api);
const item = { template: { id: "form-1", version: 2, status: "DRAFT" }, authority_available: true, operations: [{ command: "forms.template.transition", can_act: true, allowed_targets: ["PENDING_APPROVAL"] }] } as any;
api.loadContext.mockResolvedValue({ actor: { id: "maker" } });
api.resolveAuthority.mockResolvedValue({ principal: { id: "maker" } });

it("sends the current draft for approval using its recorded version", async () => {
  forms.transitionFormTemplateRevision.mockResolvedValue({ id: "form-1", status: "PENDING_APPROVAL", version: 3 });
  const reload = vi.fn(), saved = vi.fn(); render(<FormActions item={item} onReload={reload} onSaved={saved}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Send for approval" }));
  expect(await screen.findByText("Form status updated.")).toBeTruthy();
  expect(forms.transitionFormTemplateRevision).toHaveBeenCalledWith("form-1", 2, "PENDING_APPROVAL");
  expect(reload).toHaveBeenCalled();
  expect(saved).toHaveBeenCalledWith({ id: "form-1", status: "PENDING_APPROVAL", version: 3 });
});

it("does not expose transitions without the server's authority decision", () => {
  render(<FormActions item={{ ...item, authority_available: false }} onReload={() => {}}/>);
  expect(screen.queryByRole("button", { name: "Send for approval" })).toBeNull();
});

it("does not repeat an unconfirmed transition", async () => {
  forms.transitionFormTemplateRevision.mockRejectedValue(new Error("connection lost"));
  render(<FormActions item={item} onReload={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Send for approval" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Send for approval" })).toBeNull();
});

it("does not offer a transition when materiality-three authority is denied", async () => {
  api.resolveAuthority.mockResolvedValueOnce({ principal: { id: "another" } });
  render(<FormActions item={item} onReload={() => {}}/>);
  expect(await screen.findByText("Form action unavailable. Check your role.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Send for approval" })).toBeNull();
});

it("does not confirm a transition with a mismatched version", async () => {
  forms.transitionFormTemplateRevision.mockResolvedValueOnce({ id: "form-1", status: "PENDING_APPROVAL", version: 2 });
  const saved = vi.fn(); render(<FormActions item={item} onReload={() => {}} onSaved={saved}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Send for approval" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("Form status updated.")).toBeNull();
  expect(saved).not.toHaveBeenCalled();
});

it("keeps the confirmed result when refreshing the form throws", async () => {
  forms.transitionFormTemplateRevision.mockResolvedValueOnce({ id: "form-1", status: "PENDING_APPROVAL", version: 3 });
  const reload = vi.fn(() => { throw new Error("refresh failed"); });
  render(<FormActions item={item} onReload={reload}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Send for approval" }));
  expect(await screen.findByText("Form status updated.")).toBeTruthy();
  expect(screen.queryByText("Form status unknown. Reload before trying again.")).toBeNull();
});

it("does not offer transitions to service identities", async () => {
  api.loadContext.mockResolvedValueOnce({ actor: { id: "maker", kind: "SERVICE" } });
  render(<FormActions item={item} onReload={() => {}}/>);
  expect(await screen.findByText("Form action unavailable. Check your role.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Send for approval" })).toBeNull();
});
