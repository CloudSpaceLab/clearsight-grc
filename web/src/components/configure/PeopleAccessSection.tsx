import type { RuntimeContext } from "../../api";
import { IdentityAccessPanel, type IdentityAccessArea } from "../IdentityAccessPanel";
import { GovernanceAdminPanel } from "../GovernanceAdminPanel";
import { WorkspaceErrorBoundary } from "../WorkspaceErrorBoundary";
import { ActionLink, Button, Tabs, WorkspaceSwitcher } from "../ui";
import { AuthorityRoutingSection } from "./AuthorityRoutingSection";
import { peopleAccessHash, peopleAccessRoute, type PeopleAccessSelection, type PeopleAccessView } from "./peopleAccessRouting";
import "./people-access-workspace.css";

const views = [
  { id: "people" as const, label: "People" },
  { id: "roles" as const, label: "Roles" },
  { id: "policies" as const, label: "Approval routes" },
  { id: "delegations" as const, label: "Delegations" },
  { id: "organization" as const, label: "Organization settings", isContextual: true },
  { id: "checks" as const, label: "Routing checks", isContextual: true },
];
const organizationAreas: Array<{ id: IdentityAccessArea; label: string }> = [
  { id: "positions", label: "Organization" }, { id: "reporting", label: "Reporting lines" },
  { id: "directory", label: "Provisioning" }, { id: "escalation", label: "Escalation routes" },
];

type Props = { hash: string; onNavigate: (hash: string) => void; runtimeContext?: RuntimeContext | null };

export function PeopleAccessSection({ hash, onNavigate, runtimeContext }: Props) {
  const scopeKey = runtimeContext ? `${runtimeContext.tenant.id}:${runtimeContext.legal_entity.id}:${runtimeContext.actor.id}` : "pending";
  return <PeopleAccessWorkspace key={scopeKey} hash={hash} onNavigate={onNavigate} runtimeContext={runtimeContext} scopeKey={scopeKey}/>;
}

function PeopleAccessWorkspace({ hash, onNavigate, runtimeContext, scopeKey }: Props & { scopeKey: string }) {
  const route = peopleAccessRoute(hash);
  function open(view: PeopleAccessView, area?: IdentityAccessArea) {
    onNavigate(peopleAccessHash(view, route.selections, area));
  }
  function select(key: PeopleAccessSelection, value: string | undefined) {
    onNavigate(peopleAccessHash(route.view, { ...route.selections, [key]: value }, route.area));
  }
  const identityProps = {
    scopeKey,
    canRead: runtimeContext?.capabilities?.identity_read !== false,
    organizationName: runtimeContext?.tenant.name,
    legalEntityName: runtimeContext?.legal_entity.name,
  };

  return <section className="people-access-workspace" aria-labelledby="people-access-heading">
    <header className="people-access-workspace__heading">
      <svg className="people-access-workspace__wave" viewBox="0 0 750 160" preserveAspectRatio="none" aria-hidden="true">
        <path fill="currentColor" d="M0 160Q120 32 235 103T475 65T750 100V160Z"/>
        <path fill="currentColor" d="M0 160Q150 98 270 38T530 91T750 18V160Z"/>
      </svg>
      <div>
        <div className="people-access-workspace__crumb"><ActionLink href="#configure">Settings</ActionLink><span aria-hidden="true">›</span><span>People &amp; access</span></div>
        <h1 id="people-access-heading">People &amp; access</h1>
      </div>
      <div className="people-access-workspace__actions">
        <Button variant="secondary" onPress={() => open(route.view === "organization" ? "people" : "organization")}>{route.view === "organization" ? "Back to People" : "Organization settings"}</Button>
      </div>
    </header>
    <Tabs ariaLabel="People and access views" variant="surface" items={views} selectedKey={route.view} onSelectionChange={(view) => open(view)} retainVisitedPanels>
      {(view) => !runtimeContext ? <p role="status">Loading people and access…</p> : view === "people" ? <IdentityAccessPanel {...identityProps} active={route.view === "people"} area="people" selectedPersonID={route.selections.person} onSelectPerson={(id) => select("person", id)} onAreaChange={(area) => open("organization", area)}/>
        : view === "roles" ? <IdentityAccessPanel {...identityProps} active={route.view === "roles"} area="roles" selectedRoleID={route.selections.role} onSelectRole={(id) => select("role", id)} onAreaChange={(area) => open("organization", area)}/>
          : view === "policies" || view === "delegations" ? <>
            <WorkspaceErrorBoundary label="Approval routes and delegations"><GovernanceAdminPanel runtimeContext={runtimeContext} scopeKey={scopeKey} view={view} active={route.view === view} onOpenRoutingChecks={() => open("checks")}
              selectedPolicyID={route.selections.policy} onSelectPolicy={(id) => select("policy", id)}
              selectedDelegationID={route.selections.delegation} onSelectDelegation={(id) => select("delegation", id)}/></WorkspaceErrorBoundary>
          </> : view === "checks" ? <AuthorityRoutingSection checksOnly active={route.view === "checks"}/>
            : <section className="people-access-workspace__organization" aria-label="Organization settings">
              <WorkspaceSwitcher ariaLabel="Organization settings" compactLabel="Organization area" items={organizationAreas} selectedKey={route.area === "people" || route.area === "roles" ? "positions" : route.area} onSelectionChange={(area) => open("organization", area)}/>
              <IdentityAccessPanel key={route.area} {...identityProps} active={route.view === "organization"} area={route.area === "people" || route.area === "roles" ? "positions" : route.area}/>
            </section>}
    </Tabs>
  </section>;
}
