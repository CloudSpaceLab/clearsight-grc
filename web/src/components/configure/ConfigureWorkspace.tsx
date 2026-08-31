import { useEffect, useState } from "react";
import { AIGovernanceSection } from "./AIGovernanceSection";
import { AutomationSection } from "./AutomationSection";
import { AuthorityRoutingSection } from "./AuthorityRoutingSection";
import { ConfigureNavigation } from "./ConfigureNavigation";
import { ConfigureOverview } from "./ConfigureOverview";
import { DataIntegrationsSection } from "./DataIntegrationsSection";
import { PeopleAccessSection } from "./PeopleAccessSection";
import { SystemOperationsSection } from "./SystemOperationsSection";
import { configurationHash, configurationSectionFromHash, type ConfigurationSection } from "./configurationSections";

type Props = {
  importsEnabled: boolean;
  canReconcileProjection: boolean;
  onOpenImports: () => void;
};

export function ConfigureWorkspace({ importsEnabled, canReconcileProjection, onOpenImports }: Props) {
  const [section, setSection] = useState<ConfigurationSection>(() => configurationSectionFromHash(window.location.hash));

  useEffect(() => {
    const sync = () => setSection(configurationSectionFromHash(window.location.hash));
    window.addEventListener("hashchange", sync);
    window.addEventListener("popstate", sync);
    return () => {
      window.removeEventListener("hashchange", sync);
      window.removeEventListener("popstate", sync);
    };
  }, []);

  function select(next: ConfigurationSection) {
    setSection(next);
    const hash = configurationHash(next);
    if (window.location.hash !== hash) window.history.pushState(null, "", hash);
  }

  return <div className="configure-workspace" id="configure-workspace">
    <header className="topbar configure-topbar">
      <div><span className="eyebrow">Restricted administration</span><h1>Configuration</h1><p>Keep ClearSight connected, governed and operational without mixing administration into daily bank work.</p></div>
    </header>
    <div className="configure-shell">
      <ConfigureNavigation active={section} onSelect={select}/>
      <section className="configure-content" aria-label="Selected configuration area">
        {section === "overview" && <ConfigureOverview importsEnabled={importsEnabled} onOpen={select}/>} 
        {section === "access" && <PeopleAccessSection/>}
        {section === "authority" && <AuthorityRoutingSection/>}
        {section === "data" && <DataIntegrationsSection importsEnabled={importsEnabled} onOpenImports={onOpenImports}/>} 
        {section === "automation" && <AutomationSection/>}
        {section === "ai" && <AIGovernanceSection/>}
        {section === "operations" && <SystemOperationsSection canReconcile={canReconcileProjection}/>} 
      </section>
    </div>
  </div>;
}
