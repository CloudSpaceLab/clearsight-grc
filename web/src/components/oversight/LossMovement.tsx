import { useMemo } from "react";
import type { LossPeriodBundle } from "../../metricApi";
import { MetricTrend, Notice } from "../ui";
import {
  lossComparisonLabel,
  lossMovementPoints,
  lossPeriodValue,
  lossPresentationMode,
} from "./lossOverviewPresentation";

type LoadState = "loading" | "live" | "unavailable";

type Props = {
  bundle: LossPeriodBundle | null;
  state: LoadState;
};

export function LossMovement({ bundle, state }: Props) {
  const points = useMemo(() => bundle ? lossMovementPoints(bundle) : [], [bundle]);
  const mode = bundle ? lossPresentationMode(bundle) : "events";
  const gapThresholdMs = bundle?.flow_resolution === "WEEK"
    ? 8 * 24 * 60 * 60 * 1000
    : 36 * 60 * 60 * 1000;

  return <section className="loss-movement" aria-labelledby="loss-movement-heading">
    <div className="section-header">
      <div>
        <h2 id="loss-movement-heading">Loss trend</h2>
        <p>{mode === "money" ? "Net Loss flow across the selected period." : "Loss events across the selected period."}</p>
      </div>
    </div>

    {bundle && <div className="risk-movement__summary" aria-live="polite">
      <strong>{lossPeriodValue(bundle)}</strong>
      <span>{lossComparisonLabel(bundle)}</span>
    </div>}

    {state === "loading" && <p className="oversight-today-status" role="status" aria-busy="true">Loading loss trend…</p>}
    {state === "unavailable" && <Notice tone="warning">Loss trend is unavailable for this period.</Notice>}
    {state === "live" && bundle && points.length > 0 && <MetricTrend
      ariaLabel={mode === "money" ? "Net operational loss trend" : "Operational loss event trend"}
      points={points}
      gapThresholdMs={gapThresholdMs}
    />}
    {state === "live" && bundle && points.length === 0 && <p className="risk-movement__empty">No Loss flow in this period.</p>}
    {state === "live" && bundle?.mixed_currencies && <p className="loss-movement__currency-note">Amounts remain separated: {bundle.currencies.map((item) => item.currency).join(", ")}.</p>}
  </section>;
}
