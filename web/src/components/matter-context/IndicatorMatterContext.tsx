import { useEffect, useState } from "react";
import { loadIndicatorPopulation } from "../../indicatorApi";
import type { IndicatorPopulationItem } from "../../indicatorTypes";
import { loadMonitoringResult } from "../../monitoringApi";
import type { MonitoringResult } from "../../monitoringTypes";
import { Button, Notice, StatusBadge, Surface } from "../ui";
import { IndicatorLimit, IndicatorObservedValue } from "../indicators/IndicatorValue";
import {
  formatIndicatorCoverage,
  formatIndicatorDate,
  formatIndicatorPeriod,
  monitoringBandLabel,
  nativeConditionLabel,
  nativeConditionTone,
} from "../indicators/indicatorPresentation";

type Props = {
  resultID: string;
  onOpenIndicator?: (indicatorID: string, kind?: "KRI" | "KCI") => void;
  loadResult?: typeof loadMonitoringResult;
  loadIndicators?: typeof loadIndicatorPopulation;
};

type LoadState = "loading" | "live" | "unavailable";

export function IndicatorMatterContext({
  resultID,
  onOpenIndicator,
  loadResult = loadMonitoringResult,
  loadIndicators = loadIndicatorPopulation,
}: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [result, setResult] = useState<MonitoringResult>();
  const [indicator, setIndicator] = useState<IndicatorPopulationItem>();
  const [indicatorAvailable, setIndicatorAvailable] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setResult(undefined);
    setIndicator(undefined);
    setIndicatorAvailable(true);

    void loadResult(resultID, controller.signal).then(async (value) => {
      if (controller.signal.aborted) return;
      if (value.id !== resultID) {
        setState("unavailable");
        return;
      }
      setResult(value);
      setState("live");
      try {
        const page = await loadIndicators({ checkID: value.monitoring_check_id, limit: 2 }, controller.signal);
        if (controller.signal.aborted) return;
        setIndicator(page.items.find((item) => item.indicator.check_id === value.monitoring_check_id));
        setIndicatorAvailable(page.complete !== false);
      } catch {
        if (controller.signal.aborted) return;
        setIndicatorAvailable(false);
      }
    }).catch(() => {
      if (controller.signal.aborted) return;
      setState("unavailable");
    });

    return () => controller.abort();
  }, [loadIndicators, loadResult, resultID]);

  if (state === "loading") return <p className="matter-domain-context-state" role="status">Loading source observation…</p>;
  if (state === "unavailable" || !result) {
    return <Notice tone="warning">Source observation is unavailable. Issue work remains available.</Notice>;
  }

  const measurement = result.evaluation.measurement;
  const title = indicator?.indicator.check_name ?? "Monitoring result";
  const kind = indicator?.indicator.link.kind;

  return <Surface>
    <section className="matter-domain-context" aria-label="Indicator observation context">
      <div className="matter-record-section-heading">
        <div>
          <span className="eyebrow">{kind ? kind + " observation" : "Indicator observation"}</span>
          <h2>{title}</h2>
        </div>
        {indicator && onOpenIndicator && <Button
          variant="secondary"
          size="compact"
          onPress={() => onOpenIndicator(result.monitoring_check_id, kind)}
        >Open indicator</Button>}
      </div>
      {!indicatorAvailable && <Notice tone="warning">Current indicator details are unavailable. The source observation remains available.</Notice>}
      <dl className="matter-record-facts">
        <div><dt>Observed value</dt><dd><IndicatorObservedValue measurement={measurement} score={result.evaluation.score} denominator={100}/></dd></div>
        <div><dt>Approved limit</dt><dd><IndicatorLimit measurement={measurement}/></dd></div>
        <div><dt>Condition</dt><dd>{measurement
          ? <StatusBadge tone={nativeConditionTone(measurement.condition)}>{nativeConditionLabel(measurement.condition)}</StatusBadge>
          : "Native value unavailable"}</dd></div>
        <div><dt>Business period</dt><dd>{formatIndicatorPeriod(measurement?.reporting_period_start, measurement?.reporting_period_end)}</dd></div>
        <div><dt>Observed</dt><dd>{formatIndicatorDate(result.evaluated_at)}</dd></div>
        <div><dt>Concern</dt><dd>{monitoringBandLabel(result.evaluation.band)}</dd></div>
        <div><dt>Coverage</dt><dd>{formatIndicatorCoverage(result.evaluation.coverage)}</dd></div>
      </dl>
    </section>
  </Surface>;
}
