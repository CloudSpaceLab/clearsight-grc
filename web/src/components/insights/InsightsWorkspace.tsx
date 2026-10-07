import type { InsightView } from "../../appRouting";
import type { ReportingPeriodQuery } from "../../reportingPeriod";
import { RiskLossInsights } from "./RiskLossInsights";
import { useEffect, useMemo, useState } from "react";
import type { ScopeNode } from "../../api";
import { loadIndicatorPopulation } from "../../indicatorApi";
import type { IndicatorPopulationItem, IndicatorPopulationPage } from "../../indicatorTypes";
import type { RiskIndicatorKind } from "../../riskTypes";
import { IndicatorDetail } from "../indicators/IndicatorDetail";
import { IndicatorMovement, indicatorMovementText } from "../indicators/IndicatorMovement";
import { IndicatorLimit, IndicatorObservedValue, indicatorLimitAccessibleText, indicatorObservedAccessibleText } from "../indicators/IndicatorValue";
import { formatIndicatorDate, formatIndicatorPeriod, indicatorStateLabel, indicatorTone } from "../indicators/indicatorPresentation";
import { Button, DataTable, EmptyState, FocusedSheet, Notice, StatusBadge, Surface, WorkspaceSwitcher, type DataColumn } from "../ui";
import "./insights.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  organizationScopeID?: string;
  organizationScopeName?: string;
  organizationScopes?: ScopeNode[];
  kind: RiskIndicatorKind;
  view?: InsightView;
  riskLossPeriod?: ReportingPeriodQuery;
  scopeAuthorized?: boolean;
  onOpenLoss?: (lossID: string) => void;
  onOpenOrganizationScope?: (scopeID: string) => void;
  targetID?: string;
  onKindChange: (kind: RiskIndicatorKind) => void;
  onTarget: (id: string | undefined, kind: RiskIndicatorKind) => void;
  onOpenProgram?: (programID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  onOpenRisk?: (riskID: string) => void;
  onOpenReports?: () => void;
};

type LoadState = "loading" | "live" | "error";

const indicatorKinds: ReadonlyArray<{ id: RiskIndicatorKind; label: string }> = [
  { id: "KRI", label: "Risk indicators" },
  { id: "KCI", label: "Control indicators" },
];

