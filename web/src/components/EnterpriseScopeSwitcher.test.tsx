import { fireEvent, render, screen, within } from "@testing-library/react";
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

  it("adds search only for larger authorized hierarchies and filters by name, code or jurisdiction", async () => {
    render(<EnterpriseScopeSwitcher hierarchy={hierarchy(8)} currentScopeID="entity-ng" onSelectionChange={() => undefined}/>);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });
    const search = within(dialog).getByPlaceholderText("Search legal entities");
    fireEvent.change(search, { target: { value: "GH" } });

    expect(within(dialog).getByRole("button", { name: /Clear Bank Ghana/ })).toBeTruthy();
    expect(within(dialog).queryByRole("button", { name: /Clear Bank Nigeria/ })).toBeNull();
  });

  it("shows department paths as organization context and keeps them non-selectable", async () => {
    const onManageOrganization = vi.fn();
    render(<EnterpriseScopeSwitcher
      hierarchy={hierarchy()}
      currentScopeID="entity-ng"
      organizationAreas={[["BANK", "RISK"], ["BANK", "OPERATIONS", "PAYMENTS"], ["bank", "risk"]]}
      onSelectionChange={() => undefined}
      onManageOrganization={onManageOrganization}
    />);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    const dialog = await screen.findByRole("dialog", { name: "Change organization scope" });

    expect(within(dialog).getByText("BANK / RISK")).toBeTruthy();
    expect(within(dialog).getByText("BANK / OPERATIONS / PAYMENTS")).toBeTruthy();
    expect(within(dialog).queryByRole("button", { name: /BANK \/ RISK/ })).toBeNull();
    expect(within(dialog).getByText(/Home area filtering is not available/)).toBeTruthy();

    fireEvent.click(within(dialog).getByRole("button", { name: "Organization & access" }));
    expect(onManageOrganization).toHaveBeenCalledTimes(1);
  });

  it("does not expose a search field for a small hierarchy", async () => {
    render(<EnterpriseScopeSwitcher hierarchy={hierarchy()} currentScopeID="entity-ng" onSelectionChange={() => undefined}/>);

    fireEvent.click(screen.getByRole("button", { name: "Organization scope, Clear Bank Nigeria" }));
    await screen.findByRole("dialog", { name: "Change organization scope" });

    expect(screen.queryByPlaceholderText("Search legal entities")).toBeNull();
  });
});
