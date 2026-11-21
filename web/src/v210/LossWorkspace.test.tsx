import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { LossWorkspace, formatMinor } from "./LossWorkspace";
import { ApiError } from "../http";
const api = vi.hoisted(() => ({ listLosses: vi.fn(), getLoss: vi.fn() }));
vi.mock("../lossApi", () => api);
const loss = { id: "loss/1", code: "L-01", title: "Service interruption", currency: "NGN", status: "ACTIVE", version: 1, occurred_at: "2026-11-20", event_type: "OTHER" };
beforeEach(() => { vi.resetAllMocks(); api.listLosses.mockResolvedValue({ items: [{ loss, totals: { currency: "NGN", net_loss_minor: 2500 } }] }); api.getLoss.mockResolvedValue({ loss, totals: { currency: "NGN", net_loss_minor: 2500 }, recoveries: [] }); });
it("clears loss events if access is denied during pagination", async () => {
  api.listLosses.mockResolvedValueOnce({ items: [{ loss, totals: { currency: "NGN", net_loss_minor: 2500 } }], next_cursor: "next" }).mockRejectedValueOnce(new ApiError(403, "Denied"));
  render(<LossWorkspace/>);
  expect(await screen.findByText("Service interruption")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Loss events unavailable.");
  expect(screen.queryByText("Service interruption")).toBeNull();
});
it("shows the exact loss and its original currency", async () => {
  render(<LossWorkspace/>);
  expect((await screen.findByRole("link", { name: "Service interruption" })).getAttribute("href")).toBe("#/risks/losses/loss%2F1");
  expect(screen.getByText(/NGN.*25/)).toBeTruthy();
});
it("rejects a different loss returned for the requested identifier", async () => {
  api.getLoss.mockResolvedValue({ loss: { ...loss, id: "other" }, totals: {}, recoveries: [] });
  render(<LossWorkspace id="loss/1"/>);
  expect(await screen.findByRole("heading", { name: "Loss unavailable" })).toBeTruthy();
});
it("does not display an imprecise amount", () => {
  expect(formatMinor(Number.MAX_SAFE_INTEGER + 1, "NGN")).toBe("Amount unavailable");
});
