import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ProgramResponseReviews } from "./ProgramResponseReviews";

const api = vi.hoisted(() => ({ loadProgramResponseReviews: vi.fn() }));
vi.mock("./programResponseReviewsApi", () => api);
const scope = { tenantID: "bank", legalEntityID: "entity", principalID: "reviewer" };
const page = { scope: { tenant_id: "bank", legal_entity_id: "entity", principal_id: "reviewer" }, coverage: "PROGRAM_RESPONSE_REVIEWS", items: [{ id: "revision/1", title: "Access review", revision: 2, subject_type: "PROGRAM", state: "FINAL", current: true, submitted_by_display_name: "Ada Okafor", completed_at: "2026-11-20T10:00:00Z" }] };
beforeEach(() => vi.resetAllMocks());

it("shows the named respondent and opens the exact response", async () => {
  api.loadProgramResponseReviews.mockResolvedValue(page);
  render(<ProgramResponseReviews scope={scope}/>);
  expect(await screen.findByText(/Ada Okafor/)).toBeTruthy();
  expect(screen.getByText(/Submitted.*2026.*Response version 2/)).toBeTruthy();
  expect(screen.getByRole("link", { name: "Review Access review" }).getAttribute("href")).toBe("#/forms/responses/revision%2F1");
});

it("does not display results for a different signed-in person", async () => {
  api.loadProgramResponseReviews.mockResolvedValue({ ...page, scope: { ...page.scope, principal_id: "other" } });
  render(<ProgramResponseReviews scope={scope}/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("Access review")).toBeNull();
});

it("does not offer a retry when Program reviews are not available", async () => {
  const { ApiError } = await import("../http");
  api.loadProgramResponseReviews.mockRejectedValue(new ApiError(404, "Unavailable", "not_found"));
  render(<ProgramResponseReviews scope={scope}/>);
  expect(await screen.findByText("Program reviews unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Try again" })).toBeNull();
});

it("does not treat an earlier response as pending work", async () => {
  api.loadProgramResponseReviews.mockResolvedValue({ ...page, items: [{ ...page.items[0], current: false }] });
  render(<ProgramResponseReviews scope={scope}/>);
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("Access review")).toBeNull();
});

it("clears previously loaded reviews if later access is denied", async () => {
  const { ApiError } = await import("../http");
  api.loadProgramResponseReviews.mockResolvedValueOnce({ ...page, next_cursor: "next" }).mockRejectedValueOnce(new ApiError(403, "Denied", "forbidden"));
  render(<ProgramResponseReviews scope={scope}/>);
  await screen.findByText("Access review");
  fireEvent.click(screen.getByRole("button", { name: "Load more reviews" }));
  expect(await screen.findByRole("alert")).toBeTruthy();
  expect(screen.queryByText("Access review")).toBeNull();
});
