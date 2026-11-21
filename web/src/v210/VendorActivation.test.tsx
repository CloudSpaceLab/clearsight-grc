import { act, fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { VendorActivation } from "./VendorActivation";
const api = vi.hoisted(() => ({ loadVendorActivation: vi.fn(), activateVendorRelationship: vi.fn() }));
const authority = vi.hoisted(() => ({ checkVendorAuthority: vi.fn() }));
vi.mock("../vendorApi", () => api);
vi.mock("./vendorAuthority", () => authority);

const result = { eligible: true, policy: { id: "policy", policy_number: 3, version: 5, status: "ACTIVE", effective_from: "2026-01-01T00:00:00Z" }, relationship: { id: "rel-1", tenant_id: "bank", legal_entity_id: "entity", version: 4, status: "PROPOSED" }, gates: [{ code: "REVIEW", satisfied: true, explanation: "The assessment is complete." }] };
beforeEach(() => { vi.clearAllMocks(); api.loadVendorActivation.mockResolvedValue(result); authority.checkVendorAuthority.mockResolvedValue(true); });

it("identifies the effective policy and labels its decision authority gate", async () => {
  api.loadVendorActivation.mockResolvedValue({ ...result, gates: [{ code: "DECISION_AUTHORITY", satisfied: true, explanation: "Decision makers remain in the authority route." }] });
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  expect(await screen.findByText("Decision authority")).toBeTruthy();
  expect(screen.getByText("Decision makers remain in the authority route.")).toBeTruthy();
  expect(screen.getByText("Policy 3 · Version 5", { exact: false }).querySelector("time")?.getAttribute("datetime")).toBe("2026-01-01T00:00:00Z");
});

it("requires a nonempty rationale before confirmation", async () => {
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  const button = screen.getByRole("button", { name: "Confirm activation" }) as HTMLButtonElement;
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "   " } });
  expect(button.disabled).toBe(true);
  expect(api.activateVendorRelationship).not.toHaveBeenCalled();
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Approved" } });
  expect(button.disabled).toBe(false);
});

it("reports an already active read as a refresh, not a new activation", async () => {
  const active = { ...result.relationship, status: "ACTIVE", version: 6 };
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: active });
  const changed = vi.fn(), refreshed = vi.fn();
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed} onRefreshed={refreshed}/>);
  expect(await screen.findByText("Relationship active.")).toBeTruthy();
  expect(refreshed).toHaveBeenCalledWith(active);
  expect(changed).not.toHaveBeenCalled();
  expect(api.activateVendorRelationship).not.toHaveBeenCalled();
});

it("rejects a contradictory state at the selected relationship version", async () => {
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: { ...result.relationship, status: "ACTIVE" } });
  const refreshed = vi.fn();
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}} onRefreshed={refreshed}/>);
  expect(await screen.findByText("Activation requirements unavailable.")).toBeTruthy();
  expect(refreshed).not.toHaveBeenCalled();
  expect(screen.queryByText("Relationship active.")).toBeNull();
});

it("shows each current policy gate and sends the current relationship version", async () => {
  api.activateVendorRelationship.mockResolvedValue({ ...result, relationship: { ...result.relationship, version: 5, status: "ACTIVE" }, receipt: { id: "receipt", relationship_version: 5, activated_at: "2026-11-20T10:00:00Z" } });
  const changed = vi.fn();
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  expect(await screen.findByText("The assessment is complete.")).toBeTruthy();
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  await vi.waitFor(() => expect(api.activateVendorRelationship).toHaveBeenCalledWith("rel-1", expect.objectContaining({ expected_version: 4, rationale: "Required approvals completed.", intended_effective_at: "0001-01-01T00:00:00Z" })));
  expect(await screen.findByText("Relationship activated.")).toBeTruthy();
  expect(changed).toHaveBeenCalledWith(expect.objectContaining({ status: "ACTIVE", version: 5 }));
});

it("preserves a confirmed activation if the relationship refresh fails", async () => {
  api.activateVendorRelationship.mockResolvedValue({ ...result, relationship: { ...result.relationship, version: 5, status: "ACTIVE" }, receipt: { id: "receipt", relationship_version: 5, activated_at: "2026-11-20T10:00:00Z" } });
  const changed = vi.fn(() => { throw new Error("refresh failed"); });
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  expect(await screen.findByText("Relationship activated.")).toBeTruthy();
  const refreshError = screen.getByText("Relationship details could not be refreshed. Reload the relationship.");
  expect(refreshError.classList.contains("v-status--high")).toBe(true);
  expect(refreshError.closest('[role="alert"]')).toBeTruthy();
  expect(screen.queryByText(/Activation status could not be confirmed/)).toBeNull();
  expect(screen.queryByRole("button", { name: "Confirm activation" })).toBeNull();
  expect(api.activateVendorRelationship).toHaveBeenCalledTimes(1);
});

