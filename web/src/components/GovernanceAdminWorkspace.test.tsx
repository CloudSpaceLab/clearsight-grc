import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { GovernanceAdminWorkspace, type GovernanceDelegationItem, type GovernancePolicyItem } from "./GovernanceAdminWorkspace";

it("shows a bounded governance inventory in business language", () => {
  render(<GovernanceAdminWorkspace
    policies={[{
      id: "policy-1", code: "HIGH_VALUE_PAYMENT", name: "High-value payment approval", status: "ACTIVE",
      legalEntity: { id: "entity-1", label: "ClearSight Bank Plc" }, currentVersion: 4, version: 7,
      effectiveFrom: "2026-08-01T00:00:00Z", maker: { id: "maker-1", label: "Ada Okafor" }, checker: { id: "checker-1", label: "Tunde Bello" },
    }]}
    delegations={[{
      id: "delegation-1", status: "ACTIVE", legalEntity: { id: "entity-1", label: "ClearSight Bank Plc" },
      from: { id: "person-1", label: "Ada Okafor" }, to: { id: "person-2", label: "Tunde Bello" }, responsibility: "AUTHORIZER",
      startsAt: "2026-08-20T08:00:00Z", endsAt: "2026-08-30T17:00:00Z", reason: "Annual leave cover", version: 3,
      maker: { id: "maker-1", label: "Ada Okafor" }, checker: { id: "checker-1", label: "Tunde Bello" },
    }]}
    eligiblePeople={[]}
    actorId="checker-1"
    canConfigure={false}
    loadState="ready"
    createDelegation={vi.fn()}
    policyAction={vi.fn()}
    delegationAction={vi.fn()}
  />);

  expect(screen.getByRole("heading", { name: "Governance policies and delegations" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "High-value payment approval" })).toBeTruthy();
  expect(screen.getAllByText("ClearSight Bank Plc").length).toBeGreaterThan(0);
  expect(screen.getAllByText("Active", { selector: "strong" })).toHaveLength(2);
  expect(screen.getByText(/Policy version 4 · record version 7/)).toBeTruthy();
  expect(screen.getAllByText(/Made by Ada Okafor · checked by Tunde Bello/)).toHaveLength(2);
  expect(screen.getByRole("heading", { name: "Ada Okafor to Tunde Bello" })).toBeTruthy();
  expect(screen.getByText("Authorizer", { selector: "span" })).toBeTruthy();
  expect(screen.queryByText(/JSON/i)).toBeNull();
});

it("offers policy submission or independent approval without allowing self-approval", async () => {
  const policyAction = vi.fn().mockResolvedValue(undefined);
  render(<GovernanceAdminWorkspace
    policies={[
      policy({ id: "draft", name: "Draft route", status: "DRAFT", maker: { id: "actor-1", label: "Current administrator" }, version: 2 }),
      policy({ id: "own-review", name: "Own proposed route", status: "PENDING_APPROVAL", maker: { id: "actor-1", label: "Current administrator" }, version: 5 }),
      policy({ id: "independent-review", name: "Independent route", status: "PENDING_APPROVAL", maker: { id: "maker-2", label: "Other administrator" }, version: 8 }),
    ]}
    delegations={[]}
    eligiblePeople={[]}
    actorId="actor-1"
    canConfigure
    loadState="ready"
    createDelegation={vi.fn()}
    policyAction={policyAction}
    delegationAction={vi.fn()}
  />);

  expect(screen.queryByRole("button", { name: "Approve Own proposed route" })).toBeNull();
  expect(screen.getByText(/Another authorized person must approve Own proposed route/)).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Submit Draft route for approval" }));
  await waitFor(() => expect(policyAction).toHaveBeenCalledWith({ policyId: "draft", action: "submit", expectedVersion: 2 }));

  fireEvent.click(screen.getByRole("button", { name: "Approve Independent route" }));
  await waitFor(() => expect(policyAction).toHaveBeenCalledWith({ policyId: "independent-review", action: "approve", expectedVersion: 8 }));
});

