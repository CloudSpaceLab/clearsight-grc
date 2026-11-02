import { useState } from "react";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import type { RuntimeContext } from "../../api";
import { EnterpriseSetupSection } from "./EnterpriseSetupSection";

const identityProps = vi.hoisted(() => vi.fn());
vi.mock("../IdentityAccessPanel", () => ({ IdentityAccessPanel: (props: {
  scopeKey: string; canRead: boolean; selectedOrganizationScopeID?: string; onSelectOrganizationScope: (id: string | undefined) => void;
}) => {
  identityProps(props);
  return <section aria-label="Organization records"><span>{props.canRead ? "Allowed" : "Restricted"}</span><span>{props.selectedOrganizationScopeID || "No selected unit"}</span><button onClick={() => props.onSelectOrganizationScope("area-2")}>Select second area</button><button onClick={() => props.onSelectOrganizationScope(undefined)}>Clear selected area</button></section>;
} }));

const context: RuntimeContext = {
  tenant: { id: "tenant", name: "Example Bank Group" }, legal_entity: { id: "entity", name: "Example Bank Nigeria" },
  actor: { id: "actor", name: "Administrator" }, mode: "test", capabilities: { identity_read: true, config_read: true },
  scope_hierarchy: { state: "CURRENT_ONLY", root: { id: "tenant", name: "Example Bank Group", kind: "ORGANIZATION" }, current: { id: "entity", name: "Example Bank Nigeria", code: "NG", kind: "LEGAL_ENTITY" }, legal_entities: [] },
};
function Host({ initial = "#configure/access/positions", importsEnabled = true, onOpenImports = vi.fn(), runtime = context }: {
  initial?: string; importsEnabled?: boolean; onOpenImports?: () => void; runtime?: RuntimeContext | null;
}) {
  const [hash, setHash] = useState(initial);
  return <EnterpriseSetupSection hash={hash} onNavigate={(next) => { window.history.pushState(null, "", next); setHash(next); }} runtimeContext={runtime} importsEnabled={importsEnabled} onOpenImports={onOpenImports}/>;
}
beforeEach(() => { vi.clearAllMocks(); window.history.replaceState(null, "", "#configure/access/positions"); });

it("uses the complete source-order Setup menu and verified scope without a readiness score", () => {
  render(<Host/>);
  expect(screen.getAllByRole("heading", { level: 1 }).map((node) => node.textContent)).toEqual(["Enterprise setup"]);
  expect(within(screen.getByRole("tablist", { name: "Enterprise configuration" })).getAllByRole("tab").map((node) => node.textContent)).toEqual(["Organization", "Groups", "People & access", "Jurisdictions", "Frameworks", "Data imports"]);
  expect(screen.queryByRole("tablist", { name: "People and access views" })).toBeNull();
  expect(screen.queryByRole("progressbar")).toBeNull();
  expect(screen.getByText("Example Bank Nigeria")).toBeTruthy();
  expect(identityProps).toHaveBeenLastCalledWith(expect.objectContaining({ area: "positions", active: true, scopeKey: "tenant:entity:actor", legalEntityCode: "NG", importsEnabled: true }));
  expect(screen.queryByRole("navigation", { name: "Organization settings" })).toBeNull();
});

it("retains selected unit and People identities through Groups and Jurisdictions", () => {
  render(<Host initial="#configure/access/positions?unit=area-1&person=person-2"/>);
  fireEvent.click(screen.getByRole("tab", { name: "Groups" }));
  expect(screen.getByRole("heading", { name: "Organization groups are unavailable" })).toBeTruthy();
  expect(screen.getByRole("link", { name: "Open directory groups" })).toHaveProperty("hash", "#configure/access/directory?person=person-2");
  fireEvent.click(screen.getByRole("tab", { name: "Jurisdictions" }));
  expect(screen.getByRole("heading", { name: "Jurisdiction selection is unavailable" })).toBeTruthy();
  expect(screen.queryByRole("checkbox")).toBeNull();
  fireEvent.click(screen.getByRole("tab", { name: "Organization" }));
  expect(screen.getByText("area-1")).toBeTruthy();
  expect(new URLSearchParams(window.location.hash.split("?")[1]).get("person")).toBe("person-2");
  fireEvent.click(screen.getByRole("button", { name: "Select second area" }));
  expect(new URLSearchParams(window.location.hash.split("?")[1]).get("unit")).toBe("area-2");
  fireEvent.click(screen.getByRole("button", { name: "Clear selected area" }));
  expect(new URLSearchParams(window.location.hash.split("?")[1]).has("unit")).toBe(false);
});

