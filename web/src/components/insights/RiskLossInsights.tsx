import type { ReportingPeriodQuery } from "../../reportingPeriod";
import { useRiskLossMetrics } from "./useRiskLossMetrics";
import { Notice } from "../ui";
import { DomainPostureSummary } from "../oversight/DomainPostureSummary";
import { OrganizationRiskSummary } from "../oversight/OrganizationRiskSummary";
import { OrganizationLossSummary } from "../oversight/OrganizationLossSummary";
import { RiskMovement } from "../oversight/RiskMovement";
import { LossMovement } from "../oversight/LossMovement";
import "../../oversight.css";

type Props = {
  period: ReportingPeriodQuery;
  legalEntityID?: string;
  organizationScopeID?: string;
  scopeAuthorized: boolean;
  onOpenRisk?: (id: string) => void;
  onOpenLoss?: (id: string) => void;
  onOpenScope?: (id: string) => void;
};

export function RiskLossInsights({ period, legalEntityID, organizationScopeID, scopeAuthorized, onOpenRisk, onOpenLoss, onOpenScope }: Props) {
  const { domain, loss, domainState, lossState } = useRiskLossMetrics({
    metricPeriod: period, metricState: "live", scopeAuthorized, legalEntityID,
    organizationScopeID, riskScope: "legal-entity", requireScopeID: false,
  });

  if (!scopeAuthorized) {
    return <Notice tone="warning">The saved organization scope is not active or authorized. Select the original department in the organization switcher before viewing its Insights.</Notice>;
  }

  return <section aria-label="Risk and loss analysis" className="insights-risk-loss">
    <p className="insights-risk-loss__basis">
      Loss period: <strong>{period.start_date} – {period.end_date}</strong>.
      Risk posture is current as of its governed source revision; it is not a period-flow total.
      {organizationScopeID && <> Risk posture is for the legal entity. Loss flow is filtered to the selected organization area.</>}
    </p>
    <DomainPostureSummary
      bundle={domain}
      state={domainState}
      lossBundle={loss}
      lossState={lossState}
      organizationScopeID={organizationScopeID}
      onOpenRisk={onOpenRisk}
      onOpenLoss={onOpenLoss}
    />
    <div className="oversight-risk-overview">
      <OrganizationRiskSummary bundle={domain} organizationScopeID={organizationScopeID} onOpenScope={onOpenScope}/>
      <RiskMovement bundle={domain} organizationScopeID={organizationScopeID}/>
    </div>
    <div className="oversight-loss-overview">
      <OrganizationLossSummary bundle={loss} state={lossState} onOpenScope={onOpenScope}/>
      <LossMovement bundle={loss} state={lossState}/>
    </div>
  </section>;
}
