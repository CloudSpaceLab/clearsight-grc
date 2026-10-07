import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { createReportRun, getMatterBoardBriefAvailability } from "../reportingApi";
import { MatterBoardBriefAction } from "./MatterBoardBriefAction";

vi.mock("../reportingApi", () => ({
  createReportRun: vi.fn(),
  getMatterBoardBriefAvailability: vi.fn(),
}));

const definition = {
  id: "report-def-1",
  tenant_id: "bank-1",
  legal_entity_id: "entity-1",
  code: "MATTER-BOARD-BRIEF",
  name: "Matter board brief",
  description: "",
  dataset: "MATTER_BOARD_BRIEF" as const,
  scope_kind: "MATTER" as const,
  scope_ref: "matter-1",
  format: "PDF" as const,
  status: "ACTIVE" as const,
  current_version: 3,
  effective: true,
  checksum: "a".repeat(64),
  maker_id: "maker-1",
  created_at: "2026-10-06T10:00:00Z",
  updated_at: "2026-10-06T10:00:00Z",
  version: 3,
};

beforeEach(() => {
  vi.clearAllMocks();
});

it("queues the exact governed definition when generation is authorised", async () => {
  vi.mocked(getMatterBoardBriefAvailability).mockResolvedValue({
    definition,
    can_run: true,
    authority_available: true,
  });
  vi.mocked(createReportRun).mockResolvedValue({} as never);

  render(<MatterBoardBriefAction matterID="matter-1"/>);

  fireEvent.click(await screen.findByRole("button", { name: "Generate board brief" }));
  await waitFor(() => expect(createReportRun).toHaveBeenCalledWith("report-def-1", 3));
  expect(await screen.findByText("Board brief queued.")).toBeTruthy();
  expect(screen.getByRole("link", { name: "Open Reports" }).getAttribute("href")).toBe("#reports");
});

it("explains a missing board brief setup and links to Reports", async () => {
  vi.mocked(getMatterBoardBriefAvailability).mockResolvedValue({
    can_run: false,
    authority_available: true,
    reason: "No active board brief setup is available for this issue.",
  });

  render(<MatterBoardBriefAction matterID="matter-1"/>);

  expect(await screen.findByText(/No active board brief setup is available for this issue\./)).toBeTruthy();
  expect(screen.getByRole("link", { name: "Open Reports" }).getAttribute("href")).toBe("#reports");
  expect(screen.queryByRole("button", { name: "Generate board brief" })).toBeNull();
  expect(createReportRun).not.toHaveBeenCalled();
});

it("explains an authority limitation without enabling generation", async () => {
  vi.mocked(getMatterBoardBriefAvailability).mockResolvedValue({
    definition,
    can_run: false,
    authority_available: true,
    reason: "Board brief generation is not assigned to you.",
  });

  render(<MatterBoardBriefAction matterID="matter-1"/>);

  expect(await screen.findByText("Board brief generation is not assigned to you.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Generate board brief" })).toBeNull();
  expect(createReportRun).not.toHaveBeenCalled();
});
