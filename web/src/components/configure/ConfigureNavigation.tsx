import { configurationAreas, type ConfigurationSection } from "./configurationSections";

type Props = {
  active: ConfigurationSection;
  onSelect: (section: ConfigurationSection) => void;
};

export function ConfigureNavigation({ active, onSelect }: Props) {
  return <nav className="configure-navigation" aria-label="Configuration areas">
    <button type="button" className={active === "overview" ? "active" : ""} aria-current={active === "overview" ? "page" : undefined} onClick={() => onSelect("overview")}>
      <span>Overview</span><small>Administrative status and entry points</small>
    </button>
    {configurationAreas.map((area) => <button key={area.id} type="button" className={active === area.id ? "active" : ""} aria-current={active === area.id ? "page" : undefined} onClick={() => onSelect(area.id)}>
      <span>{area.label}</span><small>{area.description}</small>
    </button>)}
  </nav>;
}