it("offers delegation submission or independent approval without maker, delegator, or delegate approval", async () => {
  const delegationAction = vi.fn().mockResolvedValue(undefined);
  render(<GovernanceAdminWorkspace
    policies={[]}
    delegations={[
      delegation({ id: "draft", status: "DRAFT", maker: { id: "actor-1", label: "Current administrator" }, version: 2 }),
      delegation({ id: "participant-review", status: "PENDING_APPROVAL", from: { id: "actor-1", label: "Current administrator" }, maker: { id: "maker-2", label: "Other administrator" }, version: 4 }),
      delegation({ id: "independent-review", status: "PENDING_APPROVAL", maker: { id: "maker-2", label: "Other administrator" }, version: 6 }),
    ]}
    eligiblePeople={[]}
    actorId="actor-1"
    canConfigure
    loadState="ready"
    createDelegation={vi.fn()}
    policyAction={vi.fn()}
    delegationAction={delegationAction}
  />);

  expect(screen.queryByRole("button", { name: /Approve Current administrator to Tunde Bello/ })).toBeNull();
  expect(screen.getByText(/A person who did not make, give, or receive this delegation must approve it/)).toBeTruthy();

  fireEvent.click(screen.getByRole("button", { name: "Submit Ada Okafor to Tunde Bello for approval" }));
  await waitFor(() => expect(delegationAction).toHaveBeenCalledWith({ delegationId: "draft", action: "submit", expectedVersion: 2 }));

  fireEvent.click(screen.getByRole("button", { name: "Approve Ada Okafor to Tunde Bello" }));
  await waitFor(() => expect(delegationAction).toHaveBeenCalledWith({ delegationId: "independent-review", action: "approve", expectedVersion: 6 }));
});

