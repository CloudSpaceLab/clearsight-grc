import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { MatterResponses } from "./MatterResponses";
import { ApiError } from "../http";

const api = vi.hoisted(() => ({ loadCompletedResponses: vi.fn() }));
vi.mock("../formsDistributionApi", () => api);
beforeEach(() => vi.resetAllMocks());
const item = { id: "response-1", subject_type: "MATTER", subject_id: "issue-1", title: "Owner confirmation", revision: 1, current: true, completed_at: "2026-11-20T12:00:00Z" };

it("loads scoped current responses and opens the exact response", async () => {
  api.loadCompletedResponses.mockResolvedValue({ items: [item] });
  render(<MatterResponses matterID="issue-1"/>);
  expect(await screen.findByText("Owner confirmation")).toBeTruthy();
  expect(api.loadCompletedResponses).toHaveBeenCalledWith({ subject_type: "MATTER", subject_id: "issue-1", current_only: true, sort: "COMPLETED_DESC", limit: 20 });
  expect(screen.getByRole("link", { name: "View Owner confirmation" }).getAttribute("href")).toBe("#/forms/responses/response-1");
});

it("loads later response pages without claiming totals", async () => {
  api.loadCompletedResponses.mockResolvedValueOnce({ items: [item], next_cursor: "next" }).mockResolvedValueOnce({ items: [{ ...item, id: "response-2", title: "Evidence response" }] });
  render(<MatterResponses matterID="issue-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more responses" }));
  expect(await screen.findByText("Evidence response")).toBeTruthy();
  expect(screen.getByText("Owner confirmation")).toBeTruthy();
  expect(api.loadCompletedResponses).toHaveBeenLastCalledWith(expect.objectContaining({ cursor: "next", subject_id: "issue-1" }));
  expect(screen.queryByText(/responses total/i)).toBeNull();
});

it("rejects a response from another issue", async () => {
  api.loadCompletedResponses.mockResolvedValue({ items: [{ ...item, subject_id: "other" }] });
  render(<MatterResponses matterID="issue-1"/>);
  expect(await screen.findByText("Responses unavailable.")).toBeTruthy();
  expect(screen.queryByText("Owner confirmation")).toBeNull();
});

it("clears previously loaded responses if access is revoked", async () => {
  api.loadCompletedResponses.mockResolvedValueOnce({ items: [item], next_cursor: "next" }).mockRejectedValueOnce(new ApiError(403, "Forbidden"));
  render(<MatterResponses matterID="issue-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more responses" }));
  expect(await screen.findByText("Responses unavailable.")).toBeTruthy();
  expect(screen.queryByText("Owner confirmation")).toBeNull();
});

it("keeps earlier responses and retries a failed later page", async () => {
  api.loadCompletedResponses.mockResolvedValueOnce({ items: [item], next_cursor: "next" }).mockRejectedValueOnce(new Error("unavailable")).mockResolvedValueOnce({ items: [{ ...item, id: "response-2", title: "Evidence response" }] });
  render(<MatterResponses matterID="issue-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more responses" }));
  expect(await screen.findByText("More responses unavailable.")).toBeTruthy();
  expect(screen.getByText("Owner confirmation")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Retry responses" }));
  expect(await screen.findByText("Evidence response")).toBeTruthy();
});

it("stops repeated cursors", async () => {
  api.loadCompletedResponses.mockResolvedValueOnce({ items: [item], next_cursor: "next" }).mockResolvedValueOnce({ items: [], next_cursor: "next" });
  render(<MatterResponses matterID="issue-1"/>);
  fireEvent.click(await screen.findByRole("button", { name: "Load more responses" }));
  expect(await screen.findByText("More responses unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Load more responses" })).toBeNull();
});

it("does not display an earlier issue's late response", async () => {
  let resolve!: (page: unknown) => void;
  api.loadCompletedResponses.mockReturnValueOnce(new Promise((done) => { resolve = done; })).mockResolvedValueOnce({ items: [] });
  const view = render(<MatterResponses matterID="issue-1"/>);
  view.rerender(<MatterResponses matterID="issue-2"/>);
  expect(await screen.findByText("No current responses available to you for this issue.")).toBeTruthy();
  resolve({ items: [item] });
  await vi.waitFor(() => expect(screen.queryByText("Owner confirmation")).toBeNull());
});
