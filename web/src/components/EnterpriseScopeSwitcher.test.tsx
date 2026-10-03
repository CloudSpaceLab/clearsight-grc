import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ScopeHierarchy } from "../api";
import { EnterpriseScopeSwitcher } from "./EnterpriseScopeSwitcher";

function hierarchy(count = 2): ScopeHierarchy {
  const legal_entities = Array.from({ length: count }, (_, index) => ({
    id: index === 0 ? "entity-ng" : `entity-${index}`,
    code: index === 0 ? "bank-ng" : `bank-${index}`,
    name: index === 0 ? "Clear Bank Nigeria" : index === 1 ? "Clear Bank Ghana" : `Clear Bank Entity ${index + 1}`,
    kind: "LEGAL_ENTITY" as const,
    parent_id: "tenant",
    jurisdiction: index === 0 ? "NG" : index === 1 ? "GH" : `R${index}`,
    current: index === 0 || undefined,
  }));
  return {
    state: "COMPLETE",
    root: { id: "tenant", code: "clear-bank", name: "Clear Bank", kind: "ORGANIZATION" },
    current: legal_entities[0]!,
    legal_entities,
  };
}

describe("EnterpriseScopeSwitcher", () => {
  it("shows the organization as a non-selectable parent and authorized legal entities as children", async () => {
    const onSelectionChange = vi.fn();
    render(<EnterpriseScopeSwitcher hierarchy={hierarchy()} currentScopeID="entity-ng" onSelectionChange={onSelectionChange}/>);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));

    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    expect(within(dialog).getByText("Clear Bank")).toBeTruthy();
    expect(within(dialog).queryByRole("button", { name: /^Clear Bank$/ })).toBeNull();
    expect(within(dialog).getByRole("button", { name: /Clear Bank Nigeria/ }).getAttribute("aria-pressed")).toBe("true");

    fireEvent.click(within(dialog).getByRole("button", { name: /Clear Bank Ghana/ }));
    expect(onSelectionChange).toHaveBeenCalledWith("entity-1");
    expect(screen.queryByRole("dialog", { name: "Change organization scope" })).toBeNull();
  });

  it("uses the organization root only as Group read mode when explicitly authorized", async () => {
    const onSelectionChange = vi.fn();
    const onGroupSelectionChange = vi.fn();
    render(<EnterpriseScopeSwitcher
      hierarchy={hierarchy()}
      currentScopeID="entity-ng"
      canSelectGroup
      onSelectionChange={onSelectionChange}
      onGroupSelectionChange={onGroupSelectionChange}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(within(dialog).getByRole("button", { name: /^Clear Bank$/ }));

    expect(onGroupSelectionChange).toHaveBeenCalledWith(true);
    expect(onSelectionChange).not.toHaveBeenCalled();
  });

  it("exits Group read mode through the current legal entity without a session switch", async () => {
    const onSelectionChange = vi.fn();
    const onGroupSelectionChange = vi.fn();
    render(<EnterpriseScopeSwitcher
      hierarchy={hierarchy()}
      currentScopeID="entity-ng"
      canSelectGroup
      isGroupSelected
      onSelectionChange={onSelectionChange}
      onGroupSelectionChange={onGroupSelectionChange}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank · Group" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    expect(within(dialog).getByRole("button", { name: /^Clear Bank$/ }).getAttribute("aria-pressed")).toBe("true");

    fireEvent.click(within(dialog).getByRole("button", { name: /Clear Bank Nigeria/ }));
    expect(onGroupSelectionChange).toHaveBeenCalledWith(false);
    expect(onSelectionChange).not.toHaveBeenCalled();
  });

  it("adds search only for larger authorized hierarchies and filters by name, code or jurisdiction", async () => {
    render(<EnterpriseScopeSwitcher hierarchy={hierarchy(8)} currentScopeID="entity-ng" onSelectionChange={() => undefined}/>);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    const search = within(dialog).getByPlaceholderText("Search scope");
    fireEvent.change(search, { target: { value: "GH" } });

    expect(within(dialog).getByRole("button", { name: /Clear Bank Ghana/ })).toBeTruthy();
    expect(within(dialog).queryByRole("button", { name: /Clear Bank Nigeria/ })).toBeNull();
  });

  it("selects only server-filterable organization scopes and keeps context-only ancestors inert", async () => {
    const onManageOrganization = vi.fn();
    const onOrganizationScopeChange = vi.fn();
    const scopedHierarchy = hierarchy();
    scopedHierarchy.organization_scopes = [
      { id: "scope-bank", code: "BANK", name: "BANK", kind: "BUSINESS_UNIT", department_path: ["BANK"] },
      { id: "scope-risk", code: "RISK", name: "RISK", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "RISK"], filterable: true },
      { id: "scope-ops", code: "OPS", name: "OPERATIONS", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "OPERATIONS"] },
    ];
    render(<EnterpriseScopeSwitcher
      hierarchy={scopedHierarchy}
      currentScopeID="entity-ng"
      onSelectionChange={() => undefined}
      onOrganizationScopeChange={onOrganizationScopeChange}
      onManageOrganization={onManageOrganization}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });

    expect(within(dialog).getByRole("button", { name: /RISK/ })).toBeTruthy();
    expect(within(dialog).getByText("OPERATIONS")).toBeTruthy();
    expect(within(dialog).queryByRole("button", { name: /OPERATIONS/ })).toBeNull();

    fireEvent.click(within(dialog).getByRole("button", { name: /RISK/ }));
    expect(onOrganizationScopeChange).toHaveBeenCalledWith(expect.objectContaining({ id: "scope-risk", filterable: true }));

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const managementDialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(within(managementDialog).getByRole("button", { name: "Organization & access" }));
    expect(onManageOrganization).toHaveBeenCalledTimes(1);
  });

  it("searches organization areas and keeps their parent path visible", async () => {
    const scopedHierarchy = hierarchy();
    scopedHierarchy.organization_scopes = [
      { id: "scope-bank", code: "BANK", name: "BANK", kind: "BUSINESS_UNIT", department_path: ["BANK"] },
      { id: "scope-risk", code: "RISK", name: "RISK", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "RISK"] },
      { id: "scope-payments", code: "PAYMENTS", name: "PAYMENTS", kind: "FUNCTION", parent_id: "scope-risk", department_path: ["BANK", "RISK", "PAYMENTS"], filterable: true },
      { id: "scope-ops", code: "OPS", name: "OPERATIONS", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "OPERATIONS"] },
      { id: "scope-a", code: "A", name: "AREA A", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "A"] },
      { id: "scope-b", code: "B", name: "AREA B", kind: "DEPARTMENT", parent_id: "scope-bank", department_path: ["BANK", "B"] },
    ];
    render(<EnterpriseScopeSwitcher hierarchy={scopedHierarchy} currentScopeID="entity-ng" onSelectionChange={() => undefined} onOrganizationScopeChange={() => undefined}/>);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    const search = within(dialog).getByPlaceholderText("Search scope");
    fireEvent.change(search, { target: { value: "payments" } });

    expect(within(dialog).getByText("BANK")).toBeTruthy();
    expect(within(dialog).getByText("RISK")).toBeTruthy();
    expect(within(dialog).getByRole("button", { name: /PAYMENTS/ })).toBeTruthy();
    expect(within(dialog).queryByText("OPERATIONS")).toBeNull();
  });

  it("searches beyond a truncated compact hierarchy and selects the authorized remote area", async () => {
    const onOrganizationScopeChange = vi.fn();
    const searchOrganizationAreas = vi.fn().mockResolvedValue({
      items: [{ id: "scope-lagos", code: "LAGOS", name: "Lagos Island", kind: "BRANCH", department_path: ["BANK", "BRANCHES", "LAGOS"], filterable: true }],
      has_more: false,
    });
    const scopedHierarchy = hierarchy();
    scopedHierarchy.organization_scopes = [
      { id: "scope-bank", code: "BANK", name: "BANK", kind: "BUSINESS_UNIT", department_path: ["BANK"] },
    ];
    scopedHierarchy.organization_scopes_truncated = true;
    render(<EnterpriseScopeSwitcher
      hierarchy={scopedHierarchy}
      currentScopeID="entity-ng"
      onSelectionChange={() => undefined}
      onOrganizationScopeChange={onOrganizationScopeChange}
      searchOrganizationAreas={searchOrganizationAreas}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    expect(within(dialog).getByText("More areas available. Search to find them.")).toBeTruthy();

    fireEvent.change(within(dialog).getByPlaceholderText("Search scope"), { target: { value: "Lagos Island" } });
    await waitFor(() => expect(searchOrganizationAreas).toHaveBeenCalledWith("Lagos Island", 30));
    fireEvent.click(await within(dialog).findByRole("button", { name: /Lagos Island/ }));

    expect(onOrganizationScopeChange).toHaveBeenCalledWith(expect.objectContaining({
      id: "scope-lagos",
      department_path: ["BANK", "BRANCHES", "LAGOS"],
      filterable: true,
    }));
  });

  it("clears an active subordinate Home scope when the current legal entity is selected", async () => {
    const onOrganizationScopeChange = vi.fn();
    const scopedHierarchy = hierarchy();
    scopedHierarchy.organization_scopes = [
      { id: "scope-risk", code: "RISK", name: "Risk", kind: "DEPARTMENT", parent_id: "entity-ng", department_path: ["BANK", "RISK"], filterable: true },
    ];
    render(<EnterpriseScopeSwitcher
      hierarchy={scopedHierarchy}
      currentScopeID="entity-ng"
      activeOrganizationScopeID="scope-risk"
      onSelectionChange={() => undefined}
      onOrganizationScopeChange={onOrganizationScopeChange}
    />);

    fireEvent.click(screen.getByRole("button", { name: /Organization scope, Clear Bank Nigeria · Risk/ }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    fireEvent.click(within(dialog).getByRole("button", { name: /Clear Bank Nigeria/ }));
    expect(onOrganizationScopeChange).toHaveBeenCalledWith(undefined);
  });

  it("does not expose a search field for a small hierarchy", async () => {
    render(<EnterpriseScopeSwitcher hierarchy={hierarchy()} currentScopeID="entity-ng" onSelectionChange={() => undefined}/>);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    await screen.findByRole("dialog", { name: "Change organization scope" });

    expect(screen.queryByPlaceholderText("Search scope")).toBeNull();
  });
});
