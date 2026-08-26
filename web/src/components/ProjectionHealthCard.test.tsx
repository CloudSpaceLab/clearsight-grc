import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ProjectionHealth } from "../operationsTypes";
import { ProjectionHealthCard } from "./ProjectionHealthCard";

const health: ProjectionHealth = {
  tenant_id: "bank-1",
  projection: "program_state",
  display_name: "Program status",
  state: "DELAYED",
  pending: 2,
  failed: 0,
  oldest_pending: "2026-08-26T10:00:00Z",
  last_completed: "2026-08-26T09:00:00Z",
  lag_seconds: 600,
  updated_at: "2026-08-26T10:10:00Z",
};

describe("ProjectionHealthCard reconciliation authority", () => {
  it("explains the required operator and disables reconciliation for a read-only actor", () => {
    const onReconcile = vi.fn();
    render(<ProjectionHealthCard health={health} canReconcile={false} onReconcile={onReconcile}/>);

    expect((screen.getByRole("button", { name: "Check status records" }) as HTMLButtonElement).disabled).toBe(true);
    expect(screen.getByText("Platform Operations must check these records because your access is read-only.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Check status records" }));
    expect(onReconcile).not.toHaveBeenCalled();
  });

  it("allows an operations-write actor to reconcile status records", async () => {
    const onReconcile = vi.fn().mockResolvedValue({ tenant_id: "bank-1", checked: 2, queued: 1, already_queued: 0, current: 1 });
    render(<ProjectionHealthCard health={health} canReconcile onReconcile={onReconcile}/>);

    fireEvent.click(screen.getByRole("button", { name: "Check status records" }));

    await waitFor(() => expect(onReconcile).toHaveBeenCalledTimes(1));
    expect(await screen.findByText("Checked 2 Programs. 1 new status update was queued.")).toBeTruthy();
  });
});
