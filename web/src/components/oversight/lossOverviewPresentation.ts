import type { LossPeriodBundle, MoneyValue } from "../../metricApi";
import type { MetricTrendDatum } from "../ui";
import { formatLossMoneyExact } from "../losses/lossPresentation";

export type LossPresentationMode = "money" | "events";

export function lossPresentationMode(bundle: LossPeriodBundle): LossPresentationMode {
  return !bundle.mixed_currencies && bundle.net_loss ? "money" : "events";
}

export function lossPeriodValue(bundle: LossPeriodBundle) {
  if (lossPresentationMode(bundle) === "money" && bundle.net_loss) {
    return formatLossMoneyExact(bundle.net_loss.minor_units, bundle.net_loss.currency);
  }
  return `${bundle.event_count} ${bundle.event_count === 1 ? "event" : "events"}`;
}

export function lossComparisonLabel(bundle: LossPeriodBundle) {
  const comparison = bundle.comparison;
  if (lossPresentationMode(bundle) === "money") {
    if (comparison.comparison_quality !== "COMPLETE" || !comparison.net_delta) return "No comparable amount";
    const minor = parseMoneyMinor(comparison.net_delta);
    if (minor === undefined) return "No comparable amount";
    const amount = formatLossMoneyExact(bigintAbs(minor), comparison.net_delta.currency);
    if (comparison.direction === "IMPROVED") return `${amount} lower vs prior period`;
    if (comparison.direction === "WORSENED") return `${amount} higher vs prior period`;
    if (comparison.direction === "UNCHANGED") return "No change vs prior period";
    return "No comparable amount";
  }
  const delta = comparison.event_delta;
  if (delta > 0) return `${delta} more ${delta === 1 ? "event" : "events"} vs prior period`;
  if (delta < 0) {
    const count = Math.abs(delta);
    return `${count} fewer ${count === 1 ? "event" : "events"} vs prior period`;
  }
  return "No event-count change";
}

export function lossMovementPoints(bundle: LossPeriodBundle): MetricTrendDatum[] {
  if (lossPresentationMode(bundle) === "events") {
    return bundle.flow_points.map((point, index) => ({
      id: `${point.start}-${index}`,
      at: point.start,
      label: shortDate(point.start),
      value: point.loss_event_count,
      displayValue: `${point.loss_event_count} ${point.loss_event_count === 1 ? "event" : "events"}`,
    }));
  }

  const currency = bundle.net_loss?.currency;
  if (!currency) return [];
  const values = bundle.flow_points.map((point) => parseMoneyMinor(point.net_loss, currency) ?? 0n);
  const maxAbsolute = values.reduce((max, value) => bigintMax(max, bigintAbs(value)), 0n);

  return bundle.flow_points.map((point, index) => {
    const minor = values[index] ?? 0n;
    return {
      id: `${point.start}-${index}`,
      at: point.start,
      label: shortDate(point.start),
      value: normalizedMinor(minor, maxAbsolute),
      displayValue: formatLossMoneyExact(minor, currency),
    };
  });
}

export function parseMoneyMinor(value: MoneyValue | undefined, expectedCurrency?: string) {
  if (!value || (expectedCurrency && value.currency !== expectedCurrency)) return undefined;
  try { return BigInt(value.minor_units); } catch { return undefined; }
}

export function normalizedMinor(value: bigint, maximum: bigint) {
  if (maximum <= 0n) return 0;
  const scale = 1_000_000n;
  return Number(value * scale / maximum) / Number(scale);
}

export function bigintAbs(value: bigint) { return value < 0n ? -value : value; }
export function bigintMax(left: bigint, right: bigint) { return left > right ? left : right; }

function shortDate(value: string) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value.slice(0, 10);
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", timeZone: "UTC" }).format(parsed);
}
