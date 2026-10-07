import { useEffect, useState } from "react";
import {
  loadDomainMetrics,
  loadLossPeriodMetrics,
  type DomainMetricBundle,
  type LossPeriodBundle,
} from "../../metricApi";
import type { ReportingPeriodQuery } from "../../reportingPeriod";
import { Notice } from "../ui";
import { DomainPostureSummary } from "../oversight/DomainPostureSummary";
import { OrganizationRiskSummary } from "../oversight/OrganizationRiskSummary";
import { OrganizationLossSummary } from "../oversight/OrganizationLossSummary";
import { RiskMovement } from "../oversight/RiskMovement";
import { LossMovement } from "../oversight/LossMovement";
import "../../oversight.css";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  period: ReportingPeriodQuery;
  organizationScopeID?: string;
  scopeAuthorized: boolean;
  onOpenRisk?: (id: string) => void;
  onOpenLoss?: (id: string) => void;
  onOpenScope?: (id: string) => void;
};

export function RiskLossInsights({ period, organizationScopeID, scopeAuthorized, onOpenRisk, onOpenLoss, onOpenScope }: Props) {
  const [domain, setDomain] = useState<DomainMetricBundle | null>(null);
  const [loss, setLoss] = useState<LossPeriodBundle | null>(null);
  const [domainState, setDomainState] = useState<LoadState>("loading");
  const [lossState, setLossState] = useState<LoadState>("loading");

  useEffect(() => {
    setDomain(null);
    setLoss(null);
    if (!scopeAuthorized) return;
    const controller = new AbortController();
    setDomainState("loading");
    setLossState("loading");
    void loadDomainMetrics(organizationScopeID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setDomain(value);
      setDomainState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setDomainState("unavailable");
    });
    void loadLossPeriodMetrics(period, organizationScopeID, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      // Never render Loss amounts or flows from a different requested scope or reporting interval.
      if (value.period_start.slice(0, 10) !== period.start_date ||
          value.period_end.slice(0, 10) !== period.end_date ||
          (value.scope_kind === "ORGANIZATION_SCOPE" ? value.scope_id !== organizationScopeID : Boolean(organizationScopeID))) {
        setLossState("unavailable");
        return;
      }
      setLoss(value);
      setLossState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setLossState("unavailable");
    });
    return () => controller.abort();
  }, [period.start_date, period.end_date, organizationScopeID, scopeAuthorized]);

  if (!scopeAuthorized) {
    return <Notice tone="warning">The saved organization scope is not active or authorized. Select the original department in the organization switcher before viewing its Insights.</Notice>;
  }

  return <section aria-label="Risk and loss analysis" className="insights-risk-loss">
    <p className="insights-risk-loss__basis">
      Loss period: <strong>{period.start_date} – {period.end_date}</strong>.
      Risk posture is current as of its governed source revision; it is not a period-flow total.
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
