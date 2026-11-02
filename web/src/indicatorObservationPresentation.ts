import type { MetricTrendDatum } from "./components/ui/MetricTrend";
import type { MonitoringNativeMeasurement, MonitoringResult } from "./monitoringTypes";
import type { RiskIndicatorDetail } from "./riskTypes";
import { formatIndicatorMeasurement } from "./components/indicators/IndicatorValue";

function signature(value: MonitoringNativeMeasurement) {
  return JSON.stringify([
    value.field, value.unit, value.currency ?? "", value.duration_unit ?? "",
    value.precision ?? 0, value.limits ?? [],
  ]);
}

export function comparableIndicatorObservations(indicator: RiskIndicatorDetail, results: readonly MonitoringResult[]) {
  const expected = indicator.native_measurement;
  if (!expected) return { points: [] as MetricTrendDatum[], omitted: results.length, available: false, range: undefined };
  const key = signature(expected);
  const sorted = [...results].sort((a, b) => Date.parse(b.evaluated_at) - Date.parse(a.evaluated_at));
  const points: MetricTrendDatum[] = [];
  let omitted = 0;
  for (const row of sorted) {
    const measure = row.evaluation.measurement;
    const text = measure?.value?.trim() ?? "";
    // The trend is a visual aid, not a financial calculator. Avoid numeric precision loss.
    const safeDecimal = /^[+-]?(?:\d+(?:\.\d+)?|\.\d+)$/.test(text) && text.replace(/[^\d]/g, "").length <= 12;
    const value = safeDecimal ? Number(text) : NaN;
    if (row.monitoring_check_id !== indicator.check_id
      || row.monitoring_check_version !== indicator.check_version
      || !measure || signature(measure) !== key
      || measure.condition === "UNKNOWN"
      || !Number.isFinite(value) || !Number.isFinite(Date.parse(row.evaluated_at))) {
      omitted++;
      continue;
    }
    if (points.length >= 12) break;
    points.push({
      id: row.id, at: row.evaluated_at,
      label: new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(row.evaluated_at)),
      value, displayValue: formatIndicatorMeasurement(text, expected),
    });
  }
  const low = points.reduce<MetricTrendDatum | undefined>((min, point) => !min || point.value < min.value ? point : min, undefined);
  const high = points.reduce<MetricTrendDatum | undefined>((max, point) => !max || point.value > max.value ? point : max, undefined);
  return {
    points: points.reverse(), omitted, available: points.length >= 2,
    range: low && high ? { min: low.displayValue ?? String(low.value), max: high.displayValue ?? String(high.value) } : undefined,
  };
}
