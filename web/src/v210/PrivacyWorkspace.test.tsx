import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { PrivacyWorkspace } from "./PrivacyWorkspace";
const api = vi.hoisted(() => ({ listProcessingActivities: vi.fn(), fetchProcessingActivity: vi.fn() }));
vi.mock("../ropaApi", () => api);
const activity = { id: "activity/1", code: "PA-01", name: "Customer records", status: "OPEN", version: 2, purpose: "Provide services", lawful_basis: "Contract", owner_display_name: "Ada" };
beforeEach(() => { vi.resetAllMocks(); api.listProcessingActivities.mockResolvedValue({ rows: [activity], has_more: false }); api.fetchProcessingActivity.mockResolvedValue({ activity, state_label: "Open" }); });
it("searches the scoped processing register", async () => {
  render(<PrivacyWorkspace/>);
  expect((await screen.findByRole("link", { name: "Customer records" })).getAttribute("href")).toBe("#/privacy/activity%2F1");
  fireEvent.change(screen.getByLabelText("Find a processing activity"), { target: { value: "customer" } });
  fireEvent.click(screen.getByRole("button", { name: "Search" }));
  await waitFor(() => expect(api.listProcessingActivities).toHaveBeenCalledWith(expect.objectContaining({ search: "customer" }), expect.any(AbortSignal)));
});
it("rejects a different activity returned for the requested identifier", async () => {
  api.fetchProcessingActivity.mockResolvedValue({ activity: { ...activity, id: "other" } });
  render(<PrivacyWorkspace id="activity/1"/>);
  expect(await screen.findByRole("heading", { name: "Processing activity unavailable" })).toBeTruthy();
  expect(screen.queryByText("Customer records")).toBeNull();
});
