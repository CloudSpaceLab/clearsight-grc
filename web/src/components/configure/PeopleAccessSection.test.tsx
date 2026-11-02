import { useEffect, useState } from "react";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import type { RuntimeContext } from "../../api";
import { PeopleAccessSection } from "./PeopleAccessSection";

const loads = vi.hoisted(() => ({ identity: vi.fn(), governance: vi.fn() }));
vi.mock("../IdentityAccessPanel", () => ({ IdentityAccessPanel: ({ area, active, scopeKey, canRead, selectedPersonID, selectedRoleID, onSelectPerson, onAreaChange }: {
  area: string; active: boolean; scopeKey: string; canRead: boolean; selectedPersonID?: string; selectedRoleID?: string; onSelectPerson?: (id: string) => void; onAreaChange?: (area: string) => void;
}) => {
  const [search, setSearch] = useState("");
  useEffect(() => { loads.identity(area, scopeKey); }, []);
  return <section aria-label={`Identity ${area}`} data-active={active}><p>{canRead ? "Allowed" : "Restricted"}</p><label>Directory search<input value={search} onChange={(event) => setSearch(event.target.value)}/></label><span>{selectedPersonID || selectedRoleID || "No explicit selection"}</span><button onClick={() => onSelectPerson?.("person-2")}>Select second person</button><button onClick={() => onAreaChange?.("directory")}>View group mappings</button></section>;
} }));
vi.mock("../GovernanceAdminPanel", () => ({ GovernanceAdminPanel: ({ view, active, runtimeContext, onOpenRoutingChecks }: { view: string; active: boolean; runtimeContext: RuntimeContext; onOpenRoutingChecks: () => void }) => {
  useEffect(() => { loads.governance(view, runtimeContext.legal_entity.id); }, []);
  return <section aria-label={`Governance ${view}`} data-active={active}>Governance records{view === "policies" && <button onClick={onOpenRoutingChecks}>Routing checks</button>}</section>;
} }));
vi.mock("./AuthorityRoutingSection", () => ({ AuthorityRoutingSection: ({ checksOnly }: { checksOnly: boolean }) => <div>{checksOnly ? "Routing checks only" : "Combined routing"}</div> }));

const context: RuntimeContext = { tenant: { id: "tenant", name: "Example Bank Group" }, legal_entity: { id: "entity", name: "Example Bank Nigeria" }, actor: { id: "actor", name: "Administrator" }, mode: "test", capabilities: { identity_read: true, config_read: true } };
function Host({ runtime = context, initial = "#configure/access" }: { runtime?: RuntimeContext | null; initial?: string }) {
  const [hash, setHash] = useState(initial);
  return <PeopleAccessSection hash={hash} onNavigate={(next) => { window.history.pushState(null, "", next); setHash(next); }} runtimeContext={runtime}/>;
}

beforeEach(() => { vi.clearAllMocks(); window.history.replaceState(null, "", "#configure/access"); });

it("shows source tabs, lazily mounts readers, and retains directory state when changing views", () => {
  render(<Host/>);
  const menu = screen.getByRole("tablist", { name: "People and access views" });
  expect(within(menu).getAllByRole("tab").map((tab) => tab.textContent)).toEqual(["People", "Roles", "Approval routes", "Delegations"]);
  expect(loads.identity).toHaveBeenCalledTimes(1);
  expect(loads.governance).not.toHaveBeenCalled();
  const people = screen.getByRole("region", { name: "Identity people" });
  fireEvent.change(within(people).getByLabelText("Directory search"), { target: { value: "Amina" } });
  fireEvent.click(within(people).getByRole("button", { name: "Select second person" }));
  expect(window.location.hash).toBe("#configure/access?person=person-2");
  fireEvent.click(within(menu).getByRole("tab", { name: "Roles" }));
  expect(screen.getByRole("region", { name: "Identity roles" })).toBeTruthy();
  fireEvent.click(within(menu).getByRole("tab", { name: "People" }));
  expect(screen.getByRole("region", { name: "Identity people" })).toBe(people);
  expect(within(people).getByLabelText("Directory search")).toHaveProperty("value", "Amina");
  expect(within(people).getByText("person-2")).toBeTruthy();
  expect(loads.identity).toHaveBeenCalledTimes(2);
  expect(screen.getByRole("tablist", { name: "People and access views" })).toBe(menu);
});

it("keeps governance and organization operations reachable without duplicating the primary menu", () => {
  render(<Host initial="#configure/authority/delegations"/>);
  expect(screen.getByRole("region", { name: "Governance delegations" })).toBeTruthy();
  expect(loads.identity).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("tab", { name: "Approval routes" }));
  expect(screen.getByRole("region", { name: "Governance policies" })).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Routing checks" }));
  expect(screen.getByText("Routing checks only")).toBeTruthy();
  fireEvent.click(screen.getByRole("button", { name: "Organization settings" }));
  expect(window.location.hash).toBe("#configure/access/positions");
  expect(screen.queryByRole("navigation", { name: "Configuration areas" })).toBeNull();
  expect(screen.getByRole("link", { name: "Settings" })).toHaveProperty("hash", "#configure");
});

it("waits for verified context and resets retained state when the actor or legal entity changes", () => {
  const { rerender } = render(<Host runtime={null}/>);
  expect(screen.getByText("Loading people and access…")).toBeTruthy();
  expect(loads.identity).not.toHaveBeenCalled();
  rerender(<Host runtime={context}/>);
  const first = screen.getByRole("region", { name: "Identity people" });
  fireEvent.change(within(first).getByLabelText("Directory search"), { target: { value: "Old scope" } });
  rerender(<Host runtime={{ ...context, legal_entity: { id: "new-entity", name: "New entity" }, capabilities: { identity_read: false } }}/>);
  const next = screen.getByRole("region", { name: "Identity people" });
  expect(next).not.toBe(first);
  expect(within(next).getByLabelText("Directory search")).toHaveProperty("value", "");
  expect(within(next).getByText("Restricted")).toBeTruthy();
  expect(loads.identity).toHaveBeenLastCalledWith("people", "tenant:new-entity:actor");
});

it("hands selected People records to the canonical Enterprise Setup owner", () => {
  const navigate = vi.fn();
  render(<PeopleAccessSection hash="#configure/access?person=person-2&role=role-3" onNavigate={navigate} runtimeContext={context}/>);
  fireEvent.click(screen.getByRole("button", { name: "Organization settings" }));
  expect(navigate).toHaveBeenCalledWith("#configure/access/positions?person=person-2&role=role-3");
});

it("remounts the retained secondary owners when navigating between Provisioning and Reporting lines", () => {
  const navigate = vi.fn();
  const { rerender } = render(<PeopleAccessSection hash="#configure/access/directory" onNavigate={navigate} runtimeContext={context}/>);
  const provisioning = screen.getByRole("region", { name: "Identity directory" });
  fireEvent.change(within(provisioning).getByLabelText("Directory search"), { target: { value: "Previous form state" } });
  rerender(<PeopleAccessSection hash="#configure/access/reporting" onNavigate={navigate} runtimeContext={context}/>);
  expect(screen.getByRole("region", { name: "Identity reporting" })).not.toBe(provisioning);
  expect(provisioning.isConnected).toBe(false);
  rerender(<PeopleAccessSection hash="#configure/access/directory" onNavigate={navigate} runtimeContext={context}/>);
  const next = screen.getByRole("region", { name: "Identity directory" });
  expect(next).not.toBe(provisioning);
  expect(within(next).getByLabelText("Directory search")).toHaveProperty("value", "");
});