export function InsightsWorkspace({
  organizationName,
  legalEntityName,
  organizationScopeID,
  organizationScopeName,
  organizationScopes = [],
  kind,
  view,
  riskLossPeriod,
  scopeAuthorized = true,
  onOpenLoss,
  onOpenOrganizationScope,
  targetID,
  onKindChange,
  onTarget,
  onOpenProgram,
  onOpenMatter,
  onOpenRisk,
  onOpenReports,
}: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [page, setPage] = useState<IndicatorPopulationPage>({ items: [], complete: true });
  const [retry, setRetry] = useState(0);
  const [cursorStack, setCursorStack] = useState<string[]>([]);
  const cursor = cursorStack[cursorStack.length - 1];
  const [targetItem, setTargetItem] = useState<IndicatorPopulationItem>();
  const [targetState, setTargetState] = useState<LoadState>("live");

  useEffect(() => {
    setCursorStack([]);
  }, [kind, view, organizationScopeID]);

  useEffect(() => {
    if (view === "risk-loss") return;
    const controller = new AbortController();
    setState("loading");
    void loadIndicatorPopulation({ kind, organizationScopeID, cursor, limit: 50 }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPage(value);
      setState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setPage({ items: [], complete: false });
      setState("error");
    });
    return () => controller.abort();
  }, [cursor, kind, organizationScopeID, retry, view]);

  useEffect(() => {
    if (view === "risk-loss") return;
    if (!targetID) {
      setTargetItem(undefined);
      setTargetState("live");
      return;
    }
    const inPage = page.items.find((item) => item.indicator.check_id === targetID);
    if (inPage) {
      setTargetItem(inPage);
      setTargetState("live");
      return;
    }
    const controller = new AbortController();
    setTargetState("loading");
    void loadIndicatorPopulation({ kind, organizationScopeID, checkID: targetID, limit: 1 }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setTargetItem(value.items[0]);
      setTargetState("live");
    }).catch(() => {
      if (controller.signal.aborted) return;
      setTargetItem(undefined);
      setTargetState("error");
    });
    return () => controller.abort();
  }, [kind, organizationScopeID, page.items, targetID, view]);

  const scopeNames = useMemo(
    () => new Map(organizationScopes.map((scope) => [scope.id, scope.department_path?.join(" / ") || scope.name])),
    [organizationScopes],
  );
  const columns: readonly DataColumn<IndicatorPopulationItem>[] = [
    {
      id: "indicator",
      header: "Indicator",
      mobileLayout: "full-width",
      render: (item) => <span className="insights-indicator__identity"><strong>{item.indicator.check_name}</strong><small>{item.indicator.link.kind} · {item.indicator.check_code}</small></span>,
      accessibleText: (item) => `${item.indicator.check_name}. ${item.indicator.link.kind}. ${item.indicator.check_code}`,
    },
    {
      id: "scope",
      header: "Scope / owner",
      render: (item) => <span className="insights-indicator__scope"><strong>{indicatorScopeLabel(item, scopeNames, organizationScopeName, legalEntityName)}</strong><small>{item.indicator.owner_display_name || "Owner not assigned"}</small></span>,
      accessibleText: (item) => `${indicatorScopeLabel(item, scopeNames, organizationScopeName, legalEntityName)}. ${item.indicator.owner_display_name || "Owner not assigned"}`,
    },
    {
      id: "value",
      header: "Current value",
      kind: "number",
      render: (item) => <span className="insights-indicator__measure"><IndicatorObservedValue measurement={item.indicator.native_measurement} score={item.indicator.score} denominator={item.indicator.denominator}/><small>{formatIndicatorPeriod(item.indicator.native_measurement?.reporting_period_start, item.indicator.native_measurement?.reporting_period_end)}</small></span>,
      accessibleText: (item) => `${indicatorObservedAccessibleText(item.indicator.native_measurement, item.indicator.score, item.indicator.denominator)} ${formatIndicatorPeriod(item.indicator.native_measurement?.reporting_period_start, item.indicator.native_measurement?.reporting_period_end)}`,
    },
    {
      id: "limit",
      header: "Limit / target",
      kind: "number",
      render: (item) => <IndicatorLimit measurement={item.indicator.native_measurement}/>,
      accessibleText: (item) => indicatorLimitAccessibleText(item.indicator.native_measurement),
    },
    {
      id: "condition",
      header: "Condition",
      kind: "status",
      render: (item) => <StatusBadge tone={indicatorTone(item.indicator.state)}>{indicatorStateLabel(item.indicator.state)}</StatusBadge>,
      accessibleText: (item) => `${indicatorStateLabel(item.indicator.state)}. ${item.indicator.reason}`,
    },
    {
      id: "movement",
      header: "Movement",
      kind: "number",
      render: (item) => <IndicatorMovement movement={item.indicator.movement} measurement={item.indicator.native_measurement}/>,
      accessibleText: (item) => indicatorMovementText(item.indicator.movement, item.indicator.native_measurement),
    },
    {
      id: "updated",
      header: "Updated",
      render: (item) => item.indicator.evaluated_at ? formatIndicatorDate(item.indicator.evaluated_at) : "No current result",
      accessibleText: (item) => item.indicator.evaluated_at ? formatIndicatorDate(item.indicator.evaluated_at) : "No current result",
    },
  ];

  const scopeLabel = organizationScopeName || legalEntityName || "current scope";
  if (view === "risk-loss") {
    return <section className="insights-workspace">
      <header className="topbar">
        <div>
          <span className="eyebrow">{organizationName}</span>
          <h1>Insights</h1>
          <p>Governed Risk posture and selected-period Loss analysis for {scopeLabel}.</p>
        </div>
        {onOpenReports && <Button variant="secondary" onPress={onOpenReports}>Reports</Button>}
      </header>
      <WorkspaceSwitcher
        ariaLabel="Insight lenses"
        compactLabel="Insight lens"
        items={[...indicatorKinds, { id: "risk-loss", label: "Risk & loss" }]}
        selectedKey="risk-loss"
        onSelectionChange={(next) => { if (next === "KRI" || next === "KCI") onKindChange(next); }}
      />
      {riskLossPeriod
        ? <RiskLossInsights
          period={riskLossPeriod}
          organizationScopeID={organizationScopeID}
          scopeAuthorized={scopeAuthorized}
          onOpenRisk={onOpenRisk}
          onOpenLoss={onOpenLoss}
          onOpenScope={onOpenOrganizationScope}
        />
        : <Notice tone="warning">This saved Risk/Loss period or scope is invalid. Open Home to select a current authorized reporting period.</Notice>}
    </section>;
  }

  return <section className="insights-workspace">
    <header className="topbar">
      <div>
        <span className="eyebrow">{organizationName}</span>
        <h1>Insights</h1>
        <p>Current indicators for {scopeLabel}.</p>
      </div>
      {onOpenReports && <Button variant="secondary" onPress={onOpenReports}>Reports</Button>}
    </header>

    <WorkspaceSwitcher ariaLabel="Insight lenses" compactLabel="Insight lens" items={indicatorKinds} selectedKey={kind} onSelectionChange={onKindChange}/>

    <Surface>
      <div className="section-header insights-indicator__heading">
        <div>
          <h2>{kind === "KRI" ? "Risk indicators" : "Control indicators"}</h2>
          <p>Native values, approved limits and movement from the current governed indicator population.</p>
        </div>
      </div>

      {!page.complete && state !== "error" && <Notice tone="warning">Some indicator detail is unavailable in this scope. Available values remain unchanged.</Notice>}
      {page.truncated && <Notice tone="info">More indicators are available on the next page.</Notice>}
      {state === "error" && <EmptyState population="Indicators" title="Indicators could not be loaded" description="The current indicator population is unavailable." action={<Button variant="secondary" onPress={() => setRetry((value) => value + 1)}>Try again</Button>} role="alert"/>}
      {state === "live" && page.items.length === 0 && <EmptyState population={kind === "KRI" ? "Risk indicators" : "Control indicators"} title="No indicators in this scope" description="No current governed indicators are linked to Risks in the selected organization scope."/>}
      {(page.items.length > 0 || state === "loading") && <DataTable
        ariaLabel={kind === "KRI" ? "Risk indicators" : "Control indicators"}
        rows={page.items}
        rowKey={(item) => item.indicator.check_id}
        rowName={(item) => item.indicator.check_name}
        columns={columns}
        onRowAction={(item) => onTarget(item.indicator.check_id, kind)}
        rowActionLabel="Review indicator"
        isLoading={state === "loading"}
        pagination={(cursorStack.length > 0 || page.next_cursor) ? {
          label: "Indicator pages",
          onPrevious: cursorStack.length > 0 ? () => setCursorStack((current) => current.slice(0, -1)) : undefined,
          onNext: page.next_cursor ? () => setCursorStack((current) => [...current, page.next_cursor!]) : undefined,
          isLoading: state === "loading",
        } : undefined}
      />}
    </Surface>

    {targetID && targetState === "loading" && <span className="cs-sr-only" role="status">Loading selected indicator…</span>}
    {targetID && targetState === "error" && <Notice tone="error">The selected indicator could not be loaded. <Button variant="quiet" size="compact" onPress={() => onTarget(undefined, kind)}>Close selection</Button></Notice>}
    {targetID && targetState === "live" && !targetItem && <Notice tone="warning">The selected indicator is not in the current authorized scope. <Button variant="quiet" size="compact" onPress={() => onTarget(undefined, kind)}>Close selection</Button></Notice>}
    {targetItem && <FocusedSheet label={`${targetItem.indicator.check_name} indicator`} size="wide" onClose={() => onTarget(undefined, kind)}>
      <IndicatorDetail indicator={targetItem.indicator} onOpenProgram={onOpenProgram} onOpenMatter={onOpenMatter}/>
      <LinkedRisks item={targetItem} onOpenRisk={onOpenRisk}/>
    </FocusedSheet>}
  </section>;
}

