import { useEffect, useMemo, useState } from "react";
import {
  loadDomainMetrics,
  loadLossPeriodMetrics,
  type DomainMetricBundle,
  type LossPeriodBundle,
  type MoneyValue,
} from "../../metricApi";
import type { ReportingPeriodQuery } from "../../reportingPeriod";
import { parseMoneyMinor } from "../oversight/lossOverviewPresentation";

type LoadState = "loading" | "live" | "unavailable";
export type RiskLossMetricOptions = {
  metricPeriod?: ReportingPeriodQuery;
  metricState: "idle" | LoadState;
  scopeAuthorized: boolean;
  legalEntityID?: string;
  organizationScopeID?: string;
  riskScope?: "selected" | "legal-entity";
  requireScopeID?: boolean;
  refreshKey?: number;
  loadDomain?: typeof loadDomainMetrics;
  loadLoss?: typeof loadLossPeriodMetrics;
};
type Metrics = {
  domain: DomainMetricBundle | null;
  loss: LossPeriodBundle | null;
  domainState: LoadState;
  lossState: LoadState;
};

export function useRiskLossMetrics({
  metricPeriod, metricState, scopeAuthorized, legalEntityID, organizationScopeID,
  riskScope = "selected", requireScopeID = true,
  refreshKey = 0, loadDomain = loadDomainMetrics, loadLoss = loadLossPeriodMetrics,
}: RiskLossMetricOptions): Metrics {
  const startDate = metricPeriod?.start_date;
  const endDate = metricPeriod?.end_date;
  const request = useMemo(() => ({
    startDate, endDate, metricState, scopeAuthorized, legalEntityID,
    organizationScopeID, riskScope, requireScopeID, refreshKey, loadDomain, loadLoss,
  }), [startDate, endDate, metricState, scopeAuthorized, legalEntityID,
    organizationScopeID, riskScope, requireScopeID, refreshKey, loadDomain, loadLoss]);
  const [result, setResult] = useState<(Metrics & { request: typeof request }) | null>(null);
  const expectedScope = organizationScopeID || legalEntityID;
  const hasScope = Boolean(expectedScope) || !requireScopeID;
  const enabled = scopeAuthorized && metricState === "live" && hasScope
    && validDate(startDate) && validDate(endDate) && startDate <= endDate;
  const pending = scopeAuthorized && hasScope && (enabled || metricState === "loading");

  useEffect(() => {
    if (!enabled) return;
    const controller = new AbortController();
    const period = { start_date: request.startDate!, end_date: request.endDate! };
    const scopeID = request.organizationScopeID || request.legalEntityID;
    const scopeKind = request.organizationScopeID ? "ORGANIZATION_SCOPE" : "LEGAL_ENTITY";
    const riskOrganizationScope = request.riskScope === "legal-entity" ? undefined : request.organizationScopeID;
    const riskScopeID = riskOrganizationScope || request.legalEntityID;
    const riskScopeKind = riskOrganizationScope ? "ORGANIZATION_SCOPE" : "LEGAL_ENTITY";
    setResult({ request, domain: null, loss: null, domainState: "loading", lossState: "loading" });

    function publish(update: Partial<Metrics>) {
      if (controller.signal.aborted) return;
      setResult((current) => current?.request === request ? { ...current, ...update } : current);
    }

    async function readDomain() {
      try {
        const value = await request.loadDomain(riskOrganizationScope, controller.signal);
        if (!matchesScope(value, riskScopeID, riskScopeKind)
          || !Array.isArray(value.items) || value.items.some((item) => !item || typeof item.id !== "string")) {
          publish({ domainState: "unavailable" });
          return;
        }
        publish({ domain: value, domainState: "live" });
      } catch {
        publish({ domainState: "unavailable" });
      }
    }

    async function readLoss() {
      try {
        const value = await request.loadLoss(period, request.organizationScopeID, controller.signal);
        if (!matchesScope(value, scopeID, scopeKind)
          || responseDate(value.period_start) !== period.start_date
          || responseDate(value.period_end) !== period.end_date
          || !hasLossPresentationFields(value)) {
          publish({ lossState: "unavailable" });
          return;
        }
        publish({ loss: value, lossState: "live" });
      } catch {
        publish({ lossState: "unavailable" });
      }
    }

    void readDomain();
    void readLoss();
    return () => controller.abort();
  }, [enabled, request]);

  // Suppress the previous selection during render, before effect cleanup runs.
  if (!enabled || result?.request !== request) {
    return { domain: null, loss: null, domainState: pending ? "loading" : "unavailable", lossState: pending ? "loading" : "unavailable" };
  }
  return result;
}

function matchesScope(value: DomainMetricBundle | LossPeriodBundle | undefined, scopeID: string | undefined, scopeKind: string) {
  return Boolean(value && value.scope_kind === scopeKind && typeof value.scope_id === "string"
    && value.scope_id.length > 0 && (!scopeID || value.scope_id === scopeID));
}

function validDate(value: string | undefined): value is string {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const at = Date.parse(`${value}T00:00:00Z`);
  return Number.isFinite(at) && new Date(at).toISOString().slice(0, 10) === value;
}

function responseDate(value: string | undefined) {
  if (typeof value !== "string" || !Number.isFinite(Date.parse(value))) return undefined;
  const date = value.slice(0, 10);
  return validDate(date) ? date : undefined;
}

// Validate only fields consumed by the Overview amount and LossMovement.
function hasLossPresentationFields(value: LossPeriodBundle) {
  const comparison = value.comparison;
  const currency = !value.mixed_currencies ? value.net_loss?.currency : undefined;
  return typeof value.mixed_currencies === "boolean"
    && Number.isSafeInteger(value.event_count) && value.event_count >= 0
    && (value.flow_resolution === "DAY" || value.flow_resolution === "WEEK")
    && validOptionalMoney(value.net_loss)
    && Array.isArray(value.flow_points) && value.flow_points.every((point) => point
      && responseDate(point.start) !== undefined
      && Number.isSafeInteger(point.loss_event_count) && point.loss_event_count >= 0
      && (!currency || point.net_loss !== undefined)
      && validOptionalMoney(point.net_loss, currency))
    && Array.isArray(value.currencies) && value.currencies.every((item) => item
      && typeof item.currency === "string" && item.currency.trim().length > 0)
    && comparison && typeof comparison === "object"
    && ["COMPLETE", "LIMITED", "MISSING"].includes(comparison.comparison_quality)
    && ["IMPROVED", "WORSENED", "UNCHANGED", "UNKNOWN"].includes(comparison.direction)
    && Number.isSafeInteger(comparison.event_delta)
    && validOptionalMoney(comparison.net_delta, currency);
}

function validOptionalMoney(value: MoneyValue | undefined, currency?: string) {
  return value === undefined || Boolean(value
    && typeof value.currency === "string" && value.currency.trim().length > 0
    && typeof value.minor_units === "string" && value.minor_units.trim().length > 0
    && parseMoneyMinor(value, currency) !== undefined);
}
