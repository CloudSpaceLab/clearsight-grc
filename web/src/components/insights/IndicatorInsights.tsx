import { useEffect, useState } from "react";
import { loadIndicatorInsights, type IndicatorInsight, type IndicatorInsightsPage } from "../../indicatorInsightsApi";
import type { RiskIndicatorKind } from "../../riskTypes";
import { IndicatorDetail } from "../indicators/IndicatorDetail";
import { IndicatorValue, indicatorValueAccessibleText } from "../indicators/IndicatorValue";
import { formatIndicatorCoverage, formatIndicatorDate, indicatorStateLabel, indicatorTone } from "../indicators/indicatorPresentation";
import { Button, DataTable, EmptyState, FilterBar, FocusedSheet, Notice, SelectField, StatusBadge, type DataColumn } from "../ui";
import "./insights.css";

type Props = {
  legalEntityName?: string;
  onOpenProgram?: (programID: string) => void;
  onOpenMatter?: (matterID: string) => void;
  loadPage?: (params: { kind?: RiskIndicatorKind; cursor?: string; limit?: number }, signal?: AbortSignal) => Promise<IndicatorInsightsPage>;
};

type LoadState = "loading" | "live" | "error";

const kindOptions: ReadonlyArray<{ id: RiskIndicatorKind; label: string }> = [
  { id: "KRI", label: "Key risk indicators" },
  { id: "KCI", label: "Key control indicators" },
];

