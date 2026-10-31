import type { ComponentProps } from "react";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { OrganizationScope, OrganizationScopeRevision } from "../../identityAccessApi";
import { OrganizationScopeManager } from "./OrganizationScopeManager";

type ManagerProps = ComponentProps<typeof OrganizationScopeManager>;
const risk: OrganizationScope = {
  id: "scope-risk", legal_entity_id: "entity-a", code: "RISK", name: "Risk", kind: "DEPARTMENT",
  department_path: ["BANK", "RISK"], origin: "MANAGED", status: "ACTIVE", valid_from: "2026-01-01T00:00:00Z", version: 7,
};
const operations: OrganizationScope = { ...risk, id: "scope-operations", code: "OPS", name: "Operations", parent_scope_id: risk.id, department_path: ["BANK", "RISK", "OPS"], version: 3 };
const branch: OrganizationScope = { ...risk, id: "scope-branch", code: "LAGOS", name: "Lagos Branch", kind: "BRANCH", department_path: ["BANK", "LAGOS"], version: 2 };
const imported: OrganizationScope = { ...risk, id: "scope-imported", code: "IMPORTED", name: "Imported operations", kind: "ORGANIZATION_UNIT", origin: "LEGACY_DEPARTMENT_PATH", department_path: ["BANK", "IMPORTED"], version: 11 };

function revision(area: OrganizationScope, maker = "other-maker"): OrganizationScopeRevision {
  return {
    id: `revision-${area.id}`, scope_id: area.id, operation: "UPDATE", base_version: area.version,
    proposed_code: area.code, proposed_name: `Updated ${area.name}`, proposed_kind: "FUNCTION",
    maker_id: maker, status: "PENDING", created_at: "2026-10-01T09:00:00Z",
    impact: { child_scopes: 1, positions: 2, access_mappings: 3, open_matters: 4 },
  };
}

function props(overrides: Partial<ManagerProps> = {}) {
  return {
    scopes: [risk, operations, branch, imported], revisions: [], actorPrincipalID: "checker", canConfigure: true,
    legalEntityName: "Example Bank", legalEntityCode: "EXAMPLE",
    onPropose: vi.fn().mockResolvedValue(true), onApprove: vi.fn().mockResolvedValue(true), onReject: vi.fn().mockResolvedValue(true),
    ...overrides,
  } satisfies ManagerProps;
}

function inspect(name: string) { fireEvent.click(screen.getByRole("button", { name: `Inspect ${name}` })); }
function openChanges() {
  fireEvent.click(screen.getByRole("button", { name: "Review area changes" }));
  return screen.getByRole("dialog", { name: "Area changes" });
}

