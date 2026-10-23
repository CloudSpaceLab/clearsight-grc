import { useEffect, useId, useRef, useState, type MouseEvent } from "react";
import { routeHash, type View, type WorkspaceTarget, type WorkTab } from "../appRouting";
import { riskLossDestinations } from "../riskLossNavigation";
import { NavigationIcon } from "./NavigationIcon";
import { Monogram } from "./Monogram";
import { ActionLink, FocusedSheet, IconButton } from "./ui";
import "../app-navigation.css";

type Destination = {
  id: string;
  label: string;
  view: View;
  target?: WorkspaceTarget;
  icon?: "frameworks";
};

// Adapt the v2 menu to backed application routes. Both rail and sheet use this list.
const destinations: readonly Destination[] = [
  { id: "home", label: "Home", view: "oversight" },
  { id: "frameworks", label: "Frameworks", view: "programs", target: { programsPage: "library" }, icon: "frameworks" },
  { id: "programs", label: "Programs", view: "programs" },
  { id: "forms", label: "Forms", view: "forms" },
  { id: "vendors", label: "Vendors", view: "vendors" },
  { id: "risk", label: "Risk & Losses", view: "risks" },
  { id: "privacy", label: "Privacy", view: "ropa" },
  { id: "work", label: "Work & Approvals", view: "work" },
  { id: "reports", label: "Reports", view: "reports" },
];

function selectedDestination(view: View, target: WorkspaceTarget) {
  if (view === "programs") return target.programsPage === "library" ? "frameworks" : "programs";
  if (view === "insights" && target.insightsView === "risk-loss") return "reports";
  if (view === "risks" || view === "rcsa" || view === "losses" || view === "insights") return "risk";
  if (view === "configure" || view === "imports") return "settings";
  if (view === "notifications") return "work";
  if (view === "today") return "home";
  return destinations.find((item) => item.view === view)?.id;
}

type AppNavigationProps = {
  activeView: View;
  target: WorkspaceTarget;
  configureEnabled: boolean;
  actorName: string;
  onNavigate: (view: View, target?: WorkspaceTarget, tab?: WorkTab) => void;
};

export function AppNavigation({ activeView, target, configureEnabled, actorName, onNavigate }: AppNavigationProps) {
  const [menuOpen, setMenuOpen] = useState(false);
  const selected = selectedDestination(activeView, target);
  const [riskExpanded, setRiskExpanded] = useState(selected === "risk");
  const sidebarRef = useRef<HTMLElement>(null);
  const riskID = useId();

  useEffect(() => {
    setRiskExpanded(selected === "risk");
    setMenuOpen(false);
  }, [activeView, selected, target]);

  useEffect(() => {
    const compact = window.matchMedia("(max-width: 1024px)");
    const onResize = () => {
      if (compact.matches) return;
      setMenuOpen(false);
      // The compact opener is hidden after expansion. Restore to a visible route.
      if (menuOpen) requestAnimationFrame(() => {
        const sidebar = sidebarRef.current;
        (sidebar?.querySelector<HTMLAnchorElement>('[aria-current="page"]')
          ?? sidebar?.querySelector<HTMLAnchorElement>("[data-navigation-item]"))?.focus();
      });
    };
    compact.addEventListener("change", onResize);
    return () => compact.removeEventListener("change", onResize);
  }, [menuOpen]);

  function activate(event: MouseEvent, item: Destination) {
    // Keep browser link semantics for new tabs/windows and modified clicks.
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.altKey || event.shiftKey) return;
    event.preventDefault();
    setMenuOpen(false);
    onNavigate(item.view, item.target ?? {}, item.view === "work" ? "assigned" : undefined);
  }

  function navigation(mode: "rail" | "sheet") {
    const childrenID = `${riskID}-${mode}`;
    return <nav className="app-navigation__workspaces" aria-label="Workspaces">
      <ul className="app-navigation__items">
        {destinations.map((item) => <li key={item.id} className={item.id === "risk" ? "app-navigation__risk" : undefined}>
          <div className="app-navigation__row">
            <ActionLink variant="navigation" href={routeHash(item.view, item.target ?? {}, "assigned")}
              aria-label={item.label} title={item.label} aria-current={selected === item.id ? "page" : undefined}
              data-navigation-item={item.id} onClick={(event) => activate(event, item)}>
              <NavigationIcon view={item.icon ?? item.view}/><span className="app-navigation__label">{item.label}</span>
            </ActionLink>
            {item.id === "risk" && <div className="app-navigation__disclosure">
              <IconButton variant="quiet" aria-label={riskExpanded ? "Hide risk sections" : "Show risk sections"}
                aria-expanded={riskExpanded} aria-controls={childrenID} onPress={() => setRiskExpanded((expanded) => !expanded)}>
                <svg aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8"><path d={riskExpanded ? "m6 15 6-6 6 6" : "m6 9 6 6 6-6"}/></svg>
              </IconButton>
            </div>}
          </div>
          {item.id === "risk" && <ul id={childrenID} className="app-navigation__children" hidden={!riskExpanded}>
            {riskLossDestinations.map((child) => <li key={child.id}>
              <ActionLink variant="navigation" href={routeHash(child.view, {}, "assigned")}
                aria-label={child.label} title={child.label} data-navigation-child={child.id}
                aria-current={selected === "risk" && activeView === child.view ? "location" : undefined}
                onClick={(event) => activate(event, child)}>
                <NavigationIcon view={child.view}/><span className="app-navigation__label">{child.label}</span>
              </ActionLink>
            </li>)}
          </ul>}
        </li>)}
      </ul>
      <div className="sidebar-secondary">
        {configureEnabled && <ActionLink variant="navigation" href="#configure" aria-label="Settings" title="Settings"
          aria-current={selected === "settings" ? "page" : undefined}
          onClick={(event) => activate(event, { id: "settings", label: "Settings", view: "configure" })}>
          <NavigationIcon view="configure"/><span className="app-navigation__label">Settings</span>
        </ActionLink>}
        <div className="app-navigation__actor" role="group" title={actorName} aria-label={`Signed in as ${actorName}`}>
          <Monogram name={actorName} className="avatar" decorative/>
          <span className="app-navigation__actor-name">{actorName}</span>
        </div>
      </div>
    </nav>;
  }

  return <>
    <aside ref={sidebarRef} className="sidebar app-navigation" aria-label="Primary navigation">
      <div className="sidebar-brand" role="img" aria-label="ClearSight">
        <span className="brand-mark" aria-hidden="true"/>
        <span className="sidebar-brand__name"><strong>ClearSight</strong><small>Governance, risk &amp; compliance</small></span>
      </div>
      <div className="app-navigation__opener">
        <IconButton variant="quiet" aria-label="Open navigation" aria-haspopup="dialog" aria-expanded={menuOpen}
          onPress={() => setMenuOpen(true)}>
          <svg aria-hidden="true" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round"><path d="M4 6h16M4 12h16M4 18h16"/></svg>
        </IconButton>
      </div>
      {navigation("rail")}
    </aside>
    {menuOpen && <FocusedSheet label="Navigation" closeLabel="Close navigation" placement="start"
      panelClassName="app-navigation-sheet" onClose={() => setMenuOpen(false)}>
      <div className="app-navigation-sheet__heading"><h2>Navigation</h2><p>ClearSight</p></div>
      {navigation("sheet")}
    </FocusedSheet>}
  </>;
}
