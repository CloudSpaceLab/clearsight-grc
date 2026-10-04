import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { IdentityAccessPanel } from "./IdentityAccessPanel";

const api = vi.hoisted(() => ({
  loadIdentityAccessOverview: vi.fn(),
  createIdentitySource: vi.fn(),
  createGroupRoleBinding: vi.fn(),
  rotateIdentitySourceToken: vi.fn(),
  revokeIdentitySource: vi.fn(),
  retireGroupRoleBinding: vi.fn(),
  previewEscalation: vi.fn(),
  proposeEscalationGuardRevision: vi.fn(),
  approveEscalationGuardRevision: vi.fn(),
  proposeOrganizationScope: vi.fn(),
  approveOrganizationScope: vi.fn(),
  rejectOrganizationScope: vi.fn(),
  proposeOrganizationPosition: vi.fn(),
  approveOrganizationPosition: vi.fn(),
  rejectOrganizationPosition: vi.fn(),
  restoreOrganizationPosition: vi.fn(),
}));

vi.mock("../identityAccessApi", () => api);

beforeEach(() => {
  vi.clearAllMocks();
  api.loadIdentityAccessOverview.mockResolvedValue({
    sign_in: { mode: "oidc", issuer: "https://id.bank.test", assurance_level: "MFA" },
    actor_principal_id: "actor-1",
    can_configure: true,
    can_configure_organization: true,
    can_configure_escalation: false,
    sources: [{ id: "source-1", code: "ENTRA", status: "ACTIVE", subject_attribute: "externalId", active_users: 12, active_groups: 2 }],
    people: [
      { id: "person-cro", display_name: "Ada Okafor", status: "ACTIVE" },
      { id: "person-owner", display_name: "Chidi Eze", status: "ACTIVE" },
      { id: "person-new", display_name: "Nneka Obi", status: "ACTIVE" },
    ],
    groups: [{ id: "group-1", display_name: "Risk Operations", source_code: "ENTRA", source_state: "ACTIVE", member_count: 8 }],
    roles: [{ id: "role-1", code: "RISK_REVIEWER", name: "Risk reviewer", capabilities: ["program_read"] }],
    legal_entities: [],
    bindings: [],
    organization_scopes: [
      { id: "scope-risk", legal_entity_id: "entity-1", code: "RISK", name: "RISK", kind: "ORGANIZATION_UNIT", department_path: ["BANK", "RISK"], origin: "LEGACY_DEPARTMENT_PATH", status: "ACTIVE", valid_from: "2026-01-01T00:00:00Z", version: 1 },
      { id: "scope-operations", legal_entity_id: "entity-1", parent_scope_id: "scope-risk", code: "OPERATIONS", name: "OPERATIONS", kind: "ORGANIZATION_UNIT", department_path: ["BANK", "RISK", "OPERATIONS"], origin: "LEGACY_DEPARTMENT_PATH", status: "ACTIVE", valid_from: "2026-01-01T00:00:00Z", version: 1 },
    ],
    organization_scope_revisions: [],
    organization_position_revisions: [],
    organization_position_history: [],
    positions: [
      {
        id: "position-cro",
        code: "CRO",
        title: "Chief Risk Officer",
        function_name: "Risk",
        department_path: ["BANK", "RISK"],
        organization_scope_id: "scope-risk",
        occupant_principal_id: "person-cro",
        occupant_name: "Ada Okafor",
        occupant_status: "ACTIVE",
        role_codes: ["CRO"],
        valid_from: "2026-01-01T00:00:00Z",
        version: 3,
      },
      {
        id: "position-owner",
        code: "PROGRAM_OWNER",
        title: "Program Owner",
        function_name: "Risk Operations",
        department_path: ["BANK", "RISK", "OPERATIONS"],
        organization_scope_id: "scope-operations",
        parent_position_id: "position-cro",
        parent_position_code: "CRO",
        parent_position_title: "Chief Risk Officer",
        occupant_principal_id: "person-owner",
        occupant_name: "Chidi Eze",
        occupant_status: "ACTIVE",
        role_codes: ["PROGRAM_OWNER"],
        valid_from: "2026-01-01T00:00:00Z",
        version: 4,
      },
    ],
    escalation: { pending_timers: 0, escalated_tasks: 0, unresolved_24h: 0, failed_timers: 0 },
    escalation_policies: [],
  });
  api.createIdentitySource.mockResolvedValue({
    source: { id: "source-2", code: "OKTA", status: "ACTIVE", subject_attribute: "externalId", active_users: 0, active_groups: 0 },
    token: "one-time-token",
  });
  api.createGroupRoleBinding.mockResolvedValue({
    id: "binding-1",
    group_id: "group-1",
    group_name: "Risk Operations",
    role_template_id: "role-1",
    role_code: "RISK_REVIEWER",
    legal_entity_id: "entity-1",
    legal_entity: "ClearSight Bank Plc",
    department_path: ["BANK", "RISK"],
    valid_from: "2026-08-31T00:00:00Z",
  });
});