it("hands People selection back to its existing workspace without copying Setup parameters", () => {
  const navigate = vi.fn();
  render(<EnterpriseSetupSection hash="#configure/access/positions?unit=area-1&person=person-2&setup=groups" runtimeContext={context} importsEnabled onOpenImports={vi.fn()} onNavigate={navigate}/>);
  fireEvent.click(screen.getByRole("tab", { name: "People & access" }));
  expect(navigate).toHaveBeenCalledWith("#configure/access?person=person-2");
});

it("uses a hash handoff for Frameworks so the App route observes the change", () => {
  render(<Host/>);
  fireEvent.click(screen.getByRole("tab", { name: "Frameworks" }));
  expect(window.location.hash).toBe("#programs/library");
});

it("lets keyboard users traverse all Setup destinations before explicitly opening one", () => {
  const navigate = vi.fn(), openImports = vi.fn();
  render(<EnterpriseSetupSection hash="#configure/access/positions" runtimeContext={context} importsEnabled onOpenImports={openImports} onNavigate={navigate}/>);
  screen.getByRole("tab", { name: "Organization" }).focus();
  for (const name of ["Groups", "People & access", "Jurisdictions", "Frameworks", "Data imports"]) {
    fireEvent.keyDown(document.activeElement as HTMLElement, { key: "ArrowRight" });
    expect(document.activeElement).toBe(screen.getByRole("tab", { name }));
    expect(screen.getByRole("tab", { name: "Organization", selected: true })).toBeTruthy();
    expect(navigate).not.toHaveBeenCalled();
    expect(openImports).not.toHaveBeenCalled();
    expect(window.location.hash).toBe("#configure/access/positions");
  }
  fireEvent.keyDown(document.activeElement as HTMLElement, { key: "Enter" });
  fireEvent.keyUp(document.activeElement as HTMLElement, { key: "Enter" });
  expect(openImports).toHaveBeenCalledOnce();
});

it("opens permitted imports through the owning App callback", () => {
  const openImports = vi.fn();
  render(<Host onOpenImports={openImports}/>);
  fireEvent.click(screen.getByRole("tab", { name: "Data imports" }));
  expect(openImports).toHaveBeenCalledOnce();
});

it("keeps denied Imports in a real unavailable panel and returns to the selected unit", () => {
  const openImports = vi.fn();
  render(<Host initial="#configure/access/positions?unit=area-1" importsEnabled={false} onOpenImports={openImports}/>);
  fireEvent.click(screen.getByRole("tab", { name: "Data imports" }));
  expect(screen.getByRole("heading", { name: "Data imports are unavailable" })).toBeTruthy();
  expect(screen.getByRole("tab", { name: "Data imports" }).getAttribute("aria-selected")).toBe("true");
  expect(openImports).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Return to Organization" }));
  expect(screen.getByText("area-1")).toBeTruthy();
});

it("waits for verified context and retires scope-local presentation on actor/entity changes", () => {
  const { rerender } = render(<EnterpriseSetupSection hash="#configure/access/positions?unit=area-1" runtimeContext={null} importsEnabled={false} onOpenImports={vi.fn()} onNavigate={vi.fn()}/>);
  expect(screen.getByRole("status").textContent).toContain("Loading organization settings");
  expect(identityProps).not.toHaveBeenCalled();
  rerender(<EnterpriseSetupSection hash="#configure/access/positions?unit=area-1" runtimeContext={context} importsEnabled={false} onOpenImports={vi.fn()} onNavigate={vi.fn()}/>);
  fireEvent.click(screen.getByRole("tab", { name: "Data imports" }));
  const changed = { ...context, legal_entity: { id: "other-entity", name: "Other entity" }, actor: { id: "other-actor", name: "Other actor" }, capabilities: { identity_read: false } };
  rerender(<EnterpriseSetupSection hash="#configure/access/positions?unit=area-1" runtimeContext={changed} importsEnabled={false} onOpenImports={vi.fn()} onNavigate={vi.fn()}/>);
  expect(screen.getByText("Restricted")).toBeTruthy();
  expect(identityProps).toHaveBeenLastCalledWith(expect.objectContaining({ scopeKey: "tenant:other-entity:other-actor", canRead: false, legalEntityCode: undefined }));
  expect(screen.queryByRole("heading", { name: "Data imports are unavailable" })).toBeNull();
});
