import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { LossRecord, LossTotals } from "../../lossTypes";
import { LossEntryDialog, LossRecoveryDialog } from "./LossAuthoringDialog";

const loss: LossRecord = {
  id: "loss-1",
  tenant_id: "tenant-1",
  legal_entity_id: "entity-1",
  code: "LOSS-001",
  title: "Duplicate settlement",
  event_type: "EXECUTION_DELIVERY_PROCESS_MANAGEMENT",
  cause: "Duplicate instruction",
  description: "",
  gross_amount_minor: 500000000,
  currency: "NGN",
  occurred_at: "2026-10-03T17:00:00Z",
  discovered_at: "2026-10-03T18:00:00Z",
  owner_principal_id: "owner-1",
  status: "ACTIVE",
  version: 2,
  created_at: "2026-10-03T18:00:00Z",
  updated_at: "2026-10-03T19:00:00Z",
};

describe("LossAuthoringDialog", () => {
  it("records a scoped canonical loss with an exact integer amount", async () => {
    const created = { ...loss, organization_scope_id: "scope-payments", gross_amount_minor: 123456 };
    const submit = vi.fn().mockResolvedValue(created);
    const onCreated = vi.fn();

    render(<LossEntryDialog
      organizationScopeID="scope-payments"
      organizationScopeName="BANK / PAYMENTS"
      legalEntityName="Clear Bank Nigeria"
      onClose={vi.fn()}
      onCreated={onCreated}
      submit={submit}
    />);

    const dialog = screen.getByRole("dialog", { name: "Record loss" });
    fireEvent.change(within(dialog).getByLabelText(/^Loss code/), { target: { value: "loss-2026-009" } });
    fireEvent.change(within(dialog).getByLabelText(/^Title/), { target: { value: "Settlement correction" } });
    fireEvent.click(within(dialog).getByRole("button", { name: /Event type/i }));
    fireEvent.click(await screen.findByRole("option", { name: "Execution, delivery & process management" }));
    fireEvent.change(within(dialog).getByLabelText(/^Gross loss/), { target: { value: "1234.56" } });
    fireEvent.change(within(dialog).getByLabelText(/^Occurred/), { target: { value: "2026-10-03T17:00" } });
    fireEvent.change(within(dialog).getByLabelText(/^Discovered/), { target: { value: "2026-10-03T18:00" } });
    fireEvent.change(within(dialog).getByLabelText(/^Cause/), { target: { value: "Duplicate instruction" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Record loss" }));

    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({
      organization_scope_id: "scope-payments",
      code: "LOSS-2026-009",
      event_type: "EXECUTION_DELIVERY_PROCESS_MANAGEMENT",
      gross_amount_minor: 123456,
      currency: "NGN",
      occurred_at: expect.stringMatching(/^2026-10-03T/),
      discovered_at: expect.stringMatching(/^2026-10-03T/),
    })));
    expect(onCreated).toHaveBeenCalledWith(created);
  });

  it("uses reversal semantics when a loss is already fully recovered", async () => {
    const totals: LossTotals = {
      gross_amount_minor: 500000000,
      recovered_amount_minor: 500000000,
      net_loss_minor: 0,
      currency: "NGN",
      recovery_status: "FULL",
    };
    const submit = vi.fn();

    render(<LossRecoveryDialog loss={loss} totals={totals} onClose={vi.fn()} onRecorded={vi.fn()} submit={submit}/>);

    const dialog = screen.getByRole("dialog", { name: "Record recovery entry" });
    expect(within(dialog).getByRole("button", { name: "Record reversal" })).toBeTruthy();

    fireEvent.change(within(dialog).getByLabelText(/^Amount/), { target: { value: "6000000.00" } });
    fireEvent.change(within(dialog).getByLabelText(/^Recorded at/), { target: { value: "2026-10-04T10:00" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Record reversal" }));

    expect(await within(dialog).findByText("Reversal exceeds the recovered amount.")).toBeTruthy();
    expect(submit).not.toHaveBeenCalled();
  });
});
