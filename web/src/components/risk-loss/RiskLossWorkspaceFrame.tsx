import type { ReactNode } from "react";
import { riskLossDestinations, type RiskLossSection } from "../../riskLossNavigation";
import { Button, Tabs } from "../ui";
import "./risk-loss-workspace.css";

const sections = riskLossDestinations.map(({ view, label }) => ({ id: view, label }));

type Props = {
  section: RiskLossSection;
  enabled: boolean;
  organizationName?: string;
  legalEntityName?: string;
  organizationScopeName?: string;
  onSectionChange: (section: RiskLossSection) => void;
  onOpenReports: () => void;
  children: ReactNode;
};

export function RiskLossWorkspaceFrame({ section, enabled, organizationName, legalEntityName, organizationScopeName, onSectionChange, onOpenReports, children }: Props) {
  // Exact Risk/Loss records and the governed Reports analysis retain their own headings.
  if (!enabled) return <>{children}</>;

  const scope = [organizationName, legalEntityName, organizationScopeName].filter(Boolean);
  return <section className="risk-loss-workspace" aria-label="Risk and loss workspace">
    <header className="risk-loss-workspace__heading">
      <div>
        {scope.length > 0 && <p className="risk-loss-workspace__scope">{scope.join(" · ")}</p>}
        <h1>Risk &amp; Losses</h1>
      </div>
      <svg className="risk-loss-workspace__wave" viewBox="0 0 600 130" preserveAspectRatio="none" aria-hidden="true">
        <path d="M0 127C83 85 137 141 206 91S326 101 399 43 502 110 600 64V130H0Z" fill="currentColor"/>
        <path d="M10 130C127 111 166 51 246 75S350 112 419 69 560 71 600 46V130H10Z" fill="currentColor"/>
      </svg>
      <Button variant="secondary" onPress={onOpenReports}>View reports</Button>
    </header>
    <Tabs ariaLabel="Risk and loss sections" items={sections} selectedKey={section} onSelectionChange={onSectionChange}>
      {() => children}
    </Tabs>
  </section>;
}
