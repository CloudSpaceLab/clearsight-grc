import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { GovernanceAdminPanel } from "./GovernanceAdminPanel";
import type { RuntimeContext } from "../api";
import type { GovernanceInventory } from "../governanceAdminApi";

const { loadGovernanceInventory, createGovernanceDelegation, createGovernancePolicyDraft, searchGovernanceDelegationCandidates, transitionGovernancePolicy, transitionGovernanceDelegation, loadIdentityAccessOverview, loadContext } = vi.hoisted(() => ({
  loadGovernanceInventory: vi.fn(),
  createGovernanceDelegation: vi.fn(),
  createGovernancePolicyDraft: vi.fn(),
  searchGovernanceDelegationCandidates: vi.fn(),
  transitionGovernancePolicy: vi.fn(),
  transitionGovernanceDelegation: vi.fn(),
  loadIdentityAccessOverview: vi.fn(),
  loadContext: vi.fn(),
}));

vi.mock("../governanceAdminApi", () => ({ loadGovernanceInventory, createGovernanceDelegation, createGovernancePolicyDraft, searchGovernanceDelegationCandidates, transitionGovernancePolicy, transitionGovernanceDelegation }));
vi.mock("../identityAccessApi", () => ({ loadIdentityAccessOverview }));
vi.mock("../api", () => ({ loadContext }));

beforeEach(() => {
  vi.clearAllMocks();
  loadContext.mockResolvedValue({ tenant: { id: "bank", name: "ClearSight Bank" }, legal_entity: { id: "entity-1", name: "ClearSight Bank Plc" }, actor: { id: "actor-1", name: "Ada Okafor" }, mode: "production", capabilities: { config_write: true } });
  loadIdentityAccessOverview.mockResolvedValue({ actor_principal_id: "actor-1", can_configure: true, people: [{ id: "actor-1", display_name: "Ada Okafor", status: "ACTIVE" }, { id: "person-2", display_name: "Tunde Bello", status: "ACTIVE" }], legal_entities: [{ id: "entity-1", name: "ClearSight Bank Plc" }], roles: [{ id: "role-1", code: "CONTROL_ASSURANCE", name: "Control Assurance", capabilities: [] }] });
  searchGovernanceDelegationCandidates.mockResolvedValue({ items: [{ principal_id: "actor-1", display_name: "Ada Okafor", context_label: "Risk lead", can_give: true, can_receive: true }, { principal_id: "person-2", display_name: "Tunde Bello", context_label: "Reviewer", can_give: false, can_receive: true }], has_more: false });
  loadGovernanceInventory.mockResolvedValue({
    policies: [{ id: "policy-1", code: "PAYMENT", name: "Payment approval", status: "DRAFT", legal_entity_id: "entity-1", current_version: 1, version: 2, maker_id: "actor-1" }],
    delegations: [],
    policiesAvailable: true,
    delegationsAvailable: true,
  });
  transitionGovernancePolicy.mockResolvedValue({ id: "policy-1", code: "PAYMENT", name: "Payment approval", status: "PENDING_APPROVAL", legal_entity_id: "entity-1", current_version: 1, version: 3, maker_id: "actor-1" });
});

it("waits for verified context before requesting governance or directory data", async () => {
  let resolve!: (value: RuntimeContext) => void;
  loadContext.mockReturnValueOnce(new Promise<RuntimeContext>((done) => { resolve = done; }));
  render(<GovernanceAdminPanel view="policies"/>);
  expect(loadGovernanceInventory).not.toHaveBeenCalled();
  expect(loadIdentityAccessOverview).not.toHaveBeenCalled();
  await act(async () => { resolve(contextFor()); });
  expect(await screen.findByRole("heading", { name: "Payment approval" })).toBeTruthy();
  expect(loadGovernanceInventory.mock.calls[0]?.[0]).toBeInstanceOf(AbortSignal);
});

it("uses the supplied verified runtime context without another context request", async () => {
  render(<GovernanceAdminPanel view="policies" runtimeContext={contextFor()}/>);
  expect(await screen.findByRole("heading", { name: "Payment approval" })).toBeTruthy();
  expect(loadContext).not.toHaveBeenCalled();
});

