import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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
    expect(screen.getByRole("cell", { name: /Gross:.*5,000,000/ })).toBeTruthy();
    expect(screen.getByRole("cell", { name: /Recovered:.*2,000,000.*Partly recovered/ })).toBeTruthy();
    expect(screen.getByRole("cell", { name: /Net:.*3,000,000.*Active/ })).toBeTruthy();
    fireEvent.click(screen.getByText("Financial impact for this page"));
    expect(screen.getByRole("region", { name: "Financial impact for displayed losses" })).toBeTruthy();
    expect(screen.getByRole("list", { name: "NGN financial exposure from loaded losses" })).toBeTruthy();
    expect(screen.getByText("Net outstanding")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Show NGN losses" }));
    await waitFor(() => expect(loadPage).toHaveBeenCalledWith(expect.objectContaining({ currency: "NGN", limit: 25 }), expect.any(AbortSignal)));

    fireEvent.click(screen.getByRole("button", { name: /Open loss/i }));
    expect(onOpenLoss).toHaveBeenCalledWith("loss-1");
  });

  it("puts the ledger before secondary currency totals and names the loaded population", async () => {
    render(<LossRegister embedded onOpenLoss={vi.fn()} loadPage={vi.fn().mockResolvedValue({ ...page, next_cursor: "next-loss-page" })}/>);

    const table = await screen.findByRole("table", { name: "Operational loss register" });
    const disclosure = screen.getByText("Financial impact for this page").closest("details")!;
    expect(disclosure.open).toBe(false);
    expect(table.compareDocumentPosition(disclosure) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Loss ledger", level: 2 })).toBeTruthy();
    expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
    fireEvent.click(screen.getByText("Financial impact for this page"));
    const summary = screen.getByRole("region", { name: "Financial impact for displayed losses" });
    expect(within(summary).getByText("1 on this page · More pages")).toBeTruthy();
    expect(within(summary).getByRole("list", { name: "NGN financial exposure from loaded losses" })).toBeTruthy();
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
    expect(screen.getByText("Recorded losses and recoveries for BANK / PAYMENTS.")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Filters" }));
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

  it("starts a new organization scope at its first page without reusing a previous cursor", async () => {
    const loadPage = vi.fn().mockResolvedValue({ ...page, next_cursor: "cursor-scope-a" });
    const onOpenLoss = vi.fn();
    const { rerender } = render(<LossRegister organizationScopeID="scope-a" onOpenLoss={onOpenLoss} loadPage={loadPage}/>);
    await screen.findByText("Duplicate settlement");
    fireEvent.click(screen.getByRole("button", { name: "Load next page" }));
    await waitFor(() => expect(loadPage).toHaveBeenLastCalledWith(expect.objectContaining({ organizationScopeID: "scope-a", cursor: "cursor-scope-a" }), expect.any(AbortSignal)));

    rerender(<LossRegister organizationScopeID="scope-b" onOpenLoss={onOpenLoss} loadPage={loadPage}/>);
    await waitFor(() => expect(loadPage).toHaveBeenLastCalledWith(expect.objectContaining({ organizationScopeID: "scope-b", cursor: undefined }), expect.any(AbortSignal)));
    expect(loadPage.mock.calls.filter(([params]) => params.organizationScopeID === "scope-b").every(([params]) => params.cursor === undefined)).toBe(true);

    rerender(<LossRegister organizationScopeID="scope-a" onOpenLoss={onOpenLoss} loadPage={loadPage}/>);
    await waitFor(() => expect(loadPage).toHaveBeenLastCalledWith(expect.objectContaining({ organizationScopeID: "scope-a", cursor: undefined }), expect.any(AbortSignal)));
  });

  it("offers canonical loss authoring without changing register pagination", async () => {
    const onRecordLoss = vi.fn();
    render(<LossRegister onOpenLoss={() => {}} onRecordLoss={onRecordLoss} loadPage={vi.fn().mockResolvedValue(page)}/>);

    await screen.findByText("Duplicate settlement");
    fireEvent.click(screen.getByRole("button", { name: "Record loss" }));
    expect(onRecordLoss).toHaveBeenCalledTimes(1);
  });
});
