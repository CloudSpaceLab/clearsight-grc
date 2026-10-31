import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type {
  LegalEntityDataBoundary,
  LegalEntityDataBoundaryRevision,
  OrganizationPosition,
  OrganizationPositionRevision,
  OrganizationScope,
} from "../../identityAccessApi";
import { OrganizationInventory } from "./OrganizationInventory";

function boundary(overrides: Partial<LegalEntityDataBoundary> = {}): LegalEntityDataBoundary {
  return {
    legal_entity_id: "entity-a",
    legal_entity_code: "ENTITY-A",
    legal_entity_name: "Entity A",
    jurisdiction: "NG",
    residency_region: "",
    detail_transfer_mode: "AGGREGATE_ONLY",
    allowed_destination_regions: [],
    version: 0,
    configured: false,
    ...overrides,
  };
}

function pending(overrides: Partial<LegalEntityDataBoundaryRevision> = {}): LegalEntityDataBoundaryRevision {
  return {
    id: "revision-1",
    legal_entity_id: "entity-a",
    base_version: 0,
    proposed_residency_region: "NG",
    proposed_detail_transfer_mode: "AGGREGATE_ONLY",
    proposed_destination_regions: [],
    maker_id: "maker-1",
    status: "PENDING",
    created_at: "2026-10-04T05:00:00Z",
    ...overrides,
  };
}