it("keeps returned policy rules and the checksum visible through the runtime adapter", async () => {
  const definition = { rules: [{ id: "recorded-rule", responsibility: "REVIEWER", object_type: "MATTER", min_materiality: 0, selector: { kind: "ROLE", ref: "CRO" } }], escalations: [] };
  loadGovernanceInventory.mockResolvedValueOnce({ policies: [{ id: "policy-1", code: "PAYMENT", name: "Payment approval", status: "PENDING_APPROVAL", legal_entity_id: "entity-1", current_version: 3, version: 8, maker_id: "person-2", definition, checksum: "b".repeat(64) }], delegations: [], policiesAvailable: true, delegationsAvailable: true });
  render(<GovernanceAdminPanel view="policies" runtimeContext={contextFor()}/>);
  expect(await screen.findByRole("heading", { name: "Rules · Policy version 3" })).toBeTruthy();
  expect(screen.getByText("CRO")).toBeTruthy();
  expect(screen.getByText("b".repeat(64))).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "View full policy definition" }));
  expect(JSON.parse(screen.getByRole("region", { name: "Full policy definition" }).textContent!)).toEqual(definition);
});

it("does not load protected lists when runtime governance read access is denied", async () => {
  render(<GovernanceAdminPanel view="policies" runtimeContext={{ ...contextFor(), capabilities: { config_read: false, config_write: false } }}/>);
  expect(await screen.findByRole("heading", { name: "Approval routes access restricted" })).toBeTruthy();
  expect(loadGovernanceInventory).not.toHaveBeenCalled();
  expect(loadIdentityAccessOverview).not.toHaveBeenCalled();
});