function indicatorScopeLabel(item: IndicatorPopulationItem, scopeNames: Map<string, string>, selectedScope?: string, legalEntity?: string) {
  const ids = [...new Set(item.risks.map((risk) => risk.organization_scope_id).filter((id): id is string => Boolean(id)))];
  const labels = ids.map((id) => scopeNames.get(id) || id);
  if (item.risk_count > item.risks.length || labels.length > 2) return `${item.risk_count} linked Risks across multiple scopes`;
  if (labels.length === 1) return labels[0]!;
  if (labels.length === 2) return labels.join(" · ");
  return selectedScope || legalEntity || "Legal entity";
}

function LinkedRisks({ item, onOpenRisk }: { item: IndicatorPopulationItem; onOpenRisk?: (riskID: string) => void }) {
  return <section className="insights-indicator__risks" aria-labelledby="indicator-linked-risks">
    <div className="section-header">
      <div>
        <h3 id="indicator-linked-risks">Linked Risks</h3>
        <p>{item.risk_count === 1 ? "1 Risk uses this indicator." : `${item.risk_count} Risks use this indicator.`}</p>
      </div>
    </div>
    {item.kind_conflict && <Notice tone="warning">Indicator classification differs across linked Risks and needs review.</Notice>}
    <ul>{item.risks.map((risk) => <li key={risk.id}>
      <span><strong>{risk.code}</strong><small>{risk.name}</small></span>
      {onOpenRisk && <Button variant="secondary" size="compact" onPress={() => onOpenRisk(risk.id)}>Open Risk</Button>}
    </li>)}</ul>
    {item.risks_truncated && <p className="insights-indicator__risk-note">Showing {item.risks.length} of {item.risk_count} linked Risks.</p>}
  </section>;
}
