import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { activateVendorRelationship, loadVendorActivation } from "../vendorApi";
import type { VendorActivationResult, VendorRelationship } from "../vendorTypes";
import { VendorActivationPanel } from "./VendorActivationPanel";

vi.mock("../vendorApi", () => ({ activateVendorRelationship: vi.fn(), loadVendorActivation: vi.fn() }));

const relationship: VendorRelationship = { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card processing", business_owner_principal_id: "owner", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-09-01T10:00:00Z", updated_at: "2026-09-01T10:00:00Z", version: 4 };
const ready: VendorActivationResult = { eligible: true, relationship, policy: { id: "policy-1", policy_number: 2, version: 3, effective_from: "2026-09-01T00:00:00Z", status: "ACTIVE" }, gates: [
  { code: "CURRENT_ASSESSMENT", satisfied: true, explanation: "The completed onboarding assessment is current." },
  { code: "ADDRESS_OUTCOME", satisfied: true, explanation: "The address issue is closed with a passing independent outcome check." },
] };

beforeEach(() => { vi.clearAllMocks(); vi.mocked(loadVendorActivation).mockResolvedValue(ready); });

describe("VendorActivationPanel", () => {
  it("shows exact policy gates and enables activation only with a recorded rationale", async () => {
    const active = { ...relationship, status: "ACTIVE" as const, version: 5, effective_from: "2026-09-02T10:00:00Z" };
    vi.mocked(activateVendorRelationship).mockResolvedValue({ ...ready, relationship: active, receipt: { id: "receipt-1", relationship_version: 5, activated_at: "2026-09-02T10:00:00Z" } });
    const onActivated = vi.fn();
    render(<VendorActivationPanel relationship={relationship} onActivated={onActivated}/>);

    expect(await screen.findByText("Policy 2, version 3 applies from", { exact: false })).toBeTruthy();
    expect(screen.getByText("Current onboarding assessment")).toBeTruthy();
    const button = screen.getByRole("button", { name: "Activate vendor relationship" });
    expect((button as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "All current policy gates and independent evidence checks have passed." } });
    expect((button as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(button);

    await waitFor(() => expect(activateVendorRelationship).toHaveBeenCalledWith("relationship-1", expect.objectContaining({ expected_version: 4, rationale: "All current policy gates and independent evidence checks have passed." })));
    expect(onActivated).toHaveBeenCalledWith(active);
  });

  it("shows incomplete gates without presenting an activation action", async () => {
    vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, eligible: false, gates: [{ code: "ADDRESS_OUTCOME", satisfied: false, explanation: "Address verification must be independently confirmed and closed." }] });
    render(<VendorActivationPanel relationship={relationship} onActivated={vi.fn()}/>);
    expect(await screen.findByText("Address verification must be independently confirmed and closed.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Activate vendor relationship" })).toBeNull();
  });

  it("does not query activation gates after the relationship is active", () => {
    render(<VendorActivationPanel relationship={{ ...relationship, status: "ACTIVE" }} onActivated={vi.fn()}/>);
    expect(screen.getByRole("heading", { name: "Vendor relationship active" })).toBeTruthy();
    expect(loadVendorActivation).not.toHaveBeenCalled();
  });
});
