import { useMemo } from "react";
import type { LossOrganizationFlow, LossPeriodBundle } from "../../metricApi";
import { formatLossMoneyExact } from "../losses/lossPresentation";
import { EmptyState, Notice, RankedBarList, type RankedBarItem } from "../ui";
import {
  bigintMax,
  lossPresentationMode,
  normalizedMinor,
  parseMoneyMinor,
} from "./lossOverviewPresentation";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  bundle: LossPeriodBundle | null;
  state: LoadState;
  onOpenScope?: (scopeID: string) => void;
};

type RowModel = {
  id: string;
  label: string;
  scopeID?: string;
  kind: LossOrganizationFlow["kind"];
  events: number;
  contributors: number;
  minor?: bigint;
  currency?: string;
  currencyCount: number;
};

export function OrganizationLossSummary({ bundle, state, onOpenScope }: Props) {
  const rows = useMemo(() => bundle ? lossRows(bundle) : [], [bundle]);
  const items = useMemo(() => rankedRows(rows, bundle), [rows, bundle]);

  return <section className="organization-loss-summary" aria-labelledby="organization-loss-summary-heading">
    <div className="section-header">
      <div>
        <span className="eyebrow">Where Loss is concentrated</span>
        <h2 id="organization-loss-summary-heading">Loss concentration</h2>
        <p>{bundle && lossPresentationMode(bundle) === "money" ? "Net Loss flow by area." : "Loss events by area; currencies stay separate."}</p>
      </div>
    </div>

    {state === "loading" && <p className="oversight-today-status" role="status" aria-busy="true">Loading Loss concentration…</p>}
    {state === "unavailable" && <Notice tone="warning">Loss concentration is unavailable for this period.</Notice>}
    {state === "live" && bundle && bundle.event_count === 0 && bundle.contributing_loss_count === 0 && <EmptyState
      population="Operational Loss flow in the selected period"
      title="No Loss flow in this period"
      description="No active Loss or recovery contributed to this scope during the selected period."
    />}
    {state === "live" && bundle && bundle.contributing_loss_count > 0 && <RankedBarList
      ariaLabel={lossPresentationMode(bundle) === "money"
        ? "Net operational Loss by organization area"
        : "Operational Loss events by organization area"}
      items={items}
      onAction={onOpenScope ? (item) => {
        const row = rows.find((candidate) => candidate.id === item.id);
        if (row?.scopeID) onOpenScope(row.scopeID);
      } : undefined}
    />}
  </section>;
}

function lossRows(bundle: LossPeriodBundle): RowModel[] {
  const mode = lossPresentationMode(bundle);
  const currency = bundle.net_loss?.currency;
  const organizations = bundle.organization_breakdown
    .filter((item) => item.kind === "ORGANIZATION_SCOPE")
    .map((item) => rowFromFlow(item, mode, currency))
    .sort((left, right) => compareRows(right, left, mode));

  const visible = organizations.slice(0, 6);
  const hidden = organizations.slice(6);
  const residual = bundle.organization_breakdown
    .filter((item) => item.kind !== "ORGANIZATION_SCOPE")
    .map((item) => rowFromFlow(item, mode, currency));

  if (!hidden.length) return [...visible, ...residual];

  const other: RowModel = {
    id: "other-areas",
    label: "Other areas",
    kind: "UNAVAILABLE",
    events: hidden.reduce((sum, item) => sum + item.events, 0),
    contributors: hidden.reduce((sum, item) => sum + item.contributors, 0),
    currencyCount: mode === "money" ? 1 : new Set(hidden.map((item) => item.currency).filter(Boolean)).size,
  };
  if (mode === "money") {
    other.currency = currency;
    other.minor = hidden.reduce((sum, item) => sum + (item.minor ?? 0n), 0n);
  }
  return [...visible, other, ...residual];
}

function rowFromFlow(flow: LossOrganizationFlow, mode: "money" | "events", expectedCurrency?: string): RowModel {
  return {
    id: flow.key,
    label: flow.label,
    scopeID: flow.scope_id,
    kind: flow.kind,
    events: flow.loss_event_count,
    contributors: flow.contributing_loss_count,
    minor: mode === "money" ? parseMoneyMinor(flow.net_loss, expectedCurrency) ?? 0n : undefined,
    currency: mode === "money" ? expectedCurrency : flow.currencies.length === 1 ? flow.currencies[0]?.currency : undefined,
    currencyCount: flow.currencies.length,
  };
}

function rankedRows(rows: RowModel[], bundle: LossPeriodBundle | null): RankedBarItem[] {
  if (!bundle) return [];
  const mode = lossPresentationMode(bundle);

  if (mode === "events") {
    return rows.map((row) => ({
      id: row.id,
      label: row.label,
      value: row.events,
      displayValue: `${row.events} ${row.events === 1 ? "event" : "events"}`,
      meta: rowMeta(row, mode),
      actionLabel: row.scopeID ? `Open ${row.label}` : undefined,
      isDisabled: !row.scopeID,
    }));
  }

  const currency = bundle.net_loss?.currency;
  if (!currency) return [];
  const maxPositive = rows.reduce((max, row) => bigintMax(max, (row.minor ?? 0n) > 0n ? row.minor! : 0n), 0n);

  return rows.map((row) => {
    const minor = row.minor ?? 0n;
    return {
      id: row.id,
      label: row.label,
      value: normalizedMinor(minor > 0n ? minor : 0n, maxPositive),
      displayValue: formatLossMoneyExact(minor, currency),
      meta: rowMeta(row, mode),
      actionLabel: row.scopeID ? `Open ${row.label}` : undefined,
      isDisabled: !row.scopeID,
    };
  });
}

function compareRows(left: RowModel, right: RowModel, mode: "money" | "events") {
  if (mode === "events") return left.events - right.events || right.label.localeCompare(left.label);
  const leftMinor = left.minor ?? 0n;
  const rightMinor = right.minor ?? 0n;
  if (leftMinor === rightMinor) return right.label.localeCompare(left.label);
  return leftMinor > rightMinor ? 1 : -1;
}

function rowMeta(row: RowModel, mode: "money" | "events") {
  const contributors = `${row.contributors} contributing ${row.contributors === 1 ? "Loss" : "Losses"}`;
  const flowState = mode === "money" && row.minor !== undefined
    ? row.minor < 0n ? "Net recovery · " : row.minor === 0n ? "No net Loss · " : ""
    : "";
  if (row.kind === "DIRECT") return `Direct · ${flowState}${contributors}`;
  if (row.kind === "UNATTRIBUTED") return `No area recorded · ${flowState}${contributors}`;
  if (row.kind === "UNAVAILABLE") {
    return row.id === "other-areas" ? `${flowState}${contributors} across remaining areas` : `Area unavailable · ${flowState}${contributors}`;
  }
  if (mode === "events" && row.currencyCount > 1) return `${row.currencyCount} currencies · ${contributors}`;
  return `${flowState}${contributors}`;
}
