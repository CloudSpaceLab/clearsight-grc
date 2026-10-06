import { IndicatorInsights } from "./IndicatorInsights";
import "./insights.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  onOpenProgram?: (programID: string) => void;
  onOpenMatter?: (matterID: string) => void;
};

export function InsightsWorkspace({ organizationName, legalEntityName, onOpenProgram, onOpenMatter }: Props) {
  return <div className="insights-workspace">
    <header className="topbar">
      <div>
        <span className="eyebrow">{organizationName || "Enterprise governance"}</span>
        <h1>Insights</h1>
        <p>Governed indicators and movement for {legalEntityName || "the current legal entity"}.</p>
      </div>
    </header>
    <IndicatorInsights
      legalEntityName={legalEntityName}
      onOpenProgram={onOpenProgram}
      onOpenMatter={onOpenMatter}
    />
  </div>;
}
