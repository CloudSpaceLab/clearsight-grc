import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { LossAggregate } from "../../lossTypes";
import { LossRecord } from "./LossRecord";

const aggregate: LossAggregate = {
  loss: {
    id: "loss-1",
    tenant_id: "tenant-1",
    legal_entity_id: "entity-1",
    organization_scope_id: "scope-payments",
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
  owner_display_name: "Nneka Okafor",
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

const loadRiskRecord = vi.fn().mockResolvedValue({
  risk: { code: "RISK-042", name: "Settlement processing risk" },
});

describe("LossRecord", () => {
  it("renders authorized names without principal or linked-record identifiers", async () => {
    const onOpenRisk = vi.fn();
    const onOpenMatter = vi.fn();
    const openIntervention = vi.fn().mockResolvedValue({
      loss: { ...aggregate.loss, matter_id: "matter-1", version: 3 },
      matter: { id: "matter-1", reference: "MAT-001", status: "INITIAL_REVIEW" },
    });
    render(<LossRecord
      lossID="loss-1"
      organizationScopeID="scope-payments"
      organizationScopeName="BANK / PAYMENTS"
      onBack={() => {}}
      onOpenRisk={onOpenRisk}
      onOpenMatter={onOpenMatter}
      loadLoss={vi.fn().mockResolvedValue(aggregate)}
      loadRiskRecord={loadRiskRecord}
      openIntervention={openIntervention}
    />);

    expect(await screen.findByRole("heading", { name: "Duplicate settlement" })).toBeTruthy();
    expect(await screen.findByText("RISK-042 · Settlement processing risk")).toBeTruthy();
    expect(screen.getByText("Nneka Okafor")).toBeTruthy();
    expect(screen.getByText("BANK / PAYMENTS")).toBeTruthy();
    expect(screen.queryByText("principal-secret-id")).toBeNull();
    expect(screen.queryByText("risk-secret-id")).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Open linked risk" }));
    expect(onOpenRisk).toHaveBeenCalledWith("risk-secret-id");

    fireEvent.click(screen.getByRole("button", { name: "Open intervention" }));
    await waitFor(() => expect(openIntervention).toHaveBeenCalledWith("loss-1", 2));
    await waitFor(() => expect(onOpenMatter).toHaveBeenCalledWith("matter-1"));
  });

  it("reads the existing intervention lifecycle before labelling it", async () => {
    const onOpenMatter = vi.fn();
    const openIntervention = vi.fn();
    const loadMatterRecord = vi.fn().mockResolvedValue({
      matter: { reference: "MAT-017", title: "Resolve settlement exception" },
      status_label: "Outcome check",
    });
    render(<LossRecord
      lossID="loss-1"
      onBack={() => {}}
      onOpenMatter={onOpenMatter}
      loadLoss={vi.fn().mockResolvedValue({ ...aggregate, loss: { ...aggregate.loss, matter_id: "matter-existing" } })}
      loadRiskRecord={loadRiskRecord}
      loadMatterRecord={loadMatterRecord}
      openIntervention={openIntervention}
    />);

    await screen.findByRole("heading", { name: "Duplicate settlement" });
    expect(await screen.findByText("MAT-017 · Resolve settlement exception · Outcome check")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "View intervention" }));
    expect(onOpenMatter).toHaveBeenCalledWith("matter-existing");
    expect(openIntervention).not.toHaveBeenCalled();
  });

  it("distinguishes a linked record whose authorized detail is unavailable", async () => {
    render(<LossRecord
      lossID="loss-1"
      onBack={() => {}}
      loadLoss={vi.fn().mockResolvedValue(aggregate)}
      loadRiskRecord={vi.fn().mockRejectedValue(new Error("forbidden"))}
    />);

    expect(await screen.findByText("Linked risk unavailable")).toBeTruthy();
    expect(screen.queryByText("risk-secret-id")).toBeNull();
  });

  it("records an exact recovery against the current loss version and reloads totals", async () => {
    const withoutRisk = { ...aggregate, loss: { ...aggregate.loss, risk_id: undefined } };
    const loadLoss = vi.fn().mockResolvedValue(withoutRisk);
    const recordRecovery = vi.fn().mockResolvedValue({
      loss: { ...withoutRisk.loss, version: 3 },
      recovery: {
        id: "recovery-2",
        loss_id: "loss-1",
        loss_version: 3,
        kind: "RECOVERY",
        amount_minor: 100000000,
        currency: "NGN",
        reference: "Bank adjustment",
        recovered_at: "2026-10-04T10:00:00Z",
        created_at: "2026-10-04T10:00:00Z",
      },
    });

    render(<LossRecord
      lossID="loss-1"
      onBack={() => {}}
      loadLoss={loadLoss}
      recordRecovery={recordRecovery}
    />);

    await screen.findByRole("heading", { name: "Duplicate settlement" });
    fireEvent.click(screen.getByRole("button", { name: "Record recovery" }));
    const dialog = await screen.findByRole("dialog", { name: "Record recovery entry" });
    fireEvent.change(within(dialog).getByLabelText(/^Amount/), { target: { value: "1000000.00" } });
    fireEvent.change(within(dialog).getByLabelText(/^Recorded at/), { target: { value: "2026-10-04T10:00" } });
    fireEvent.change(within(dialog).getByLabelText("Reference"), { target: { value: "Bank adjustment" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Record recovery" }));

    await waitFor(() => expect(recordRecovery).toHaveBeenCalledWith("loss-1", expect.objectContaining({
      expected_version: 2,
      kind: "RECOVERY",
      amount_minor: 100000000,
      reference: "Bank adjustment",
    })));
    await waitFor(() => expect(loadLoss).toHaveBeenCalledTimes(2));
  });
});