export function IndicatorInsights({
  legalEntityName,
  onOpenProgram,
  onOpenMatter,
  loadPage = loadIndicatorInsights,
}: Props) {
  const [kind, setKind] = useState<RiskIndicatorKind>();
  const [cursors, setCursors] = useState<string[]>([]);
  const [page, setPage] = useState<IndicatorInsightsPage>({ items: [], complete: true, generated_at: "" });
  const [state, setState] = useState<LoadState>("loading");
  const [selected, setSelected] = useState<IndicatorInsight>();
  const [retry, setRetry] = useState(0);
  const currentCursor = cursors[cursors.length - 1];

  useEffect(() => {
    const controller = new AbortController();
    setState("loading");
    setSelected(undefined);
    void loadPage({ kind, cursor: currentCursor, limit: 25 }, controller.signal).then((value) => {
      if (controller.signal.aborted) return;
      setPage(value);
      setState("live");
    }).catch((error: unknown) => {
      if (controller.signal.aborted || isAbortError(error)) return;
      setPage({ items: [], complete: false, generated_at: "" });
      setState("error");
    });
    return () => controller.abort();
  }, [currentCursor, kind, loadPage, retry]);

  function changeKind(value: RiskIndicatorKind | undefined) {
    setKind(value);
    setCursors([]);
  }

  const pagination = cursors.length > 0 || page.next_cursor ? {
    label: "Indicator pages",
    previousLabel: "Load previous page",
    nextLabel: "Load next page",
    onPrevious: cursors.length > 0 ? () => setCursors((value) => value.slice(0, -1)) : undefined,
    onNext: page.next_cursor ? () => setCursors((value) => [...value, page.next_cursor!]) : undefined,
    isLoading: state === "loading",
  } : undefined;

  const columns: readonly DataColumn<IndicatorInsight>[] = [
    {
      id: "indicator",
      header: "Indicator",
      mobileLayout: "full-width",
      render: (item) => <span className="indicator-insights__identity">
        <strong>{item.check_name}</strong>
        <small>{item.kind} · {item.check_code} · {item.program_name}</small>
      </span>,
      accessibleText: (item) => \`\${item.kind}, \${item.check_name}, \${item.check_code}, \${item.program_name}\`,
    },
    {
      id: "value",
      header: "Current value",
      mobileLayout: "full-width",
      render: (item) => <IndicatorValue measurement={item.native_measurement} score={item.score} denominator={item.denominator}/>,
      accessibleText: (item) => indicatorValueAccessibleText(item.native_measurement, item.score, item.denominator),
    },
    {
      id: "condition",
      header: "Condition",
      kind: "status",
      render: (item) => <span className="indicator-insights__state">
        <StatusBadge tone={indicatorTone(item.state)}>{indicatorStateLabel(item.state)}</StatusBadge>
        <small>{item.reason}</small>
      </span>,
      accessibleText: (item) => \`\${indicatorStateLabel(item.state)}. \${item.reason}\`,
    },
    {
      id: "quality",
      header: "Quality",
      render: (item) => item.coverage === undefined
        ? "No result"
        : <span className="indicator-insights__stack"><strong>{formatIndicatorCoverage(item.coverage)}</strong><small>Minimum {formatIndicatorCoverage(item.minimum_coverage)}</small></span>,
      accessibleText: (item) => item.coverage === undefined
        ? "No current result"
        : \`\${formatIndicatorCoverage(item.coverage)} coverage, minimum \${formatIndicatorCoverage(item.minimum_coverage)}\`,
    },
    {
      id: "owner",
      header: "Owner",
      render: (item) => item.owner_display_name || "Not assigned",
      accessibleText: (item) => item.owner_display_name || "Owner not assigned",
    },
    {
      id: "risks",
      header: "Linked risks",
      kind: "numeric",
      render: (item) => item.risk_count,
      accessibleText: (item) => \`\${item.risk_count} linked \${item.risk_count === 1 ? "risk" : "risks"}\`,
    },
    {
      id: "updated",
      header: "Observed",
      render: (item) => item.evaluated_at ? formatIndicatorDate(item.evaluated_at) : "No result",
      accessibleText: (item) => item.evaluated_at ? formatIndicatorDate(item.evaluated_at) : "No current result",
    },
  ];

  return <section className="indicator-insights" aria-labelledby="indicator-insights-heading">
    <div className="section-header indicator-insights__heading">
      <div>
        <h2 id="indicator-insights-heading">Indicators</h2>
        <p>Current governed KRI and KCI measurements for {legalEntityName || "this legal entity"}.</p>
      </div>
    </div>

    <FilterBar
      label="Indicator filters"
      fields={<SelectField
        label="Indicator type"
        value={kind}
        placeholder="KRI and KCI"
        options={kindOptions}
        onChange={changeKind}
      />}
      resultCount={state === "live" ? page.items.length : undefined}
      resultLabel={(count) => \`\${count} shown\`}
      onClear={kind ? () => changeKind(undefined) : undefined}
    />

    {!page.complete && state === "live" && <Notice tone="warning">Some Indicator details are unavailable under the current source or access state.</Notice>}
    {state === "loading" && page.items.length === 0 && <p role="status">Loading Indicators…</p>}
    {state === "error" && <Notice tone="error"><span>Indicators unavailable.</span> <Button variant="secondary" size="compact" onPress={() => setRetry((value) => value + 1)}>Try again</Button></Notice>}
    {state === "live" && page.items.length === 0 && <EmptyState
      population={kind ? \`\${kind} Indicators\` : "KRI and KCI Indicators"}
      title="No Indicators in this scope"
      description={kind ? \`No \${kind} records were returned for the current legal entity.\` : "No governed KRI or KCI records were returned for the current legal entity."}
    />}
    {page.items.length > 0 && <DataTable
      ariaLabel="Indicator operating view"
      rows={page.items}
      rowKey={(item) => \`\${item.kind}:\${item.check_id}:\${item.check_version}:\${item.program_id}\`}
      rowName={(item) => \`\${item.kind}, \${item.check_name}, \${indicatorStateLabel(item.state)}\`}
      columns={columns}
      onRowAction={setSelected}
      rowActionLabel="Open Indicator"
      isLoading={state === "loading"}
      pagination={pagination}
    />}

    {selected && <FocusedSheet label={\`\${selected.kind} · \${selected.check_name}\`} size="wide" onClose={() => setSelected(undefined)}>
      <IndicatorDetail indicator={selected} onOpenProgram={onOpenProgram} onOpenMatter={onOpenMatter}/>
    </FocusedSheet>}
  </section>;
}

function isAbortError(error: unknown) {
  return typeof error === "object" && error !== null && "name" in error && error.name === "AbortError";
}
