import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { LossAggregate } from "../../lossTypes";
import { LossRecord } from "./LossRecord";

const aggregate: LossAggregate = {
  loss: {
    id: "loss-1",
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    code: "LOSS-001",
    title: "Duplicate settlement",
    event_type: "EXECUTION_DELIVERY_PROCESS_MANAGEMENT",
    cause: "Duplicate instruction",
    description: "Duplicate settlement completed before correction.",
    gross_amount_minor: 500000000,
    currency: "NGN",
    occurred_at: "2026-10-03T17:00:00Z",
    discovered_at: "2026-10-03T18:00:00Z",
    risk_id: "risk-secret-id",
    owner_principal_id: "principal-secret-id",
    status: "ACTIVE",
    version: 2,
    created_at: "2026-10-03T18:00:00Z",
    updated_at: "2026-10-03T19:00:00Z",
  },
  totals: {
    gross_amount_minor: 500000000,
    recovered_amount_minor: 200000000,
    net_loss_minor: 300000000,
    currency: "NGN",
    recovery_status: "PARTIAL",
  },
  recoveries: [{
    id: "recovery-1",
    loss_id: "loss-1",
    loss_version: 2,
    kind: "RECOVERY",
    amount_minor: 200000000,
    currency: "NGN",
    reference: "Insurer settlement",
    recovered_at: "2026-10-03T19:00:00Z",
    created_at: "2026-10-03T19:00:00Z",
  }],
};

describe("LossRecord", () => {
  it("does not render principal identifiers and opens canonical linked records", async () => {
    const onOpenRisk = vi.fn();
    const onOpenMatter = vi.fn();
    const openIntervention = vi.fn().mockResolvedValue({
      loss: { ...aggregate.loss, matter_id: "matter-1", version: 3 },
      matter: { id: "matter-1", reference: "MAT-001", status: "INITIAL_REVIEW" },
    });
    render(<LossRecord
      lossID="loss-1"
      onBack={() => {}}
      onOpenRisk={onOpenRisk}
      onOpenMatter={onOpenMatter}
      loadLoss={vi.fn().mockResolvedValue(aggregate)}
      openIntervention={openIntervention}
    />);

    expect(await screen.findByRole("heading", { name: "Duplicate settlement" })).toBeTruthy();
    expect(screen.queryByText("principal-secret-id")).toBeNull();
    expect(screen.queryByText("risk-secret-id")).toBeNull();
    expect(screen.getByText("Assigned")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Open linked risk" }));
    expect(onOpenRisk).toHaveBeenCalledWith("risk-secret-id");

    fireEvent.click(screen.getByRole("button", { name: "Open intervention" }));
    await waitFor(() => expect(openIntervention).toHaveBeenCalledWith("loss-1", 2));
    await waitFor(() => expect(onOpenMatter).toHaveBeenCalledWith("matter-1"));
  });

  it("opens an existing intervention without creating another Matter", async () => {
    const onOpenMatter = vi.fn();
    const openIntervention = vi.fn();
    render(<LossRecord
      lossID="loss-1"
      onBack={() => {}}
      onOpenMatter={onOpenMatter}
      loadLoss={vi.fn().mockResolvedValue({ ...aggregate, loss: { ...aggregate.loss, matter_id: "matter-existing" } })}
      openIntervention={openIntervention}
    />);

    await screen.findByRole("heading", { name: "Duplicate settlement" });
    fireEvent.click(screen.getByRole("button", { name: "Open intervention" }));
    expect(onOpenMatter).toHaveBeenCalledWith("matter-existing");
    expect(openIntervention).not.toHaveBeenCalled();
  });
});
