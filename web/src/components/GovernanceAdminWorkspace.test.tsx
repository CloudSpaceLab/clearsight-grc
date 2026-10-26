import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { GovernanceAdminWorkspace, type GovernanceDelegationItem, type GovernancePolicyItem } from "./GovernanceAdminWorkspace";

const viewActions = { createDelegation: vi.fn(), policyAction: vi.fn(), delegationAction: vi.fn() };

it("uses one bounded policy list and selected context in the People workspace", () => {
  const policies = Array.from({ length: 51 }, (_, index) => policy({ id: `p-${index}`, name: `Route ${index}`, version: index + 1 }));
  render(<GovernanceAdminWorkspace view="policies" policies={policies} delegations={[]} eligiblePeople={[]} actorId="actor-1" canConfigure={false} loadState="ready" selectedPolicyID="p-3" {...viewActions}/>);
  expect(screen.getAllByRole("button", { name: /^Inspect policy / })).toHaveLength(50);
  expect(screen.queryByRole("button", { name: "Inspect policy Route 50" })).toBeNull();
  expect(screen.getByRole("heading", { name: "Route 3" })).toBeTruthy();
  expect(screen.getByText("More records are available.")).toBeTruthy();
  expect(screen.getByRole("article", { name: "Policies: 50. Loaded" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Governance policies and delegations" })).toBeNull();
});

it("keeps an unknown selected policy separate from the first returned policy", () => {
  const onSelectPolicy = vi.fn();
  render(<GovernanceAdminWorkspace view="policies" policies={[policy({ name: "Payment route", maker: { id: "actor-1", label: "Current administrator" }, status: "DRAFT" })]} delegations={[]} eligiblePeople={[]} actorId="actor-1" canConfigure loadState="ready" selectedPolicyID="not-returned" onSelectPolicy={onSelectPolicy} {...viewActions}/>);
  expect(screen.getByRole("heading", { name: "Policy not in this list" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: "Submit Payment route for approval" })).toBeNull();
  fireEvent.click(screen.getByRole("button", { name: "Return to listed routing policies" }));
  expect(onSelectPolicy).toHaveBeenCalledWith(undefined);
});

it("restores the implicit first policy when controlled navigation returns to a queryless route", () => {
  const props = { view: "policies" as const, policies: [policy({ id: "first", name: "First policy" }), policy({ id: "second", name: "Second policy" })], delegations: [], eligiblePeople: [], actorId: "actor-1", canConfigure: false, loadState: "ready" as const, onSelectPolicy: vi.fn(), ...viewActions };
  const view = render(<GovernanceAdminWorkspace {...props}/>);
  fireEvent.click(screen.getByRole("button", { name: "Inspect policy Second policy" }));
  expect(props.onSelectPolicy).toHaveBeenCalledWith("second");
  view.rerender(<GovernanceAdminWorkspace {...props} selectedPolicyID="second"/>);
  expect(screen.getByRole("heading", { name: "Second policy" })).toBeTruthy();
  view.rerender(<GovernanceAdminWorkspace {...props}/>);
  expect(screen.getByRole("heading", { name: "First policy" })).toBeTruthy();
});

it("restores the implicit first delegation when controlled navigation returns to a queryless route", () => {
  const props = { view: "delegations" as const, policies: [], delegations: [delegation({ id: "first" }), delegation({ id: "second", from: { id: "second-person", label: "Second person" } })], eligiblePeople: [], actorId: "actor-1", canConfigure: false, loadState: "ready" as const, onSelectDelegation: vi.fn(), ...viewActions };
  const view = render(<GovernanceAdminWorkspace {...props}/>);
  fireEvent.click(screen.getByRole("button", { name: "Inspect delegation Second person to Tunde Bello" }));
  expect(props.onSelectDelegation).toHaveBeenCalledWith("second");
  view.rerender(<GovernanceAdminWorkspace {...props} selectedDelegationID="second"/>);
  expect(screen.getByRole("heading", { name: "Second person to Tunde Bello" })).toBeTruthy();
  view.rerender(<GovernanceAdminWorkspace {...props}/>);
  expect(screen.getByRole("heading", { name: "Ada Okafor to Tunde Bello" })).toBeTruthy();
});

it("shows current policy rules without inferring approval stages and preserves the complete definition", () => {
  const definition = { rules: [
    { id: "payment-review", legal_entity_id: "entity-1", object_type: "MATTER", object_id: "issue-7", responsibility: "REVIEWER", decision_type: "matter.close", min_materiality: 0, priority: 100, selector: { kind: "ROLE", ref: "CRO" } },
    { id: "response-check", responsibility: "REVIEWER", lifecycle_type: "RESPONSE", lifecycle_state: "PREPARED", lifecycle_subtype: "REGULATORY", min_materiality: 2, priority: 0 },
  ], escalations: [{ id: "late-review", trigger: "OVERDUE", steps: [{ after: "0s", responsibility: "ACCOUNTABLE_OWNER", department_levels_up: 0 }] }], future_setting: { retained: true } };
  render(<GovernanceAdminWorkspace view="policies" policies={[policy({ definition, checksum: "a".repeat(64) })]} delegations={[]} eligiblePeople={[]} actorId="actor-1" canConfigure={false} loadState="ready" {...viewActions}/>);
  const detail = screen.getByRole("complementary");
  expect(within(detail).getByRole("heading", { name: "Rules · Policy version 1" })).toBeTruthy();
  for (const text of ["CRO", "Issues and changes", "issue-7", "matter.close", "Response", "Prepared", "REGULATORY", "a".repeat(64)]) expect(within(detail).getByText(text)).toBeTruthy();
  expect(within(detail).getAllByText("Minimum materiality")[0]?.nextElementSibling?.textContent).toBe("0");
  expect(within(detail).getAllByText("Priority")[0]?.nextElementSibling?.textContent).toBe("100");
  const toggle = within(detail).getByRole("button", { name: "View full policy definition" });
  expect(toggle.getAttribute("aria-expanded")).toBe("false");
  fireEvent.click(toggle);
  expect(JSON.parse(within(detail).getByRole("region", { name: "Full policy definition" }).textContent!)).toEqual(definition);
  expect(within(detail).getByRole("button", { name: "Hide policy definition" }).getAttribute("aria-expanded")).toBe("true");
});

it.each([undefined, null, { rules: [] }, { rules: [{ id: "bad", responsibility: "REVIEWER", selector: { kind: "ROLE" } }] }, { rules: [{ id: "bad", responsibility: "REVIEWER", lifecycle_type: "RESPONSE" }] }])("does not approve a policy whose returned rules cannot be reviewed: %j", (definition) => {
  const policyAction = vi.fn();
  render(<GovernanceAdminWorkspace view="policies" policies={[policy({ status: "PENDING_APPROVAL", definition })]} delegations={[]} eligiblePeople={[]} actorId="checker-1" canConfigure loadState="ready" {...viewActions} policyAction={policyAction}/>);
  const approve = screen.getByRole("button", { name: "Approve Payment route" }) as HTMLButtonElement;
  expect(approve.disabled).toBe(true);
  expect(screen.getByText(/Approval is unavailable until the rules can be reviewed/)).toBeTruthy();
  fireEvent.click(approve);
  expect(policyAction).not.toHaveBeenCalled();
  expect(screen.getByRole("textbox", { name: "Changes needed for Payment route" })).toBeTruthy();
});

it("shows both exact instants for a same-day delegation with an explicit timezone", () => {
  render(<GovernanceAdminWorkspace view="delegations" policies={[]} delegations={[delegation({ startsAt: "2026-11-01T09:00:00+01:00", endsAt: "2026-11-01T18:00:00+01:00" })]} eligiblePeople={[]} actorId="actor-1" canConfigure={false} loadState="ready" {...viewActions}/>);
  const detail = screen.getByRole("complementary");
  const dates = within(detail).getByText("Effective dates").nextElementSibling?.textContent;
  expect(dates).toMatch(/08:00:00 UTC/);
  expect(dates).toMatch(/17:00:00 UTC/);
});

it("does not show denied rows or a successful empty result in a source-shaped view", () => {
  render(<GovernanceAdminWorkspace view="policies" policies={[policy({ name: "Restricted route" })]} delegations={[]} eligiblePeople={[]} actorId="actor-1" canConfigure loadState="degraded" policiesState="restricted" {...viewActions}/>);
  expect(screen.getByRole("heading", { name: "Approval routes access restricted" })).toBeTruthy();
  expect(screen.queryByRole("button", { name: /Inspect policy/ })).toBeNull();
  expect(screen.queryByText("No routing policies returned")).toBeNull();
  expect(screen.getByRole("article", { name: "Policies: —. Unavailable" })).toBeTruthy();
});

it("preserves unavailable policy details while disabling material actions", () => {
  render(<GovernanceAdminWorkspace view="policies" policies={[policy({ name: "Payment route", status: "DRAFT", maker: { id: "actor-1", label: "Current administrator" } })]} delegations={[]} eligiblePeople={[]} actorId="actor-1" canConfigure loadState="degraded" policiesState="unavailable" {...viewActions}/>);
  expect(screen.getByRole("heading", { name: "Payment route" })).toBeTruthy();
  expect((screen.getByRole("button", { name: "Submit Payment route for approval" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.getByRole("article", { name: "Policies: 1. Previous list" })).toBeTruthy();
});

it("retains narrower delegation scope instead of describing it as whole-entity authority", () => {
  render(<GovernanceAdminWorkspace view="delegations" policies={[]} delegations={[delegation({ scope: { legal_entity_id: "entity-1", object_type: "MATTER", object_id: "matter-123", decision_type: "matter.close", min_materiality: 2, max_materiality: 3 } })]} eligiblePeople={[]} actorId="actor-1" canConfigure={false} loadState="ready" {...viewActions}/>);
  const detail = screen.getByRole("complementary");
  expect(within(detail).getByText("One record · Issues and changes")).toBeTruthy();
  expect(within(detail).getByText("matter-123")).toBeTruthy();
  expect(within(detail).getByText("matter.close")).toBeTruthy();
  expect(within(detail).queryByText("Whole legal entity")).toBeNull();
});

it("closes a retained view's composer when the route becomes inactive", () => {
  const props = { view: "delegations" as const, policies: [], delegations: [], eligiblePeople: [], actorId: "actor-1", canConfigure: true, loadState: "ready" as const, ...viewActions };
  const view = render(<GovernanceAdminWorkspace {...props} active/>);
  fireEvent.click(screen.getByRole("button", { name: "Propose delegation" }));
  expect(screen.getByRole("dialog", { name: "New delegation" })).toBeTruthy();
  view.rerender(<GovernanceAdminWorkspace {...props} active={false}/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  view.rerender(<GovernanceAdminWorkspace {...props} active/>);
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("does not announce zero policy records before the initial read completes", () => {
  render(<GovernanceAdminWorkspace view="policies" policies={[]} delegations={[]} eligiblePeople={[]} actorId="actor-1" canConfigure={false} loadState="loading" {...viewActions}/>);
  expect(screen.getByText("Count unavailable")).toBeTruthy();
  expect(screen.queryByText("0 routing policies in this list")).toBeNull();
});

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

it("keeps creation composers closed until an administrator explicitly opens one", () => {
  render(<GovernanceAdminWorkspace
    policies={[]}
    delegations={[]}
    eligiblePeople={[]}
    currentEntity={{ id: "entity-1", label: "ClearSight Bank Plc" }}
    policyRoles={[{ code: "CONTROL_ASSURANCE", label: "Control Assurance" }]}
    actorId="actor-1"
    canConfigure
    loadState="ready"
    createDelegation={vi.fn()}
    createPolicyDraft={vi.fn()}
    policyAction={vi.fn()}
    delegationAction={vi.fn()}
  />);

  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByRole("button", { name: /Legal entity/ })).toBeNull();
  expect(screen.queryByRole("textbox", { name: /^Policy name/ })).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "New delegation" }));
  expect(screen.getByRole("dialog", { name: "New delegation" })).toBeTruthy();
  expect(screen.getByRole("button", { name: /Legal entity/ })).toBeTruthy();
  expect(screen.queryByRole("textbox", { name: /^Policy name/ })).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "Close delegation form" }));
  expect(screen.queryByRole("dialog")).toBeNull();

  fireEvent.click(screen.getByRole("button", { name: "New routing policy" }));
  expect(screen.getByRole("dialog", { name: "New routing policy" })).toBeTruthy();
  expect(screen.getByRole("textbox", { name: /^Policy name/ })).toBeTruthy();
  expect(screen.queryByRole("button", { name: /Legal entity/ })).toBeNull();
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

  fireEvent.click(screen.getByRole("button", { name: "New delegation" }));
  await choose("Legal entity", "ClearSight Bank Plc");
  fireEvent.click(screen.getByRole("button", { name: /Person giving authority/ }));
  expect(screen.queryByRole("option", { name: "Nneka Eze" })).toBeNull();
  fireEvent.click(screen.getByRole("option", { name: "Ada Okafor" }));
  await choose("Person receiving authority", "Tunde Bello");
  await choose("Responsibility", "Reviewer");
  expect(screen.getByLabelText(/^Starts at/).getAttribute("type")).toBe("datetime-local");
  expect(screen.getByLabelText(/^Ends at/).getAttribute("type")).toBe("datetime-local");
  fireEvent.change(screen.getByLabelText(/^Starts at/), { target: { value: "2026-09-01T09:00" } });
  fireEvent.change(screen.getByLabelText(/^Ends at/), { target: { value: "2026-09-08T17:00" } });
  fireEvent.change(screen.getByRole("textbox", { name: /^Reason/ }), { target: { value: "Annual leave cover for payment review" } });
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
  expect(screen.queryByRole("dialog", { name: "New delegation" })).toBeNull();
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

  fireEvent.click(screen.getByRole("button", { name: "New delegation" }));
  await choose("Responsibility", "Reviewer");
  await waitFor(() => expect(loadDelegationCandidates).toHaveBeenCalledWith("REVIEWER", ""));
  await choose("Person giving authority", "Ada Okafor · Risk assurance lead");
  await choose("Person receiving authority", "Tunde Bello · Payment reviewer");
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

  fireEvent.click(screen.getByRole("button", { name: "New routing policy" }));
  fireEvent.change(screen.getByRole("textbox", { name: /^Policy name/ }), { target: { value: "Payment review route" } });
  fireEvent.change(screen.getByRole("textbox", { name: /^Policy code/ }), { target: { value: "PAYMENT_REVIEW" } });
  await choose("Policy responsibility", "Reviewer");
  await choose("Responsible role", "Control Assurance");
  await choose("Applies to", "Issues and changes");
  fireEvent.change(screen.getByRole("spinbutton", { name: "Minimum materiality" }), { target: { value: "2" } });
  fireEvent.change(screen.getByLabelText("Effective from"), { target: { value: "2026-09-01T00:00" } });
  fireEvent.click(screen.getByRole("button", { name: "Create policy draft" }));

  await waitFor(() => expect(createPolicyDraft).toHaveBeenCalledWith({
    legalEntityId: "entity-1", code: "PAYMENT_REVIEW", name: "Payment review route", responsibility: "REVIEWER",
    roleCode: "CONTROL_ASSURANCE", objectType: "MATTER", decisionType: "", minMateriality: 2, priority: 100,
    effectiveFrom: new Date("2026-09-01T00:00").toISOString(),
  }));
  expect(screen.queryByRole("dialog", { name: "New routing policy" })).toBeNull();
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
  expect((screen.getByRole("button", { name: "New delegation" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.queryByRole("dialog")).toBeNull();
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

  expect(screen.getByRole("alert").textContent).toBe("Governance data unavailable. Refresh before making changes.");
  expect(screen.queryByText(/No routing policies were returned/)).toBeNull();
  expect(screen.queryByText(/No delegations were returned/)).toBeNull();
  expect((screen.getByRole("button", { name: "New delegation" }) as HTMLButtonElement).disabled).toBe(true);
  expect(screen.queryByRole("dialog")).toBeNull();
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
    definition: { rules: [{ id: "review", responsibility: "REVIEWER", selector: { kind: "ROLE", ref: "CONTROL_ASSURANCE" } }] }, checksum: "a".repeat(64),
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

async function choose(label: string, option: string) {
  const trigger = screen.getByRole("button", { name: new RegExp(label) }) as HTMLButtonElement;
  await waitFor(() => expect(trigger.disabled).toBe(false));
  fireEvent.click(trigger);
  fireEvent.click(await screen.findByRole("option", { name: option }));
}
