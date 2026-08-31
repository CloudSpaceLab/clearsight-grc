import { configurationAreas, type ConfigurationSection } from "./configurationSections";

type Props = {
  importsEnabled: boolean;
  onOpen: (section: ConfigurationSection) => void;
};

export function ConfigureOverview({ importsEnabled, onOpen }: Props) {
  return <section className="configure-overview" aria-labelledby="configuration-overview-heading">
    <header className="configure-domain-header">
      <div><span className="eyebrow">Administration</span><h2 id="configuration-overview-heading">Control plane</h2><p>Open one administrative area at a time. Operational approvals and assigned work remain in Today and Work.</p></div>
    </header>
    <div className="configure-area-list">
      {configurationAreas.map((area) => {
        const support = area.id === "data" && !importsEnabled ? "Document import is unavailable in this deployment." : supportText(area.id);
        return <button className="configure-area-row" type="button" key={area.id} onClick={() => onOpen(area.id)}>
          <span className="configure-area-copy"><strong>{area.label}</strong><small>{area.description}</small></span>
          <span className="configure-area-state"><span>{support}</span><b aria-hidden="true">›</b></span>
        </button>;
      })}
    </div>
  </section>;
}

function supportText(section: Exclude<ConfigurationSection, "overview">) {
  switch (section) {
    case "access": return "Directory and access administration";
    case "authority": return "Routing integrity and governed authority";
    case "data": return "Sources and ingestion";
    case "automation": return "Governed automation";
    case "ai": return "Governed AI workloads and policies";
    case "operations": return "Operational processing health";
  }
}
