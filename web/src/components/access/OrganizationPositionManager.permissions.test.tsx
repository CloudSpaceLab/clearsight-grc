import type { ComponentProps } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { OrganizationPosition, OrganizationPositionRevision, OrganizationPositionRoleRevision } from "../../identityAccessApi";
import { OrganizationPositionManager } from "./OrganizationPositionManager";

type ManagerProps = ComponentProps<typeof OrganizationPositionManager>;

const position: OrganizationPosition = {
  id: "position-risk",
  code: "RISK_MANAGER",
  title: "Risk Manager",
  department_path: [],
  role_codes: [],
  valid_from: "2026-01-01T00:00:00Z",
  version: 4,
};

const history: OrganizationPositionRevision = {
  id: "applied-revision",
  position_id: position.id,
  operation: "UPDATE",
  base_version: 2,
  base: { code: position.code, title: "Risk Officer" },
  proposed: { code: position.code, title: "Senior Risk Officer" },
  maker_id: "maker",
  checker_id: "checker",
  status: "APPLIED",
  impact: {
    child_positions: 0,
    responsibility_assignments: 0,
    authority_grants: 0,
    active_role_bindings: 0,
    active_programs_owned: 0,
    open_matters_owned: 0,
    open_actions_owned: 0,
  },
  created_at: "2026-10-01T09:00:00Z",
  applied_at: "2026-10-01T10:00:00Z",
};

const pendingRole: OrganizationPositionRoleRevision = {
  id: "pending-role-revision",
  position_id: position.id,
  role_template_id: "workspace-reader",
  operation: "ADD",
  base_position_version: position.version,
  base_role_version: 1,
  role_code: "WORKSPACE_READER",
  role_name: "Workspace reader",
  capabilities: ["CONFIG_READ"],
  maker_id: "maker",
  status: "PENDING",
  created_at: "2026-10-02T09:00:00Z",
};

function managerProps(overrides: Partial<ManagerProps> = {}) {
  return {
    positions: [position],
    allPositions: [position],
    scopes: [],
    people: [],
    revisions: [],
    history: [history],
    roleRevisions: [],
    roles: [],
    actorPrincipalID: "reader",
    canConfigure: true,
    scopeByID: new Map(),
    onPropose: vi.fn().mockResolvedValue(true),
    onApprove: vi.fn().mockResolvedValue(true),
    onReject: vi.fn().mockResolvedValue(true),
    onRestore: vi.fn().mockResolvedValue(true),
    ...overrides,
  } satisfies ManagerProps;
}

describe("Organization position restore permissions", () => {
  it("keeps applied history readable without exposing or invoking Restore for a reader", () => {
    const props = managerProps({ canConfigure: false });
    render(<OrganizationPositionManager {...props}/>);

    fireEvent.click(screen.getByText("History"));

    expect(screen.getByText("Applied")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Restore", hidden: true })).toBeNull();
    expect(props.onRestore).not.toHaveBeenCalled();
  });

  it("removes Restore when configuration permission is revoked while history is open", () => {
    const props = managerProps();
    const { rerender } = render(<OrganizationPositionManager {...props}/>);
    fireEvent.click(screen.getByText("History"));
    expect(screen.getByRole("button", { name: "Restore" })).toBeTruthy();

    rerender(<OrganizationPositionManager {...props} canConfigure={false}/>);

    expect(screen.getByText("Applied")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Restore", hidden: true })).toBeNull();
    expect(props.onRestore).not.toHaveBeenCalled();
  });

  const blockers: Array<{ name: string; overrides: Partial<ManagerProps> }> = [
    { name: "another change is busy", overrides: { isBusy: true } },
    { name: "the position has a pending change", overrides: { revisions: [{ ...history, id: "pending-position", status: "PENDING" }] } },
    { name: "the position has a scheduled change", overrides: { revisions: [{ ...history, id: "scheduled-position", status: "SCHEDULED", effective_from: "2026-12-01T09:00:00Z" }] } },
    { name: "the position has a pending role change", overrides: { roleRevisions: [pendingRole] } },
  ];

  it.each(blockers)("blocks Restore while $name and allows the exact history revision after it clears", ({ overrides }) => {
    const props = managerProps(overrides);
    const { rerender } = render(<OrganizationPositionManager {...props}/>);
    fireEvent.click(screen.getByText("History"));
    const restore = screen.getByRole("button", { name: "Restore" }) as HTMLButtonElement;

    expect(restore.disabled).toBe(true);
    fireEvent.click(restore);
    expect(props.onRestore).not.toHaveBeenCalled();

    rerender(<OrganizationPositionManager {...props} isBusy={false} revisions={[]} roleRevisions={[]}/>);
    const availableRestore = screen.getByRole("button", { name: "Restore" }) as HTMLButtonElement;
    expect(availableRestore.disabled).toBe(false);
    fireEvent.click(availableRestore);
    expect(props.onRestore).toHaveBeenCalledExactlyOnceWith(history);
  });
});