it("withholds activation when a current policy gate is unsatisfied", async () => {
  api.loadVendorActivation.mockResolvedValue({ ...result, eligible: false, gates: [{ code: "APPROVAL", satisfied: false, explanation: "Approval is required." }] });
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  expect(await screen.findByText("Approval is required.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Activate relationship" })).toBeNull();
  expect(api.activateVendorRelationship).not.toHaveBeenCalled();
});

it("does not offer activation if the authorizer route is unavailable", async () => {
  authority.checkVendorAuthority.mockRejectedValue(new Error("unavailable"));
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  expect(await screen.findByText("Activation unavailable. Check your role or reload the requirements.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Activate relationship" })).toBeNull();
});

it("does not permit a duplicate activation after an uncertain response", async () => {
  api.activateVendorRelationship.mockRejectedValue(new Error("connection lost"));
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  expect(await screen.findByText("Activation status could not be confirmed. Reload the requirements before trying again.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Confirm activation" })).toBeNull();
});

it("requires the requirements to be reloaded after a policy rejection", async () => {
  const { ApiError } = await import("../http");
  api.activateVendorRelationship.mockRejectedValue(new ApiError(422, "Approval is required.", "activation_gates_incomplete"));
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  expect(await screen.findByText("Approval is required.")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Reload requirements" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Confirm activation" })).toBeNull();
});

it.each([409, 422, 503])("retains the rationale after %s while reloading requirements and uses the newly read version", async (status) => {
  const { ApiError } = await import("../http");
  api.activateVendorRelationship.mockRejectedValueOnce(new ApiError(status, "Requirements changed."));
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: { ...result.relationship, version: 6 } });
  fireEvent.click(await screen.findByRole("button", { name: "Reload requirements" }));
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  expect((screen.getByLabelText("Activation rationale") as HTMLTextAreaElement).value).toBe("Required approvals completed.");
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  await vi.waitFor(() => expect(api.activateVendorRelationship).toHaveBeenLastCalledWith("rel-1", expect.objectContaining({ expected_version: 6 })));
});

it("ignores a late retry read after another relationship has loaded", async () => {
  let resolve!: (value: unknown) => void;
  api.loadVendorActivation.mockRejectedValueOnce(new Error("offline")).mockReturnValueOnce(new Promise((yes) => { resolve = yes; }));
  const refreshed = vi.fn();
  const view = render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}} onRefreshed={refreshed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Reload requirements" }));
  const next = { ...result.relationship, id: "rel-2" };
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: next, policy: { ...result.policy, policy_number: 9 } });
  view.rerender(<VendorActivation relationshipId="rel-2" relationshipScope={next} onActivated={() => {}} onRefreshed={refreshed}/>);
  expect(await screen.findByText(/Policy 9 · Version 5/)).toBeTruthy();
  await act(async () => resolve({ ...result, relationship: { ...result.relationship, status: "ACTIVE" } }));
  expect(screen.queryByText(/Policy 3 · Version 5/)).toBeNull();
  expect(refreshed).not.toHaveBeenCalled();
});

it("does not use requirements returned for another relationship", async () => {
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: { ...result.relationship, id: "other" } });
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  expect(await screen.findByText("Activation requirements unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Activate relationship" })).toBeNull();
});

it.each(["tenant_id", "legal_entity_id"] as const)("does not use requirements without %s", async (field) => {
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: { ...result.relationship, [field]: "" } });
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  expect(await screen.findByText("Activation requirements unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Activate relationship" })).toBeNull();
});

it.each(["tenant_id", "legal_entity_id"] as const)("rejects requirements outside the selected relationship %s", async (field) => {
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: { ...result.relationship, [field]: "other" } });
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  expect(await screen.findByText("Activation requirements unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Activate relationship" })).toBeNull();
});

it("rejects requirements older than the selected relationship", async () => {
  render(<VendorActivation relationshipId="rel-1" relationshipScope={{ ...result.relationship, version: 5 }} onActivated={() => {}}/>);
  expect(await screen.findByText("Activation requirements unavailable.")).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Activate relationship" })).toBeNull();
});

it.each(["tenant_id", "legal_entity_id", "version", "status"] as const)("invalidates an in-flight activation when selected %s changes", async (field) => {
  let resolve!: (value: unknown) => void;
  api.activateVendorRelationship.mockReturnValue(new Promise((yes) => { resolve = yes; }));
  const changed = vi.fn();
  const view = render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  const scope = { ...result.relationship, [field]: field === "version" ? 5 : field === "status" ? "ACTIVE" : "other" };
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: scope });
  view.rerender(<VendorActivation relationshipId="rel-1" relationshipScope={scope} onActivated={changed}/>);
  if (field === "status") await screen.findByText("Relationship active.");
  else await screen.findByRole("button", { name: "Activate relationship" });
  await act(async () => resolve({ ...result, relationship: { ...result.relationship, status: "ACTIVE", version: 5 }, receipt: { id: "receipt", relationship_version: 5 } }));
  expect(changed).not.toHaveBeenCalled();
  expect(screen.queryByText("Relationship activated.")).toBeNull();
});