it("creates a typed whole-entity delegation from labelled people and responsibility controls", async () => {
  const createDelegation = vi.fn().mockResolvedValue(undefined);
  render(<GovernanceAdminWorkspace
    policies={[]}
    delegations={[]}
    eligiblePeople={[
      { id: "person-ada", label: "Ada Okafor", legalEntity: { id: "entity-1", label: "ClearSight Bank Plc" } },
      { id: "person-tunde", label: "Tunde Bello", legalEntity: { id: "entity-1", label: "ClearSight Bank Plc" } },
      { id: "person-nneka", label: "Nneka Eze", legalEntity: { id: "entity-2", label: "ClearSight Microfinance Bank" } },
    ]}
    actorId="actor-1"
    canConfigure
    loadState="ready"
    createDelegation={createDelegation}
    policyAction={vi.fn()}
    delegationAction={vi.fn()}
  />);

  fireEvent.change(screen.getByRole("combobox", { name: "Legal entity" }), { target: { value: "entity-1" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Person giving authority" }), { target: { value: "person-ada" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Person receiving authority" }), { target: { value: "person-tunde" } });
  expect(screen.queryByRole("option", { name: "Nneka Eze" })).toBeNull();
  fireEvent.change(screen.getByRole("combobox", { name: "Responsibility" }), { target: { value: "REVIEWER" } });
  expect(screen.getByLabelText("Starts at").getAttribute("type")).toBe("datetime-local");
  expect(screen.getByLabelText("Ends at").getAttribute("type")).toBe("datetime-local");
  fireEvent.change(screen.getByLabelText("Starts at"), { target: { value: "2026-09-01T09:00" } });
  fireEvent.change(screen.getByLabelText("Ends at"), { target: { value: "2026-09-08T17:00" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Reason" }), { target: { value: "Annual leave cover for payment review" } });
  fireEvent.click(screen.getByRole("button", { name: "Create delegation draft" }));

  await waitFor(() => expect(createDelegation).toHaveBeenCalledWith({
    legalEntityId: "entity-1",
    fromPrincipalId: "person-ada",
    toPrincipalId: "person-tunde",
    responsibility: "REVIEWER",
    scope: { kind: "LEGAL_ENTITY" },
    startsAt: new Date("2026-09-01T09:00").toISOString(),
    endsAt: new Date("2026-09-08T17:00").toISOString(),
    reason: "Annual leave cover for payment review",
  }));
  expect(screen.queryByText(/person-ada|person-tunde|\{.*kind.*\}/i)).toBeNull();
});

it("loads responsibility-scoped labelled people before creating a delegation", async () => {
  const loadDelegationCandidates = vi.fn().mockResolvedValue({
    items: [
      { principal_id: "person-ada", display_name: "Ada Okafor", context_label: "Risk assurance lead", can_give: true, can_receive: true },
      { principal_id: "person-tunde", display_name: "Tunde Bello", context_label: "Payment reviewer", can_give: false, can_receive: true },
    ], has_more: false,
  });
  const createDelegation = vi.fn().mockResolvedValue(undefined);
  render(<GovernanceAdminWorkspace
    policies={[]} delegations={[]} eligiblePeople={[]} currentEntity={{ id: "entity-1", label: "ClearSight Bank Plc" }} policyRoles={[]}
    actorId="actor-1" canConfigure loadState="ready" loadDelegationCandidates={loadDelegationCandidates}
    createDelegation={createDelegation} createPolicyDraft={vi.fn()} policyAction={vi.fn()} delegationAction={vi.fn()}
  />);

  fireEvent.change(screen.getByRole("combobox", { name: "Responsibility" }), { target: { value: "REVIEWER" } });
  await waitFor(() => expect(loadDelegationCandidates).toHaveBeenCalledWith("REVIEWER", ""));
  expect((await screen.findAllByRole("option", { name: "Ada Okafor · Risk assurance lead" })).length).toBeGreaterThan(0);
  expect(screen.getByRole("option", { name: "Tunde Bello · Payment reviewer" })).toBeTruthy();
  fireEvent.change(screen.getByRole("combobox", { name: "Person giving authority" }), { target: { value: "person-ada" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Person receiving authority" }), { target: { value: "person-tunde" } });
  expect(screen.queryByText(/person-ada|person-tunde/)).toBeNull();
});

it("creates a routing policy draft from business fields and labelled role choices", async () => {
  const createPolicyDraft = vi.fn().mockResolvedValue(undefined);
  render(<GovernanceAdminWorkspace
    policies={[]} delegations={[]} eligiblePeople={[]} currentEntity={{ id: "entity-1", label: "ClearSight Bank Plc" }}
    policyRoles={[{ code: "CONTROL_ASSURANCE", label: "Control Assurance" }]}
    actorId="actor-1" canConfigure loadState="ready" createDelegation={vi.fn()} createPolicyDraft={createPolicyDraft}
    policyAction={vi.fn()} delegationAction={vi.fn()}
  />);

  fireEvent.change(screen.getByRole("textbox", { name: "Policy name" }), { target: { value: "Payment review route" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Policy code" }), { target: { value: "PAYMENT_REVIEW" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Policy responsibility" }), { target: { value: "REVIEWER" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Responsible role" }), { target: { value: "CONTROL_ASSURANCE" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Applies to" }), { target: { value: "MATTER" } });
  fireEvent.change(screen.getByRole("spinbutton", { name: "Minimum materiality" }), { target: { value: "2" } });
  fireEvent.change(screen.getByLabelText("Effective from"), { target: { value: "2026-09-01T00:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Create policy draft" }));

  await waitFor(() => expect(createPolicyDraft).toHaveBeenCalledWith({
    legalEntityId: "entity-1", code: "PAYMENT_REVIEW", name: "Payment review route", responsibility: "REVIEWER",
    roleCode: "CONTROL_ASSURANCE", objectType: "MATTER", decisionType: "", minMateriality: 2, priority: 100,
    effectiveFrom: new Date("2026-09-01T00:00").toISOString(),
  }));
  expect(screen.queryByText(/\{"rules"|CONTROL_ASSURANCE/)).toBeNull();
});

it("lets an independent checker return a pending policy with a reason", async () => {
  const policyAction = vi.fn().mockResolvedValue(undefined);
  render(<GovernanceAdminWorkspace
    policies={[policy({ id: "pending", name: "Payment route", status: "PENDING_APPROVAL", maker: { id: "maker-2", label: "Other administrator" }, version: 8 })]}
    delegations={[]} eligiblePeople={[]} actorId="actor-1" canConfigure loadState="ready"
    createDelegation={vi.fn()} policyAction={policyAction} delegationAction={vi.fn()}
  />);

  fireEvent.change(screen.getByRole("textbox", { name: "Changes needed for Payment route" }), { target: { value: "Narrow the route to payment issues." } });
  fireEvent.click(screen.getByRole("button", { name: "Return Payment route for changes" }));
  await waitFor(() => expect(policyAction).toHaveBeenCalledWith({ policyId: "pending", action: "reject", expectedVersion: 8, rationale: "Narrow the route to payment issues." }));
});

it("preserves loaded inventory and disables every mutation when authority is degraded", () => {
  const policyAction = vi.fn();
  render(<GovernanceAdminWorkspace
    policies={[policy({ id: "draft", name: "Payment approval route", status: "DRAFT", maker: { id: "actor-1", label: "Current administrator" } })]}
    delegations={[]}
    eligiblePeople={[{ id: "person-ada", label: "Ada Okafor", legalEntity: { id: "entity-1", label: "ClearSight Bank Plc" } }]}
    actorId="actor-1"
    canConfigure
    loadState="degraded"
    degradedReason="Current approval authority could not be confirmed."
    createDelegation={vi.fn()}
    policyAction={policyAction}
    delegationAction={vi.fn()}
  />);

  expect(screen.getByRole("heading", { name: "Payment approval route" })).toBeTruthy();
  expect(screen.getByRole("alert").textContent).toMatch(/Current approval authority could not be confirmed.*Existing records remain available/i);
  expect((screen.getByRole("button", { name: "Submit Payment approval route for approval" }) as HTMLButtonElement).disabled).toBe(true);
  expect((screen.getByRole("combobox", { name: "Legal entity" }) as HTMLSelectElement).disabled).toBe(true);
  expect((screen.getByRole("button", { name: "Create delegation draft" }) as HTMLButtonElement).disabled).toBe(true);
  expect(policyAction).not.toHaveBeenCalled();
});

it("states when the current legal-entity population is unavailable and no prior inventory exists", () => {
  render(<GovernanceAdminWorkspace
    policies={[]}
    delegations={[]}
    eligiblePeople={[]}
    actorId="actor-1"
    canConfigure
    loadState="unavailable"
    createDelegation={vi.fn()}
    policyAction={vi.fn()}
    delegationAction={vi.fn()}
  />);

  expect(screen.getByRole("alert").textContent).toBe("The current legal-entity governance population could not be loaded. Refresh this page to try again; no changes are available.");
  expect(screen.queryByText(/No routing policies were returned/)).toBeNull();
  expect(screen.queryByText(/No delegations were returned/)).toBeNull();
  expect((screen.getByRole("button", { name: "Create delegation draft" }) as HTMLButtonElement).disabled).toBe(true);
});

it("offers one reason-bearing terminal action for active records to an eligible actor", async () => {
  const policyAction = vi.fn().mockResolvedValue(undefined);
  const delegationAction = vi.fn().mockResolvedValue(undefined);
  render(<GovernanceAdminWorkspace
    policies={[policy({ id: "active-policy", name: "Legacy payment route", status: "ACTIVE", version: 9 })]}
    delegations={[delegation({ id: "active-delegation", status: "ACTIVE", from: { id: "actor-1", label: "Current administrator" }, checker: { id: "checker-1", label: "Independent checker" }, version: 5 })]}
    eligiblePeople={[]}
    actorId="actor-1"
    canConfigure
    loadState="ready"
    createDelegation={vi.fn()}
    policyAction={policyAction}
    delegationAction={delegationAction}
  />);

  fireEvent.change(screen.getByRole("textbox", { name: "Reason to retire Legacy payment route" }), { target: { value: "Superseded by the approved 2027 route" } });
  fireEvent.click(screen.getByRole("button", { name: "Retire Legacy payment route" }));
  await waitFor(() => expect(policyAction).toHaveBeenCalledWith({ policyId: "active-policy", action: "retire", expectedVersion: 9, rationale: "Superseded by the approved 2027 route" }));

  fireEvent.change(screen.getByRole("textbox", { name: "Reason to revoke Current administrator to Tunde Bello" }), { target: { value: "Leave ended early" } });
  fireEvent.click(screen.getByRole("button", { name: "Revoke Current administrator to Tunde Bello" }));
  await waitFor(() => expect(delegationAction).toHaveBeenCalledWith({ delegationId: "active-delegation", action: "revoke", expectedVersion: 5, rationale: "Leave ended early" }));
});

it("caps each inventory at fifty records and discloses the supplied total", () => {
  render(<GovernanceAdminWorkspace
    policies={Array.from({ length: 51 }, (_, index) => policy({ id: `policy-${index}`, name: `Route ${index}` }))}
    delegations={Array.from({ length: 51 }, (_, index) => delegation({ id: `delegation-${index}`, from: { id: `from-${index}`, label: `Delegator ${index}` } }))}
    eligiblePeople={[]}
    actorId="actor-1"
    canConfigure={false}
    loadState="ready"
    createDelegation={vi.fn()}
    policyAction={vi.fn()}
    delegationAction={vi.fn()}
  />);

  expect(screen.getByRole("heading", { name: "Route 49" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Route 50" })).toBeNull();
  expect(screen.getByText("Showing the first 50 routing policies. More records are available.")).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Delegator 49 to Tunde Bello" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Delegator 50 to Tunde Bello" })).toBeNull();
  expect(screen.getByText("Showing the first 50 delegations. More records are available.")).toBeTruthy();
});

function policy(overrides: Partial<GovernancePolicyItem> = {}): GovernancePolicyItem {
  return {
    id: "policy-1", code: "PAYMENT_ROUTE", name: "Payment route", status: "ACTIVE",
    legalEntity: { id: "entity-1", label: "ClearSight Bank Plc" }, currentVersion: 1, version: 1,
    maker: { id: "maker-1", label: "Ada Okafor" }, ...overrides,
  };
}

function delegation(overrides: Partial<GovernanceDelegationItem> = {}): GovernanceDelegationItem {
  return {
    id: "delegation-1", status: "ACTIVE", legalEntity: { id: "entity-1", label: "ClearSight Bank Plc" },
    from: { id: "person-1", label: "Ada Okafor" }, to: { id: "person-2", label: "Tunde Bello" }, responsibility: "AUTHORIZER",
    startsAt: "2026-08-20T08:00:00Z", endsAt: "2026-08-30T17:00:00Z", reason: "Annual leave cover", version: 1,
    maker: { id: "maker-1", label: "Ada Okafor" }, ...overrides,
  };
}