describe("OrganizationScopeManager contracts", () => {
  it("keeps an unreturned controlled selection explicit and never selects another area after refresh", () => {
    const select = vi.fn();
    const input = props({ selectedScopeID: "not-returned", onSelectScope: select });
    const { rerender } = render(<OrganizationScopeManager {...input}/>);
    let context = screen.getByRole("complementary", { name: "Area not in this list" });
    expect(within(context).getByText(/selected area was not returned for this legal entity/)).toBeTruthy();
    expect(within(context).queryByRole("button", { name: "Edit area" })).toBeNull();
    expect(screen.getByRole("button", { name: "Inspect Example Bank" }).getAttribute("aria-pressed")).toBe("false");
    fireEvent.click(within(context).getByRole("button", { name: "View legal entity" }));
    expect(select).toHaveBeenCalledExactlyOnceWith(undefined);

    rerender(<OrganizationScopeManager {...input} selectedScopeID={risk.id}/>);
    expect(screen.getByRole("complementary", { name: "Risk" })).toBeTruthy();
    rerender(<OrganizationScopeManager {...input} selectedScopeID={risk.id} scopes={[branch]}/>);
    context = screen.getByRole("complementary", { name: "Area not in this list" });
    expect(within(context).queryByRole("button", { name: "Edit area" })).toBeNull();
    expect(screen.getByRole("button", { name: "Inspect Lagos Branch" }).getAttribute("aria-pressed")).toBe("false");
    expect(input.onPropose).not.toHaveBeenCalled();
  });

  it("requires an explicit managed Type for an imported area and keeps Move and Retire blocked after proposing", async () => {
    const input = props();
    render(<OrganizationScopeManager {...input}/>);
    inspect(imported.name);
    for (const name of ["Move area", "Retire area"]) expect(screen.getByRole("button", { name })).toHaveProperty("disabled", true);
    fireEvent.click(screen.getByRole("button", { name: "Edit area" }));
    const dialog = screen.getByRole("dialog", { name: "Edit Imported operations" });
    const submit = within(dialog).getByRole("button", { name: "Propose change" });
    expect(submit).toHaveProperty("disabled", true);
    expect(within(dialog).getByRole("button", { name: /Type/ }).textContent).toContain("Choose type");
    fireEvent.click(submit);
    expect(input.onPropose).not.toHaveBeenCalled();
    fireEvent.keyDown(within(dialog).getByRole("button", { name: /Type/ }), { key: "ArrowDown" });
    fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: "Function" }));
    fireEvent.click(submit);
    await waitFor(() => expect(input.onPropose).toHaveBeenCalledExactlyOnceWith({ operation: "UPDATE", scope_id: imported.id, name: imported.name, kind: "FUNCTION", expected_version: 11 }));
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Edit Imported operations" })).toBeNull());
    for (const name of ["Move area", "Retire area"]) expect(screen.getByRole("button", { name })).toHaveProperty("disabled", true);
    expect(within(screen.getByRole("complementary", { name: imported.name })).getByText("Organization Unit")).toBeTruthy();
  });

  it("excludes the current area and its descendants from Move and sends the exact chosen parent and version", async () => {
    const input = props();
    render(<OrganizationScopeManager {...input}/>);
    inspect(risk.name);
    fireEvent.click(screen.getByRole("button", { name: "Move area" }));
    const dialog = screen.getByRole("dialog", { name: "Move Risk" });
    fireEvent.keyDown(within(dialog).getByRole("button", { name: /New parent/ }), { key: "ArrowDown" });
    const list = await screen.findByRole("listbox");
    expect(within(list).queryByRole("option", { name: /BANK \/ RISK/ })).toBeNull();
    fireEvent.click(within(list).getByRole("option", { name: "BANK / LAGOSBranch" }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Propose change" }));
    await waitFor(() => expect(input.onPropose).toHaveBeenCalledExactlyOnceWith({ operation: "MOVE", scope_id: risk.id, parent_scope_id: branch.id, expected_version: 7 }));
  });

  it.each([
    { destination: "a loaded parent", option: "BANK / LAGOSBranch", parentID: branch.id },
    { destination: "the legal entity", option: "Legal entity", parentID: undefined },
  ])("requires explicit selection of $destination when the current Move parent is unavailable", async ({ option, parentID }) => {
    const orphan = { ...risk, parent_scope_id: "scope-not-loaded" };
    const input = props({ scopes: [orphan, operations, branch] });
    render(<OrganizationScopeManager {...input}/>);
    inspect(risk.name);
    fireEvent.click(screen.getByRole("button", { name: "Move area" }));
    const dialog = screen.getByRole("dialog", { name: "Move Risk" });
    const parent = within(dialog).getByRole("button", { name: /New parent/ });
    const submit = within(dialog).getByRole("button", { name: "Propose change" });
    expect(parent.textContent).toContain("Current parent unavailable");
    expect(within(dialog).getByText(/Choose a loaded area or the legal entity before proposing a move/)).toBeTruthy();
    expect(submit).toHaveProperty("disabled", true);
    fireEvent.click(submit);
    expect(input.onPropose).not.toHaveBeenCalled();

    fireEvent.keyDown(parent, { key: "ArrowDown" });
    fireEvent.click(within(await screen.findByRole("listbox")).getByRole("option", { name: option }));
    expect(parent.textContent).not.toContain("Current parent unavailable");
    expect(submit).toHaveProperty("disabled", false);
    fireEvent.click(submit);
    await waitFor(() => expect(input.onPropose).toHaveBeenCalledExactlyOnceWith({ operation: "MOVE", scope_id: risk.id, parent_scope_id: parentID, expected_version: 7 }));
  });

  it("proposes retirement against the selected area's exact version", async () => {
    const input = props();
    render(<OrganizationScopeManager {...input}/>);
    inspect(branch.name);
    fireEvent.click(screen.getByRole("button", { name: "Retire area" }));
    const dialog = screen.getByRole("dialog", { name: "Retire Lagos Branch" });
    expect(within(dialog).getByText(/Blocked if the area has child areas, positions, access mappings or open matters/)).toBeTruthy();
    fireEvent.click(within(dialog).getByRole("button", { name: "Propose retirement" }));
    await waitFor(() => expect(input.onPropose).toHaveBeenCalledExactlyOnceWith({ operation: "RETIRE", scope_id: branch.id, expected_version: 2 }));
  });

  it("keeps area facts and proposals readable without exposing any mutation to a reader", () => {
    const input = props({ canConfigure: false, revisions: [revision(risk)] });
    render(<OrganizationScopeManager {...input}/>);
    inspect(risk.name);
    expect(screen.getByRole("complementary", { name: "Risk" })).toBeTruthy();
    for (const name of ["Add area", "Edit area", "Move area", "Retire area"]) expect(screen.queryByRole("button", { name })).toBeNull();
    const changes = openChanges();
    expect(within(changes).getByText("Edit Risk")).toBeTruthy();
    expect(within(changes).queryByRole("button", { name: "Approve" })).toBeNull();
    expect(within(changes).queryByRole("button", { name: "Reject" })).toBeNull();
    for (const action of [input.onPropose, input.onApprove, input.onReject]) expect(action).not.toHaveBeenCalled();
  });

  it.each(["Approve", "Reject"] as const)("attributes maker controls and submits %s with the reviewed revision and trimmed rationale", async (action) => {
    const own = revision(risk, "checker"), other = revision(branch);
    const input = props({ revisions: [own, other] });
    render(<OrganizationScopeManager {...input}/>);
    const changes = openChanges();
    const ownRow = within(changes).getByText("Edit Risk").closest("li")!;
    const otherRow = within(changes).getByText("Edit Lagos Branch").closest("li")!;
    expect(within(ownRow).getByText("Awaiting approval")).toBeTruthy();
    for (const name of ["Approve", "Reject"]) {
      expect(within(ownRow).queryByRole("button", { name })).toBeNull();
      expect(within(otherRow).getByRole("button", { name })).toHaveProperty("disabled", false);
    }
    fireEvent.click(within(otherRow).getByRole("button", { name: action }));
    const dialog = screen.getByRole("dialog", { name: `${action} change` });
    expect(within(dialog).getByText("Updated Lagos Branch")).toBeTruthy();
    expect(within(dialog).getByText("Function")).toBeTruthy();
    expect(within(dialog).getByText("Base version").nextElementSibling?.textContent).toBe("2");
    expect(within(dialog).getByText("Proposed by").nextElementSibling?.textContent).toBe("other-maker");
    expect(within(dialog).getByText("1 child areas · 2 positions · 3 access mappings · 4 open matters")).toBeTruthy();
    const submit = within(dialog).getByRole("button", { name: action });
    expect(submit).toHaveProperty("disabled", true);
    fireEvent.change(within(dialog).getByRole("textbox", { name: /Rationale/ }), { target: { value: "   " } });
    expect(submit).toHaveProperty("disabled", true);
    fireEvent.change(within(dialog).getByRole("textbox", { name: /Rationale/ }), { target: { value: "  Reviewed the area change  " } });
    fireEvent.click(submit);
    await waitFor(() => expect(action === "Approve" ? input.onApprove : input.onReject).toHaveBeenCalledExactlyOnceWith(other, "Reviewed the area change"));
    expect(action === "Approve" ? input.onReject : input.onApprove).not.toHaveBeenCalled();
  });

  it("closes an unsaved editor when the controlled selection changes and does not restore its draft", () => {
    const input = props({ selectedScopeID: risk.id, onSelectScope: vi.fn() });
    const { rerender } = render(<OrganizationScopeManager {...input}/>);
    fireEvent.click(screen.getByRole("button", { name: "Edit area" }));
    fireEvent.change(within(screen.getByRole("dialog", { name: "Edit Risk" })).getByRole("textbox", { name: /^Name/ }), { target: { value: "Unsaved title" } });
    rerender(<OrganizationScopeManager {...input} selectedScopeID={branch.id}/>);
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByRole("complementary", { name: branch.name })).toBeTruthy();
    rerender(<OrganizationScopeManager {...input}/>);
    fireEvent.click(screen.getByRole("button", { name: "Edit area" }));
    expect(within(screen.getByRole("dialog", { name: "Edit Risk" })).getByRole("textbox", { name: /^Name/ })).toHaveProperty("value", "Risk");
    expect(input.onPropose).not.toHaveBeenCalled();
  });

  it.each(["editor", "decision"] as const)("retires an open %s when configuration permission is revoked", (kind) => {
    const input = props({ selectedScopeID: risk.id, onSelectScope: vi.fn(), revisions: kind === "decision" ? [revision(risk)] : [] });
    const { rerender } = render(<OrganizationScopeManager {...input}/>);
    if (kind === "editor") fireEvent.click(screen.getByRole("button", { name: "Edit area" }));
    else fireEvent.click(within(openChanges()).getByRole("button", { name: "Approve" }));
    const label = kind === "editor" ? "Edit Risk" : "Approve change";
    expect(screen.getByRole("dialog", { name: label })).toBeTruthy();
    rerender(<OrganizationScopeManager {...input} canConfigure={false}/>);
    expect(screen.queryByRole("dialog", { name: label })).toBeNull();
    rerender(<OrganizationScopeManager {...input}/>);
    expect(screen.queryByRole("dialog", { name: label })).toBeNull();
    for (const action of [input.onPropose, input.onApprove, input.onReject]) expect(action).not.toHaveBeenCalled();
  });
});
