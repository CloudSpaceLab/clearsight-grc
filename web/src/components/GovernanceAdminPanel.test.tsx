import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { GovernanceAdminPanel } from "./GovernanceAdminPanel";

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

it("uses server-scoped people and labelled roles for creation workflows", async () => {
  createGovernancePolicyDraft.mockResolvedValue({ id: "policy-2", code: "NEW_ROUTE", name: "New route", status: "DRAFT", legal_entity_id: "entity-1", current_version: 1, version: 1, maker_id: "actor-1" });
  render(<GovernanceAdminPanel/>);
  await screen.findByRole("heading", { name: "Payment approval" });

  fireEvent.change(screen.getByRole("combobox", { name: "Responsibility" }), { target: { value: "REVIEWER" } });
  await waitFor(() => expect(searchGovernanceDelegationCandidates).toHaveBeenCalledWith("REVIEWER", ""));
  expect((await screen.findAllByRole("option", { name: "Ada Okafor · Risk lead" })).length).toBeGreaterThan(0);
  expect(screen.getByRole("option", { name: "Control Assurance" })).toBeTruthy();
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
