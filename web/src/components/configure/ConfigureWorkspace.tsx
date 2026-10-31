import { useEffect, useState } from "react";
import type { RuntimeContext } from "../../api";
import "../../configure-workspace.css";
import { AIGovernanceSection } from "./AIGovernanceSection";
import { AutomationSection } from "./AutomationSection";
import { ConfigureNavigation } from "./ConfigureNavigation";
import { ConfigureOverview } from "./ConfigureOverview";
import { DataIntegrationsSection } from "./DataIntegrationsSection";
import { EnterpriseSetupSection } from "./EnterpriseSetupSection";
import { PeopleAccessSection } from "./PeopleAccessSection";
import { SystemOperationsSection } from "./SystemOperationsSection";
import { configurationHash, configurationSectionFromHash, type ConfigurationSection } from "./configurationSections";
import { enterpriseSetupRoute } from "./enterpriseSetupRouting";

type Props = {
  importsEnabled: boolean;
  canReconcileProjection: boolean;
  onOpenImports: () => void;
  runtimeContext?: RuntimeContext | null;
};

export function ConfigureWorkspace({ importsEnabled, canReconcileProjection, onOpenImports, runtimeContext }: Props) {
  const [hash, setHash] = useState(() => window.location.hash);
  const section = configurationSectionFromHash(hash);

  useEffect(() => {
    const sync = () => setHash(window.location.hash);
    window.addEventListener("hashchange", sync);
    window.addEventListener("popstate", sync);
    return () => {
      window.removeEventListener("hashchange", sync);
      window.removeEventListener("popstate", sync);
    };
  }, []);

  function select(next: ConfigurationSection) {
    navigate(configurationHash(next));
  }

  function navigate(next: string) {
    setHash(next);
    if (window.location.hash !== next) window.history.pushState(null, "", next);
  }

  if (enterpriseSetupRoute(hash)) return <EnterpriseSetupSection hash={hash} onNavigate={navigate} runtimeContext={runtimeContext} importsEnabled={importsEnabled} onOpenImports={onOpenImports}/>;
  if (section === "access" || section === "authority") return <PeopleAccessSection hash={hash} onNavigate={navigate} runtimeContext={runtimeContext}/>;

  return <div className="configure-workspace" id="configure-workspace">
    <header className="topbar configure-topbar">
      <div><span className="eyebrow">Restricted administration</span><h1>Configuration</h1><p>Manage access, approval routes, integrations and system settings.</p></div>
    </header>
    <div className="configure-shell">
      <ConfigureNavigation active={section} onSelect={select}/>
      <section className="configure-content" aria-label="Selected configuration area">
        {section === "overview" && <ConfigureOverview importsEnabled={importsEnabled} onOpen={select}/>} 
        {section === "data" && <DataIntegrationsSection importsEnabled={importsEnabled} onOpenImports={onOpenImports}/>} 
        {section === "automation" && <AutomationSection/>}
        {section === "ai" && <AIGovernanceSection/>}
        {section === "operations" && <SystemOperationsSection canReconcile={canReconcileProjection}/>} 
      </section>
    </div>
  </div>;
}
