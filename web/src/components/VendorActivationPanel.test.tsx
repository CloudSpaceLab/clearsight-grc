import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { activateVendorRelationship, loadVendorActivation } from "../vendorApi";
import type { VendorActivationResult, VendorRelationship } from "../vendorTypes";
import { VendorActivationPanel } from "./VendorActivationPanel";
import { ApiError } from "../http";

vi.mock("../vendorApi", () => ({ activateVendorRelationship: vi.fn(), loadVendorActivation: vi.fn() }));

const relationship: VendorRelationship = { id: "relationship-1", tenant_id: "bank", legal_entity_id: "entity", vendor_id: "vendor-1", service_name: "Card processing", business_owner_principal_id: "owner", criticality: "IMPORTANT", privacy_role: "PROCESSOR", status: "PROPOSED", created_at: "2026-09-01T10:00:00Z", updated_at: "2026-09-01T10:00:00Z", version: 4 };
const ready: VendorActivationResult = { eligible: true, relationship, policy: { id: "policy-1", policy_number: 2, version: 3, effective_from: "2026-09-01T00:00:00Z", status: "ACTIVE" }, gates: [
  { code: "CURRENT_ASSESSMENT", satisfied: true, explanation: "The completed onboarding assessment is current." },
  { code: "ADDRESS_OUTCOME", satisfied: true, explanation: "The address issue is closed with a passing independent outcome check." },
] };

beforeEach(() => { vi.resetAllMocks(); vi.mocked(loadVendorActivation).mockResolvedValue(ready); });

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const rationale = "All current policy gates and independent evidence checks have passed.";
async function beginActivation() {
  await screen.findByLabelText("Activation rationale");
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: rationale } });
  fireEvent.click(screen.getByRole("button", { name: "Activate vendor relationship" }));
}

