import type { MonitoringNativeMeasurement, SourceOperator } from "../../monitoringTypes";
import "./indicator.css";

type Props = { measurement?: MonitoringNativeMeasurement; score?: number; denominator: number };

export function IndicatorValue({ measurement, score, denominator }: Props) {
  const concern = concernText(score, denominator);
  if (!measurement) return <span className="indicator-value"><strong>{concern}</strong><small>Native value unavailable</small></span>;
  return <span className="indicator-value">
    <strong>{measurement.value ? formatMeasurement(measurement.value, measurement) : "No observed value"}</strong>
    {measurement.limits?.length ? <small>{limitText(measurement)}</small> : <small>No limit configured</small>}
    <small>{concern}</small>
  </span>;
}

export function indicatorValueAccessibleText(measurement: MonitoringNativeMeasurement | undefined, score: number | undefined, denominator: number) {
  const concern = concernText(score, denominator);
  if (!measurement) return `${concern}. Native value unavailable.`;
  const value = measurement.value ? formatMeasurement(measurement.value, measurement) : "No observed value";
  const limits = measurement.limits?.length ? limitText(measurement) : "No limit configured";
  return `${value}. ${limits}. ${concern}.`;
}
function concernText(score: number | undefined, denominator: number) {
  if (score === undefined) return "Concern not assessed";
  return `${formatConcern(score)} / ${denominator} concern`;
}
function limitText(measurement: MonitoringNativeMeasurement) {
  const limits = measurement.limits ?? [];
  const prefix = limits.length === 1 ? "Limit" : "Limits";
  return `${prefix} ${limits.map((limit) => `${operatorSymbol(limit.operator)} ${formatMeasurement(limit.expected, measurement)}`).join(" · ")}`;
}
function operatorSymbol(operator: SourceOperator) {
  switch (operator) {
    case "EQUALS": return "=";
    case "NOT_EQUALS": return "≠";
    case "GREATER_THAN": return ">";
    case "GREATER_OR_EQUAL": return "≥";
    case "LESS_THAN": return "<";
    case "LESS_OR_EQUAL": return "≤";
    default: return operator;
  }
}
function formatMeasurement(value: string, measurement: MonitoringNativeMeasurement) {
  const numeric = groupExactDecimal(value, measurement.precision ?? 0);
  switch (measurement.unit) {
    case "PERCENT": return `${numeric}%`;
    case "MONEY": return `${measurement.currency ?? "Currency"} ${numeric}`;
    case "DURATION": return `${numeric} ${durationLabel(measurement.duration_unit)}`;
    default: return numeric;
  }
}
function durationLabel(unit: MonitoringNativeMeasurement["duration_unit"]) {
  switch (unit) {
    case "SECONDS": return "sec";
    case "MINUTES": return "min";
    case "HOURS": return "hr";
    case "DAYS": return "days";
    default: return "duration";
  }
}
function groupExactDecimal(value: string, minimumFractionDigits: number) {
  const trimmed = value.trim();
  const match = /^([+-]?)(\d+)(?:\.(\d+))?$/.exec(trimmed);
  if (!match) return trimmed;
  const sign = match[1] ?? "";
  const integer = match[2] ?? "";
  const sourceFraction = match[3] ?? "";
  const fraction = sourceFraction.padEnd(Math.max(sourceFraction.length, minimumFractionDigits), "0");
  const grouped = integer.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return `${sign}${grouped}${fraction ? `.${fraction}` : ""}`;
}
function formatConcern(value: number) {
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}
