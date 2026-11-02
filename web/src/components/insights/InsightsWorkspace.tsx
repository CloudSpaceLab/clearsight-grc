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
import { Button, DataTable, EmptyState, FocusedSheet, Notice, SelectField, StatusBadge, WorkspaceSwitcher, type DataColumn } from "../ui";
import "./insights.css";

type Props = {
  organizationName?: string;
  legalEntityName?: string;
  embedded?: boolean;
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
type PopulationRead = { key: string; state: LoadState; page: IndicatorPopulationPage };
type SelectedRead = { key: string; state: LoadState; item?: IndicatorPopulationItem };

const emptyPopulation: IndicatorPopulationPage = { items: [], complete: true };

const indicatorKinds: ReadonlyArray<{ id: RiskIndicatorKind; label: string }> = [
  { id: "KRI", label: "Risk indicators" },
  { id: "KCI", label: "Control indicators" },
];

export function InsightsWorkspace({
  organizationName,
  legalEntityName,
  embedded = false,
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
  const contextKey = JSON.stringify([kind, organizationScopeID, view]);
  const [pagination, setPagination] = useState({ key: contextKey, cursors: [] as string[] });
  const cursorStack = pagination.key === contextKey ? pagination.cursors : [];
  const cursor = cursorStack[cursorStack.length - 1];
  const populationKey = JSON.stringify([contextKey, cursor]);
  const selectedKey = JSON.stringify([contextKey, targetID]);
  const [population, setPopulation] = useState<PopulationRead>();
  const state = population?.key === populationKey ? population.state : "loading";
  const page = population?.key === populationKey ? population.page : emptyPopulation;
  const [retry, setRetry] = useState(0);
  const [selected, setSelected] = useState<SelectedRead>();
  const targetState = selected?.key === selectedKey ? selected.state : "loading";
  const targetItem = targetID && selected?.key === selectedKey && selected.item?.indicator.check_id === targetID ? selected.item : undefined;

  const setCursorStack = (update: (current: string[]) => string[]) => {
    setPagination((current) => ({ key: contextKey, cursors: update(current.key === contextKey ? current.cursors : []) }));
  };

  useEffect(() => {
    setPagination((current) => current.key === contextKey ? current : { key: contextKey, cursors: [] });
  }, [contextKey]);

  useEffect(() => {
    if (view === "risk-loss") return;
    const controller = new AbortController();
    setPopulation({ key: populationKey, state: "loading", page: emptyPopulation });
    void loadIndicatorPopulation({ kind, organizationScopeID, cursor, limit: 50 }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPopulation({ key: populationKey, state: "live", page: value });
    }).catch(() => {
      if (controller.signal.aborted) return;
      setPopulation({ key: populationKey, state: "error", page: { items: [], complete: false } });
    });
    return () => controller.abort();
  }, [cursor, kind, organizationScopeID, populationKey, retry, view]);

  useEffect(() => {
    if (view === "risk-loss") return;
    if (!targetID) {
      setSelected(undefined);
      return;
    }
    const inPage = page.items.find((item) => item.indicator.check_id === targetID);
    if (inPage) {
      setSelected({ key: selectedKey, state: "live", item: inPage });
      return;
    }
    const controller = new AbortController();
    setSelected({ key: selectedKey, state: "loading" });
    void loadIndicatorPopulation({ kind, organizationScopeID, checkID: targetID, limit: 1 }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      const item = value.items.find((candidate) => candidate.indicator.check_id === targetID);
      setSelected({ key: selectedKey, state: !item && (!value.complete || value.items.length > 0) ? "error" : "live", item });
    }).catch(() => {
      if (controller.signal.aborted) return;
      setSelected({ key: selectedKey, state: "error" });
    });
    return () => controller.abort();
  }, [kind, organizationScopeID, page.items, selectedKey, targetID, view]);

  const scopeNames = useMemo(
    () => new Map(organizationScopes.map((scope) => [scope.id, scope.department_path?.join(" / ") || scope.name])),
    [organizationScopes],
  );
  const columns: readonly DataColumn<IndicatorPopulationItem>[] = [
    {
      id: "indicator",
      header: "Indicator",
      mobileLayout: "full-width",
      render: (item) => <span className="insights-indicator__identity"><strong>{item.indicator.check_name}</strong><small>{item.indicator.link.kind} · {item.indicator.check_code}</small><small>{item.indicator.evaluated_at ? `Updated ${formatIndicatorDate(item.indicator.evaluated_at)}` : "No current result"}</small></span>,
      accessibleText: (item) => `${item.indicator.check_name}. ${item.indicator.link.kind}. ${item.indicator.check_code}. ${item.indicator.evaluated_at ? `Updated ${formatIndicatorDate(item.indicator.evaluated_at)}` : "No current result"}`,
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
  ];

  const scopeLabel = organizationScopeName || legalEntityName || "current scope";
  if (view === "risk-loss") {
    return <section className="insights-workspace">
      <header className="topbar cs-workspace-header">
        <div>
          <span className="eyebrow">{organizationName}</span>
          <h1>Insights</h1>
          <p>Risk position and selected-period losses for {scopeLabel}.</p>
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

  return <section className="insights-workspace insights-indicator-register" data-embedded={embedded || undefined}>
    <header className="insights-indicator__heading">
      <div>
        {!embedded && <span className="eyebrow">{organizationName}</span>}
        {embedded ? <h2>Key indicators</h2> : <h1>Key indicators</h1>}
        <p>Current indicators for {scopeLabel}.</p>
      </div>
      {!embedded && onOpenReports && <Button variant="secondary" onPress={onOpenReports}>Reports</Button>}
    </header>

    <div className="insights-indicator__controls">
      <div className="insights-indicator__type">
        <SelectField label="Indicator type" value={kind} placeholder="Select indicator type" options={indicatorKinds} allowsEmpty={false} onChange={(next) => { if (next) onKindChange(next); }}/>
      </div>
      <p className="insights-indicator__count">{state === "loading" ? "Loading indicators…" : state === "live" ? !page.complete && page.items.length === 0 ? "Indicator count unavailable" : `${page.items.length} ${page.items.length === 1 ? "indicator" : "indicators"} shown${page.next_cursor ? " · More available" : ""}` : "Indicator population unavailable"}</p>
    </div>

    <div className="insights-indicator__population">
      {!page.complete && state === "live" && <Notice tone="warning">Some indicator detail is unavailable in this scope. Available values remain unchanged. <Button variant="quiet" size="compact" onPress={() => setRetry((value) => value + 1)}>Retry</Button></Notice>}
      {page.truncated && state === "live" && <Notice tone="info">More indicators are available on the next page.</Notice>}
      {state === "error" && <EmptyState population="Indicators" title="Indicators could not be loaded" description="The current indicator population is unavailable." action={<Button variant="secondary" onPress={() => setRetry((value) => value + 1)}>Try again</Button>} role="alert"/>}
      {state === "live" && page.complete && page.items.length === 0 && <EmptyState population={kind === "KRI" ? "Risk indicators" : "Control indicators"} title="No indicators in this scope" description={kind === "KRI" ? "No risk indicators linked in this scope." : "No control indicators linked in this scope."}/>}
      {(page.items.length > 0 || state === "loading") && <DataTable
        ariaLabel={kind === "KRI" ? "Risk indicators" : "Control indicators"}
        rows={page.items}
        rowKey={(item) => item.indicator.check_id}
        rowName={(item) => item.indicator.check_name}
        columns={columns}
        onRowAction={(item) => onTarget(item.indicator.check_id, kind)}
        rowActionLabel="Review indicator"
        responsiveTo="container"
        isLoading={state === "loading"}
        pagination={(cursorStack.length > 0 || page.next_cursor) ? {
          label: "Indicator pages",
          onPrevious: cursorStack.length > 0 ? () => setCursorStack((current) => current.slice(0, -1)) : undefined,
          onNext: page.next_cursor ? () => setCursorStack((current) => [...current, page.next_cursor!]) : undefined,
          isLoading: state === "loading",
        } : undefined}
      />}
    </div>

    {targetID && targetState === "loading" && <span className="cs-sr-only" role="status">Loading selected indicator…</span>}
    {targetID && targetState === "error" && <Notice tone="error">The selected indicator could not be loaded. <Button variant="quiet" size="compact" onPress={() => onTarget(undefined, kind)}>Close selection</Button></Notice>}
    {targetID && targetState === "live" && !targetItem && <Notice tone="warning">The selected indicator is not in the current authorized scope. <Button variant="quiet" size="compact" onPress={() => onTarget(undefined, kind)}>Close selection</Button></Notice>}
    {targetItem && <FocusedSheet label={`${targetItem.indicator.check_name} indicator`} panelClassName="insights-indicator-inspector" onClose={() => onTarget(undefined, kind)}>
      <IndicatorDetail indicator={targetItem.indicator} presentation="inspector" onOpenProgram={onOpenProgram} onOpenMatter={onOpenMatter}/>
      <LinkedRisks item={targetItem} onOpenRisk={onOpenRisk}/>
    </FocusedSheet>}
  </section>;
}

function indicatorScopeLabel(item: IndicatorPopulationItem, scopeNames: Map<string, string>, selectedScope?: string, legalEntity?: string) {
  const ids = [...new Set(item.risks.map((risk) => risk.organization_scope_id).filter((id): id is string => Boolean(id)))];
  const riskCount = `${item.risk_count} linked ${item.risk_count === 1 ? "Risk" : "Risks"}`;
  if (ids.length > 2) return `${riskCount} across multiple areas`;
  const labels = ids.map((id) => scopeNames.get(id));
  if (item.risks_truncated || item.risk_count > item.risks.length || labels.some((label) => !label)) return `${riskCount} · scope details incomplete`;
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
