import type { LossPeriodBundle, MoneyValue } from "../../metricApi";
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
    const amount = formatLossMoneyExact(comparison.net_delta.minor_units, comparison.net_delta.currency);
    if (comparison.direction === "IMPROVED") return `${amount} improved vs prior period`;
    if (comparison.direction === "WORSENED") return `${prefixPositive(amount)} worse vs prior period`;
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

function prefixPositive(value: string) {
  return value.startsWith("-") || value.startsWith("+") ? value : `+${value}`;
}