describe("VendorActivationPanel", () => {
  it.each([409, 422, 503])("discards eligibility after %s and reloads the command version while retaining the rationale", async (status) => {
    vi.mocked(activateVendorRelationship).mockRejectedValueOnce(new ApiError(status, "changed"));
    const onActivated = vi.fn();
    render(<VendorActivationPanel relationship={relationship} onActivated={onActivated}/>);
    await beginActivation();
    const reload = await screen.findByRole("button", { name: "Reload activation checks" });
    expect((reload as HTMLButtonElement).disabled).toBe(false);
    expect(screen.queryByText("Ready for authorization")).toBeNull();
    expect(screen.queryByRole("button", { name: "Activate vendor relationship" })).toBeNull();
    expect(screen.queryByText(/remains unchanged|has not changed|did not complete/)).toBeNull();
    expect(onActivated).not.toHaveBeenCalled();
    vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, relationship: { ...relationship, version: 6 } });
    vi.mocked(activateVendorRelationship).mockResolvedValue({ ...ready, relationship: { ...relationship, version: 7, status: "ACTIVE" } });
    fireEvent.click(reload);
    expect((await screen.findByLabelText("Activation rationale") as HTMLTextAreaElement).value).toBe(rationale);
    fireEvent.click(screen.getByRole("button", { name: "Activate vendor relationship" }));
    await waitFor(() => expect(activateVendorRelationship).toHaveBeenLastCalledWith(relationship.id, expect.objectContaining({ expected_version: 6 })));
  });

  it("ignores a late retry read after another relationship has loaded", async () => {
    const pending = deferred<VendorActivationResult>();
    vi.mocked(loadVendorActivation).mockRejectedValueOnce(new Error("offline")).mockReturnValueOnce(pending.promise);
    const onActivated = vi.fn();
    const onRefreshed = vi.fn();
    const view = render(<VendorActivationPanel relationship={relationship} onActivated={onActivated} onRefreshed={onRefreshed}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Reload activation checks" }));
    const second = { ...relationship, id: "relationship-2", service_name: "Second service" };
    vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, relationship: second, policy: { ...ready.policy, policy_number: 9 } });
    view.rerender(<VendorActivationPanel relationship={second} onActivated={onActivated} onRefreshed={onRefreshed}/>);
    await screen.findByText("Policy 9, version 3 applies from", { exact: false });
    await act(async () => pending.resolve(ready));
    expect(screen.queryByText("Policy 2, version 3 applies from", { exact: false })).toBeNull();
    expect(onRefreshed).not.toHaveBeenCalledWith(relationship);
  });

  it.each(["success", "error"])("ignores late activation %s after selection changes", async (outcome) => {
    const pending = deferred<VendorActivationResult>();
    vi.mocked(activateVendorRelationship).mockReturnValue(pending.promise);
    const onActivated = vi.fn();
    const view = render(<VendorActivationPanel relationship={relationship} onActivated={onActivated}/>);
    await beginActivation();
    const second = { ...relationship, id: "relationship-2" };
    vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, relationship: second });
    view.rerender(<VendorActivationPanel relationship={second} onActivated={onActivated}/>);
    expect((await screen.findByLabelText("Activation rationale") as HTMLTextAreaElement).value).toBe("");
    await act(async () => outcome === "success" ? pending.resolve({ ...ready, relationship: { ...relationship, status: "ACTIVE" } }) : pending.reject(new ApiError(409, "changed")));
    expect(onActivated).not.toHaveBeenCalled();
    expect(screen.queryByText(/relationship or activation policy changed/i)).toBeNull();
    fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: rationale } });
    expect((screen.getByRole("button", { name: "Activate vendor relationship" }) as HTMLButtonElement).disabled).toBe(false);
  });

  it.each(["unmount", "active", "version", "tenant", "entity"])("invalidates a pending command on %s", async (change) => {
    const pending = deferred<VendorActivationResult>();
    vi.mocked(activateVendorRelationship).mockReturnValue(pending.promise);
    const onActivated = vi.fn();
    const view = render(<VendorActivationPanel relationship={relationship} onActivated={onActivated}/>);
    await beginActivation();
    if (change === "unmount") view.unmount();
    else {
      const next = { ...relationship, ...(change === "active" ? { status: "ACTIVE" as const } : change === "version" ? { version: 6 } : change === "tenant" ? { tenant_id: "other" } : { legal_entity_id: "other" }) };
      vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, relationship: next });
      view.rerender(<VendorActivationPanel relationship={next} onActivated={onActivated}/>);
    }
    await act(async () => pending.resolve({ ...ready, relationship: { ...relationship, status: "ACTIVE" } }));
    expect(onActivated).not.toHaveBeenCalled();
  });

  it.each(["id", "tenant_id", "legal_entity_id"] as const)("rejects activation reads from a different %s", async (field) => {
    vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, relationship: { ...relationship, [field]: "other" } });
    render(<VendorActivationPanel relationship={relationship} onActivated={vi.fn()}/>);
    await screen.findByRole("button", { name: "Reload activation checks" });
    expect(screen.queryByText("Ready for authorization")).toBeNull();
  });

  it.each(["id", "tenant_id", "legal_entity_id"] as const)("rejects activation responses from a different %s", async (field) => {
    vi.mocked(activateVendorRelationship).mockResolvedValue({ ...ready, relationship: { ...relationship, [field]: "other", status: "ACTIVE" } });
    const onActivated = vi.fn();
    render(<VendorActivationPanel relationship={relationship} onActivated={onActivated}/>);
    await beginActivation();
    await screen.findByRole("button", { name: "Reload activation checks" });
    expect(onActivated).not.toHaveBeenCalled();
  });

  it("shows an already active read without claiming a new activation command", async () => {
    const active = { ...relationship, status: "ACTIVE" as const, version: 6 };
    vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, relationship: active });
    const onActivated = vi.fn();
    const onRefreshed = vi.fn();
    render(<VendorActivationPanel relationship={relationship} onActivated={onActivated} onRefreshed={onRefreshed}/>);
    await screen.findByRole("heading", { name: "Vendor relationship active" });
    expect(onRefreshed).toHaveBeenCalledWith(active);
    expect(onActivated).not.toHaveBeenCalled();
    expect(activateVendorRelationship).not.toHaveBeenCalled();
  });

  it("issues only one command for synchronous activation clicks", async () => {
    const pending = deferred<VendorActivationResult>();
    vi.mocked(activateVendorRelationship).mockReturnValue(pending.promise);
    render(<VendorActivationPanel relationship={relationship} onActivated={vi.fn()}/>);
    await screen.findByLabelText("Activation rationale");
    fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: rationale } });
    const button = screen.getByRole("button", { name: "Activate vendor relationship" });
    act(() => { button.click(); button.click(); });
    expect(activateVendorRelationship).toHaveBeenCalledTimes(1);
  });

  it("names the decision authority gate in working language", async () => {
    vi.mocked(loadVendorActivation).mockResolvedValue({ ...ready, gates: [{ code: "DECISION_AUTHORITY", satisfied: true, explanation: "The recorded decision makers remain in the current authority route." }] });
    render(<VendorActivationPanel relationship={relationship} onActivated={vi.fn()}/>);
    expect(await screen.findByText("Decision authority")).toBeTruthy();
  });
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

it("refreshes activation checks after assessment progress without changing the relationship", async () => {
  const relationship = { id: "service", version: 1, status: "PROPOSED", service_name: "Payment service" } as VendorRelationship;
  const pending = { eligible: false, relationship, policy: { id: "policy", status: "ACTIVE", policy_number: 1, version: 1, effective_from: "2026-01-01T00:00:00Z" }, gates: [{ code: "CURRENT_ASSESSMENT", satisfied: false, explanation: "Assessment review is pending." }] } as VendorActivationResult;
  vi.mocked(loadVendorActivation).mockResolvedValueOnce(pending).mockResolvedValueOnce({ ...pending, eligible: true, gates: [{ code: "CURRENT_ASSESSMENT", satisfied: true, explanation: "Assessment review is complete." }] });
  const { rerender } = render(<VendorActivationPanel relationship={relationship} reviewVersion={3} onActivated={vi.fn()}/>);
  expect(await screen.findByText("Assessment review is pending.")).toBeTruthy();
  rerender(<VendorActivationPanel relationship={relationship} reviewVersion={4} onActivated={vi.fn()}/>);
  await waitFor(() => expect(loadVendorActivation).toHaveBeenCalledTimes(2));
  expect(await screen.findByText("Ready for authorization")).toBeTruthy();
});
