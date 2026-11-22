import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FormsRegister } from "./FormsRegister";
const api = vi.hoisted(() => ({ loadFormTemplatePage: vi.fn() }));
vi.mock("../formsApi", () => api);
beforeEach(() => { api.loadFormTemplatePage.mockReset(); });
it("uses verified facets and the loaded forms register", async () => {
  api.loadFormTemplatePage.mockResolvedValue({ items: [{ template: { id: "f1", name: "Vendor assessment", status: "ACTIVE", version: 2, fields: [] } }], facets: { status: { ACTIVE: 5, DRAFT: 3, PENDING_APPROVAL: 1 } } });
  render(<FormsRegister/>);
  expect(await screen.findByRole("link", { name: "Vendor assessment" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Forms by status" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Form library" })).toBeTruthy();
  expect(screen.getByText("5 active")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Draft 3" }));
  await vi.waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenLastCalledWith({ limit: 20, status: "DRAFT", cursor: undefined }, undefined, { statusFacets: true }));
});
it("does not invent counts when status facets are missing or invalid", async () => {
  api.loadFormTemplatePage.mockResolvedValue({ items: [], facets: { status: { ACTIVE: -1 } } });
  render(<FormsRegister/>);
  expect(await screen.findByText("No forms found.")).toBeTruthy();
  expect(screen.queryByText("0 active")).toBeNull();
  expect(screen.getByText("Active forms").parentElement?.textContent).toContain("—");
});
it("labels status counts as matching forms after a search or tag filter", async () => {
  api.loadFormTemplatePage.mockResolvedValue({ items: [], facets: { status: { ACTIVE: 2 } } });
  render(<FormsRegister/>);
  expect(await screen.findByRole("heading", { name: "Forms by status" })).toBeTruthy();
  fireEvent.change(screen.getByLabelText("Tag"), { target: { value: "vendor" } });
  fireEvent.click(screen.getByRole("button", { name: "Search" }));
  expect(await screen.findByRole("heading", { name: "Matching forms by status" })).toBeTruthy();
});
