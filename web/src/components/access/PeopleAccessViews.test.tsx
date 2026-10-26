import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "../../http";
import type { IdentityAccessOverview } from "../../identityAccessApi";
import { IdentityAccessPanel } from "../IdentityAccessPanel";

const api = vi.hoisted(() => ({ loadIdentityAccessOverview: vi.fn(), createIdentitySource: vi.fn() }));
vi.mock("../../identityAccessApi", async (original) => ({ ...await original<typeof import("../../identityAccessApi")>(), ...api }));

function overview(): IdentityAccessOverview {
  return {
    sign_in: { mode: "oidc" }, actor_principal_id: "reader", can_configure: false, can_configure_organization: false, can_configure_escalation: false,
    sources: [{ id: "source", code: "Bank directory", status: "REVOKED", subject_attribute: "externalId", active_users: 0, active_groups: 0 }],
    people: Array.from({ length: 12 }, (_, index) => ({ id: `person-${index + 1}`, display_name: index === 0 ? "Amina Yusuf" : index === 11 ? "Zara Nwosu" : `Directory person ${index + 1}`, user_name: index === 0 ? "aminy" : `user-${index + 1}`, status: index === 1 ? "INACTIVE" : "ACTIVE", source_code: "Bank directory", source_state: "REVOKED" })),
    groups: [], bindings: [{ id: "binding", group_id: "group", group_name: "Risk officers", role_template_id: "role-risk", role_code: "RISK_REVIEWER", legal_entity_id: "bank-ng", legal_entity: "Bank Nigeria", department_path: ["Risk"], valid_from: "2026-01-01" }],
    roles: [{ id: "role-risk", code: "RISK_REVIEWER", name: "Risk reviewer", capabilities: ["PROGRAM_READ", "RISK_READ"], version: 2, organization_editable: false }, { id: "role-access", code: "IDENTITY_ADMIN", name: "Access administrator", capabilities: ["IDENTITY_READ"], version: 4, organization_editable: false }],
    positions: [{ id: "position-1", code: "RISK", title: "Senior risk reviewer", occupant_principal_id: "person-1", department_path: ["Risk"], role_codes: ["RISK_REVIEWER"], valid_from: "2026-01-01", version: 3 }],
    legal_entities: [], organization_scopes: [], organization_scope_revisions: [], organization_position_revisions: [], organization_position_history: [], organization_position_role_revisions: [], data_boundary: { legal_entity_id: "bank-ng", legal_entity_code: "NG", legal_entity_name: "Bank Nigeria", residency_region: "", detail_transfer_mode: "AGGREGATE_ONLY", allowed_destination_regions: [], version: 0, configured: false }, data_boundary_revisions: [],
    escalation: { pending_timers: 0, escalated_tasks: 0, unresolved_24h: 0, failed_timers: 0 }, escalation_policies: [],
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  api.loadIdentityAccessOverview.mockResolvedValue(overview());
});

