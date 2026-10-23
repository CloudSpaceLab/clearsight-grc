import { routeHash } from "../../appRouting";
import { riskLossSections, type RiskLossSection } from "../../riskLossNavigation";
import { WorkspaceSwitcher } from "../ui";
import "./risk-loss-navigation.css";

export function RiskLossNavigation({ selectedKey }: { selectedKey: RiskLossSection }) {
  return <div className="risk-loss-navigation">
    <WorkspaceSwitcher ariaLabel="Risk & Losses sections" compactLabel="Risk & Losses section"
      items={riskLossSections} selectedKey={selectedKey}
      onSelectionChange={(section) => { window.location.hash = routeHash(section, {}, "assigned"); }}/>
  </div>;
}
