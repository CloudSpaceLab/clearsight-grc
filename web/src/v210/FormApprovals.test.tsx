import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { FormApprovals } from "./FormApprovals";
import { ApiError } from "../http";

const api = vi.hoisted(() => ({ loadFormTemplatePage: vi.fn() }));
vi.mock("../formsApi", () => api);
const form = (id: string, canAct: boolean) => ({ template: { id, name: `Form ${id}`, version: 2, status: "PENDING_APPROVAL" }, authority_available: true, operations: [{ command: "forms.template.transition", can_act: canAct, allowed_targets: ["ACTIVE", "REJECTED"] }] });
beforeEach(() => vi.resetAllMocks());

it("shows actionable form approvals with a link to review the exact form", async () => {
  api.loadFormTemplatePage.mockResolvedValue({ items: [form("allowed", true), form("other", false)] });
  render(<FormApprovals/>);
  const link = await screen.findByRole("link", { name: "Review Form allowed" });
  expect(link.getAttribute("href")).toBe("#/forms/allowed?approval_version=2");
  expect(screen.queryByText("Form other")).toBeNull();
  expect(api.loadFormTemplatePage).toHaveBeenCalledWith({ status: "PENDING_APPROVAL", limit: 20 }, expect.any(AbortSignal));
});

it("encodes the form ID separately from its version", async () => {
  api.loadFormTemplatePage.mockResolvedValue({ items: [form("unit/one", true)] });
  render(<FormApprovals/>);
  expect((await screen.findByRole("link", { name: "Review Form unit/one" })).getAttribute("href")).toBe("#/forms/unit%2Fone?approval_version=2");
});

it("retains pagination when a page has no actionable form approvals", async () => {
  api.loadFormTemplatePage.mockResolvedValueOnce({ items: [form("other", false)], next_cursor: "next" }).mockResolvedValueOnce({ items: [form("allowed", true)] });
  render(<FormApprovals/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more forms" }));
  expect(await screen.findByRole("link", { name: "Review Form allowed" })).toBeTruthy();
  await waitFor(() => expect(api.loadFormTemplatePage).toHaveBeenLastCalledWith({ status: "PENDING_APPROVAL", limit: 20, cursor: "next" }));
});

it("does not present unavailable authority as an empty approval list", async () => {
  api.loadFormTemplatePage.mockResolvedValue({ items: [{ ...form("unknown", false), authority_available: false }] });
  render(<FormApprovals/>);
  expect((await screen.findByRole("alert")).textContent).toContain("Form approval access unavailable.");
  expect(screen.queryByText("No forms available for your approval.")).toBeNull();
});

it("does not show a revision that is no longer pending", async () => {
  api.loadFormTemplatePage.mockResolvedValue({ items: [{ ...form("changed", true), template: { ...form("changed", true).template, status: "ACTIVE" } }] });
  render(<FormApprovals/>);
  expect(await screen.findByText("No forms available for your approval.")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Review Form changed" })).toBeNull();
});

it("clears earlier form data if access is denied during pagination", async () => {
  api.loadFormTemplatePage.mockResolvedValueOnce({ items: [form("allowed", true)], next_cursor: "next" }).mockRejectedValueOnce(new ApiError(403, "Denied"));
  render(<FormApprovals/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more forms" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByRole("link", { name: "Review Form allowed" })).toBeNull();
});