describe("People and role views", () => {
  it("shows the entire returned directory and scopes local search and position facts honestly", async () => {
    render(<IdentityAccessPanel area="people" organizationName="North Bank Group" legalEntityName="Bank Nigeria"/>);
    const directory = await screen.findByRole("region", { name: "People directory" });
    expect(within(directory).getAllByRole("button", { name: /^Inspect / })).toHaveLength(12);
    expect(within(directory).getByRole("button", { name: "Inspect Zara Nwosu" })).toBeTruthy();
    expect(screen.getByText("Up to 50 profiles across North Bank Group.")).toBeTruthy();
    const detail = screen.getByRole("complementary", { name: "Amina Yusuf" });
    expect(within(detail).getByText("Username")).toBeTruthy();
    expect(within(detail).queryByText("Email")).toBeNull();
    expect(within(detail).getByText("Revoked")).toBeTruthy();
    expect(within(detail).getByText("Profile active")).toBeTruthy();
    expect(within(detail).getByText("Bank Nigeria · Up to 50 positions returned.")).toBeTruthy();
    fireEvent.change(screen.getByRole("searchbox", { name: "Search people" }), { target: { value: "zara" } });
    expect(within(directory).getAllByRole("button", { name: /^Inspect / })).toHaveLength(1);
    expect(screen.getByText("1 of 12 loaded profiles")).toBeTruthy();
    expect(screen.getByRole("complementary", { name: "Zara Nwosu" })).toBeTruthy();
    expect(screen.getByText("No position for this person was returned in this list.")).toBeTruthy();
  });

  it("filters profile status and provisioning source without inferring authority", async () => {
    const value = overview();
    value.people[1] = { id: "person-2", display_name: "Bola Adeyemi", status: "INACTIVE" };
    api.loadIdentityAccessOverview.mockResolvedValue(value);
    render(<IdentityAccessPanel area="people"/>);
    await screen.findByRole("region", { name: "People directory" });
    fireEvent.keyDown(screen.getByRole("button", { name: /Profile status/ }), { key: "ArrowDown" });
    fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "Inactive" }));
    expect(screen.getByText("1 of 12 loaded profiles")).toBeTruthy();
    const personWithoutSource = screen.getByRole("complementary", { name: "Bola Adeyemi" });
    expect(within(personWithoutSource).getByText("Source not returned")).toBeTruthy();
    expect(screen.queryByText("Local profile")).toBeNull();
    fireEvent.keyDown(screen.getByRole("button", { name: /Provisioning source/ }), { key: "ArrowDown" });
    fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "Source not returned" }));
    expect(screen.getByText("1 of 12 loaded profiles")).toBeTruthy();
    fireEvent.keyDown(screen.getByRole("button", { name: /Provisioning source/ }), { key: "ArrowDown" });
    fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "Bank directory" }));
    expect(screen.getByRole("heading", { name: "No people match these filters" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(screen.getByText("12 of 12 loaded profiles")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Invite person" })).toBeNull();
  });

  it("retains search through view changes and handles an explicit missing person without substituting identity", async () => {
    const onSelectPerson = vi.fn();
    const props = { scopeKey: "tenant:entity:actor", onSelectPerson };
    const { rerender } = render(<IdentityAccessPanel {...props} area="people"/>);
    await screen.findByRole("region", { name: "People directory" });
    fireEvent.change(screen.getByRole("searchbox", { name: "Search people" }), { target: { value: "Amina" } });
    rerender(<IdentityAccessPanel {...props} area="roles"/>);
    expect(screen.getByRole("region", { name: "Role catalog" })).toBeTruthy();
    rerender(<IdentityAccessPanel {...props} area="people" selectedPersonID="not-returned"/>);
    expect(screen.getByRole("searchbox", { name: "Search people" })).toHaveProperty("value", "Amina");
    expect(screen.getByRole("heading", { name: "Person not in this list" })).toBeTruthy();
    expect(screen.queryByRole("complementary", { name: "Amina Yusuf" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Return to listed people" }));
    expect(onSelectPerson).toHaveBeenCalledWith(undefined);
    expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1);
  });

  it("renders actual capabilities and mappings without fabricated role holder counts", async () => {
    const onSelectRole = vi.fn();
    const onAreaChange = vi.fn();
    const { rerender } = render(<IdentityAccessPanel area="roles" onSelectRole={onSelectRole} onAreaChange={onAreaChange}/>);
    await screen.findByRole("region", { name: "Role catalog" });
    const role = screen.getByRole("complementary", { name: "Risk reviewer" });
    expect(within(role).getByText("Program Read")).toBeTruthy();
    expect(within(role).getByText("Risk officers")).toBeTruthy();
    expect(screen.queryByText(/\d+ assigned/)).toBeNull();
    fireEvent.click(within(role).getByRole("button", { name: "View group mappings" }));
    expect(onAreaChange).toHaveBeenCalledWith("directory");
    fireEvent.click(screen.getByRole("button", { name: "Inspect role Access administrator" }));
    expect(onSelectRole).toHaveBeenCalledWith("role-access");
    rerender(<IdentityAccessPanel area="roles" selectedRoleID="missing" onSelectRole={onSelectRole}/>);
    expect(screen.getByRole("heading", { name: "Role not in this list" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Return to listed roles" }));
    expect(onSelectRole).toHaveBeenLastCalledWith(undefined);
  });

  it("restores implicit first selections when browser history removes controlled record queries", async () => {
    const onSelectPerson = vi.fn();
    const onSelectRole = vi.fn();
    const { rerender } = render(<IdentityAccessPanel area="people" onSelectPerson={onSelectPerson} onSelectRole={onSelectRole}/>);
    fireEvent.click(await screen.findByRole("button", { name: "Inspect Zara Nwosu" }));
    expect(onSelectPerson).toHaveBeenCalledWith("person-12");
    rerender(<IdentityAccessPanel area="people" selectedPersonID="person-12" onSelectPerson={onSelectPerson} onSelectRole={onSelectRole}/>);
    expect(screen.getByRole("complementary", { name: "Zara Nwosu" })).toBeTruthy();
    rerender(<IdentityAccessPanel area="people" onSelectPerson={onSelectPerson} onSelectRole={onSelectRole}/>);
    expect(screen.getByRole("complementary", { name: "Amina Yusuf" })).toBeTruthy();
    rerender(<IdentityAccessPanel area="roles" onSelectPerson={onSelectPerson} onSelectRole={onSelectRole}/>);
    fireEvent.click(screen.getByRole("button", { name: "Inspect role Access administrator" }));
    expect(onSelectRole).toHaveBeenCalledWith("role-access");
    rerender(<IdentityAccessPanel area="roles" selectedRoleID="role-access" onSelectPerson={onSelectPerson} onSelectRole={onSelectRole}/>);
    expect(screen.getByRole("complementary", { name: "Access administrator" })).toBeTruthy();
    rerender(<IdentityAccessPanel area="roles" onSelectPerson={onSelectPerson} onSelectRole={onSelectRole}/>);
    expect(screen.getByRole("complementary", { name: "Risk reviewer" })).toBeTruthy();
  });

  it("keeps 50 returned profiles visible to local search without asserting a complete directory", async () => {
    const value = overview();
    value.people = Array.from({ length: 50 }, (_, index) => ({ id: `person-${index}`, display_name: `Directory person ${index}`, status: "ACTIVE" }));
    api.loadIdentityAccessOverview.mockResolvedValue(value);
    render(<IdentityAccessPanel area="people"/>);
    const directory = await screen.findByRole("region", { name: "People directory" });
    expect(within(directory).getAllByRole("button", { name: /^Inspect / })).toHaveLength(50);
    fireEvent.change(screen.getByRole("searchbox", { name: "Search people" }), { target: { value: "person 49" } });
    expect(screen.getByRole("button", { name: "Inspect Directory person 49" })).toBeTruthy();
    expect(screen.getByText("1 of 50 loaded profiles")).toBeTruthy();
    expect(screen.queryByText(/all 50 people/i)).toBeNull();
  });
});

describe("access read recovery and isolation", () => {
  it("does not restore a one-time token or duplicate a source after leaving a pending creation", async () => {
    const value = overview();
    value.can_configure = true;
    api.loadIdentityAccessOverview.mockResolvedValue(value);
    const created = { id: "second-source", code: "SECOND", status: "ACTIVE", subject_attribute: "externalId", active_users: 0, active_groups: 0 };
    let resolveCreation!: (result: { source: typeof created; token: string }) => void;
    api.createIdentitySource.mockImplementationOnce(() => new Promise((resolve) => { resolveCreation = resolve; }));
    const { rerender } = render(<IdentityAccessPanel area="directory" active/>);
    fireEvent.click(await screen.findByRole("button", { name: "Add source" }));
    const dialog = screen.getByRole("dialog", { name: "Add provisioning source" });
    fireEvent.change(within(dialog).getByRole("textbox", { name: "Code" }), { target: { value: "SECOND" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Create source" }));
    await waitFor(() => expect(api.createIdentitySource).toHaveBeenCalledTimes(1));
    rerender(<IdentityAccessPanel area="directory" active={false}/>);
    expect(screen.queryByRole("dialog")).toBeNull();
    api.loadIdentityAccessOverview.mockResolvedValueOnce({ ...value, sources: [...value.sources, created] });
    rerender(<IdentityAccessPanel area="directory" active/>);
    await screen.findByText("SECOND");
    await act(async () => resolveCreation({ source: created, token: "late-one-time-token" }));
    expect(screen.queryByText("late-one-time-token")).toBeNull();
    expect(screen.getAllByText("SECOND")).toHaveLength(1);
  });

  it("rechecks a retained view on activation without remounting its filters", async () => {
    const { rerender } = render(<IdentityAccessPanel area="people" active/>);
    const search = await screen.findByRole("searchbox", { name: "Search people" });
    fireEvent.change(search, { target: { value: "Amina" } });
    let resolveRefresh!: (value: IdentityAccessOverview) => void;
    api.loadIdentityAccessOverview.mockImplementationOnce(() => new Promise<IdentityAccessOverview>((resolve) => { resolveRefresh = resolve; }));
    rerender(<IdentityAccessPanel area="people" active={false}/>);
    rerender(<IdentityAccessPanel area="people" active/>);
    await screen.findByText("Checking the latest people and access settings…");
    expect(screen.getByRole("searchbox", { name: "Search people" })).toBe(search);
    expect(search).toHaveProperty("value", "Amina");
    const next = overview();
    next.positions = [];
    await act(async () => resolveRefresh(next));
    expect(screen.getByRole("searchbox", { name: "Search people" })).toBe(search);
    expect(screen.getByText("No position for this person was returned in this list.")).toBeTruthy();
    expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(2);
  });

  it("blocks changes after failed revalidation and removes retained records on authorization failure", async () => {
    const value = overview();
    value.can_configure = true;
    api.loadIdentityAccessOverview.mockResolvedValue(value);
    const { rerender } = render(<IdentityAccessPanel area="directory" active/>);
    await screen.findByRole("button", { name: "Add source" });
    api.loadIdentityAccessOverview.mockRejectedValueOnce(new ApiError(503, "Unavailable"));
    rerender(<IdentityAccessPanel area="directory" active={false}/>);
    rerender(<IdentityAccessPanel area="directory" active/>);
    await screen.findByRole("button", { name: "Retry access settings" });
    expect(screen.queryByRole("button", { name: "Add source" })).toBeNull();
    expect(screen.getByText("Bank directory")).toBeTruthy();
    api.loadIdentityAccessOverview.mockRejectedValueOnce(new ApiError(403, "Denied"));
    fireEvent.click(screen.getByRole("button", { name: "Retry access settings" }));
    await screen.findByText("People & access restricted.");
    expect(screen.queryByText("Bank directory")).toBeNull();
  });

  it("does not request denied access and aborts old scope reads before accepting new data", async () => {
    const { rerender } = render(<IdentityAccessPanel area="people" scopeKey="first" canRead={false}/>);
    expect(api.loadIdentityAccessOverview).not.toHaveBeenCalled();
    expect(screen.getByText("People & access restricted.")).toBeTruthy();
    let resolveFirst!: (value: IdentityAccessOverview) => void;
    api.loadIdentityAccessOverview.mockImplementationOnce(() => new Promise<IdentityAccessOverview>((resolve) => { resolveFirst = resolve; }));
    rerender(<IdentityAccessPanel area="people" scopeKey="first"/>);
    await waitFor(() => expect(api.loadIdentityAccessOverview).toHaveBeenCalledTimes(1));
    const firstSignal = api.loadIdentityAccessOverview.mock.calls[0]?.[0] as AbortSignal;
    const next = overview();
    next.people = [{ id: "new", display_name: "New scope reader", status: "ACTIVE" }];
    api.loadIdentityAccessOverview.mockResolvedValueOnce(next);
    rerender(<IdentityAccessPanel area="people" scopeKey="second"/>);
    await screen.findByRole("button", { name: "Inspect New scope reader" });
    expect(firstSignal.aborted).toBe(true);
    await act(async () => resolveFirst(overview()));
    expect(screen.queryByRole("button", { name: "Inspect Amina Yusuf" })).toBeNull();
  });

  it("distinguishes forbidden, unavailable and successful empty reads with retry", async () => {
    api.loadIdentityAccessOverview.mockRejectedValueOnce(new ApiError(403, "Not allowed"));
    const { rerender } = render(<IdentityAccessPanel area="people" scopeKey="denied"/>);
    await screen.findByText("People & access restricted.");
    expect(screen.queryByText("No people returned")).toBeNull();
    api.loadIdentityAccessOverview.mockRejectedValueOnce(new ApiError(503, "Unavailable"));
    rerender(<IdentityAccessPanel area="people" scopeKey="error"/>);
    await screen.findByText("People and access settings could not be loaded.");
    const empty = overview();
    empty.people = []; empty.roles = []; empty.positions = [];
    api.loadIdentityAccessOverview.mockResolvedValueOnce(empty);
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByRole("heading", { name: "No people returned" })).toBeTruthy();
    expect(screen.getByText("0 of 0 loaded profiles")).toBeTruthy();
    expect(screen.queryByText("People & access restricted.")).toBeNull();
  });
});
