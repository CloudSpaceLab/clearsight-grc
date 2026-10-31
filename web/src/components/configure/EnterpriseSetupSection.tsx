import { useEffect, useState } from "react";
import type { RuntimeContext } from "../../api";
import { IdentityAccessPanel } from "../IdentityAccessPanel";
import { ActionLink, Button, Tabs } from "../ui";
import { enterpriseSetupHash, enterpriseSetupRoute } from "./enterpriseSetupRouting";
import { peopleAccessHash } from "./peopleAccessRouting";
import "./enterprise-setup-workspace.css";

type SetupTab = "organization" | "groups" | "people" | "jurisdictions" | "frameworks" | "imports";
const setupTabs: ReadonlyArray<{ id: SetupTab; label: string }> = [
  { id: "organization", label: "Organization" },
  { id: "groups", label: "Groups" },
  { id: "people", label: "People & access" },
  { id: "jurisdictions", label: "Jurisdictions" },
  { id: "frameworks", label: "Frameworks" },
  { id: "imports", label: "Data imports" },
];

type Props = {
  hash: string;
  onNavigate: (hash: string) => void;
  runtimeContext?: RuntimeContext | null;
  importsEnabled: boolean;
  onOpenImports: () => void;
};

export function EnterpriseSetupSection(props: Props) {
  const context = props.runtimeContext;
  const scopeKey = context ? `${context.tenant.id}:${context.legal_entity.id}:${context.actor.id}` : "pending";
  return <EnterpriseSetupWorkspace key={scopeKey} {...props} scopeKey={scopeKey}/>;
}

function EnterpriseSetupWorkspace({ hash, onNavigate, runtimeContext, importsEnabled, onOpenImports, scopeKey }: Props & { scopeKey: string }) {
  const route = enterpriseSetupRoute(hash);
  const [unavailableImports, setUnavailableImports] = useState<string | null>(null);
  useEffect(() => setUnavailableImports(null), [hash, importsEnabled]);
  if (!route) return null;

  const peopleHash = peopleAccessHash("people", route.selections);
  const organizationHash = (view: "organization" | "groups" | "jurisdictions", unit = route.unit) => enterpriseSetupHash(view, unit, route.selections);
  const currentEntity = runtimeContext?.scope_hierarchy?.legal_entities.find((entity) => entity.id === runtimeContext.legal_entity.id)
    ?? (runtimeContext?.scope_hierarchy?.current.id === runtimeContext?.legal_entity.id ? runtimeContext?.scope_hierarchy?.current : undefined);

  function select(tab: SetupTab) {
    setUnavailableImports(null);
    if (tab === "people") onNavigate(peopleHash);
    else if (tab === "frameworks") window.location.hash = "#programs/library";
    else if (tab === "imports") {
      if (importsEnabled) onOpenImports();
      else setUnavailableImports(hash);
    } else onNavigate(organizationHash(tab));
  }

  return <section className="enterprise-setup-workspace" aria-labelledby="enterprise-setup-heading">
    <header className="enterprise-setup-workspace__heading">
      <svg className="enterprise-setup-workspace__wave" viewBox="0 0 750 160" preserveAspectRatio="none" aria-hidden="true">
        <path fill="currentColor" d="M0 160Q120 32 235 103T475 65T750 100V160Z"/>
        <path fill="currentColor" d="M0 160Q150 98 270 38T530 91T750 18V160Z"/>
      </svg>
      <div className="enterprise-setup-workspace__heading-copy">
        <div className="enterprise-setup-workspace__crumb"><ActionLink href="#configure">Settings</ActionLink><span aria-hidden="true">›</span><span>Enterprise</span></div>
        <h1 id="enterprise-setup-heading">Enterprise setup</h1>
        <p>Organization, access, frameworks and data</p>
      </div>
      <Button variant="secondary" onPress={() => select("people")}>People &amp; access</Button>
    </header>
    <section className="enterprise-setup-workspace__banner" aria-label="Organization change scope">
      <div><strong>{runtimeContext?.legal_entity.name || "Organization scope is loading"}</strong><p>Organization changes require independent approval.</p></div>
      {runtimeContext?.tenant.name && <span>{runtimeContext.tenant.name}</span>}
    </section>
    <Tabs ariaLabel="Enterprise configuration" variant="surface" keyboardActivation="manual" items={setupTabs} selectedKey={unavailableImports === hash && !importsEnabled ? "imports" : route.view} onSelectionChange={select}>
      {(view) => view === "organization" ? <>
        {!runtimeContext ? <p role="status">Loading organization settings…</p> : <IdentityAccessPanel
          area="positions" active scopeKey={scopeKey}
          canRead={runtimeContext.capabilities?.identity_read !== false}
          organizationName={runtimeContext.tenant.name}
          legalEntityName={runtimeContext.legal_entity.name}
          legalEntityCode={currentEntity?.code}
          importsEnabled={importsEnabled}
          selectedOrganizationScopeID={route.unit}
          onSelectOrganizationScope={(id: string | undefined) => onNavigate(enterpriseSetupHash("organization", id, route.selections))}
        />}
      </> : view === "groups" ? <section className="enterprise-setup-workspace__unavailable" aria-labelledby="setup-groups-heading">
        <h2 id="setup-groups-heading">Organization groups are unavailable</h2>
        <p>Groups of branches, departments and teams cannot be managed here. Directory groups and their access mappings remain available in Provisioning.</p>
        <ActionLink href={peopleAccessHash("organization", route.selections, "directory")}>Open directory groups</ActionLink>
      </section> : view === "jurisdictions" ? <section className="enterprise-setup-workspace__unavailable" aria-labelledby="setup-jurisdictions-heading">
        <h2 id="setup-jurisdictions-heading">Jurisdiction selection is unavailable</h2>
        <p>Operating jurisdictions cannot be selected here. Review each framework’s source and jurisdiction before proposing it for a Program.</p>
        <ActionLink href="#programs/library">Browse framework references</ActionLink>
      </section> : view === "imports" ? <section className="enterprise-setup-workspace__unavailable" aria-labelledby="setup-imports-heading">
        <h2 id="setup-imports-heading">Data imports are unavailable</h2>
        <p>Document import is unavailable in this deployment.</p>
        <Button variant="secondary" onPress={() => select("organization")}>Return to Organization</Button>
      </section> : null}
    </Tabs>
  </section>;
}