it("keeps access inventory primary and opens one focused creation workflow at a time", async () => {
  render(<IdentityAccessPanel/>);

  await screen.findByRole("heading", { name: "Organization & access" });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByRole("textbox", { name: "Code" })).toBeNull();
  expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1);

  expect(screen.getByRole("tab", { name: "Organization", selected: true })).toBeTruthy();
  expect(screen.getByText("Ada Okafor")).toBeTruthy();
  expect(screen.getByText("PROGRAM_OWNER")).toBeTruthy();
  expect(screen.getByText("Department / area")).toBeTruthy();
  expect(screen.getByText("Active organization scopes in this legal entity")).toBeTruthy();

  fireEvent.click(screen.getByRole("tab", { name: "Reporting lines" }));
  expect(screen.getByText((_, element) => element?.tagName === "P" && element.textContent === "Chidi Eze reports to Ada Okafor")).toBeTruthy();
  expect(screen.getByText(/Reporting lines do not grant approval authority/)).toBeTruthy();

  fireEvent.click(screen.getByRole("tab", { name: "Directory groups & access" }));

  fireEvent.click(screen.getByRole("button", { name: "Add source" }));
  const sourceDialog = screen.getByRole("dialog", { name: "Add provisioning source" });
  fireEvent.change(within(sourceDialog).getByRole("textbox", { name: "Code" }), { target: { value: "OKTA" } });
  fireEvent.click(within(sourceDialog).getByRole("button", { name: "Create source" }));

  await waitFor(() => expect(api.createIdentitySource).toHaveBeenCalledWith({ code: "OKTA", identity_issuer: undefined, subject_attribute: "externalId" }));
  await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add provisioning source" })).toBeNull());
  expect(screen.getByText("OKTA")).toBeTruthy();
  expect(screen.getByText("Provisioning token — shown once")).toBeTruthy();
  expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1);

  fireEvent.click(screen.getByRole("button", { name: "Add mapping" }));
  const bindingDialog = screen.getByRole("dialog", { name: "Add group role mapping" });
  fireEvent.change(within(bindingDialog).getByRole("combobox", { name: "Directory group" }), { target: { value: "group-1" } });
  fireEvent.change(within(bindingDialog).getByRole("combobox", { name: "Role" }), { target: { value: "role-1" } });
  fireEvent.change(within(bindingDialog).getByRole("textbox", { name: "Department path (optional)" }), { target: { value: "BANK / RISK" } });
  fireEvent.click(within(bindingDialog).getByRole("button", { name: "Add mapping" }));

  await waitFor(() => expect(api.createGroupRoleBinding).toHaveBeenCalledWith({ group_id: "group-1", role_template_id: "role-1", department_path: ["BANK", "RISK"] }));
  await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add group role mapping" })).toBeNull());
  expect(screen.getByText("Risk Operations → RISK_REVIEWER")).toBeTruthy();
  expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1);
});


