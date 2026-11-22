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
it("clears previously loaded activities if access is revoked during pagination", async () => {
  api.listProcessingActivities.mockResolvedValueOnce({ rows: [activity], has_more: true, next_cursor: "next" })
    .mockRejectedValueOnce(Object.assign(new Error("Forbidden"), { kind: "forbidden" }));
  render(<PrivacyWorkspace/>);
  expect(await screen.findByRole("link", { name: "Customer records" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Load more" }));
  expect((await screen.findByRole("alert")).textContent).toContain("Processing activities unavailable");
  expect(screen.queryByText("Customer records")).toBeNull();
});
it("shows recorded data sensitivity, safeguards, systems and named reviews", async () => {
  api.fetchProcessingActivity.mockResolvedValue({ activity: { ...activity,
    data_categories: [{ category: "Biometric data", sensitivity: "SENSITIVE_BY_LAW" }],
    recipients: [{ recipient: "Cloud processor", recipient_kind: "EXTERNAL", country_code: "GB", is_cross_border: true, transfer_basis: "STANDARD_CONTRACT_CLAUSES" }],
    systems: [{ system_name: "Customer platform", system_kind: "APPLICATION" }],
    reviews: [{ id: "review-1", due_date: "2026-02-01", completed_at: "2026-01-31T00:00:00Z", outcome: "CONFIRMED", reviewer_principal_id: "reviewer-1", reviewer_display_name: "Chika Nwosu" }],
  }, state_label: "Open" });
  render(<PrivacyWorkspace id="activity/1"/>);
  expect(await screen.findByText("Biometric data")).toBeTruthy();
  expect(screen.getByText("Sensitive by law")).toBeTruthy();
  expect(screen.getByText(/Standard contract clauses/)).toBeTruthy();
  expect(screen.getByText(/GB/)).toBeTruthy();
  expect(screen.getByText("Customer platform")).toBeTruthy();
  expect(screen.getByText("Chika Nwosu")).toBeTruthy();
  expect(screen.getByText("Confirmed")).toBeTruthy();
  expect(screen.queryByText("reviewer-1")).toBeNull();
});
