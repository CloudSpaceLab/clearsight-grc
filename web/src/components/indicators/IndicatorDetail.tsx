import { useEffect, useState } from "react";
import { loadMonitoringResults } from "../../monitoringApi";
import type { MonitoringResult } from "../../monitoringTypes";
import type { RiskIndicatorDetail } from "../../riskTypes";
import { Button, DataTable, EmptyState, Notice, StatusBadge, Surface, type DataColumn } from "../ui";
import { IndicatorValue, indicatorValueAccessibleText } from "./IndicatorValue";
import { formatIndicatorCoverage, formatIndicatorDate, indicatorStateLabel, indicatorTone, monitoringBandLabel, nativeConditionLabel, nativeConditionTone } from "./indicatorPresentation";
import "./indicator.css";

type Props = {
  indicator: RiskIndicatorDetail;
  onOpenProgram?: (programID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  loadResults?: (checkID: string, version?: number) => Promise<MonitoringResult[]>;
};

type LoadState = "loading" | "live" | "error";

export function IndicatorDetail({
  indicator,
  onOpenProgram,
  onOpenMatter,
  loadResults = loadMonitoringResults,
}: Props) {
  const [state, setState] = useState<LoadState>("loading");
  const [results, setResults] = useState<MonitoringResult[]>([]);
  const [retry, setRetry] = useState(0);

  useEffect(() => {
    let active = true;
    setState("loading");
    void loadResults(indicator.check_id, indicator.check_version).then((value) => {
      if (!active) return;
      setResults(value);
      setState("live");
    }).catch(() => {
      if (!active) return;
      setResults([]);
      setState("error");
    });
    return () => { active = false; };
  }, [indicator.check_id, indicator.check_version, loadResults, retry]);

  const columns: readonly DataColumn<MonitoringResult>[] = [
    {
      id: "observed",
      header: "Observed",
      render: (item) => formatIndicatorDate(item.evaluated_at),
      accessibleText: (item) => formatIndicatorDate(item.evaluated_at),
    },
    {
      id: "value",
      header: "Value",
      mobileLayout: "full-width",
      render: (item) => <IndicatorValue measurement={item.evaluation.measurement} score={item.evaluation.score} denominator={100}/>,
      accessibleText: (item) => indicatorValueAccessibleText(item.evaluation.measurement, item.evaluation.score, 100),
    },
    {
      id: "condition",
      header: "Condition",
      kind: "status",
      render: (item) => item.evaluation.measurement
        ? <StatusBadge tone={nativeConditionTone(item.evaluation.measurement.condition)}>{nativeConditionLabel(item.evaluation.measurement.condition)}</StatusBadge>
        : "—",
      accessibleText: (item) => item.evaluation.measurement ? nativeConditionLabel(item.evaluation.measurement.condition) : "Native condition unavailable",
    },
    {
      id: "concern",
      header: "Concern",
      render: (item) => monitoringBandLabel(item.evaluation.band),
      accessibleText: (item) => monitoringBandLabel(item.evaluation.band),
    },
    {
      id: "coverage",
      header: "Coverage",
      render: (item) => formatIndicatorCoverage(item.evaluation.coverage),
      accessibleText: (item) => formatIndicatorCoverage(item.evaluation.coverage),
    },
  ];

  return <div className="indicator-detail">
    <header className="cs-sheet-heading">
      <p>{indicator.link.kind} · {indicator.check_code}</p>
      <h2>{indicator.check_name}</h2>
      <p>{indicator.claim}</p>
    </header>

    <Surface>
      <dl className="indicator-detail__summary" aria-label="Current indicator state">
        <div><dt>Current state</dt><dd><StatusBadge tone={indicatorTone(indicator.state)}>{indicatorStateLabel(indicator.state)}</StatusBadge></dd></div>
        <div><dt>Current value</dt><dd><IndicatorValue measurement={indicator.native_measurement} score={indicator.score} denominator={indicator.denominator}/></dd></div>
        <div><dt>Coverage</dt><dd>{indicator.coverage === undefined ? "No current coverage" : `${formatIndicatorCoverage(indicator.coverage)} · minimum ${formatIndicatorCoverage(indicator.minimum_coverage)}`}</dd></div>
        <div><dt>Updated</dt><dd>{indicator.evaluated_at ? formatIndicatorDate(indicator.evaluated_at) : "No current result"}</dd></div>
      </dl>
    </Surface>

    <dl className="cs-sheet-facts">
      <div><dt>Program</dt><dd>{indicator.program_name}</dd></div>
      <div><dt>Owner</dt><dd>{indicator.owner_display_name || "Not assigned"}</dd></div>
      <div><dt>Reviewer</dt><dd>{indicator.reviewer_display_name || "Not assigned"}</dd></div>
      <div><dt>Freshness limit</dt><dd>{indicator.freshness_minutes} minutes</dd></div>
    </dl>

    <div className="indicator-detail__actions">
      {onOpenProgram && <Button variant="secondary" size="compact" onPress={() => onOpenProgram(indicator.program_id)}>Open Program</Button>}
      {indicator.intervention && onOpenMatter && <Button size="compact" onPress={() => onOpenMatter(indicator.intervention!.matter_id)}>Open intervention</Button>}
    </div>

    <section className="indicator-detail__history" aria-labelledby="indicator-history-heading">
      <div className="section-header">
        <div>
          <h3 id="indicator-history-heading">Recent observations</h3>
          <p>Results from monitoring check revision {indicator.check_version}. Earlier check definitions are excluded.</p>
        </div>
      </div>
      {state === "loading" && results.length === 0 && <p role="status">Loading indicator history…</p>}
      {state === "error" && <Notice tone="error"><span>Indicator history could not be loaded.</span> <Button variant="secondary" size="compact" onPress={() => setRetry((value) => value + 1)}>Try again</Button></Notice>}
      {state === "live" && results.length === 0 && <EmptyState population={indicator.check_name} title="No observations for this revision" description="No monitoring result has been recorded for the current check revision."/>}
      {results.length > 0 && <DataTable
        ariaLabel={`${indicator.check_name} observation history`}
        rows={results}
        rowKey={(item) => item.id}
        rowName={(item) => `${formatIndicatorDate(item.evaluated_at)}, ${monitoringBandLabel(item.evaluation.band)}`}
        columns={columns}
        isLoading={state === "loading"}
      />}
    </section>
  </div>;
}