it("proposes a new position with occupant and reporting line from the Organization tab", async () => {
  api.proposeOrganizationPosition.mockResolvedValue({
    id: "position-revision-1",
    position_id: "position-new",
    operation: "CREATE",
    base_version: 0,
    base: { code: "", title: "" },
    proposed: {
      code: "RISK_ANALYST",
      title: "Risk Analyst",
      function_name: "Risk",
      organization_scope_id: "scope-risk",
      parent_position_id: "position-cro",
      occupant_principal_id: "person-new",
    },
    maker_id: "actor-1",
    status: "PENDING",
    impact: { child_positions: 0, responsibility_assignments: 0, authority_grants: 0, active_role_bindings: 0, active_programs_owned: 0, open_matters_owned: 0, open_actions_owned: 0 },
    created_at: "2026-10-03T12:00:00Z",
  });

  render(<IdentityAccessPanel/>);
  await screen.findByRole("heading", { name: "Organization & access" });

  fireEvent.click(screen.getByRole("button", { name: "Add position" }));
  const dialog = screen.getByRole("dialog", { name: "Add position" });
  fireEvent.change(within(dialog).getByRole("textbox", { name: /^Code/ }), { target: { value: "RISK_ANALYST" } });
  fireEvent.change(within(dialog).getByRole("textbox", { name: /^Title/ }), { target: { value: "Risk Analyst" } });
  fireEvent.change(within(dialog).getByRole("textbox", { name: /^Function/ }), { target: { value: "Risk" } });
  const organizationArea = within(dialog).getByRole("button", { name: /Organization area/ });
  fireEvent.keyDown(organizationArea, { key: "ArrowDown" });
  fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "BANK / RISKOrganization Unit" }));
  const reportsTo = within(dialog).getByRole("button", { name: /Reports to/ });
  fireEvent.keyDown(reportsTo, { key: "ArrowDown" });
  fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "Chief Risk OfficerCRO" }));
  const occupant = within(dialog).getByRole("button", { name: /Current occupant/ });
  fireEvent.keyDown(occupant, { key: "ArrowDown" });
  fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "Nneka ObiActive" }));
  fireEvent.click(within(dialog).getByRole("button", { name: "Propose change" }));

  await waitFor(() => expect(api.proposeOrganizationPosition).toHaveBeenCalledWith({
    operation: "CREATE",
    code: "RISK_ANALYST",
    title: "Risk Analyst",
    function_name: "Risk",
    organization_scope_id: "scope-risk",
    parent_position_id: "position-cro",
    occupant_principal_id: "person-new",
  }));
  expect(await screen.findByText("Position change proposed.")).toBeTruthy();
});

