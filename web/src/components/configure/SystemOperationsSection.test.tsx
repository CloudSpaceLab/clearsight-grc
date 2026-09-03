import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { SystemOperationsSection } from "./SystemOperationsSection";

const api = vi.hoisted(() => ({ loadProjectionHealth: vi.fn(), loadBackgroundJobs: vi.fn(), reconcileProgramState: vi.fn(), retryBackgroundJob: vi.fn() }));
vi.mock("../../api", () => api);
vi.mock("../ProjectionHealthCard", () => ({ ProjectionHealthCard: () => <div>Projection health</div> }));

it("shows safe terminal-job facts and requires a reason before governed retry", async () => {
  api.loadProjectionHealth.mockResolvedValue([]);
  api.loadBackgroundJobs
    .mockResolvedValueOnce({ queues: [], jobs: [{ id: "job-1", queue: "outbox-delivery", kind: "FORM_DISTRIBUTION_OPEN", state: "DEAD_LETTERED", attempts: 5, failure_code: "INVALID_TENANT_IDENTIFIER" }] })
    .mockResolvedValueOnce({ queues: [], jobs: [] });
  api.retryBackgroundJob.mockResolvedValue({ job_id: "job-1", queue: "outbox-delivery", previous_attempts: 5, state: "READY", retried_at: "2026-09-02T12:00:00Z" });

  render(<SystemOperationsSection canReconcile/>);
  await screen.findByText("FORM_DISTRIBUTION_OPEN");
  expect(screen.getByText("outbox-delivery · INVALID_TENANT_IDENTIFIER · 5 attempts")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Review retry" }));
  const submit = await screen.findByRole("button", { name: "Schedule retry" });
  expect((submit as HTMLButtonElement).disabled).toBe(true);
  const rationale = await screen.findByRole("textbox", { name: /Why is this retry safe now\?/ });
  fireEvent.change(rationale, { target: { value: "The tenant lookup defect is fixed and the event is safe to retry." } });
  fireEvent.click(submit);
  await waitFor(() => expect(api.retryBackgroundJob).toHaveBeenCalledWith("job-1", "outbox-delivery", 5, "The tenant lookup defect is fixed and the event is safe to retry."));
});
