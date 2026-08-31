import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
}));

vi.mock("../identityAccessApi", () => api);

beforeEach(() => {
  vi.clearAllMocks();
  api.loadIdentityAccessOverview.mockResolvedValue({
    sign_in: { mode: "oidc", issuer: "https://id.bank.test", assurance_level: "MFA" },
    actor_principal_id: "actor-1",
    can_configure: true,
    can_configure_escalation: false,
    sources: [{ id: "source-1", code: "ENTRA", status: "ACTIVE", subject_attribute: "externalId", active_users: 12, active_groups: 2 }],
    people: [],
    groups: [{ id: "group-1", display_name: "Risk Operations", source_code: "ENTRA", source_state: "ACTIVE", member_count: 8 }],
    roles: [{ id: "role-1", code: "RISK_REVIEWER", name: "Risk reviewer", capabilities: ["program_read"] }],
    legal_entities: [],
    bindings: [],
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

  await screen.findByRole("heading", { name: "Enterprise access" });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.queryByRole("textbox", { name: "Code" })).toBeNull();
  expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1);

  fireEvent.click(screen.getByRole("button", { name: "Add source" }));
  expect(screen.getByRole("dialog", { name: "Add provisioning source" })).toBeTruthy();
  fireEvent.change(screen.getByRole("textbox", { name: "Code" }), { target: { value: "OKTA" } });
  fireEvent.click(screen.getByRole("button", { name: "Create source" }));

  await waitFor(() => expect(api.createIdentitySource).toHaveBeenCalledWith({ code: "OKTA", identity_issuer: undefined, subject_attribute: "externalId" }));
  await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add provisioning source" })).toBeNull());
  expect(screen.getByText("OKTA")).toBeTruthy();
  expect(screen.getByText("Provisioning token — shown once")).toBeTruthy();
  expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1);

  fireEvent.click(screen.getByRole("button", { name: "Add mapping" }));
  expect(screen.getByRole("dialog", { name: "Add group role mapping" })).toBeTruthy();
  fireEvent.change(screen.getByRole("combobox", { name: "Directory group" }), { target: { value: "group-1" } });
  fireEvent.change(screen.getByRole("combobox", { name: "Role" }), { target: { value: "role-1" } });
  fireEvent.change(screen.getByRole("textbox", { name: "Department path (optional)" }), { target: { value: "BANK / RISK" } });
  fireEvent.click(screen.getByRole("button", { name: "Add mapping" }));

  await waitFor(() => expect(api.createGroupRoleBinding).toHaveBeenCalledWith({ group_id: "group-1", role_template_id: "role-1", department_path: ["BANK", "RISK"] }));
  await waitFor(() => expect(screen.queryByRole("dialog", { name: "Add group role mapping" })).toBeNull());
  expect(screen.getByText("Risk Operations → RISK_REVIEWER")).toBeTruthy();
  expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1);
});