it("shows pending position changes for independent approval", async () => {
  const base = await api.loadIdentityAccessOverview();
  api.loadIdentityAccessOverview.mockClear();
  api.loadIdentityAccessOverview.mockResolvedValue({
    ...base,
    actor_principal_id: "checker-1",
    organization_position_revisions: [{
      id: "position-revision-2",
      position_id: "position-owner",
      operation: "UPDATE",
      base_version: 4,
      base: {
        code: "PROGRAM_OWNER",
        title: "Program Owner",
        function_name: "Risk Operations",
        organization_scope_id: "scope-operations",
        parent_position_id: "position-cro",
        occupant_principal_id: "person-owner",
      },
      proposed: {
        code: "PROGRAM_OWNER",
        title: "Senior Program Owner",
        function_name: "Risk Operations",
        organization_scope_id: "scope-operations",
        parent_position_id: "position-cro",
        occupant_principal_id: "person-owner",
      },
      maker_id: "maker-1",
      status: "PENDING",
      impact: { child_positions: 0, responsibility_assignments: 1, authority_grants: 0, active_role_bindings: 1, active_programs_owned: 0, open_matters_owned: 0, open_actions_owned: 0 },
      created_at: "2026-10-03T12:00:00Z",
    }],
  });
  api.approveOrganizationPosition.mockResolvedValue(undefined);

  render(<IdentityAccessPanel/>);
  await screen.findByText("Pending position changes");

  fireEvent.click(screen.getByRole("button", { name: "Approve" }));
  const dialog = screen.getByRole("dialog", { name: "Approve position change" });
  fireEvent.change(within(dialog).getByRole("textbox", { name: /^Rationale/ }), { target: { value: "Reporting line checked" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Approve" }));

  await waitFor(() => expect(api.approveOrganizationPosition).toHaveBeenCalledWith("position-revision-2", "Reporting line checked"));
});

it("shows active work impact before approving an occupant change", async () => {
  const base = await api.loadIdentityAccessOverview();
  api.loadIdentityAccessOverview.mockClear();
  api.loadIdentityAccessOverview.mockResolvedValue({
    ...base,
    actor_principal_id: "checker-1",
    organization_position_revisions: [{
      id: "position-revision-impact",
      position_id: "position-owner",
      operation: "UPDATE",
      base_version: 4,
      base: {
        code: "PROGRAM_OWNER",
        title: "Program Owner",
        function_name: "Risk Operations",
        organization_scope_id: "scope-operations",
        parent_position_id: "position-cro",
        occupant_principal_id: "person-owner",
      },
      proposed: {
        code: "PROGRAM_OWNER",
        title: "Program Owner",
        function_name: "Risk Operations",
        organization_scope_id: "scope-operations",
        parent_position_id: "position-cro",
        occupant_principal_id: "person-new",
      },
      maker_id: "maker-1",
      status: "PENDING",
      impact: {
        child_positions: 0,
        responsibility_assignments: 1,
        authority_grants: 0,
        active_role_bindings: 1,
        active_programs_owned: 1,
        open_matters_owned: 2,
        open_actions_owned: 3,
      },
      created_at: "2026-10-03T12:00:00Z",
    }],
  });

  render(<IdentityAccessPanel/>);
  await screen.findByText("Pending position changes");
  fireEvent.click(screen.getByRole("button", { name: "Approve" }));

  const dialog = screen.getByRole("dialog", { name: "Approve position change" });
  expect(within(dialog).getByText(/6 active work items stay with the current owner or assignee/)).toBeTruthy();
  expect(within(dialog).getByText(/This position change does not reassign work/)).toBeTruthy();
});

it("proposes a governed restore from applied position history", async () => {
  const base = await api.loadIdentityAccessOverview();
  api.loadIdentityAccessOverview.mockClear();
  api.loadIdentityAccessOverview.mockResolvedValue({
    ...base,
    organization_position_history: [{
      id: "position-history-1",
      position_id: "position-owner",
      operation: "UPDATE",
      base_version: 3,
      base: {
        code: "PROGRAM_OWNER",
        title: "Program Owner",
        function_name: "Risk Operations",
        organization_scope_id: "scope-operations",
        parent_position_id: "position-cro",
        occupant_principal_id: "person-new",
      },
      proposed: {
        code: "PROGRAM_OWNER",
        title: "Program Owner",
        function_name: "Risk Operations",
        organization_scope_id: "scope-operations",
        parent_position_id: "position-cro",
        occupant_principal_id: "person-owner",
      },
      maker_id: "maker-1",
      checker_id: "checker-1",
      status: "APPLIED",
      rationale: "Prior approved state",
      impact: {
        child_positions: 0,
        responsibility_assignments: 1,
        authority_grants: 0,
        active_role_bindings: 1,
        active_programs_owned: 1,
        open_matters_owned: 1,
        open_actions_owned: 1,
      },
      created_at: "2026-10-01T12:00:00Z",
      decided_at: "2026-10-01T12:05:00Z",
      applied_at: "2026-10-01T12:05:00Z",
    }],
  });
  api.restoreOrganizationPosition.mockResolvedValue({
    id: "position-restore-1",
    position_id: "position-owner",
    operation: "UPDATE",
    base_version: 4,
    restored_from_revision_id: "position-history-1",
    base: {
      code: "PROGRAM_OWNER",
      title: "Program Owner",
      function_name: "Risk Operations",
      organization_scope_id: "scope-operations",
      parent_position_id: "position-cro",
      occupant_principal_id: "person-new",
    },
    proposed: {
      code: "PROGRAM_OWNER",
      title: "Program Owner",
      function_name: "Risk Operations",
      organization_scope_id: "scope-operations",
      parent_position_id: "position-cro",
      occupant_principal_id: "person-owner",
    },
    maker_id: "actor-1",
    status: "PENDING",
    impact: {
      child_positions: 0,
      responsibility_assignments: 1,
      authority_grants: 0,
      active_role_bindings: 1,
      active_programs_owned: 1,
      open_matters_owned: 1,
      open_actions_owned: 1,
    },
    created_at: "2026-10-03T12:00:00Z",
  });

  render(<IdentityAccessPanel/>);
  await screen.findByRole("heading", { name: "Organization & access" });
  fireEvent.click(screen.getByText("History"));
  fireEvent.click(screen.getByRole("button", { name: "Restore" }));

  await waitFor(() => expect(api.restoreOrganizationPosition).toHaveBeenCalledWith("position-history-1"));
  expect(await screen.findByText("Restore proposed.")).toBeTruthy();
});

it("proposes a new organization area from the Organization tab", async () => {
  api.proposeOrganizationScope.mockResolvedValue({
    id: "revision-1",
    scope_id: "scope-new",
    operation: "CREATE",
    base_version: 0,
    proposed_code: "LAGOS_ISLAND",
    proposed_name: "Lagos Island",
    proposed_kind: "BRANCH",
    maker_id: "actor-1",
    status: "PENDING",
    impact: { child_scopes: 0, positions: 0, access_mappings: 0, open_matters: 0 },
    created_at: "2026-10-03T12:00:00Z",
  });

  render(<IdentityAccessPanel/>);
  await screen.findByRole("heading", { name: "Organization & access" });

  fireEvent.click(screen.getByRole("button", { name: "Add area" }));
  const dialog = screen.getByRole("dialog", { name: "Add area" });
  fireEvent.change(within(dialog).getByRole("textbox", { name: /^Code/ }), { target: { value: "LAGOS_ISLAND" } });
  fireEvent.change(within(dialog).getByRole("textbox", { name: /^Name/ }), { target: { value: "Lagos Island" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Propose change" }));

  await waitFor(() => expect(api.proposeOrganizationScope).toHaveBeenCalledWith({
    operation: "CREATE",
    parent_scope_id: undefined,
    code: "LAGOS_ISLAND",
    name: "Lagos Island",
    kind: "DEPARTMENT",
  }));
  await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add area" })).toBeNull());
  expect(screen.getByText("Change proposed.")).toBeTruthy();
});

it("shows pending organization changes for independent approval", async () => {
  const base = await api.loadIdentityAccessOverview();
  api.loadIdentityAccessOverview.mockClear();
  api.loadIdentityAccessOverview.mockResolvedValue({
    ...base,
    actor_principal_id: "checker-1",
    organization_scope_revisions: [{
      id: "revision-2",
      scope_id: "scope-risk",
      operation: "UPDATE",
      base_version: 1,
      proposed_name: "Enterprise Risk",
      proposed_kind: "DEPARTMENT",
      maker_id: "maker-1",
      status: "PENDING",
      impact: { child_scopes: 1, positions: 1, access_mappings: 0, open_matters: 2 },
      created_at: "2026-10-03T12:00:00Z",
    }],
  });
  api.approveOrganizationScope.mockResolvedValue(undefined);

  render(<IdentityAccessPanel/>);
  await screen.findByText("Pending changes");

  fireEvent.click(screen.getByRole("button", { name: "Approve" }));
  const dialog = screen.getByRole("dialog", { name: "Approve change" });
  fireEvent.change(within(dialog).getByRole("textbox", { name: /^Rationale/ }), { target: { value: "Checked" } });
  fireEvent.click(within(dialog).getByRole("button", { name: "Approve" }));

  await waitFor(() => expect(api.approveOrganizationScope).toHaveBeenCalledWith("revision-2", "Checked"));
});