it("does not load activation requirements for an already active relationship", () => {
  render(<VendorActivation relationshipId="rel-1" relationshipScope={{ ...result.relationship, status: "ACTIVE" }} onActivated={() => {}}/>);
  expect(screen.getByText("Relationship active.")).toBeTruthy();
  expect(api.loadVendorActivation).not.toHaveBeenCalled();
  expect(screen.queryByRole("button", { name: "Activate relationship" })).toBeNull();
});

it("does not report success without an activation receipt", async () => {
  api.activateVendorRelationship.mockResolvedValue({ ...result, relationship: { ...result.relationship, status: "ACTIVE" } });
  const changed = vi.fn();
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  expect(await screen.findByText("Activation status could not be confirmed. Reload the requirements before trying again.")).toBeTruthy();
  expect(changed).not.toHaveBeenCalled();
});

it.each(["id", "tenant_id", "legal_entity_id"] as const)("does not accept an activation response from another %s", async (field) => {
  api.activateVendorRelationship.mockResolvedValue({ ...result, relationship: { ...result.relationship, [field]: "other", status: "ACTIVE", version: 5 }, receipt: { id: "receipt", relationship_version: 5 } });
  const changed = vi.fn();
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  expect(await screen.findByText("Activation status could not be confirmed. Reload the requirements before trying again.")).toBeTruthy();
  expect(changed).not.toHaveBeenCalled();
});

it("rejects an activation receipt for a different relationship version", async () => {
  api.activateVendorRelationship.mockResolvedValue({ ...result, relationship: { ...result.relationship, status: "ACTIVE", version: 5 }, receipt: { id: "receipt", relationship_version: 4 } });
  const changed = vi.fn();
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  expect(await screen.findByText("Activation status could not be confirmed. Reload the requirements before trying again.")).toBeTruthy();
  expect(changed).not.toHaveBeenCalled();
});

it.each(["success", "failure"])("ignores a late activation %s after the relationship changes", async (outcome) => {
  let resolve!: (value: unknown) => void;
  let reject!: (error: Error) => void;
  api.activateVendorRelationship.mockReturnValue(new Promise((yes, no) => { resolve = yes; reject = no; }));
  const changed = vi.fn();
  const view = render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  api.loadVendorActivation.mockResolvedValue({ ...result, relationship: { ...result.relationship, id: "rel-2" } });
  view.rerender(<VendorActivation relationshipId="rel-2" relationshipScope={{ ...result.relationship, id: "rel-2" }} onActivated={changed}/>);
  await screen.findByRole("button", { name: "Activate relationship" });
  await act(async () => outcome === "success" ? resolve({ ...result, relationship: { ...result.relationship, status: "ACTIVE" }, receipt: { id: "receipt" } }) : reject(new Error("connection lost")));
  expect(changed).not.toHaveBeenCalled();
  expect(screen.queryByText("Relationship activated.")).toBeNull();
  expect(screen.queryByText(/Activation status could not be confirmed/)).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Activate relationship" }));
  expect((screen.getByLabelText("Activation rationale") as HTMLTextAreaElement).value).toBe("");
});

it("sends only one command for synchronous confirmation events", async () => {
  api.activateVendorRelationship.mockReturnValue(new Promise(() => {}));
  render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={() => {}}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  const button = screen.getByRole("button", { name: "Confirm activation" });
  act(() => { button.click(); button.click(); });
  expect(api.activateVendorRelationship).toHaveBeenCalledTimes(1);
});

it("ignores a completed activation after unmount", async () => {
  let resolve!: (value: unknown) => void;
  api.activateVendorRelationship.mockReturnValue(new Promise((yes) => { resolve = yes; }));
  const changed = vi.fn();
  const view = render(<VendorActivation relationshipId="rel-1" relationshipScope={result.relationship} onActivated={changed}/>);
  fireEvent.click(await screen.findByRole("button", { name: "Activate relationship" }));
  fireEvent.change(screen.getByLabelText("Activation rationale"), { target: { value: "Required approvals completed." } });
  fireEvent.click(screen.getByRole("button", { name: "Confirm activation" }));
  view.unmount();
  await act(async () => resolve({ ...result, relationship: { ...result.relationship, status: "ACTIVE" }, receipt: { id: "receipt" } }));
  expect(changed).not.toHaveBeenCalled();
});