it("revalidates on return and clears a population whose read access was revoked", async () => {
  const context = contextFor();
  const view = render(<GovernanceAdminPanel view="policies" runtimeContext={context} active/>);
  expect(await screen.findByRole("heading", { name: "Payment approval" })).toBeTruthy();
  loadGovernanceInventory.mockResolvedValueOnce({ policies: [], delegations: [], policiesAvailable: false, delegationsAvailable: true, policiesState: "restricted", delegationsState: "ready" });
  view.rerender(<GovernanceAdminPanel view="policies" runtimeContext={context} active={false}/>);
  view.rerender(<GovernanceAdminPanel view="policies" runtimeContext={context} active/>);
  expect(await screen.findByRole("heading", { name: "Approval routes access restricted" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Payment approval" })).toBeNull();
  expect(screen.queryByRole("button", { name: /Inspect policy/ })).toBeNull();
  expect(loadGovernanceInventory).toHaveBeenCalledTimes(2);
});

it("retains the selected record during a transient revalidation failure and pauses commands", async () => {
  const context = contextFor();
  const view = render(<GovernanceAdminPanel view="policies" runtimeContext={context} selectedPolicyID="policy-1" active/>);
  expect(await screen.findByRole("heading", { name: "Payment approval" })).toBeTruthy();
  loadGovernanceInventory.mockResolvedValueOnce({ policies: [], delegations: [], policiesAvailable: false, delegationsAvailable: true, policiesState: "unavailable", delegationsState: "ready" });
  view.rerender(<GovernanceAdminPanel view="policies" runtimeContext={context} selectedPolicyID="policy-1" active={false}/>);
  view.rerender(<GovernanceAdminPanel view="policies" runtimeContext={context} selectedPolicyID="policy-1" active/>);
  await screen.findByText(/Routing policies are unavailable.*Previously loaded records may be out of date/);
  expect(screen.getByRole("heading", { name: "Payment approval" })).toBeTruthy();
  expect((screen.getByRole("button", { name: "Submit Payment approval for approval" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.getByRole("button", { name: "Reload governance records" })).toBeTruthy();
});

it("aborts obsolete scope reads and never displays their late records in another entity", async () => {
  let resolveOld!: (value: GovernanceInventory) => void;
  loadGovernanceInventory.mockReturnValueOnce(new Promise<GovernanceInventory>((done) => { resolveOld = done; }));
  const view = render(<GovernanceAdminPanel view="policies" runtimeContext={contextFor()}/>);
  await waitFor(() => expect(loadGovernanceInventory).toHaveBeenCalledTimes(1));
  const signal = loadGovernanceInventory.mock.calls[0]![0] as AbortSignal;
  loadGovernanceInventory.mockResolvedValueOnce({ policies: [], delegations: [], policiesAvailable: true, delegationsAvailable: true });
  view.rerender(<GovernanceAdminPanel view="policies" runtimeContext={contextFor("entity-2")}/>);
  expect(signal.aborted).toBe(true);
  await screen.findByRole("heading", { name: "No routing policies returned" });
  await act(async () => { resolveOld({ policies: [{ id: "old", tenant_id: "bank", legal_entity_id: "entity-1", code: "OLD", name: "Old entity policy", status: "ACTIVE", current_version: 1, version: 1, maker_id: "actor-1" }], delegations: [], policiesAvailable: true, delegationsAvailable: true }); });
  expect(screen.queryByText("Old entity policy")).toBeNull();
});

it("keeps people outside the loaded directory identifiable by their distinct references", async () => {
  loadGovernanceInventory.mockResolvedValueOnce({ policies: [], delegations: [{ id: "delegation-outside-list", tenant_id: "bank", legal_entity_id: "entity-1", from_principal_id: "person-beyond-50", to_principal_id: "person-beyond-51", responsibility: "REVIEWER", scope: { legal_entity_id: "entity-1" }, starts_at: "2026-10-01T00:00:00Z", ends_at: "2026-11-01T00:00:00Z", status: "PENDING_APPROVAL", reason: "Leave cover", maker_id: "person-beyond-50", version: 3 }], policiesAvailable: true, delegationsAvailable: true });
  render(<GovernanceAdminPanel view="delegations" runtimeContext={contextFor()}/>);
  expect(await screen.findByRole("heading", { name: "Name unavailable · person-beyond-50 to Name unavailable · person-beyond-51" })).toBeTruthy();
  expect(screen.getAllByText("Name unavailable · person-beyond-50").length).toBeGreaterThan(0);
  expect(screen.getAllByText("Name unavailable · person-beyond-51").length).toBeGreaterThan(0);
});

it("refreshes a returned view when its earlier command finishes after reactivation", async () => {
  const context = contextFor();
  const before = { id: "policy-1", tenant_id: "bank", legal_entity_id: "entity-1", code: "PAYMENT", name: "Payment approval", status: "DRAFT", current_version: 1, version: 2, maker_id: "actor-1" };
  const after = { ...before, status: "PENDING_APPROVAL", version: 3 };
  loadGovernanceInventory
    .mockResolvedValueOnce({ policies: [before], delegations: [], policiesAvailable: true, delegationsAvailable: true })
    .mockResolvedValueOnce({ policies: [before], delegations: [], policiesAvailable: true, delegationsAvailable: true })
    .mockResolvedValueOnce({ policies: [after], delegations: [], policiesAvailable: true, delegationsAvailable: true });
  let finish!: (value: typeof after) => void;
  transitionGovernancePolicy.mockReturnValueOnce(new Promise<typeof after>((resolve) => { finish = resolve; }));
  const view = render(<GovernanceAdminPanel view="policies" runtimeContext={context} active/>);
  fireEvent.click(await screen.findByRole("button", { name: "Submit Payment approval for approval" }));
  await waitFor(() => expect(transitionGovernancePolicy).toHaveBeenCalledTimes(1));
  view.rerender(<GovernanceAdminPanel view="policies" runtimeContext={context} active={false}/>);
  view.rerender(<GovernanceAdminPanel view="policies" runtimeContext={context} active/>);
  await waitFor(() => expect(loadGovernanceInventory).toHaveBeenCalledTimes(2));
  expect((screen.getByRole("button", { name: "Submitting…" }) as HTMLButtonElement).disabled).toBe(true);
  await act(async () => { finish(after); });
  await screen.findByText(/Another authorized person must approve Payment approval/);
  expect(loadGovernanceInventory).toHaveBeenCalledTimes(3);
});

function contextFor(entity = "entity-1"): RuntimeContext {
  return { tenant: { id: "bank", name: "ClearSight Bank" }, legal_entity: { id: entity, name: entity === "entity-1" ? "ClearSight Bank Plc" : "ClearSight Finance" }, actor: { id: "actor-1", name: "Ada Okafor" }, mode: "production", capabilities: { config_read: true, config_write: true, identity_read: true } };
}

it("uses server-scoped people and labelled roles for explicit creation workflows", async () => {
  createGovernancePolicyDraft.mockResolvedValue({ id: "policy-2", code: "NEW_ROUTE", name: "New route", status: "DRAFT", legal_entity_id: "entity-1", current_version: 1, version: 1, maker_id: "actor-1" });
  render(<GovernanceAdminPanel/>);
  await screen.findByRole("heading", { name: "Payment approval" });

  fireEvent.click(screen.getByRole("button", { name: "New delegation" }));
  fireEvent.click(screen.getByRole("button", { name: /Responsibility/ }));
  fireEvent.click(await screen.findByRole("option", { name: "Reviewer" }));
  await waitFor(() => expect(searchGovernanceDelegationCandidates).toHaveBeenCalledWith("REVIEWER", ""));
  await waitFor(() => expect((screen.getByRole("button", { name: /Person giving authority/ }) as HTMLButtonElement).disabled).toBe(false));
  fireEvent.click(screen.getByRole("button", { name: /Person giving authority/ }));
  fireEvent.click(await screen.findByRole("option", { name: "Ada Okafor · Risk lead" }));

  fireEvent.click(screen.getByRole("button", { name: "Close delegation form" }));
  fireEvent.click(screen.getByRole("button", { name: "New routing policy" }));
  fireEvent.click(screen.getByRole("button", { name: /Responsible role/ }));
  expect(await screen.findByRole("option", { name: "Control Assurance" })).toBeTruthy();
});

it("loads labelled entity-scoped governance records and refreshes after an action", async () => {
  render(<GovernanceAdminPanel/>);

  expect(await screen.findByRole("heading", { name: "Payment approval" })).toBeTruthy();
  expect(screen.getAllByText("ClearSight Bank Plc").length).toBeGreaterThan(0);
  expect(screen.getByText(/Made by Ada Okafor/)).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Submit Payment approval for approval" }));
  await waitFor(() => expect(transitionGovernancePolicy).toHaveBeenCalledWith("policy-1", "submit", 2, undefined));
  await waitFor(() => expect(loadGovernanceInventory).toHaveBeenCalledTimes(2));
});

it("keeps the inventory readable and marks authority labels degraded when identity administration is unavailable", async () => {
  loadIdentityAccessOverview.mockRejectedValue(new Error("unavailable"));

  render(<GovernanceAdminPanel/>);

  expect(await screen.findByRole("heading", { name: "Payment approval" })).toBeTruthy();
  expect(screen.getByRole("alert").textContent).toMatch(/people and legal-entity labels could not be confirmed.*records remain available.*changes are disabled/i);
  expect((screen.getByRole("button", { name: "Submit Payment approval for approval" }) as HTMLButtonElement).disabled).toBe(true);
});

it("shows the latest rejection reason, reviewer label, date, and resulting record version", async () => {
  loadGovernanceInventory.mockResolvedValue({
    policies: [{
      id: "policy-1", code: "PAYMENT", name: "Payment approval", status: "DRAFT", legal_entity_id: "entity-1", current_version: 1, version: 5, maker_id: "actor-1",
      latest_decision: { from_state: "PENDING_APPROVAL", to_state: "DRAFT", actor_id: "person-2", rationale: "Limit the route to payment issues.", decided_at: "2026-08-26T10:00:00Z", record_version: 5 },
    }],
    delegations: [], policiesAvailable: true, delegationsAvailable: true,
  });

  render(<GovernanceAdminPanel/>);

  expect(await screen.findByText(/Returned for changes by Tunde Bello/)).toBeTruthy();
  expect(screen.getByText(/Limit the route to payment issues\./)).toBeTruthy();
  expect(screen.getByText(/Record version 5/)).toBeTruthy();
});
