import { act, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "../../http";
import { AuthorityRoutingSection } from "./AuthorityRoutingSection";

const api = vi.hoisted(() => ({ loadIntegrity: vi.fn(), loadWorkflowTasks: vi.fn() }));
vi.mock("../../api", () => api);
vi.mock("../GovernanceAdminPanel", () => ({ GovernanceAdminPanel: () => <div>Combined governance</div> }));

beforeEach(() => { vi.clearAllMocks(); api.loadIntegrity.mockResolvedValue([]); api.loadWorkflowTasks.mockResolvedValue([]); });

it("does not claim an eligible route or complete task assignment while checks load or return empty", async () => {
  let resolve: (value: []) => void;
  api.loadIntegrity.mockReturnValue(new Promise<[]>((done) => { resolve = done; }));
  render(<AuthorityRoutingSection checksOnly/>);
  expect(screen.getByText("Checking routing findings for the current scope.")).toBeTruthy();
  expect(screen.queryByText(/eligible path|Every open task|No blocking/)).toBeNull();
  await act(async () => resolve!([]));
  expect(screen.getByText("No findings returned")).toBeTruthy();
  expect(screen.getByText("No workflow tasks returned")).toBeTruthy();
  expect(screen.queryByText("Combined governance")).toBeNull();
});

it("keeps failed routing reads separate from available tasks and denied reads separate from empty", async () => {
  api.loadIntegrity.mockRejectedValue(new ApiError(403, "Denied"));
  api.loadWorkflowTasks.mockResolvedValue([{ id: "task", title: "Confirm the review date", responsibility: "REVIEWER", step_key: "REVIEW", status: "OPEN" }]);
  render(<AuthorityRoutingSection checksOnly/>);
  expect(await screen.findByText("Routing findings are restricted with your current access.")).toBeTruthy();
  expect(screen.getByText("Confirm the review date")).toBeTruthy();
  expect(screen.getByText("1 of 1 returned tasks")).toBeTruthy();
  expect(screen.queryByText("No findings returned")).toBeNull();
});

it("rechecks on return and ignores a late response from the inactive visit", async () => {
  let resolveOld: (value: unknown[]) => void;
  api.loadIntegrity.mockReturnValueOnce(new Promise((done) => { resolveOld = done; })).mockResolvedValue([]);
  const { rerender } = render(<AuthorityRoutingSection checksOnly/>);
  rerender(<AuthorityRoutingSection checksOnly active={false}/>);
  rerender(<AuthorityRoutingSection checksOnly/>);
  await screen.findByText("No findings returned");
  await act(async () => resolveOld!([{ type: "old", summary: "Old scope finding", severity: "HIGH", required_action: "Old action" }]));
  expect(screen.queryByText("Old scope finding")).toBeNull();
  expect(api.loadIntegrity).toHaveBeenCalledTimes(2);
});
