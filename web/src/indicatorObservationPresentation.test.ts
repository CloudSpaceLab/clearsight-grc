import { expect, it } from "vitest";
import type { MonitoringNativeMeasurement, MonitoringResult } from "./monitoringTypes";
import type { RiskIndicatorDetail } from "./riskTypes";
import { comparableIndicatorObservations } from "./indicatorObservationPresentation";

const measurement: MonitoringNativeMeasurement = {
  field: "success_rate", unit: "PERCENT", precision: 2,
  value: "98.70", limits: [{ operator: "GREATER_OR_EQUAL", expected: "99.50" }],
};
const indicator = {
  check_id: "check-1", check_version: 4, native_measurement: measurement,
  check_name: "Success rate",
} as RiskIndicatorDetail;

function result(id: string, value: string, extra: Partial<MonitoringNativeMeasurement> = {}, version = 4): MonitoringResult {
  return {
    id, monitoring_check_id: "check-1", monitoring_check_version: version,
    evaluated_at: id === "older" ? "2026-10-06T09:00:00Z" : "2026-10-06T10:00:00Z",
    evaluation: {
      band: "LOW", coverage: 1,
      measurement: { ...measurement, value, condition: "WITHIN", ...extra },
    },
  };
}

it("plots comparable numeric native observations without substituting concern points", () => {
  const view = comparableIndicatorObservations(indicator, [
    result("newer", "98.70"), result("older", "99.80"),
  ]);
  expect(view.available).toBe(true);
  expect(view.points.map((point) => point.value)).toEqual([99.8, 98.7]);
  expect(view.points[0]?.displayValue).toBe("99.80%");
  expect(view.omitted).toBe(0);
});

it("excludes changed thresholds, changed unit, unknown, stale revision, and unsafe values", () => {
  const view = comparableIndicatorObservations(indicator, [
    result("valid", "98.70"),
    result("threshold", "90", { limits: [{ operator: "GREATER_OR_EQUAL", expected: "85" }] }),
    result("unit", "90", { unit: "COUNT" }),
    result("unknown", "90", { condition: "UNKNOWN" }),
    result("wrong-revision", "90", {}, 3),
    result("large", "12345678901234567890"),
  ]);
  expect(view.points).toHaveLength(1);
  expect(view.available).toBe(false);
  expect(view.omitted).toBe(5);
});

it("does not plot synthetic native values when only a concern score exists", () => {
  const view = comparableIndicatorObservations({ ...indicator, native_measurement: undefined }, [
    result("newer", "98.70"), result("older", "99.80"),
  ]);
  expect(view.available).toBe(false);
  expect(view.points).toHaveLength(0);
});
