import type { MonitoringNativeMeasurement } from "../../monitoringTypes";
import type { RiskIndicatorMovement } from "../../riskTypes";
import { formatIndicatorMeasurement } from "./IndicatorValue";

export function IndicatorMovement({ movement, measurement }: { movement?: RiskIndicatorMovement; measurement?: MonitoringNativeMeasurement }) {
  if (!movement || !measurement) return <span className="indicator-movement indicator-movement--empty">Not enough history</span>;
  const text = indicatorMovementText(movement, measurement);
  const arrow = movement.direction === "UP" ? "↑" : movement.direction === "DOWN" ? "↓" : "→";
  return <span className="indicator-movement">
    <span aria-hidden="true">{arrow}</span>
    <span>{text}</span>
  </span>;
}

export function indicatorMovementText(movement: RiskIndicatorMovement | undefined, measurement?: MonitoringNativeMeasurement) {
  if (!movement || !measurement) return "Not enough history";
  if (movement.direction === "FLAT") return "No change";
  if (measurement.unit === "PERCENT") {
    const formatted = formatIndicatorMeasurement(movement.delta, { ...measurement, unit: "COUNT" });
    return `${formatted} pp`;
  }
  return formatIndicatorMeasurement(movement.delta, measurement);
}