describe("OrganizationInventory data boundary", () => {
  it("keeps an unconfigured legal entity aggregate-only by default and proposes through maker-checker", async () => {
    const propose = vi.fn().mockResolvedValue(true);
    render(<OrganizationInventory
      positions={[]}
      people={[]}
      scopes={[]}
      mode="positions"
      dataBoundary={boundary()}
      actorPrincipalID="maker-1"
      canConfigure
      onProposeDataBoundary={propose}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Data boundary" }));
    expect(screen.getByText("Aggregate only", { selector: ".cs-status-badge" })).toBeTruthy();
    expect(screen.getByText("Group totals only · Cross-entity records restricted.")).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Residency region"), { target: { value: "NG" } });
    fireEvent.click(screen.getByRole("button", { name: "Propose change" }));

    await waitFor(() => expect(propose).toHaveBeenCalledWith({
      residency_region: "NG",
      detail_transfer_mode: "AGGREGATE_ONLY",
      allowed_destination_regions: [],
      expected_version: 0,
    }));
  });

  it("requires another administrator to approve a pending boundary", async () => {
    const approve = vi.fn().mockResolvedValue(true);
    const reject = vi.fn().mockResolvedValue(true);
    render(<OrganizationInventory
      positions={[]}
      people={[]}
      scopes={[]}
      mode="positions"
      dataBoundary={boundary({ configured: true, residency_region: "NG", version: 2 })}
      dataBoundaryRevisions={[pending({ base_version: 2, maker_id: "maker-other", proposed_residency_region: "NG-PRIMARY" })]}
      actorPrincipalID="checker-1"
      canConfigure
      onApproveDataBoundary={approve}
      onRejectDataBoundary={reject}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Data boundary" }));
    expect(screen.getByText("Proposed · NG-PRIMARY")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Propose change" })).toBeNull();

    fireEvent.change(screen.getByLabelText("Decision rationale"), { target: { value: "Residency reviewed" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => expect(approve).toHaveBeenCalledWith(
      expect.objectContaining({ id: "revision-1", maker_id: "maker-other" }),
      "Residency reviewed",
    ));
  });

  it("does not show approval controls to the maker of the pending boundary", () => {
    render(<OrganizationInventory
      positions={[]}
      people={[]}
      scopes={[]}
      mode="positions"
      dataBoundary={boundary()}
      dataBoundaryRevisions={[pending({ maker_id: "maker-1" })]}
      actorPrincipalID="maker-1"
      canConfigure
    />);

    fireEvent.click(screen.getByRole("button", { name: "Data boundary" }));
    expect(screen.getByText("Awaiting independent approval")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
  });
});


const scope: OrganizationScope = {
  id: "scope-risk",
  legal_entity_id: "entity-a",
  code: "RISK",
  name: "Risk",
  kind: "DEPARTMENT",
  department_path: ["BANK", "RISK"],
  origin: "MANAGED",
  status: "ACTIVE",
  valid_from: "2026-01-01T00:00:00Z",
  version: 1,
};

const position: OrganizationPosition = {
  id: "position-risk",
  code: "RISK_MANAGER",
  title: "Risk Manager",
  function_name: "Risk",
  department_path: ["BANK", "RISK"],
  organization_scope_id: scope.id,
  occupant_principal_id: "alice",
  occupant_name: "Alice",
  occupant_status: "ACTIVE",
  role_codes: [],
  valid_from: "2026-01-01T00:00:00Z",
  version: 4,
};

function positionRevision(overrides: Partial<OrganizationPositionRevision> = {}): OrganizationPositionRevision {
  return {
    id: "position-revision-1",
    position_id: position.id,
    operation: "UPDATE",
    base_version: 4,
    base: {
      code: position.code,
      title: position.title,
      function_name: "Risk",
      organization_scope_id: scope.id,
      occupant_principal_id: "alice",
    },
    proposed: {
      code: position.code,
      title: position.title,
      function_name: "Risk",
      organization_scope_id: scope.id,
      occupant_principal_id: "bob",
    },
    maker_id: "maker-other",
    status: "PENDING",
    rationale: "",
    impact: {
      child_positions: 0,
      responsibility_assignments: 1,
      authority_grants: 0,
      active_role_bindings: 0,
      active_programs_owned: 0,
      open_matters_owned: 1,
      open_actions_owned: 0,
    },
    created_at: "2026-10-04T05:00:00Z",
    ...overrides,
  };
}

describe("OrganizationInventory area filtering", () => {
  function input(scopes: OrganizationScope[], positions: OrganizationPosition[], selectedScopeID?: string) {
    return {
      scopes, positions, people: [], mode: "positions" as const, selectedScopeID, onSelectScope: vi.fn(),
      onProposePosition: vi.fn().mockResolvedValue(true),
      onApprovePosition: vi.fn().mockResolvedValue(true),
      onRejectPosition: vi.fn().mockResolvedValue(true),
    };
  }

  it("uses exact scope ancestry despite stale or duplicate paths and excludes missing or cyclic ancestry", () => {
    const child = { ...scope, id: "scope-child", code: "CHILD", name: "Child", parent_scope_id: scope.id, department_path: ["OLD", "CHILD"] };
    const descendant = { ...scope, id: "scope-descendant", code: "TEAM", name: "Team", parent_scope_id: child.id, department_path: ["OLD", "CHILD", "TEAM"] };
    const duplicate = { ...scope, id: "scope-duplicate", code: "DUPLICATE", name: "Unrelated root", department_path: ["bank", "risk"] };
    const wrongParent = { ...scope, id: "scope-wrong-parent", code: "WRONG", name: "Unrelated child", parent_scope_id: duplicate.id, department_path: ["BANK", "RISK", "WRONG"] };
    const orphan = { ...scope, id: "scope-orphan", code: "ORPHAN", name: "Missing parent", parent_scope_id: "scope-not-loaded", department_path: ["BANK", "RISK", "ORPHAN"] };
    const cycleA = { ...scope, id: "scope-cycle-a", code: "CYCLE_A", name: "Cycle A", parent_scope_id: "scope-cycle-b", department_path: ["BANK", "RISK", "CYCLE_A"] };
    const cycleB = { ...scope, id: "scope-cycle-b", code: "CYCLE_B", name: "Cycle B", parent_scope_id: cycleA.id, department_path: ["BANK", "RISK", "CYCLE_B"] };
    const scopes = [scope, child, descendant, duplicate, wrongParent, orphan, cycleA, cycleB];
    const positions = scopes.map((area) => ({ ...position, id: `position-${area.id}`, code: `POSITION_${area.code}`, title: `${area.name} position`, organization_scope_id: area.id }));
    render(<OrganizationInventory {...input(scopes, positions, scope.id)}/>);
    fireEvent.click(screen.getByRole("button", { name: "Positions & roles" }));
    const dialog = screen.getByRole("dialog", { name: "Positions & roles" });
    expect(within(dialog).getByText("3 of 8 loaded positions · Up to 50 positions are returned.")).toBeTruthy();
    for (const area of [scope, child, descendant]) expect(within(dialog).getByText(`${area.name} position`)).toBeTruthy();
    for (const area of [duplicate, wrongParent, orphan, cycleA, cycleB]) expect(within(dialog).queryByText(`${area.name} position`)).toBeNull();
  });

  it("shows all loaded positions without an area and keeps a missing selection empty until the filter is cleared", async () => {
    const positions = [
      position,
      { ...position, id: "position-missing", code: "MISSING", title: "Unreturned area position", organization_scope_id: "scope-not-loaded" },
      { ...position, id: "position-unassigned", code: "UNASSIGNED", title: "Entity position", organization_scope_id: undefined },
    ];
    const props = input([scope], positions);
    const { rerender } = render(<OrganizationInventory {...props}/>);
    fireEvent.click(screen.getByRole("button", { name: "Positions & roles" }));
    let dialog = screen.getByRole("dialog", { name: "Positions & roles" });
    for (const item of positions) expect(within(dialog).getByText(item.title)).toBeTruthy();

    rerender(<OrganizationInventory {...props} selectedScopeID="scope-not-loaded"/>);
    expect(screen.queryByRole("dialog", { name: "Positions & roles" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Positions & roles" }));
    dialog = screen.getByRole("dialog", { name: "Positions & roles" });
    expect(within(dialog).getByText("0 of 3 loaded positions · Up to 50 positions are returned.")).toBeTruthy();
    expect(within(dialog).getByText(/The selected area is not in the loaded list/)).toBeTruthy();
    for (const item of positions) expect(within(dialog).queryByText(item.title)).toBeNull();

    fireEvent.keyDown(within(dialog).getByRole("button", { name: /Department \/ area/ }), { key: "ArrowDown" });
    fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "All areas" }));
    for (const item of positions) expect(within(dialog).getByText(item.title)).toBeTruthy();
  });
});

describe("OrganizationInventory position governance", () => {
  const people = [
    { id: "alice", display_name: "Alice", status: "ACTIVE", user_name: "", source_code: "", source_state: "" },
    { id: "bob", display_name: "Bob", status: "ACTIVE", user_name: "", source_code: "", source_state: "" },
  ];

  it("sends an optional effective time with a position change proposal", async () => {
    const propose = vi.fn().mockResolvedValue(true);
    render(<OrganizationInventory
      positions={[position]}
      people={people}
      scopes={[scope]}
      mode="positions"
      actorPrincipalID="maker-1"
      canConfigure
      onProposePosition={propose}
      onApprovePosition={vi.fn().mockResolvedValue(true)}
      onRejectPosition={vi.fn().mockResolvedValue(true)}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Positions & roles" }));
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    fireEvent.change(screen.getByLabelText("Effective from"), { target: { value: "2026-10-10T09:30" } });
    fireEvent.click(screen.getByRole("button", { name: "Propose change" }));

    await waitFor(() => expect(propose).toHaveBeenCalledTimes(1));
    const input = propose.mock.calls[0]![0];
    expect(input.position_id).toBe(position.id);
    expect(new Date(input.effective_from).valueOf()).toBe(new Date("2026-10-10T09:30").valueOf());
  });

  it("shows current and proposed route candidates before approval without applying the change", async () => {
    const simulate = vi.fn().mockResolvedValue({
      revision_id: "position-revision-1",
      position_id: position.id,
      source_position_version: 4,
      effective_at: "2026-10-04T12:00:00Z",
      checked: 1,
      truncated: false,
      scenarios: [{
        object_type: "MATTER",
        object_id: "*",
        responsibility: "ACCOUNTABLE_OWNER",
        decision_type: "matter.assign",
        materiality: 0,
        current: { status: "RESOLVED", candidate_ids: ["alice"], policy_version: "POSITION-SIM:v1" },
        proposed: { status: "RESOLVED", candidate_ids: ["bob"], policy_version: "POSITION-SIM:v1" },
        changed: true,
      }],
    });
    render(<OrganizationInventory
      positions={[position]}
      people={people}
      scopes={[scope]}
      positionRevisions={[positionRevision()]}
      mode="positions"
      actorPrincipalID="checker-1"
      canConfigure
      onProposePosition={vi.fn().mockResolvedValue(true)}
      onApprovePosition={vi.fn().mockResolvedValue(true)}
      onRejectPosition={vi.fn().mockResolvedValue(true)}
      onSimulatePosition={simulate}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Positions & roles" }));
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));
    fireEvent.click(screen.getByRole("button", { name: "Check route impact" }));

    await waitFor(() => expect(simulate).toHaveBeenCalledWith(expect.objectContaining({ id: "position-revision-1" })));
    expect(screen.getByText("1 route checked")).toBeTruthy();
    expect(screen.getByText("1 changed")).toBeTruthy();
    expect(screen.getByText("Alice → Bob")).toBeTruthy();
  });

  it("keeps a scheduled change visible and removes duplicate edit or approval actions", () => {
    render(<OrganizationInventory
      positions={[position]}
      people={people}
      scopes={[scope]}
      positionRevisions={[positionRevision({
        status: "SCHEDULED",
        effective_from: "2026-10-10T09:30:00Z",
        checker_id: "checker-1",
        decided_at: "2026-10-04T06:00:00Z",
      })]}
      mode="positions"
      actorPrincipalID="another-admin"
      canConfigure
      onProposePosition={vi.fn().mockResolvedValue(true)}
      onApprovePosition={vi.fn().mockResolvedValue(true)}
      onRejectPosition={vi.fn().mockResolvedValue(true)}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Positions & roles" }));
    expect(screen.getByText("Scheduled", { selector: ".cs-status-badge" })).toBeTruthy();
    expect(screen.getByText("Current configuration remains active until this change applies.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
    expect(screen.getByRole("button", { name: "Edit" }).hasAttribute("disabled")).toBe(true);
  });
});
