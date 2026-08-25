import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../http";
import type { VendorRelationshipAggregate } from "../vendorTypes";
import { createVendorRelationship, loadVendorRelationship, loadVendorRelationships, updateVendorRelationship } from "../vendorApi";
import { VendorsWorkspace } from "./VendorsWorkspace";

vi.mock("../vendorApi", () => ({
  createVendorRelationship: vi.fn(),
  loadVendorRelationship: vi.fn(),
  loadVendorRelationships: vi.fn(),
  updateVendorRelationship: vi.fn(),
}));

const record: VendorRelationshipAggregate = {
  vendor: { id: "vendor-1", tenant_id: "bank", legal_name: "Acme Processing Limited", trading_name: "Acme", registration_ref: "RC-10001", jurisdiction: "Nigeria", source_id: "procurement", external_ref: "vendor-10001", status: "ACTIVE", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
  relationship: { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card transaction processing", business_owner_principal_id: "owner-1", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-08-25T12:00:00Z", updated_at: "2026-08-25T12:00:00Z", version: 1 },
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [record] });
  vi.mocked(loadVendorRelationship).mockResolvedValue(record);
});

describe("VendorsWorkspace", () => {
  it("shows the scoped vendor register and record details", async () => {
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    expect(await screen.findByRole("heading", { name: "Vendors" })).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: /Acme Processing Limited/ }));
    expect(screen.getByText("Card transaction processing")).toBeTruthy();
    expect(screen.getByText("owner-1")).toBeTruthy();
    expect(screen.getByText("Version 1")).toBeTruthy();
  });

  it("states the exact empty population and next action", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria"/>);
    expect(await screen.findByText("No vendor relationships found for Clear Bank Nigeria.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Add vendor" })).toBeTruthy();
  });

  it("creates a vendor relationship without browser-supplied identity", async () => {
    vi.mocked(loadVendorRelationships).mockResolvedValue({ items: [] });
    vi.mocked(createVendorRelationship).mockResolvedValue(record);
    const onTarget = vi.fn();
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" onTarget={onTarget}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Add vendor" }));
    fireEvent.change(screen.getByLabelText("Legal name"), { target: { value: "Acme Processing Limited" } });
    fireEvent.change(screen.getByLabelText("Service supplied"), { target: { value: "Card transaction processing" } });
    fireEvent.change(screen.getByLabelText("Criticality"), { target: { value: "IMPORTANT" } });
    fireEvent.change(screen.getByLabelText("Privacy role"), { target: { value: "PROCESSOR" } });
    fireEvent.click(screen.getByRole("button", { name: "Add vendor relationship" }));
    await waitFor(() => expect(createVendorRelationship).toHaveBeenCalled());
    const call = vi.mocked(createVendorRelationship).mock.calls[0];
    if (!call) throw new Error("createVendorRelationship was not called");
    expect(call[0]).toEqual(expect.not.objectContaining({ tenant_id: expect.anything(), legal_entity_id: expect.anything(), actor_id: expect.anything() }));
    expect(onTarget).toHaveBeenCalledWith("relationship-1");
    expect(await screen.findByText("Vendor relationship added.")).toBeTruthy();
  });

  it("preserves entered values when a concurrent update wins", async () => {
    vi.mocked(updateVendorRelationship).mockRejectedValue(new ApiError(409, "This vendor relationship changed."));
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    expect(screen.queryByLabelText("Legal name")).toBeNull();
    expect(screen.getByText("These details are shared across the bank and cannot be changed from this service relationship.")).toBeTruthy();
    const service = screen.getByLabelText("Service supplied") as HTMLInputElement;
    fireEvent.change(service, { target: { value: "Card processing and settlement" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor relationship" }));
    expect(await screen.findByText("This record changed. Your entries are still here; reload the record before saving again.")).toBeTruthy();
    expect(service.value).toBe("Card processing and settlement");
  });

  it("updates only relationship-scoped fields", async () => {
    vi.mocked(updateVendorRelationship).mockResolvedValue({ ...record, relationship: { ...record.relationship, service_name: "Card settlement", version: 2 } });
    render(<VendorsWorkspace organizationName="Clear Bank" legalEntityName="Clear Bank Nigeria" targetID="relationship-1"/>);
    fireEvent.click(await screen.findByRole("button", { name: "Edit vendor relationship" }));
    fireEvent.change(screen.getByLabelText("Service supplied"), { target: { value: "Card settlement" } });
    fireEvent.click(screen.getByRole("button", { name: "Save vendor relationship" }));
    await waitFor(() => expect(updateVendorRelationship).toHaveBeenCalled());
    const call = vi.mocked(updateVendorRelationship).mock.calls[0];
    if (!call) throw new Error("updateVendorRelationship was not called");
    expect(call[1]).toEqual({
      expected_version: 1, service_name: "Card settlement", criticality: "IMPORTANT", privacy_role: "PROCESSOR",
      effective_from: undefined, renewal_at: undefined,
    });
    expect(call[1]).toEqual(expect.not.objectContaining({ legal_name: expect.anything(), registration_ref: expect.anything() }));
  });
});
