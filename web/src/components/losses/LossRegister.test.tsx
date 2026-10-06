import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { LossPage } from "../../lossTypes";
import { LossRegister } from "./LossRegister";

const page: LossPage = {
  items: [{
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
  }],
};

describe("LossRegister", () => {
  it("shows derived financial truth and opens the exact loss", async () => {
    const onOpenLoss = vi.fn();
    const loadPage = vi.fn().mockResolvedValue(page);
    render(<LossRegister organizationName="Clear Bank" legalEntityName="Nigeria" onOpenLoss={onOpenLoss} loadPage={loadPage}/>);

    expect(await screen.findByText("Duplicate settlement")).toBeTruthy();
    expect(screen.getByText(/net$/i)).toBeTruthy();
    expect(screen.getByText(/Gross .* Recovered/i)).toBeTruthy();
    expect(screen.getByRole("cell", { name: "Recovery: Partly recovered" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Open loss/i }));
    expect(onOpenLoss).toHaveBeenCalledWith("loss-1");
  });

  it("binds the list to organization scope and exposes supported loss filters", async () => {
    const loadPage = vi.fn().mockResolvedValue({ items: [] });
    render(<LossRegister
      legalEntityName="Clear Bank Nigeria"
      organizationScopeID="scope-payments"
      organizationScopeName="BANK / PAYMENTS"
      onOpenLoss={() => {}}
      loadPage={loadPage}
    />);

    await waitFor(() => expect(loadPage).toHaveBeenCalledWith(
      expect.objectContaining({ organizationScopeID: "scope-payments", limit: 25 }),
      expect.any(AbortSignal),
    ));
    expect(screen.getByText("Operational losses and recoveries for BANK / PAYMENTS.")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: /Recovery/i }));
    fireEvent.click(await screen.findByRole("option", { name: "Partly recovered" }));
    fireEvent.click(screen.getByRole("button", { name: /Event type/i }));
    fireEvent.click(await screen.findByRole("option", { name: "Execution, delivery & process management" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Currency" }), { target: { value: "ngn" } });

    await waitFor(() => expect(loadPage).toHaveBeenLastCalledWith(
      expect.objectContaining({
        organizationScopeID: "scope-payments",
        recoveryStatus: "PARTIAL",
        eventType: "EXECUTION_DELIVERY_PROCESS_MANAGEMENT",
        currency: "NGN",
        limit: 25,
      }),
      expect.any(AbortSignal),
    ));
  });

  it("offers canonical loss authoring without changing register pagination", async () => {
    const onRecordLoss = vi.fn();
    render(<LossRegister onOpenLoss={() => {}} onRecordLoss={onRecordLoss} loadPage={vi.fn().mockResolvedValue(page)}/>);

    await screen.findByText("Duplicate settlement");
    fireEvent.click(screen.getByRole("button", { name: "Record loss" }));
    expect(onRecordLoss).toHaveBeenCalledTimes(1);
  });
});
